CREATE TABLE want_keep.ledger_decisions (
 household_id uuid NOT NULL, id uuid NOT NULL, actor_id uuid NOT NULL,
 kind text NOT NULL CHECK(kind IN ('correction','exclusion','undo','automated')),
 reason text NOT NULL CHECK(length(reason) BETWEEN 1 AND 2000),
 recorded_at timestamptz NOT NULL, recorded_ns want_keep.submicro NOT NULL, undo_of uuid,
 PRIMARY KEY(household_id,id),
 FOREIGN KEY(household_id,actor_id) REFERENCES want_keep.memberships(household_id,user_id),
 FOREIGN KEY(household_id,undo_of) REFERENCES want_keep.ledger_decisions(household_id,id),
 CHECK((kind='undo')=(undo_of IS NOT NULL))
);
CREATE UNIQUE INDEX ledger_decision_undo ON want_keep.ledger_decisions(household_id,undo_of) WHERE undo_of IS NOT NULL;
CREATE TABLE want_keep.ledger_decision_entries (
 household_id uuid NOT NULL, decision_id uuid NOT NULL, operation_id uuid NOT NULL,
 before_revision want_keep.revision NOT NULL, after_revision want_keep.revision NOT NULL,
 fields text[] NOT NULL CHECK(cardinality(fields)>0 AND fields <@ ARRAY['principal','fees','occurred_at','payer','merchant','note','accounting']::text[]),
 PRIMARY KEY(household_id,decision_id,operation_id),
 FOREIGN KEY(household_id,decision_id) REFERENCES want_keep.ledger_decisions(household_id,id),
 FOREIGN KEY(household_id,operation_id,before_revision) REFERENCES want_keep.operation_revisions(household_id,operation_id,revision),
 FOREIGN KEY(household_id,operation_id,after_revision) REFERENCES want_keep.operation_revisions(household_id,operation_id,revision) DEFERRABLE INITIALLY DEFERRED,
 CHECK(after_revision=before_revision+1)
);
CREATE TABLE want_keep.ledger_revision_audit (
 household_id uuid NOT NULL, operation_id uuid NOT NULL, revision want_keep.revision NOT NULL,
 accounting_state text NOT NULL CHECK(accounting_state IN ('included','excluded')),
 decision_id uuid, recorded_at timestamptz, recorded_ns want_keep.submicro,
 PRIMARY KEY(household_id,operation_id,revision),
 FOREIGN KEY(household_id,operation_id,revision) REFERENCES want_keep.operation_revisions(household_id,operation_id,revision),
 FOREIGN KEY(household_id,decision_id) REFERENCES want_keep.ledger_decisions(household_id,id),
 CHECK((recorded_at IS NULL)=(recorded_ns IS NULL))
);
CREATE TABLE want_keep.ledger_field_origins (
 household_id uuid NOT NULL, operation_id uuid NOT NULL, revision want_keep.revision NOT NULL,
 field text NOT NULL CHECK(field IN ('principal','fees','occurred_at','payer','merchant','note','accounting','legacy_all')),
 changed_revision want_keep.revision NOT NULL, protected boolean NOT NULL, decision_id uuid, protection_revision want_keep.revision,
 PRIMARY KEY(household_id,operation_id,revision,field),
 FOREIGN KEY(household_id,operation_id,revision) REFERENCES want_keep.operation_revisions(household_id,operation_id,revision),
 FOREIGN KEY(household_id,decision_id) REFERENCES want_keep.ledger_decisions(household_id,id),
 CHECK(changed_revision<=revision), CHECK(protected=(protection_revision IS NOT NULL)), CHECK(protection_revision<=revision)
);
INSERT INTO want_keep.ledger_revision_audit(household_id,operation_id,revision,accounting_state)
 SELECT household_id,operation_id,revision,'included' FROM want_keep.operation_revisions;
INSERT INTO want_keep.ledger_field_origins(household_id,operation_id,revision,field,changed_revision,protected,protection_revision)
 SELECT household_id,operation_id,revision,'legacy_all',revision,true,revision FROM want_keep.operation_revisions WHERE human_override;
