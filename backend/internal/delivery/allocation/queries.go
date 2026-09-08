package allocation

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
	allocationdomain "github.com/pchkauu/want-keep/backend/internal/allocation/domain"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/application"
)

type page struct {
	access identity.Access
	after  string
	limit  int
}

func (p page) cursor(id string) string {
	payload, _ := json.Marshal([]string{string(p.access.Principal.HouseholdID()), string(p.access.Principal.UserID()), id})
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	hash := hmac.New(sha256.New, []byte(p.access.Token))
	_, _ = hash.Write([]byte("allocation-rules/" + encoded))
	return encoded + "." + base64.RawURLEncoding.EncodeToString(hash.Sum(nil))
}

func (s *Server) page(r *http.Request) (page, error) {
	access, err := s.guard.Authorize(r, s.sessions, false)
	if err != nil {
		return page{}, err
	}
	p := page{access: access, limit: 50}
	values := r.URL.Query()
	for key, values := range values {
		if len(values) != 1 || key != "limit" && key != "cursor" {
			return p, contract.ErrInvalidRequest
		}
	}
	if value := values.Get("limit"); value != "" {
		p.limit, err = strconv.Atoi(value)
		if err != nil || p.limit < 1 || p.limit > 100 {
			return p, contract.ErrInvalidRequest
		}
	}
	if value := values.Get("cursor"); value != "" {
		encoded, _, ok := strings.Cut(value, ".")
		raw, decodeErr := base64.RawURLEncoding.DecodeString(encoded)
		if !ok || decodeErr != nil {
			return p, contract.ErrInvalidRequest
		}
		var scope []string
		if json.Unmarshal(raw, &scope) != nil || len(scope) != 3 {
			return p, contract.ErrInvalidRequest
		}
		p.after = scope[2]
		id, parseErr := uuid.Parse(p.after)
		if parseErr != nil || id.Version() != 4 || id.String() != p.after || !hmac.Equal([]byte(value), []byte(p.cursor(p.after))) {
			return p, contract.ErrInvalidRequest
		}
	}
	return p, nil
}

func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	page, err := s.page(r)
	if err != nil {
		s.problem(w, err)
		return
	}
	out := generated.AllocationRulePage{Items: []generated.AllocationRule{}}
	err = s.reads.WithinFinancialRead(r.Context(), page.access.Principal, func(ctx context.Context) error {
		items, next, readErr := s.service.Rules(ctx, page.access.Principal, page.after, page.limit)
		if readErr != nil {
			return readErr
		}
		for _, item := range items {
			out.Items = append(out.Items, ruleDTO(item))
		}
		if next != "" {
			value := page.cursor(next)
			out.NextCursor = &value
		}
		return nil
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
	id, err := resourceID(r, "ruleId")
	if err != nil {
		s.problem(w, err)
		return
	}
	var out generated.AllocationRule
	err = s.reads.WithinFinancialRead(r.Context(), access.Principal, func(ctx context.Context) error {
		value, readErr := s.service.Rule(ctx, access.Principal, id)
		if readErr == nil {
			out = ruleDTO(value)
		}
		return readErr
	})
	if err != nil {
		s.problem(w, err)
		return
	}
	s.write(w, http.StatusOK, out)
}

func (s *Server) preview(w http.ResponseWriter, r *http.Request) {
	access, err := s.guard.Authorize(r, s.sessions, false)
	if err != nil {
		s.problem(w, err)
		return
	}
	var input generated.AllocationRulePreviewInput
	if err = s.decode(r, "AllocationRulePreviewInput", &input); err != nil {
		s.problem(w, err)
		return
	}
	merchantID, categoryID := "", ""
	if input.MerchantId != nil {
		merchantID = *input.MerchantId
	}
	if input.CategoryId != nil {
		categoryID = *input.CategoryId
	}
	var resolution allocationdomain.Resolution
	err = s.reads.WithinFinancialRead(r.Context(), access.Principal, func(ctx context.Context) error {
		var readErr error
		resolution, readErr = s.service.Preview(ctx, access.Principal, merchantID, categoryID)
		return readErr
	})
	if err != nil {
		s.problem(w, err)
		return
	}
	s.write(w, http.StatusOK, previewDTO(resolution))
}

func ruleDTO(rule allocationdomain.Rule) generated.AllocationRule {
	out := generated.AllocationRule{Id: rule.ID, Revision: int64(rule.Revision), Priority: rule.Priority, State: generated.AllocationRuleState(rule.State), Condition: generated.AllocationRuleCondition{}, Shares: []generated.AllocationRuleShare{}}
	if rule.Condition.MerchantID != "" {
		out.Condition.MerchantId = &rule.Condition.MerchantID
	}
	if rule.Condition.CategoryID != "" {
		out.Condition.CategoryId = &rule.Condition.CategoryID
	}
	for _, share := range rule.Shares {
		out.Shares = append(out.Shares, generated.AllocationRuleShare{MemberId: string(share.MemberID), Share: share.Value})
	}
	return out
}

func previewDTO(value allocationdomain.Resolution) generated.AllocationRulePreview {
	out := generated.AllocationRulePreview{State: generated.AllocationRulePreviewState(value.State), Reason: generated.AllocationRulePreviewReason(value.Reason), Shares: []generated.AllocationRuleShare{}, Rules: []generated.AllocationRuleReference{}}
	if value.State == "resolved" {
		out.Reason = "matched"
	}
	for _, share := range value.Shares {
		out.Shares = append(out.Shares, generated.AllocationRuleShare{MemberId: string(share.MemberID), Share: share.Value})
	}
	for _, rule := range value.Rules {
		out.Rules = append(out.Rules, generated.AllocationRuleReference{RuleId: rule.ID, Revision: int64(rule.Revision)})
	}
	return out
}
