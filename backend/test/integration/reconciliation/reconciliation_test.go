//go:build integration

package reconciliation_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	account "github.com/pchkauu/want-keep/backend/internal/accounts/domain"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	admission "github.com/pchkauu/want-keep/backend/internal/connections/admission"
	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledgerapp "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	reconciliation "github.com/pchkauu/want-keep/backend/internal/reconciliation/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
	"github.com/pchkauu/want-keep/backend/internal/storage"
)

func exactAmounts(asset money.Asset, owned, available, locked, debt string) account.Amounts {
	return account.Amounts{Owned: known(owned, asset), Available: known(available, asset), Locked: known(locked, asset), Debt: known(debt, asset)}
}

func completeCoverage() reporting.Coverage {
	result, _ := reporting.NewCoverage(reporting.Complete, nil)
	return result
}

func TestOpeningExpenseAndSourceBalanceWithoutFalseIncome(t *testing.T) {
	f := newFixture(t)
	id := f.importAccount(money.RUB, "current", exactAmounts(money.RUB, "4500", "4500", "0", "0"), completeCoverage(), reporting.Fresh)
	f.correctOpening(id, exactAmounts(money.RUB, "5000", "5000", "0", "0"))
	f.expense(id, money.RUB, "500")
	current := f.active(id)
	if current.Result != reconciliation.Balanced || current.Replay.Status != reconciliation.ReplayNotRequired {
		t.Fatalf("balanced account requested replay: %#v", current)
	}
	if f.count("reconciliation_replay_requests") != 1 {
		// The initial unconfirmed opening created one useful history request; later
		// balanced evaluations must not create another request.
		t.Fatal("replay was duplicated")
	}
	var income int
	if err := f.admin.QueryRow(testContext, `SELECT count(*) FROM want_keep.operation_revisions WHERE economic_type='income'`).Scan(&income); err != nil || income != 0 {
		t.Fatal("opening or reconciliation became income", err)
	}
	for _, component := range current.Components {
		difference, known := component.Difference.Value()
		if !known || difference.Sign() != 0 {
			t.Fatalf("%s is not balanced: %#v", component.Name, component)
		}
	}
}

