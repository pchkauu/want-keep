package processor

import (
	"bytes"
	"context"
	"encoding/json"
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

type boundedOutput struct {
	bytes.Buffer
	maximum  int
	exceeded bool
}

func (b *boundedOutput) Write(p []byte) (int, error) {
	if len(p) > b.maximum-b.Len() {
		b.exceeded = true
		return 0, domain.ErrInvalid
	}
	return b.Buffer.Write(p)
}
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
		return nil, domain.ErrInvalid
	}
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return nil, domain.ErrInvalid
		}
		return nil, domain.ErrUnavailable
	}
	return out.Bytes(), nil
}

type pdfStructure struct {
	Version int               `json:"version"`
	Pages   []json.RawMessage `json:"pages"`
	Encrypt struct {
		Encrypted bool `json:"encrypted"`
	} `json:"encrypt"`
	Objects []json.RawMessage `json:"qpdf"`
}

func (e *Engine) inspectStructure(data []byte) (int, domain.Reason) {
	var structure pdfStructure
	if json.Unmarshal(data, &structure) != nil || structure.Version != 2 || len(structure.Objects) != 2 {
		return 0, domain.InvalidDocument
	}
	if structure.Encrypt.Encrypted {
		return 0, domain.UnsupportedContent
	}
	if len(structure.Pages) < 1 {
		return 0, domain.InvalidDocument
	}
	if len(structure.Pages) > domain.MaxPages {
		return 0, domain.LimitExceeded
	}
	var objects map[string]any
	if json.Unmarshal(structure.Objects[1], &objects) != nil || len(objects) == 0 {
		return 0, domain.InvalidDocument
	}
	remaining := 200000
	if !e.passiveObject(objects, 0, &remaining) {
		return 0, domain.UnsupportedContent
	}
	return len(structure.Pages), domain.NoReason
}
func (e *Engine) passiveObject(value any, depth int, remaining *int) bool {
	*remaining--
	if depth > 128 || *remaining < 0 {
		return false
	}
	switch v := value.(type) {
	case map[string]any:
		for key, item := range v {
			switch key {
			case "/JS", "/JavaScript", "/EmbeddedFiles", "/EF", "/RichMedia", "/RichMediaContent", "/XFA", "/OpenAction", "/AA", "/Movie", "/Sound", "/3D", "/3DD", "/RichMediaSettings":
				return false
			}
			if key == "/S" {
				switch item {
				case "/JavaScript", "/Launch", "/SubmitForm", "/ImportData", "/GoToR", "/GoToE", "/Rendition", "/Movie", "/Sound":
					return false
				}
			}
			if !e.passiveObject(item, depth+1, remaining) {
				return false
			}
		}
	case []any:
		for _, item := range v {
			if !e.passiveObject(item, depth+1, remaining) {
				return false
			}
		}
	}
	return true
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
	count, reason := e.inspectStructure(structure)
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
	if errors.Is(err, context.DeadlineExceeded) {
		return application.Inspection{Reason: domain.LimitExceeded}, nil
	}
	return application.Inspection{}, err
}
