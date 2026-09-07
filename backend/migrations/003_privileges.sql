CREATE FUNCTION want_keep.reject_history_change() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN RAISE EXCEPTION 'immutable history' USING ERRCODE='55000'; END;
$$;
DO $$
DECLARE relation text;
BEGIN
 FOREACH relation IN ARRAY ARRAY['balance_snapshots','operation_revisions','postings','reservation_revisions','reservations','source_revisions','source_provenance','admission_events','outbox','quarantine'] LOOP
  EXECUTE format('CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.%I FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change()',relation);
 END LOOP;
END;
$$;
GRANT USAGE ON SCHEMA want_keep TO want_keep_app, want_keep_maintenance;
GRANT SELECT,INSERT ON ALL TABLES IN SCHEMA want_keep TO want_keep_app;
GRANT UPDATE ON want_keep.households,want_keep.memberships,want_keep.connections,want_keep.accounts,want_keep.operations,want_keep.goals,want_keep.deployment_admissions TO want_keep_app;
GRANT SELECT ON want_keep.command_tombstones,want_keep.command_details TO want_keep_maintenance;
GRANT DELETE ON want_keep.command_tombstones,want_keep.command_details TO want_keep_maintenance;
REVOKE ALL ON FUNCTION want_keep.reject_history_change() FROM PUBLIC;

GRANT UPDATE(revision,ambiguous,operation_id) ON want_keep.source_records TO want_keep_app;

GRANT UPDATE(status,completed_at,completed_ns,result_type,result_id,result_revision,failure_code) ON want_keep.command_tombstones TO want_keep_app;
GRANT UPDATE(state,attempt,lease_token,lease_until,available_at,cancel_requested,cursor,coverage,gaps) ON want_keep.jobs TO want_keep_app;
