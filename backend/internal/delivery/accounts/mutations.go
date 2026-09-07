package accounts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"

	accounts "github.com/pchkauu/want-keep/backend/internal/accounts/application"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/application"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func (s *Server) execute(w http.ResponseWriter, r *http.Request, a identity.Access, kind, id string, input any, apply func(context.Context) (command.Result, error)) {
	if len(r.Header.Values("Idempotency-Key")) != 1 {
		s.problem(w, contract.ErrInvalidRequest)
		return
	}
	payload, err := json.Marshal(struct {
		Resource string `json:"resource"`
		Input    any    `json:"input"`
	}{id, input})
	if err != nil {
		s.problem(w, err)
		return
	}
	hash := sha256.Sum256(payload)
	request := commands.Request{ID: r.Header.Get("Idempotency-Key"), Kind: kind, PayloadHash: hex.EncodeToString(hash[:])}
	if _, err = s.executor.Register(r.Context(), a.Principal, request); err != nil {
		s.problem(w, err)
		return
	}
	var c command.Command
	err = s.sessions.WithinSession(r.Context(), a.Token, func(ctx context.Context, current identity.Access) error {
		if current.Principal != a.Principal {
			return contract.ErrInvalidRequest
		}
		var e error
		c, e = s.executor.ExecuteRegistered(ctx, a.Principal, request, apply)
		return e
	})
	if err != nil {
		s.problem(w, err)
		return
	}
	err = s.reads.WithinAccountRead(r.Context(), a.Principal, func(ctx context.Context) error {
		var e error
		c, e = s.queries.Read(ctx, a.Principal, c.ID(), s.now())
		if errors.Is(e, command.ErrCommandExpired) {
			return nil
		}
		return e
	})
	if err != nil {
		s.problem(w, err)
		return
	}
	s.commandResponse(w, a.Principal, c, 202)
}
func (s *Server) create(w http.ResponseWriter, r *http.Request) {
	a, err := s.guard.Authorize(r, s.sessions, true)
	if err != nil {
		s.problem(w, err)
		return
	}
	var input generated.AccountCreate
	if err = s.decode(r, "AccountCreate", &input); err != nil {
		s.problem(w, err)
		return
	}
	ownership, err := s.ownershipFromDTO(input.Ownership, a.Principal)
	if err != nil {
		s.problem(w, err)
		return
	}
	date, err := calendar.ParseDate(input.OpeningDate)
	if err != nil {
		s.problem(w, err)
		return
	}
	balance, err := (contract.MoneyConverter{}).FromDTO(input.OpeningBalance)
	if err != nil {
		s.problem(w, err)
		return
	}
	domainInput := accounts.CreateInput{Name: input.Name, Ownership: ownership, Asset: balance.Asset(), Date: date, Balance: balance}
	domainInput.Asset = money.Asset(input.Asset)
	s.execute(w, r, a, "accounts.create", "", input, func(ctx context.Context) (command.Result, error) {
		return s.service.Create(ctx, a.Principal, domainInput)
	})
}
func (s *Server) ownership(w http.ResponseWriter, r *http.Request) {
	a, err := s.guard.Authorize(r, s.sessions, true)
	if err != nil {
		s.problem(w, err)
		return
	}
	id, err := s.resourceID(r, "accountId")
	if err != nil {
		s.problem(w, err)
		return
	}
	var input generated.AccountOwnershipChange
	if err = s.decode(r, "AccountOwnershipChange", &input); err != nil {
		s.problem(w, err)
		return
	}
	next, err := s.ownershipFromDTO(input.Ownership, a.Principal)
	if err != nil {
		s.problem(w, err)
		return
	}
	s.execute(w, r, a, "accounts.ownership", id, input, func(ctx context.Context) (command.Result, error) {
		return s.service.ChangeOwnership(ctx, a.Principal, id, uint64(input.ExpectedRevision), next, input.Reason)
	})
}
func (s *Server) correct(w http.ResponseWriter, r *http.Request) {
	a, err := s.guard.Authorize(r, s.sessions, true)
	if err != nil {
		s.problem(w, err)
		return
	}
	id, err := s.resourceID(r, "accountId")
	if err != nil {
		s.problem(w, err)
		return
	}
	var input generated.OpeningCorrection
	if err = s.decode(r, "OpeningCorrection", &input); err != nil {
		s.problem(w, err)
		return
	}
	date, err := calendar.ParseDate(input.Date)
	if err != nil {
		s.problem(w, err)
		return
	}
	values, err := s.amountsFromDTO(input.Balances)
	if err != nil {
		s.problem(w, err)
		return
	}
	c := accounts.Correction{ExpectedRevision: uint64(input.ExpectedRevision), Date: date, Amounts: values, Reason: input.Reason}
	s.execute(w, r, a, "accounts.opening_correction", id, input, func(ctx context.Context) (command.Result, error) {
		return s.service.CorrectOpening(ctx, a.Principal, id, c)
	})
}
