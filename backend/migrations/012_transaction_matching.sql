ALTER TABLE want_keep.ledger_decisions DROP CONSTRAINT ledger_decisions_kind_check;
ALTER TABLE want_keep.ledger_decisions ADD CHECK(kind IN ('correction','exclusion','undo','automated','matching'));
ALTER TABLE want_keep.ledger_decision_entries DROP CONSTRAINT ledger_decision_entries_fields_check;
ALTER TABLE want_keep.ledger_decision_entries ADD CHECK(cardinality(fields)>0 AND fields <@ ARRAY['principal','fees','occurred_at','payer','merchant','note','accounting','matching']::text[]);
ALTER TABLE want_keep.ledger_field_origins DROP CONSTRAINT ledger_field_origins_field_check;
ALTER TABLE want_keep.ledger_field_origins ADD CHECK(field IN ('principal','fees','occurred_at','payer','merchant','note','accounting','legacy_all','matching'));
ALTER TABLE want_keep.ledger_source_facts DROP CONSTRAINT ledger_source_facts_conflict_check;
ALTER TABLE want_keep.ledger_source_facts ADD CHECK(conflict IN ('','protected_fields','invalid_merge','legacy_protection','matching_conflict'));

CREATE TABLE want_keep.matching_cases (
 household_id uuid NOT NULL REFERENCES want_keep.households(id), id uuid NOT NULL,
 revision want_keep.revision NOT NULL,
 PRIMARY KEY(household_id,id)
);
CREATE TABLE want_keep.matching_revisions (
 household_id uuid NOT NULL, id uuid NOT NULL, revision want_keep.revision NOT NULL,
 primary_id uuid NOT NULL, kind text NOT NULL CHECK(kind IN ('payment','transfer','exchange')),
 state text NOT NULL CHECK(state IN ('clarification','waiting_side','linked','separate','unlinked','conflict')),
 actor_id uuid NOT NULL, at timestamptz NOT NULL, at_ns want_keep.submicro NOT NULL,
 reason text NOT NULL CHECK(length(reason) BETWEEN 1 AND 2000), decision_id uuid,
 candidates_complete boolean NOT NULL,
 PRIMARY KEY(household_id,id,revision),
 FOREIGN KEY(household_id,id) REFERENCES want_keep.matching_cases(household_id,id),
 FOREIGN KEY(household_id,primary_id) REFERENCES want_keep.operations(household_id,id) DEFERRABLE INITIALLY DEFERRED,
 FOREIGN KEY(household_id,actor_id) REFERENCES want_keep.memberships(household_id,user_id),
 FOREIGN KEY(household_id,decision_id) REFERENCES want_keep.ledger_decisions(household_id,id) DEFERRABLE INITIALLY DEFERRED
);
CREATE TABLE want_keep.matching_members (
 household_id uuid NOT NULL, group_id uuid NOT NULL, group_revision want_keep.revision NOT NULL,
 operation_id uuid NOT NULL, operation_revision want_keep.revision NOT NULL,
 PRIMARY KEY(household_id,group_id,group_revision,operation_id),
 FOREIGN KEY(household_id,group_id,group_revision) REFERENCES want_keep.matching_revisions(household_id,id,revision),
 FOREIGN KEY(household_id,operation_id,operation_revision) REFERENCES want_keep.operation_revisions(household_id,operation_id,revision) DEFERRABLE INITIALLY DEFERRED
);
CREATE INDEX matching_members_operation ON want_keep.matching_members(household_id,operation_id,group_id);
CREATE TABLE want_keep.matching_candidates (
 household_id uuid NOT NULL, group_id uuid NOT NULL, group_revision want_keep.revision NOT NULL,
 operation_id uuid NOT NULL, operation_revision want_keep.revision NOT NULL,
 reason text NOT NULL CHECK(length(reason) BETWEEN 1 AND 2000),
 PRIMARY KEY(household_id,group_id,group_revision,operation_id),
 FOREIGN KEY(household_id,group_id,group_revision) REFERENCES want_keep.matching_revisions(household_id,id,revision),
 FOREIGN KEY(household_id,operation_id,operation_revision) REFERENCES want_keep.operation_revisions(household_id,operation_id,revision) DEFERRABLE INITIALLY DEFERRED
);
CREATE TABLE want_keep.ledger_participations (
 household_id uuid NOT NULL, operation_id uuid NOT NULL, revision want_keep.revision NOT NULL,
 group_id uuid NOT NULL, kind text NOT NULL CHECK(kind IN ('payment','transfer','exchange')),
 state text NOT NULL CHECK(state IN ('waiting','linked')),
 PRIMARY KEY(household_id,operation_id,revision),
 FOREIGN KEY(household_id,operation_id,revision) REFERENCES want_keep.operation_revisions(household_id,operation_id,revision),
 FOREIGN KEY(household_id,group_id) REFERENCES want_keep.matching_cases(household_id,id)
);
CREATE TABLE want_keep.ledger_contributions (
 household_id uuid NOT NULL, operation_id uuid NOT NULL, revision want_keep.revision NOT NULL,
 position integer NOT NULL CHECK(position>=0 AND position<1000), carrier_id uuid NOT NULL,
 carrier_position integer NOT NULL CHECK(carrier_position>=0 AND carrier_position<1000),
 effect_state text NOT NULL CHECK(effect_state IN ('draft','pending','posted','reversed','cancelled')), effect_at timestamptz NOT NULL, effect_ns want_keep.submicro NOT NULL,
 component_id text NOT NULL CHECK(length(component_id) BETWEEN 1 AND 100),
 role text NOT NULL CHECK(role IN ('payment','outgoing','incoming','fee')),
 PRIMARY KEY(household_id,operation_id,revision,position),
 FOREIGN KEY(household_id,operation_id,revision) REFERENCES want_keep.ledger_participations(household_id,operation_id,revision),
 FOREIGN KEY(household_id,carrier_id) REFERENCES want_keep.operations(household_id,id) DEFERRABLE INITIALLY DEFERRED
);
CREATE TABLE want_keep.matching_active_members (
 household_id uuid NOT NULL, operation_id uuid NOT NULL, group_id uuid NOT NULL,
 PRIMARY KEY(household_id,operation_id),
 FOREIGN KEY(household_id,operation_id) REFERENCES want_keep.operations(household_id,id) DEFERRABLE INITIALLY DEFERRED,
 FOREIGN KEY(household_id,group_id) REFERENCES want_keep.matching_cases(household_id,id)
);
CREATE TABLE want_keep.matching_carriers (
 household_id uuid NOT NULL, group_id uuid NOT NULL, operation_id uuid NOT NULL,
 position integer NOT NULL, account_id uuid NOT NULL, asset want_keep.asset NOT NULL,
 component_id text NOT NULL CHECK(length(component_id) BETWEEN 1 AND 100),
 role text NOT NULL CHECK(role IN ('payment','outgoing','incoming','fee')),
 PRIMARY KEY(household_id,group_id,component_id),
 UNIQUE(household_id,operation_id,position),
 FOREIGN KEY(household_id,group_id) REFERENCES want_keep.matching_cases(household_id,id),
 FOREIGN KEY(household_id,operation_id) REFERENCES want_keep.operations(household_id,id),
 FOREIGN KEY(household_id,account_id) REFERENCES want_keep.accounts(household_id,id)
);
CREATE TABLE want_keep.ledger_fee_ids (
 household_id uuid NOT NULL, operation_id uuid NOT NULL, revision want_keep.revision NOT NULL,
 position integer NOT NULL, fee_id text NOT NULL CHECK(length(fee_id) BETWEEN 1 AND 2000),
 PRIMARY KEY(household_id,operation_id,revision,position),
 FOREIGN KEY(household_id,operation_id,revision,position) REFERENCES want_keep.postings(household_id,operation_id,revision,position)
);
CREATE TABLE want_keep.ledger_correspondences (
 household_id uuid NOT NULL, operation_id uuid NOT NULL, revision want_keep.revision NOT NULL,
 digest bytea NOT NULL CHECK(octet_length(digest)=32), kind text NOT NULL,
 namespace text NOT NULL, reference text NOT NULL, network text NOT NULL, movement text NOT NULL,
 from_account_id uuid, to_account_id uuid,
 PRIMARY KEY(household_id,operation_id,revision),
 FOREIGN KEY(household_id,operation_id,revision) REFERENCES want_keep.operation_revisions(household_id,operation_id,revision),
 FOREIGN KEY(household_id,from_account_id) REFERENCES want_keep.accounts(household_id,id),
 FOREIGN KEY(household_id,to_account_id) REFERENCES want_keep.accounts(household_id,id)
);
CREATE INDEX ledger_correspondences_identity ON want_keep.ledger_correspondences(household_id,digest);
CREATE INDEX matching_revisions_page ON want_keep.matching_revisions(household_id,at,at_ns,id);

DO $$ DECLARE relation text; BEGIN
 FOREACH relation IN ARRAY ARRAY['matching_revisions','matching_members','matching_candidates','ledger_participations','ledger_contributions','ledger_correspondences','ledger_fee_ids'] LOOP
  EXECUTE format('CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.%I FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change()',relation);
  EXECUTE format('GRANT SELECT,INSERT ON want_keep.%I TO want_keep_app',relation);
 END LOOP;
END $$;
GRANT SELECT,INSERT,UPDATE ON want_keep.matching_cases TO want_keep_app;
GRANT SELECT,INSERT,DELETE ON want_keep.matching_active_members,want_keep.matching_carriers TO want_keep_app;
