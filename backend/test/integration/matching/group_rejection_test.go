//go:build integration

package matching_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	matching "github.com/pchkauu/want-keep/backend/internal/matching/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func TestSeparateSurvivesLinkedCandidateExpansion(t *testing.T) {
	f := newFixture(t)
	account := f.create(money.RUB, "1000")
	gate, connection := f.admit(), f.connection(f.p)
	b := f.revision(uuid.NewString(), account, "-100", money.RUB, 1)
	a := f.revision(uuid.NewString(), account, "-100", money.RUB, 1)
	for _, r := range []*ledger.Revision{&b, &a} {
		if _, err := f.importSource(gate, connection, f.sourceInput(r, r.OperationID)); err != nil {
			t.Fatal(err)
		}
	}
	g, found, err := f.store.MatchingForOperation(testContext, f.p, a.OperationID)
	if err != nil || !found {
		t.Fatal(found, err)
	}
	c := f.client(f.p)
	c.result("/matching/"+g.ID+"/resolve", map[string]any{"expectedRevision": g.Revision, "decision": "separate", "reason": "A and B are distinct payments", "expectedRevisions": f.versions(a.OperationID)})
	f.balance(account, "owned", "800")
	proof := &ledger.Correspondence{Kind: "payment", Namespace: "synthetic:payment", Reference: "third-evidence"}
	third := f.revision(uuid.NewString(), account, "-100", money.RUB, 1)
	third.Correspondence = proof
	if _, err := f.importSource(gate, connection, f.sourceInput(&third, third.OperationID)); err != nil {
		t.Fatal(err)
	}
	f.link(matching.Payment, b.OperationID, b.OperationID, third.OperationID)
	f.balance(account, "owned", "800")
	rejected, err := f.store.MatchingRejected(testContext, f.p, []string{a.OperationID, b.OperationID})
	if err != nil || !rejected {
		t.Fatal("missing rejected pair", rejected, err)
	}
	a.Correspondence = proof
	in := f.sourceInput(&a, a.OperationID)
	in.Classification, in.ExpectedRevision, in.PayloadHash = "correction", 1, strings.Repeat("b", 64)
	if _, err := f.importSource(gate, connection, in); err != nil {
		t.Fatal(err)
	}
	after, found, err := f.store.MatchingForOperation(testContext, f.p, a.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("retained rejection A/B=%t; A active group=%t state=%s members=%d", rejected, found, after.State, len(after.Members))
	f.balance(account, "owned", "800")
	if found {
		t.Fatal("confirmed separate expense acquired matching participation", after.State)
	}
	current := f.current(a.OperationID)
	if current.Correspondence == nil || *current.Correspondence != *proof {
		t.Fatal("new source evidence was discarded")
	}
	cases, revisions := f.count("matching_revisions"), f.count("operation_revisions")
	if out, err := f.importSource(gate, connection, in); err != nil || !out.Duplicate {
		t.Fatal(out, err)
	}
	if f.count("matching_revisions") != cases || f.count("operation_revisions") != revisions {
		t.Fatal("replay recreated matching or financial changes")
	}
	f.balance(account, "owned", "800")
}
