//go:build integration

package audit_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/security"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func TestHouseholdIsolationPayloadReplaysAndRequestGuards(t *testing.T) {
	f := newFixture(t)
	a, b := f.client(f.p), f.client(f.q)
	r := a.expense(f.create(money.RUB, "5000"), money.RUB, "500")
	key := uuid.NewString()
	in := map[string]any{"expectedRevision": 1, "reason": "Synthetic reason", "merchant": "Market"}
	path := "/transactions/" + r.Id + "/corrections"
	first := a.call("POST", path, key, in, 202)
	a.call("POST", path, key, in, 202)
	result := decode[generated.CommandSucceeded](t, first)
	if result.Status != "succeeded" {
		t.Fatal(first.Body.String())
	}
	in["merchant"] = "Another"
	a.call("POST", path, key, in, 409)
	a.call("POST", "/transactions/"+uuid.NewString()+"/corrections", key, in, 409)
	b.call("GET", "/commands/"+key, "", nil, 404)
	for _, field := range []string{"actorId", "householdId", "state", "postedAt", "origin"} {
		payload := map[string]any{"expectedRevision": 2, "reason": "attack", "note": "note", field: uuid.NewString()}
		a.call("POST", path, uuid.NewString(), payload, 400)
	}
	other := f.otherFamily()
	foreign := other.client(other.p)
	for _, suffix := range []string{"", "/history", "/revisions/1"} {
		foreign.call("GET", "/transactions/"+r.Id+suffix, "", nil, 404)
	}
	foreign.call("POST", path, uuid.NewString(), map[string]any{"expectedRevision": 2, "reason": "foreign", "note": "bad"}, 202)
	if a.transaction(r.Id).Revision != 2 {
		t.Fatal("foreign correction applied")
	}
	raw := `{"expectedRevision":2,"reason":"csrf","note":"bad"}`
	req := httptest.NewRequest("POST", "http://localhost/api/v1"+path, strings.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://localhost")
	req.Header.Set("Idempotency-Key", uuid.NewString())
	req.AddCookie(&http.Cookie{Name: security.SessionCookie, Value: string(a.token)})
	rr := httptest.NewRecorder()
	a.handler.ServeHTTP(rr, req)
	if rr.Code != 401 {
		t.Fatal("CSRF bypass", rr.Code)
	}
	for _, body := range []string{`{"expectedRevision":2,"reason":"none"}`, `{"expectedRevision":2,"reason":"invalid","principal":[{"accountId":"` + r.Postings[0].AccountId + `","role":"principal","money":{"asset":"RUB","amount":700}}]}`} {
		req = httptest.NewRequest("POST", "http://localhost/api/v1"+path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "http://localhost")
		req.Header.Set("X-CSRF-Token", a.token.CSRF())
		req.Header.Set("Idempotency-Key", uuid.NewString())
		req.AddCookie(&http.Cookie{Name: security.SessionCookie, Value: string(a.token)})
		rr = httptest.NewRecorder()
		a.handler.ServeHTTP(rr, req)
		if rr.Code != 400 {
			t.Fatal("invalid shape", rr.Code, rr.Body.String())
		}
	}
	// Both account ownership and original payer remain independent from the correcting actor.
	view := b.correct(a.transaction(r.Id), map[string]any{"note": "Partner correction"})
	if view.ActorId != string(f.q.UserID()) {
		t.Fatal("actor was not the session owner")
	}
	if err := f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error {
		_, _, e := f.store.CurrentLedgerRevision(ctx, foreign.p, r.Id)
		return e
	}); err == nil {
		t.Fatal("principal switched inside transaction")
	}
}
