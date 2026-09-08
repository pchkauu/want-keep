package domain

import (
	"errors"
	"regexp"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

var (
	ErrInvalidCommand   = errors.New("invalid command")
	ErrDuplicateCommand = errors.New("command key has different content")
	ErrVersionConflict  = errors.New("version conflict")
	ErrFinalCommand     = errors.New("command already final")
	commandIDPattern    = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	payloadHashPattern  = regexp.MustCompile(`^[0-9a-f]{64}$`)
	commandTypePattern  = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$`)
	errorCodePattern    = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
)

type Status string

const MaxRevision uint64 = 9007199254740991

const (
	Pending   Status = "pending"
	Succeeded Status = "succeeded"
	Failed    Status = "failed"
)

type Result struct {
	ResourceType string
	ResourceID   string
	Revision     uint64
}

// Command contains durable metadata only. Payloads and financial effects are stored by their owners.
type Command struct {
	id              string
	householdID     household.HouseholdID
	actorID         household.UserID
	kind            string
	payloadHash     string
	status          Status
	result          Result
	errorCode       string
	currentRevision uint64
	registeredAt    calendar.Instant
	completedAt     calendar.Instant
}

func NewCommand(id, kind, payloadHash string, principal household.Principal, registeredAt calendar.Instant) (Command, error) {
	if err := principal.RequireHousehold(principal.HouseholdID()); err != nil {
		return Command{}, err
	}
	if !commandIDPattern.MatchString(id) || !commandTypePattern.MatchString(kind) || !payloadHashPattern.MatchString(payloadHash) || registeredAt.String() == "" {
		return Command{}, ErrInvalidCommand
	}
	return Command{id: id, kind: kind, payloadHash: payloadHash, householdID: principal.HouseholdID(), actorID: principal.UserID(), status: Pending, registeredAt: registeredAt}, nil
}

func (c Command) ID() string             { return c.id }
func (c Command) Kind() string           { return c.kind }
func (c Command) Status() Status         { return c.status }
func (c Command) Result() (Result, bool) { return c.result, c.status == Succeeded }
func (c Command) ErrorCode() string      { return c.errorCode }
func (c Command) CurrentRevision() (uint64, bool) {
	return c.currentRevision, c.status == Failed && c.currentRevision != 0
}

func (c Command) RequireVisible(principal household.Principal) error {
	if err := principal.RequireHousehold(c.householdID); err != nil {
		return err
	}
	if c.actorID != principal.UserID() {
		return household.ErrForbidden
	}
	return nil
}

// CheckReplay must precede a fresh expected-revision check for an already registered command.
func (c Command) CheckReplay(principal household.Principal, kind, payloadHash string, now calendar.Instant) error {
	if err := c.RequireVisible(principal); err != nil {
		return err
	}
	if err := c.requireRetained(now); err != nil {
		return err
	}
	if c.kind != kind || c.payloadHash != payloadHash {
		return ErrDuplicateCommand
	}
	return nil
}

func (c Command) RequireRevision(expected, current uint64) error {
	if c.status != Pending {
		return ErrFinalCommand
	}
	if expected == 0 || expected > MaxRevision || current > MaxRevision || expected != current {
		return ErrVersionConflict
	}
	return nil
}

func (c Command) Succeed(result Result, completedAt calendar.Instant) (Command, error) {
	if c.status != Pending {
		return Command{}, ErrFinalCommand
	}
	if result.ResourceType == "" || result.ResourceID == "" || result.Revision == 0 || result.Revision > MaxRevision || !c.acceptsTime(completedAt) {
		return Command{}, ErrInvalidCommand
	}
	c.status, c.result, c.completedAt = Succeeded, result, completedAt
	return c, nil
}

func (c Command) Fail(code string, currentRevision uint64, completedAt calendar.Instant) (Command, error) {
	if c.status != Pending {
		return Command{}, ErrFinalCommand
	}
	if !errorCodePattern.MatchString(code) || currentRevision > MaxRevision || currentRevision != 0 && code != "version_conflict" || !c.acceptsTime(completedAt) {
		return Command{}, ErrInvalidCommand
	}
	c.status, c.errorCode, c.currentRevision, c.completedAt = Failed, code, currentRevision, completedAt
	return c, nil
}
