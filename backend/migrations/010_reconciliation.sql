ALTER TABLE want_keep.jobs
 ADD COLUMN replay_request_id uuid,
 ADD COLUMN range_from timestamptz,
 ADD COLUMN range_from_ns want_keep.submicro,
 ADD COLUMN range_to timestamptz,
 ADD COLUMN range_to_ns want_keep.submicro,
 ADD CONSTRAINT jobs_replay_range CHECK(
  (replay_request_id IS NULL AND range_from IS NULL AND range_from_ns IS NULL AND range_to IS NULL AND range_to_ns IS NULL)
  OR (kind='sync' AND replay_request_id IS NOT NULL AND range_from IS NOT NULL AND range_from_ns IS NOT NULL AND range_to IS NOT NULL AND range_to_ns IS NOT NULL
      AND (range_from,range_from_ns)<(range_to,range_to_ns)
      AND range_to-range_from<=INTERVAL '90 days')
 );
CREATE UNIQUE INDEX one_replay_job ON want_keep.jobs(household_id,replay_request_id) WHERE replay_request_id IS NOT NULL;

CREATE TABLE want_keep.reconciliations (
 household_id uuid NOT NULL, id uuid NOT NULL, account_id uuid NOT NULL, observation_id uuid NOT NULL,
 current_revision want_keep.revision NOT NULL, lifecycle text NOT NULL CHECK(lifecycle IN ('open','resolved','superseded')),
 PRIMARY KEY(household_id,id), UNIQUE(household_id,id,account_id),
 FOREIGN KEY(household_id,account_id) REFERENCES want_keep.accounts(household_id,id),
 FOREIGN KEY(household_id,observation_id,account_id) REFERENCES want_keep.account_observations(household_id,id,account_id)
);
CREATE UNIQUE INDEX one_open_reconciliation ON want_keep.reconciliations(household_id,account_id) WHERE lifecycle='open';

CREATE TABLE want_keep.reconciliation_replay_requests (
 household_id uuid NOT NULL, id uuid NOT NULL, reconciliation_id uuid NOT NULL, connection_id uuid NOT NULL,
 status text NOT NULL CHECK(status IN ('pending','completed','failed','unavailable')), reason text NOT NULL,
 job_id uuid, range_from timestamptz NOT NULL, range_from_ns want_keep.submicro NOT NULL,
 range_to timestamptz NOT NULL, range_to_ns want_keep.submicro NOT NULL,
 binding jsonb NOT NULL, admission_revision want_keep.revision NOT NULL, connection_generation want_keep.revision NOT NULL,
 PRIMARY KEY(household_id,id), UNIQUE(household_id,reconciliation_id,id),
 FOREIGN KEY(household_id,reconciliation_id) REFERENCES want_keep.reconciliations(household_id,id) DEFERRABLE INITIALLY DEFERRED,
 FOREIGN KEY(household_id,connection_id) REFERENCES want_keep.connections(household_id,id),
 FOREIGN KEY(household_id,job_id) REFERENCES want_keep.jobs(household_id,id),
 CHECK((range_from,range_from_ns)<(range_to,range_to_ns)),
 CHECK(range_to-range_from<=INTERVAL '90 days'),
 CHECK((status='pending' AND reason='') OR (status!='pending' AND length(reason) BETWEEN 1 AND 2000))
);

CREATE TABLE want_keep.reconciliation_revisions (
 household_id uuid NOT NULL, reconciliation_id uuid NOT NULL, revision want_keep.revision NOT NULL,
 lifecycle text NOT NULL CHECK(lifecycle IN ('open','resolved','superseded')),
 result text NOT NULL CHECK(result IN ('balanced','discrepant','incomplete')),
 replay_status text NOT NULL CHECK(replay_status IN ('not_required','pending','completed','failed','unavailable')),
 replay_request_id uuid,
 source_as_of timestamptz NOT NULL, source_as_of_ns want_keep.submicro NOT NULL,
 evaluated_at timestamptz NOT NULL, evaluated_at_ns want_keep.submicro NOT NULL,
 coverage text NOT NULL CHECK(coverage IN ('complete','partial','unavailable')), coverage_reasons text[] NOT NULL,
 freshness text NOT NULL CHECK(freshness IN ('fresh','stale','unknown')),
 PRIMARY KEY(household_id,reconciliation_id,revision),
 FOREIGN KEY(household_id,reconciliation_id) REFERENCES want_keep.reconciliations(household_id,id),
 FOREIGN KEY(household_id,replay_request_id) REFERENCES want_keep.reconciliation_replay_requests(household_id,id) DEFERRABLE INITIALLY DEFERRED,
 CHECK((coverage='complete' AND cardinality(coverage_reasons)=0) OR (coverage!='complete' AND cardinality(coverage_reasons)>0)),
 CHECK((replay_status='not_required' AND replay_request_id IS NULL) OR (replay_status!='not_required' AND replay_request_id IS NOT NULL)),
 CHECK((source_as_of,source_as_of_ns)<=(evaluated_at,evaluated_at_ns))
);

