CREATE TABLE want_keep.allocation_rules (
 household_id uuid NOT NULL,
 id uuid NOT NULL,
 revision want_keep.revision NOT NULL,
 priority integer NOT NULL CHECK(priority BETWEEN 1 AND 1000),
 state text NOT NULL CHECK(state IN ('active','archived')),
 merchant_id uuid,
 category_id uuid,
 actor_id uuid NOT NULL,
 PRIMARY KEY(household_id,id),
 FOREIGN KEY(household_id) REFERENCES want_keep.households(id),
 FOREIGN KEY(household_id,merchant_id) REFERENCES want_keep.merchants(household_id,id),
 FOREIGN KEY(household_id,category_id) REFERENCES want_keep.categories(household_id,id),
 FOREIGN KEY(household_id,actor_id) REFERENCES want_keep.memberships(household_id,user_id),
 CHECK(merchant_id IS NOT NULL OR category_id IS NOT NULL)
);
CREATE INDEX allocation_rules_match ON want_keep.allocation_rules(household_id,state,priority,merchant_id,category_id,id);

CREATE TABLE want_keep.allocation_rule_revisions (
 household_id uuid NOT NULL,
 rule_id uuid NOT NULL,
 revision want_keep.revision NOT NULL,
 priority integer NOT NULL CHECK(priority BETWEEN 1 AND 1000),
 state text NOT NULL CHECK(state IN ('active','archived')),
 merchant_id uuid,
 category_id uuid,
 actor_id uuid NOT NULL,
 command_id uuid,
 recorded_at timestamptz NOT NULL,
 recorded_ns want_keep.submicro NOT NULL,
 PRIMARY KEY(household_id,rule_id,revision),
 FOREIGN KEY(household_id,rule_id) REFERENCES want_keep.allocation_rules(household_id,id) DEFERRABLE INITIALLY DEFERRED,
 FOREIGN KEY(household_id,merchant_id) REFERENCES want_keep.merchants(household_id,id),
 FOREIGN KEY(household_id,category_id) REFERENCES want_keep.categories(household_id,id),
 FOREIGN KEY(household_id,actor_id) REFERENCES want_keep.memberships(household_id,user_id),
 CHECK(merchant_id IS NOT NULL OR category_id IS NOT NULL)
);
CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.allocation_rule_revisions FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change();

CREATE TABLE want_keep.allocation_rule_shares (
 household_id uuid NOT NULL,
 rule_id uuid NOT NULL,
 revision want_keep.revision NOT NULL,
 position integer NOT NULL CHECK(position BETWEEN 0 AND 999),
 member_id uuid NOT NULL,
 share want_keep.amount NOT NULL CHECK(share>0 AND share<=100),
 PRIMARY KEY(household_id,rule_id,revision,position),
 UNIQUE(household_id,rule_id,revision,member_id),
 FOREIGN KEY(household_id,rule_id,revision) REFERENCES want_keep.allocation_rule_revisions(household_id,rule_id,revision),
 FOREIGN KEY(household_id,member_id) REFERENCES want_keep.memberships(household_id,id)
);
CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.allocation_rule_shares FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change();

CREATE TABLE want_keep.ledger_allocation_snapshots (
 household_id uuid NOT NULL,
 operation_id uuid NOT NULL,
 revision want_keep.revision NOT NULL,
 position integer NOT NULL CHECK(position BETWEEN 0 AND 1000),
 item_id uuid,
 state text NOT NULL CHECK(state IN ('resolved','partial','unresolved','not_applicable')),
 purpose text NOT NULL CHECK(purpose IN ('','personal','shared')),
 mode text NOT NULL CHECK(mode IN ('','amounts','shares','equal','unresolved','composite')),
 origin text NOT NULL CHECK(origin IN ('explicit_purchase','explicit_item','rule','equal_default','unresolved','mixed','not_applicable')),
 reason text NOT NULL CHECK(length(reason)<=2000),
 fallback_mode text NOT NULL DEFAULT '' CHECK(fallback_mode IN ('','amounts','shares','equal','unresolved')),
 fallback_purpose text NOT NULL DEFAULT '' CHECK(fallback_purpose IN ('','personal','shared')),
 fallback_origin text NOT NULL DEFAULT '' CHECK(fallback_origin IN ('','explicit_purchase','explicit_item','rule','equal_default','unresolved')),
 fallback_reason text NOT NULL DEFAULT '' CHECK(length(fallback_reason)<=2000),
 PRIMARY KEY(household_id,operation_id,revision,position),
 UNIQUE(household_id,operation_id,revision,item_id),
 FOREIGN KEY(household_id,operation_id,revision) REFERENCES want_keep.operation_revisions(household_id,operation_id,revision),
 FOREIGN KEY(household_id,operation_id,revision,item_id) REFERENCES want_keep.receipt_items(household_id,operation_id,revision,id),
 CHECK((position=0 AND item_id IS NULL) OR (position>0 AND item_id IS NOT NULL)),
 CHECK((state='not_applicable' AND purpose='' AND mode='' AND origin='not_applicable' AND reason='') OR state<>'not_applicable'),
 CHECK((mode='composite')=(fallback_mode<>'')),
 CHECK((fallback_mode='unresolved' AND fallback_purpose='' AND fallback_origin='unresolved' AND length(fallback_reason)>0) OR (fallback_mode IN ('amounts','shares','equal') AND fallback_purpose IN ('personal','shared') AND fallback_origin<>'' ) OR fallback_mode='')
);
CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.ledger_allocation_snapshots FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change();

