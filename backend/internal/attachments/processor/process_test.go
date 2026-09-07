package processor

import (
	"context"
	"errors"
	domain "github.com/pchkauu/want-keep/backend/internal/attachments/domain"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestProcessDeadlineAndCrashNeverReturnAcceptance(t *testing.T) {
	for _, body := range []string{"exec sleep 10", "kill -KILL $$", "printf invalid"} {
		t.Run(body, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "parser")
			if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o700); err != nil {
				t.Fatal(err)
			}
			workspaces, err := openWorkspaces(filepath.Join(t.TempDir(), "workspaces"))
			if err != nil {
				t.Fatal(err)
			}
			process := Process{executable: path, workspaces: workspaces}
			defer process.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			started := time.Now()
			result, err := process.Inspect(ctx, "application/pdf", []byte("synthetic"))
			if err == nil || len(result.Pages) != 0 || time.Since(started) > time.Second {
				t.Fatal("crash or timeout accepted or parser child survived", err)
			}
		})
	}
}

func TestSupervisorReclaimsKilledInspection(t *testing.T) {
	endings := []string{"exec sleep 10", "kill -KILL $$"}
	if runtime.GOOS == "linux" {
		endings = append(endings, "sleep 10 >/dev/null 2>&1 &\nkill -KILL $$")
	}
	for _, ending := range endings {
		t.Run(ending, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "parser")
			body := "#!/bin/sh\nprintf synthetic > \"$TMPDIR/input.pdf\"\nprintf partial > \"$TMPDIR/page-1.png\"\n" + ending + "\n"
			if err := os.WriteFile(path, []byte(body), 0o700); err != nil {
				t.Fatal(err)
			}
			spaces, err := openWorkspaces(filepath.Join(dir, "workspaces"))
			if err != nil {
				t.Fatal(err)
			}
			process := Process{executable: path, workspaces: spaces}
			defer process.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			done := make(chan error, 1)
			go func() { _, err := process.Inspect(ctx, "application/pdf", []byte("synthetic")); done <- err }()
			if ending == "exec sleep 10" {
				deadline := time.NewTimer(time.Second)
				defer deadline.Stop()
				tick := time.NewTicker(time.Millisecond)
				defer tick.Stop()
				ready := false
				for !ready {
					names, _ := filepath.Glob(filepath.Join(dir, "workspaces", "inspection-*", "input.pdf"))
					if len(names) > 0 {
						ready = true
						break
					}
					select {
					case <-deadline.C:
						t.Fatal("parser did not create workspace")
					case <-tick.C:
					}
				}
				cancel()
			}
			if err := <-done; !errors.Is(err, domain.ErrUnavailable) {
				t.Fatal("killed inspection accepted", err)
			}
			names, err := os.ReadDir(filepath.Join(dir, "workspaces"))
			if err != nil || len(names) != 1 || names[0].Name() != ".lock" {
				t.Fatal("document scratch survived", names, err)
			}
			if err := os.WriteFile(path, []byte("#!/bin/sh\nprintf '{\"Reason\":\"invalid_document\",\"Pages\":null}'\n"), 0o700); err != nil {
				t.Fatal(err)
			}
			if _, err := process.Inspect(context.Background(), "application/pdf", []byte("next")); err != nil {
				t.Fatal("next inspection cannot run", err)
			}
		})
	}
}
func TestWorkspaceRestartAndExclusiveOwnership(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workspaces")
	first, err := openWorkspaces(path)
	if err != nil {
		t.Fatal(err)
	}
	name, err := first.create()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(first.path(name), "input.pdf"), []byte("synthetic"), 0o600); err != nil {
		t.Fatal(err)
	}
	if second, err := openWorkspaces(path); err == nil {
		second.close()
		t.Fatal("second supervisor acquired live workspace")
	}
	if _, err := os.Stat(filepath.Join(first.path(name), "input.pdf")); err != nil {
		t.Fatal("active workspace removed")
	}
	first.close()
	restarted, err := openWorkspaces(path)
	if err != nil {
		t.Fatal(err)
	}
	defer restarted.close()
	if _, err := os.Stat(filepath.Join(path, name)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("restart kept abandoned workspace", err)
	}
	outside := t.TempDir()
	file := filepath.Join(outside, "untouched")
	if err := os.WriteFile(file, []byte("sentinel"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(path, name)); err != nil {
		t.Fatal(err)
	}
	if err := restarted.recover(); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(file); err != nil || string(data) != "sentinel" {
		t.Fatal("cleanup followed symlink", err)
	}
}
