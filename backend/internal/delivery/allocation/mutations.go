package allocation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	allocationapp "github.com/pchkauu/want-keep/backend/internal/allocation/application"
	allocationdomain "github.com/pchkauu/want-keep/backend/internal/allocation/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/application"
)

func (s *Server) execute(w http.ResponseWriter, r *http.Request, access identity.Access, kind, resource string, input any, apply func(context.Context) (command.Result, error)) {
	if len(r.Header.Values("Idempotency-Key")) != 1 {
		s.problem(w, contract.ErrInvalidRequest)
		return
	}
	payload, err := json.Marshal(struct {
		Resource string `json:"resource"`
		Input    any    `json:"input"`
	}{resource, input})
	if err != nil {
		s.problem(w, err)
		return
	}
	hash := sha256.Sum256(payload)
	registered, err := s.mutations.Execute(r.Context(), access, commands.Request{ID: r.Header.Get("Idempotency-Key"), Kind: kind, PayloadHash: hex.EncodeToString(hash[:])}, apply)
	if err != nil {
		s.problem(w, err)
		return
	}
	err = s.reads.WithinFinancialRead(r.Context(), access.Principal, func(ctx context.Context) error {
		var readErr error
		registered, readErr = s.commands.Read(ctx, access.Principal, registered.ID(), s.now())
		if errors.Is(readErr, command.ErrCommandExpired) {
			return nil
		}
		return readErr
	})
	if err != nil {
		s.problem(w, err)
		return
	}
	s.commandResponse(w, access.Principal, registered)
}

func (s *Server) create(w http.ResponseWriter, r *http.Request) {
	access, err := s.guard.Authorize(r, s.sessions, true)
	if err != nil {
		s.problem(w, err)
		return
	}
	var input generated.AllocationRuleCreateInput
	if err = s.decode(r, "AllocationRuleCreateInput", &input); err != nil {
		s.problem(w, contract.ErrInvalidRequest)
		return
	}
	domainInput := createRuleInput(input)
	s.execute(w, r, access, "allocation_rules.create", "", input, func(ctx context.Context) (command.Result, error) {
		return s.service.CreateRule(ctx, access.Principal, domainInput)
	})
}

func (s *Server) change(w http.ResponseWriter, r *http.Request) {
	access, err := s.guard.Authorize(r, s.sessions, true)
	if err != nil {
		s.problem(w, err)
		return
	}
	id, err := resourceID(r, "ruleId")
	if err != nil {
		s.problem(w, err)
		return
	}
	var input generated.AllocationRuleChangeInput
	if err = s.decode(r, "AllocationRuleChangeInput", &input); err != nil {
		s.problem(w, contract.ErrInvalidRequest)
		return
	}
	domainInput := changeRuleInput(input)
	s.execute(w, r, access, "allocation_rules.change", id, input, func(ctx context.Context) (command.Result, error) {
		return s.service.ChangeRule(ctx, access.Principal, id, uint64(input.ExpectedRevision), domainInput)
	})
}

func createRuleInput(input generated.AllocationRuleCreateInput) allocationapp.RuleInput {
	result := allocationapp.RuleInput{Priority: input.Priority, State: allocationdomain.State(input.State)}
	if input.Condition.MerchantId != nil {
		result.Condition.MerchantID = *input.Condition.MerchantId
	}
	if input.Condition.CategoryId != nil {
		result.Condition.CategoryID = *input.Condition.CategoryId
	}
	for _, share := range input.Shares {
		result.Shares = append(result.Shares, allocationdomain.Share{MemberID: householdMembershipID(share.MemberId), Value: share.Share})
	}
	return result
}

func changeRuleInput(input generated.AllocationRuleChangeInput) allocationapp.RuleInput {
	result := allocationapp.RuleInput{Priority: input.Priority, State: allocationdomain.State(input.State)}
	if input.Condition.MerchantId != nil {
		result.Condition.MerchantID = *input.Condition.MerchantId
	}
	if input.Condition.CategoryId != nil {
		result.Condition.CategoryID = *input.Condition.CategoryId
	}
	for _, share := range input.Shares {
		result.Shares = append(result.Shares, allocationdomain.Share{MemberID: householdMembershipID(share.MemberId), Value: share.Share})
	}
	return result
}

func resourceID(r *http.Request, name string) (string, error) {
	raw := r.PathValue(name)
	id, err := uuid.Parse(raw)
	if err != nil || id.Version() != 4 || id.String() != raw {
		return "", contract.ErrInvalidRequest
	}
	return raw, nil
}

func householdMembershipID(value string) household.MembershipID { return household.MembershipID(value) }
