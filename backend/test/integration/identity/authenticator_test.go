//go:build integration

package identity_test

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"testing"

	"github.com/fxamacker/cbor/v2"
)

type authenticator struct {
	key    crypto.Signer
	id     []byte
	handle string
	count  uint32
}

func newAuthenticator(t *testing.T, handle string, useRSA bool) *authenticator {
	t.Helper()
	var key crypto.Signer
	var err error
	if useRSA {
		key, err = rsa.GenerateKey(rand.Reader, 2048)
	} else {
		key, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	}
	if err != nil {
		t.Fatal(err)
	}
	id := make([]byte, 32)
	if _, err = rand.Read(id); err != nil {
		t.Fatal(err)
	}
	return &authenticator{key: key, id: id, handle: handle}
}
func (a *authenticator) clientData(t *testing.T, kind, challenge, origin string) []byte {
	t.Helper()
	data, err := json.Marshal(map[string]any{"type": kind, "challenge": challenge, "origin": origin, "crossOrigin": false})
	if err != nil {
		t.Fatal(err)
	}
	return data
}
func (a *authenticator) authData(rp string, flags byte) []byte {
	rpHash := sha256.Sum256([]byte(rp))
	data := append([]byte{}, rpHash[:]...)
	data = append(data, flags)
	return binary.BigEndian.AppendUint32(data, a.count)
}
func (a *authenticator) registration(t *testing.T, challenge, origin, rp string, flags byte) map[string]any {
	t.Helper()
	client := a.clientData(t, "webauthn.create", challenge, origin)
	var cose map[int]any
	switch key := a.key.Public().(type) {
	case *ecdsa.PublicKey:
		cose = map[int]any{1: 2, 3: -7, -1: 1, -2: key.X.FillBytes(make([]byte, 32)), -3: key.Y.FillBytes(make([]byte, 32))}
	case *rsa.PublicKey:
		cose = map[int]any{1: 3, 3: -257, -1: key.N.Bytes(), -2: []byte{1, 0, 1}}
	}
	pub, err := cbor.Marshal(cose)
	if err != nil {
		t.Fatal(err)
	}
	data := a.authData(rp, flags|64)
	data = append(data, make([]byte, 16)...)
	data = binary.BigEndian.AppendUint16(data, uint16(len(a.id)))
	data = append(data, a.id...)
	data = append(data, pub...)
	object, err := cbor.Marshal(map[string]any{"fmt": "none", "authData": data, "attStmt": map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}
	return map[string]any{"id": base64.RawURLEncoding.EncodeToString(a.id), "rawId": base64.RawURLEncoding.EncodeToString(a.id), "type": "public-key", "clientDataJSON": base64.RawURLEncoding.EncodeToString(client), "attestationObject": base64.RawURLEncoding.EncodeToString(object), "transports": []string{"internal"}}
}
func (a *authenticator) assertion(t *testing.T, challenge, origin, rp string, flags byte) map[string]any {
	t.Helper()
	a.count++
	client := a.clientData(t, "webauthn.get", challenge, origin)
	data := a.authData(rp, flags)
	hash := sha256.Sum256(client)
	signed := append(append([]byte{}, data...), hash[:]...)
	digest := sha256.Sum256(signed)
	signature, err := a.key.Sign(rand.Reader, digest[:], crypto.SHA256)
	if err != nil {
		t.Fatal(err)
	}
	return map[string]any{"id": base64.RawURLEncoding.EncodeToString(a.id), "rawId": base64.RawURLEncoding.EncodeToString(a.id), "type": "public-key", "clientDataJSON": base64.RawURLEncoding.EncodeToString(client), "authenticatorData": base64.RawURLEncoding.EncodeToString(data), "signature": base64.RawURLEncoding.EncodeToString(signature), "userHandle": a.handle}
}
