CREATE TABLE want_keep.household_invitations (
 household_id uuid NOT NULL REFERENCES want_keep.households(id), id uuid NOT NULL,
 invited_by uuid NOT NULL, token_hash text NOT NULL UNIQUE CHECK(token_hash ~ '^[0-9a-f]{64}$'),
 created_at timestamptz NOT NULL, expires_at timestamptz NOT NULL,
 status text NOT NULL CHECK(status IN ('active','revoked','accepted')), accepted_user_id uuid,
 PRIMARY KEY(household_id,id),
 FOREIGN KEY(household_id,invited_by) REFERENCES want_keep.memberships(household_id,user_id),
 FOREIGN KEY(household_id,accepted_user_id) REFERENCES want_keep.memberships(household_id,user_id),
 CHECK(expires_at=created_at+INTERVAL '24 hours'),
 CHECK((status='accepted')=(accepted_user_id IS NOT NULL))
);
CREATE UNIQUE INDEX one_active_invitation ON want_keep.household_invitations(household_id) WHERE status='active';
ALTER TABLE want_keep.households ADD COLUMN invitation_revision want_keep.revision NOT NULL DEFAULT 1;
ALTER TABLE want_keep.households ADD COLUMN current_invitation_id uuid;
ALTER TABLE want_keep.households ADD CONSTRAINT current_household_invitation
 FOREIGN KEY(id,current_invitation_id) REFERENCES want_keep.household_invitations(household_id,id);

CREATE TABLE want_keep.household_invitation_audit (
 household_id uuid NOT NULL, invitation_id uuid NOT NULL, revision want_keep.revision NOT NULL,
 actor_id uuid NOT NULL, event text NOT NULL CHECK(event IN ('invitation_issued','invitation_revoked','invitation_accepted')),
 occurred_at timestamptz NOT NULL,
 PRIMARY KEY(household_id,revision),
 FOREIGN KEY(household_id,invitation_id) REFERENCES want_keep.household_invitations(household_id,id),
 FOREIGN KEY(household_id,actor_id) REFERENCES want_keep.memberships(household_id,user_id)
);
CREATE TRIGGER immutable_invitation_audit BEFORE UPDATE OR DELETE ON want_keep.household_invitation_audit
 FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change();

ALTER TABLE want_keep.identity_attempts DROP CONSTRAINT identity_attempts_purpose_check;
ALTER TABLE want_keep.identity_attempts ADD CONSTRAINT identity_attempts_purpose_check
 CHECK(purpose IN ('login','reauthentication','bootstrap','add_passkey','recovery','recovery_grant','invitation'));
ALTER TABLE want_keep.identity_attempts ADD COLUMN invitation_revision want_keep.revision;
ALTER TABLE want_keep.identity_attempts ADD CONSTRAINT invitation_attempt_binding
 CHECK((purpose='invitation')=(invitation_revision IS NOT NULL));
ALTER TABLE want_keep.identity_audit DROP CONSTRAINT identity_audit_event_check;
ALTER TABLE want_keep.identity_audit ADD CONSTRAINT identity_audit_event_check
 CHECK(event IN ('bootstrap_completed','login_completed','reauthentication_completed','passkey_added','passkey_revoked','session_revoked','recovery_started','recovery_completed','recovery_codes_replaced','ceremony_rejected','counter_rejected','invitation_enrollment_completed'));

GRANT SELECT,INSERT ON want_keep.household_invitations,want_keep.household_invitation_audit TO want_keep_app;
GRANT UPDATE(status,accepted_user_id) ON want_keep.household_invitations TO want_keep_app;
GRANT UPDATE(invitation_revision,current_invitation_id) ON want_keep.households TO want_keep_app;
