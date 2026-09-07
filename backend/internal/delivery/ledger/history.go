package ledger

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"

	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/application"
	application "github.com/pchkauu/want-keep/backend/internal/ledger/application"
)

type historyPage struct {
	access        identity.Access
	transactionID string
}

func (p historyPage) cursor(revision uint64) string {
	payload := strconv.FormatUint(revision, 10)
	h := hmac.New(sha256.New, []byte(p.access.Token))
	_, _ = h.Write([]byte("ledger-history/" + string(p.access.Principal.HouseholdID()) + "/" + string(p.access.Principal.UserID()) + "/" + p.transactionID + "/" + payload))
	return payload + "." + base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
func (s *Server) history(w http.ResponseWriter, r *http.Request) {
	a, err := s.guard.Authorize(r, s.sessions, false)
	if err != nil {
		s.problem(w, err)
		return
	}
	id, err := s.transactionID(r)
	if err != nil {
		s.problem(w, err)
		return
	}
	p := historyPage{a, id}
	limit := 50
	var before uint64
	for key, values := range r.URL.Query() {
		if len(values) != 1 || key != "limit" && key != "cursor" {
			s.problem(w, contract.ErrInvalidRequest)
			return
		}
	}
	if value := r.URL.Query().Get("limit"); value != "" {
		limit, err = strconv.Atoi(value)
		if err != nil || limit < 1 || limit > 100 {
			s.problem(w, contract.ErrInvalidRequest)
			return
		}
	}
	if value := r.URL.Query().Get("cursor"); value != "" {
		raw, _, ok := strings.Cut(value, ".")
		before, err = strconv.ParseUint(raw, 10, 64)
		if !ok || err != nil || before < 1 || before > 9007199254740991 || !hmac.Equal([]byte(value), []byte(p.cursor(before))) {
			s.problem(w, contract.ErrInvalidRequest)
			return
		}
	}
	out := generated.TransactionHistoryPage{Items: []generated.TransactionHistoryEntry{}}
	err = s.reads.WithinFinancialRead(r.Context(), a.Principal, func(ctx context.Context) error {
		entries, next, e := s.queries.History(ctx, a.Principal, id, before, limit)
		if e != nil {
			return e
		}
		for _, entry := range entries {
			v, e := s.historyDTO(a, entry)
			if e != nil {
				return e
			}
			out.Items = append(out.Items, v)
		}
		if next > 0 {
			cursor := p.cursor(next)
			out.NextCursor = &cursor
		}
		return nil
	})
	if err != nil {
		s.problem(w, err)
		return
	}
	s.write(w, 200, out)
}
func (s *Server) revision(w http.ResponseWriter, r *http.Request) {
	a, err := s.guard.Authorize(r, s.sessions, false)
	if err != nil {
		s.problem(w, err)
		return
	}
	id, err := s.transactionID(r)
	if err != nil {
		s.problem(w, err)
		return
	}
	v, err := strconv.ParseUint(r.PathValue("revision"), 10, 64)
	if err != nil || v < 1 || v > 9007199254740991 {
		s.problem(w, contract.ErrInvalidRequest)
		return
	}
	var out generated.Transaction
	err = s.reads.WithinFinancialRead(r.Context(), a.Principal, func(ctx context.Context) error {
		view, e := s.queries.Revision(ctx, a.Principal, id, v)
		if e != nil {
			return e
		}
		out, e = s.transactionDTO(a.Principal, view)
		return e
	})
	if err != nil {
		s.problem(w, err)
		return
	}
	s.write(w, 200, out)
}
func (s *Server) historyDTO(a identity.Access, e application.HistoryEntry) (generated.TransactionHistoryEntry, error) {
	r := e.Current.Revision
	transaction, err := s.transactionDTO(a.Principal, e.Current)
	if err != nil {
		return generated.TransactionHistoryEntry{}, err
	}
	out := generated.TransactionHistoryEntry{Transaction: transaction, Reason: r.Reason, Fields: []generated.LedgerField{}, Affected: []generated.DecisionRevision{}, UndoAvailable: e.UndoReason == "available", UndoReason: generated.TransactionHistoryEntryUndoReason(e.UndoReason)}
	out.Evidence = []generated.DecisionEvidence{}
	if r.RecordedAt.String() != "" {
		at := r.RecordedAt.String()
		out.RecordedAt = &at
	}
	if e.Before != nil {
		v, err := s.transactionDTO(a.Principal, *e.Before)
		if err != nil {
			return out, err
		}
		out.Before = &v
	}
	if e.Decision != nil {
		d := e.Decision
		for _, ref := range d.Evidence {
			out.Evidence = append(out.Evidence, generated.DecisionEvidence{Kind: generated.DecisionEvidenceKind(ref.Kind), Id: ref.ID, Revision: int64(ref.Revision)})
		}
		out.DecisionId = &d.ID
		kind := generated.TransactionHistoryEntryDecisionKind(d.Kind)
		out.DecisionKind = &kind
		if d.UndoOf != "" {
			out.UndoOf = &d.UndoOf
		}
		for _, entry := range d.Entries {
			if entry.OperationID == r.OperationID {
				for _, f := range entry.Fields {
					out.Fields = append(out.Fields, generated.LedgerField(f))
				}
			}
		}
	}
	for _, v := range e.Affected {
		out.Affected = append(out.Affected, generated.DecisionRevision{TransactionId: v.OperationID, ExpectedRevision: int64(v.Revision)})
	}
	return out, nil
}
