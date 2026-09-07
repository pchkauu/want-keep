package application

import (
	"context"
	"time"

	account "github.com/pchkauu/want-keep/backend/internal/accounts/domain"
	domain "github.com/pchkauu/want-keep/backend/internal/attachments/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

type Repository interface {
	WithinHousehold(context.Context, household.Principal, func(context.Context) error) error
	Account(context.Context, household.Principal, string) (account.Account, error)
	ReserveAttachment(context.Context, household.Principal, domain.Upload) (domain.Attachment, error)
	Attachment(context.Context, household.Principal, string) (domain.Attachment, error)
	ReadyAttachment(context.Context, household.Principal, string) error
	ClaimAttachment(context.Context) (domain.Attempt, bool, error)
	CompleteAttachment(context.Context, domain.Attempt, []domain.Page, domain.Reason) error
	DeferAttachment(context.Context, domain.Attempt, domain.Reason) error
	AttachmentPrincipal(context.Context, domain.Attempt) (household.Principal, error)
}
type Blobs interface {
	Available() bool
	Put(context.Context, domain.Object, []byte) error
	Read(context.Context, domain.Object) ([]byte, error)
}
type Preview struct {
	Width, Height int
	PNG           []byte
}
type Inspection struct {
	Pages  []Preview
	Reason domain.Reason
}
type Processor interface {
	Ready(context.Context) bool
	Inspect(context.Context, string, []byte) (Inspection, error)
}
type Status struct{ Attachments, Processor bool }

type Service struct {
	repo      Repository
	blobs     Blobs
	processor Processor
}

func NewService(r Repository, b Blobs, p Processor) *Service {
	return &Service{repo: r, blobs: b, processor: p}
}
func (s *Service) Status(ctx context.Context) Status {
	return Status{Attachments: s.blobs != nil && s.blobs.Available(), Processor: s.processor != nil && s.processor.Ready(ctx)}
}
func (s *Service) Run(ctx context.Context) error {
	timer := time.NewTicker(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			if err := s.ProcessNext(ctx); err != nil && ctx.Err() != nil {
				return ctx.Err()
			}
		}
	}
}
