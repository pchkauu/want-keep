package attachments

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	domain "github.com/pchkauu/want-keep/backend/internal/attachments/domain"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/domain"
)

func (s *Server) write(w http.ResponseWriter, value any) {
	data, err := json.Marshal(value)
	if err != nil {
		s.problem(w, domain.ErrUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)
}
func (s *Server) toAPI(a domain.Attachment) generated.Attachment {
	result := generated.Attachment{Id: generated.ID(a.Upload.ID), AccountId: generated.ID(a.Upload.AccountID), ActorId: generated.ID(a.ActorID), Name: a.Upload.Name, MediaType: generated.AttachmentMediaType(a.Upload.MediaType), SizeBytes: int(a.Upload.Size), State: generated.AttachmentState(a.State), Reason: generated.AttachmentReason(a.Reason), PreviewPages: make([]generated.AttachmentPreviewPage, len(a.Pages))}
	if a.State == domain.Accepted {
		count := len(a.Pages)
		result.PageCount = &count
		result.PreviewReady = true
	}
	for i, p := range a.Pages {
		result.PreviewPages[i] = generated.AttachmentPreviewPage{Page: p.Number, Width: p.Width, Height: p.Height}
	}
	return result
}
func (s *Server) problem(w http.ResponseWriter, err error) {
	status, code, message := 503, generated.ErrorCode("private_storage_unavailable"), "Private storage or document processing is temporarily unavailable."
	switch {
	case errors.Is(err, identity.ErrUnauthorized):
		status, code, message = 401, "unauthorized", "Sign in with your own passkey."
	case errors.Is(err, identity.ErrAttempt), errors.Is(err, domain.ErrInvalid):
		status, code, message = 400, "invalid_attachment", "Check the selected account, file format and upload limits."
	case errors.Is(err, domain.ErrNotFound):
		status, code, message = 404, "not_found", "The attachment is unavailable."
	case errors.Is(err, domain.ErrConflict):
		status, code, message = 409, "upload_conflict", "This upload ID belongs to a different request."
	case errors.Is(err, domain.ErrNotReady):
		status, code, message = 409, "attachment_not_ready", "The attachment has not passed validation. Check its processing status."
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(generated.APIError{Version: "1", Code: code, Message: message, Violations: []generated.FieldViolation{}, Retryable: status == 503, CorrelationId: uuid.NewString()})
}
