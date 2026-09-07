package storage

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

// Invitation coordination follows the identity lock, then the household lock. It never impersonates the inviter.
func (s *Store) WithinInvitationHousehold(ctx context.Context, id household.HouseholdID, fn func(context.Context) error) error {
	scope, err := s.scope(ctx)
	if err != nil {
		return err
	}
	if !scope.identityLocked || scope.householdLocked || scope.admissionKey != "" || id == "" {
		return ErrTransactionRequired
	}
	if scope.invitationHouseholdID != "" {
		if scope.invitationHouseholdID != id {
			return household.ErrForbidden
		}
		return fn(ctx)
	}
	var found string
	if err = scope.tx.QueryRow(ctx, "SELECT id FROM want_keep.households WHERE id=$1 FOR UPDATE", id).Scan(&found); err != nil {
		return s.invitationError(err)
	}
	scope.invitationHouseholdID = id
	return fn(ctx)
}

func (s *Store) HouseholdDetails(ctx context.Context, id household.HouseholdID) (household.Details, error) {
	d := household.Details{Household: household.Household{ID: id}, Members: []household.Member{}}
	scope, err := s.scope(ctx)
	if err != nil {
		return d, err
	}
	if !scope.identityLocked {
		return d, ErrTransactionRequired
	}
	err = scope.tx.QueryRow(ctx, "SELECT name,timezone,max_members FROM want_keep.households WHERE id=$1", id).Scan(&d.Household.Name, &d.Timezone, &d.Maximum)
	if err != nil {
		return d, s.invitationError(err)
	}
	rows, err := scope.tx.Query(ctx, `SELECT m.id,m.user_id,m.active,u.name FROM want_keep.memberships m JOIN want_keep.users u ON u.id=m.user_id WHERE m.household_id=$1 ORDER BY m.id`, id)
	if err != nil {
		return d, err
	}
	defer rows.Close()
	for rows.Next() {
		m := household.Member{Membership: household.Membership{HouseholdID: id}}
		if err = rows.Scan(&m.Membership.ID, &m.Membership.UserID, &m.Membership.Active, &m.User.Name); err != nil {
			return d, err
		}
		m.User.ID = m.Membership.UserID
		d.Members = append(d.Members, m)
	}
	return d, rows.Err()
}

const invitationColumns = `id,household_id,invited_by,token_hash,created_at,expires_at,status,COALESCE(accepted_user_id::text,'')`

type invitationRow struct{ invitation household.Invitation }

func (r *invitationRow) scan(row pgx.Row) error {
	i := &r.invitation
	if err := row.Scan(&i.ID, &i.HouseholdID, &i.InvitedBy, &i.TokenHash, &i.CreatedAt, &i.ExpiresAt, &i.Status, &i.AcceptedUserID); err != nil {
		return err
	}
	return i.Validate()
}

func (s *Store) InvitationByHash(ctx context.Context, hash string) (household.Invitation, error) {
	var r invitationRow
	scope, err := s.scope(ctx)
	if err != nil {
		return r.invitation, err
	}
	if !scope.identityLocked {
		return r.invitation, ErrTransactionRequired
	}
	err = r.scan(scope.tx.QueryRow(ctx, "SELECT "+invitationColumns+" FROM want_keep.household_invitations WHERE token_hash=$1", hash))
	return r.invitation, s.invitationError(err)
}

func (s *Store) InvitationState(ctx context.Context, id household.HouseholdID) (household.InvitationState, error) {
	state := household.InvitationState{HouseholdID: id}
	scope, err := s.scope(ctx)
	if err != nil {
		return state, err
	}
	if !scope.identityLocked {
		return state, ErrTransactionRequired
	}
	var current *string
	err = scope.tx.QueryRow(ctx, "SELECT invitation_revision,current_invitation_id::text FROM want_keep.households WHERE id=$1", id).Scan(&state.Revision, &current)
	if err != nil {
		return state, s.invitationError(err)
	}
	if current != nil {
		var r invitationRow
		if err = r.scan(scope.tx.QueryRow(ctx, "SELECT "+invitationColumns+" FROM want_keep.household_invitations WHERE household_id=$1 AND id=$2", id, *current)); err != nil {
			return state, s.invitationError(err)
		}
		state.Current = &r.invitation
	}
	return state, state.Validate()
}

