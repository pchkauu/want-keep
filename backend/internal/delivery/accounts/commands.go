package accounts

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

func (s *Server) commandDTO(c command.Command) (generated.CommandStatus, error) {
	x := c.Snapshot()
	var out generated.CommandStatus
	var err error
	switch x.Status {
	case command.Pending:
		err = out.FromCommandPending(generated.CommandPending{Id: x.ID, Type: x.Kind, Status: "pending", RegisteredAt: x.RegisteredAt.String()})
	case command.Succeeded:
		err = out.FromCommandSucceeded(generated.CommandSucceeded{Id: x.ID, Type: x.Kind, Status: "succeeded", RegisteredAt: x.RegisteredAt.String(), CompletedAt: x.CompletedAt.String(), Result: generated.CommandResult{Type: x.Result.ResourceType, Id: x.Result.ResourceID, Revision: int64(x.Result.Revision)}})
	case command.Failed:
		err = out.FromCommandFailed(generated.CommandFailed{Id: x.ID, Type: x.Kind, Status: "failed", RegisteredAt: x.RegisteredAt.String(), CompletedAt: x.CompletedAt.String(), Error: generated.APIError{Version: "1", Code: generated.ErrorCode(x.ErrorCode), Message: "The change was not applied. Check the request, permissions or revision.", CorrelationId: uuid.NewString(), Violations: []generated.FieldViolation{}, Retryable: false}})
	default:
		return out, command.ErrInvalidCommand
	}
	return out, err
}
func (s *Server) commandResponse(w http.ResponseWriter, p household.Principal, c command.Command, status int) {
	err := c.RequireDetail(p, s.now())
	if errors.Is(err, command.ErrCommandExpired) {
		x := c.Snapshot()
		outcome := command.Outcome{CommandID: x.ID, Status: x.Status, Result: x.Result, FailureCode: x.ErrorCode}
		dto, e := s.boundary.ExpiredCommandToDTO(uuid.NewString(), &outcome)
		if e != nil {
			s.problem(w, e)
			return
		}
		s.write(w, 410, dto)
		return
	}
	if err != nil {
		s.problem(w, err)
		return
	}
	dto, err := s.commandDTO(c)
	if err != nil {
		s.problem(w, err)
		return
	}
	s.write(w, status, dto)
}
func (s *Server) command(w http.ResponseWriter, r *http.Request) {
	a, err := s.guard.Authorize(r, s.sessions, false)
	if err != nil {
		s.problem(w, err)
		return
	}
	id, err := s.resourceID(r, "commandId")
	if err != nil {
		s.problem(w, err)
		return
	}
	var c command.Command
	err = s.reads.WithinAccountRead(r.Context(), a.Principal, func(ctx context.Context) error {
		var e error
		c, e = s.queries.Read(ctx, a.Principal, id, s.now())
		if errors.Is(e, command.ErrCommandExpired) {
			return nil
		}
		return e
	})
	if err != nil {
		s.problem(w, err)
		return
	}
	s.commandResponse(w, a.Principal, c, 200)
}
func (s *Server) recent(w http.ResponseWriter, r *http.Request) {
	p, err := s.page(r, "commands")
	if err != nil {
		s.problem(w, err)
		return
	}
	out := generated.CommandStatusPage{Items: []generated.CommandStatus{}}
	out.Quality, err = s.pageQuality()
	if err != nil {
		s.problem(w, err)
		return
	}
	err = s.reads.WithinAccountRead(r.Context(), p.access.Principal, func(ctx context.Context) error {
		entries, next, e := s.queries.Recent(ctx, p.access.Principal, s.now(), p.after, p.limit)
		if e != nil {
			return e
		}
		for _, c := range entries {
			dto, e := s.commandDTO(c)
			if e != nil {
				return e
			}
			out.Items = append(out.Items, dto)
		}
		if next != "" {
			cursor := p.cursor(next)
			out.NextCursor = &cursor
		}
		return nil
	})
	if err != nil {
		s.problem(w, err)
		return
	}
	s.write(w, 200, out)
}
