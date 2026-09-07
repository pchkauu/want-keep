package identity

import (
	"encoding/base64"
	"time"

	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	application "github.com/pchkauu/want-keep/backend/internal/identity/application"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/domain"
)

func (s *Server) meDTO(a application.Access) generated.Me {
	p := a.Profile
	return generated.Me{User: generated.User{Id: string(p.UserID), Name: p.Name}, Membership: generated.Membership{Id: string(p.MembershipID), HouseholdId: string(p.HouseholdID), UserId: string(p.UserID), Role: "member", Status: "active"}, Preferences: generated.Preferences{Locale: generated.Locale(p.Locale), ReportingAsset: generated.Asset(p.ReportingAsset), PushEnabled: false, DecorativeEffectsEnabled: true}, CsrfToken: a.Token.CSRF(), Session: s.sessionDTO(a.Session, a.Session.ID)}
}
func (s *Server) sessionDTO(x identity.Session, current string) generated.Session {
	idle := x.LastActivityAt.Add(identity.IdleLifetime)
	absolute := x.CreatedAt.Add(identity.SessionLifetime)
	if absolute.Before(idle) {
		idle = absolute
	}
	return generated.Session{Id: x.ID, Name: x.Name, Current: x.ID == current, CreatedAt: x.CreatedAt.UTC().Format(time.RFC3339Nano), AuthenticatedAt: x.AuthenticatedAt.UTC().Format(time.RFC3339Nano), ExpiresAt: absolute.UTC().Format(time.RFC3339Nano), IdleExpiresAt: idle.UTC().Format(time.RFC3339Nano)}
}
func (s *Server) enrollmentDTO(e application.Enrollment) generated.EnrollmentOptions {
	exclude := make([]string, len(e.ExcludeIDs))
	for i, id := range e.ExcludeIDs {
		exclude[i] = base64.RawURLEncoding.EncodeToString(id)
	}
	return generated.EnrollmentOptions{AttemptId: e.Attempt.ID, Challenge: e.Attempt.Challenge, RpId: e.Attempt.RPID, RpName: "Want Keep", UserId: base64.RawURLEncoding.EncodeToString(e.Profile.Handle), UserName: string(e.Profile.UserID), UserDisplayName: e.Profile.Name, Algorithms: []int{-7, -257}, Timeout: 300000, ResidentKey: "required", UserVerification: "required", Attestation: "none", ExcludeCredentials: exclude}
}
