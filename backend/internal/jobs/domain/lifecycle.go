package domain

import "time"

const (
	DeadlineExceeded  Reason = "deadline_exceeded"
	AttemptsExhausted Reason = "attempts_exhausted"
	Cancellation      Reason = "canceled"
	MembershipRevoked Reason = "membership_revoked"
)

type Outcome struct {
	State   State
	Reason  Reason
	Attempt int
}

// Complete determines the result of a fenced execution attempt.
func (j Job) Complete(state State, reason Reason) (Outcome, error) {
	if state != Ready && state != Waiting && state != Succeeded && state != Failed && state != Unresolved {
		return Outcome{}, ErrInvalidJob
	}
	if state == Waiting && !reason.Waiting() {
		return Outcome{}, ErrInvalidJob
	}
	out := Outcome{state, reason, j.Attempt}
	switch {
	case state == Succeeded:
		out.Reason = ""
	case j.ExternalStarted || state == Unresolved:
		out.State, out.Reason = Unresolved, ExternalUnknown
	case j.CancelRequested:
		out.State, out.Reason = Canceled, Cancellation
	case state == Ready && j.Attempt >= j.MaxAttempts:
		out.State, out.Reason = Failed, AttemptsExhausted
	case state == Waiting:
		if j.State == Running && out.Attempt > 0 {
			out.Attempt--
		}
	case state == Ready && reason == "":
		out.Reason = TemporaryFailure
	case state == Failed && reason == "":
		out.Reason = PermanentFailure
	}
	return out, nil
}

// Reconcile is called only after trusted evidence has resolved the external action.
func (j Job) Reconcile(continueWork bool) Outcome {
	j.ExternalStarted = false
	state := Succeeded
	if continueWork {
		state = Ready
	}
	out, _ := j.Complete(state, "")
	return out
}

// Recover retains unstarted dependency work even if the worker was offline past its deadline.
func (j Job) Recover(now time.Time, membershipActive bool, dependency Reason) (Outcome, bool) {
	unchanged := Outcome{j.State, j.Reason, j.Attempt}
	if j.State != Ready && j.State != Running {
		return unchanged, false
	}
	if j.State == Running && now.Before(j.LeaseUntil) && now.Before(j.Deadline) && !j.CancelRequested {
		return unchanged, false
	}
	out := Outcome{Failed, "", j.Attempt}
	switch {
	case j.ExternalStarted:
		out.State, out.Reason = Unresolved, ExternalUnknown
	case j.CancelRequested:
		out.State, out.Reason = Canceled, Cancellation
	case j.State == Ready && j.Attempt == 0 && membershipActive && dependency.Waiting():
		out.State, out.Reason = Waiting, dependency
	case !now.Before(j.Deadline):
		out.Reason = DeadlineExceeded
	case j.Attempt >= j.MaxAttempts:
		out.Reason = AttemptsExhausted
	case !membershipActive:
		out.Reason = MembershipRevoked
	case dependency.Waiting():
		out.State, out.Reason = Waiting, dependency
	case j.State == Running:
		out.State, out.Reason = Ready, TemporaryFailure
	default:
		return unchanged, false
	}
	return out, true
}
