ALTER TABLE want_keep.ledger_decision_entries DROP CONSTRAINT ledger_decision_entries_fields_check;
ALTER TABLE want_keep.ledger_decision_entries ADD CHECK(cardinality(fields)>0 AND fields <@ ARRAY['principal','fees','occurred_at','payer','merchant','note','accounting','matching','contribution']::text[]);
ALTER TABLE want_keep.ledger_field_origins DROP CONSTRAINT ledger_field_origins_field_check;
ALTER TABLE want_keep.ledger_field_origins ADD CHECK(field IN ('principal','fees','occurred_at','payer','merchant','note','accounting','legacy_all','matching','contribution'));
ALTER TABLE want_keep.ledger_field_origins ADD CHECK(field<>'contribution' OR NOT protected);

CREATE TABLE want_keep.matching_decision_groups (
 household_id uuid NOT NULL, decision_id uuid NOT NULL,
 group_id uuid NOT NULL, group_revision want_keep.revision NOT NULL,
 PRIMARY KEY(household_id,decision_id,group_id),
 FOREIGN KEY(household_id,decision_id) REFERENCES want_keep.ledger_decisions(household_id,id) DEFERRABLE INITIALLY DEFERRED,
 FOREIGN KEY(household_id,group_id,group_revision) REFERENCES want_keep.matching_revisions(household_id,id,revision)
);
CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.matching_decision_groups FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change();
GRANT SELECT,INSERT ON want_keep.matching_decision_groups TO want_keep_app;
