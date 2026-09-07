package processor

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	application "github.com/pchkauu/want-keep/backend/internal/attachments/application"
	domain "github.com/pchkauu/want-keep/backend/internal/attachments/domain"
)

type Server struct {
	mu     sync.Mutex
	engine interface {
		Inspect(context.Context, string, []byte) (application.Inspection, error)
	}
}

func NewServer(engine interface {
	Inspect(context.Context, string, []byte) (application.Inspection, error)
}) *Server {
	return &Server{engine: engine}
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method == http.MethodGet && r.URL.Path == "/ready" {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = io.WriteString(w, "want-keep-document-processor/1\n")
		return
	}
	if r.Method != http.MethodPost || r.URL.Path != "/inspect" {
		w.WriteHeader(404)
		return
	}
	id, err := uuid.Parse(r.Header.Get("X-Request-ID"))
	if err != nil || id.Version() != 4 {
		w.WriteHeader(400)
		return
	}
	if !s.mu.TryLock() {
		w.WriteHeader(503)
		return
	}
	defer s.mu.Unlock()
	r.Body = http.MaxBytesReader(w, r.Body, domain.MaxBytes)
	data, err := io.ReadAll(r.Body)
	if err != nil || len(data) == 0 {
		w.WriteHeader(400)
		return
	}
	defer clear(data)
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	result, err := s.engine.Inspect(ctx, r.Header.Get("Content-Type"), data)
	if err != nil {
		w.WriteHeader(503)
		return
	}
	out := response{Version: ProtocolVersion, RequestID: id.String(), Hash: domain.Hash(data), Reason: result.Reason, Pages: make([]page, len(result.Pages))}
	for i, p := range result.Pages {
		out.Pages[i] = page{Width: p.Width, Height: p.Height, PNG: p.PNG}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}
