package allocation

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"

	"github.com/google/uuid"
	allocationapp "github.com/pchkauu/want-keep/backend/internal/allocation/application"
	allocationdomain "github.com/pchkauu/want-keep/backend/internal/allocation/domain"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/security"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/domain"
)

type Sessions interface {
	security.Sessions
	commands.SessionTransactions
}

type ReadTransactions interface {
	WithinFinancialRead(context.Context, household.Principal, func(context.Context) error) error
}

type Server struct {
	service   *allocationapp.Service
	mutations *commands.Authenticated
	commands  *commands.Queries
	sessions  Sessions
	reads     ReadTransactions
	guard     *security.Guard
	boundary  *contract.Boundary
	now       func() calendar.Instant
	mux       *http.ServeMux
}

func New(service *allocationapp.Service, executor *commands.Executor, queries *commands.Queries, sessions Sessions, reads ReadTransactions, config security.Config, now func() calendar.Instant) (*Server, error) {
	if service == nil || executor == nil || queries == nil || sessions == nil || reads == nil || now == nil {
		return nil, allocationdomain.ErrInvalidRule
	}
	guard, err := security.New(config)
	if err != nil {
		return nil, err
	}
	boundary, err := contract.NewBoundary()
	if err != nil {
		return nil, err
	}
	server := &Server{service: service, mutations: commands.NewAuthenticated(executor, sessions), commands: queries, sessions: sessions, reads: reads, guard: guard, boundary: boundary, now: now, mux: http.NewServeMux()}
	server.mux.HandleFunc("GET /api/v1/allocation-rules", server.list)
	server.mux.HandleFunc("POST /api/v1/allocation-rules", server.create)
	server.mux.HandleFunc("POST /api/v1/allocation-rules/preview", server.preview)
	server.mux.HandleFunc("GET /api/v1/allocation-rules/{ruleId}", server.read)
	server.mux.HandleFunc("POST /api/v1/allocation-rules/{ruleId}", server.change)
	return server, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := s.guard.Check(w, r); err != nil {
		s.problem(w, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 128*1024)
	s.mux.ServeHTTP(w, r)
}

func (s *Server) decode(r *http.Request, schema string, value any) error {
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		return contract.ErrInvalidRequest
	}
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return contract.ErrInvalidRequest
	}
	return s.boundary.Decode(schema, data, value)
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
	var rejection commands.Rejection
	if errors.As(err, &rejection) {
		if rejection.CurrentRevision != 0 {
			revision := generated.Revision(rejection.CurrentRevision)
			response.Body.CurrentRevision = &revision
		}
		switch rejection.Code {
		case "not_found":
			response.Status, response.Body.Code, response.Body.Message = 404, "not_found", "Allocation rule not found."
		case "version_conflict", "decision_conflict":
			response.Status, response.Body.Code, response.Body.Message = 409, generated.ErrorCode(rejection.Code), "The allocation rule changed. Refresh it before trying again."
		case "invalid_allocation", "no_change", "invalid_request":
			response.Status, response.Body.Code, response.Body.Message = 422, generated.ErrorCode(rejection.Code), "Check the allocation rule fields and revision."
		case "forbidden":
			response.Status, response.Body.Code, response.Body.Message = 403, "forbidden", "You cannot access this resource."
		}
	}
	switch {
	case errors.Is(err, allocationdomain.ErrInvalidRule):
		response.Status, response.Body.Code, response.Body.Message = 422, "invalid_allocation", "Check the rule conditions, members and exact shares."
	case errors.Is(err, allocationdomain.ErrRuleNotFound):
		response.Status, response.Body.Code, response.Body.Message = 404, "not_found", "Allocation rule not found."
	case errors.Is(err, identity.ErrUnauthorized):
		response.Status, response.Body.Code, response.Body.Message = 401, "unauthorized", "Sign in to continue."
	case errors.Is(err, identity.ErrAttempt):
		response.Status, response.Body.Code, response.Body.Message = 400, "invalid_request", "Check the request fields."
	}
	s.write(w, response.Status, response.Body)
}

func (s *Server) commandResponse(w http.ResponseWriter, principal household.Principal, value command.Command) {
	if errors.Is(value.RequireDetail(principal, s.now()), command.ErrCommandExpired) {
		snapshot := value.Snapshot()
		out, err := s.boundary.ExpiredCommandToDTO(uuid.NewString(), &command.Outcome{CommandID: snapshot.ID, Status: snapshot.Status, Result: snapshot.Result, FailureCode: snapshot.ErrorCode})
		if err != nil {
			s.problem(w, err)
			return
		}
		s.write(w, http.StatusGone, out)
		return
	}
	out, err := s.boundary.CommandToDTO(value, uuid.NewString())
	if err != nil {
		s.problem(w, err)
		return
	}
	s.write(w, http.StatusAccepted, out)
}
