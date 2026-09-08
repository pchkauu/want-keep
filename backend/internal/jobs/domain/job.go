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
	Kind                 Kind
	ConnectionID         string
	SecretPurpose        connections.SecretPurpose
	ConnectionGeneration uint64
	Binding              connections.Binding
	AdmissionRevision    int64
	State                State
	Reason               Reason
	ExternalStarted      bool
	ResourceID           string
	ResourceRevision     uint64
	Attempt, MaxAttempts int
	LeaseToken           string
	LeaseUntil, Deadline time.Time
	CancelRequested      bool
	Cursor, Coverage     string
	Gaps                 []string
	ReplayRequestID      string
	RangeFrom, RangeTo   time.Time
}

func (j Job) RequireAttempt(issued Job, now time.Time) error {
	if j.HouseholdID != issued.HouseholdID || j.ID != issued.ID || j.ActorID != issued.ActorID || j.SecretPurpose != issued.SecretPurpose || j.Kind != issued.Kind || j.Binding != issued.Binding || j.AdmissionRevision != issued.AdmissionRevision || j.ConnectionGeneration != issued.ConnectionGeneration || j.ConnectionID != issued.ConnectionID || j.ResourceID != issued.ResourceID || j.ResourceRevision != issued.ResourceRevision || j.ReplayRequestID != issued.ReplayRequestID || j.Cursor != issued.Cursor || !j.RangeFrom.Equal(issued.RangeFrom) || !j.RangeTo.Equal(issued.RangeTo) || j.State != Running || j.CancelRequested || j.LeaseToken == "" || j.LeaseToken != issued.LeaseToken || j.Attempt != issued.Attempt || !now.Before(j.LeaseUntil) || !now.Before(j.Deadline) {
		return ErrStaleAttempt
	}
	return nil
}

func (j Job) ValidateReplay() error {
	if j.ReplayRequestID == "" || j.RangeFrom.IsZero() || j.RangeTo.IsZero() || !j.RangeFrom.Before(j.RangeTo) || j.RangeTo.Sub(j.RangeFrom) > 90*24*time.Hour {
		return ErrInvalidJob
	}
	return nil
}
