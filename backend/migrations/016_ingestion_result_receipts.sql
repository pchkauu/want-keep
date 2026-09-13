CREATE TABLE want_keep.ingestion_result_receipts (
 household_id uuid NOT NULL,
 id uuid NOT NULL,
 job_id uuid NOT NULL,
 lease_token uuid NOT NULL,
 attempt integer NOT NULL CHECK(attempt BETWEEN 1 AND 100),
 input_cursor text NOT NULL CHECK(length(input_cursor) <= 2000),
 evidence_ref text NOT NULL CHECK(length(evidence_ref) BETWEEN 1 AND 2000),
 kind text NOT NULL CHECK(kind IN ('page','provider_outcome','rejected_result','stale_result')),
 recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(household_id,id),
 FOREIGN KEY(household_id,job_id) REFERENCES want_keep.jobs(household_id,id)
);
CREATE INDEX ingestion_result_receipts_job_idx ON want_keep.ingestion_result_receipts(household_id,job_id);
CREATE UNIQUE INDEX ingestion_result_receipts_evidence_idx ON want_keep.ingestion_result_receipts(household_id,job_id,evidence_ref);
CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.ingestion_result_receipts FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change();
GRANT SELECT,INSERT ON want_keep.ingestion_result_receipts TO want_keep_app;