CREATE TABLE want_keep.ledger_allocation_inputs (
 household_id uuid NOT NULL,
 operation_id uuid NOT NULL,
 revision want_keep.revision NOT NULL,
 snapshot_position integer NOT NULL,
 position integer NOT NULL CHECK(position BETWEEN 0 AND 999),
 member_id uuid NOT NULL,
 amount want_keep.amount,
 asset want_keep.asset,
 share want_keep.amount,
 PRIMARY KEY(household_id,operation_id,revision,snapshot_position,position),
 UNIQUE NULLS NOT DISTINCT(household_id,operation_id,revision,snapshot_position,member_id,asset),
 FOREIGN KEY(household_id,operation_id,revision,snapshot_position) REFERENCES want_keep.ledger_allocation_snapshots(household_id,operation_id,revision,position),
 FOREIGN KEY(household_id,member_id) REFERENCES want_keep.memberships(household_id,id),
 CHECK((amount IS NOT NULL AND asset IS NOT NULL AND share IS NULL AND amount>=0) OR (amount IS NULL AND asset IS NULL AND share IS NOT NULL AND share>0) OR (amount IS NULL AND asset IS NULL AND share IS NULL))
);
CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.ledger_allocation_inputs FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change();

CREATE TABLE want_keep.ledger_allocation_fallback_inputs (
 household_id uuid NOT NULL,
 operation_id uuid NOT NULL,
 revision want_keep.revision NOT NULL,
 snapshot_position integer NOT NULL DEFAULT 0 CHECK(snapshot_position=0),
 position integer NOT NULL CHECK(position BETWEEN 0 AND 999),
 member_id uuid NOT NULL,
 amount want_keep.amount,
 asset want_keep.asset,
 share want_keep.amount,
 PRIMARY KEY(household_id,operation_id,revision,position),
 UNIQUE NULLS NOT DISTINCT(household_id,operation_id,revision,member_id,asset),
 FOREIGN KEY(household_id,operation_id,revision,snapshot_position) REFERENCES want_keep.ledger_allocation_snapshots(household_id,operation_id,revision,position),
 FOREIGN KEY(household_id,member_id) REFERENCES want_keep.memberships(household_id,id),
 CHECK((amount IS NOT NULL AND asset IS NOT NULL AND share IS NULL AND amount>=0) OR (amount IS NULL AND asset IS NULL AND share IS NOT NULL AND share>0) OR (amount IS NULL AND asset IS NULL AND share IS NULL))
);
CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.ledger_allocation_fallback_inputs FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change();

CREATE TABLE want_keep.ledger_allocation_member_amounts (
 household_id uuid NOT NULL,
 operation_id uuid NOT NULL,
 revision want_keep.revision NOT NULL,
 snapshot_position integer NOT NULL,
 member_id uuid NOT NULL,
 asset want_keep.asset NOT NULL,
 amount want_keep.amount NOT NULL CHECK(amount>0),
 PRIMARY KEY(household_id,operation_id,revision,snapshot_position,member_id,asset),
 FOREIGN KEY(household_id,operation_id,revision,snapshot_position) REFERENCES want_keep.ledger_allocation_snapshots(household_id,operation_id,revision,position),
 FOREIGN KEY(household_id,member_id) REFERENCES want_keep.memberships(household_id,id)
);
CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.ledger_allocation_member_amounts FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change();

