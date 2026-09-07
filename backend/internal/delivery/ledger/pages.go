package ledger

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/application"
	application "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

type page struct {
	access identity.Access
	filter application.Filter
	after  application.Cursor
	limit  int
	scope  string
}

func (p page) cursor(c application.Cursor) string {
	payload := base64.RawURLEncoding.EncodeToString([]byte(c.At.String() + "/" + c.ID))
	h := hmac.New(sha256.New, []byte(p.access.Token))
	_, _ = h.Write([]byte("ledger/" + string(p.access.Principal.HouseholdID()) + "/" + string(p.access.Principal.UserID()) + "/" + p.scope + "/" + payload))
	return payload + "." + base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
func (s *Server) page(r *http.Request) (page, error) {
	a, err := s.guard.Authorize(r, s.sessions, false)
	if err != nil {
		return page{}, err
	}
	p := page{access: a, limit: 50}
	values := r.URL.Query()
	for k, v := range values {
		if len(v) != 1 {
			return p, contract.ErrInvalidRequest
		}
		switch k {
		case "accountId", "from", "to", "type", "state", "search", "cursor", "limit":
		case "categoryId":
			return p, ledger.ErrFeatureUnavailable
		default:
			return p, contract.ErrInvalidRequest
		}
	}
	if value := values.Get("limit"); value != "" {
		p.limit, err = strconv.Atoi(value)
		if err != nil || p.limit < 1 || p.limit > 100 {
			return p, contract.ErrInvalidRequest
		}
	}
	p.filter = application.Filter{AccountID: values.Get("accountId"), Type: ledger.Type(values.Get("type")), State: ledger.State(values.Get("state")), Search: values.Get("search")}
	if p.filter.AccountID != "" {
		id, e := uuid.Parse(p.filter.AccountID)
		if e != nil || id.Version() != 4 || id.String() != p.filter.AccountID {
			return p, contract.ErrInvalidRequest
		}
	}
	if value := values.Get("from"); value != "" {
		p.filter.From, err = calendar.ParseDate(value)
		if err != nil {
			return p, err
		}
	}
	if value := values.Get("to"); value != "" {
		p.filter.To, err = calendar.ParseDate(value)
		if err != nil {
			return p, err
		}
	}
	if err = p.filter.Validate(); err != nil {
		return p, err
	}
	scope, _ := json.Marshal([]string{p.filter.AccountID, p.filter.From.String(), p.filter.To.String(), string(p.filter.Type), string(p.filter.State), p.filter.Search})
	p.scope = string(scope)
	if value := values.Get("cursor"); value != "" {
		if len(value) > 2048 {
			return p, contract.ErrInvalidRequest
		}
		payload, _, ok := strings.Cut(value, ".")
		raw, e := base64.RawURLEncoding.DecodeString(payload)
		if !ok || e != nil {
			return p, contract.ErrInvalidRequest
		}
		at, id, ok := strings.Cut(string(raw), "/")
		parsed, e := uuid.Parse(id)
		if !ok || e != nil || parsed.Version() != 4 || parsed.String() != id {
			return p, contract.ErrInvalidRequest
		}
		stamp, e := calendar.ParseInstant(at)
		if e != nil {
			return p, e
		}
		p.after = application.Cursor{At: stamp, ID: id}
		if !hmac.Equal([]byte(value), []byte(p.cursor(p.after))) {
			return p, contract.ErrInvalidRequest
		}
	}
	return p, nil
}
func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	p, err := s.page(r)
	if err != nil {
		s.problem(w, err)
		return
	}
	out := generated.TransactionPage{Items: []generated.Transaction{}}
	err = s.reads.WithinFinancialRead(r.Context(), p.access.Principal, func(ctx context.Context) error {
		views, next, e := s.queries.List(ctx, p.access.Principal, p.filter, p.after, p.limit)
		if e != nil {
			return e
		}
		coverage, e := s.queries.Coverage(ctx, p.access.Principal)
		if e != nil {
			return e
		}
		reasons := coverage.Reasons()
		partial := false
		for _, v := range views {
			dto, e := s.transactionDTO(p.access.Principal, v)
			if e != nil {
				return e
			}
			out.Items = append(out.Items, dto)
			partial = partial || v.Coverage.State() != reporting.Complete
		}
		if partial {
			reasons = append(reasons, "page_has_partial_transactions")
		}
		if len(reasons) > 0 {
			coverage, e = reporting.NewCoverage(reporting.Partial, reasons)
			if e != nil {
				return e
			}
		}
		out.Quality.Coverage, e = s.boundary.CoverageToDTO(coverage)
		if e != nil {
			return e
		}
		out.Quality.Freshness = "unknown"
		if next != nil {
			value := p.cursor(*next)
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
