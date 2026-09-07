# Evidence task-1.3 — storage and transactions

[Русский](task-1.3-storage.md)

## Result and compatibility

Contract 10 now has pgx v5.10.0 storage, sequential SQL migrations and application services for commands, ledger, reservations and admission. The PostgreSQL suite uses 17.11 at a pinned digest. No public HTTP routes or generated DTOs changed. These are the first migrations of an unreleased product; production data was untouched. After application, checksums protect migrations from edits; schema corrections require later files.

The SDD remains **Ready for development**. Task-1.3 does not establish application operational readiness. Publication, review and CI status are recorded in [issue #13](https://github.com/pchkauu/want-keep/issues/13).

## Ownership and internal contracts

| Area | Owner and rule |
|---|---|
| Transactions | Application interfaces; storage hides pgx and transaction context. `WithinHousehold` uses READ COMMITTED, the household row lock and current membership. Nested work shares Store and principal; crossing Stores is rejected. Admission locks always precede household locks. |
| Commands | `commands/application.Executor`: separate short pending registration; replay before expectedRevision; effect, revision, audit, outbox and terminal outcome commit atomically. Callbacks perform DB work only. Infrastructure failure leaves pending for reconciliation; a confirmed business rejection is recorded after rollback. |
| Ledger | `ledger/application.Writer`: immutable revision, previous revision, actor and command reference; differences between revisions update known balances. Unknown stays unknown. Confirmed expenses are not rejected because an existing reserve becomes underfunded. |
| Goals | `goals/application.Service`: personal ownership, current revision, currency and funding account are checked under the household lock. Virtual/dedicated transitions and release of the old reserve are atomic. |
| Sources | `ledger/application.Sources`: D-39 identity includes household/provider/stable account/product/log/record ID; connections/jobs are provenance. Digests accelerate lookup, but full keys and external owners must match. An explicit normalization decision determines correction/ambiguity; hashes cannot authorize corrections. Manual overrides survive reimport. |
| Admission | `connections/admission.Service`: exact binding, provider/host evidence and monotonic revision persist atomically. No-ops preserve revision; rebind never resets it. Snapshot restoration does not replay transitions. |
| Jobs | Bounded batches, SKIP LOCKED, per-lease tokens, deadlines and at most five attempts. An old attempt cannot finish after lease expiry/replacement. Uncertain external effects become unresolved and are not retried automatically. |

User/AI handlers receive no admission write API. The future composition root gives this application service to a trusted operator process; ordinary commands use a validated server principal. SQL privileges do not replace principal validation at the application boundary.

## Precision and persistence

Unscaled `NUMERIC` preserves all six assets and decimal strings up to 256 characters. Boundary conversion uses strings, never floats; NaN/Infinity and unsupported assets fail. Knowledge, coverage and freshness remain separate. UTC Instant uses TIMESTAMPTZ truncated to microseconds plus a 0–999 nanosecond remainder; boundary comparisons use both parts. Date, Month and IANA timezone remain distinct.

Composite foreign keys preserve household scope. Source revisions, postings, balance snapshots, audit, outbox and quarantine reject changes/deletion. Application UPDATE privileges exclude command identity and issued job binding/revision/generation. Commands store no copies of documents, messages or secrets; evidence references point to files managed by subsequent tasks.

## D-41 and recovery

Detail is available for 90 × 24 hours after terminal/reconciled outcome, tombstones for 400 × 24 hours; unresolved commands do not expire. Recent includes terminal commands younger than 30 × 24 hours and all pending commands; its cursor is the final UUID in registration timestamp/ns/ID order. Status is actor-scoped, with independent `ResultAuthorizer` checks for outcome links.

Detail and tombstone cleanup are independent, bounded to 1–1000 rows. Tombstones remain until physical detail cleanup finishes. Detail currently acts as a marker; compact status/hash/outcome belong to the tombstone. Expired detail returns `command_expired` with an authorized link. Financial audit is independent of retention: surviving command references prevent a repeated effect even after tombstone cleanup. Neither `not_found` nor timeout permits a new key.

## D-43 and collector handoff

Admission checks and enqueue are atomic. Issued jobs carry immutable binding, admissionRevision and connection generation. `BeforeRead` runs immediately before IO; IO stays outside transactions. `CommitPage` validates current admitted state, connection generation, lease and input cursor. Source/page, ledger/outbox and the next checkpoint commit atomically. Revoke/change cancels queued jobs and requests cancellation of running ones; a late result goes to quarantine without financial writes or checkpoint advancement.

Task-3.3 connects the real collector to these calls, carries the issued Job unchanged and retains raw evidence before CommitPage. Task-3.2 owns full balance reconciliation; task-4.x normalizes specific platforms. Known balance persistence and ledger deltas are tested synthetically here; complete snapshot/history authority policy remains with reconciliation.

## Checks and evidence limits

- `make check`: Go unit/architecture checks, web/collector lint/typecheck/tests/build, documentation and reproducible OpenAPI.
- `make test-integration AREA=storage`: real isolated PostgreSQL and an unprivileged application role; missing DB fails, test caching is disabled.
- `make test-storage-race`: the same real transaction scenarios under the Go race detector.
- `git diff --check`.

The suite covers empty/repeated/concurrent migrations, invalid-SQL rollback, checksum/unknown-version rejection; six assets, maximum precision, nanoseconds/dates/unknown; household permissions; concurrent duplicate commands and revisions; lost acknowledgements, forced DB disconnect before commit and recovery of one effect; 800+800/1000 and 80+30/100, reservation currencies/modes; D-39 reconnect/corrections/collisions/checkpoint/manual overrides; exact 30/90/400-day boundaries; lease/restart/outbox; concurrent evidence, A→B→A, revoke before/after IO and quarantine.

REQ-012/029/059/061/062/064/069/070/072/076/088 → AC-012/029/059/061/062/086/090/092/106 → task-1.3. AC-012 still needs product AI-link corrections and report recomputation. AC-106 still needs the real collector and deployed provider/host evidence. HTTP/passkey/CSRF, live banking, browser E2E, production deployment and the complete financial lifecycle were not exercised and belong to subsequent tasks.

## Local execution and operations handoff

From the repository root:

```sh
docker compose -f backend/test/integration/storage/compose.yaml up -d --wait
export WANT_KEEP_TEST_DATABASE_URL='postgres://postgres:synthetic-admin@127.0.0.1:55432/want_keep_test?sslmode=disable'
make test-integration AREA=storage
make test-storage-race
```

Compose uses only synthetic credentials, loopback and tmpfs. The suite creates an isolated database per test and never deletes existing databases. Afterwards, `docker compose -f backend/test/integration/storage/compose.yaml down` releases only this test service; tmpfs data is discarded.

Task-8.1 provisions separate migration, application (`want_keep_app`) and maintenance (`want_keep_maintenance`) roles with individual secrets. The latter two must exist before migrations, without SUPERUSER/CREATEDB/CREATEROLE/BYPASSRLS or membership in the migration role. The migration role owns the database/schema; application receives SELECT/INSERT and restricted UPDATE, maintenance SELECT/DELETE for commands only. Commands neither generate production passwords nor escalate privileges. Production requires verified TLS without plaintext fallback; development/test permit loopback only.

```sh
cd backend
# DSNs arrive through protected environment variables; their values are never printed.
WANT_KEEP_ENV=production go run ./cmd/migrate
WANT_KEEP_ENV=production go run ./cmd/command-retention -mode details -batch 100
WANT_KEEP_ENV=production go run ./cmd/command-retention -mode tombstones -batch 100
```

The variables are `WANT_KEEP_MIGRATION_DATABASE_URL` and `WANT_KEEP_MAINTENANCE_DATABASE_URL`. Retention CLI executes one bounded batch. The task-8.x scheduler runs both modes independently, observes errors/lag and drains remaining work under a work limit. Application rollback does not delete schema/history. Production major, secrets, scheduling and monitoring belong to task-8.1/8.3; production was not changed here.

Review: a confirmed correction with the current source revision resolves ambiguity even with an unchanged hash; replay creates no effect. A stale lease/result/cursor is quarantined after page rollback; ordinary storage failures remain errors rather than being treated as successful handling. A PostgreSQL barrier test covers lease expiry during apply.
