package identity

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	application "github.com/pchkauu/want-keep/backend/internal/identity/application"
)

type accessPage struct {
	after  string
	limit  int
	kind   string
	access application.Access
}

func (p accessPage) cursor(id string) string {
	h := hmac.New(sha256.New, []byte(p.access.Token))
	_, _ = h.Write([]byte("cursor/" + p.kind + "/" + string(p.access.Profile.UserID) + "/" + id))
	return id + "." + base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
func (s *Server) page(w http.ResponseWriter, r *http.Request, kind string) (accessPage, bool) {
	a, ok := s.authorize(w, r, false)
	if !ok {
		return accessPage{}, false
	}
	p := accessPage{limit: 50, kind: kind, access: a}
	for key, values := range r.URL.Query() {
		if (key != "limit" && key != "cursor") || len(values) != 1 {
			s.problem(w, contract.ErrInvalidRequest)
			return p, false
		}
	}
	if raw := r.URL.Query().Get("limit"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 1 || v > 100 {
			s.problem(w, contract.ErrInvalidRequest)
			return p, false
		}
		p.limit = v
	}
	if raw := r.URL.Query().Get("cursor"); raw != "" {
		id, _, ok := strings.Cut(raw, ".")
		if !ok || !hmac.Equal([]byte(raw), []byte(p.cursor(id))) {
			s.problem(w, contract.ErrInvalidRequest)
			return p, false
		}
		p.after = id
	}
	return p, true
}
func (s *Server) quality() generated.DataQuality {
	var coverage generated.Coverage
	_ = coverage.FromCompleteCoverage(generated.CompleteCoverage{State: "complete", Reasons: []string{}})
	return generated.DataQuality{Coverage: coverage, Freshness: "fresh"}
}
func (s *Server) passkeys(w http.ResponseWriter, r *http.Request) {
	p, ok := s.page(w, r, "passkeys")
	if !ok {
		return
	}
	keys, err := s.service.Passkeys(r.Context(), p.access.Token)
	if err != nil {
		s.problem(w, err)
		return
	}
	result := generated.PasskeyPage{Items: []generated.Passkey{}, Quality: s.quality()}
	for _, key := range keys {
		if key.ID <= p.after {
			continue
		}
		if len(result.Items) == p.limit {
			cursor := p.cursor(result.Items[len(result.Items)-1].Id)
			result.NextCursor = &cursor
			break
		}
		result.Items = append(result.Items, generated.Passkey{Id: key.ID, Name: key.Name, CreatedAt: key.CreatedAt.UTC().Format(time.RFC3339Nano)})
	}
	s.write(w, 200, result)
}
func (s *Server) sessions(w http.ResponseWriter, r *http.Request) {
	p, ok := s.page(w, r, "sessions")
	if !ok {
		return
	}
	sessions, err := s.service.Sessions(r.Context(), p.access.Token)
	if err != nil {
		s.problem(w, err)
		return
	}
	result := generated.SessionPage{Items: []generated.Session{}, Quality: s.quality()}
	for _, session := range sessions {
		if session.ID <= p.after {
			continue
		}
		if len(result.Items) == p.limit {
			cursor := p.cursor(result.Items[len(result.Items)-1].Id)
			result.NextCursor = &cursor
			break
		}
		result.Items = append(result.Items, s.sessionDTO(session, p.access.Session.ID))
	}
	s.write(w, 200, result)
}
func (s *Server) revokePasskey(w http.ResponseWriter, r *http.Request) {
	a, ok := s.authorize(w, r, true)
	if !ok {
		return
	}
	id := r.PathValue("credentialId")
	if _, err := uuid.Parse(id); err != nil {
		s.problem(w, contract.ErrInvalidRequest)
		return
	}
	if err := s.service.RevokePasskey(r.Context(), a.Token, id); err != nil {
		s.problem(w, err)
		return
	}
	if a.Session.CredentialID == id {
		s.clearCookie(w)
	}
	w.WriteHeader(204)
}
func (s *Server) revokeSession(w http.ResponseWriter, r *http.Request) {
	a, ok := s.authorize(w, r, true)
	if !ok {
		return
	}
	id := r.PathValue("credentialId")
	if _, err := uuid.Parse(id); err != nil {
		s.problem(w, contract.ErrInvalidRequest)
		return
	}
	if err := s.service.RevokeSession(r.Context(), a.Token, id); err != nil {
		s.problem(w, err)
		return
	}
	if a.Session.ID == id {
		s.clearCookie(w)
	}
	w.WriteHeader(204)
}
func (s *Server) replaceCodes(w http.ResponseWriter, r *http.Request) {
	var input generated.EmptyInput
	if !s.decode(w, r, "EmptyInput", &input) {
		return
	}
	a, ok := s.authorize(w, r, true)
	if !ok {
		return
	}
	codes, err := s.service.ReplaceCodes(r.Context(), a.Token)
	if err != nil {
		s.problem(w, err)
		return
	}
	s.write(w, 200, generated.RecoveryCodes{Codes: codes})
}
