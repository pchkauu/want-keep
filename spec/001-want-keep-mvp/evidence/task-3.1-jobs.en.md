# task-3.1 — durable jobs

[Русский](task-3.1-jobs.md)

## Result and boundaries

Implemented PostgreSQL scheduling, sync/outbox/ai queues, workers, leases/heartbeats, bounded retries, dependency waiting, local-effect receipts and unknown-external-outcome reconciliation. SDD remains Ready for development. This is task-3.1 infrastructure; real provider/collector/OpenAI handlers, AI monetary reservations, connection HTTP and UI remain with task-3.2/3.3/4.x/5.x/7.x/8.x.

`transaction.changed` produces exactly one AI job per household/operation/revision through the existing ledger review request. Other events remain in the immutable outbox waiting for a consumer; they are neither lost nor reported as delivered. Missing sync/AI handlers produce `waiting/handler_unavailable`; ordinary accounting continues. No paid calls or production provisioning were performed.

## Contracts and compatibility

- `jobs/domain` owns states, reasons, retry policy and progress. `jobs/application` owns workers, scheduling, execution, outbox routing and the trusted reconciliation port. `storage` hides SQL/pgx; `cmd/worker` composes implementations. No new HTTP routes, generated DTOs or dependencies.
- Defaults: one executor per queue, one-second polling, a 60-second lease, a 20-second heartbeat and five attempts. Backoff doubles from five seconds to five minutes, reduced by 0–20% jitter. Scheduler checks every 30 seconds; manual and hourly requests advance the next run by one hour. Missed hours coalesce into one run.
- Execution deadlines are bounded to 24 hours. The original job `deadline` stays immutable; server-owned `run_deadline` is assigned on dependency resume or proven absence of an external effect. Waiting consumes no execution attempts; unresolved work never resumes automatically.
- Workers claim only available slots, reconstruct the principal from persisted actor and current membership, heartbeat and cancel the handler after lease loss. Handlers must honor context cancellation and must not change financial records during `Prepare`.
- `Prepare` performs external work; `Result.Apply` performs only local transactional DB work. Executor checks household/actor/job/attempt/token/target and lease before the effect and at terminal commit. Effect, receipt and acknowledgement are atomic. Source handlers use admission `CommitPage`; generic execution cannot bypass the source fence.
- Before a non-repeatable external action, handlers must call `Execution.BeginExternal`. The marker commits before IO; an unknown response or crash after it becomes unresolved, including on the last attempt. Trusted `ReconcileJob` takes the exact attempt/token, an evidence reference and confirmed/absent. Identical reconciliation is idempotent; conflicting evidence is rejected. Only absent permits a bounded retry; local apply and confirmed outcome are atomic.
- `CompleteReview` binds the result to the AI job's operation/revision and calls the existing ledger service. Stale responses cannot change newer revisions; workers do not independently interpret money.
- Admission locks precede household locks. A source page, postings/outbox and checkpoint commit together; retries and replacement jobs after terminal failure retain unfinished progress. Coverage gaps remain. `last_success_at` changes only on completion; partial coverage stays explicitly partial.
- Disconnect or admission change cancels the old attempt; unknown external effects stay unresolved. Historical uncertainty does not permit new automatic execution. MFA remains the external-account owner's action.
- Failure reasons are closed codes; logs contain queue/code, never secrets, DSNs, cursors, payloads or financial messages.

Confirmed sync reconciliation requires a `Page` with the same evidence reference. After validating current admission and generation, the trusted transaction enables existing source/account application contracts and atomically saves the page, omissions, checkpoint and reconciliation. The last page adds a receipt and success timestamp; an intermediate page continues from the new cursor. Normal running-attempt fencing remains intact. Invalidation/disconnect preserve unknown outcomes even for legacy jobs without an external marker.

## Startup and migration

`010_durable_jobs.sql` follows 009 and retains outbox/history/cursors and existing jobs. Stop old workers before migration; migrate using the operator role, then start the new binary with the application role. Startup neither migrates nor deletes data; there is no automatic schema rollback. `run_deadline` does not weaken SQL protection of the original `deadline`.

Backfill selects the unique active job in the current generation. With no active job and ambiguous terminal history, it never guesses chronology from deadlines: it retains `legacy_checkpoint_ambiguous`, and the next import conservatively replays history from the beginning through existing deduplication. Old job rows and cursors remain available for investigation.

```sh
cd backend
go run ./cmd/migrate
go run ./cmd/worker
```

Environment: `WANT_KEEP_ENV`, `WANT_KEEP_DATABASE_URL`; migrations use the separate `WANT_KEEP_MIGRATION_DATABASE_URL`. DSNs come from the protected environment. Optional `WANT_KEEP_JOB_BINDINGS_FILE` is an operator-owned JSON array of exact non-secret `connections.Binding` values for the running deployment; its environment must match the process. An empty list disables provider scheduling. The file cannot establish admission: a persisted combined provider/host pass is required. Their owning tasks register sync and AI handlers in the composition root. A registered handler automatically resumes only `handler_unavailable`; other reasons are resumed by their trusted owner through `ResumeWaiting` after resolving the cause.

## Verification and handoff

```sh
make bootstrap
make check
make test-integration AREA=jobs
make test-jobs-race
```

The jobs suite uses the repository's isolated PostgreSQL 17.11 and application role with a separate database per test. It covers concurrent workers/schedulers, manual coalescing, lease replacement, bounded retries, unknown outcomes and reconciliation, outbox→AI, revision binding, waiting, checkpoint continuity after failure, cancellation and migration. A child process is killed before commit, after commit and after the external marker. These are local synthetic integration tests; they do not prove platform access or provider idempotency.

Local verification on Go 1.26.5, Node.js 24.19.0 and PostgreSQL 17.11: `make bootstrap`, `make check`, jobs integration/race passed. Affected storage, accounts, identity, household, ledger and audit integration/race, plus the privacy isolation suite passed. SDD checks cover RU/EN, traceability and reproducible task cards; OpenAPI generation consistency and `git diff --check` pass. The test PostgreSQL tmpfs filled during consecutive full runs; only the isolated test container was recreated and the checks passed on rerun.

Actual check results, publication/review/CI and merge SHA are recorded in Issue #26 and its PR. This task unblocks task-3.2 and the task-5.1 dependency. Before external IO, handlers must obey read admission, the explicit external marker and the separate task-5.1 budget contract; unverified gateway capabilities are not claimed as implemented.
