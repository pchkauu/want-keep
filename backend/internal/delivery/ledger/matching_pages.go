package ledger

import (
	"context"
	"crypto/hmac"
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	journal "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	application "github.com/pchkauu/want-keep/backend/internal/matching/application"
	matching "github.com/pchkauu/want-keep/backend/internal/matching/domain"
)

func (s *Server) matchingList(w http.ResponseWriter, r *http.Request) {
	a, err := s.guard.Authorize(r, s.sessions, false)
	if err != nil {
		s.problem(w, err)
		return
	}
	values := r.URL.Query()
	for k, v := range values {
		if len(v) != 1 || k != "limit" && k != "cursor" && k != "state" {
			s.problem(w, contract.ErrInvalidRequest)
			return
		}
	}
	state := matching.State(values.Get("state"))
	if state != "" && !state.Valid() {
		s.problem(w, contract.ErrInvalidRequest)
		return
	}
	p := page{access: a, scope: "matching/" + string(state), limit: 50}
	if v := values.Get("limit"); v != "" {
		p.limit, err = strconv.Atoi(v)
		if err != nil || p.limit < 1 || p.limit > 100 {
			s.problem(w, contract.ErrInvalidRequest)
			return
		}
	}
	if v := values.Get("cursor"); v != "" {
		if len(v) > 2048 {
			s.problem(w, contract.ErrInvalidRequest)
			return
		}
		payload, _, ok := strings.Cut(v, ".")
		raw, e := base64.RawURLEncoding.DecodeString(payload)
		if !ok || e != nil {
			s.problem(w, contract.ErrInvalidRequest)
			return
		}
		at, id, ok := strings.Cut(string(raw), "/")
		parsed, e := uuid.Parse(id)
		if !ok || e != nil || parsed.Version() != 4 || parsed.String() != id {
			s.problem(w, contract.ErrInvalidRequest)
			return
		}
		instant, e := calendar.ParseInstant(at)
		if e != nil {
			s.problem(w, e)
			return
		}
		p.after = journal.Cursor{At: instant, ID: id}
		if !hmac.Equal([]byte(v), []byte(p.cursor(p.after))) {
			s.problem(w, contract.ErrInvalidRequest)
			return
		}
	}
	out := generated.MatchingPage{Items: []generated.MatchingCase{}}
	err = s.reads.WithinFinancialRead(r.Context(), a.Principal, func(ctx context.Context) error {
		views, next, err := s.matching.List(ctx, a.Principal, state, application.Cursor{At: p.after.At, ID: p.after.ID}, p.limit)
		if err != nil {
			return err
		}
		for _, v := range views {
			out.Items = append(out.Items, s.matchingDTO(v))
		}
		if next != nil {
			value := p.cursor(journal.Cursor{At: next.At, ID: next.ID})
			out.NextCursor = &value
		}
		return nil
	})
	if err != nil {
		s.problem(w, err)
		return
	}
	s.write(w, 200, out)
}
