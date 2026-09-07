package ledger

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
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
)

func (s *Server) execute(w http.ResponseWriter, r *http.Request, a identity.Access, kind string, input any, apply func(context.Context) (command.Result, error)) {
	if len(r.Header.Values("Idempotency-Key")) != 1 {
		s.problem(w, contract.ErrInvalidRequest)
		return
	}
	payload, err := json.Marshal(input)
	if err != nil {
		s.problem(w, err)
		return
	}
	hash := sha256.Sum256(payload)
	request := commands.Request{ID: r.Header.Get("Idempotency-Key"), Kind: kind, PayloadHash: hex.EncodeToString(hash[:])}
	c, err := s.mutations.Execute(r.Context(), a, request, apply)
	if err != nil {
		s.problem(w, err)
		return
	}
	err = s.reads.WithinFinancialRead(r.Context(), a.Principal, func(ctx context.Context) error {
		var e error
		c, e = s.commands.Read(ctx, a.Principal, c.ID(), s.now())
		if errors.Is(e, command.ErrCommandExpired) {
			return nil
		}
		return e
	})
	if err != nil {
		s.problem(w, err)
		return
	}
	if errors.Is(c.RequireDetail(a.Principal, s.now()), command.ErrCommandExpired) {
		x := c.Snapshot()
		outcome := command.Outcome{CommandID: x.ID, Status: x.Status, Result: x.Result, FailureCode: x.ErrorCode}
		out, e := s.boundary.ExpiredCommandToDTO(uuid.NewString(), &outcome)
		if e != nil {
			s.problem(w, e)
			return
		}
		s.write(w, 410, out)
		return
	}
	out, err := s.boundary.CommandToDTO(c, uuid.NewString())
	if err != nil {
		s.problem(w, err)
		return
	}
	s.write(w, 202, out)
}
func (s *Server) create(w http.ResponseWriter, r *http.Request) {
	a, err := s.guard.Authorize(r, s.sessions, true)
	if err != nil {
		s.problem(w, err)
		return
	}
	var in generated.TransactionCreate
	if err = s.decode(r, "TransactionCreate", &in); err != nil {
		s.problem(w, err)
		return
	}
	input, err := s.createInput(in)
	if err != nil {
		s.problem(w, err)
		return
	}
	if input.Unsupported {
		s.problem(w, ledger.ErrFeatureUnavailable)
		return
	}
	s.execute(w, r, a, "transactions.create", in, func(ctx context.Context) (command.Result, error) { return s.service.Create(ctx, a.Principal, input) })
}
func (s *Server) transfer(w http.ResponseWriter, r *http.Request) {
	a, err := s.guard.Authorize(r, s.sessions, true)
	if err != nil {
		s.problem(w, err)
		return
	}
	var in generated.TransferCreate
	if err = s.decode(r, "TransferCreate", &in); err != nil {
		s.problem(w, err)
		return
	}
	input, err := s.transferInput(in)
	if err != nil {
		s.problem(w, err)
		return
	}
	if input.Existing {
		s.execute(w, r, a, "transactions.links", in, func(ctx context.Context) (command.Result, error) {
			return s.matching.LinkTransfer(ctx, a.Principal, input, s.existingMembers(in.ExistingTransactions))
		})
		return
	}
	s.execute(w, r, a, "transactions.transfer", in, func(ctx context.Context) (command.Result, error) { return s.service.Transfer(ctx, a.Principal, input) })
}
