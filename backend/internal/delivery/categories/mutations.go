package categories

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"

	catalog "github.com/pchkauu/want-keep/backend/internal/categories/application"
	category "github.com/pchkauu/want-keep/backend/internal/categories/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
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

func (s *Server) createCategory(w http.ResponseWriter, r *http.Request) {
	access, err := s.guard.Authorize(r, s.sessions, true)
	if err != nil {
		s.problem(w, err)
		return
	}
	var input generated.CategoryInput
	if err = s.decode(r, "CategoryInput", &input); err != nil {
		s.problem(w, err)
		return
	}
	domainInput := catalog.CategoryInput{Name: input.Name}
	if input.ParentId != nil {
		domainInput.ParentID = *input.ParentId
	}
	s.execute(w, r, access, "categories.create", "", input, func(ctx context.Context) (command.Result, error) {
		return s.service.CreateCategory(ctx, access.Principal, domainInput)
	})
}

func (s *Server) changeCategory(w http.ResponseWriter, r *http.Request) {
	access, err := s.guard.Authorize(r, s.sessions, true)
	if err != nil {
		s.problem(w, err)
		return
	}
	id, err := resourceID(r, "categoryId")
	if err != nil {
		s.problem(w, err)
		return
	}
	var input generated.CategoryChange
	if err = s.decode(r, "CategoryChange", &input); err != nil {
		s.problem(w, err)
		return
	}
	change := category.Change{}
	if input.NameAction != nil {
		change.NameAction = string(*input.NameAction)
		if change.NameAction == "set" && input.Name != nil {
			change.Name = *input.Name
		} else if change.NameAction == "set" || input.Name != nil {
			s.problem(w, contract.ErrInvalidRequest)
			return
		}
	}
	if input.ParentAction != nil {
		change.ParentSet = true
		if *input.ParentAction == "set" && input.ParentId != nil {
			change.ParentID = *input.ParentId
		} else if *input.ParentAction != "clear" || input.ParentId != nil {
			s.problem(w, contract.ErrInvalidRequest)
			return
		}
	} else if input.ParentId != nil {
		s.problem(w, contract.ErrInvalidRequest)
		return
	}
	if input.State != nil {
		change.State = category.State(*input.State)
	}
	s.execute(w, r, access, "categories.change", id, input, func(ctx context.Context) (command.Result, error) {
		return s.service.ChangeCategory(ctx, access.Principal, id, uint64(input.ExpectedRevision), change)
	})
}

func (s *Server) createMerchant(w http.ResponseWriter, r *http.Request) {
	access, err := s.guard.Authorize(r, s.sessions, true)
	if err != nil {
		s.problem(w, err)
		return
	}
	var input generated.MerchantInput
	if err = s.decode(r, "MerchantInput", &input); err != nil {
		s.problem(w, err)
		return
	}
	domainInput := catalog.MerchantInput{Name: input.Name, Aliases: input.Aliases}
	s.execute(w, r, access, "merchants.create", "", input, func(ctx context.Context) (command.Result, error) {
		return s.service.CreateMerchant(ctx, access.Principal, domainInput)
	})
}

func (s *Server) changeMerchant(w http.ResponseWriter, r *http.Request) {
	access, err := s.guard.Authorize(r, s.sessions, true)
	if err != nil {
		s.problem(w, err)
		return
	}
	id, err := resourceID(r, "merchantId")
	if err != nil {
		s.problem(w, err)
		return
	}
	var input generated.MerchantChange
	if err = s.decode(r, "MerchantChange", &input); err != nil {
		s.problem(w, err)
		return
	}
	change := catalog.MerchantChange{}
	if input.Name != nil {
		change.Name = *input.Name
	}
	if input.State != nil {
		change.State = category.State(*input.State)
	}
	if input.AliasesToAdd != nil {
		change.AliasesToAdd = *input.AliasesToAdd
	}
	if input.AliasIdsToArchive != nil {
		change.AliasIDsToArchive = *input.AliasIdsToArchive
	}
	s.execute(w, r, access, "merchants.change", id, input, func(ctx context.Context) (command.Result, error) {
		return s.service.ChangeMerchant(ctx, access.Principal, id, uint64(input.ExpectedRevision), change)
	})
}
