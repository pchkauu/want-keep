package reconciliation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/application"
	application "github.com/pchkauu/want-keep/backend/internal/reconciliation/application"
	reconciliation "github.com/pchkauu/want-keep/backend/internal/reconciliation/domain"
)

func (s *Server) execute(w http.ResponseWriter, r *http.Request, access identity.Access, resourceID string, input any, apply func(context.Context) (command.Result, error)) {
	if len(r.Header.Values("Idempotency-Key")) != 1 {
		s.problem(w, contract.ErrInvalidRequest)
		return
	}
	payload, err := json.Marshal(struct {
		ResourceID string `json:"resourceId"`
		Input      any    `json:"input"`
	}{ResourceID: resourceID, Input: input})
	if err != nil {
		s.problem(w, err)
		return
	}
	hash := sha256.Sum256(payload)
	request := commands.Request{ID: r.Header.Get("Idempotency-Key"), Kind: "reconciliations.resolve", PayloadHash: hex.EncodeToString(hash[:])}
	result, err := s.mutations.Execute(r.Context(), access, request, apply)
	if err != nil {
		s.problem(w, err)
		return
	}
	err = s.reads.WithinFinancialRead(r.Context(), access.Principal, func(ctx context.Context) error {
		var queryErr error
		result, queryErr = s.commands.Read(ctx, access.Principal, result.ID(), s.now())
		if errors.Is(queryErr, command.ErrCommandExpired) {
			return nil
		}
		return queryErr
	})
	if err != nil {
		s.problem(w, err)
		return
	}
	if errors.Is(result.RequireDetail(access.Principal, s.now()), command.ErrCommandExpired) {
		snapshot := result.Snapshot()
		outcome := command.Outcome{CommandID: snapshot.ID, Status: snapshot.Status, Result: snapshot.Result, FailureCode: snapshot.ErrorCode, CurrentRevision: snapshot.CurrentRevision}
		out, convertErr := s.boundary.ExpiredCommandToDTO(uuid.NewString(), &outcome)
		if convertErr != nil {
			s.problem(w, convertErr)
			return
		}
		s.write(w, http.StatusGone, out)
		return
	}
	out, err := s.boundary.CommandToDTO(result, uuid.NewString())
	if err != nil {
		s.problem(w, err)
		return
	}
	s.write(w, http.StatusAccepted, out)
}

func (s *Server) resolve(w http.ResponseWriter, r *http.Request) {
	access, err := s.guard.Authorize(r, s.sessions, true)
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
	var input generated.ReconciliationResolutionInput
	if err = s.decode(r, "ReconciliationResolutionInput", &input); err != nil {
		s.problem(w, err)
		return
	}
	components := make([]reconciliation.ComponentName, len(input.Components))
	for index, component := range input.Components {
		components[index] = reconciliation.ComponentName(component)
	}
	request := application.ResolutionInput{ExpectedRevision: uint64(input.ExpectedRevision), Reason: input.Reason, Components: components}
	s.execute(w, r, access, id, input, func(ctx context.Context) (command.Result, error) {
		return s.service.Resolve(ctx, access.Principal, id, request)
	})
}
