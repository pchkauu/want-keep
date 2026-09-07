package webauthn

import (
	"bytes"
	"encoding/json"
	"net/url"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/protocol/webauthncose"
	library "github.com/go-webauthn/webauthn/webauthn"
	application "github.com/pchkauu/want-keep/backend/internal/identity/application"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/domain"
)

type Verifier struct {
	client       *library.WebAuthn
	rpID, origin string
}

func New(rpID, origin string) (*Verifier, error) {
	u, err := url.Parse(origin)
	if err != nil || u.Hostname() != rpID || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "https" && !(u.Scheme == "http" && rpID == "localhost")) {
		return nil, identity.ErrAttempt
	}
	client, err := library.New(&library.Config{RPID: rpID, RPDisplayName: "Want Keep", RPOrigins: []string{origin}, AttestationPreference: protocol.PreferNoAttestation, AuthenticatorSelection: protocol.AuthenticatorSelection{ResidentKey: protocol.ResidentKeyRequirementRequired, UserVerification: protocol.VerificationRequired}, RPAllowCrossOrigin: false})
	if err != nil {
		return nil, identity.ErrAttempt
	}
	return &Verifier{client: client, rpID: rpID, origin: origin}, nil
}

type user struct {
	profile     identity.Profile
	credentials []library.Credential
}

func (u user) WebAuthnID() []byte                        { return u.profile.Handle }
func (u user) WebAuthnName() string                      { return string(u.profile.UserID) }
func (u user) WebAuthnDisplayName() string               { return u.profile.Name }
func (u user) WebAuthnCredentials() []library.Credential { return u.credentials }

type registrationResponse struct {
	ID       string           `json:"id"`
	RawID    string           `json:"rawId"`
	Type     string           `json:"type"`
	Response registrationData `json:"response"`
}
type registrationData struct {
	ClientDataJSON    string   `json:"clientDataJSON"`
	AttestationObject string   `json:"attestationObject"`
	Transports        []string `json:"transports"`
}
type assertionResponse struct {
	ID       string        `json:"id"`
	RawID    string        `json:"rawId"`
	Type     string        `json:"type"`
	Response assertionData `json:"response"`
}
type assertionData struct {
	ClientDataJSON    string `json:"clientDataJSON"`
	AuthenticatorData string `json:"authenticatorData"`
	Signature         string `json:"signature"`
	UserHandle        string `json:"userHandle"`
}

