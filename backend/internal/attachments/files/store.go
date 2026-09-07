package files

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"syscall"
	"time"

	domain "github.com/pchkauu/want-keep/backend/internal/attachments/domain"
	"github.com/pchkauu/want-keep/backend/internal/privacy/cryptobox"
)

type Store struct {
	root *os.Root
	keys *cryptobox.Keyring
}

func Open(path string, keys *cryptobox.Keyring) (*Store, error) {
	if !keys.Available() {
		return nil, domain.ErrUnavailable
	}
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0o700 || info.Mode()&os.ModeSymlink != 0 {
		return nil, domain.ErrUnavailable
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || st.Uid != uint32(os.Geteuid()) {
		return nil, domain.ErrUnavailable
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, domain.ErrUnavailable
	}
	opened, err := root.Stat(".")
	if err != nil || !os.SameFile(info, opened) {
		_ = root.Close()
		return nil, domain.ErrUnavailable
	}
	return &Store{root: root, keys: keys}, nil
}
func (s *Store) Close() error {
	if s == nil {
		return nil
	}
	return s.root.Close()
}
func (s *Store) Available() bool {
	if s == nil || s.root == nil || !s.keys.Available() {
		return false
	}
	info, err := s.root.Stat(".")
	return err == nil && info.IsDir() && info.Mode().Perm() == 0o700
}

// Pending names are never stored in metadata or served. Published objects are never age-deleted.
func (s *Store) CleanupPending(ctx context.Context, now time.Time) (int, error) {
	if !s.Available() {
		return 0, domain.ErrUnavailable
	}
	dir, err := s.root.Open(".")
	if err != nil {
		return 0, domain.ErrUnavailable
	}
	defer dir.Close()
	removed := 0
	for {
		entries, err := dir.ReadDir(100)
		if err != nil && !errors.Is(err, io.EOF) {
			return removed, domain.ErrUnavailable
		}
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				return removed, err
			}
			name := entry.Name()
			suffix, ok := strings.CutPrefix(name, "pending-")
			if !ok || len(suffix) != 32 {
				continue
			}
			if _, err := hex.DecodeString(suffix); err != nil {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				return removed, domain.ErrUnavailable
			}
			if !info.Mode().IsRegular() || !info.ModTime().Before(now.Add(-24*time.Hour)) {
				continue
			}
			if err = s.root.Remove(name); err != nil && !errors.Is(err, os.ErrNotExist) {
				return removed, domain.ErrUnavailable
			}
			removed++
			if removed >= 100 {
				return removed, nil
			}
		}
		if errors.Is(err, io.EOF) {
			return removed, nil
		}
	}
}
func (s *Store) location(o domain.Object) (string, []byte, error) {
	if o.Validate() != nil {
		return "", nil, domain.ErrInvalid
	}
	aad, err := json.Marshal(struct {
		Version                                      int
		Household, Attachment, Object, Purpose, Hash string
		Size                                         int64
	}{1, string(o.HouseholdID), o.AttachmentID, o.ObjectID, o.Purpose, o.Hash, o.Size})
	if err != nil {
		return "", nil, domain.ErrInvalid
	}
	name := domain.Hash([]byte(string(o.HouseholdID)+"/"+o.ObjectID+"/"+o.Purpose)) + ".blob"
	return name, aad, nil
}

func (s *Store) Put(ctx context.Context, o domain.Object, plain []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !s.Available() {
		return domain.ErrUnavailable
	}
	name, aad, err := s.location(o)
	if err != nil {
		return err
	}
	if int64(len(plain)) != o.Size || domain.Hash(plain) != o.Hash {
		return domain.ErrInvalid
	}
	data, err := s.keys.Seal(plain, aad)
	if err != nil {
		return domain.ErrUnavailable
	}
	var nonce [16]byte
	if _, err = rand.Read(nonce[:]); err != nil {
		return domain.ErrUnavailable
	}
	temporary := "pending-" + hex.EncodeToString(nonce[:])
	f, err := s.root.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return domain.ErrUnavailable
	}
	defer s.root.Remove(temporary)
	if _, err = f.Write(data); err != nil {
		_ = f.Close()
		return domain.ErrUnavailable
	}
	if err = f.Sync(); err != nil {
		_ = f.Close()
		return domain.ErrUnavailable
	}
	if err = f.Close(); err != nil {
		return domain.ErrUnavailable
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	// Link publishes a complete immutable object without replacing an existing writer's result.
	if err = s.root.Link(temporary, name); err != nil {
		if !errors.Is(err, os.ErrExist) {
			return domain.ErrUnavailable
		}
		existing, readErr := s.Read(ctx, o)
		if readErr != nil {
			return readErr
		}
		defer clear(existing)
		if !bytes.Equal(existing, plain) {
			return domain.ErrConflict
		}
	}
	dir, err := s.root.Open(".")
	if err != nil {
		return domain.ErrUnavailable
	}
	defer dir.Close()
	if err = dir.Sync(); err != nil {
		return domain.ErrUnavailable
	}
	return nil
}
func (s *Store) Read(ctx context.Context, o domain.Object) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !s.Available() {
		return nil, domain.ErrUnavailable
	}
	name, aad, err := s.location(o)
	if err != nil {
		return nil, err
	}
	f, err := s.root.OpenFile(name, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, domain.ErrUnavailable
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
		return nil, domain.ErrUnavailable
	}
	max := int64(domain.MaxPreviewBytes*2 + 4096)
	data, err := io.ReadAll(io.LimitReader(f, max+1))
	if err != nil || int64(len(data)) > max {
		return nil, domain.ErrUnavailable
	}
	plain, err := s.keys.Open(data, aad)
	if err != nil {
		return nil, domain.ErrUnavailable
	}
	if int64(len(plain)) != o.Size || domain.Hash(plain) != o.Hash {
		clear(plain)
		return nil, domain.ErrUnavailable
	}
	if err = ctx.Err(); err != nil {
		clear(plain)
		return nil, err
	}
	return plain, nil
}
