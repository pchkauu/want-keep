//go:build integration

package ledger_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	admission "github.com/pchkauu/want-keep/backend/internal/connections/admission"
	journal "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func TestSourceTransitionsReplayOverrideAndAdmissionFence(t *testing.T) {
	f := newFixture(t)
	gate := f.admit()
	connection := f.connection(f.p)
	id := f.create(money.RUB, "5000")
	r := f.revision(uuid.NewString(), id, "-500", money.RUB, 1)
	r.State = ledger.Pending
	in := ledger.SourceInput{Key: ledger.SourceKey{HouseholdID: f.family.ID, Provider: "raiffeisen", ExternalAccountID: "synthetic-stable-account", Product: "current", Log: "statement", RecordID: "synthetic-operation"}, PayloadHash: strings.Repeat("a", 64), EvidenceRef: "synthetic:pending", Classification: "new", Operation: &r}
	first, err := f.importSource(gate, connection, in)
	if err != nil {
		t.Fatal(err)
	}
	f.balance(id, "available", "4500")
	f.balance(id, "owned", "5000")
	before := f.count("outbox")
	result, err := f.importSource(gate, f.connection(f.p), in)
	if err != nil || !result.Duplicate || result.Record.ID != first.Record.ID || f.count("outbox") != before {
		t.Fatal("reconnect repeated event", err)
	}
	r.State = ledger.Posted
	r.Revision = 2
	r.PostedAt = f.now
	in.Operation = &r
	in.Classification = "correction"
	in.ExpectedRevision = 1
	in.PayloadHash = strings.Repeat("b", 64)
	in.EvidenceRef = "synthetic:posted"
	if _, err = f.importSource(gate, connection, in); err != nil {
		t.Fatal(err)
	}
	f.balance(id, "owned", "4500")
	f.balance(id, "locked", "0")
	if _, err = f.importSource(gate, connection, in); err != nil {
		t.Fatal(err)
	}
	f.balance(id, "owned", "4500")
	r.Revision = 3
	r.Postings[0].Money = cash("-450", money.RUB)
	r.HumanOverride = true
	if _, err = f.write(r, request()); err != nil {
		t.Fatal(err)
	}
	r.Revision = 4
	r.HumanOverride = false
	r.Postings[0].Money = cash("-550", money.RUB)
	in.ExpectedRevision = 2
	in.PayloadHash = strings.Repeat("c", 64)
	in.Operation = &r
	result, err = f.importSource(gate, connection, in)
	if err != nil || !result.PreservedOverride {
		t.Fatal("manual fact overwritten", err)
	}
	f.balance(id, "owned", "4550")
	job := f.issued(gate, connection)
	changed := binding()
	changed.AllowlistRevision = "2"
	if _, err = gate.Rebind(testContext, changed); err != nil {
		t.Fatal(err)
	}
	count := f.count("source_revisions")
	called := false
	applied, err := gate.CommitPage(testContext, f.p, job, admission.Page{EvidenceRef: "synthetic:stale", Coverage: "complete", Complete: true}, func(context.Context) error { called = true; return nil })
	if err != nil || applied || called || f.count("source_revisions") != count {
		t.Fatal("stale IO committed", err)
	}
	if f.count("quarantine") != 1 {
		t.Fatal("missing stale evidence")
	}
}

func TestUnknownStatusAndIncompleteLegsKeepEvidenceWithoutPosting(t *testing.T) {
	f := newFixture(t)
	gate := f.admit()
	connection := f.connection(f.p)
	id := f.create(money.RUB, "1000")
	for _, bad := range []string{"unknown_status", "missing_leg", "not_normalized"} {
		r := f.revision(uuid.NewString(), id, "-100", money.RUB, 1)
		in := ledger.SourceInput{Key: ledger.SourceKey{HouseholdID: f.family.ID, Provider: "raiffeisen", ExternalAccountID: "stable", Product: "current", Log: "statement", RecordID: bad}, PayloadHash: strings.Repeat("a", 64), EvidenceRef: "synthetic:" + bad, Classification: "new", Operation: &r}
		switch bad {
		case "unknown_status":
			r.State = "unrecognized"
		case "missing_leg":
			r.Type = ledger.Transfer
		case "not_normalized":
			in.Operation = nil
			in.UnresolvedReason = "unsupported_asset"
		}
		if _, err := f.importSource(gate, connection, in); err != nil {
			t.Fatal(err)
		}
	}
	f.balance(id, "owned", "1000")
	if f.count("operation_revisions") != 1 || f.count("source_revisions") != 3 || f.count("quarantine") != 3 {
		t.Fatal("omitted source lost or posted")
	}
	var bad int
	if err := f.admin.QueryRow(testContext, `SELECT count(*) FROM want_keep.jobs WHERE kind='sync' AND (coverage!='partial' OR NOT('transaction_unresolved'=ANY(gaps)))`).Scan(&bad); err != nil || bad != 0 {
		t.Fatal("incomplete page advertised complete", err)
	}
}

