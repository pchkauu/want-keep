CREATE SCHEMA want_keep;
REVOKE ALL ON SCHEMA want_keep FROM PUBLIC;

CREATE DOMAIN want_keep.revision AS bigint CHECK (VALUE BETWEEN 1 AND 9007199254740991);
CREATE DOMAIN want_keep.asset AS text CHECK (VALUE IN ('RUB','USD','USDT','USDC','BTC','ETH'));
CREATE DOMAIN want_keep.amount AS numeric CHECK (VALUE NOT IN ('NaN'::numeric,'Infinity'::numeric,'-Infinity'::numeric) AND length(VALUE::text) <= 256);
CREATE DOMAIN want_keep.submicro AS smallint CHECK (VALUE BETWEEN 0 AND 999);

CREATE TABLE want_keep.users (id uuid PRIMARY KEY, name text NOT NULL CHECK(length(name) BETWEEN 1 AND 2000));
CREATE TABLE want_keep.households (id uuid PRIMARY KEY, name text NOT NULL, timezone text NOT NULL, max_members integer NOT NULL CHECK(max_members>0));
CREATE TABLE want_keep.memberships (
 household_id uuid NOT NULL REFERENCES want_keep.households(id), id uuid NOT NULL, user_id uuid NOT NULL REFERENCES want_keep.users(id), active boolean NOT NULL,
 PRIMARY KEY(household_id,id), UNIQUE(household_id,user_id)
);
CREATE TABLE want_keep.connections (
 household_id uuid NOT NULL REFERENCES want_keep.households(id), id uuid NOT NULL, provider text NOT NULL,
 external_owner_id uuid NOT NULL, generation want_keep.revision NOT NULL, authorized boolean NOT NULL,
 PRIMARY KEY(household_id,id), FOREIGN KEY(household_id,external_owner_id) REFERENCES want_keep.memberships(household_id,user_id)
);
CREATE TABLE want_keep.external_accounts (
 household_id uuid NOT NULL, id uuid NOT NULL, provider text NOT NULL, stable_id text NOT NULL CHECK(length(stable_id) BETWEEN 1 AND 2000),
 identity_digest bytea NOT NULL CHECK(octet_length(identity_digest)=32), external_owner_id uuid NOT NULL,
 PRIMARY KEY(household_id,id), UNIQUE(household_id,provider,identity_digest),
 FOREIGN KEY(household_id,external_owner_id) REFERENCES want_keep.memberships(household_id,user_id)
);
CREATE TABLE want_keep.accounts (
 household_id uuid NOT NULL REFERENCES want_keep.households(id), id uuid NOT NULL, name text NOT NULL, asset want_keep.asset NOT NULL,
 scope text NOT NULL CHECK(scope IN ('personal','household')), owner_id uuid, product text NOT NULL,
 revision want_keep.revision NOT NULL, opening_date date NOT NULL CHECK(opening_date BETWEEN '0001-01-01' AND '9999-12-31'),
 external_account_id uuid,
 PRIMARY KEY(household_id,id), UNIQUE(household_id,id,asset),
 FOREIGN KEY(household_id,owner_id) REFERENCES want_keep.memberships(household_id,user_id),
 FOREIGN KEY(household_id,external_account_id) REFERENCES want_keep.external_accounts(household_id,id),
 CHECK((scope='personal' AND owner_id IS NOT NULL) OR (scope='household' AND owner_id IS NULL))
);
CREATE UNIQUE INDEX accounts_external_asset ON want_keep.accounts(household_id,external_account_id,asset) WHERE external_account_id IS NOT NULL;
CREATE TABLE want_keep.balance_snapshots (
 household_id uuid NOT NULL, account_id uuid NOT NULL, revision want_keep.revision NOT NULL,
 field text NOT NULL CHECK(field IN ('owned','available','locked','debt')),
 knowledge text NOT NULL CHECK(knowledge IN ('known','unknown','unavailable')), amount want_keep.amount, asset want_keep.asset NOT NULL, reason text NOT NULL,
 coverage text NOT NULL CHECK(coverage IN ('complete','partial','unavailable')), coverage_reasons text[] NOT NULL,
 freshness text NOT NULL CHECK(freshness IN ('fresh','stale','unknown')),
 observed_at timestamptz NOT NULL, observed_ns want_keep.submicro NOT NULL,
 PRIMARY KEY(household_id,account_id,revision,field), FOREIGN KEY(household_id,account_id,asset) REFERENCES want_keep.accounts(household_id,id,asset),
 CHECK((knowledge='known' AND amount IS NOT NULL AND reason='') OR (knowledge!='known' AND amount IS NULL AND length(reason)>0)),
 CHECK((coverage='complete' AND cardinality(coverage_reasons)=0) OR (coverage!='complete' AND cardinality(coverage_reasons)>0))
);
CREATE TABLE want_keep.operations (
 household_id uuid NOT NULL REFERENCES want_keep.households(id), id uuid NOT NULL, revision want_keep.revision NOT NULL,
 PRIMARY KEY(household_id,id)
);
CREATE TABLE want_keep.operation_revisions (
 household_id uuid NOT NULL, operation_id uuid NOT NULL, revision want_keep.revision NOT NULL, previous_revision want_keep.revision,
 actor_id uuid NOT NULL, command_id uuid, reason text NOT NULL CHECK(length(reason) BETWEEN 1 AND 2000),
 economic_type text NOT NULL, state text NOT NULL CHECK(state IN ('draft','pending','posted','reversed')),
 occurred_at timestamptz NOT NULL, occurred_ns want_keep.submicro NOT NULL, cash_date date NOT NULL,
 expense_month date CHECK(expense_month=DATE_TRUNC('month',expense_month)::date), human_override boolean NOT NULL,
 payer_state text NOT NULL CHECK(payer_state IN ('known','unknown','not_applicable')), payer_member_id uuid,
 PRIMARY KEY(household_id,operation_id,revision), FOREIGN KEY(household_id,operation_id) REFERENCES want_keep.operations(household_id,id),
 FOREIGN KEY(household_id,operation_id,previous_revision) REFERENCES want_keep.operation_revisions(household_id,operation_id,revision),
 FOREIGN KEY(household_id,actor_id) REFERENCES want_keep.memberships(household_id,user_id),
 FOREIGN KEY(household_id,payer_member_id) REFERENCES want_keep.memberships(household_id,id),
 CHECK((revision=1 AND previous_revision IS NULL) OR (previous_revision=revision-1)),
 CHECK((payer_state='known' AND payer_member_id IS NOT NULL) OR (payer_state!='known' AND payer_member_id IS NULL))
);
CREATE INDEX operation_command ON want_keep.operation_revisions(household_id,actor_id,command_id) WHERE command_id IS NOT NULL;
CREATE TABLE want_keep.postings (
 household_id uuid NOT NULL, operation_id uuid NOT NULL, revision want_keep.revision NOT NULL, position integer NOT NULL CHECK(position>=0),
 account_id uuid NOT NULL, amount want_keep.amount NOT NULL, asset want_keep.asset NOT NULL,
 role text NOT NULL CHECK(role IN ('principal','fee','interest','funding','pnl','reward')),
 PRIMARY KEY(household_id,operation_id,revision,position),
 FOREIGN KEY(household_id,operation_id,revision) REFERENCES want_keep.operation_revisions(household_id,operation_id,revision),
 FOREIGN KEY(household_id,account_id,asset) REFERENCES want_keep.accounts(household_id,id,asset)
);
CREATE INDEX postings_account ON want_keep.postings(household_id,account_id,operation_id,revision);
CREATE TABLE want_keep.goals (
 household_id uuid NOT NULL REFERENCES want_keep.households(id), id uuid NOT NULL, scope text NOT NULL CHECK(scope IN ('personal','household')), owner_id uuid,
 asset want_keep.asset NOT NULL, revision want_keep.revision NOT NULL, PRIMARY KEY(household_id,id),
 FOREIGN KEY(household_id,owner_id) REFERENCES want_keep.memberships(household_id,user_id),
 CHECK((scope='personal' AND owner_id IS NOT NULL) OR (scope='household' AND owner_id IS NULL))
);
CREATE TABLE want_keep.reservation_revisions (
 household_id uuid NOT NULL, goal_id uuid NOT NULL, revision want_keep.revision NOT NULL, actor_id uuid NOT NULL, command_id uuid,
 reason text NOT NULL CHECK(length(reason) BETWEEN 1 AND 2000), PRIMARY KEY(household_id,goal_id,revision),
 FOREIGN KEY(household_id,goal_id) REFERENCES want_keep.goals(household_id,id), FOREIGN KEY(household_id,actor_id) REFERENCES want_keep.memberships(household_id,user_id)
);
CREATE INDEX reservation_command ON want_keep.reservation_revisions(household_id,actor_id,command_id) WHERE command_id IS NOT NULL;
CREATE TABLE want_keep.reservations (
 household_id uuid NOT NULL, goal_id uuid NOT NULL, revision want_keep.revision NOT NULL, account_id uuid NOT NULL,
 mode text NOT NULL CHECK(mode IN ('virtual','dedicated')), amount want_keep.amount, asset want_keep.asset NOT NULL,
 PRIMARY KEY(household_id,goal_id,revision,account_id), FOREIGN KEY(household_id,goal_id,revision) REFERENCES want_keep.reservation_revisions(household_id,goal_id,revision),
 FOREIGN KEY(household_id,account_id,asset) REFERENCES want_keep.accounts(household_id,id,asset),
 CHECK((mode='virtual' AND amount IS NOT NULL AND amount>=0) OR (mode='dedicated' AND amount IS NULL))
);
CREATE INDEX reservations_account ON want_keep.reservations(household_id,account_id);
CREATE TABLE want_keep.command_tombstones (
 household_id uuid NOT NULL, actor_id uuid NOT NULL, id uuid NOT NULL, kind text NOT NULL, payload_hash text NOT NULL CHECK(payload_hash ~ '^[0-9a-f]{64}$'),
 status text NOT NULL CHECK(status IN ('pending','succeeded','failed')),
 registered_at timestamptz NOT NULL, registered_ns want_keep.submicro NOT NULL, completed_at timestamptz, completed_ns want_keep.submicro,
 result_type text, result_id uuid, result_revision want_keep.revision, failure_code text,
 PRIMARY KEY(household_id,actor_id,id), FOREIGN KEY(household_id,actor_id) REFERENCES want_keep.memberships(household_id,user_id),
 CHECK((status='pending' AND completed_at IS NULL AND completed_ns IS NULL AND result_type IS NULL AND result_id IS NULL AND result_revision IS NULL AND failure_code IS NULL)
 OR (status='succeeded' AND completed_at IS NOT NULL AND completed_ns IS NOT NULL AND result_type IS NOT NULL AND result_id IS NOT NULL AND result_revision IS NOT NULL AND failure_code IS NULL)
 OR (status='failed' AND completed_at IS NOT NULL AND completed_ns IS NOT NULL AND result_type IS NULL AND result_id IS NULL AND result_revision IS NULL AND failure_code IS NOT NULL)),
 CHECK(completed_at IS NULL OR (completed_at,completed_ns)>=(registered_at,registered_ns))
);
CREATE INDEX command_recent ON want_keep.command_tombstones(household_id,actor_id,registered_at DESC,registered_ns DESC,id);
CREATE INDEX command_cleanup ON want_keep.command_tombstones(completed_at,completed_ns) WHERE status!='pending';
CREATE TABLE want_keep.command_details (
 household_id uuid NOT NULL, actor_id uuid NOT NULL, command_id uuid NOT NULL,
 PRIMARY KEY(household_id,actor_id,command_id), FOREIGN KEY(household_id,actor_id,command_id) REFERENCES want_keep.command_tombstones(household_id,actor_id,id)
);
