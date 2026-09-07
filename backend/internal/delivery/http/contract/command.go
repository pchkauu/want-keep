package contract

import (
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
)

// OutcomeToDTO requires the application to authorize the referenced resource before calling it.
func (b *Boundary) OutcomeToDTO(outcome command.Outcome) (generated.CommandOutcome, error) {
	var out generated.CommandOutcome
	var err error
	switch outcome.Status {
	case command.Succeeded:
		err = out.FromSucceededOutcome(generated.SucceededOutcome{CommandId: outcome.CommandID, Status: "succeeded", Result: generated.CommandResult{Type: outcome.Result.ResourceType, Id: outcome.Result.ResourceID, Revision: int64(outcome.Result.Revision)}})
	case command.Failed:
		err = out.FromFailedOutcome(generated.FailedOutcome{CommandId: outcome.CommandID, Status: "failed", FailureCode: generated.ErrorCode(outcome.FailureCode)})
	default:
		return out, ErrInvalidRequest
	}
	if err != nil {
		return out, err
	}
	return out, b.validateDTO("CommandOutcome", out)
}

func (b *Boundary) ExpiredCommandToDTO(correlationID string, authorizedOutcome *command.Outcome) (generated.ExpiredCommand, error) {
	problem := (ErrorConverter{}).ToResponse(command.ErrCommandExpired, correlationID).Body
	out := generated.ExpiredCommand{Version: "1", Code: "command_expired", Message: problem.Message, Violations: problem.Violations, Retryable: false, CorrelationId: correlationID}
	if authorizedOutcome != nil {
		mapped, err := b.OutcomeToDTO(*authorizedOutcome)
		if err != nil {
			return out, err
		}
		out.Outcome = &mapped
	}
	return out, b.validateDTO("ExpiredCommand", out)
}

func (b *Boundary) CommandToDTO(c command.Command, correlationID string) (generated.CommandStatus, error) {
	x := c.Snapshot()
	var out generated.CommandStatus
	var err error
	switch x.Status {
	case command.Pending:
		err = out.FromCommandPending(generated.CommandPending{Id: x.ID, Type: x.Kind, Status: "pending", RegisteredAt: x.RegisteredAt.String()})
	case command.Succeeded:
		err = out.FromCommandSucceeded(generated.CommandSucceeded{Id: x.ID, Type: x.Kind, Status: "succeeded", RegisteredAt: x.RegisteredAt.String(), CompletedAt: x.CompletedAt.String(), Result: generated.CommandResult{Type: x.Result.ResourceType, Id: x.Result.ResourceID, Revision: int64(x.Result.Revision)}})
	case command.Failed:
		err = out.FromCommandFailed(generated.CommandFailed{Id: x.ID, Type: x.Kind, Status: "failed", RegisteredAt: x.RegisteredAt.String(), CompletedAt: x.CompletedAt.String(), Error: generated.APIError{Version: "1", Code: generated.ErrorCode(x.ErrorCode), Message: "The change was not applied. Check the request, permissions or revision.", CorrelationId: correlationID, Violations: []generated.FieldViolation{}, Retryable: false}})
	default:
		return out, command.ErrInvalidCommand
	}
	return out, err
}