func TestSourceSnapshotIsNotChangedOrReusedWithoutCoverage(t *testing.T) {
	f := newFixture(t)
	gate := f.admit()
	input := f.input(f.connection(f.p))
	out := f.importAccount(gate, input)
	id := out.Account.ID
	r := f.revision(uuid.NewString(), id, "-20", money.RUB, 1)
	// A purchase predating the snapshot can still be missing from its coverage.
	r.OccurredAt = instant("2026-09-01T12:00:00Z")
	r.State = ledger.Pending
	if _, err := f.write(r, request()); err != nil {
		t.Fatal(err)
	}
	for _, state := range []ledger.State{ledger.Pending, ledger.Posted} {
		if state == ledger.Posted {
			r.State = state
			r.Revision = 2
			r.PostedAt = f.now
			if _, err := f.write(r, request()); err != nil {
				t.Fatal(err)
			}
		}
		view, err := f.service().Read(testContext, f.p, id)
		if err != nil {
			t.Fatal(err)
		}
		if _, known := view.FundingAvailability().Value(); known {
			t.Fatal("snapshot coverage guessed", state)
		}
		owned, _ := view.Source.Amounts.Owned.Value()
		if owned.Amount() != "100" || f.count("account_observations") != 1 {
			t.Fatal("source changed")
		}
		if err = f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error {
			funding, err := f.store.Funding(ctx, f.p, uuid.NewString(), []string{id})
			if err == nil {
				if _, known := funding[id].Available.Value(); known {
					t.Fatal("reservation reused uncertain money")
				}
			}
			return err
		}); err != nil {
			t.Fatal(err)
		}
	}
	queries := journal.NewQueries(f.store)
	v, err := queries.Read(testContext, f.p, r.OperationID)
	if err != nil || v.Coverage.State() != "partial" {
		t.Fatal("source uncertainty hidden", err)
	}
}

func TestPnLFundingAndMiningKeepNativeEconomicMeaning(t *testing.T) {
	f := newFixture(t)
	id := f.create(money.USDT, "1000")
	other := f.create(money.USDT, "0")
	r := f.revision(uuid.NewString(), id, "10", money.USDT, 1)
	r.Type = ledger.TradeResult
	r.PnLBasis = ledger.NetPnL
	r.Postings[0].Role = ledger.PnL
	r.Postings = append(r.Postings, ledger.Posting{AccountID: id, Money: cash("-1", money.USDT), Role: ledger.Fee, Treatment: ledger.Included}, ledger.Posting{AccountID: id, Money: cash("50", money.USDT), Role: ledger.PnL, Treatment: ledger.Valuation})
	if _, err := f.write(r, request()); err != nil {
		t.Fatal(err)
	}
	f.balance(id, "owned", "1010")
	r.OperationID = uuid.NewString()
	r.PnLBasis = ledger.GrossPnL
	r.Postings[1].Treatment = ledger.Movement
	if _, err := f.write(r, request()); err != nil {
		t.Fatal(err)
	}
	f.balance(id, "owned", "1019")
	r = f.revision(uuid.NewString(), id, "-2", money.USDT, 1)
	r.Type = ledger.Yield
	r.Postings[0].Role = ledger.Funding
	if _, err := f.write(r, request()); err != nil {
		t.Fatal(err)
	}
	f.balance(id, "owned", "1017")
	r = f.revision(uuid.NewString(), id, "0.000000000000000123", money.USDT, 1)
	r.Type = ledger.Yield
	r.Postings[0].Role = ledger.Reward
	if _, err := f.write(r, request()); err != nil {
		t.Fatal(err)
	}
	r = f.revision(uuid.NewString(), id, "-17", money.USDT, 1)
	r.Type = ledger.Transfer
	r.Postings = append(r.Postings, ledger.Posting{AccountID: other, Money: cash("17", money.USDT), Role: ledger.Principal})
	if _, err := f.write(r, request()); err != nil {
		t.Fatal(err)
	}
	f.balance(id, "owned", "1000.000000000000000123")
	f.balance(other, "owned", "17")
	stored, _, err := f.store.CurrentLedgerRevision(testContext, f.p, r.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	components, err := stored.Components()
	if err != nil || len(components) != 0 {
		t.Fatal("reward transfer recognized again", err)
	}
}
