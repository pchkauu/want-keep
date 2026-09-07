package cryptobox

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestKeyringIntegrityAndRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keyring")
	if err := Generate(path, "attachments"); err != nil {
		t.Fatal(err)
	}
	first, err := Load(path, "attachments")
	if err != nil {
		t.Fatal(err)
	}
	plain := []byte("synthetic-private-receipt")
	aad := []byte("household-A/attachment-A/original/1")
	a, err := first.Seal(plain, aad)
	if err != nil {
		t.Fatal(err)
	}
	b, err := first.Seal(plain, aad)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(a, b) || bytes.Contains(a, plain) {
		t.Fatal("reused nonce or plaintext ciphertext")
	}
	restarted, err := Load(path, "attachments")
	if err != nil {
		t.Fatal(err)
	}
	got, err := restarted.Open(a, aad)
	if err != nil || !bytes.Equal(got, plain) {
		t.Fatalf("restart: %v", err)
	}
	if _, err = restarted.Open(a, []byte("household-B/attachment-A/original/1")); err == nil {
		t.Fatal("foreign AAD accepted")
	}
	var envelope envelope
	if err = json.Unmarshal(a, &envelope); err != nil {
		t.Fatal(err)
	}
	envelope.Data[len(envelope.Data)-1] ^= 1
	damaged, _ := json.Marshal(envelope)
	if _, err = restarted.Open(damaged, aad); err == nil {
		t.Fatal("damaged tag accepted")
	}
	envelope.KeyID = "unknown"
	unknown, _ := json.Marshal(envelope)
	if _, err = restarted.Open(unknown, aad); err == nil {
		t.Fatal("unknown key accepted")
	}
	if Generate(path, "attachments") == nil {
		t.Fatal("key overwritten")
	}
	if _, err = Load(path, "connections"); err == nil {
		t.Fatal("cross-purpose keyring accepted")
	}
}
func TestKeyringRejectsUnsafeFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "keys")
	if err := Generate(path, "connections"); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(link, "connections"); err == nil {
		t.Fatal("symlink accepted")
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path, "connections"); err == nil {
		t.Fatal("public keyring accepted")
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"version":1,"purpose":"connections","active":"bad","keys":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path, "connections"); err == nil {
		t.Fatal("empty keyring accepted")
	}
	var missing *Keyring
	if _, err := missing.Seal([]byte("private"), []byte("context")); err == nil {
		t.Fatal("missing key fallback")
	}
}
func TestKeyringRetainsOldReadKeys(t *testing.T) {
	dir := t.TempDir()
	oldPath, newPath := filepath.Join(dir, "old"), filepath.Join(dir, "new")
	for _, path := range []string{oldPath, newPath} {
		if err := Generate(path, "attachments"); err != nil {
			t.Fatal(err)
		}
	}
	old, err := Load(oldPath, "attachments")
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := old.Seal([]byte("old document"), []byte("reference"))
	if err != nil {
		t.Fatal(err)
	}
	var prior, current keyFile
	data, _ := os.ReadFile(oldPath)
	if json.Unmarshal(data, &prior) != nil {
		t.Fatal("old key")
	}
	data, _ = os.ReadFile(newPath)
	if json.Unmarshal(data, &current) != nil {
		t.Fatal("new key")
	}
	current.Keys = append(current.Keys, prior.Keys...)
	data, _ = json.Marshal(current)
	if err = os.WriteFile(newPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	next, err := Load(newPath, "attachments")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = next.Open(sealed, []byte("reference")); err != nil {
		t.Fatal(err)
	}
	newCipher, err := next.Seal([]byte("next"), []byte("reference"))
	if err != nil {
		t.Fatal(err)
	}
	var e envelope
	if json.Unmarshal(newCipher, &e) != nil || e.KeyID != current.Active {
		t.Fatal("inactive write key")
	}
}
