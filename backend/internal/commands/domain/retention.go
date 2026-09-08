package domain

import (
	"errors"
	"time"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

var (
	ErrCommandExpired  = errors.New("command detail expired")
	ErrCommandNotFound = errors.New("command recovery unavailable")
)

const (
	recentRetention    = 30 * 24 * time.Hour
	detailRetention    = 90 * 24 * time.Hour
	tombstoneRetention = 400 * 24 * time.Hour
)

// Outcome is compact recovery metadata, not the original command document or response body.
type Outcome struct {
	CommandID       string
	Status          Status
	Result          Result
	FailureCode     string
	CurrentRevision uint64
}

func (c Command) RequireDetail(principal household.Principal, now calendar.Instant) error {
	if err := c.RequireVisible(principal); err != nil {
		return err
	}
	if err := c.requireRetained(now); err != nil {
		return err
	}
	if c.status != Pending && now.Time().Sub(c.completedAt.Time()) >= detailRetention {
		return ErrCommandExpired
	}
	return nil
}

func (c Command) InRecent(principal household.Principal, now calendar.Instant) (bool, error) {
	if err := c.RequireVisible(principal); err != nil {
		return false, err
	}
	if !c.acceptsTime(now) || (c.completedAt.String() != "" && now.Time().Before(c.completedAt.Time())) {
		return false, ErrInvalidCommand
	}
	return c.status == Pending || now.Time().Sub(c.completedAt.Time()) < recentRetention, nil
}

// Recover is also used after detail cleanup. Storage must retain these fields in the tombstone.
func (c Command) Recover(principal household.Principal, kind, hash string, now calendar.Instant) (Outcome, error) {
	if err := c.CheckReplay(principal, kind, hash, now); err != nil {
		return Outcome{}, err
	}
	return Outcome{CommandID: c.id, Status: c.status, Result: c.result, FailureCode: c.errorCode, CurrentRevision: c.currentRevision}, nil
}

func (c Command) requireRetained(now calendar.Instant) error {
	if !c.acceptsTime(now) || (c.completedAt.String() != "" && now.Time().Before(c.completedAt.Time())) {
		return ErrInvalidCommand
	}
	if c.status != Pending && now.Time().Sub(c.completedAt.Time()) >= tombstoneRetention {
		return ErrCommandNotFound
	}
	return nil
}

func (c Command) acceptsTime(at calendar.Instant) bool {
	return c.registeredAt.String() != "" && at.String() != "" && !at.Time().Before(c.registeredAt.Time())
}

// RetentionCutoffs shares the domain's exact UTC durations with bounded cleanup queries.
func RetentionCutoffs(now calendar.Instant) (detail, tombstone calendar.Instant, err error) {
	if now.String() == "" {
		return detail, tombstone, ErrInvalidCommand
	}
	detail, err = calendar.ParseInstant(now.Time().Add(-detailRetention).Format(time.RFC3339Nano))
	if err != nil {
		return detail, tombstone, err
	}
	tombstone, err = calendar.ParseInstant(now.Time().Add(-tombstoneRetention).Format(time.RFC3339Nano))
	return
}
