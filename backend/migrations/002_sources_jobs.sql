CREATE TABLE want_keep.source_records (
 household_id uuid NOT NULL, id uuid NOT NULL, provider text NOT NULL, external_account_id uuid NOT NULL,
 product text NOT NULL, log text NOT NULL, provider_record_id text NOT NULL, identity_digest bytea NOT NULL CHECK(octet_length(identity_digest)=32),
 revision want_keep.revision NOT NULL, ambiguous boolean NOT NULL, operation_id uuid,
 PRIMARY KEY(household_id,id), UNIQUE(household_id,identity_digest), FOREIGN KEY(household_id,external_account_id) REFERENCES want_keep.external_accounts(household_id,id),
 FOREIGN KEY(household_id,operation_id) REFERENCES want_keep.operations(household_id,id) DEFERRABLE INITIALLY DEFERRED
);
CREATE TABLE want_keep.source_revisions (
 household_id uuid NOT NULL, source_id uuid NOT NULL, revision want_keep.revision NOT NULL, payload_hash text NOT NULL CHECK(payload_hash ~ '^[0-9a-f]{64}$'),
 evidence_ref text NOT NULL CHECK(length(evidence_ref) BETWEEN 1 AND 2000), classification text NOT NULL CHECK(classification IN ('new','correction','ambiguous')),
 fetched_at timestamptz NOT NULL, fetched_ns want_keep.submicro NOT NULL,
 PRIMARY KEY(household_id,source_id,revision), FOREIGN KEY(household_id,source_id) REFERENCES want_keep.source_records(household_id,id)
);
CREATE TABLE want_keep.source_provenance (
 household_id uuid NOT NULL, id uuid NOT NULL, source_id uuid NOT NULL, revision want_keep.revision NOT NULL, connection_id uuid NOT NULL, job_id uuid NOT NULL,
 evidence_ref text NOT NULL,
 PRIMARY KEY(household_id,id), UNIQUE(household_id,source_id,revision,connection_id,job_id,evidence_ref), FOREIGN KEY(household_id,source_id,revision) REFERENCES want_keep.source_revisions(household_id,source_id,revision),
 FOREIGN KEY(household_id,connection_id) REFERENCES want_keep.connections(household_id,id)
);
CREATE TABLE want_keep.deployment_admissions (
 provider text NOT NULL, environment text NOT NULL, binding jsonb NOT NULL, revision want_keep.revision NOT NULL,
 provider_result text, provider_at timestamptz, provider_ns want_keep.submicro,
 host_result text, host_at timestamptz, host_ns want_keep.submicro,
 PRIMARY KEY(provider,environment),
 CHECK(provider_result IN ('passed','failed','revoked')), CHECK(host_result IN ('passed','failed','revoked')),
 CHECK((provider_result IS NULL AND provider_at IS NULL AND provider_ns IS NULL) OR (provider_result IS NOT NULL AND provider_at IS NOT NULL AND provider_ns IS NOT NULL)),
 CHECK((host_result IS NULL AND host_at IS NULL AND host_ns IS NULL) OR (host_result IS NOT NULL AND host_at IS NOT NULL AND host_ns IS NOT NULL))
);
CREATE TABLE want_keep.admission_events (
 provider text NOT NULL, environment text NOT NULL, revision want_keep.revision NOT NULL, snapshot jsonb NOT NULL,
 PRIMARY KEY(provider,environment,revision), FOREIGN KEY(provider,environment) REFERENCES want_keep.deployment_admissions(provider,environment)
);
CREATE TABLE want_keep.jobs (
 household_id uuid NOT NULL, id uuid NOT NULL, actor_id uuid NOT NULL, kind text NOT NULL CHECK(kind IN ('sync','outbox')),
 connection_id uuid, connection_generation want_keep.revision, binding jsonb, admission_revision want_keep.revision,
 state text NOT NULL CHECK(state IN ('ready','running','succeeded','failed','canceled','unresolved')),
 attempt integer NOT NULL DEFAULT 0 CHECK(attempt>=0), max_attempts integer NOT NULL CHECK(max_attempts BETWEEN 1 AND 100),
 lease_token uuid, lease_until timestamptz, available_at timestamptz NOT NULL, deadline timestamptz NOT NULL, cancel_requested boolean NOT NULL DEFAULT false,
 cursor text NOT NULL DEFAULT '', coverage text NOT NULL DEFAULT 'unavailable' CHECK(coverage IN ('complete','partial','unavailable')), gaps text[] NOT NULL DEFAULT '{}',
 PRIMARY KEY(household_id,id), FOREIGN KEY(household_id,actor_id) REFERENCES want_keep.memberships(household_id,user_id),
 FOREIGN KEY(household_id,connection_id) REFERENCES want_keep.connections(household_id,id),
 CHECK((kind='sync' AND connection_id IS NOT NULL AND connection_generation IS NOT NULL AND binding IS NOT NULL AND admission_revision IS NOT NULL)
 OR (kind='outbox' AND connection_id IS NULL AND connection_generation IS NULL AND binding IS NULL AND admission_revision IS NULL)),
 CHECK((state='running' AND lease_token IS NOT NULL AND lease_until IS NOT NULL) OR state!='running')
);
CREATE UNIQUE INDEX one_active_sync ON want_keep.jobs(household_id,connection_id) WHERE kind='sync' AND state IN ('ready','running') AND NOT cancel_requested;
CREATE INDEX job_claim ON want_keep.jobs(available_at,id) WHERE state IN ('ready','running');
CREATE TABLE want_keep.outbox (
 household_id uuid NOT NULL, id uuid NOT NULL, actor_id uuid NOT NULL, resource_type text NOT NULL, resource_id uuid NOT NULL, revision want_keep.revision NOT NULL,
 event_type text NOT NULL, PRIMARY KEY(household_id,id), UNIQUE(household_id,resource_type,resource_id,revision,event_type),
 FOREIGN KEY(household_id,actor_id) REFERENCES want_keep.memberships(household_id,user_id), FOREIGN KEY(household_id,id) REFERENCES want_keep.jobs(household_id,id)
);
ALTER TABLE want_keep.source_provenance ADD FOREIGN KEY(household_id,job_id) REFERENCES want_keep.jobs(household_id,id);
CREATE TABLE want_keep.quarantine (
 household_id uuid NOT NULL, id uuid NOT NULL, job_id uuid NOT NULL, evidence_ref text NOT NULL, reason text NOT NULL,
 PRIMARY KEY(household_id,id), UNIQUE(household_id,job_id,evidence_ref,reason), FOREIGN KEY(household_id,job_id) REFERENCES want_keep.jobs(household_id,id)
);
