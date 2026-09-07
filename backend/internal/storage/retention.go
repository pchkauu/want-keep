package storage

import (
	"context"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
)

func (s *Store) CleanupCommandDetails(ctx context.Context, now calendar.Instant, batch int) (int64, error) {
	if batch < 1 || batch > 1000 {
		return 0, command.ErrInvalidCommand
	}
	cutoff, _, err := command.RetentionCutoffs(now)
	if err != nil {
		return 0, err
	}
	at, ns := splitInstant(cutoff)
	tag, err := s.pool.Exec(ctx, `WITH expired AS (SELECT d.household_id,d.actor_id,d.command_id FROM want_keep.command_details d JOIN want_keep.command_tombstones c ON (c.household_id,c.actor_id,c.id)=(d.household_id,d.actor_id,d.command_id) WHERE c.status!='pending' AND (c.completed_at,c.completed_ns)<=($1,$2) ORDER BY c.completed_at,c.completed_ns,c.id LIMIT $3) DELETE FROM want_keep.command_details d USING expired e WHERE (d.household_id,d.actor_id,d.command_id)=(e.household_id,e.actor_id,e.command_id)`, at, ns, batch)
	return tag.RowsAffected(), err
}
func (s *Store) CleanupCommandTombstones(ctx context.Context, now calendar.Instant, batch int) (int64, error) {
	if batch < 1 || batch > 1000 {
		return 0, command.ErrInvalidCommand
	}
	_, cutoff, err := command.RetentionCutoffs(now)
	if err != nil {
		return 0, err
	}
	at, ns := splitInstant(cutoff)
	tag, err := s.pool.Exec(ctx, `WITH expired AS (SELECT c.household_id,c.actor_id,c.id FROM want_keep.command_tombstones c WHERE c.status!='pending' AND (c.completed_at,c.completed_ns)<=($1,$2) AND NOT EXISTS(SELECT 1 FROM want_keep.command_details d WHERE (d.household_id,d.actor_id,d.command_id)=(c.household_id,c.actor_id,c.id)) ORDER BY c.completed_at,c.completed_ns,c.id LIMIT $3) DELETE FROM want_keep.command_tombstones c USING expired e WHERE (c.household_id,c.actor_id,c.id)=(e.household_id,e.actor_id,e.id)`, at, ns, batch)
	return tag.RowsAffected(), err
}
