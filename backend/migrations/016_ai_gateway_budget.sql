CREATE TABLE want_keep.ai_attempts (
 household_id uuid NOT NULL, id uuid NOT NULL, job_id uuid NOT NULL, actor_id uuid NOT NULL,
 resource_id uuid NOT NULL, resource_revision want_keep.revision NOT NULL,
 purpose text NOT NULL CHECK(purpose IN ('transaction_review','receipt_page','chat_insight','complex_clarification')),
 model text NOT NULL CHECK(model='gpt-5.6-terra'), qualification text NOT NULL CHECK(qualification='terra_xhigh'),
 request_fingerprint text NOT NULL CHECK(request_fingerprint ~ '^[0-9a-f]{64}$'),
 prompt_fingerprint text NOT NULL CHECK(prompt_fingerprint ~ '^[0-9a-f]{64}$'),
 schema_fingerprint text NOT NULL CHECK(schema_fingerprint ~ '^[0-9a-f]{64}$'),
 config_fingerprint text NOT NULL CHECK(config_fingerprint ~ '^[0-9a-f]{64}$'),
 allowed_input jsonb NOT NULL CHECK(jsonb_typeof(allowed_input)='array'),
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
 counted_input_tokens bigint CHECK(counted_input_tokens BETWEEN 1 AND 262144),
 reservation_usd numeric CHECK(reservation_usd>=0 AND length(reservation_usd::text)<=128 AND reservation_usd::text NOT IN ('NaN','Infinity','-Infinity')),
 external_started boolean NOT NULL DEFAULT false,
 provider_id text CHECK(provider_id IS NULL OR length(provider_id) BETWEEN 1 AND 200),
 provider_model text CHECK(provider_model IS NULL OR length(provider_model) BETWEEN 1 AND 200),
 input_tokens bigint CHECK(input_tokens>=0), cached_tokens bigint CHECK(cached_tokens>=0),
 cache_write_tokens bigint CHECK(cache_write_tokens>=0), output_tokens bigint CHECK(output_tokens>=0),
 reasoning_tokens bigint CHECK(reasoning_tokens>=0),
 actual_usd numeric CHECK(actual_usd>=0 AND length(actual_usd::text)<=128 AND actual_usd::text NOT IN ('NaN','Infinity','-Infinity')),
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
 CHECK(state NOT IN ('completed','refused','incomplete','schema_error') OR provider_id IS NOT NULL AND provider_model IS NOT NULL),
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

CREATE FUNCTION want_keep.reconcile_ai_attempt(
 p_attempt_id uuid, p_outcome text, p_actual numeric, p_evidence_ref text
) RETURNS void
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $$
DECLARE
 v_family uuid;
 v_job_id uuid;
 v_job record;
 v_state record;
 v_now timestamptz := clock_timestamp();
 v_next_state text;
 v_terminal boolean;
BEGIN
 IF p_attempt_id IS NULL OR p_outcome NOT IN ('charged','not_charged')
    OR p_actual IS NULL OR p_actual < 0 OR length(p_actual::text) > 128
    OR p_actual::text IN ('NaN','Infinity','-Infinity')
    OR p_evidence_ref IS NULL OR length(p_evidence_ref) NOT BETWEEN 1 AND 2000
    OR p_evidence_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/?#=&%+~-]*$'
    OR (p_outcome = 'not_charged' AND p_actual <> 0)
    OR (p_outcome = 'charged' AND p_actual <= 0) THEN
  RAISE EXCEPTION 'invalid AI reconciliation' USING ERRCODE = '22023';
 END IF;

 SELECT a.household_id,a.job_id INTO v_family,v_job_id
 FROM want_keep.ai_attempts a WHERE a.id=p_attempt_id;
 IF NOT FOUND THEN
  RAISE EXCEPTION 'AI attempt not found' USING ERRCODE = 'P0002';
 END IF;

 SELECT j.kind,j.state,j.external_started,j.lease_until INTO v_job
 FROM want_keep.jobs j
 WHERE (j.household_id,j.id)=(v_family,v_job_id)
 FOR UPDATE;
 IF NOT FOUND OR v_job.kind <> 'ai' THEN
  RAISE EXCEPTION 'invalid AI reconciliation job' USING ERRCODE = '22023';
 END IF;

 SELECT s.* INTO v_state
 FROM want_keep.ai_attempt_states s
 WHERE (s.household_id,s.attempt_id)=(v_family,p_attempt_id)
 ORDER BY s.revision DESC LIMIT 1 FOR UPDATE;
 IF NOT FOUND OR NOT v_state.external_started THEN
  RAISE EXCEPTION 'invalid AI reconciliation state' USING ERRCODE = '22023';
 END IF;

 v_terminal := v_state.reconciliation_state='pending'
  AND v_state.state IN ('completed','refused','incomplete','schema_error')
  AND NOT v_job.external_started
  AND ((v_state.state='completed' AND v_job.state='succeeded')
   OR (v_state.state<>'completed' AND v_job.state='failed'));
 IF NOT v_terminal THEN
  IF NOT v_job.external_started OR v_job.state NOT IN ('running','unresolved')
     OR NOT (v_state.reconciliation_state='pending' OR v_state.state='reserved')
     OR v_state.state NOT IN ('reserved','unknown') THEN
   RAISE EXCEPTION 'invalid AI reconciliation state' USING ERRCODE = '22023';
  END IF;
  IF v_job.state='running' AND v_job.lease_until IS NOT NULL AND v_job.lease_until>v_now THEN
   RAISE EXCEPTION 'AI provider call lease is live' USING ERRCODE = '55006';
  END IF;
 END IF;

 INSERT INTO want_keep.ai_attempt_states(
  household_id,attempt_id,revision,state,counted_input_tokens,reservation_usd,
  external_started,provider_id,provider_model,input_tokens,cached_tokens,
  cache_write_tokens,output_tokens,reasoning_tokens,actual_usd,conservative_cost,
  structured_output,validation_state,code,reconciliation_state,evidence_ref,
  recorded_at,recorded_ns
 ) VALUES (
  v_family,p_attempt_id,v_state.revision+1,
  CASE WHEN v_terminal THEN v_state.state ELSE 'unknown' END,v_state.counted_input_tokens,
  v_state.reservation_usd,false,v_state.provider_id,v_state.provider_model,
  v_state.input_tokens,v_state.cached_tokens,v_state.cache_write_tokens,
  v_state.output_tokens,v_state.reasoning_tokens,p_actual,v_state.conservative_cost,
  v_state.structured_output,
  CASE WHEN v_terminal THEN v_state.validation_state ELSE '' END,
  v_state.code,'resolved',p_evidence_ref,v_now,0
 );

 IF NOT v_terminal THEN
  v_next_state := CASE WHEN p_outcome='not_charged' THEN 'ready' ELSE 'failed' END;
  UPDATE want_keep.jobs SET
   state=v_next_state,
   reason=CASE WHEN p_outcome='not_charged' THEN 'temporary_failure' ELSE 'permanent_failure' END,
   external_started=false,lease_token=NULL,lease_until=NULL,
   available_at=CASE WHEN p_outcome='not_charged' THEN v_now ELSE available_at END,
   run_deadline=CASE WHEN p_outcome='not_charged' THEN v_now+INTERVAL '24 hours' ELSE run_deadline END
  WHERE (household_id,id)=(v_family,v_job_id) AND kind='ai' AND state IN ('running','unresolved');
  IF NOT FOUND THEN
   RAISE EXCEPTION 'stale AI reconciliation job' USING ERRCODE = '22023';
  END IF;
 END IF;
END;
$$;

REVOKE ALL ON FUNCTION want_keep.reconcile_ai_attempt(uuid,text,numeric,text) FROM PUBLIC;
GRANT SELECT,INSERT ON want_keep.ai_attempts,want_keep.ai_attempt_states TO want_keep_app;
GRANT EXECUTE ON FUNCTION want_keep.reconcile_ai_attempt(uuid,text,numeric,text) TO want_keep_maintenance;
