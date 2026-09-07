package processor

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	domain "github.com/pchkauu/want-keep/backend/internal/attachments/domain"
)

// The supervisor holds the lock across its lifetime. Only it may reclaim a
// previous inspection's files; a killed parser cannot run deferred cleanup.
type workspaces struct {
	root *os.Root
	lock *os.File
}

func openWorkspaces(path string) (*workspaces, error) {
	if err := os.Mkdir(path, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
		return nil, domain.ErrUnavailable
	}
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0o700 {
		return nil, domain.ErrUnavailable
	}
	owner, ok := info.Sys().(*syscall.Stat_t)
	if !ok || owner.Uid != uint32(os.Geteuid()) {
		return nil, domain.ErrUnavailable
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, domain.ErrUnavailable
	}
	current, err := root.Stat(".")
	if err != nil || !os.SameFile(info, current) {
		_ = root.Close()
		return nil, domain.ErrUnavailable
	}
	lock, err := root.OpenFile(".lock", os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		_ = root.Close()
		return nil, domain.ErrUnavailable
	}
	state, err := lock.Stat()
	if err != nil || !state.Mode().IsRegular() || state.Mode().Perm() != 0o600 || syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB) != nil {
		_ = lock.Close()
		_ = root.Close()
		return nil, domain.ErrUnavailable
	}
	w := &workspaces{root: root, lock: lock}
	if err = w.recover(); err != nil {
		w.close()
		return nil, err
	}
	return w, nil
}
func (w *workspaces) recover() error {
	directory, err := w.root.Open(".")
	if err != nil {
		return domain.ErrUnavailable
	}
	defer directory.Close()
	for {
		names, err := directory.Readdirnames(100)
		for _, name := range names {
			if strings.HasPrefix(name, "inspection-") {
				suffix := strings.TrimPrefix(name, "inspection-")
				id, decodeErr := hex.DecodeString(suffix)
				if decodeErr != nil || len(id) != 16 {
					return domain.ErrUnavailable
				}
				if err := w.root.RemoveAll(name); err != nil {
					return domain.ErrUnavailable
				}
			} else if name != ".lock" {
				return domain.ErrUnavailable
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return domain.ErrUnavailable
		}
	}
	return nil
}
func (w *workspaces) create() (string, error) {
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return "", domain.ErrUnavailable
	}
	name := "inspection-" + hex.EncodeToString(id[:])
	if err := w.root.Mkdir(name, 0o700); err != nil {
		return "", domain.ErrUnavailable
	}
	return name, nil
}
func (w *workspaces) path(name string) string { return filepath.Join(w.root.Name(), name) }
func (w *workspaces) remove(name string) error {
	if err := w.root.RemoveAll(name); err != nil {
		return domain.ErrUnavailable
	}
	return nil
}
func (w *workspaces) close() { _ = w.lock.Close(); _ = w.root.Close() }
