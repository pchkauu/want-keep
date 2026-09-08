CREATE TABLE want_keep.ai_attempts (
 household_id uuid NOT NULL, id uuid NOT NULL, job_id uuid NOT NULL, actor_id uuid NOT NULL,
 resource_id uuid NOT NULL, resource_revision want_keep.revision NOT NULL,
 purpose text NOT NULL CHECK(purpose IN ('transaction_review','receipt_page','chat_insight','complex_clarification')),
 model text NOT NULL CHECK(model='gpt-5.6-terra'), qualification text NOT NULL CHECK(qualification='terra_xhigh'),
 request_fingerprint text NOT NULL CHECK(request_fingerprint ~ '^[0-9a-f]{64}$'),
 prompt_fingerprint text NOT NULL CHECK(prompt_fingerprint ~ '^[0-9a-f]{64}$'),
 schema_fingerprint text NOT NULL CHECK(schema_fingerprint ~ '^[0-9a-f]{64}$'),
 config_fingerprint text NOT NULL CHECK(config_fingerprint ~ '^[0-9a-f]{64}$'),
 allowed_input jsonb NOT NULL CHECK(jsonb_typeof(allowed_input)='object'),
 maximum_output_tokens bigint NOT NULL CHECK(maximum_output_tokens BETWEEN 1 AND 8192),
 budget_month date NOT NULL CHECK(budget_month=date_trunc('month',budget_month)::date),
 created_at timestamptz NOT NULL, created_ns want_keep.submicro NOT NULL,
 PRIMARY KEY(household_id,id), UNIQUE(id), UNIQUE(household_id,job_id,id),
 FOREIGN KEY(household_id,job_id) REFERENCES want_keep.jobs(household_id,id),
 FOREIGN KEY(household_id,actor_id) REFERENCES want_keep.memberships(household_id,user_id),
 FOREIGN KEY(household_id,resource_id,resource_revision) REFERENCES want_keep.ledger_review_requests(household_id,operation_id,revision)
);

CREATE TABLE want_keep.ai_attempt_states (
 household_id uuid NOT NULL, attempt_id uuid NOT NULL, revision want_keep.revision NOT NULL,
 state text NOT NULL CHECK(state IN ('counting','reserved','completed','refused','incomplete','schema_error','known_rejection','unknown')),
 counted_input_tokens bigint CHECK(counted_input_tokens BETWEEN 0 AND 262144),
 reservation_usd numeric CHECK(reservation_usd>=0 AND reservation_usd::text NOT IN ('NaN','Infinity','-Infinity')),
 external_started boolean NOT NULL DEFAULT false,
 provider_id text CHECK(provider_id IS NULL OR length(provider_id) BETWEEN 1 AND 200),
 input_tokens bigint CHECK(input_tokens>=0), cached_tokens bigint CHECK(cached_tokens>=0),
 cache_write_tokens bigint CHECK(cache_write_tokens>=0), output_tokens bigint CHECK(output_tokens>=0),
 reasoning_tokens bigint CHECK(reasoning_tokens>=0),
 actual_usd numeric CHECK(actual_usd>=0 AND actual_usd::text NOT IN ('NaN','Infinity','-Infinity')),
 conservative_cost boolean NOT NULL DEFAULT false,
 structured_output jsonb CHECK(structured_output IS NULL OR jsonb_typeof(structured_output)='object'),
 validation_state text NOT NULL DEFAULT '' CHECK(validation_state IN ('','pending_validation')),
 code text NOT NULL DEFAULT '' CHECK(length(code)<=100),
 reconciliation_state text NOT NULL DEFAULT 'none' CHECK(reconciliation_state IN ('none','pending','resolved')),
 evidence_ref text CHECK(evidence_ref IS NULL OR length(evidence_ref) BETWEEN 1 AND 2000),
 recorded_at timestamptz NOT NULL, recorded_ns want_keep.submicro NOT NULL,
 PRIMARY KEY(household_id,attempt_id,revision),
 FOREIGN KEY(household_id,attempt_id) REFERENCES want_keep.ai_attempts(household_id,id),
 CHECK(cached_tokens IS NULL OR input_tokens IS NOT NULL AND cached_tokens<=input_tokens),
 CHECK(cache_write_tokens IS NULL OR input_tokens IS NOT NULL AND cached_tokens IS NOT NULL AND cache_write_tokens<=input_tokens-cached_tokens),
 CHECK(reasoning_tokens IS NULL OR output_tokens IS NOT NULL AND reasoning_tokens<=output_tokens),
	CHECK((state='counting')=(counted_input_tokens IS NULL AND reservation_usd IS NULL)),
	CHECK(state='counting' OR counted_input_tokens IS NOT NULL OR state IN ('known_rejection','refused')),
	CHECK(state IN ('counting','known_rejection','refused') OR reservation_usd IS NOT NULL),
	CHECK((input_tokens IS NULL AND cached_tokens IS NULL AND output_tokens IS NULL AND reasoning_tokens IS NULL) OR (input_tokens IS NOT NULL AND cached_tokens IS NOT NULL AND output_tokens IS NOT NULL AND reasoning_tokens IS NOT NULL)),
	CHECK(state NOT IN ('completed','incomplete','schema_error') OR actual_usd IS NOT NULL),
 CHECK((validation_state='pending_validation')=(state='completed')),
 CHECK((reconciliation_state='resolved')=(evidence_ref IS NOT NULL)),
 CHECK(NOT external_started OR state IN ('reserved','completed','refused','incomplete','schema_error','known_rejection','unknown'))
);

CREATE INDEX ai_attempt_job ON want_keep.ai_attempts(household_id,job_id,created_at,id);
CREATE INDEX ai_attempt_budget ON want_keep.ai_attempts(household_id,budget_month,id);
CREATE INDEX ai_attempt_state_latest ON want_keep.ai_attempt_states(household_id,attempt_id,revision DESC);

CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.ai_attempts
 FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change();
CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.ai_attempt_states
 FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change();

GRANT SELECT,INSERT ON want_keep.ai_attempts,want_keep.ai_attempt_states TO want_keep_app;
GRANT SELECT,INSERT ON want_keep.ai_attempts,want_keep.ai_attempt_states TO want_keep_maintenance;
GRANT SELECT ON want_keep.jobs TO want_keep_maintenance;
GRANT UPDATE(state,reason,external_started,lease_token,lease_until,available_at,run_deadline) ON want_keep.jobs TO want_keep_maintenance;
