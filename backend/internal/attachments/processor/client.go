package processor

import (
	"bytes"
	"context"
	"encoding/json"
	"image/png"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	application "github.com/pchkauu/want-keep/backend/internal/attachments/application"
	domain "github.com/pchkauu/want-keep/backend/internal/attachments/domain"
)

type Client struct{ http *http.Client }

func NewClient(socket string) (*Client, error) {
	if !filepath.IsAbs(socket) {
		return nil, domain.ErrUnavailable
	}
	dialer := &net.Dialer{Timeout: time.Second}
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return dialer.DialContext(ctx, "unix", socket)
	}, DisableKeepAlives: true, MaxResponseHeaderBytes: 4096}
	return &Client{http: &http.Client{Transport: transport, Timeout: 35 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}
func (c *Client) Ready(ctx context.Context) bool {
	if c == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://processor/ready", nil)
	if err != nil {
		return false
	}
	resp, err := c.http.Do(r)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 64))
	return err == nil && resp.StatusCode == 200 && string(data) == "want-keep-document-processor/1\n"
}
func (c *Client) Inspect(ctx context.Context, media string, data []byte) (application.Inspection, error) {
	var empty application.Inspection
	if c == nil || len(data) < 1 || len(data) > domain.MaxBytes {
		return empty, domain.ErrUnavailable
	}
	id := uuid.NewString()
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://processor/inspect", bytes.NewReader(data))
	if err != nil {
		return empty, err
	}
	r.Header.Set("Content-Type", media)
	r.Header.Set("X-Request-ID", id)
	resp, err := c.http.Do(r)
	if err != nil {
		if ctx.Err() != nil {
			return empty, ctx.Err()
		}
		return empty, domain.ErrUnavailable
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return empty, domain.ErrUnavailable
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponseBytes+1))
	if err != nil || len(raw) > MaxResponseBytes {
		return empty, domain.ErrUnavailable
	}
	var result response
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(&result) != nil || d.Decode(new(any)) != io.EOF || result.Version != ProtocolVersion || result.RequestID != id || result.Hash != domain.Hash(data) || !result.Reason.Valid() {
		return empty, domain.ErrUnavailable
	}
	if result.Reason.Rejects() {
		if len(result.Pages) != 0 {
			return empty, domain.ErrUnavailable
		}
		return result.result(), nil
	}
	if result.Reason != domain.NoReason || len(result.Pages) < 1 || len(result.Pages) > domain.MaxPages || (media != "application/pdf" && len(result.Pages) != 1) {
		return empty, domain.ErrUnavailable
	}
	for _, p := range result.Pages {
		if len(p.PNG) < 1 || len(p.PNG) > domain.MaxPreviewBytes || p.Width < 1 || p.Height < 1 || p.Width > 2048 || p.Height > 2048 {
			return empty, domain.ErrUnavailable
		}
		config, err := png.DecodeConfig(bytes.NewReader(p.PNG))
		if err != nil || config.Width != p.Width || config.Height != p.Height {
			return empty, domain.ErrUnavailable
		}
		if _, err = png.Decode(bytes.NewReader(p.PNG)); err != nil {
			return empty, domain.ErrUnavailable
		}
	}
	return result.result(), nil
}