func (s *Store) SaveInvitation(ctx context.Context, i household.Invitation) error {
	scope, err := s.scope(ctx)
	if err != nil {
		return err
	}
	if scope.invitationHouseholdID != i.HouseholdID || !scope.identityLocked {
		return ErrTransactionRequired
	}
	if err = i.Validate(); err != nil {
		return err
	}
	tag, err := scope.tx.Exec(ctx, `INSERT INTO want_keep.household_invitations(household_id,id,invited_by,token_hash,created_at,expires_at,status,accepted_user_id)
 VALUES($1,$2,$3,$4,$5,$6,$7,NULLIF($8,'')::uuid)
 ON CONFLICT(household_id,id) DO UPDATE SET status=EXCLUDED.status,accepted_user_id=EXCLUDED.accepted_user_id
 WHERE household_invitations.status='active' AND household_invitations.invited_by=EXCLUDED.invited_by
 AND household_invitations.token_hash=EXCLUDED.token_hash AND household_invitations.created_at=EXCLUDED.created_at AND household_invitations.expires_at=EXCLUDED.expires_at`, i.HouseholdID, i.ID, i.InvitedBy, i.TokenHash, i.CreatedAt, i.ExpiresAt, i.Status, string(i.AcceptedUserID))
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return household.ErrInvitationRevision
	}
	return nil
}

func (s *Store) SaveInvitationState(ctx context.Context, state household.InvitationState, expected uint64) error {
	scope, err := s.scope(ctx)
	if err != nil {
		return err
	}
	if !scope.identityLocked || scope.invitationHouseholdID != state.HouseholdID {
		return ErrTransactionRequired
	}
	if err = state.Validate(); err != nil {
		return err
	}
	if state.Current == nil || expected >= household.MaxInvitationRevision || state.Revision != expected+1 {
		return household.ErrInvitationRevision
	}
	tag, err := scope.tx.Exec(ctx, "UPDATE want_keep.households SET invitation_revision=$3,current_invitation_id=$4 WHERE id=$1 AND invitation_revision=$2", state.HouseholdID, expected, state.Revision, state.Current.ID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return household.ErrInvitationRevision
	}
	return nil
}

func (s *Store) JoinHousehold(ctx context.Context, user household.User, member household.Membership) error {
	scope, err := s.scope(ctx)
	if err != nil {
		return err
	}
	if !scope.identityLocked || scope.invitationHouseholdID != member.HouseholdID {
		return ErrTransactionRequired
	}
	if user.ID == "" || member.ID == "" || member.UserID != user.ID || !member.Active {
		return household.ErrInvalidMembership
	}
	d, err := s.HouseholdDetails(ctx, member.HouseholdID)
	if err != nil {
		return err
	}
	if err = d.RequireSpace(); err != nil {
		return err
	}
	if _, err = scope.tx.Exec(ctx, "INSERT INTO want_keep.users(id,name) VALUES($1,$2)", user.ID, user.Name); err != nil {
		return err
	}
	_, err = scope.tx.Exec(ctx, "INSERT INTO want_keep.memberships(household_id,id,user_id,active) VALUES($1,$2,$3,true)", member.HouseholdID, member.ID, member.UserID)
	return err
}

func (s *Store) InvitationAudit(ctx context.Context, family household.HouseholdID, actor household.UserID, event, id string, revision uint64, now time.Time) error {
	scope, err := s.scope(ctx)
	if err != nil {
		return err
	}
	if !scope.identityLocked || scope.invitationHouseholdID != family {
		return ErrTransactionRequired
	}
	_, err = scope.tx.Exec(ctx, "INSERT INTO want_keep.household_invitation_audit(household_id,invitation_id,revision,actor_id,event,occurred_at) VALUES($1,$2,$3,$4,$5,$6)", family, id, revision, actor, event, now)
	return err
}

func (s *Store) MembershipJoined(ctx context.Context, m household.Membership, revision uint64) error {
	scope, err := s.scope(ctx)
	if err != nil {
		return err
	}
	if !scope.identityLocked || scope.invitationHouseholdID != m.HouseholdID {
		return ErrTransactionRequired
	}
	id := newID()
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.jobs(household_id,id,actor_id,kind,state,max_attempts,available_at,deadline)
 VALUES($1,$2,$3,'outbox','ready',5,clock_timestamp(),clock_timestamp()+INTERVAL '24 hours')`, m.HouseholdID, id, m.UserID)
	if err != nil {
		return err
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.outbox(household_id,id,actor_id,resource_type,resource_id,revision,event_type)
 VALUES($1,$2,$3,'membership',$4,$5,'household.member_joined')`, m.HouseholdID, id, m.UserID, m.ID, revision)
	return err
}

func (s *Store) invitationError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return household.ErrInvitation
	}
	return err
}
