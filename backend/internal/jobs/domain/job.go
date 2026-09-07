package domain

import (
	"errors"
	"time"

	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

var ErrStaleAttempt = errors.New("stale job attempt")
var ErrInvalidJob = errors.New("invalid job")

type Job struct {
	ID                   string
	HouseholdID          household.HouseholdID
	ActorID              household.UserID
	Kind, ConnectionID   string
	SecretPurpose        connections.SecretPurpose
	ConnectionGeneration uint64
	Binding              connections.Binding
	AdmissionRevision    int64
	State                string
	Attempt, MaxAttempts int
	LeaseToken           string
	LeaseUntil, Deadline time.Time
	CancelRequested      bool
	Cursor, Coverage     string
	Gaps                 []string
}

func (j Job) RequireAttempt(issued Job, now time.Time) error {
	if j.HouseholdID != issued.HouseholdID || j.ID != issued.ID || j.ActorID != issued.ActorID || j.SecretPurpose != issued.SecretPurpose || j.Kind != issued.Kind || j.Binding != issued.Binding || j.AdmissionRevision != issued.AdmissionRevision || j.ConnectionGeneration != issued.ConnectionGeneration || j.ConnectionID != issued.ConnectionID || j.State != "running" || j.CancelRequested || j.LeaseToken == "" || j.LeaseToken != issued.LeaseToken || j.Attempt != issued.Attempt || !now.Before(j.LeaseUntil) || !now.Before(j.Deadline) {
		return ErrStaleAttempt
	}
	return nil
}