CREATE TABLE want_keep.ledger_allocation_unallocated (
 household_id uuid NOT NULL,
 operation_id uuid NOT NULL,
 revision want_keep.revision NOT NULL,
 snapshot_position integer NOT NULL,
 asset want_keep.asset NOT NULL,
 amount want_keep.amount NOT NULL CHECK(amount>0),
 PRIMARY KEY(household_id,operation_id,revision,snapshot_position,asset),
 FOREIGN KEY(household_id,operation_id,revision,snapshot_position) REFERENCES want_keep.ledger_allocation_snapshots(household_id,operation_id,revision,position)
);
CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.ledger_allocation_unallocated FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change();

CREATE TABLE want_keep.ledger_allocation_rule_refs (
 household_id uuid NOT NULL,
 operation_id uuid NOT NULL,
 revision want_keep.revision NOT NULL,
 snapshot_position integer NOT NULL,
 rule_id uuid NOT NULL,
 rule_revision want_keep.revision NOT NULL,
 PRIMARY KEY(household_id,operation_id,revision,snapshot_position,rule_id),
 FOREIGN KEY(household_id,operation_id,revision,snapshot_position) REFERENCES want_keep.ledger_allocation_snapshots(household_id,operation_id,revision,position),
 FOREIGN KEY(household_id,rule_id,rule_revision) REFERENCES want_keep.allocation_rule_revisions(household_id,rule_id,revision)
);
CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.ledger_allocation_rule_refs FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change();

CREATE TABLE want_keep.ledger_allocation_fallback_rule_refs (
 household_id uuid NOT NULL,
 operation_id uuid NOT NULL,
 revision want_keep.revision NOT NULL,
 snapshot_position integer NOT NULL DEFAULT 0 CHECK(snapshot_position=0),
 rule_id uuid NOT NULL,
 rule_revision want_keep.revision NOT NULL,
 PRIMARY KEY(household_id,operation_id,revision,rule_id),
 FOREIGN KEY(household_id,operation_id,revision,snapshot_position) REFERENCES want_keep.ledger_allocation_snapshots(household_id,operation_id,revision,position),
 FOREIGN KEY(household_id,rule_id,rule_revision) REFERENCES want_keep.allocation_rule_revisions(household_id,rule_id,revision)
);
CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.ledger_allocation_fallback_rule_refs FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change();

ALTER TABLE want_keep.ledger_decision_entries DROP CONSTRAINT ledger_decision_entries_fields_check;
ALTER TABLE want_keep.ledger_decision_entries ADD CONSTRAINT ledger_decision_entries_fields_check
 CHECK(cardinality(fields)>0 AND fields <@ ARRAY['principal','fees','occurred_at','payer','merchant','note','accounting','category','merchant_identity','receipt_items','matching','contribution','allocation']::text[]);
ALTER TABLE want_keep.ledger_field_origins DROP CONSTRAINT ledger_field_origins_field_check;
ALTER TABLE want_keep.ledger_field_origins ADD CONSTRAINT ledger_field_origins_field_check
 CHECK(field IN ('principal','fees','occurred_at','payer','merchant','note','accounting','category','merchant_identity','receipt_items','matching','contribution','allocation','legacy_all'));

CREATE TEMP TABLE allocation_backfill_components ON COMMIT DROP AS
SELECT r.household_id,r.operation_id,r.revision,p.position,p.role,p.asset,-p.amount AS amount
FROM want_keep.operation_revisions r
JOIN want_keep.ledger_revision_audit a USING(household_id,operation_id,revision)
JOIN want_keep.postings p USING(household_id,operation_id,revision)
LEFT JOIN want_keep.ledger_participations participation USING(household_id,operation_id,revision)
LEFT JOIN want_keep.ledger_contributions contribution USING(household_id,operation_id,revision,position)
WHERE a.accounting_state='included'
 AND r.economic_type<>'income'
 AND p.amount<0 AND p.treatment IN ('','movement')
 AND (r.economic_type='expense' AND p.role='principal' OR p.role IN ('fee','interest'))
 AND (participation.operation_id IS NULL OR participation.state='retained' OR participation.state='linked' AND contribution.carrier_id=r.operation_id)
 AND NOT COALESCE(participation.state='linked' AND participation.kind IN ('transfer','exchange') AND p.role='principal',false);

