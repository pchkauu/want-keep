package reconciliation

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
	application "github.com/pchkauu/want-keep/backend/internal/reconciliation/application"
	reconciliation "github.com/pchkauu/want-keep/backend/internal/reconciliation/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

type page struct {
	access identity.Access
	filter application.Filter
	after  application.Cursor
	limit  int
	scope  string
}

func (p page) cursor(cursor application.Cursor) string {
	payload := base64.RawURLEncoding.EncodeToString([]byte(cursor.At.String() + "/" + cursor.ID))
	mac := hmac.New(sha256.New, []byte(p.access.Token))
	_, _ = mac.Write([]byte("reconciliation/" + string(p.access.Principal.HouseholdID()) + "/" + string(p.access.Principal.UserID()) + "/" + p.scope + "/" + payload))
	return payload + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (s *Server) page(r *http.Request) (page, error) {
	access, err := s.guard.Authorize(r, s.sessions, false)
	if err != nil {
		return page{}, err
	}
	result := page{access: access, limit: 50}
	values := r.URL.Query()
	for key, entries := range values {
		if len(entries) != 1 {
			return result, contract.ErrInvalidRequest
		}
		switch key {
		case "accountId", "lifecycle", "result", "cursor", "limit":
		default:
			return result, contract.ErrInvalidRequest
		}
	}
	if value := values.Get("limit"); value != "" {
		result.limit, err = strconv.Atoi(value)
		if err != nil || result.limit < 1 || result.limit > 100 {
			return result, contract.ErrInvalidRequest
		}
	}
	result.filter = application.Filter{AccountID: values.Get("accountId"), Lifecycle: reconciliation.Lifecycle(values.Get("lifecycle")), Result: reconciliation.Result(values.Get("result"))}
	if result.filter.AccountID != "" {
		id, parseErr := uuid.Parse(result.filter.AccountID)
		if parseErr != nil || id.Version() != 4 || id.String() != result.filter.AccountID {
			return result, contract.ErrInvalidRequest
		}
	}
	if err = result.filter.Validate(); err != nil {
		return result, err
	}
	scope, _ := json.Marshal([]string{result.filter.AccountID, string(result.filter.Lifecycle), string(result.filter.Result)})
	result.scope = string(scope)
	if value := values.Get("cursor"); value != "" {
		if len(value) > 2048 {
			return result, contract.ErrInvalidRequest
		}
		payload, _, found := strings.Cut(value, ".")
		raw, decodeErr := base64.RawURLEncoding.DecodeString(payload)
		if !found || decodeErr != nil {
			return result, contract.ErrInvalidRequest
		}
		at, id, found := strings.Cut(string(raw), "/")
		parsed, parseErr := uuid.Parse(id)
		if !found || parseErr != nil || parsed.Version() != 4 || parsed.String() != id {
			return result, contract.ErrInvalidRequest
		}
		stamp, parseErr := calendar.ParseInstant(at)
		if parseErr != nil {
			return result, parseErr
		}
		result.after = application.Cursor{At: stamp, ID: id}
		if !hmac.Equal([]byte(value), []byte(result.cursor(result.after))) {
			return result, contract.ErrInvalidRequest
		}
	}
	return result, nil
}

func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	page, err := s.page(r)
	if err != nil {
		s.problem(w, err)
		return
	}
	out := generated.ReconciliationPage{Items: []generated.Reconciliation{}}
	err = s.reads.WithinFinancialRead(r.Context(), page.access.Principal, func(ctx context.Context) error {
		values, next, queryErr := s.service.List(ctx, page.access.Principal, page.filter, page.after, page.limit)
		if queryErr != nil {
			return queryErr
		}
		for _, value := range values {
			dto, mapErr := s.toDTO(value)
			if mapErr != nil {
				return mapErr
			}
			out.Items = append(out.Items, dto)
		}
		coverage, _ := reporting.NewCoverage(reporting.Complete, nil)
		out.Quality.Coverage, queryErr = s.boundary.CoverageToDTO(coverage)
		out.Quality.Freshness = generated.Freshness(reporting.UnknownFreshness)
		if next != nil {
			cursor := page.cursor(*next)
			out.NextCursor = &cursor
		}
		return queryErr
	})
	if err != nil {
		s.problem(w, err)
		return
	}
	s.write(w, http.StatusOK, out)
}

func (s *Server) read(w http.ResponseWriter, r *http.Request) {
	access, err := s.guard.Authorize(r, s.sessions, false)
	if err != nil {
		s.problem(w, err)
		return
	}
	id := r.PathValue("reconciliationId")
	parsed, err := uuid.Parse(id)
	if err != nil || parsed.Version() != 4 || parsed.String() != id {
		s.problem(w, contract.ErrInvalidRequest)
		return
	}
	var out generated.Reconciliation
	err = s.reads.WithinFinancialRead(r.Context(), access.Principal, func(ctx context.Context) error {
		value, queryErr := s.service.Read(ctx, access.Principal, id)
		if queryErr != nil {
			return queryErr
		}
		out, queryErr = s.toDTO(value)
		return queryErr
	})
	if err != nil {
		s.problem(w, err)
		return
	}
	s.write(w, http.StatusOK, out)
}
