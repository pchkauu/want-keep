# task-5.1 — OpenAI gateway and spend control

[Русский](task-5.1-openai-gateway.md)

## Outcome and boundaries

The Responses API gateway, exact USD 50 household limit per UTC month, durable AI attempts, worker integration and operator reconciliation of unknown charges are implemented. The selected `gpt-5.6-terra` with `reasoning.effort=xhigh` is materialized from `terra_xhigh` into a checked runtime contract. The historical 206/206 result establishes only the source synthetic qualification: runtime USDC and pseudonymous case-envelope adaptations are locally checked and block production until a separate live qualification. Local and CI tests use a fake HTTP transport; no live OpenAI API or production call was made.

Task-5.1 persists model output as `pending_validation`. It cannot change the financial ledger: authority validation and proposal application remain in task-5.2, receipt processing in task-5.3 and insights in task-5.5. Public OpenAPI is unchanged.

## Executable contract

- `ai/domain` owns exact decimal USD, usage that distinguishes absent from zero `cache_write_tokens`, `$2/$0.20/$2.50/$12` pricing and attempt states. Reservation includes `counted input + 32`, maximum output and conservative cache write. Reasoning is included in output once.
- `gateways/openai` pins `openai-go/v3` v3.56.0, disables SDK retries and sends only agreed Input Tokens and Responses API fields. Generation is foreground with `store=false`, default service tier, explicit cache and disabled truncation and parallel tools. The API key is read only from a private absolute file.
- `scripts/generate-openai-runtime.py` renders the runtime contract from `evidence/openai.prompts.json`; `make check-ai-runtime-contract` compares a temporary regeneration and rejects prompt/schema/config drift. The source-qualification fingerprint and runtime-schema fingerprint remain explicit and distinct. Runtime input contains exactly one pseudonymous qualified case for the ledger revision; the runtime schema supports RUB, USD, USDT, USDC, BTC and ETH. This is locally checked compatibility; `production_admitted=false` rejects real use until a new live evaluation.
- Migration 016 stores immutable attempt identity and append-only states with request/prompt/schema/config fingerprints, permitted input, provider ID, returned model, usage, reservation, actual cost, output and reconciliation. The application role has SELECT/INSERT without UPDATE/DELETE. The maintenance role has no direct AI/job table access and may execute only the constrained `SECURITY DEFINER` reconciliation transition.
- Before a paid call, storage checks the global unresolved barrier, `actual + reserved + unknown`, two slots and the monthly limit under the household lock. It checks the same constraints again after the Input Tokens response and before reservation. Missing/zero count is rejected. The external-started marker commits separately after reservation; provider IO never runs in a transaction.
- A known retryable rejection permits one new attempt; the next such rejection fails the job permanently. A timeout or ambiguous transport outcome after the marker becomes `unknown`, retains provider observation and reservation and leaves the job `unresolved`. Missing/contradictory usage or a returned-model mismatch also becomes `unknown`. Month rollover or changing key/project never clears the barrier. An operator records exact charged cost or proven no charge by request ID and a safe structural evidence reference; a live lease cannot be reconciled. Reconciliation of a known terminal outcome above reservation preserves its outcome/output/validation and never resumes its job. A trusted resumer returns a budget-waiting job to `ready` when its saved reservation fits again or a new UTC month starts; the final reservation check remains atomic.
- Every known provider outcome and its terminal/retry job state commit atomically. Success additionally commits usage, output, `pending_validation` and the job receipt. A job-transition error rolls the settlement back. Recovery/cancellation terminalizes only counting/reserved attempts proven not to have reached the provider; external-started work remains unresolved. Re-running a terminal job never calls the provider again. The OpenAI projection is limited to one ledger revision, replaces internal IDs with stable local pseudonyms and excludes credentials, sessions, raw source payloads and other-household data.

## Verified coverage

| Link | Proven by task-5.1 | Remains with product owner |
| --- | --- | --- |
| AC-018 | Each existing review job gets a version-bound AI attempt; terminal replay does not call the gateway | Application and stale-result UX in task-5.2 |
| AC-022 | Permitted ledger projection, private key file, safe persistence/diagnostics and disclosed retention | Receipts in task-5.3 and production review |
| AC-051 | USD 50, reservations, unknown barrier, waiting and independent financial queues | System status UI in task-7.14 |
| AC-058 | Closed provider/job codes without financial text | Combined operational UI/observability |
| AC-059 | Exact decimal reserve/usage without float | Other financial calculations |
| AC-069 | Refusal/incomplete/schema/known rejection/unknown are distinct and retries bounded | Proposal application in task-5.2 |
| AC-085/090 | Principal and household come from the job and storage revalidates them; household budgets are isolated | Chat authority and UI |

Synthetic checks cover the exact `$49.9 + $0.099999` boundary, barrier recheck between count and reserve, waiting and automatic resumption after UTC rollover, three concurrent calls, a separate household, unknown outcome, live-lease reconciliation rejection, terminal over-reservation reconciliation, least-privilege maintenance, settlement/job rollback, safe orphan-attempt cleanup, provider model/usage evidence, USDC and exact HTTP shape. The mandatory suite fails when isolated PostgreSQL is unavailable.

## Operational handoff

The worker uses `WANT_KEEP_OPENAI_API_KEY_FILE`; after a valid gateway appears on restart it returns `gateway_unavailable` jobs to `ready`. Optional `WANT_KEEP_OPENAI_PROJECT_ID` and test/development `WANT_KEEP_OPENAI_BASE_URL` never change household accounting. Production rejects a custom base URL and the current runtime contract until a new live qualification. Reconciliation uses the maintenance DSN through `go run ./cmd/ai-reconcile --request-id ... --outcome charged|not_charged --actual-usd ... --evidence-ref ...`; `charged` requires an exact positive cost, and no plaintext key or financial text is passed as an argument.

OpenAI may retain abuse-monitoring data for up to 30 days under current API data controls. `store=false` disables normal application-state storage of the response but does not override that policy. Before production, task-8.1 validates secret provisioning, egress, pricing, retention and the operational runbook. A live call requires separate authorization and spend.

The SDD remains **Ready for development**. Task-5.1 does not establish task-5.2, UI or production readiness.
