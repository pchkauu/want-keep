ALTER TABLE want_keep.accounts ADD COLUMN network_digest bytea NOT NULL DEFAULT sha256(''::bytea) CHECK(octet_length(network_digest)=32), ADD COLUMN network text NOT NULL DEFAULT '' CHECK(length(network)<=2000), ADD COLUMN external_asset_code text NOT NULL DEFAULT '' CHECK(length(external_asset_code)<=2000);
DROP INDEX want_keep.accounts_external_asset;
CREATE UNIQUE INDEX accounts_external_product_asset ON want_keep.accounts(household_id,external_account_id,product,asset,network_digest) WHERE external_account_id IS NOT NULL;
ALTER TABLE want_keep.balance_snapshots ADD COLUMN basis text NOT NULL DEFAULT 'legacy' CHECK(basis IN ('legacy','ledger'));
CREATE TABLE want_keep.account_openings (
 household_id uuid NOT NULL,account_id uuid NOT NULL,revision want_keep.revision NOT NULL,
 operation_id uuid,date date NOT NULL CHECK(date BETWEEN '0001-01-01' AND '9999-12-31'),timezone text NOT NULL,confirmed boolean NOT NULL,
 actor_id uuid NOT NULL,reason text NOT NULL CHECK(length(reason) BETWEEN 1 AND 2000),at timestamptz NOT NULL,at_ns want_keep.submicro NOT NULL,
 PRIMARY KEY(household_id,account_id,revision),
 FOREIGN KEY(household_id,account_id) REFERENCES want_keep.accounts(household_id,id),
 FOREIGN KEY(household_id,operation_id) REFERENCES want_keep.operations(household_id,id),
 FOREIGN KEY(household_id,actor_id) REFERENCES want_keep.memberships(household_id,user_id),
 CHECK(NOT confirmed OR operation_id IS NOT NULL)
);
CREATE TABLE want_keep.opening_amounts (
 household_id uuid NOT NULL,account_id uuid NOT NULL,revision want_keep.revision NOT NULL,
 field text NOT NULL CHECK(field IN ('owned','available','locked','debt')),
 knowledge text NOT NULL CHECK(knowledge IN ('known','unknown','unavailable')),amount want_keep.amount,asset want_keep.asset NOT NULL,reason text NOT NULL,
 PRIMARY KEY(household_id,account_id,revision,field),
 FOREIGN KEY(household_id,account_id,revision) REFERENCES want_keep.account_openings(household_id,account_id,revision),
 FOREIGN KEY(household_id,account_id,asset) REFERENCES want_keep.accounts(household_id,id,asset),
 CHECK((knowledge='known' AND amount IS NOT NULL AND reason='') OR (knowledge!='known' AND amount IS NULL AND length(reason)>0))
);
CREATE TABLE want_keep.account_observations (
 household_id uuid NOT NULL,id uuid NOT NULL,account_id uuid NOT NULL,connection_id uuid NOT NULL,job_id uuid NOT NULL,evidence_ref text NOT NULL CHECK(length(evidence_ref) BETWEEN 1 AND 2000),
 as_of timestamptz NOT NULL,as_of_ns want_keep.submicro NOT NULL,fetched_at timestamptz NOT NULL,fetched_ns want_keep.submicro NOT NULL,
 own_available boolean NOT NULL,coverage text NOT NULL CHECK(coverage IN ('complete','partial','unavailable')),coverage_reasons text[] NOT NULL,freshness text NOT NULL CHECK(freshness IN ('fresh','stale','unknown')),
 PRIMARY KEY(household_id,id),UNIQUE(household_id,id,account_id),
 FOREIGN KEY(household_id,account_id) REFERENCES want_keep.accounts(household_id,id),
 FOREIGN KEY(household_id,connection_id) REFERENCES want_keep.connections(household_id,id),
 FOREIGN KEY(household_id,job_id) REFERENCES want_keep.jobs(household_id,id),
 CHECK((as_of,as_of_ns)<=(fetched_at,fetched_ns)),
 CHECK((coverage='complete' AND cardinality(coverage_reasons)=0) OR (coverage!='complete' AND cardinality(coverage_reasons)>0))
);
CREATE INDEX account_observations_latest ON want_keep.account_observations(household_id,account_id,as_of DESC,as_of_ns DESC,fetched_at DESC,fetched_ns DESC,id);
CREATE TABLE want_keep.observation_amounts (
 household_id uuid NOT NULL,observation_id uuid NOT NULL,account_id uuid NOT NULL,
 field text NOT NULL CHECK(field IN ('owned','available','locked','debt','credit_limit')),
 knowledge text NOT NULL CHECK(knowledge IN ('known','unknown','unavailable')),amount want_keep.amount,asset want_keep.asset NOT NULL,reason text NOT NULL,
 PRIMARY KEY(household_id,observation_id,field),
 FOREIGN KEY(household_id,observation_id,account_id) REFERENCES want_keep.account_observations(household_id,id,account_id),
 FOREIGN KEY(household_id,account_id,asset) REFERENCES want_keep.accounts(household_id,id,asset),
 CHECK((knowledge='known' AND amount IS NOT NULL AND reason='') OR (knowledge!='known' AND amount IS NULL AND length(reason)>0))
);
CREATE TABLE want_keep.card_aliases (
 household_id uuid NOT NULL,id uuid NOT NULL,account_id uuid NOT NULL,label text NOT NULL CHECK(length(label) BETWEEN 1 AND 100),last_four text NOT NULL CHECK(last_four ~ '^[0-9]{4}$'),
 PRIMARY KEY(household_id,id),FOREIGN KEY(household_id,account_id) REFERENCES want_keep.accounts(household_id,id)
);
CREATE TABLE want_keep.account_events (
 household_id uuid NOT NULL,id uuid NOT NULL,account_id uuid NOT NULL,revision want_keep.revision NOT NULL,actor_id uuid NOT NULL,command_id uuid,
 kind text NOT NULL CHECK(kind IN ('created','opening_corrected','ownership_changed')),reason text NOT NULL CHECK(length(reason) BETWEEN 1 AND 2000),
 origin text NOT NULL CHECK(origin IN ('interactive','live_sync','historical_backfill')),eligible boolean NOT NULL,
 at timestamptz NOT NULL,at_ns want_keep.submicro NOT NULL,
 PRIMARY KEY(household_id,id),UNIQUE(household_id,account_id,revision,kind),
 FOREIGN KEY(household_id,account_id) REFERENCES want_keep.accounts(household_id,id),
 FOREIGN KEY(household_id,actor_id) REFERENCES want_keep.memberships(household_id,user_id),
 CHECK(NOT eligible OR (kind='created' AND origin!='historical_backfill'))
);
DO $$ DECLARE relation text; BEGIN
 FOREACH relation IN ARRAY ARRAY['account_openings','opening_amounts','account_observations','observation_amounts','card_aliases','account_events'] LOOP
 EXECUTE format('CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.%I FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change()',relation);
 EXECUTE format('GRANT SELECT,INSERT ON want_keep.%I TO want_keep_app',relation);
 END LOOP;
END $$;
