package attachments

import (
	"errors"
	"io"
	"net/http"

	domain "github.com/pchkauu/want-keep/backend/internal/attachments/domain"
)

func (s *Server) upload(w http.ResponseWriter, r *http.Request) {
	if _, err := s.guard.Authorize(r, s.sessions, true); err != nil {
		s.problem(w, err)
		return
	}
	select {
	case s.uploads <- struct{}{}:
		defer func() { <-s.uploads }()
	default:
		s.problem(w, domain.ErrUnavailable)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, domain.MaxBytes+65536)
	reader, err := r.MultipartReader()
	if err != nil {
		s.problem(w, domain.ErrInvalid)
		return
	}
	var input domain.Upload
	var data []byte
	seen := map[string]bool{}
	defer func() { clear(data) }()
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			s.problem(w, domain.ErrInvalid)
			return
		}
		name := part.FormName()
		if seen[name] || len(seen) >= 3 {
			s.problem(w, domain.ErrInvalid)
			return
		}
		seen[name] = true
		switch name {
		case "uploadId", "accountId":
			value, err := io.ReadAll(io.LimitReader(part, 65))
			if err != nil || len(value) > 64 || part.FileName() != "" {
				s.problem(w, domain.ErrInvalid)
				return
			}
			if name == "uploadId" {
				input.ID = string(value)
			} else {
				input.AccountID = string(value)
			}
		case "file":
			input.Name = part.FileName()
			input.MediaType = part.Header.Get("Content-Type")
			data, err = io.ReadAll(io.LimitReader(part, domain.MaxBytes+1))
			if err != nil || len(data) > domain.MaxBytes {
				s.problem(w, domain.ErrInvalid)
				return
			}
		default:
			s.problem(w, domain.ErrInvalid)
			return
		}
		if err = part.Close(); err != nil {
			s.problem(w, domain.ErrInvalid)
			return
		}
	}
	if len(seen) != 3 {
		s.problem(w, domain.ErrInvalid)
		return
	}
	input.Size = int64(len(data))
	input.Hash = domain.Hash(data)
	access, err := s.guard.Authorize(r, s.sessions, true)
	if err != nil {
		s.problem(w, err)
		return
	}
	a, err := s.service.Upload(r.Context(), access.Principal, input, data)
	if err != nil {
		s.problem(w, err)
		return
	}
	s.write(w, s.toAPI(a))
}
