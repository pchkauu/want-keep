package identity

import (
	"crypto/hmac"
	"net/http"

	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	application "github.com/pchkauu/want-keep/backend/internal/identity/application"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/domain"
)

func (s *Server) loginOptions(w http.ResponseWriter, r *http.Request) {
	var input generated.LoginOptionsInput
	if !s.decode(w, r, "LoginOptionsInput", &input) {
		return
	}
	purpose := identity.Login
	if input.Purpose != nil {
		purpose = identity.Purpose(*input.Purpose)
	}
	if purpose == identity.Reauthentication {
		if _, ok := s.authorize(w, r, true); !ok {
			return
		}
	}
	request, err := s.request(w, r, true)
	if err != nil {
		s.problem(w, err)
		return
	}
	if purpose == identity.Login && request.Session.Valid(32) {
		if _, err = s.service.Me(r.Context(), request.Session); err == identity.ErrUnauthorized {
			s.clearCookie(w)
			request.Session = ""
		} else if err != nil {
			s.problem(w, err)
			return
		}
	}
	a, err := s.service.BeginLogin(r.Context(), request, purpose)
	if err != nil {
		s.problem(w, err)
		return
	}
	s.write(w, 200, generated.LoginOptions{AttemptId: a.ID, Challenge: a.Challenge, RpId: a.RPID, Timeout: 300000, UserVerification: "required"})
}
func (s *Server) loginVerify(w http.ResponseWriter, r *http.Request) {
	var input generated.LoginVerifyInput
	if !s.decode(w, r, "LoginVerifyInput", &input) {
		return
	}
	request, err := s.request(w, r, false)
	if err != nil {
		s.problem(w, err)
		return
	}
	// If there is an existing session, any replacement/reauthentication also proves its CSRF binding.
	if request.Session.Valid(32) && !hmac.Equal([]byte(request.Session.CSRF()), []byte(r.Header.Get("X-CSRF-Token"))) {
		s.problem(w, identity.ErrUnauthorized)
		return
	}
	c := input.Credential
	handle := ""
	if c.UserHandle != nil {
		handle = *c.UserHandle
	}
	access, err := s.service.CompleteLogin(r.Context(), request, input.AttemptId, application.Assertion{ID: c.Id, RawID: c.RawId, ClientDataJSON: c.ClientDataJSON, AuthenticatorData: c.AuthenticatorData, Signature: c.Signature, UserHandle: handle})
	if err != nil {
		s.problem(w, err)
		return
	}
	s.cookie(w, access)
	s.write(w, 200, s.meDTO(access))
}
func (s *Server) bootstrap(w http.ResponseWriter, r *http.Request) {
	var input generated.SetupInput
	if !s.decode(w, r, "SetupInput", &input) {
		return
	}
	request, err := s.request(w, r, true)
	if err != nil {
		s.problem(w, err)
		return
	}
	enrollment, err := s.service.BeginBootstrap(r.Context(), request, identity.Token(*input.BootstrapToken), identity.Setup{Name: input.Name, HouseholdName: input.HouseholdName, Timezone: input.Timezone, Locale: string(input.Locale), ReportingAsset: string(input.ReportingAsset)})
	if err != nil {
		s.problem(w, err)
		return
	}
	s.write(w, 200, s.enrollmentDTO(enrollment))
}
func (s *Server) enrollmentOptions(w http.ResponseWriter, r *http.Request) {
	var input generated.EnrollmentInput
	if !s.decode(w, r, "EnrollmentInput", &input) {
		return
	}
	if input.Purpose == "add_passkey" {
		if _, ok := s.authorize(w, r, true); !ok {
			return
		}
	}
	request, err := s.request(w, r, false)
	if err != nil {
		s.problem(w, err)
		return
	}
	token := ""
	if input.AuthorizationToken != nil {
		token = *input.AuthorizationToken
	}
	enrollment, err := s.service.BeginEnrollment(r.Context(), request, identity.Purpose(input.Purpose), identity.Token(token))
	if err != nil {
		s.problem(w, err)
		return
	}
	s.write(w, 200, s.enrollmentDTO(enrollment))
}
func (s *Server) enrollmentVerify(w http.ResponseWriter, r *http.Request) {
	var input generated.EnrollmentVerifyInput
	if !s.decode(w, r, "EnrollmentVerifyInput", &input) {
		return
	}
	request, err := s.request(w, r, false)
	if err != nil {
		s.problem(w, err)
		return
	}
	if request.Session.Valid(32) && !hmac.Equal([]byte(request.Session.CSRF()), []byte(r.Header.Get("X-CSRF-Token"))) {
		s.problem(w, identity.ErrUnauthorized)
		return
	}
	c := input.Credential
	transports := make([]string, len(c.Transports))
	for i, t := range c.Transports {
		transports[i] = string(t)
	}
	result, err := s.service.CompleteEnrollment(r.Context(), request, input.AttemptId, input.Name, application.Registration{ID: c.Id, RawID: c.RawId, ClientDataJSON: c.ClientDataJSON, AttestationObject: c.AttestationObject, Transports: transports})
	if err != nil {
		s.problem(w, err)
		return
	}
	if result.Purpose == identity.AddPasskey {
		s.write(w, 200, generated.AdditionalPasskeyResult{Purpose: "add_passkey", Me: s.meDTO(result.Access)})
		return
	}
	s.cookie(w, result.Access)
	s.write(w, 200, generated.InitialEnrollmentResult{Purpose: generated.InitialEnrollmentResultPurpose(result.Purpose), Me: s.meDTO(result.Access), RecoveryCodes: generated.RecoveryCodes{Codes: result.Codes}})
}
func (s *Server) recovery(w http.ResponseWriter, r *http.Request) {
	var input generated.RecoveryInput
	if !s.decode(w, r, "RecoveryInput", &input) {
		return
	}
	request, err := s.request(w, r, true)
	if err != nil {
		s.problem(w, err)
		return
	}
	// An expired cookie must not prevent recovery after a browser reload without its old CSRF token.
	if request.Session.Valid(32) {
		if _, err = s.service.Me(r.Context(), request.Session); err == identity.ErrUnauthorized {
			s.clearCookie(w)
			request.Session = ""
		} else if err != nil {
			s.problem(w, err)
			return
		}
	}
	result, err := s.service.Recover(r.Context(), request, identity.Token(*input.RecoveryCode))
	if err != nil {
		s.problem(w, err)
		return
	}
	s.write(w, 200, generated.RecoveryAttempt{AttemptId: result.Attempt.ID, EnrollmentToken: string(result.Token), ExpiresAt: result.Attempt.ExpiresAt.UTC().Format("2006-01-02T15:04:05.999999999Z")})
}
func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	a, ok := s.authorize(w, r, false)
	if ok {
		s.write(w, 200, s.meDTO(a))
	}
}
func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	a, ok := s.authorize(w, r, true)
	if !ok {
		return
	}
	if err := s.service.Logout(r.Context(), a.Token); err != nil {
		s.problem(w, err)
		return
	}
	s.clearCookie(w)
	w.WriteHeader(204)
}
func (s *Server) activity(w http.ResponseWriter, r *http.Request) {
	var input generated.EmptyInput
	if !s.decode(w, r, "EmptyInput", &input) {
		return
	}
	a, ok := s.authorize(w, r, true)
	if !ok {
		return
	}
	if err := s.service.Activity(r.Context(), a.Token); err != nil {
		s.problem(w, err)
		return
	}
	w.WriteHeader(204)
}