func (v *Verifier) Register(p identity.Profile, a identity.Attempt, input application.Registration) (identity.Credential, error) {
	if a.RPID != v.rpID || a.Origin != v.origin {
		return identity.Credential{}, identity.ErrAttempt
	}
	data, err := json.Marshal(registrationResponse{ID: input.ID, RawID: input.RawID, Type: "public-key", Response: registrationData{ClientDataJSON: input.ClientDataJSON, AttestationObject: input.AttestationObject, Transports: input.Transports}})
	if err != nil {
		return identity.Credential{}, identity.ErrAttempt
	}
	parsed, err := protocol.ParseCredentialCreationResponseBytes(data)
	if err != nil {
		return identity.Credential{}, identity.ErrAttempt
	}
	if parsed.Response.AttestationObject.AuthData.Flags.HasExtensions() {
		return identity.Credential{}, identity.ErrAttempt
	}
	// Application checks the persisted expiry with its injected clock; the library owns cryptographic verification.
	session := library.SessionData{Challenge: a.Challenge, RelyingPartyID: a.RPID, Origin: a.Origin, UserID: p.Handle, UserVerification: protocol.VerificationRequired, CredParams: []protocol.CredentialParameter{{Type: protocol.PublicKeyCredentialType, Algorithm: webauthncose.AlgES256}, {Type: protocol.PublicKeyCredentialType, Algorithm: webauthncose.AlgRS256}}}
	c, err := v.client.CreateCredential(user{profile: p}, session, parsed)
	if err != nil {
		return identity.Credential{}, identity.ErrAttempt
	}
	return v.fromLibrary(*c), nil
}
func (v *Verifier) Authenticate(p identity.Profile, a identity.Attempt, c identity.Credential, input application.Assertion) (identity.Credential, error) {
	if a.RPID != v.rpID || a.Origin != v.origin || c.RPID != v.rpID || c.Revoked {
		return identity.Credential{}, identity.ErrAttempt
	}
	data, err := json.Marshal(assertionResponse{ID: input.ID, RawID: input.RawID, Type: "public-key", Response: assertionData{ClientDataJSON: input.ClientDataJSON, AuthenticatorData: input.AuthenticatorData, Signature: input.Signature, UserHandle: input.UserHandle}})
	if err != nil {
		return identity.Credential{}, identity.ErrAttempt
	}
	parsed, err := protocol.ParseCredentialRequestResponseBytes(data)
	if err != nil {
		return identity.Credential{}, identity.ErrAttempt
	}
	if parsed.Response.AuthenticatorData.Flags.HasExtensions() {
		return identity.Credential{}, identity.ErrAttempt
	}
	session := library.SessionData{Challenge: a.Challenge, RelyingPartyID: a.RPID, Origin: a.Origin, UserVerification: protocol.VerificationRequired}
	_, updated, err := v.client.ValidatePasskeyLogin(func(rawID, handle []byte) (library.User, error) {
		if !bytes.Equal(rawID, c.RawID) || !bytes.Equal(handle, p.Handle) {
			return nil, identity.ErrUnauthorized
		}
		return user{profile: p, credentials: []library.Credential{v.toLibrary(c)}}, nil
	}, session, parsed)
	if err != nil {
		return identity.Credential{}, identity.ErrAttempt
	}
	if updated.Authenticator.CloneWarning {
		return identity.Credential{}, identity.ErrCounter
	}
	c.SignCount = updated.Authenticator.SignCount
	c.BackupState = updated.Flags.BackupState
	c.UserVerified = updated.Flags.UserVerified
	return c, nil
}
func (v *Verifier) fromLibrary(c library.Credential) identity.Credential {
	transports := make([]string, len(c.Transport))
	for i, t := range c.Transport {
		transports[i] = string(t)
	}
	return identity.Credential{RPID: v.rpID, RawID: c.ID, PublicKey: c.PublicKey, AAGUID: c.Authenticator.AAGUID, AttestationObject: c.Attestation.Object, AttestationClientData: c.Attestation.ClientDataJSON, AttestationClientHash: c.Attestation.ClientDataHash, AttestationFormat: c.AttestationFormat, AttestationType: c.AttestationType, AttestationAlgorithm: c.Attestation.PublicKeyAlgorithm, Transports: transports, SignCount: c.Authenticator.SignCount, BackupEligible: c.Flags.BackupEligible, BackupState: c.Flags.BackupState, UserVerified: c.Flags.UserVerified}
}
func (v *Verifier) toLibrary(c identity.Credential) library.Credential {
	transports := make([]protocol.AuthenticatorTransport, len(c.Transports))
	for i, t := range c.Transports {
		transports[i] = protocol.AuthenticatorTransport(t)
	}
	return library.Credential{ID: c.RawID, PublicKey: c.PublicKey, Transport: transports, Flags: library.CredentialFlags{BackupEligible: c.BackupEligible, BackupState: c.BackupState, UserVerified: c.UserVerified, UserPresent: true}, Authenticator: library.Authenticator{AAGUID: c.AAGUID, SignCount: c.SignCount}, AttestationFormat: c.AttestationFormat, AttestationType: c.AttestationType, Attestation: library.CredentialAttestation{ClientDataJSON: c.AttestationClientData, ClientDataHash: c.AttestationClientHash, PublicKeyAlgorithm: c.AttestationAlgorithm, Object: c.AttestationObject}}
}
