package accounts

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/application"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

type page struct {
	access      identity.Access
	kind, after string
	limit       int
}

func (p page) cursor(id string) string {
	h := hmac.New(sha256.New, []byte(p.access.Token))
	_, _ = h.Write([]byte(p.kind + "/" + string(p.access.Principal.HouseholdID()) + "/" + string(p.access.Principal.UserID()) + "/" + id))
	return id + "." + base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
func (s *Server) page(r *http.Request, kind string) (page, error) {
	a, err := s.guard.Authorize(r, s.sessions, false)
	if err != nil {
		return page{}, err
	}
	p := page{access: a, kind: kind, limit: 50}
	for key, v := range r.URL.Query() {
		if (key != "limit" && key != "cursor") || len(v) != 1 {
			return p, contract.ErrInvalidRequest
		}
	}
	if value := r.URL.Query().Get("limit"); value != "" {
		p.limit, err = strconv.Atoi(value)
		if err != nil || p.limit < 1 || p.limit > 100 {
			return p, contract.ErrInvalidRequest
		}
	}
	if value := r.URL.Query().Get("cursor"); value != "" {
		id, _, ok := strings.Cut(value, ".")
		parsed, err := uuid.Parse(id)
		if !ok || err != nil || parsed.Version() != 4 || parsed.String() != id || !hmac.Equal([]byte(value), []byte(p.cursor(id))) {
			return p, contract.ErrInvalidRequest
		}
		p.after = id
	}
	return p, nil
}
func (s *Server) pageQuality() (generated.DataQuality, error) {
	c, _ := reporting.NewCoverage(reporting.Complete, nil)
	return s.quality(c, reporting.Fresh)
}
func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	p, err := s.page(r, "accounts")
	if err != nil {
		s.problem(w, err)
		return
	}
	out := generated.AccountPage{Items: []generated.Account{}}
	out.Quality, err = s.pageQuality()
	if err != nil {
		s.problem(w, err)
		return
	}
	err = s.reads.WithinFinancialRead(r.Context(), p.access.Principal, func(ctx context.Context) error {
		views, next, e := s.service.List(ctx, p.access.Principal, p.after, p.limit)
		if e != nil {
			return e
		}
		for _, v := range views {
			dto, e := s.accountDTO(v)
			if e != nil {
				return e
			}
			out.Items = append(out.Items, dto)
		}
		if next != "" {
			cursor := p.cursor(next)
			out.NextCursor = &cursor
		}
		return nil
	})
	if err != nil {
		s.problem(w, err)
		return
	}
	s.write(w, 200, out)
}
