package application

import (
	"context"
	"errors"

	domain "github.com/pchkauu/want-keep/backend/internal/attachments/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

func (s *Service) Upload(ctx context.Context, p household.Principal, u domain.Upload, data []byte) (domain.Attachment, error) {
	if err := u.Validate(); err != nil {
		return domain.Attachment{}, err
	}
	if int64(len(data)) != u.Size || domain.Hash(data) != u.Hash {
		return domain.Attachment{}, domain.ErrInvalid
	}
	if s.blobs == nil || !s.blobs.Available() {
		return domain.Attachment{}, domain.ErrUnavailable
	}
	var a domain.Attachment
	err := s.repo.WithinHousehold(ctx, p, func(ctx context.Context) error {
		account, err := s.repo.Account(ctx, p, u.AccountID)
		if err != nil {
			return domain.ErrNotFound
		}
		if account.Ownership.RequireRead(p) != nil {
			return domain.ErrNotFound
		}
		a, err = s.repo.ReserveAttachment(ctx, p, u)
		return err
	})
	if err != nil {
		return a, err
	}
	if a.OriginalReady {
		return a, nil
	}
	if err = s.blobs.Put(ctx, a.Object(0), data); err != nil {
		return a, err
	}
	err = s.repo.WithinHousehold(ctx, p, func(ctx context.Context) error { return s.repo.ReadyAttachment(ctx, p, u.ID) })
	if err != nil {
		return a, err
	}
	return s.repo.Attachment(ctx, p, u.ID)
}
func (s *Service) Metadata(ctx context.Context, p household.Principal, id string) (domain.Attachment, error) {
	a, err := s.repo.Attachment(ctx, p, id)
	if err != nil {
		return a, err
	}
	return a, a.RequireRead(p)
}
func (s *Service) Content(ctx context.Context, p household.Principal, id string, page int) (domain.Attachment, []byte, error) {
	a, err := s.Metadata(ctx, p, id)
	if err != nil {
		return a, nil, err
	}
	if err = a.RequireContent(p); err != nil {
		return a, nil, err
	}
	if page < 0 || page > len(a.Pages) {
		return a, nil, domain.ErrNotFound
	}
	if s.blobs == nil {
		return a, nil, domain.ErrUnavailable
	}
	data, err := s.blobs.Read(ctx, a.Object(page))
	if err != nil {
		return a, nil, err
	}
	// A slow file read must not authorize a now-inactive membership.
	if _, err = s.Metadata(ctx, p, id); err != nil {
		clear(data)
		return a, nil, err
	}
	return a, data, nil
}

// ReadForAnalysis is the only document input exposed to the future AI application service.
// Its caller must supply the authenticated household job principal; vault references are not accepted.
func (s *Service) ReadForAnalysis(ctx context.Context, p household.Principal, id string) ([]byte, string, error) {
	a, data, err := s.Content(ctx, p, id, 0)
	return data, a.Upload.MediaType, err
}

func (s *Service) ProcessNext(ctx context.Context) error {
	if s.blobs == nil || !s.blobs.Available() || s.processor == nil || !s.processor.Ready(ctx) {
		return domain.ErrUnavailable
	}
	attempt, exists, err := s.repo.ClaimAttachment(ctx)
	if err != nil || !exists {
		return err
	}
	if _, err = s.repo.AttachmentPrincipal(ctx, attempt); err != nil {
		return s.repo.DeferAttachment(ctx, attempt, domain.WaitingProcessor)
	}
	data, err := s.blobs.Read(ctx, attempt.Attachment.Object(0))
	if err != nil {
		return s.repo.DeferAttachment(ctx, attempt, domain.WaitingStorage)
	}
	defer clear(data)
	result, err := s.processor.Inspect(ctx, attempt.Attachment.Upload.MediaType, data)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return err
		}
		return s.repo.DeferAttachment(ctx, attempt, domain.WaitingProcessor)
	}
	if result.Reason.Rejects() {
		return s.repo.CompleteAttachment(ctx, attempt, nil, result.Reason)
	}
	if result.Reason != domain.NoReason || len(result.Pages) < 1 || len(result.Pages) > domain.MaxPages {
		return s.repo.DeferAttachment(ctx, attempt, domain.WaitingProcessor)
	}
	pages := make([]domain.Page, len(result.Pages))
	for i, preview := range result.Pages {
		pages[i] = domain.Page{Number: i + 1, Width: preview.Width, Height: preview.Height, Hash: domain.Hash(preview.PNG), Size: int64(len(preview.PNG))}
	}
	a := attempt.Attachment
	a.Pages = pages
	a.State = domain.Accepted
	a.Reason = domain.NoReason
	if a.Validate() != nil {
		return s.repo.DeferAttachment(ctx, attempt, domain.WaitingProcessor)
	}
	for i, preview := range result.Pages {
		if err = s.blobs.Put(ctx, a.Object(i+1), preview.PNG); err != nil {
			return s.repo.DeferAttachment(ctx, attempt, domain.WaitingStorage)
		}
	}
	return s.repo.CompleteAttachment(ctx, attempt, pages, domain.NoReason)
}