CREATE TABLE want_keep.reconciliation_components (
 household_id uuid NOT NULL, reconciliation_id uuid NOT NULL, revision want_keep.revision NOT NULL,
 component text NOT NULL CHECK(component IN ('owned','available','locked','debt')),
 source_knowledge text NOT NULL CHECK(source_knowledge IN ('known','unknown','unavailable')), source_amount want_keep.amount, source_reason text NOT NULL,
 ledger_knowledge text NOT NULL CHECK(ledger_knowledge IN ('known','unknown','unavailable')), ledger_amount want_keep.amount, ledger_reason text NOT NULL,
 difference_knowledge text NOT NULL CHECK(difference_knowledge IN ('known','unknown','unavailable')), difference_amount want_keep.amount, difference_reason text NOT NULL,
 asset want_keep.asset NOT NULL,
 PRIMARY KEY(household_id,reconciliation_id,revision,component),
 FOREIGN KEY(household_id,reconciliation_id,revision) REFERENCES want_keep.reconciliation_revisions(household_id,reconciliation_id,revision),
 CHECK((source_knowledge='known' AND source_amount IS NOT NULL AND source_reason='') OR (source_knowledge!='known' AND source_amount IS NULL AND length(source_reason)>0)),
 CHECK((ledger_knowledge='known' AND ledger_amount IS NOT NULL AND ledger_reason='') OR (ledger_knowledge!='known' AND ledger_amount IS NULL AND length(ledger_reason)>0)),
 CHECK((difference_knowledge='known' AND difference_amount IS NOT NULL AND difference_reason='') OR (difference_knowledge!='known' AND difference_amount IS NULL AND length(difference_reason)>0))
);

CREATE TABLE want_keep.reconciliation_explanations (
 household_id uuid NOT NULL, reconciliation_id uuid NOT NULL, revision want_keep.revision NOT NULL,
 position integer NOT NULL CHECK(position>=0), code text NOT NULL CHECK(length(code) BETWEEN 1 AND 100), message text NOT NULL CHECK(length(message) BETWEEN 1 AND 2000),
 PRIMARY KEY(household_id,reconciliation_id,revision,position),
 FOREIGN KEY(household_id,reconciliation_id,revision) REFERENCES want_keep.reconciliation_revisions(household_id,reconciliation_id,revision)
);

CREATE TABLE want_keep.reconciliation_operations (
 household_id uuid NOT NULL, reconciliation_id uuid NOT NULL, revision want_keep.revision NOT NULL, operation_id uuid NOT NULL,
 PRIMARY KEY(household_id,reconciliation_id,revision,operation_id),
 FOREIGN KEY(household_id,reconciliation_id,revision) REFERENCES want_keep.reconciliation_revisions(household_id,reconciliation_id,revision),
 FOREIGN KEY(household_id,operation_id) REFERENCES want_keep.operations(household_id,id)
);

CREATE TABLE want_keep.reconciliation_resolutions (
 household_id uuid NOT NULL, reconciliation_id uuid NOT NULL, revision want_keep.revision NOT NULL,
 actor_id uuid NOT NULL, reason text NOT NULL CHECK(length(reason) BETWEEN 1 AND 2000), adjustment_operation_id uuid NOT NULL,
 components text[] NOT NULL CHECK(cardinality(components) BETWEEN 1 AND 2 AND components <@ ARRAY['owned','debt']::text[]),
 recorded_at timestamptz NOT NULL, recorded_at_ns want_keep.submicro NOT NULL,
 PRIMARY KEY(household_id,reconciliation_id,revision),
 FOREIGN KEY(household_id,reconciliation_id,revision) REFERENCES want_keep.reconciliation_revisions(household_id,reconciliation_id,revision),
 FOREIGN KEY(household_id,actor_id) REFERENCES want_keep.memberships(household_id,user_id),
 FOREIGN KEY(household_id,adjustment_operation_id) REFERENCES want_keep.operations(household_id,id)
);

CREATE INDEX reconciliation_page ON want_keep.reconciliation_revisions(household_id,evaluated_at DESC,evaluated_at_ns DESC,reconciliation_id DESC,revision);

DO $$ DECLARE relation text; BEGIN
 FOREACH relation IN ARRAY ARRAY['reconciliation_revisions','reconciliation_components','reconciliation_explanations','reconciliation_operations','reconciliation_resolutions'] LOOP
  EXECUTE format('CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.%I FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change()',relation);
 END LOOP;
END $$;

GRANT SELECT,INSERT ON want_keep.reconciliations,want_keep.reconciliation_replay_requests,want_keep.reconciliation_revisions,want_keep.reconciliation_components,want_keep.reconciliation_explanations,want_keep.reconciliation_operations,want_keep.reconciliation_resolutions TO want_keep_app;
GRANT UPDATE(current_revision,lifecycle) ON want_keep.reconciliations TO want_keep_app;
GRANT UPDATE(status,reason,job_id) ON want_keep.reconciliation_replay_requests TO want_keep_app;
