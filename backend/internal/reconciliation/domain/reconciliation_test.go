package domain_test

import (
	"errors"
	"strings"
	"testing"

	account "github.com/pchkauu/want-keep/backend/internal/accounts/domain"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	reconciliation "github.com/pchkauu/want-keep/backend/internal/reconciliation/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

func instant(value string) calendar.Instant {
	result, err := calendar.ParseInstant(value)
	if err != nil {
		panic(err)
	}
	return result
}

func known(value string, asset money.Asset) reporting.Amount {
	amount, err := money.NewMoney(value, asset)
	if err != nil {
		panic(err)
	}
	result, err := reporting.KnownAmount(amount)
	if err != nil {
		panic(err)
	}
	return result
}

func amounts(value string, asset money.Asset) account.Amounts {
	zero := known("0", asset)
	return account.Amounts{Owned: known(value, asset), Available: known(value, asset), Locked: zero, Debt: zero}
}

func complete() reporting.Coverage {
	result, err := reporting.NewCoverage(reporting.Complete, nil)
	if err != nil {
		panic(err)
	}
	return result
}

func binding() connections.Binding {
	return connections.Binding{
		Provider: "raiffeisen", Environment: "test",
		AdapterBuildDigest:   "sha256:" + strings.Repeat("a", 64),
		CollectorImageDigest: "sha256:" + strings.Repeat("b", 64),
		ContractVersion:      "10", AllowlistRevision: "1",
		NonSecretConfigRevision: "1", OperatorPermissionRevision: "1",
	}
}

func evaluation(asset money.Asset, source, ledger string) reconciliation.Evaluation {
	return reconciliation.Evaluation{
		ID: "reconciliation", AccountID: "account", ObservationID: "observation", Revision: 1,
		SourceAsOf: instant("2026-09-08T09:00:00.123456789Z"), EvaluatedAt: instant("2026-09-08T10:00:00.987654321Z"),
		Asset: asset, Source: amounts(source, asset), Ledger: amounts(ledger, asset),
		Coverage: complete(), Freshness: reporting.Fresh, Replay: reconciliation.Replay{Status: reconciliation.ReplayNotRequired},
	}
}

func TestEvaluateExactDifferencesForEveryAsset(t *testing.T) {
	for _, test := range []struct {
		asset                money.Asset
		source, ledger, want string
	}{
		{money.RUB, "1000", "900", "100"},
		{money.USD, "1.000000000000000001", "0.999999999999999999", "0.000000000000000002"},
		{money.USDT, "100.00000001", "99.99999999", "0.00000002"},
		{money.USDC, "0.123456789", "0.023456788", "0.100000001"},
		{money.BTC, "0.00000003", "0.00000001", "0.00000002"},
		{money.ETH, "1.000000000000000001", "0.000000000000000001", "1"},
	} {
		t.Run(string(test.asset), func(t *testing.T) {
			result, err := reconciliation.Evaluate(evaluation(test.asset, test.source, test.ledger))
			if err != nil {
				t.Fatal(err)
			}
			component, found := result.Component(reconciliation.Owned)
			if !found || result.Result != reconciliation.Discrepant {
				t.Fatalf("unexpected result: %#v", result)
			}
			difference, known := component.Difference.Value()
			expected, expectedErr := money.NewMoney(test.want, test.asset)
			comparison, compareErr := difference.Compare(expected)
			if !known || expectedErr != nil || compareErr != nil || comparison != 0 || difference.Asset() != test.asset {
				t.Fatalf("difference = %v, known=%v", difference, known)
			}
		})
	}
}

func TestEvaluateKeepsFourComponentsIndependent(t *testing.T) {
	input := evaluation(money.RUB, "100", "100")
	input.Source.Available = known("80", money.RUB)
	input.Source.Locked = known("20", money.RUB)
	input.Source.Debt = known("300", money.RUB)
	input.Ledger.Available = known("75", money.RUB)
	input.Ledger.Locked = known("25", money.RUB)
	input.Ledger.Debt = known("250", money.RUB)
	result, err := reconciliation.Evaluate(input)
	if err != nil {
		t.Fatal(err)
	}
	for component, want := range map[reconciliation.ComponentName]string{
		reconciliation.Owned: "0", reconciliation.Available: "5", reconciliation.Locked: "-5", reconciliation.Debt: "50",
	} {
		value, found := result.Component(component)
		difference, known := value.Difference.Value()
		if !found || !known || difference.Amount() != want {
			t.Fatalf("%s difference = %v, found=%v known=%v", component, difference, found, known)
		}
	}
	if !reconciliation.Owned.Adjustable() || !reconciliation.Debt.Adjustable() || reconciliation.Available.Adjustable() || reconciliation.Locked.Adjustable() {
		t.Fatal("adjustable component policy changed")
	}
}

func TestUnknownCoverageAndStalenessRemainExplicit(t *testing.T) {
	input := evaluation(money.RUB, "100", "100")
	input.Ledger.Debt, _ = reporting.MissingAmount(reporting.Unknown, "transition_time_unknown")
	input.Coverage, _ = reporting.NewCoverage(reporting.Partial, []string{"history_gap"})
	input.Freshness = reporting.Stale
	result, err := reconciliation.Evaluate(input)
	if err != nil {
		t.Fatal(err)
	}
	if result.Result != reconciliation.Incomplete || len(result.Explanations) != 2 {
		t.Fatalf("quality was collapsed: %#v", result)
	}
	debt, _ := result.Component(reconciliation.Debt)
	if _, known := debt.Difference.Value(); known || debt.Difference.Reason() != "transition_time_unknown" {
		t.Fatalf("unknown debt became a number: %#v", debt)
	}
}

func TestReplayBoundsAndStableEvaluation(t *testing.T) {
	input := evaluation(money.RUB, "1000", "900")
	input.Replay = reconciliation.Replay{
		RequestID: "request", ConnectionID: "connection", Status: reconciliation.ReplayPending,
		From: instant("2026-06-10T09:00:00.123456789Z"), To: input.SourceAsOf,
		Binding: binding(), AdmissionRevision: 1, ConnectionGeneration: 1,
	}
	first, err := reconciliation.Evaluate(input)
	if err != nil {
		t.Fatal(err)
	}
	second := first
	second.Revision++
	second.EvaluatedAt = instant("2026-09-08T11:00:00Z")
	if !first.SameEvaluation(second) {
		t.Fatal("evaluation time alone created a semantic change")
	}
	input.Replay.From = instant("2026-06-09T09:00:00Z")
	if _, err = reconciliation.Evaluate(input); err == nil {
		t.Fatal("replay wider than 90 days was accepted")
	}
}

func TestReplayTransitionsAreMonotonicAndFullySpecified(t *testing.T) {
	pending := reconciliation.Replay{
		RequestID: "request", ConnectionID: "connection", Status: reconciliation.ReplayPending,
		From: instant("2026-09-01T09:00:00Z"), To: instant("2026-09-08T09:00:00Z"),
		Binding: binding(), AdmissionRevision: 1, ConnectionGeneration: 1,
	}
	withJob, changed, err := pending.Transition(reconciliation.ReplayPending, "job", "")
	if err != nil || !changed {
		t.Fatal("pending replay did not accept its job", changed, err)
	}
	completed, changed, err := withJob.Transition(reconciliation.ReplayCompleted, "job", "history_replayed")
	if err != nil || !changed || completed.Validate() != nil {
		t.Fatal("replay did not complete", changed, err)
	}
	if _, changed, err = completed.Transition(reconciliation.ReplayFailed, "job", "late_failure"); !errors.Is(err, reconciliation.ErrNotReady) || changed {
		t.Fatal("terminal replay transitioned again", changed, err)
	}
	if _, changed, err = completed.Transition(reconciliation.ReplayCompleted, "job", "history_replayed"); err != nil || changed {
		t.Fatal("identical terminal replay was not a no-op", changed, err)
	}
	invalid := reconciliation.Replay{Status: reconciliation.ReplayNotRequired, ConnectionID: "hidden"}
	if invalid.Validate() == nil {
		t.Fatal("not-required replay retained hidden state")
	}
}