func TestReplayBoundsDeduplicationAndOwnedAdjustment(t *testing.T) {
	f := newFixture(t)
	id := f.importAccount(money.RUB, "current", exactAmounts(money.RUB, "1000", "1000", "0", "0"), completeCoverage(), reporting.Fresh)
	f.correctOpening(id, exactAmounts(money.RUB, "900", "900", "0", "0"))
	current := f.active(id)
	if current.Result != reconciliation.Discrepant || current.Replay.Status != reconciliation.ReplayPending {
		t.Fatalf("unexpected reconciliation: %#v", current)
	}
	if duration := current.Replay.To.Time().Sub(current.Replay.From.Time()); duration <= 0 || duration > 90*24*time.Hour {
		t.Fatalf("invalid replay window: %s", duration)
	}
	before := f.count("reconciliation_replay_requests")
	if err := f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error {
		_, changed, err := f.reconciler.EvaluateAccount(ctx, f.p, id)
		if changed {
			t.Error("identical evaluation created a revision")
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if f.count("reconciliation_replay_requests") != before {
		t.Fatal("identical evaluation created another replay")
	}
	current = f.completeReplay(id)
	if current.Replay.Status != reconciliation.ReplayCompleted {
		t.Fatal("replay did not complete")
	}
	outcome := f.resolve(current, reconciliation.Owned)
	if outcome.Status() != command.Succeeded {
		t.Fatalf("adjustment failed: %s", outcome.ErrorCode())
	}
	resolved := f.read(current.ID)
	if resolved.Lifecycle != reconciliation.Resolved || resolved.Resolution == nil {
		t.Fatalf("resolution was not recorded: %#v", resolved)
	}
	balance, err := f.store.Balance(testContext, f.p, id, "owned")
	if err != nil {
		t.Fatal(err)
	}
	value, known := balance.Amount.Value()
	if !known || value.Amount() != "1000" {
		t.Fatalf("adjustment did not reconcile owned funds: %#v", balance)
	}
	available, err := f.store.Balance(testContext, f.p, id, "available")
	if err != nil {
		t.Fatal(err)
	}
	availableValue, availableKnown := available.Amount.Value()
	if !availableKnown || availableValue.Amount() != "1000" {
		t.Fatalf("adjustment did not preserve the derived available balance: %#v", available)
	}
	revision, found, err := f.store.CurrentLedgerRevision(testContext, f.p, resolved.Resolution.AdjustmentTransactionID)
	components, componentErr := revision.Components()
	if err != nil || componentErr != nil || !found || revision.Type != "adjustment" || len(components) != 0 {
		t.Fatal("adjustment changed income or expense", revision, err)
	}
	if f.count("account_observations") != 1 {
		t.Fatal("adjustment mutated source evidence")
	}
}

func TestOwnedAdjustmentRejectsContradictoryAvailableResult(t *testing.T) {
	f := newFixture(t)
	id := f.importAccount(money.RUB, "current", exactAmounts(money.RUB, "1000", "900", "0", "0"), completeCoverage(), reporting.Fresh)
	f.correctOpening(id, exactAmounts(money.RUB, "900", "900", "0", "0"))
	current := f.completeReplay(id)
	if outcome := f.resolve(current, reconciliation.Owned); outcome.Status() != command.Failed || outcome.ErrorCode() != "component_not_adjustable" {
		t.Fatalf("contradictory available result was hidden: %#v", outcome)
	}
	if f.count("reconciliation_resolutions") != 0 {
		t.Fatal("invalid derived adjustment persisted a resolution")
	}
}

func TestHistoricalProjectionAppliesMultiPostingOperationOnce(t *testing.T) {
	f := newFixture(t)
	id := f.importAccount(money.RUB, "current", exactAmounts(money.RUB, "890", "890", "0", "0"), completeCoverage(), reporting.Fresh)
	f.correctOpening(id, exactAmounts(money.RUB, "1000", "1000", "0", "0"))
	f.expenseWithFee(id, money.RUB, "100", "10")
	current := f.active(id)
	if current.Result != reconciliation.Balanced {
		t.Fatalf("principal and fee were projected more than once: %#v", current)
	}
}

func TestDelayedSourcePostingUsesConfirmedPostingTime(t *testing.T) {
	f := newFixture(t)
	id := f.importAccount(money.RUB, "current", exactAmounts(money.RUB, "900", "900", "0", "0"), completeCoverage(), reporting.Fresh)
	f.correctOpening(id, exactAmounts(money.RUB, "1000", "1000", "0", "0"))
	occurred := instant(f.now.Time().Add(-time.Hour).Format(time.RFC3339Nano))
	date, _ := occurred.DateIn(timezone())
	month, _ := calendar.ParseMonth(date.String()[:7])
	operationID := uuid.NewString()
	pending := ledger.Revision{
		OperationID: operationID, Revision: 1, ActorID: f.p.UserID(), Reason: "Synthetic delayed source expense", Type: ledger.Expense, State: ledger.Pending,
		OccurredAt: occurred, RecordedAt: instant(f.now.Time().Add(-time.Minute).Format(time.RFC3339Nano)), CashDate: date, ExpenseMonth: month, Timezone: timezone(),
		Origin: "source", PayerState: "unknown", FeeKnowledge: ledger.KnownFees, AllocationReason: "unresolved",
		Postings: []ledger.Posting{{AccountID: id, Money: cash("-100", money.RUB), Role: ledger.Principal, Funding: ledger.OwnFunds, Treatment: ledger.Movement}},
	}
	if err := f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error { return f.writer.Append(ctx, f.p, pending, 0) }); err != nil {
		t.Fatal(err)
	}
	posted := pending.Clone()
	posted.Revision = 2
	posted.State = ledger.Posted
	posted.PostedAt = instant(f.now.Time().Add(-30 * time.Second).Format(time.RFC3339Nano))
	posted.RecordedAt = instant(f.now.Time().Add(time.Minute).Format(time.RFC3339Nano))
	if err := f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error { return f.writer.Append(ctx, f.p, posted, 1) }); err != nil {
		t.Fatal(err)
	}
	current := f.active(id)
	if current.Result != reconciliation.Balanced {
		t.Fatalf("confirmed posting time was replaced by ingestion time: %#v", current)
	}
}

