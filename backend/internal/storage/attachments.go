package storage

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	domain "github.com/pchkauu/want-keep/backend/internal/attachments/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

const attachmentColumns = `household_id,id,actor_id,account_id,object_id,name,media_type,size_bytes,content_hash,state,reason,original_ready,created_at`

func scanAttachment(row pgx.Row) (domain.Attachment, error) {
	var a domain.Attachment
	err := row.Scan(&a.HouseholdID, &a.Upload.ID, &a.ActorID, &a.Upload.AccountID, &a.ObjectID, &a.Upload.Name, &a.Upload.MediaType, &a.Upload.Size, &a.Upload.Hash, &a.State, &a.Reason, &a.OriginalReady, &a.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return a, domain.ErrNotFound
	}
	return a, err
}
func (s *Store) Attachment(ctx context.Context, p household.Principal, id string) (domain.Attachment, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return domain.Attachment{}, domain.ErrNotFound
	}
	a, err := scanAttachment(q.QueryRow(ctx, `SELECT `+attachmentColumns+` FROM want_keep.attachments WHERE household_id=$1 AND id=$2`, p.HouseholdID(), id))
	if err != nil {
		return a, err
	}
	if a.State != domain.Accepted {
		return a, a.Validate()
	}
	rows, err := q.Query(ctx, `SELECT page,width,height,size_bytes,content_hash FROM want_keep.attachment_pages WHERE household_id=$1 AND attachment_id=$2 ORDER BY page`, p.HouseholdID(), id)
	if err != nil {
		return a, err
	}
	defer rows.Close()
	for rows.Next() {
		var page domain.Page
		if err = rows.Scan(&page.Number, &page.Width, &page.Height, &page.Size, &page.Hash); err != nil {
			return a, err
		}
		a.Pages = append(a.Pages, page)
	}
	if err = rows.Err(); err != nil {
		return a, err
	}
	return a, a.Validate()
}
func (s *Store) ReserveAttachment(ctx context.Context, p household.Principal, u domain.Upload) (domain.Attachment, error) {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return domain.Attachment{}, err
	}
	if scope.principal != p {
		return domain.Attachment{}, household.ErrForbidden
	}
	a, err := s.Attachment(ctx, p, u.ID)
	if err == nil {
		return a, a.Replay(p, u)
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return a, err
	}
	if err = u.Validate(); err != nil {
		return a, err
	}
	a, err = scanAttachment(scope.tx.QueryRow(ctx, `INSERT INTO want_keep.attachments(household_id,id,actor_id,account_id,object_id,name,media_type,size_bytes,content_hash,state,reason) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,'uploaded','storage_unavailable') RETURNING `+attachmentColumns, p.HouseholdID(), u.ID, p.UserID(), u.AccountID, newID(), u.Name, u.MediaType, u.Size, u.Hash))
	if err != nil {
		return a, err
	}
	err = s.PrivacyAudit(ctx, p, u.ID, "attachment_registered")
	return a, err
}
func (s *Store) ReadyAttachment(ctx context.Context, p household.Principal, id string) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if scope.principal != p {
		return household.ErrForbidden
	}
	tag, err := scope.tx.Exec(ctx, `UPDATE want_keep.attachments SET original_ready=true,reason='processor_unavailable' WHERE household_id=$1 AND id=$2 AND actor_id=$3 AND NOT original_ready`, p.HouseholdID(), id, p.UserID())
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return nil
	}
	return s.PrivacyAudit(ctx, p, id, "attachment_ready")
}
func (s *Store) ClaimAttachment(ctx context.Context) (domain.Attempt, bool, error) {
	var a domain.Attempt
	err := s.transact(ctx, func(ctx context.Context, scope *transactionScope) error {
		var err error
		a.Attachment, err = scanAttachment(scope.tx.QueryRow(ctx, `SELECT `+attachmentColumns+` FROM want_keep.attachments a WHERE original_ready AND available_at<=clock_timestamp() AND (state='uploaded' OR (state='validating' AND lease_until<=clock_timestamp())) AND EXISTS(SELECT 1 FROM want_keep.memberships m WHERE m.household_id=a.household_id AND m.user_id=a.actor_id AND m.active) ORDER BY available_at,created_at,id LIMIT 1 FOR UPDATE OF a SKIP LOCKED`))
		if err != nil {
			return err
		}
		a.Token = newID()
		err = scope.tx.QueryRow(ctx, `UPDATE want_keep.attachments SET state='validating',reason='',attempt=attempt+1,lease_token=$3,lease_until=clock_timestamp()+INTERVAL '60 seconds' WHERE household_id=$1 AND id=$2 RETURNING attempt,lease_until`, a.Attachment.HouseholdID, a.Attachment.Upload.ID, a.Token).Scan(&a.Number, &a.LeaseUntil)
		a.Attachment.State = domain.Validating
		a.Attachment.Reason = domain.NoReason
		return err
	})
	if errors.Is(err, domain.ErrNotFound) {
		return a, false, nil
	}
	return a, err == nil, err
}
func (s *Store) AttachmentPrincipal(ctx context.Context, a domain.Attempt) (household.Principal, error) {
	m, err := s.Membership(ctx, a.Attachment.HouseholdID, a.Attachment.ActorID)
	if err != nil {
		return household.Principal{}, err
	}
	return m.Principal()
}
func (s *Store) CompleteAttachment(ctx context.Context, a domain.Attempt, pages []domain.Page, reason domain.Reason) error {
	return s.finishAttachment(ctx, a, pages, reason, false)
}
func (s *Store) DeferAttachment(ctx context.Context, a domain.Attempt, reason domain.Reason) error {
	if reason != domain.WaitingProcessor && reason != domain.WaitingStorage {
		return domain.ErrInvalid
	}
	return s.finishAttachment(ctx, a, nil, reason, true)
}
func (s *Store) finishAttachment(ctx context.Context, issued domain.Attempt, pages []domain.Page, reason domain.Reason, retry bool) error {
	p, err := s.AttachmentPrincipal(ctx, issued)
	if err != nil {
		return err
	}
	return s.WithinHousehold(ctx, p, func(ctx context.Context) error {
		scope, err := s.familyScope(ctx)
		if err != nil {
			return err
		}
		var current domain.Attempt
		current.Attachment, err = s.Attachment(ctx, p, issued.Attachment.Upload.ID)
		if err != nil {
			return err
		}
		var now time.Time
		err = scope.tx.QueryRow(ctx, `SELECT state,COALESCE(lease_token::text,''),attempt,lease_until,clock_timestamp() FROM want_keep.attachments WHERE household_id=$1 AND id=$2 FOR UPDATE`, p.HouseholdID(), issued.Attachment.Upload.ID).Scan(&current.Attachment.State, &current.Token, &current.Number, &current.LeaseUntil, &now)
		if err != nil {
			return err
		}
		if err = issued.RequireCurrent(current, now); err != nil {
			return err
		}
		a := issued.Attachment
		a.Pages = pages
		a.Reason = reason
		a.State = domain.Accepted
		event := "attachment_accepted"
		if reason.Rejects() {
			a.State = domain.Rejected
			event = "attachment_rejected"
		}
		if retry {
			a.State = domain.Uploaded
			event = "attachment_deferred"
		}
		if err = a.Validate(); err != nil {
			return err
		}
		for _, page := range pages {
			if _, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.attachment_pages(household_id,attachment_id,page,width,height,size_bytes,content_hash) VALUES($1,$2,$3,$4,$5,$6,$7)`, p.HouseholdID(), a.Upload.ID, page.Number, page.Width, page.Height, page.Size, page.Hash); err != nil {
				return err
			}
		}
		_, err = scope.tx.Exec(ctx, `UPDATE want_keep.attachments SET state=$3,reason=$4,available_at=clock_timestamp()+LEAST(attempt,120)*INTERVAL '30 seconds' WHERE household_id=$1 AND id=$2`, p.HouseholdID(), a.Upload.ID, a.State, a.Reason)
		if err != nil {
			return err
		}
		return s.PrivacyAudit(ctx, p, a.Upload.ID, event)
	})
}
func (s *Store) PrivacyAudit(ctx context.Context, p household.Principal, id, event string) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if scope.principal != p {
		return household.ErrForbidden
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.privacy_audit(household_id,id,actor_id,resource_id,event) VALUES($1,$2,$3,$4,$5)`, p.HouseholdID(), newID(), p.UserID(), id, event)
	return err
}
