package cryptobox

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"
	"regexp"
	"syscall"
)

var ErrUnavailable = errors.New("private storage unavailable")
var keyIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

// Keyring is an outer storage primitive. Business models carry references, never keys.
type Keyring struct {
	active string
	keys   map[string]cipher.AEAD
}
type keyFile struct {
	Version int        `json:"version"`
	Purpose string     `json:"purpose"`
	Active  string     `json:"active"`
	Keys    []keyEntry `json:"keys"`
}
type keyEntry struct {
	ID  string `json:"id"`
	Key string `json:"key"`
}
type envelope struct {
	Version int    `json:"version"`
	KeyID   string `json:"keyId"`
	Data    []byte `json:"data"`
}

func Generate(path, purpose string) error {
	if purpose != "attachments" && purpose != "connections" {
		return ErrUnavailable
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return ErrUnavailable
	}
	defer clear(key)
	id := make([]byte, 16)
	if _, err := rand.Read(id); err != nil {
		return ErrUnavailable
	}
	name := base64.RawURLEncoding.EncodeToString(id)
	data, err := json.Marshal(keyFile{Version: 1, Purpose: purpose, Active: name, Keys: []keyEntry{{ID: name, Key: base64.StdEncoding.EncodeToString(key)}}})
	if err != nil {
		return ErrUnavailable
	}
	defer clear(data)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return ErrUnavailable
	}
	defer f.Close()
	if _, err = f.Write(data); err != nil {
		return ErrUnavailable
	}
	if err = f.Sync(); err != nil {
		return ErrUnavailable
	}
	return nil
}

func Load(path, purpose string) (*Keyring, error) {
	if purpose != "attachments" && purpose != "connections" {
		return nil, ErrUnavailable
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
		return nil, ErrUnavailable
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || uint32(os.Geteuid()) != stat.Uid {
		return nil, ErrUnavailable
	}
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer f.Close()
	current, err := f.Stat()
	if err != nil || !os.SameFile(info, current) {
		return nil, ErrUnavailable
	}
	data, err := io.ReadAll(io.LimitReader(f, 65537))
	if err != nil || len(data) > 65536 {
		return nil, ErrUnavailable
	}
	defer clear(data)
	var input keyFile
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if d.Decode(&input) != nil || d.Decode(new(any)) != io.EOF || input.Version != 1 || input.Purpose != purpose || !keyIDPattern.MatchString(input.Active) || len(input.Keys) < 1 || len(input.Keys) > 32 {
		return nil, ErrUnavailable
	}
	result := &Keyring{active: input.Active, keys: make(map[string]cipher.AEAD, len(input.Keys))}
	for _, entry := range input.Keys {
		if !keyIDPattern.MatchString(entry.ID) || result.keys[entry.ID] != nil {
			return nil, ErrUnavailable
		}
		key, err := base64.StdEncoding.Strict().DecodeString(entry.Key)
		if err != nil || len(key) != 32 {
			return nil, ErrUnavailable
		}
		block, err := aes.NewCipher(key)
		clear(key)
		if err != nil {
			return nil, ErrUnavailable
		}
		aead, err := cipher.NewGCMWithRandomNonce(block)
		if err != nil {
			return nil, ErrUnavailable
		}
		result.keys[entry.ID] = aead
	}
	if result.keys[result.active] == nil {
		return nil, ErrUnavailable
	}
	return result, nil
}

func (k *Keyring) String() string   { return "[private keyring]" }
func (k *Keyring) GoString() string { return k.String() }

func (k *Keyring) Available() bool { return k != nil && k.keys[k.active] != nil }
func (k *Keyring) Seal(plain, aad []byte) ([]byte, error) {
	if !k.Available() || len(aad) == 0 {
		return nil, ErrUnavailable
	}
	data := k.keys[k.active].Seal(nil, nil, plain, k.boundAAD(k.active, aad))
	return json.Marshal(envelope{Version: 1, KeyID: k.active, Data: data})
}
func (k *Keyring) Open(data, aad []byte) ([]byte, error) {
	if !k.Available() || len(aad) == 0 {
		return nil, ErrUnavailable
	}
	var e envelope
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if d.Decode(&e) != nil || d.Decode(new(any)) != io.EOF || e.Version != 1 || k.keys[e.KeyID] == nil {
		return nil, ErrUnavailable
	}
	plain, err := k.keys[e.KeyID].Open(nil, nil, e.Data, k.boundAAD(e.KeyID, aad))
	if err != nil {
		return nil, ErrUnavailable
	}
	return plain, nil
}
func (k *Keyring) boundAAD(id string, aad []byte) []byte {
	data, _ := json.Marshal(struct {
		Version int
		KeyID   string
		Context []byte
	}{1, id, aad})
	return data
}
