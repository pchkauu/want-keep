package identity

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/security"
	application "github.com/pchkauu/want-keep/backend/internal/identity/application"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/domain"
)

const SessionCookie = security.SessionCookie
const browserCookie = "want_keep_ceremony"

type Config = security.Config
type Server struct {
	service  *application.Service
	boundary *contract.Boundary
	config   Config
	guard    *security.Guard
	mux      *http.ServeMux
}

func New(service *application.Service, c Config) (*Server, error) {
	guard, err := security.New(c)
	if err != nil {
		return nil, err
	}
	if service == nil {
		return nil, identity.ErrAttempt
	}
	b, err := contract.NewBoundary()
	if err != nil {
		return nil, err
	}
	s := &Server{service: service, boundary: b, config: c, guard: guard, mux: http.NewServeMux()}
	s.mux.HandleFunc("POST /api/v1/auth/login/options", s.loginOptions)
	s.mux.HandleFunc("POST /api/v1/auth/login/verify", s.loginVerify)
	s.mux.HandleFunc("POST /api/v1/household/bootstrap", s.bootstrap)
	s.mux.HandleFunc("POST /api/v1/auth/enrollment/options", s.enrollmentOptions)
	s.mux.HandleFunc("POST /api/v1/auth/enrollment/verify", s.enrollmentVerify)
	s.mux.HandleFunc("POST /api/v1/auth/recovery", s.recovery)
	s.mux.HandleFunc("POST /api/v1/auth/logout", s.logout)
	s.mux.HandleFunc("POST /api/v1/auth/session/activity", s.activity)
	s.mux.HandleFunc("GET /api/v1/me", s.me)
	s.mux.HandleFunc("GET /api/v1/security/passkeys", s.passkeys)
	s.mux.HandleFunc("GET /api/v1/security/sessions", s.sessions)
	s.mux.HandleFunc("DELETE /api/v1/security/passkeys/{credentialId}", s.revokePasskey)
	s.mux.HandleFunc("DELETE /api/v1/security/sessions/{credentialId}", s.revokeSession)
	s.mux.HandleFunc("POST /api/v1/security/recovery-codes", s.replaceCodes)
	return s, nil
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := s.guard.Check(w, r); err != nil {
		s.problem(w, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 128*1024)
	s.mux.ServeHTTP(w, r)
}
func (s *Server) request(w http.ResponseWriter, r *http.Request, begin bool) (application.RequestContext, error) {
	var result application.RequestContext
	if cookie, err := r.Cookie(SessionCookie); err == nil {
		result.Session = identity.Token(cookie.Value)
	}
	if cookie, err := r.Cookie(browserCookie); err == nil {
		result.Browser = identity.Token(cookie.Value)
	}
	if begin {
		if !result.Browser.Valid(32) {
			token, err := identity.NewToken(32)
			if err != nil {
				return result, err
			}
			result.Browser = token
		}
		// Preserve live bindings, but let every new attempt use its full server-side lifetime.
		http.SetCookie(w, &http.Cookie{Name: browserCookie, Value: string(result.Browser), Path: "/", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: 1800})
	}
	source, err := s.guard.Source(r)
	if err != nil {
		return result, err
	}
	result.Source = source
	return result, nil
}
func (s *Server) authorize(w http.ResponseWriter, r *http.Request, mutation bool) (application.Access, bool) {
	access, err := s.guard.Authorize(r, s.service, mutation)
	if err != nil {
		s.problem(w, err)
		return application.Access{}, false
	}
	return access, true
}
func (s *Server) decode(w http.ResponseWriter, r *http.Request, name string, target any) bool {
	if r.Header.Get("Content-Type") != "application/json" {
		s.problem(w, contract.ErrInvalidRequest)
		return false
	}
	b, err := io.ReadAll(r.Body)
	if err == nil {
		err = s.boundary.Decode(name, b, target)
	}
	if err != nil {
		s.problem(w, contract.ErrInvalidRequest)
		return false
	}
	return true
}
func (s *Server) write(w http.ResponseWriter, status int, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		s.problem(w, identity.ErrUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(data)
}
func (s *Server) cookie(w http.ResponseWriter, a application.Access) {
	http.SetCookie(w, &http.Cookie{Name: SessionCookie, Value: string(a.Token), Path: "/", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: int(identity.SessionLifetime / time.Second)})
}
func (s *Server) clearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: SessionCookie, Path: "/", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: -1})
}
func (s *Server) problem(w http.ResponseWriter, err error) {
	status, code, message := 503, generated.ErrorCode("service_unavailable"), "Access could not be confirmed. Try again later."
	switch {
	case errors.Is(err, identity.ErrUnauthorized):
		status, code, message = 401, "unauthorized", "Sign in with your own passkey or use another recovery code."
	case errors.Is(err, identity.ErrAttempt), errors.Is(err, identity.ErrCounter):
		status, code, message = 400, "authentication_attempt_rejected", "This sign-in attempt was rejected. Start a new attempt."
	case errors.Is(err, identity.ErrFreshAuthentication):
		status, code, message = 403, "reauthentication_required", "Confirm your own passkey before changing access."
	case errors.Is(err, identity.ErrLastPasskey):
		status, code, message = 409, "last_passkey", "Add another passkey before removing this one."
	case errors.Is(err, identity.ErrBootstrap):
		status, code, message = 409, "bootstrap_unavailable", "Initial setup is unavailable. Use the sign-in page."
	case errors.Is(err, identity.ErrRateLimited):
		status, code, message = 429, "rate_limited", "Too many attempts. Try again later."
		w.Header().Set("Retry-After", "900")
	case errors.Is(err, contract.ErrInvalidRequest):
		status, code, message = 400, "invalid_request", "Check the submitted fields."
	}
	body := generated.APIError{Version: "1", Code: code, Message: message, Violations: []generated.FieldViolation{}, Retryable: status == 429 || status == 503, CorrelationId: uuid.NewString()}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