CREATE TABLE want_keep.ledger_review_requests (
 household_id uuid NOT NULL, operation_id uuid NOT NULL, revision want_keep.revision NOT NULL,
 PRIMARY KEY(household_id,operation_id,revision),
 FOREIGN KEY(household_id,operation_id,revision) REFERENCES want_keep.operation_revisions(household_id,operation_id,revision)
);
CREATE TABLE want_keep.ledger_review_results (
 household_id uuid NOT NULL, operation_id uuid NOT NULL, revision want_keep.revision NOT NULL,
 actor_id uuid NOT NULL, state text NOT NULL CHECK(state IN ('reviewed','clarification','failed')),
 rationale text NOT NULL CHECK(length(rationale) BETWEEN 1 AND 2000),
 payload_hash text NOT NULL CHECK(payload_hash ~ '^[0-9a-f]{64}$'),
 recorded_at timestamptz NOT NULL, recorded_ns want_keep.submicro NOT NULL,
 PRIMARY KEY(household_id,operation_id,revision),
 FOREIGN KEY(household_id,operation_id,revision) REFERENCES want_keep.ledger_review_requests(household_id,operation_id,revision),
 FOREIGN KEY(household_id,actor_id) REFERENCES want_keep.memberships(household_id,user_id)
);
CREATE TABLE want_keep.ledger_source_facts (
 household_id uuid NOT NULL, source_id uuid NOT NULL, source_revision want_keep.revision NOT NULL,
 operation_id uuid NOT NULL, fact jsonb NOT NULL CHECK(jsonb_typeof(fact)='object'),
 conflict text NOT NULL CHECK(conflict IN ('','protected_fields','invalid_merge','legacy_protection')),
 PRIMARY KEY(household_id,source_id,source_revision),
 FOREIGN KEY(household_id,source_id,source_revision) REFERENCES want_keep.source_revisions(household_id,source_id,revision),
 FOREIGN KEY(household_id,operation_id) REFERENCES want_keep.operations(household_id,id) DEFERRABLE INITIALLY DEFERRED
);
CREATE INDEX ledger_source_operation ON want_keep.ledger_source_facts(household_id,operation_id,source_id,source_revision DESC);
CREATE TABLE want_keep.ledger_revision_sources (
 household_id uuid NOT NULL, operation_id uuid NOT NULL, revision want_keep.revision NOT NULL,
 source_id uuid NOT NULL, source_revision want_keep.revision NOT NULL,
 PRIMARY KEY(household_id,operation_id,revision,source_id,source_revision),
 FOREIGN KEY(household_id,operation_id,revision) REFERENCES want_keep.operation_revisions(household_id,operation_id,revision),
 FOREIGN KEY(household_id,source_id,source_revision) REFERENCES want_keep.source_revisions(household_id,source_id,revision)
);
CREATE TABLE want_keep.ledger_decision_evidence (
 household_id uuid NOT NULL, decision_id uuid NOT NULL, kind text NOT NULL CHECK(kind IN ('source','attachment','review')),
 evidence_id uuid NOT NULL, evidence_revision want_keep.revision NOT NULL,
 PRIMARY KEY(household_id,decision_id,kind,evidence_id,evidence_revision),
 FOREIGN KEY(household_id,decision_id) REFERENCES want_keep.ledger_decisions(household_id,id)
);
CREATE TABLE want_keep.ledger_review_evidence (
 household_id uuid NOT NULL, operation_id uuid NOT NULL, revision want_keep.revision NOT NULL,
 kind text NOT NULL CHECK(kind IN ('source','attachment','review')), evidence_id uuid NOT NULL, evidence_revision want_keep.revision NOT NULL,
 PRIMARY KEY(household_id,operation_id,revision,kind,evidence_id,evidence_revision),
 FOREIGN KEY(household_id,operation_id,revision) REFERENCES want_keep.ledger_review_results(household_id,operation_id,revision)
);
DO $$ DECLARE relation text; BEGIN
 FOREACH relation IN ARRAY ARRAY['ledger_decisions','ledger_decision_entries','ledger_revision_audit','ledger_field_origins','ledger_review_requests','ledger_review_results','ledger_source_facts','ledger_decision_evidence','ledger_revision_sources','ledger_review_evidence'] LOOP
  EXECUTE format('CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.%I FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change()',relation);
  EXECUTE format('GRANT SELECT,INSERT ON want_keep.%I TO want_keep_app',relation);
 END LOOP;
END $$;
