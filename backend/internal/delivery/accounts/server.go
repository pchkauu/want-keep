package accounts

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"

	"github.com/google/uuid"
	accounts "github.com/pchkauu/want-keep/backend/internal/accounts/application"
	account "github.com/pchkauu/want-keep/backend/internal/accounts/domain"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/security"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/application"
	access "github.com/pchkauu/want-keep/backend/internal/identity/domain"
)

type Sessions interface {
	security.Sessions
	WithinSession(context.Context, access.Token, func(context.Context, identity.Access) error) error
}
type ReadTransactions interface {
	WithinAccountRead(context.Context, household.Principal, func(context.Context) error) error
}
type Server struct {
	service  *accounts.Service
	executor *commands.Executor
	queries  *commands.Queries
	sessions Sessions
	reads    ReadTransactions
	guard    *security.Guard
	boundary *contract.Boundary
	now      func() calendar.Instant
	mux      *http.ServeMux
}

func New(s *accounts.Service, e *commands.Executor, q *commands.Queries, sessions Sessions, reads ReadTransactions, c security.Config, now func() calendar.Instant) (*Server, error) {
	if s == nil || e == nil || q == nil || sessions == nil || reads == nil || now == nil {
		return nil, account.ErrInvalidAccount
	}
	guard, err := security.New(c)
	if err != nil {
		return nil, err
	}
	boundary, err := contract.NewBoundary()
	if err != nil {
		return nil, err
	}
	h := &Server{service: s, executor: e, queries: q, sessions: sessions, reads: reads, guard: guard, boundary: boundary, now: now, mux: http.NewServeMux()}
	h.mux.HandleFunc("GET /api/v1/accounts", h.list)
	h.mux.HandleFunc("POST /api/v1/accounts", h.create)
	h.mux.HandleFunc("GET /api/v1/accounts/{accountId}", h.read)
	h.mux.HandleFunc("POST /api/v1/accounts/{accountId}/ownership", h.ownership)
	h.mux.HandleFunc("POST /api/v1/accounts/{accountId}/opening-corrections", h.correct)
	h.mux.HandleFunc("GET /api/v1/commands/recent", h.recent)
	h.mux.HandleFunc("GET /api/v1/commands/{commandId}", h.command)
	return h, nil
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := s.guard.Check(w, r); err != nil {
		s.problem(w, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 128*1024)
	s.mux.ServeHTTP(w, r)
}
func (s *Server) decode(r *http.Request, name string, out any) error {
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		return contract.ErrInvalidRequest
	}
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return contract.ErrInvalidRequest
	}
	return s.boundary.Decode(name, data, out)
}
func (s *Server) write(w http.ResponseWriter, status int, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		s.problem(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(data)
}
func (s *Server) problem(w http.ResponseWriter, err error) {
	response := (contract.ErrorConverter{}).ToResponse(err, uuid.NewString())
	if errors.Is(err, account.ErrNotFound) {
		response.Status = 404
		response.Body.Code = "not_found"
		response.Body.Message = "Account not found."
	}
	if errors.Is(err, access.ErrUnauthorized) {
		response.Status = 401
		response.Body.Code = "unauthorized"
		response.Body.Message = "Sign in to continue."
	}
	if errors.Is(err, access.ErrAttempt) || errors.Is(err, account.ErrInvalidAccount) {
		response.Status = 400
		response.Body.Code = "invalid_request"
		response.Body.Message = "Check the request fields."
	}
	s.write(w, response.Status, response.Body)
}
func (s *Server) resourceID(r *http.Request, key string) (string, error) {
	raw := r.PathValue(key)
	id, err := uuid.Parse(raw)
	if err != nil || id.Version() != 4 || id.String() != raw {
		return "", contract.ErrInvalidRequest
	}
	return raw, nil
}
func (s *Server) read(w http.ResponseWriter, r *http.Request) {
	a, err := s.guard.Authorize(r, s.sessions, false)
	if err != nil {
		s.problem(w, err)
		return
	}
	id, err := s.resourceID(r, "accountId")
	if err != nil {
		s.problem(w, err)
		return
	}
	var out generated.Account
	err = s.reads.WithinAccountRead(r.Context(), a.Principal, func(ctx context.Context) error {
		v, e := s.service.Read(ctx, a.Principal, id)
		if e != nil {
			return e
		}
		out, e = s.accountDTO(v)
		return e
	})
	if err != nil {
		s.problem(w, err)
		return
	}
	s.write(w, 200, out)
}
