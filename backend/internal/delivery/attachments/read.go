package attachments

import (
	"mime"
	"net/http"
	"strconv"

	domain "github.com/pchkauu/want-keep/backend/internal/attachments/domain"
)

func (s *Server) metadata(w http.ResponseWriter, r *http.Request) {
	access, err := s.guard.Authorize(r, s.sessions, false)
	if err != nil {
		s.problem(w, err)
		return
	}
	id, err := s.attachmentID(r)
	if err != nil {
		s.problem(w, err)
		return
	}
	a, err := s.service.Metadata(r.Context(), access.Principal, id)
	if err != nil {
		s.problem(w, err)
		return
	}
	s.write(w, s.toAPI(a))
}
func (s *Server) content(w http.ResponseWriter, r *http.Request) { s.bytes(w, r, 0) }
func (s *Server) preview(w http.ResponseWriter, r *http.Request) {
	page, err := strconv.Atoi(r.PathValue("page"))
	if err != nil || page < 1 || page > domain.MaxPages {
		s.problem(w, domain.ErrNotFound)
		return
	}
	s.bytes(w, r, page)
}
func (s *Server) bytes(w http.ResponseWriter, r *http.Request, page int) {
	access, err := s.guard.Authorize(r, s.sessions, false)
	if err != nil {
		s.problem(w, err)
		return
	}
	id, err := s.attachmentID(r)
	if err != nil {
		s.problem(w, err)
		return
	}
	a, data, err := s.service.Content(r.Context(), access.Principal, id, page)
	if err != nil {
		s.problem(w, err)
		return
	}
	defer clear(data)
	if _, err = s.guard.Authorize(r, s.sessions, false); err != nil {
		s.problem(w, err)
		return
	}
	contentType := "application/octet-stream"
	disposition := "attachment"
	name := a.Upload.Name
	if page > 0 {
		contentType = "image/png"
		disposition = "inline"
		name = "preview.png"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": name}))
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	_, _ = w.Write(data)
}
