package domain

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"math"
	"time"

	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

var (
	ErrUnauthorized        = errors.New("unauthorized")
	ErrAttempt             = errors.New("authentication attempt rejected")
	ErrFreshAuthentication = errors.New("fresh authentication required")
	ErrLastPasskey         = errors.New("last passkey cannot be revoked")
	ErrBootstrap           = errors.New("bootstrap unavailable")
	ErrRateLimited         = errors.New("authentication rate limited")
	ErrCounter             = errors.New("authenticator counter rejected")
	ErrUnavailable         = errors.New("identity storage unavailable")
)

const (
	CeremonyLifetime  = 5 * time.Minute
	RecoveryLifetime  = 10 * time.Minute
	BootstrapLifetime = 30 * time.Minute
	SessionLifetime   = 12 * time.Hour
	IdleLifetime      = 30 * time.Minute
	FreshLifetime     = 5 * time.Minute
)

type Purpose string

const (
	Login            Purpose = "login"
	Reauthentication Purpose = "reauthentication"
	Bootstrap        Purpose = "bootstrap"
	AddPasskey       Purpose = "add_passkey"
	Recovery         Purpose = "recovery"
	RecoveryGrant    Purpose = "recovery_grant"
)

// Tokens are random bearer secrets. Their digests, never their plaintext, are persisted.
type Token string

func NewToken(bytes int) (Token, error) {
	if bytes != 16 && bytes != 32 {
		return "", ErrAttempt
	}
	b := make([]byte, bytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return Token(base64.RawURLEncoding.EncodeToString(b)), nil
}
func (t Token) Valid(bytes int) bool {
	b, err := base64.RawURLEncoding.DecodeString(string(t))
	return err == nil && len(b) == bytes && base64.RawURLEncoding.EncodeToString(b) == string(t)
}
func (t Token) Hash() string {
	d := sha256.Sum256([]byte(t))
	return hex.EncodeToString(d[:])
}
func (t Token) CSRF() string {
	h := hmac.New(sha256.New, []byte(t))
	_, _ = h.Write([]byte("want-keep/session-csrf/v1"))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

type Profile struct {
	UserID                       household.UserID
	HouseholdID                  household.HouseholdID
	MembershipID                 household.MembershipID
	Name, Locale, ReportingAsset string
	Handle                       []byte
	Generation                   int64
}

func (p Profile) Recovered() (Profile, error) {
	if p.Generation < 1 || p.Generation == math.MaxInt64 {
		return Profile{}, ErrAttempt
	}
	p.Generation++
	return p, nil
}

type Credential struct {
	ID                                                                                        string
	UserID                                                                                    household.UserID
	Name, RPID                                                                                string
	RawID, PublicKey, AAGUID, AttestationObject, AttestationClientData, AttestationClientHash []byte
	AttestationFormat, AttestationType                                                        string
	AttestationAlgorithm                                                                      int64
	Transports                                                                                []string
	SignCount                                                                                 uint32
	BackupEligible, BackupState, UserVerified                                                 bool
	CreatedAt                                                                                 time.Time
	Revoked                                                                                   bool
}

type Session struct {
	ID, TokenHash, CredentialID, Name          string
	UserID                                     household.UserID
	CreatedAt, AuthenticatedAt, LastActivityAt time.Time
	Revoked                                    bool
}

func (s Session) RequireActive(now time.Time) error {
	if s.ID == "" || s.Revoked || now.Before(s.CreatedAt) || !now.Before(s.CreatedAt.Add(SessionLifetime)) || !now.Before(s.LastActivityAt.Add(IdleLifetime)) {
		return ErrUnauthorized
	}
	return nil
}
func (s Session) RequireFresh(now time.Time) error {
	if err := s.RequireActive(now); err != nil {
		return err
	}
	if now.Before(s.AuthenticatedAt) || !now.Before(s.AuthenticatedAt.Add(FreshLifetime)) {
		return ErrFreshAuthentication
	}
	return nil
}
func (s Session) Activity(now time.Time) (Session, error) {
	if err := s.RequireActive(now); err != nil {
		return Session{}, err
	}
	s.LastActivityAt = now
	return s, nil
}

type Setup struct {
	Name, HouseholdName, Timezone, Locale, ReportingAsset string
	UserID                                                household.UserID
	HouseholdID                                           household.HouseholdID
	MembershipID                                          household.MembershipID
	Handle                                                []byte
}
type Attempt struct {
	ID, BrowserHash, TokenHash, SessionHash, Challenge, RPID, Origin, GrantID string
	Purpose                                                                   Purpose
	UserID                                                                    household.UserID
	Generation                                                                int64
	CreatedAt, ExpiresAt                                                      time.Time
	Consumed                                                                  bool
	Setup                                                                     *Setup
}

func (a Attempt) Require(browserHash string, now time.Time, purpose Purpose) error {
	if a.ID == "" || a.Consumed || a.BrowserHash != browserHash || a.Purpose != purpose || now.Before(a.CreatedAt) || !now.Before(a.ExpiresAt) {
		return ErrAttempt
	}
	return nil
}

type BootstrapState struct {
	TokenHash   string
	ExpiresAt   time.Time
	Initialized bool
}

func (b BootstrapState) Require(token Token, now time.Time) error {
	if b.Initialized || !token.Valid(32) || b.TokenHash != token.Hash() || !now.Before(b.ExpiresAt) {
		return ErrBootstrap
	}
	return nil
}
