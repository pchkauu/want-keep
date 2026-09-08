package categories

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"

	"github.com/google/uuid"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	catalog "github.com/pchkauu/want-keep/backend/internal/categories/application"
	category "github.com/pchkauu/want-keep/backend/internal/categories/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/security"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	access "github.com/pchkauu/want-keep/backend/internal/identity/domain"
)

type Sessions interface {
	security.Sessions
	commands.SessionTransactions
}

type ReadTransactions interface {
	WithinFinancialRead(context.Context, household.Principal, func(context.Context) error) error
}

type Server struct {
	service   *catalog.Service
	mutations *commands.Authenticated
	commands  *commands.Queries
	sessions  Sessions
	reads     ReadTransactions
	guard     *security.Guard
	boundary  *contract.Boundary
	now       func() calendar.Instant
	mux       *http.ServeMux
}

func New(service *catalog.Service, executor *commands.Executor, queries *commands.Queries, sessions Sessions, reads ReadTransactions, config security.Config, now func() calendar.Instant) (*Server, error) {
	if service == nil || executor == nil || queries == nil || sessions == nil || reads == nil || now == nil {
		return nil, category.ErrInvalidCategory
	}
	guard, err := security.New(config)
	if err != nil {
		return nil, err
	}
	boundary, err := contract.NewBoundary()
	if err != nil {
		return nil, err
	}
	s := &Server{service: service, mutations: commands.NewAuthenticated(executor, sessions), commands: queries, sessions: sessions, reads: reads, guard: guard, boundary: boundary, now: now, mux: http.NewServeMux()}
	s.mux.HandleFunc("GET /api/v1/categories", s.listCategories)
	s.mux.HandleFunc("POST /api/v1/categories", s.createCategory)
	s.mux.HandleFunc("POST /api/v1/categories/{categoryId}", s.changeCategory)
	s.mux.HandleFunc("GET /api/v1/merchants", s.listMerchants)
	s.mux.HandleFunc("POST /api/v1/merchants", s.createMerchant)
	s.mux.HandleFunc("POST /api/v1/merchants/{merchantId}", s.changeMerchant)
	return s, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := s.guard.Check(w, r); err != nil {
		s.problem(w, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 128*1024)
	s.mux.ServeHTTP(w, r)
}

func (s *Server) decode(r *http.Request, schema string, out any) error {
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		return contract.ErrInvalidRequest
	}
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return contract.ErrInvalidRequest
	}
	return s.boundary.Decode(schema, data, out)
}

func (s *Server) write(w http.ResponseWriter, status int, value any) {
	data, err := json.Marshal(value)
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
	switch {
	case errors.Is(err, category.ErrInvalidCategory):
		response.Status, response.Body.Code, response.Body.Message = 400, "invalid_request", "Check the catalog fields."
	case errors.Is(err, category.ErrNotFound):
		response.Status, response.Body.Code, response.Body.Message = 404, "not_found", "Catalog resource not found."
	case errors.Is(err, access.ErrUnauthorized):
		response.Status, response.Body.Code, response.Body.Message = 401, "unauthorized", "Sign in to continue."
	case errors.Is(err, access.ErrAttempt):
		response.Status, response.Body.Code, response.Body.Message = 400, "invalid_request", "Check the request fields."
	}
	s.write(w, response.Status, response.Body)
}

func resourceID(r *http.Request, name string) (string, error) {
	raw := r.PathValue(name)
	id, err := uuid.Parse(raw)
	if err != nil || id.Version() != 4 || id.String() != raw {
		return "", contract.ErrInvalidRequest
	}
	return raw, nil
}

func (s *Server) commandResponse(w http.ResponseWriter, p household.Principal, value command.Command) {
	if errors.Is(value.RequireDetail(p, s.now()), command.ErrCommandExpired) {
		snapshot := value.Snapshot()
		out, err := s.boundary.ExpiredCommandToDTO(uuid.NewString(), &command.Outcome{CommandID: snapshot.ID, Status: snapshot.Status, Result: snapshot.Result, FailureCode: snapshot.ErrorCode})
		if err != nil {
			s.problem(w, err)
			return
		}
		s.write(w, 410, out)
		return
	}
	out, err := s.boundary.CommandToDTO(value, uuid.NewString())
	if err != nil {
		s.problem(w, err)
		return
	}
	s.write(w, 202, out)
}
