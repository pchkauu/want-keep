package contract

import (
	"errors"
	"net/http"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	connection "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

type ErrorConverter struct{}
type ErrorResponse struct {
	Status int
	Body   generated.APIError
}

func (ErrorConverter) ToResponse(err error, correlationID string) ErrorResponse {
	response := ErrorResponse{Status: http.StatusInternalServerError, Body: generated.APIError{Version: "1", Code: "internal_error", Message: "The request could not be completed. Check its command status before retrying.", CorrelationId: correlationID, Violations: []generated.FieldViolation{}, Retryable: false}}
	switch {
	case errors.Is(err, money.ErrUnsupportedAsset):
		response.Status, response.Body.Code = 422, "unsupported_asset"
	case errors.Is(err, money.ErrAssetMismatch):
		response.Status, response.Body.Code = 422, "asset_mismatch"
	case errors.Is(err, money.ErrInvalidRate):
		response.Status, response.Body.Code = 422, "invalid_rate"
	case errors.Is(err, money.ErrInvalidMoney), errors.Is(err, money.ErrInvalidRounding):
		response.Status, response.Body.Code = 422, "invalid_money"
	case errors.Is(err, money.ErrInvalidAllocation):
		response.Status, response.Body.Code = 422, "invalid_allocation"
	case errors.Is(err, calendar.ErrInvalidTime):
		response.Status, response.Body.Code = 422, "invalid_time"
	case errors.Is(err, reporting.ErrInvalidAvailability):
		response.Status, response.Body.Code = 422, "invalid_availability"
	case errors.Is(err, household.ErrForbidden), errors.Is(err, household.ErrInvalidMembership):
		response.Status, response.Body.Code = 403, "forbidden"
	case errors.Is(err, command.ErrVersionConflict):
		response.Status, response.Body.Code = 409, "version_conflict"
	case errors.Is(err, command.ErrDuplicateCommand):
		response.Status, response.Body.Code = 409, "duplicate_command"
	case errors.Is(err, command.ErrFinalCommand):
		response.Status, response.Body.Code = 409, "command_final"
	case errors.Is(err, command.ErrCommandExpired):
		response.Status, response.Body.Code = 410, "command_expired"
	case errors.Is(err, command.ErrCommandNotFound):
		response.Status, response.Body.Code = 404, "not_found"
	case errors.Is(err, connection.ErrProviderNotAdmitted):
		response.Status, response.Body.Code = 409, "provider_not_admitted"
	case errors.Is(err, ErrInvalidRequest), errors.Is(err, command.ErrInvalidCommand), errors.Is(err, household.ErrInvalidOwnership):
		response.Status, response.Body.Code = 400, "invalid_request"
	}
	if response.Status != http.StatusInternalServerError {
		response.Body.Message = "The request was rejected. Review the indicated fields, permissions or version."
	}
	switch response.Body.Code {
	case "command_expired":
		response.Body.Message = "Command details have expired. Recover the existing outcome; do not create a new command to retry it."
	case "not_found":
		response.Body.Message = "Command recovery is unavailable. This does not establish whether an effect occurred."
	case "provider_not_admitted":
		response.Body.Message = "This connection is awaiting deployment verification. Synchronization has not started."
	}
	var field FieldError
	if errors.As(err, &field) {
		response.Body.Violations = []generated.FieldViolation{{Field: field.Field, Code: field.Code}}
	}
	return response
}