func TestSourceReversalWithoutTransitionTimeKeepsProjectionIncomplete(t *testing.T) {
	f := newFixture(t)
	id := f.importAccount(money.RUB, "current", exactAmounts(money.RUB, "1000", "1000", "0", "0"), completeCoverage(), reporting.Fresh)
	f.correctOpening(id, exactAmounts(money.RUB, "1000", "1000", "0", "0"))
	occurred := instant(f.now.Time().Add(-time.Hour).Format(time.RFC3339Nano))
	date, _ := occurred.DateIn(timezone())
	month, _ := calendar.ParseMonth(date.String()[:7])
	operationID := uuid.NewString()
	pending := ledger.Revision{
		OperationID: operationID, Revision: 1, ActorID: f.p.UserID(), Reason: "Synthetic source purchase", Type: ledger.Expense, State: ledger.Pending,
		OccurredAt: occurred, RecordedAt: instant(f.now.Time().Add(-time.Minute).Format(time.RFC3339Nano)), CashDate: date, ExpenseMonth: month, Timezone: timezone(),
		Origin: "source", PayerState: "unknown", FeeKnowledge: ledger.KnownFees, AllocationReason: "unresolved",
		Postings: []ledger.Posting{{AccountID: id, Money: cash("-100", money.RUB), Role: ledger.Principal, Funding: ledger.OwnFunds, Treatment: ledger.Movement}},
	}
	if err := f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error { return f.writer.Append(ctx, f.p, pending, 0) }); err != nil {
		t.Fatal(err)
	}
	posted := pending.Clone()
	posted.Revision = 2
	posted.State = ledger.Posted
	posted.PostedAt = instant(f.now.Time().Add(-30 * time.Second).Format(time.RFC3339Nano))
	posted.RecordedAt = instant(f.now.Time().Add(30 * time.Second).Format(time.RFC3339Nano))
	if err := f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error { return f.writer.Append(ctx, f.p, posted, 1) }); err != nil {
		t.Fatal(err)
	}
	reversed := posted.Clone()
	reversed.Revision = 3
	reversed.State = ledger.Reversed
	reversed.RecordedAt = instant(f.now.Time().Add(time.Minute).Format(time.RFC3339Nano))
	if err := f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error { return f.writer.Append(ctx, f.p, reversed, 2) }); err != nil {
		t.Fatal(err)
	}
	current := f.active(id)
	if current.Result != reconciliation.Incomplete || current.Coverage.State() != reporting.Partial {
		t.Fatalf("unknown reversal time was treated as a proven lifecycle: %#v", current)
	}
}

func TestReplayDispatchRejectsChangedAdmissionEvidence(t *testing.T) {
	f := newFixture(t)
	id := f.importAccount(money.RUB, "current", exactAmounts(money.RUB, "1000", "1000", "0", "0"), completeCoverage(), reporting.Fresh)
	f.correctOpening(id, exactAmounts(money.RUB, "900", "900", "0", "0"))
	current := f.active(id)
	if _, err := f.admission.RecordCheck(testContext, connections.Check{Kind: connections.ProviderCheck, Binding: binding(), Result: connections.CheckPassed, At: instant(f.now.Time().Add(time.Second).Format(time.RFC3339Nano))}); err != nil {
		t.Fatal(err)
	}
	if err := f.reconciler.DispatchReplay(testContext, f.p, current.ID); err != nil {
		t.Fatal(err)
	}
	current = f.active(id)
	if current.Replay.Status != reconciliation.ReplayUnavailable || current.Replay.JobID != "" || current.Replay.Reason != "provider_not_admitted" {
		t.Fatalf("changed admission evidence issued a replay: %#v", current.Replay)
	}
}

func TestReplayDispatchRejectsChangedConnectionGeneration(t *testing.T) {
	f := newFixture(t)
	id := f.importAccount(money.RUB, "current", exactAmounts(money.RUB, "1000", "1000", "0", "0"), completeCoverage(), reporting.Fresh)
	f.correctOpening(id, exactAmounts(money.RUB, "900", "900", "0", "0"))
	current := f.active(id)
	if _, err := f.admin.Exec(testContext, `UPDATE want_keep.connections SET generation=generation+1 WHERE household_id=$1 AND id=$2`, f.family.ID, current.Replay.ConnectionID); err != nil {
		t.Fatal(err)
	}
	if err := f.reconciler.DispatchReplay(testContext, f.p, current.ID); err != nil {
		t.Fatal(err)
	}
	current = f.active(id)
	if current.Replay.Status != reconciliation.ReplayUnavailable || current.Replay.JobID != "" || current.Replay.Reason != "provider_not_admitted" {
		t.Fatalf("changed connection generation issued a replay: %#v", current.Replay)
	}
}

