package reconciliation

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"

	"github.com/google/uuid"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/security"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/domain"
	application "github.com/pchkauu/want-keep/backend/internal/reconciliation/application"
	reconciliation "github.com/pchkauu/want-keep/backend/internal/reconciliation/domain"
)

type Sessions interface {
	security.Sessions
	commands.SessionTransactions
}

type ReadTransactions interface {
	WithinFinancialRead(context.Context, household.Principal, func(context.Context) error) error
}

type Server struct {
	service   *application.Service
	mutations *commands.Authenticated
	commands  *commands.Queries
	sessions  Sessions
	reads     ReadTransactions
	guard     *security.Guard
	boundary  *contract.Boundary
	now       func() calendar.Instant
	mux       *http.ServeMux
}

func New(service *application.Service, executor *commands.Executor, commandQueries *commands.Queries, sessions Sessions, reads ReadTransactions, config security.Config, now func() calendar.Instant) (*Server, error) {
	if service == nil || executor == nil || commandQueries == nil || sessions == nil || reads == nil || now == nil {
		return nil, reconciliation.ErrInvalidReconciliation
	}
	guard, err := security.New(config)
	if err != nil {
		return nil, err
	}
	boundary, err := contract.NewBoundary()
	if err != nil {
		return nil, err
	}
	server := &Server{service: service, mutations: commands.NewAuthenticated(executor, sessions), commands: commandQueries, sessions: sessions, reads: reads, guard: guard, boundary: boundary, now: now, mux: http.NewServeMux()}
	server.mux.HandleFunc("GET /api/v1/reconciliations", server.list)
	server.mux.HandleFunc("GET /api/v1/reconciliations/{reconciliationId}", server.read)
	server.mux.HandleFunc("POST /api/v1/reconciliations/{reconciliationId}/resolve", server.resolve)
	return server, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := s.guard.Check(w, r); err != nil {
		s.problem(w, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	s.mux.ServeHTTP(w, r)
}

func (s *Server) decode(r *http.Request, name string, out any) error {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return contract.ErrInvalidRequest
	}
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return contract.ErrInvalidRequest
	}
	return s.boundary.Decode(name, data, out)
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
		switch rejection.Code {
		case "not_found":
			response.Status, response.Body.Code, response.Body.Message = 404, "not_found", "Reconciliation not found."
		case "version_conflict":
			response.Status, response.Body.Code, response.Body.Message = 409, "version_conflict", "The reconciliation changed. Refresh it before trying again."
		case "reconciliation_not_ready":
			response.Status, response.Body.Code, response.Body.Message = 409, "reconciliation_not_ready", "Complete or establish that history replay is unavailable before resolving this difference."
		case "component_not_adjustable":
			response.Status, response.Body.Code, response.Body.Message = 422, "component_not_adjustable", "Available and locked amounts must be corrected through source data or transaction status."
		case "no_change", "invalid_request":
			response.Status, response.Body.Code, response.Body.Message = 422, generated.ErrorCode(rejection.Code), "Check the reconciliation version, reason and components."
		}
	}
	switch {
	case errors.Is(err, reconciliation.ErrNotFound):
		response.Status, response.Body.Code, response.Body.Message = 404, "not_found", "Reconciliation not found."
	case errors.Is(err, reconciliation.ErrNotReady):
		response.Status, response.Body.Code, response.Body.Message = 409, "reconciliation_not_ready", "Complete or establish that history replay is unavailable before resolving this difference."
	case errors.Is(err, reconciliation.ErrComponentNotAdjustable):
		response.Status, response.Body.Code, response.Body.Message = 422, "component_not_adjustable", "Available and locked amounts must be corrected through source data or transaction status."
	case errors.Is(err, reconciliation.ErrInvalidReconciliation):
		response.Status, response.Body.Code, response.Body.Message = 422, "invalid_request", "Check the reconciliation version, reason and components."
	case errors.Is(err, identity.ErrUnauthorized):
		response.Status, response.Body.Code, response.Body.Message = 401, "unauthorized", "Sign in to continue."
	case errors.Is(err, identity.ErrAttempt):
		response.Status, response.Body.Code, response.Body.Message = 400, "invalid_request", "Check the request fields."
	}
	s.write(w, response.Status, response.Body)
}
