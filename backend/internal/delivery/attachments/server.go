package attachments

import (
	"net/http"

	"github.com/google/uuid"
	application "github.com/pchkauu/want-keep/backend/internal/attachments/application"
	domain "github.com/pchkauu/want-keep/backend/internal/attachments/domain"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/security"
)

type Server struct {
	service              *application.Service
	sessions             security.Sessions
	guard                *security.Guard
	connectionsAvailable func() bool
	mux                  *http.ServeMux
	uploads              chan struct{}
}

func New(s *application.Service, sessions security.Sessions, c security.Config, connectionsAvailable func() bool) (*Server, error) {
	guard, err := security.New(c)
	if err != nil {
		return nil, err
	}
	if s == nil || sessions == nil || connectionsAvailable == nil {
		return nil, domain.ErrUnavailable
	}
	h := &Server{service: s, sessions: sessions, guard: guard, connectionsAvailable: connectionsAvailable, mux: http.NewServeMux(), uploads: make(chan struct{}, 2)}
	h.mux.HandleFunc("POST /api/v1/attachments", h.upload)
	h.mux.HandleFunc("GET /api/v1/attachments/{attachmentId}", h.metadata)
	h.mux.HandleFunc("GET /api/v1/attachments/{attachmentId}/content", h.content)
	h.mux.HandleFunc("GET /api/v1/attachments/{attachmentId}/preview/{page}", h.preview)
	h.mux.HandleFunc("GET /api/v1/system/privacy", h.status)
	return h, nil
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := s.guard.Check(w, r); err != nil {
		s.problem(w, err)
		return
	}
	s.mux.ServeHTTP(w, r)
}
func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	if _, err := s.guard.Authorize(r, s.sessions, false); err != nil {
		s.problem(w, err)
		return
	}
	status := s.service.Status(r.Context())
	result := generated.PrivacyStatus{Attachments: "unavailable", Connections: "unavailable", Processor: "unavailable"}
	if status.Attachments {
		result.Attachments = "available"
	}
	if status.Processor {
		result.Processor = "available"
	}
	if s.connectionsAvailable() {
		result.Connections = "available"
	}
	s.write(w, result)
}
func (s *Server) attachmentID(r *http.Request) (string, error) {
	value := r.PathValue("attachmentId")
	id, err := uuid.Parse(value)
	if err != nil || id.Version() != 4 || id.String() != value {
		return "", domain.ErrNotFound
	}
	return value, nil
}