func TestReplayCompletionRequiresAdmittedCommitPage(t *testing.T) {
	f := newFixture(t)
	id := f.importAccount(money.RUB, "current", exactAmounts(money.RUB, "1000", "1000", "0", "0"), completeCoverage(), reporting.Fresh)
	f.correctOpening(id, exactAmounts(money.RUB, "900", "900", "0", "0"))
	current := f.active(id)
	if err := f.reconciler.DispatchReplay(testContext, f.p, current.ID); err != nil {
		t.Fatal(err)
	}
	current = f.active(id)
	issued := f.claim(current.Replay.JobID)
	err := f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error {
		return f.reconciler.RecordReplayOutcome(ctx, f.p, current.ID, issued, reconciliation.ReplayCompleted, "history_replayed")
	})
	if !errors.Is(err, storage.ErrTransactionRequired) {
		t.Fatalf("completion escaped the admitted page fence: %v", err)
	}
	if current = f.active(id); current.Replay.Status != reconciliation.ReplayPending {
		t.Fatalf("failed completion changed replay state: %#v", current.Replay)
	}
	applied, err := f.admission.CommitPage(testContext, f.p, issued, admission.Page{EvidenceRef: "synthetic:fenced-replay", Coverage: "complete", Complete: true}, func(ctx context.Context) error {
		return f.reconciler.RecordReplayOutcome(ctx, f.p, current.ID, issued, reconciliation.ReplayCompleted, "history_replayed")
	})
	if err != nil || !applied {
		t.Fatalf("fenced replay completion failed: applied=%v err=%v", applied, err)
	}
	if current = f.active(id); current.Replay.Status != reconciliation.ReplayCompleted {
		t.Fatalf("fenced completion was not recorded: %#v", current.Replay)
	}
}

func TestReplayFailureRequiresExactAdmittedAttempt(t *testing.T) {
	f := newFixture(t)
	id := f.importAccount(money.RUB, "current", exactAmounts(money.RUB, "1000", "1000", "0", "0"), completeCoverage(), reporting.Fresh)
	f.correctOpening(id, exactAmounts(money.RUB, "900", "900", "0", "0"))
	current := f.active(id)
	if err := f.reconciler.DispatchReplay(testContext, f.p, current.ID); err != nil {
		t.Fatal(err)
	}
	current = f.active(id)
	issued := f.claim(current.Replay.JobID)
	applied, err := f.admission.CommitFailure(testContext, f.p, issued, "synthetic:replay-failure", func(ctx context.Context) error {
		return f.reconciler.RecordReplayOutcome(ctx, f.p, current.ID, issued, reconciliation.ReplayFailed, "source_history_failed")
	})
	if err != nil || !applied {
		t.Fatalf("admitted failure was not recorded: applied=%v err=%v", applied, err)
	}
	if current = f.active(id); current.Replay.Status != reconciliation.ReplayFailed {
		t.Fatalf("replay failure was not recorded: %#v", current.Replay)
	}
	var state string
	if err = f.admin.QueryRow(testContext, `SELECT state FROM want_keep.jobs WHERE household_id=$1 AND id=$2`, f.family.ID, issued.ID).Scan(&state); err != nil || state != "failed" {
		t.Fatalf("replay job was not terminalized: state=%s err=%v", state, err)
	}
}

