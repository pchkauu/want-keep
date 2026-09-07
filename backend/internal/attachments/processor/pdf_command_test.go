package processor

import (
	"context"
	"errors"
	domain "github.com/pchkauu/want-keep/backend/internal/attachments/domain"
	"os"
	"path/filepath"
	"testing"
)

func TestPDFToolFailuresPreserveRetry(t *testing.T) {
	for _, tool := range []string{"qpdf", "pdftoppm"} {
		for _, c := range []struct {
			name, body string
			want       error
		}{
			{"signal", "kill -KILL $$", domain.ErrUnavailable},
			{"unknown", "exit 99", domain.ErrUnavailable},
			{"output limit", "printf 12345", errOutputLimit},
		} {
			t.Run(tool+"/"+c.name, func(t *testing.T) {
				dir := t.TempDir()
				path := filepath.Join(dir, tool)
				if err := os.WriteFile(path, []byte("#!/bin/sh\n"+c.body+"\n"), 0o700); err != nil {
					t.Fatal(err)
				}
				engine := Engine{qpdf: filepath.Join(dir, "qpdf"), pdftoppm: filepath.Join(dir, "pdftoppm"), scratch: dir}
				_, err := engine.command(context.Background(), dir, path, 4)
				if !errors.Is(err, c.want) {
					t.Fatalf("failure=%v want=%v", err, c.want)
				}
				result, failure := engine.pdfFailure(err)
				if errors.Is(c.want, domain.ErrUnavailable) && (failure == nil || result.Reason != domain.NoReason) {
					t.Fatal("infrastructure failure became terminal", failure, result.Reason)
				}
				if errors.Is(c.want, errOutputLimit) && (failure != nil || result.Reason != domain.LimitExceeded) {
					t.Fatal("output limit is not explicit")
				}
			})
		}
	}
	for _, c := range []struct {
		tool string
		code string
		want error
	}{
		{"qpdf", "2", domain.ErrInvalid}, {"qpdf", "3", domain.ErrInvalid}, {"qpdf", "1", domain.ErrUnavailable},
		{"pdftoppm", "1", domain.ErrInvalid}, {"pdftoppm", "3", domain.ErrInvalid}, {"pdftoppm", "2", domain.ErrUnavailable},
	} {
		t.Run(c.tool+"/exit"+c.code, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, c.tool)
			if err := os.WriteFile(path, []byte("#!/bin/sh\nexit "+c.code+"\n"), 0o700); err != nil {
				t.Fatal(err)
			}
			engine := Engine{qpdf: filepath.Join(dir, "qpdf"), pdftoppm: filepath.Join(dir, "pdftoppm")}
			_, err := engine.command(context.Background(), dir, path, 100)
			if !errors.Is(err, c.want) {
				t.Fatalf("exit error=%v want=%v", err, c.want)
			}
		})
	}
}
