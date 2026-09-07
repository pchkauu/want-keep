package domain

import (
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

// Snapshot is the complete durable state; restoring it never executes a command.
type Snapshot struct {
	ID, Kind, PayloadHash     string
	HouseholdID               household.HouseholdID
	ActorID                   household.UserID
	Status                    Status
	Result                    Result
	ErrorCode                 string
	RegisteredAt, CompletedAt calendar.Instant
}

func (c Command) Snapshot() Snapshot {
	return Snapshot{c.id, c.kind, c.payloadHash, c.householdID, c.actorID, c.status, c.result, c.errorCode, c.registeredAt, c.completedAt}
}

func Restore(s Snapshot) (Command, error) {
	if s.HouseholdID == "" || s.ActorID == "" || !commandIDPattern.MatchString(s.ID) || !commandTypePattern.MatchString(s.Kind) || !payloadHashPattern.MatchString(s.PayloadHash) || s.RegisteredAt.String() == "" {
		return Command{}, ErrInvalidCommand
	}
	c := Command{id: s.ID, kind: s.Kind, payloadHash: s.PayloadHash, householdID: s.HouseholdID, actorID: s.ActorID, registeredAt: s.RegisteredAt, status: Pending}
	switch s.Status {
	case Pending:
		if s.Result != (Result{}) || s.ErrorCode != "" || s.CompletedAt.String() != "" {
			return Command{}, ErrInvalidCommand
		}
		return c, nil
	case Succeeded:
		if s.ErrorCode != "" {
			return Command{}, ErrInvalidCommand
		}
		if s.Result.ResourceType == "" || s.Result.ResourceID == "" || s.Result.Revision == 0 || s.Result.Revision > MaxRevision || !c.acceptsTime(s.CompletedAt) {
			return Command{}, ErrInvalidCommand
		}
		c.status, c.result, c.completedAt = Succeeded, s.Result, s.CompletedAt
		return c, nil
	case Failed:
		if s.Result != (Result{}) {
			return Command{}, ErrInvalidCommand
		}
		if !errorCodePattern.MatchString(s.ErrorCode) || !c.acceptsTime(s.CompletedAt) {
			return Command{}, ErrInvalidCommand
		}
		c.status, c.errorCode, c.completedAt = Failed, s.ErrorCode, s.CompletedAt
		return c, nil
	default:
		return Command{}, ErrInvalidCommand
	}
}