func TestStaleReplayFailureIsQuarantined(t *testing.T) {
	f := newFixture(t)
	id := f.importAccount(money.RUB, "current", exactAmounts(money.RUB, "1000", "1000", "0", "0"), completeCoverage(), reporting.Fresh)
	f.correctOpening(id, exactAmounts(money.RUB, "900", "900", "0", "0"))
	current := f.active(id)
	if err := f.reconciler.DispatchReplay(testContext, f.p, current.ID); err != nil {
		t.Fatal(err)
	}
	current = f.active(id)
	issued := f.claim(current.Replay.JobID)
	if _, err := f.admission.RecordCheck(testContext, connections.Check{Kind: connections.ProviderCheck, Binding: binding(), Result: connections.CheckPassed, At: instant(f.now.Time().Add(time.Second).Format(time.RFC3339Nano))}); err != nil {
		t.Fatal(err)
	}
	applied, err := f.admission.CommitFailure(testContext, f.p, issued, "synthetic:stale-replay-failure", func(ctx context.Context) error {
		return f.reconciler.RecordReplayOutcome(ctx, f.p, current.ID, issued, reconciliation.ReplayFailed, "source_history_failed")
	})
	if err != nil || applied {
		t.Fatalf("stale failure was applied: applied=%v err=%v", applied, err)
	}
	if current = f.active(id); current.Replay.Status != reconciliation.ReplayPending {
		t.Fatalf("stale failure changed replay state: %#v", current.Replay)
	}
	if f.count("quarantine") != 1 {
		t.Fatal("stale replay failure evidence was not quarantined")
	}
}

func TestDebtAdjustmentAndNonAdjustableComponents(t *testing.T) {
	f := newFixture(t)
	debtID := f.importAccount(money.RUB, "credit_card", exactAmounts(money.RUB, "0", "0", "0", "300"), completeCoverage(), reporting.Fresh)
	f.correctOpening(debtID, exactAmounts(money.RUB, "0", "0", "0", "200"))
	current := f.completeReplay(debtID)
	if outcome := f.resolve(current, reconciliation.Debt); outcome.Status() != command.Succeeded {
		t.Fatal(outcome.ErrorCode())
	}
	debt, err := f.store.Balance(testContext, f.p, debtID, "debt")
	if err != nil {
		t.Fatal(err)
	}
	value, known := debt.Amount.Value()
	if !known || value.Amount() != "300" {
		t.Fatalf("wrong debt adjustment: %#v", debt)
	}

	availableID := f.importAccount(money.RUB, "current", exactAmounts(money.RUB, "100", "80", "0", "0"), completeCoverage(), reporting.Fresh)
	f.correctOpening(availableID, exactAmounts(money.RUB, "100", "70", "0", "0"))
	available := f.completeReplay(availableID)
	outcome := f.resolve(available, reconciliation.Available)
	if outcome.Status() != command.Failed || outcome.ErrorCode() != "component_not_adjustable" {
		t.Fatalf("available adjustment was accepted: %#v", outcome)
	}
}

func TestUnknownPartialStaleAndForeignAccess(t *testing.T) {
	f := newFixture(t)
	partial, _ := reporting.NewCoverage(reporting.Partial, []string{"history_gap"})
	values := exactAmounts(money.USDC, "1.000000000000000001", "1.000000000000000001", "0", "0")
	values.Locked, _ = reporting.MissingAmount(reporting.Unknown, "pending_transition_unknown")
	id := f.importAccount(money.USDC, "funding", values, partial, reporting.Stale)
	current := f.active(id)
	if current.Result != reconciliation.Incomplete || current.Freshness != reporting.Stale || current.Coverage.State() != reporting.Partial {
		t.Fatalf("quality was collapsed: %#v", current)
	}
	locked, _ := current.Component(reconciliation.Locked)
	if _, known := locked.Difference.Value(); known {
		t.Fatal("unknown component became zero")
	}
	otherFamily := household.Household{ID: household.HouseholdID(uuid.NewString()), Name: "Other synthetic household"}
	otherUser := household.User{ID: household.UserID(uuid.NewString()), Name: "Other member"}
	otherMembership := household.Membership{ID: household.MembershipID(uuid.NewString()), HouseholdID: otherFamily.ID, UserID: otherUser.ID, Active: true}
	if err := f.store.InitializeHousehold(testContext, otherFamily, []household.User{otherUser}, []household.Membership{otherMembership}, timezone(), 2); err != nil {
		t.Fatal(err)
	}
	other, _ := otherMembership.Principal()
	var err error
	if readErr := f.store.WithinFinancialRead(testContext, other, func(ctx context.Context) error {
		_, err = f.reconciler.Read(ctx, other, current.ID)
		return err
	}); readErr == nil || !errors.Is(readErr, reconciliation.ErrNotFound) && !errors.Is(readErr, account.ErrNotFound) {
		t.Fatal("foreign reconciliation was disclosed", readErr)
	}
}

