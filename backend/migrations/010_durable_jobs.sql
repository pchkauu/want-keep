ALTER TABLE want_keep.jobs DROP CONSTRAINT jobs_kind_check;
ALTER TABLE want_keep.jobs ADD CHECK(kind IN ('sync','outbox','ai'));
ALTER TABLE want_keep.jobs DROP CONSTRAINT jobs_state_check;
ALTER TABLE want_keep.jobs ADD CHECK(state IN ('ready','running','waiting','succeeded','failed','canceled','unresolved'));
-- Keep the original connection-binding invariant for both non-source queues.
DO $$ DECLARE c text; BEGIN
 SELECT conname INTO c FROM pg_constraint WHERE conrelid='want_keep.jobs'::regclass AND contype='c' AND pg_get_constraintdef(oid) LIKE '%kind%connection_id%';
 EXECUTE format('ALTER TABLE want_keep.jobs DROP CONSTRAINT %I',c);
END $$;
ALTER TABLE want_keep.jobs ADD CHECK((kind='sync' AND connection_id IS NOT NULL AND connection_generation IS NOT NULL AND binding IS NOT NULL AND admission_revision IS NOT NULL) OR (kind IN ('outbox','ai') AND connection_id IS NULL AND connection_generation IS NULL AND binding IS NULL AND admission_revision IS NULL));
ALTER TABLE want_keep.jobs ADD COLUMN run_deadline timestamptz;
ALTER TABLE want_keep.jobs ADD COLUMN reason text NOT NULL DEFAULT '' CHECK(reason IN ('','handler_unavailable','consumer_unavailable','gateway_unavailable','budget_wait','provider_not_admitted','reauth_required','temporary_failure','permanent_failure','external_unknown','deadline_exceeded','attempts_exhausted','canceled','membership_revoked'));
ALTER TABLE want_keep.jobs ADD COLUMN external_started boolean NOT NULL DEFAULT false;
ALTER TABLE want_keep.jobs ADD COLUMN resource_id uuid;
ALTER TABLE want_keep.jobs ADD COLUMN resource_revision want_keep.revision;
ALTER TABLE want_keep.jobs ADD CHECK((kind='ai' AND resource_id IS NOT NULL AND resource_revision IS NOT NULL) OR (kind!='ai' AND resource_id IS NULL AND resource_revision IS NULL));
ALTER TABLE want_keep.jobs ADD FOREIGN KEY(household_id,resource_id,resource_revision) REFERENCES want_keep.ledger_review_requests(household_id,operation_id,revision);
CREATE UNIQUE INDEX one_review_job ON want_keep.jobs(household_id,resource_id,resource_revision) WHERE kind='ai';
UPDATE want_keep.jobs SET state='canceled' WHERE cancel_requested AND state IN ('ready','running');
DROP INDEX want_keep.one_active_sync;
CREATE UNIQUE INDEX one_active_sync ON want_keep.jobs(household_id,connection_id) WHERE kind='sync' AND state IN ('ready','running','waiting') AND NOT cancel_requested;
CREATE INDEX job_kind_claim ON want_keep.jobs(kind,available_at,id) WHERE state IN ('ready','running');
CREATE TABLE want_keep.sync_progress (
 household_id uuid NOT NULL, connection_id uuid NOT NULL, generation want_keep.revision NOT NULL,
 cursor text NOT NULL DEFAULT '', coverage text NOT NULL DEFAULT 'unavailable' CHECK(coverage IN ('complete','partial','unavailable')), gaps text[] NOT NULL DEFAULT '{}',
 last_job_id uuid, last_success_at timestamptz, completed boolean NOT NULL DEFAULT false,
 PRIMARY KEY(household_id,connection_id), FOREIGN KEY(household_id,connection_id) REFERENCES want_keep.connections(household_id,id),
 FOREIGN KEY(household_id,last_job_id) REFERENCES want_keep.jobs(household_id,id)
);
CREATE TABLE want_keep.sync_schedules (
 household_id uuid NOT NULL, connection_id uuid NOT NULL, next_due timestamptz NOT NULL,
 PRIMARY KEY(household_id,connection_id), FOREIGN KEY(household_id,connection_id) REFERENCES want_keep.connections(household_id,id)
);
INSERT INTO want_keep.sync_schedules SELECT household_id,id,clock_timestamp() FROM want_keep.connections;
-- Deadlines and UUIDs do not order legacy checkpoints. Prefer the one active job in
-- the current generation; ambiguous terminal history requires a conservative replay.
WITH candidates AS (
 SELECT j.*, count(*) OVER(PARTITION BY j.household_id,j.connection_id) AS candidate_count,
 bool_or(j.state IN ('ready','running') AND NOT j.cancel_requested) OVER(PARTITION BY j.household_id,j.connection_id) AS has_active
 FROM want_keep.jobs j JOIN want_keep.connections c ON (c.household_id,c.id,c.generation)=(j.household_id,j.connection_id,j.connection_generation)
 WHERE j.kind='sync'
), selected AS (
 SELECT DISTINCT ON(household_id,connection_id) * FROM candidates
 ORDER BY household_id,connection_id,(state IN ('ready','running') AND NOT cancel_requested) DESC,id
)
INSERT INTO want_keep.sync_progress(household_id,connection_id,generation,cursor,coverage,gaps,last_job_id,completed)
 SELECT household_id,connection_id,connection_generation,
 CASE WHEN has_active OR candidate_count=1 THEN cursor ELSE '' END,
 CASE WHEN has_active OR candidate_count=1 THEN coverage ELSE 'unavailable' END,
 CASE WHEN has_active OR candidate_count=1 THEN gaps ELSE ARRAY['legacy_checkpoint_ambiguous'] END,
 CASE WHEN has_active OR candidate_count=1 THEN id ELSE NULL END,
 (has_active OR candidate_count=1) AND state='succeeded'
 FROM selected;
CREATE TABLE want_keep.job_receipts (
 household_id uuid NOT NULL, job_id uuid NOT NULL, lease_token uuid NOT NULL, attempt integer NOT NULL,
 recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(household_id,job_id), FOREIGN KEY(household_id,job_id) REFERENCES want_keep.jobs(household_id,id)
);
CREATE TABLE want_keep.job_reconciliations (
 household_id uuid NOT NULL, job_id uuid NOT NULL, lease_token uuid NOT NULL, evidence_ref text NOT NULL CHECK(length(evidence_ref) BETWEEN 1 AND 2000),
 outcome text NOT NULL CHECK(outcome IN ('confirmed','absent')), actor_id uuid NOT NULL, recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(household_id,job_id,lease_token), FOREIGN KEY(household_id,job_id) REFERENCES want_keep.jobs(household_id,id),
 FOREIGN KEY(household_id,actor_id) REFERENCES want_keep.memberships(household_id,user_id)
);
CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.job_receipts FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change();
CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.job_reconciliations FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change();
GRANT SELECT,INSERT ON want_keep.sync_progress,want_keep.sync_schedules,want_keep.job_receipts,want_keep.job_reconciliations TO want_keep_app;
GRANT UPDATE ON want_keep.sync_progress,want_keep.sync_schedules TO want_keep_app;
GRANT UPDATE(reason,external_started,run_deadline) ON want_keep.jobs TO want_keep_app;
