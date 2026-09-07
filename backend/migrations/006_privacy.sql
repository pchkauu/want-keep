ALTER TABLE want_keep.connections ADD COLUMN secret_purpose text NOT NULL DEFAULT 'api_credentials' CHECK(secret_purpose IN ('api_credentials','oauth_tokens','browser_session'));
ALTER TABLE want_keep.jobs ADD COLUMN secret_purpose text NOT NULL DEFAULT '';
UPDATE want_keep.jobs SET secret_purpose='api_credentials' WHERE kind='sync';
ALTER TABLE want_keep.jobs ADD CHECK((kind='sync' AND secret_purpose IN ('api_credentials','oauth_tokens','browser_session')) OR (kind!='sync' AND secret_purpose=''));
CREATE TABLE want_keep.attachments (
 household_id uuid NOT NULL, id uuid NOT NULL, actor_id uuid NOT NULL, account_id uuid NOT NULL,
 object_id uuid NOT NULL UNIQUE, name text NOT NULL CHECK(length(name) BETWEEN 1 AND 2000),
 media_type text NOT NULL CHECK(media_type IN ('image/jpeg','image/png','image/webp','application/pdf')),
 size_bytes bigint NOT NULL CHECK(size_bytes BETWEEN 1 AND 10485760), content_hash text NOT NULL CHECK(content_hash ~ '^[0-9a-f]{64}$'),
 state text NOT NULL CHECK(state IN ('uploaded','validating','accepted','rejected')),
 reason text NOT NULL CHECK(reason IN ('','storage_unavailable','processor_unavailable','invalid_document','unsupported_content','limit_exceeded')),
 original_ready boolean NOT NULL DEFAULT false, created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 attempt integer NOT NULL DEFAULT 0 CHECK(attempt>=0), lease_token uuid, lease_until timestamptz,
 available_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(household_id,id),
 FOREIGN KEY(household_id,actor_id) REFERENCES want_keep.memberships(household_id,user_id),
 FOREIGN KEY(household_id,account_id) REFERENCES want_keep.accounts(household_id,id),
 CHECK(state NOT IN ('validating','accepted','rejected') OR original_ready),
 CHECK(state!='validating' OR (lease_token IS NOT NULL AND lease_until IS NOT NULL)),
 CHECK(state NOT IN ('accepted','validating') OR reason=''),
 CHECK(state!='uploaded' OR reason IN ('','storage_unavailable','processor_unavailable')),
 CHECK(state!='rejected' OR reason IN ('invalid_document','unsupported_content','limit_exceeded'))
);
CREATE INDEX attachment_claim ON want_keep.attachments(available_at,created_at,id) WHERE state IN ('uploaded','validating') AND original_ready;
CREATE TABLE want_keep.attachment_pages (
 household_id uuid NOT NULL, attachment_id uuid NOT NULL, page integer NOT NULL CHECK(page BETWEEN 1 AND 10),
 width integer NOT NULL CHECK(width BETWEEN 1 AND 2048), height integer NOT NULL CHECK(height BETWEEN 1 AND 2048),
 size_bytes bigint NOT NULL CHECK(size_bytes BETWEEN 1 AND 17825792), content_hash text NOT NULL CHECK(content_hash ~ '^[0-9a-f]{64}$'),
 PRIMARY KEY(household_id,attachment_id,page), FOREIGN KEY(household_id,attachment_id) REFERENCES want_keep.attachments(household_id,id)
);
ALTER TABLE want_keep.identity_sessions ADD UNIQUE(token_hash,user_id);
ALTER TABLE want_keep.connections ADD UNIQUE(household_id,id,external_owner_id);
CREATE TABLE want_keep.connection_secret_grants (
 household_id uuid NOT NULL, id uuid NOT NULL, connection_id uuid NOT NULL, owner_id uuid NOT NULL,
 generation want_keep.revision NOT NULL, purpose text NOT NULL CHECK(purpose IN ('api_credentials','oauth_tokens','browser_session')),
 session_hash text NOT NULL,
 expires_at timestamptz NOT NULL, consumed boolean NOT NULL DEFAULT false, revoked boolean NOT NULL DEFAULT false,
 PRIMARY KEY(household_id,id), FOREIGN KEY(household_id,connection_id) REFERENCES want_keep.connections(household_id,id),
 FOREIGN KEY(household_id,owner_id) REFERENCES want_keep.memberships(household_id,user_id),
 FOREIGN KEY(session_hash,owner_id) REFERENCES want_keep.identity_sessions(token_hash,user_id),
 FOREIGN KEY(household_id,connection_id,owner_id) REFERENCES want_keep.connections(household_id,id,external_owner_id)
);
CREATE TABLE want_keep.connection_secrets (
 household_id uuid NOT NULL, connection_id uuid NOT NULL, purpose text NOT NULL CHECK(purpose IN ('api_credentials','oauth_tokens','browser_session')),
 generation want_keep.revision NOT NULL, revision want_keep.revision NOT NULL, ciphertext bytea NOT NULL CHECK(octet_length(ciphertext) BETWEEN 1 AND 22370688),
 revoked boolean NOT NULL DEFAULT false,
 PRIMARY KEY(household_id,connection_id,purpose), FOREIGN KEY(household_id,connection_id) REFERENCES want_keep.connections(household_id,id)
);
CREATE TABLE want_keep.privacy_audit (
 household_id uuid NOT NULL, id uuid NOT NULL, actor_id uuid NOT NULL, resource_id uuid NOT NULL,
 event text NOT NULL CHECK(event IN ('attachment_registered','attachment_ready','attachment_accepted','attachment_rejected','attachment_deferred','secret_grant_created','secret_saved','connection_access_revoked')),
 at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(household_id,id), FOREIGN KEY(household_id,actor_id) REFERENCES want_keep.memberships(household_id,user_id)
);
CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.attachment_pages FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change();
CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.privacy_audit FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change();
GRANT SELECT,INSERT ON want_keep.attachments,want_keep.attachment_pages,want_keep.connection_secret_grants,want_keep.connection_secrets,want_keep.privacy_audit TO want_keep_app;
GRANT UPDATE(state,reason,original_ready,attempt,lease_token,lease_until,available_at) ON want_keep.attachments TO want_keep_app;
GRANT UPDATE(consumed,revoked) ON want_keep.connection_secret_grants TO want_keep_app;
GRANT UPDATE(generation,revision,ciphertext,revoked) ON want_keep.connection_secrets TO want_keep_app;