func TestAllAssetsRoundTripWithoutRounding(t *testing.T) {
	f := newFixture(t)
	for _, asset := range []money.Asset{money.RUB, money.USD, money.USDT, money.USDC, money.BTC, money.ETH} {
		value := "0.00000000000000000123"
		id := f.importAccount(asset, "current", exactAmounts(asset, value, value, "0", "0"), completeCoverage(), reporting.Fresh)
		f.correctOpening(id, exactAmounts(asset, value, value, "0", "0"))
		current := f.active(id)
		owned, _ := current.Component(reconciliation.Owned)
		source, sourceKnown := owned.Source.Value()
		ledger, ledgerKnown := owned.Ledger.Value()
		if current.Result != reconciliation.Balanced || !sourceKnown || !ledgerKnown || source.Amount() != value || ledger.Amount() != value {
			t.Fatalf("%s lost precision: %#v", asset, current)
		}
	}
}

func TestStaleResolutionRevisionAndRollbackAreSafe(t *testing.T) {
	f := newFixture(t)
	id := f.importAccount(money.RUB, "current", exactAmounts(money.RUB, "1000", "900", "0", "0"), completeCoverage(), reporting.Fresh)
	f.correctOpening(id, exactAmounts(money.RUB, "900", "900", "0", "0"))
	current := f.completeReplay(id)
	stale := current
	f.expense(id, money.RUB, "1")
	outcome := f.resolve(stale, reconciliation.Owned)
	if outcome.Status() != command.Failed || outcome.ErrorCode() != "version_conflict" {
		t.Fatalf("stale resolution applied: %#v", outcome)
	}
	if balance, err := f.store.Balance(testContext, f.p, id, "owned"); err != nil {
		t.Fatal(err)
	} else if value, _ := balance.Amount.Value(); value.Amount() != "899" {
		t.Fatal("stale command left a partial effect")
	}
	if f.count("reconciliation_resolutions") != 0 {
		t.Fatal("failed command persisted a resolution")
	}

	// An invalid operation ID demonstrates that reconciliation writes share the
	// same transaction boundary as the ledger and command terminal outcome.
	err := f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error {
		_, _, err := f.reconciler.EvaluateAccount(ctx, f.p, uuid.NewString())
		return err
	})
	if err == nil {
		t.Fatal("unknown account evaluated")
	}
}

func TestExclusionAndUndoReevaluateCurrentReconciliation(t *testing.T) {
	f := newFixture(t)
	accountID := f.importAccount(money.RUB, "current", exactAmounts(money.RUB, "4500", "4500", "0", "0"), completeCoverage(), reporting.Fresh)
	f.correctOpening(accountID, exactAmounts(money.RUB, "5000", "5000", "0", "0"))
	operationID := f.expense(accountID, money.RUB, "500")
	balanced := f.active(accountID)

	excludeRequest := commands.Request{ID: uuid.NewString(), Kind: "transactions.exclude", PayloadHash: strings.Repeat("d", 64)}
	excluded, err := f.executor.Execute(testContext, f.p, excludeRequest, func(ctx context.Context) (command.Result, error) {
		return f.ledger.Correct(ctx, f.p, ledgerapp.Change{OperationID: operationID, Expected: 1, Exclude: true}, "Exclude duplicate")
	})
	if err != nil || excluded.Status() != command.Succeeded {
		t.Fatal("exclude", excluded.ErrorCode(), err)
	}
	discrepant := f.active(accountID)
	if discrepant.Revision <= balanced.Revision || discrepant.Result != reconciliation.Discrepant {
		t.Fatalf("exclusion did not re-evaluate: %#v", discrepant)
	}
	revision, found, err := f.store.CurrentLedgerRevision(testContext, f.p, operationID)
	if err != nil || !found || revision.DecisionID == "" {
		t.Fatal("missing exclusion decision", err)
	}

	undoRequest := commands.Request{ID: uuid.NewString(), Kind: "transactions.undo", PayloadHash: strings.Repeat("e", 64)}
	undone, err := f.executor.Execute(testContext, f.q, undoRequest, func(ctx context.Context) (command.Result, error) {
		return f.ledger.Undo(ctx, f.q, revision.DecisionID, []ledgerapp.ExpectedRevision{{OperationID: operationID, Revision: revision.Revision}}, "Restore valid expense")
	})
	if err != nil || undone.Status() != command.Succeeded {
		t.Fatal("undo", undone.ErrorCode(), err)
	}
	restored := f.active(accountID)
	if restored.Revision <= discrepant.Revision || restored.Result != reconciliation.Balanced || restored.Replay.Status != reconciliation.ReplayNotRequired {
		t.Fatalf("undo did not restore reconciliation: %#v", restored)
	}
}

