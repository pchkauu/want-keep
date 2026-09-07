CREATE TABLE want_keep.identity_bootstrap (
 singleton boolean PRIMARY KEY DEFAULT true CHECK(singleton), token_hash text NOT NULL DEFAULT '',
 expires_at timestamptz NOT NULL DEFAULT '-infinity', initialized boolean NOT NULL DEFAULT false
);
INSERT INTO want_keep.identity_bootstrap(singleton) VALUES(true);

ALTER TABLE want_keep.memberships ADD CONSTRAINT membership_identity UNIQUE(household_id,id,user_id);
CREATE TABLE want_keep.identity_profiles (
 user_id uuid PRIMARY KEY REFERENCES want_keep.users(id), household_id uuid NOT NULL,
 membership_id uuid NOT NULL, handle bytea NOT NULL UNIQUE CHECK(octet_length(handle)=32),
 generation bigint NOT NULL CHECK(generation>0), locale text NOT NULL CHECK(locale IN ('ru','en')),
 reporting_asset want_keep.asset NOT NULL,
 FOREIGN KEY(household_id,membership_id,user_id) REFERENCES want_keep.memberships(household_id,id,user_id)
);
CREATE TABLE want_keep.identity_credentials (
 id uuid PRIMARY KEY, user_id uuid NOT NULL REFERENCES want_keep.identity_profiles(user_id),
 name text NOT NULL CHECK(length(name) BETWEEN 1 AND 2000), rp_id text NOT NULL,
 raw_id bytea NOT NULL UNIQUE CHECK(octet_length(raw_id) BETWEEN 1 AND 1500), public_key bytea NOT NULL,
 aaguid bytea NOT NULL, attestation_object bytea NOT NULL, attestation_client_data bytea NOT NULL,
 attestation_client_hash bytea NOT NULL, attestation_format text NOT NULL, attestation_type text NOT NULL,
 attestation_algorithm bigint NOT NULL, transports text[] NOT NULL,
 sign_count bigint NOT NULL CHECK(sign_count BETWEEN 0 AND 4294967295),
 backup_eligible boolean NOT NULL, backup_state boolean NOT NULL, user_verified boolean NOT NULL,
 created_at timestamptz NOT NULL, revoked boolean NOT NULL DEFAULT false,
 UNIQUE(user_id,id), CHECK(backup_eligible OR NOT backup_state)
);
CREATE INDEX identity_credentials_user ON want_keep.identity_credentials(user_id,id);
CREATE TABLE want_keep.identity_sessions (
 id uuid PRIMARY KEY, user_id uuid NOT NULL REFERENCES want_keep.identity_profiles(user_id),
 token_hash text NOT NULL UNIQUE CHECK(length(token_hash)=64), credential_id uuid NOT NULL,
 name text NOT NULL, created_at timestamptz NOT NULL, authenticated_at timestamptz NOT NULL,
 last_activity_at timestamptz NOT NULL, revoked boolean NOT NULL DEFAULT false,
 UNIQUE(user_id,id), FOREIGN KEY(user_id,credential_id) REFERENCES want_keep.identity_credentials(user_id,id),
 CHECK(authenticated_at>=created_at), CHECK(last_activity_at>=created_at)
);
CREATE INDEX identity_sessions_user ON want_keep.identity_sessions(user_id,id);
CREATE TABLE want_keep.identity_attempts (
 id uuid PRIMARY KEY, browser_hash text NOT NULL CHECK(length(browser_hash)=64), token_hash text NOT NULL DEFAULT '',
 session_hash text NOT NULL DEFAULT '', challenge text NOT NULL, rp_id text NOT NULL, origin text NOT NULL,
 grant_id text NOT NULL DEFAULT '', purpose text NOT NULL CHECK(purpose IN ('login','reauthentication','bootstrap','add_passkey','recovery','recovery_grant')),
 user_id uuid, generation bigint NOT NULL DEFAULT 0 CHECK(generation>=0), created_at timestamptz NOT NULL,
 expires_at timestamptz NOT NULL, consumed boolean NOT NULL DEFAULT false, setup jsonb,
 CHECK(expires_at>created_at)
);
CREATE UNIQUE INDEX identity_grant_token ON want_keep.identity_attempts(token_hash) WHERE token_hash<>'';
CREATE INDEX identity_attempt_expiry ON want_keep.identity_attempts(expires_at,id);
CREATE TABLE want_keep.identity_recovery_codes (
 code_hash text PRIMARY KEY CHECK(length(code_hash)=64), user_id uuid NOT NULL REFERENCES want_keep.identity_profiles(user_id),
 consumed boolean NOT NULL DEFAULT false
);
CREATE INDEX identity_recovery_user ON want_keep.identity_recovery_codes(user_id);
CREATE TABLE want_keep.identity_subscription_bindings (
 id uuid PRIMARY KEY, user_id uuid NOT NULL, session_id uuid NOT NULL, revoked boolean NOT NULL DEFAULT false,
 FOREIGN KEY(user_id,session_id) REFERENCES want_keep.identity_sessions(user_id,id)
);
CREATE INDEX identity_subscriptions_user ON want_keep.identity_subscription_bindings(user_id,session_id);
CREATE TABLE want_keep.identity_rate_limits (
 scope_hash text PRIMARY KEY, window_start timestamptz NOT NULL, attempts integer NOT NULL CHECK(attempts>0)
);
CREATE TABLE want_keep.identity_audit (
 id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY, user_id uuid REFERENCES want_keep.users(id),
 event text NOT NULL CHECK(event IN ('bootstrap_completed','login_completed','reauthentication_completed','passkey_added','passkey_revoked','session_revoked','recovery_started','recovery_completed','recovery_codes_replaced','ceremony_rejected','counter_rejected')),
 resource_id uuid, occurred_at timestamptz NOT NULL
);
CREATE TRIGGER immutable_identity_audit BEFORE UPDATE OR DELETE ON want_keep.identity_audit
 FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change();

GRANT SELECT ON want_keep.identity_bootstrap TO want_keep_app;
GRANT UPDATE(initialized) ON want_keep.identity_bootstrap TO want_keep_app;
GRANT SELECT,INSERT ON want_keep.identity_profiles,want_keep.identity_credentials,want_keep.identity_sessions,
 want_keep.identity_attempts,want_keep.identity_recovery_codes,want_keep.identity_subscription_bindings,
 want_keep.identity_rate_limits,want_keep.identity_audit TO want_keep_app;
GRANT UPDATE(generation) ON want_keep.identity_profiles TO want_keep_app;
GRANT UPDATE(sign_count,backup_state,user_verified,revoked) ON want_keep.identity_credentials TO want_keep_app;
GRANT UPDATE(authenticated_at,last_activity_at,revoked) ON want_keep.identity_sessions TO want_keep_app;
GRANT UPDATE(consumed) ON want_keep.identity_attempts,want_keep.identity_recovery_codes TO want_keep_app;
GRANT UPDATE(revoked) ON want_keep.identity_subscription_bindings TO want_keep_app;
GRANT UPDATE(window_start,attempts) ON want_keep.identity_rate_limits TO want_keep_app;
GRANT USAGE ON SEQUENCE want_keep.identity_audit_id_seq TO want_keep_app;
GRANT SELECT,DELETE ON want_keep.identity_attempts,want_keep.identity_rate_limits TO want_keep_maintenance;
