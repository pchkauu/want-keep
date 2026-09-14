package domain_test

import (
	"errors"
	"testing"
	"time"

	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
)

func TestAttemptFenceUsesImmutableLeaseIdentity(t *testing.T) {
	now := time.Now()
	current := jobs.Job{
		ID: "job", HouseholdID: "household", ActorID: "actor", Kind: jobs.Sync,
		State: jobs.Running, Attempt: 1, LeaseToken: "lease", Cursor: "page-2",
		LeaseUntil: now.Add(time.Minute), Deadline: now.Add(time.Hour),
	}
	issued := current
	issued.Cursor = "page-1"
	if err := current.RequireAttempt(issued, now); err != nil {
		t.Fatal("mutable checkpoint cursor invalidated the active lease", err)
	}
	issued.LeaseToken = "other"
	if !errors.Is(current.RequireAttempt(issued, now), jobs.ErrStaleAttempt) {
		t.Fatal("job accepted another lease token")
	}
}
