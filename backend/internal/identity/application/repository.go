package application

import (
	"context"
	"time"

	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/domain"
)

type Repository interface {
	WithinIdentity(context.Context, func(context.Context) error) error
	IdentityBootstrap(context.Context) (identity.BootstrapState, error)
	SaveIdentityBootstrap(context.Context, identity.BootstrapState) error
	IdentityProfile(context.Context, household.UserID) (identity.Profile, error)
	CreateIdentityProfile(context.Context, identity.Profile) error
	SaveIdentityGeneration(context.Context, identity.Profile) error
	IdentityMembership(context.Context, identity.Profile) (household.Principal, error)
	IdentityCredential(context.Context, []byte) (identity.Credential, error)
	IdentityCredentials(context.Context, household.UserID) ([]identity.Credential, error)
	SaveIdentityCredential(context.Context, identity.Credential) error
	IdentitySession(context.Context, string) (identity.Session, error)
	IdentitySessions(context.Context, household.UserID) ([]identity.Session, error)
	SaveIdentitySession(context.Context, identity.Session) error
	IdentityAttempt(context.Context, string) (identity.Attempt, error)
	IdentityGrant(context.Context, string) (identity.Attempt, error)
	SaveIdentityAttempt(context.Context, identity.Attempt) error
	ConsumeRecoveryCode(context.Context, string) (household.UserID, error)
	ReplaceRecoveryCodes(context.Context, household.UserID, []string) error
	RevokeIdentitySubscriptions(context.Context, household.UserID, string) error
	IdentityAudit(context.Context, household.UserID, string, string, time.Time) error
	IdentityRateLimit(context.Context, string, int, time.Time) error
}

type Registration struct {
	ID, RawID, ClientDataJSON, AttestationObject string
	Transports                                   []string
}
type Assertion struct{ ID, RawID, ClientDataJSON, AuthenticatorData, Signature, UserHandle string }
type Verifier interface {
	Register(identity.Profile, identity.Attempt, Registration) (identity.Credential, error)
	Authenticate(identity.Profile, identity.Attempt, identity.Credential, Assertion) (identity.Credential, error)
}

type RequestContext struct {
	Browser, Session identity.Token
	Source           string
}
type Access struct {
	Profile   identity.Profile
	Principal household.Principal
	Session   identity.Session
	Token     identity.Token
}
type Enrollment struct {
	Attempt    identity.Attempt
	Profile    identity.Profile
	ExcludeIDs [][]byte
}
type EnrollmentResult struct {
	Access  Access
	Codes   []string
	Purpose identity.Purpose
}
