package categories

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
	category "github.com/pchkauu/want-keep/backend/internal/categories/domain"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/application"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

type page struct {
	access identity.Access
	filter category.Filter
	kind   string
	after  string
	limit  int
}

func (p page) cursor(id string) string {
	payload, _ := json.Marshal([]string{p.kind, string(p.access.Principal.HouseholdID()), string(p.access.Principal.UserID()), string(p.filter.State), p.filter.ParentID, p.filter.Search, id})
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	hash := hmac.New(sha256.New, []byte(p.access.Token))
	_, _ = hash.Write([]byte("catalog/" + encoded))
	return encoded + "." + base64.RawURLEncoding.EncodeToString(hash.Sum(nil))
}

func (s *Server) page(r *http.Request, kind string) (page, error) {
	access, err := s.guard.Authorize(r, s.sessions, false)
	if err != nil {
		return page{}, err
	}
	p := page{access: access, kind: kind, limit: 50}
	values := r.URL.Query()
	for key, value := range values {
		if len(value) != 1 {
			return p, contract.ErrInvalidRequest
		}
		if key != "limit" && key != "cursor" && key != "state" && key != "search" && (kind != "categories" || key != "parentId") {
			return p, contract.ErrInvalidRequest
		}
	}
	p.filter = category.Filter{State: category.State(values.Get("state")), ParentID: values.Get("parentId"), Search: values.Get("search")}
	if kind != "categories" && p.filter.ParentID != "" {
		return p, contract.ErrInvalidRequest
	}
	if p.filter.ParentID != "" {
		id, parseErr := uuid.Parse(p.filter.ParentID)
		if parseErr != nil || id.Version() != 4 || id.String() != p.filter.ParentID {
			return p, contract.ErrInvalidRequest
		}
	}
	if value := values.Get("limit"); value != "" {
		p.limit, err = strconv.Atoi(value)
		if err != nil || p.limit < 1 || p.limit > 100 {
			return p, contract.ErrInvalidRequest
		}
	}
	if err = p.filter.Validate(); err != nil {
		return p, err
	}
	if value := values.Get("cursor"); value != "" {
		encoded, _, ok := strings.Cut(value, ".")
		raw, decodeErr := base64.RawURLEncoding.DecodeString(encoded)
		if !ok || decodeErr != nil {
			return p, contract.ErrInvalidRequest
		}
		var scope []string
		if json.Unmarshal(raw, &scope) != nil || len(scope) != 7 {
			return p, contract.ErrInvalidRequest
		}
		p.after = scope[6]
		id, parseErr := uuid.Parse(p.after)
		if parseErr != nil || id.Version() != 4 || id.String() != p.after || !hmac.Equal([]byte(value), []byte(p.cursor(p.after))) {
			return p, contract.ErrInvalidRequest
		}
	}
	return p, nil
}

func (s *Server) quality() (generated.DataQuality, error) {
	coverage, _ := reporting.NewCoverage(reporting.Complete, nil)
	value, err := s.boundary.CoverageToDTO(coverage)
	return generated.DataQuality{Coverage: value, Freshness: "fresh"}, err
}

func (s *Server) listCategories(w http.ResponseWriter, r *http.Request) {
	p, err := s.page(r, "categories")
	if err != nil {
		s.problem(w, err)
		return
	}
	out := generated.CategoryPage{Items: []generated.Category{}}
	out.Quality, err = s.quality()
	if err == nil {
		err = s.reads.WithinFinancialRead(r.Context(), p.access.Principal, func(ctx context.Context) error {
			items, next, readErr := s.service.Categories(ctx, p.access.Principal, p.filter, p.after, p.limit)
			if readErr != nil {
				return readErr
			}
			for _, item := range items {
				out.Items = append(out.Items, categoryDTO(item))
			}
			if next != "" {
				cursor := p.cursor(next)
				out.NextCursor = &cursor
			}
			return nil
		})
	}
	if err != nil {
		s.problem(w, err)
		return
	}
	s.write(w, 200, out)
}

func (s *Server) listMerchants(w http.ResponseWriter, r *http.Request) {
	p, err := s.page(r, "merchants")
	if err != nil {
		s.problem(w, err)
		return
	}
	out := generated.MerchantPage{Items: []generated.Merchant{}}
	out.Quality, err = s.quality()
	if err == nil {
		err = s.reads.WithinFinancialRead(r.Context(), p.access.Principal, func(ctx context.Context) error {
			items, next, readErr := s.service.Merchants(ctx, p.access.Principal, p.filter, p.after, p.limit)
			if readErr != nil {
				return readErr
			}
			for _, item := range items {
				out.Items = append(out.Items, merchantDTO(item))
			}
			if next != "" {
				cursor := p.cursor(next)
				out.NextCursor = &cursor
			}
			return nil
		})
	}
	if err != nil {
		s.problem(w, err)
		return
	}
	s.write(w, 200, out)
}

func categoryDTO(item category.Category) generated.Category {
	out := generated.Category{Id: item.ID, Revision: int64(item.Revision), Name: item.DisplayName(), State: generated.CategoryState(item.State), Origin: generated.CategoryOrigin(item.Origin)}
	if item.ParentID != "" {
		out.ParentId = &item.ParentID
	}
	if item.Origin == category.Starter {
		out.StarterKey = &item.Key
		out.Labels = &generated.LocalizedCategoryName{Ru: item.NameRU, En: item.NameEN}
	}
	if item.CustomName != "" {
		out.CustomName = &item.CustomName
	}
	return out
}

func merchantDTO(item category.Merchant) generated.Merchant {
	out := generated.Merchant{Id: item.ID, Revision: int64(item.Revision), Name: item.Name, State: generated.MerchantState(item.State), Aliases: []generated.MerchantAlias{}}
	for _, alias := range item.Aliases {
		out.Aliases = append(out.Aliases, generated.MerchantAlias{Id: alias.ID, Name: alias.Name, State: generated.MerchantAliasState(alias.State), Origin: generated.MerchantAliasOrigin(alias.Origin)})
	}
	return out
}
