package processor

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"

	application "github.com/pchkauu/want-keep/backend/internal/attachments/application"
	domain "github.com/pchkauu/want-keep/backend/internal/attachments/domain"
)

type Engine struct{ qpdf, pdftoppm, scratch string }

func NewEngine(qpdf, pdftoppm, scratch string) (*Engine, error) {
	e := &Engine{qpdf: qpdf, pdftoppm: pdftoppm, scratch: scratch}
	for _, tool := range []struct {
		path    string
		args    []string
		version string
	}{{qpdf, []string{"--version"}, "qpdf version 12.4.1"}, {pdftoppm, []string{"-v"}, "pdftoppm version 26.09.0"}} {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		cmd := exec.CommandContext(ctx, tool.path, tool.args...)
		cmd.Env = []string{"LANG=C", "LC_ALL=C", "HOME=/nonexistent", "PATH=/usr/bin:/bin:/usr/local/bin"}
		out, err := cmd.CombinedOutput()
		cancel()
		if err != nil || !strings.Contains(string(out), tool.version) {
			return nil, errors.New("document processor dependencies unavailable")
		}
	}
	return e, nil
}
func (e *Engine) Inspect(ctx context.Context, media string, data []byte) (application.Inspection, error) {
	if len(data) == 0 || len(data) > domain.MaxBytes {
		return application.Inspection{Reason: domain.LimitExceeded}, nil
	}
	switch media {
	case "image/jpeg", "image/png", "image/webp":
		return e.image(ctx, media, data)
	case "application/pdf":
		return e.pdf(ctx, data)
	}
	return application.Inspection{Reason: domain.UnsupportedContent}, nil
}

func (e *Engine) Ready(ctx context.Context) bool {
	return ctx.Err() == nil && e != nil && e.qpdf != "" && e.pdftoppm != ""
}
