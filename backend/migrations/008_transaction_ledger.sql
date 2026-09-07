ALTER TABLE want_keep.operation_revisions DROP CONSTRAINT operation_revisions_state_check;
ALTER TABLE want_keep.operation_revisions ADD CONSTRAINT operation_revisions_state_check CHECK(state IN ('draft','pending','posted','cancelled','reversed'));
ALTER TABLE want_keep.postings ADD COLUMN funding text NOT NULL DEFAULT '' CHECK(funding IN ('','own','credit','unknown'));
ALTER TABLE want_keep.postings ADD COLUMN treatment text NOT NULL DEFAULT '' CHECK(treatment IN ('','movement','included','valuation'));

CREATE TABLE want_keep.transaction_details (
 household_id uuid NOT NULL, operation_id uuid NOT NULL, revision want_keep.revision NOT NULL,
 posted_at timestamptz, posted_ns want_keep.submicro, timezone text NOT NULL,
 origin text NOT NULL CHECK(origin IN ('','manual','source')),
 fee_knowledge text NOT NULL CHECK(fee_knowledge IN ('','known','unknown')),
 pnl_basis text NOT NULL CHECK(pnl_basis IN ('','gross','net')),
 merchant text NOT NULL CHECK(length(merchant)<=2000), note text NOT NULL CHECK(length(note)<=2000),
 attachment_id uuid, allocation_reason text NOT NULL CHECK(length(allocation_reason)<=2000),
 PRIMARY KEY(household_id,operation_id,revision),
 FOREIGN KEY(household_id,operation_id,revision) REFERENCES want_keep.operation_revisions(household_id,operation_id,revision),
 FOREIGN KEY(household_id,attachment_id) REFERENCES want_keep.attachments(household_id,id),
 CHECK((posted_at IS NULL)=(posted_ns IS NULL))
);
CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.transaction_details FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change();
GRANT SELECT,INSERT ON want_keep.transaction_details TO want_keep_app;
CREATE INDEX operation_page ON want_keep.operation_revisions(household_id,occurred_at DESC,occurred_ns DESC,operation_id DESC,revision);