INSERT INTO want_keep.ledger_allocation_snapshots(household_id,operation_id,revision,position,state,purpose,mode,origin,reason,fallback_mode,fallback_purpose,fallback_origin,fallback_reason)
SELECT r.household_id,r.operation_id,r.revision,0,
 CASE WHEN EXISTS(SELECT 1 FROM allocation_backfill_components p WHERE (p.household_id,p.operation_id,p.revision)=(r.household_id,r.operation_id,r.revision)) THEN 'unresolved' ELSE 'not_applicable' END,
 '',
 CASE WHEN EXISTS(SELECT 1 FROM allocation_backfill_components p WHERE (p.household_id,p.operation_id,p.revision)=(r.household_id,r.operation_id,r.revision)) THEN 'unresolved' ELSE '' END,
 CASE WHEN EXISTS(SELECT 1 FROM allocation_backfill_components p WHERE (p.household_id,p.operation_id,p.revision)=(r.household_id,r.operation_id,r.revision)) THEN 'unresolved' ELSE 'not_applicable' END,
 CASE WHEN EXISTS(SELECT 1 FROM allocation_backfill_components p WHERE (p.household_id,p.operation_id,p.revision)=(r.household_id,r.operation_id,r.revision)) THEN 'migrated_unresolved' ELSE '' END,
 '','','',''
FROM want_keep.operation_revisions r;

INSERT INTO want_keep.ledger_allocation_snapshots(household_id,operation_id,revision,position,item_id,state,purpose,mode,origin,reason)
SELECT i.household_id,i.operation_id,i.revision,i.position+1,i.id,
 CASE WHEN r.economic_type='expense' AND i.gross>i.discount AND EXISTS(SELECT 1 FROM allocation_backfill_components p WHERE (p.household_id,p.operation_id,p.revision)=(i.household_id,i.operation_id,i.revision) AND p.role='principal') THEN 'unresolved' ELSE 'not_applicable' END,
 '',
 CASE WHEN r.economic_type='expense' AND i.gross>i.discount AND EXISTS(SELECT 1 FROM allocation_backfill_components p WHERE (p.household_id,p.operation_id,p.revision)=(i.household_id,i.operation_id,i.revision) AND p.role='principal') THEN 'unresolved' ELSE '' END,
 CASE WHEN r.economic_type='expense' AND i.gross>i.discount AND EXISTS(SELECT 1 FROM allocation_backfill_components p WHERE (p.household_id,p.operation_id,p.revision)=(i.household_id,i.operation_id,i.revision) AND p.role='principal') THEN 'unresolved' ELSE 'not_applicable' END,
 CASE WHEN r.economic_type='expense' AND i.gross>i.discount AND EXISTS(SELECT 1 FROM allocation_backfill_components p WHERE (p.household_id,p.operation_id,p.revision)=(i.household_id,i.operation_id,i.revision) AND p.role='principal') THEN 'migrated_unresolved' ELSE '' END
FROM want_keep.receipt_items i
JOIN want_keep.operation_revisions r USING(household_id,operation_id,revision);

INSERT INTO want_keep.ledger_allocation_unallocated(household_id,operation_id,revision,snapshot_position,asset,amount)
SELECT household_id,operation_id,revision,0,asset,SUM(amount)
FROM allocation_backfill_components
GROUP BY household_id,operation_id,revision,asset;

INSERT INTO want_keep.ledger_allocation_unallocated(household_id,operation_id,revision,snapshot_position,asset,amount)
SELECT i.household_id,i.operation_id,i.revision,i.position+1,i.asset,i.gross-i.discount
FROM want_keep.receipt_items i
JOIN want_keep.operation_revisions r USING(household_id,operation_id,revision)
WHERE r.economic_type='expense' AND i.gross>i.discount
 AND EXISTS(SELECT 1 FROM allocation_backfill_components p WHERE (p.household_id,p.operation_id,p.revision)=(i.household_id,i.operation_id,i.revision) AND p.role='principal');

DROP TABLE allocation_backfill_components;

GRANT SELECT,INSERT ON want_keep.allocation_rule_revisions,want_keep.allocation_rule_shares,want_keep.ledger_allocation_snapshots,want_keep.ledger_allocation_inputs,want_keep.ledger_allocation_fallback_inputs,want_keep.ledger_allocation_member_amounts,want_keep.ledger_allocation_unallocated,want_keep.ledger_allocation_rule_refs,want_keep.ledger_allocation_fallback_rule_refs TO want_keep_app;
GRANT SELECT,INSERT ON want_keep.allocation_rules TO want_keep_app;
GRANT UPDATE(revision,priority,state,merchant_id,category_id,actor_id) ON want_keep.allocation_rules TO want_keep_app;
