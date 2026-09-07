package domain

import (
	"errors"
	"time"

	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

var ErrSecretAccess = errors.New("connection authorization required")

type SecretPurpose string

const (
	APICredentials SecretPurpose = "api_credentials"
	OAuthTokens    SecretPurpose = "oauth_tokens"
	BrowserSession SecretPurpose = "browser_session"
)

func (p SecretPurpose) Valid() bool {
	return p == APICredentials || p == OAuthTokens || p == BrowserSession
}

type SecretGrant struct {
	ID, ConnectionID, SessionHash string
	HouseholdID                   household.HouseholdID
	OwnerID                       household.UserID
	Generation                    uint64
	Purpose                       SecretPurpose
	ExpiresAt                     time.Time
	Consumed, Revoked             bool
}

func (g SecretGrant) Require(p household.Principal, sessionHash string, purpose SecretPurpose, generation uint64, now time.Time) error {
	if !purpose.Valid() || g.ID == "" || g.ConnectionID == "" || g.Consumed || g.Revoked || g.Purpose != purpose || g.Generation != generation || p.UserID() != g.OwnerID || p.HouseholdID() != g.HouseholdID || g.SessionHash != sessionHash || !now.Before(g.ExpiresAt) {
		return ErrSecretAccess
	}
	return nil
}

type SecretReference struct {
	HouseholdID          household.HouseholdID
	ConnectionID         string
	Purpose              SecretPurpose
	Generation, Revision uint64
}

func (r SecretReference) Validate() error {
	if r.HouseholdID == "" || r.ConnectionID == "" || !r.Purpose.Valid() || r.Generation < 1 || r.Generation > 9007199254740991 || r.Revision < 1 || r.Revision > 9007199254740991 {
		return ErrSecretAccess
	}
	return nil
}