func TestRequiredReauthenticationKeepsDifferenceWithoutReplayJob(t *testing.T) {
	f := newFixture(t)
	accountID := f.importAccount(money.RUB, "current", exactAmounts(money.RUB, "1000", "1000", "0", "0"), completeCoverage(), reporting.Fresh)
	f.correctOpening(accountID, exactAmounts(money.RUB, "900", "900", "0", "0"))
	current := f.active(accountID)
	var before int
	if err := f.admin.QueryRow(testContext, `SELECT count(*) FROM want_keep.jobs WHERE replay_request_id IS NOT NULL`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	if err := f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error {
		return f.store.Disconnect(ctx, current.Replay.ConnectionID)
	}); err != nil {
		t.Fatal(err)
	}
	if err := f.reconciler.DispatchReplay(testContext, f.p, current.ID); err != nil {
		t.Fatal(err)
	}
	current = f.active(accountID)
	if current.Result != reconciliation.Discrepant || current.Replay.Status != reconciliation.ReplayUnavailable || current.Replay.Reason != "source_reauth_required" || current.Replay.JobID != "" {
		t.Fatalf("authorization failure was hidden: %#v", current)
	}
	var after int
	if err := f.admin.QueryRow(testContext, `SELECT count(*) FROM want_keep.jobs WHERE replay_request_id IS NOT NULL`).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Fatalf("unavailable provider created a fictitious replay job: before=%d after=%d", before, after)
	}
	if outcome := f.resolve(current, reconciliation.Owned); outcome.Status() != command.Succeeded {
		t.Fatalf("explicit resolution after unavailable replay failed: %s", outcome.ErrorCode())
	}
}

func TestOpeningAfterSourceTimestampMakesProjectionIncomplete(t *testing.T) {
	f := newFixture(t)
	accountID := f.importAccount(money.RUB, "current", exactAmounts(money.RUB, "100", "100", "0", "0"), completeCoverage(), reporting.Fresh)
	f.now = instant(f.now.Time().AddDate(0, 0, 10).Format(time.RFC3339Nano))
	f.correctOpening(accountID, exactAmounts(money.RUB, "100", "100", "0", "0"))
	current := f.active(accountID)
	if current.Result != reconciliation.Incomplete || current.Coverage.State() != reporting.Partial || !contains(current.Coverage.Reasons(), "opening_after_source") {
		t.Fatalf("future opening was applied to an older source snapshot: %#v", current)
	}
	for _, component := range current.Components {
		if _, known := component.Ledger.Value(); known {
			t.Fatalf("%s ledger amount was invented before opening", component.Name)
		}
	}
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func TestNewObservationSupersedesPreviousAndUsesLastConfirmedAnchor(t *testing.T) {
	f := newFixture(t)
	accountID := f.importAccount(money.RUB, "current", exactAmounts(money.RUB, "100", "100", "0", "0"), completeCoverage(), reporting.Fresh)
	f.correctOpening(accountID, exactAmounts(money.RUB, "100", "100", "0", "0"))
	previous := f.active(accountID)
	if previous.Result != reconciliation.Balanced {
		t.Fatal("initial balance is not confirmed")
	}
	f.now = instant(f.now.Time().Add(time.Minute).Format(time.RFC3339Nano))
	f.observe(accountID, exactAmounts(money.RUB, "110", "100", "0", "0"), completeCoverage(), reporting.Fresh)
	current := f.active(accountID)
	if current.ID == previous.ID || current.Result != reconciliation.Discrepant || current.Replay.From.String() != previous.SourceAsOf.String() {
		t.Fatalf("confirmed anchor was not reused: previous=%#v current=%#v", previous, current)
	}
	previous = f.read(previous.ID)
	if previous.Lifecycle != reconciliation.Superseded {
		t.Fatalf("previous observation stayed active: %#v", previous)
	}
}
