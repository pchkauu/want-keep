package domain

import (
	"testing"
	"time"
)

func TestJobCompletionAndReconciliation(t *testing.T) {
	for _, test := range []struct {
		name   string
		job    Job
		state  State
		reason Reason
		want   Outcome
	}{
		{"retry exhausted", Job{Attempt: 5, MaxAttempts: 5}, Ready, "", Outcome{Failed, AttemptsExhausted, 5}},
		{"unknown before retry limit", Job{Attempt: 5, MaxAttempts: 5, ExternalStarted: true}, Ready, "", Outcome{Unresolved, ExternalUnknown, 5}},
		{"canceled retry", Job{Attempt: 2, MaxAttempts: 5, CancelRequested: true}, Ready, "", Outcome{Canceled, Cancellation, 2}},
		{"dependency refund", Job{State: Running, Attempt: 5, MaxAttempts: 5}, Waiting, BudgetWait, Outcome{Waiting, BudgetWait, 4}},
		{"confirmed completion clears reason", Job{Attempt: 5, ExternalStarted: true}, Succeeded, TemporaryFailure, Outcome{Succeeded, "", 5}},
		{"permanent failure", Job{Attempt: 1}, Failed, "", Outcome{Failed, PermanentFailure, 1}},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.job.Complete(test.state, test.reason)
			if err != nil || got != test.want {
				t.Fatalf("got %+v, %v; want %+v", got, err, test.want)
			}
		})
	}
	j := Job{State: Unresolved, Attempt: 5, MaxAttempts: 5, ExternalStarted: true}
	if got := j.Reconcile(true); got != (Outcome{Failed, AttemptsExhausted, 5}) {
		t.Fatalf("exhausted reconciliation: %+v", got)
	}
	if got := j.Reconcile(false); got != (Outcome{Succeeded, "", 5}) {
		t.Fatalf("confirmed final effect: %+v", got)
	}
	if !j.ExternalStarted || j.State != Unresolved {
		t.Fatal("decision mutated source job")
	}
	if _, err := j.Complete(Waiting, ""); err != ErrInvalidJob {
		t.Fatal("invalid dependency accepted")
	}
	if _, err := j.Complete(Running, ""); err != ErrInvalidJob {
		t.Fatal("invalid completion accepted")
	}
}

func TestJobRecoveryPolicy(t *testing.T) {
	now := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name                                                  string
		state                                                 State
		attempt                                               int
		external, canceled, active, liveLease, futureDeadline bool
		dependency                                            Reason
		want                                                  Outcome
		changed                                               bool
	}{
		{name: "unstarted unavailable", state: Ready, active: true, dependency: HandlerUnavailable, want: Outcome{Waiting, HandlerUnavailable, 0}, changed: true},
		{name: "unstarted expired with handler", state: Ready, active: true, want: Outcome{Failed, DeadlineExceeded, 0}, changed: true},
		{name: "attempted expired", state: Ready, attempt: 1, active: true, dependency: HandlerUnavailable, want: Outcome{Failed, DeadlineExceeded, 1}, changed: true},
		{name: "unstarted revoked", state: Ready, futureDeadline: true, dependency: HandlerUnavailable, want: Outcome{Failed, MembershipRevoked, 0}, changed: true},
		{name: "unstarted canceled", state: Ready, canceled: true, active: true, dependency: HandlerUnavailable, want: Outcome{Canceled, Cancellation, 0}, changed: true},
		{name: "retry exhausted", state: Running, attempt: 5, active: true, futureDeadline: true, dependency: HandlerUnavailable, want: Outcome{Failed, AttemptsExhausted, 5}, changed: true},
		{name: "unknown survives cancel and expiry", state: Running, attempt: 5, external: true, canceled: true, dependency: HandlerUnavailable, want: Outcome{Unresolved, ExternalUnknown, 5}, changed: true},
		{name: "crashed attempt remains consumed", state: Running, attempt: 2, active: true, futureDeadline: true, dependency: HandlerUnavailable, want: Outcome{Waiting, HandlerUnavailable, 2}, changed: true},
		{name: "live attempt untouched", state: Running, attempt: 1, active: true, liveLease: true, futureDeadline: true, dependency: HandlerUnavailable, want: Outcome{Running, "", 1}},
		{name: "ready eligible", state: Ready, active: true, futureDeadline: true, want: Outcome{Ready, "", 0}},
		{name: "unresolved untouched", state: Unresolved, attempt: 5, external: true, want: Outcome{Unresolved, "", 5}},
	} {
		t.Run(test.name, func(t *testing.T) {
			j := Job{State: test.state, Attempt: test.attempt, MaxAttempts: 5, ExternalStarted: test.external, CancelRequested: test.canceled, Deadline: now, LeaseUntil: now}
			if test.futureDeadline {
				j.Deadline = now.Add(time.Hour)
			}
			if test.liveLease {
				j.LeaseUntil = now.Add(time.Minute)
			}
			got, changed := j.Recover(now, test.active, test.dependency)
			if got != test.want || changed != test.changed {
				t.Fatalf("got %+v/%t; want %+v/%t", got, changed, test.want, test.changed)
			}
		})
	}
}
