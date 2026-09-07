package files

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	domain "github.com/pchkauu/want-keep/backend/internal/attachments/domain"
	"github.com/pchkauu/want-keep/backend/internal/privacy/cryptobox"
)

func TestPrivateObjectsAndConcurrentPublish(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	keys := filepath.Join(t.TempDir(), "keys")
	if err := cryptobox.Generate(keys, "attachments"); err != nil {
		t.Fatal(err)
	}
	ring, err := cryptobox.Load(keys, "attachments")
	if err != nil {
		t.Fatal(err)
	}
	s, err := Open(dir, ring)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	data := []byte("synthetic original")
	o := domain.Object{HouseholdID: "family-A", AttachmentID: "a9117226-4e2c-4898-bffb-27da7a2c3810", ObjectID: "0b9e88a1-cc8a-4e9a-a29e-bca8e855179a", Purpose: "original", Size: int64(len(data)), Hash: domain.Hash(data)}
	var wg sync.WaitGroup
	failures := make(chan error, 8)
	for range 8 {
		wg.Go(func() { failures <- s.Put(context.Background(), o, data) })
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	got, err := s.Read(context.Background(), o)
	if err != nil || !bytes.Equal(got, data) {
		t.Fatal("round trip", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatal("unexpected object inventory", err)
	}
	stored, err := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	if err != nil || bytes.Contains(stored, data) {
		t.Fatal("plaintext on disk", err)
	}
	changed := o
	changed.Hash = domain.Hash([]byte("changed"))
	changed.Size = 7
	if s.Put(context.Background(), changed, []byte("changed")) == nil {
		t.Fatal("immutable object overwritten")
	}
	changed = o
	changed.ObjectID = "../../outside"
	if _, err = s.Read(context.Background(), changed); err == nil {
		t.Fatal("traversal accepted")
	}
	name, _, _ := s.location(o)
	if err = os.Rename(filepath.Join(dir, name), filepath.Join(dir, "original")); err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink(filepath.Join(dir, "original"), filepath.Join(dir, name)); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Read(context.Background(), o); err == nil {
		t.Fatal("symlink read")
	}
	if err = s.Put(context.Background(), o, data); err == nil {
		t.Fatal("symlink publish")
	}
}
func TestCleanupOnlyAbandonedPendingFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	keys := filepath.Join(t.TempDir(), "keys")
	if cryptobox.Generate(keys, "attachments") != nil {
		t.Fatal("generate")
	}
	ring, _ := cryptobox.Load(keys, "attachments")
	s, err := Open(dir, ring)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	now := time.Now()
	pending := "pending-" + strings.Repeat("a", 32)
	published := strings.Repeat("b", 64) + ".blob"
	for _, name := range []string{pending, published} {
		path := filepath.Join(dir, name)
		if os.WriteFile(path, []byte("synthetic ciphertext"), 0o600) != nil {
			t.Fatal("write")
		}
		old := now.Add(-25 * time.Hour)
		if os.Chtimes(path, old, old) != nil {
			t.Fatal("time")
		}
	}
	n, err := s.CleanupPending(context.Background(), now)
	if err != nil || n != 1 {
		t.Fatal(n, err)
	}
	if _, err = os.Stat(filepath.Join(dir, published)); err != nil {
		t.Fatal("published file removed")
	}
}
