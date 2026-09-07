package processor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	application "github.com/pchkauu/want-keep/backend/internal/attachments/application"
	domain "github.com/pchkauu/want-keep/backend/internal/attachments/domain"
)

var errOutputLimit = errors.New("document processor output limit exceeded")

type boundedOutput struct {
	buffer   bytes.Buffer
	maximum  int
	exceeded bool
}

func (b *boundedOutput) Write(p []byte) (int, error) {
	if len(p) > b.maximum-b.buffer.Len() {
		b.exceeded = true
		return 0, errOutputLimit
	}
	return b.buffer.Write(p)
}
func (b *boundedOutput) Bytes() []byte { return b.buffer.Bytes() }

func (e *Engine) command(ctx context.Context, directory, path string, maximum int, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Dir = directory
	cmd.Env = []string{"PATH=/usr/local/bin:/usr/bin:/bin", "HOME=/nonexistent", "LANG=C", "LC_ALL=C", "TMPDIR=" + directory}
	out := &boundedOutput{maximum: maximum}
	cmd.Stdout = out
	cmd.Stderr = io.Discard
	err := cmd.Run()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if out.exceeded {
		return nil, errOutputLimit
	}
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) && exit.Exited() {
			code := exit.ExitCode()
			if (path == e.qpdf && (code == 2 || code == 3)) || (path == e.pdftoppm && (code == 1 || code == 3)) {
				return nil, domain.ErrInvalid
			}
		}
		return nil, domain.ErrUnavailable
	}
	return out.Bytes(), nil
}

func (e *Engine) pdf(ctx context.Context, data []byte) (application.Inspection, error) {
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		return application.Inspection{Reason: domain.UnsupportedContent}, nil
	}
	dir, err := os.MkdirTemp(e.scratch, "document-")
	if err != nil {
		return application.Inspection{}, domain.ErrUnavailable
	}
	defer func() {
		entries, _ := os.ReadDir(dir)
		for _, entry := range entries {
			_ = os.Remove(filepath.Join(dir, entry.Name()))
		}
		_ = os.Remove(dir)
	}()
	if err = os.WriteFile(filepath.Join(dir, "input.pdf"), data, 0o600); err != nil {
		return application.Inspection{}, domain.ErrUnavailable
	}
	if _, err = e.command(ctx, dir, e.qpdf, 1024*1024, "--suppress-recovery", "--check", "input.pdf"); err != nil {
		return e.pdfFailure(err)
	}
	structure, err := e.command(ctx, dir, e.qpdf, 32*1024*1024, "--suppress-recovery", "--json=2", "--json-key=pages", "--json-key=encrypt", "--json-key=qpdf", "--json-stream-data=none", "input.pdf")
	if err != nil {
		return e.pdfFailure(err)
	}
	var policy pdfStructure
	count, reason := policy.inspect(structure)
	if reason != domain.NoReason {
		return application.Inspection{Reason: reason}, nil
	}
	if _, err = e.command(ctx, dir, e.pdftoppm, 4096, "-png", "-scale-to", "2048", "-f", "1", "-l", fmt.Sprint(count), "input.pdf", "page"); err != nil {
		return e.pdfFailure(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return application.Inspection{}, domain.ErrUnavailable
	}
	var names []string
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "page-") && strings.HasSuffix(entry.Name(), ".png") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	if len(names) != count {
		return application.Inspection{Reason: domain.InvalidDocument}, nil
	}
	result := application.Inspection{Pages: make([]application.Preview, 0, count)}
	total := 0
	for _, name := range names {
		f, err := os.OpenFile(filepath.Join(dir, name), os.O_RDONLY|syscall.O_NOFOLLOW, 0)
		if err != nil {
			return application.Inspection{}, domain.ErrUnavailable
		}
		preview, err := io.ReadAll(io.LimitReader(f, domain.MaxPreviewBytes+1))
		_ = f.Close()
		if err != nil || len(preview) > domain.MaxPreviewBytes {
			return application.Inspection{Reason: domain.LimitExceeded}, nil
		}
		total += len(preview)
		if total > 40*1024*1024 {
			return application.Inspection{Reason: domain.LimitExceeded}, nil
		}
		config, err := png.DecodeConfig(bytes.NewReader(preview))
		if err != nil || config.Width < 1 || config.Height < 1 || config.Width > 2048 || config.Height > 2048 {
			return application.Inspection{Reason: domain.InvalidDocument}, nil
		}
		result.Pages = append(result.Pages, application.Preview{Width: config.Width, Height: config.Height, PNG: preview})
	}
	return result, nil
}
func (e *Engine) pdfFailure(err error) (application.Inspection, error) {
	if errors.Is(err, domain.ErrInvalid) {
		return application.Inspection{Reason: domain.InvalidDocument}, nil
	}
	if errors.Is(err, errOutputLimit) {
		return application.Inspection{Reason: domain.LimitExceeded}, nil
	}
	return application.Inspection{}, err
}
