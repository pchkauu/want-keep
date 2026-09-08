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
	CurrentRevision           uint64
	RegisteredAt, CompletedAt calendar.Instant
}

func (c Command) Snapshot() Snapshot {
	return Snapshot{
		ID: c.id, Kind: c.kind, PayloadHash: c.payloadHash,
		HouseholdID: c.householdID, ActorID: c.actorID, Status: c.status,
		Result: c.result, ErrorCode: c.errorCode, CurrentRevision: c.currentRevision,
		RegisteredAt: c.registeredAt, CompletedAt: c.completedAt,
	}
}

func Restore(s Snapshot) (Command, error) {
	if s.HouseholdID == "" || s.ActorID == "" || !commandIDPattern.MatchString(s.ID) || !commandTypePattern.MatchString(s.Kind) || !payloadHashPattern.MatchString(s.PayloadHash) || s.RegisteredAt.String() == "" {
		return Command{}, ErrInvalidCommand
	}
	c := Command{id: s.ID, kind: s.Kind, payloadHash: s.PayloadHash, householdID: s.HouseholdID, actorID: s.ActorID, registeredAt: s.RegisteredAt, status: Pending}
	switch s.Status {
	case Pending:
		if s.Result != (Result{}) || s.ErrorCode != "" || s.CurrentRevision != 0 || s.CompletedAt.String() != "" {
			return Command{}, ErrInvalidCommand
		}
		return c, nil
	case Succeeded:
		if s.ErrorCode != "" || s.CurrentRevision != 0 {
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
		if !errorCodePattern.MatchString(s.ErrorCode) || s.CurrentRevision > MaxRevision || s.CurrentRevision != 0 && s.ErrorCode != "version_conflict" || !c.acceptsTime(s.CompletedAt) {
			return Command{}, ErrInvalidCommand
		}
		c.status, c.errorCode, c.currentRevision, c.completedAt = Failed, s.ErrorCode, s.CurrentRevision, s.CompletedAt
		return c, nil
	default:
		return Command{}, ErrInvalidCommand
	}
}
