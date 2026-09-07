# Evidence task-1.2 — domain and API

[Русский](task-1.2-domain-api.md)

## Scope

D-44, 2026-09-07: independent task-1.2 implementation before research completion was previously labeled D-37 on its branch. D-37 now retains the Alfa decision from task-0.10. Branch base and PR target are `docs/want-keep-mvp-sdd`. The SDD is Ready for development; application operational readiness is not yet established. The user accepted D-41 instead of the earlier family-lifetime command retention.

Implemented exact Money/Asset/Rate, explicit rounding and largest-remainder allocation; calendar types; User/Household/Membership and household/personal-edit policies; knowledge, coverage and freshness states; immutable command transitions and replay/revision checks. Money encapsulates apd v3.2.3. Input strings are limited to 256 characters; display scale never limits source precision. Overflow returns an error instead of truncating. Allocation accepts up to 1000 unique weights and scale 0–254; the total must be exactly representable in that quantum. Floor and half-even are explicit choices.

OpenAPI 3.0.3 describes the agreed API groups, forms, safe errors, household scope and explainable reports. `api/openapi.yaml` is the source; oapi-codegen 2.8.0 creates Go models/strict interfaces and openapi-typescript 7.13.0 creates TypeScript. The latter is isolated in the api package with TS 5.9.3 because its peer requirement is `^5.x`; the application remains on TS 6.0.3. Other application dependencies need no updates.

## Commands and trust

The UUIDv4 Idempotency-Key is created before submission and used for status lookup. Uniqueness is household+actor; operation type and payload hash are immutable. Replay is checked before rechecking an old expectedRevision: an already successful command returns its existing result. An unknown outcome stays pending until reconciliation, never becomes failed or permission to create a new key. Under D-41 detail remains for 90 days after terminal/reconciled outcome, unresolved commands until reconciliation; a tombstone lasts until reconciliation and 400 days after outcome. Source messages, files, passkeys and recovery codes are not copied into it.

Principal derives from a loaded, verified membership. This is a pure domain policy, not authentication. Real cookie/CSRF checks, database transactions, outbox and HTTP handlers are absent. They are mandatory in task-1.3/task-1.4 and product tasks. A passing schema test does not prove runtime session isolation or atomic exactly-once effects.

Schemas distinguish actorId (User), payer.memberId/allocation (Membership), personalOwnerId and externalAccountOwnerId (User). External asset codes/networks and identity do not replace Money.Asset. Provider DTOs, secrets and unconfirmed endpoints are not implemented.

## Verification and handoff

Executable checks: `make check`, `make check-contracts`, focused Go money/calendar/household/reporting/commands/delivery tests and web API fixtures. Synthetic checks cover six-asset round trips, invalid amounts, allocation, household policy, revision conflict, replay, unknown/partial/stale, schema guards and reproducible Go/TypeScript output. A missing generated artifact must fail validation. Current local and CI results are recorded in the [Issue #12](https://github.com/pchkauu/want-keep/issues/12) delivery report.

Task-1.3 owns atomic command registration and effect+final-status persistence, precision-preserving NUMERIC, durable idempotency and outbox. Task-1.4 implements trusted sessions, family authorization and CSRF; product use cases check current ownership and revision. Errors expose currentRevision only after authorized resource access.

PostgreSQL, banking, payment, browser E2E and deployment checks were not run: those implementations are outside task-1.2. This result does not claim full product AC-002/003/039/059/077/079/090 acceptance.

Task-1.2 review clarification: payer is explicit known/memberId, unknown or not_applicable, entered/corrected independently from actor and shares. Existing movement links require each ID/expectedRevision and atomic validation. Plan preview distinguishes create/update/delete and lineId; expectedRevision identifies the Budget aggregate, advanced by every line change/approval. ReturnsReport carries decimal-string dimensionless XIRR ratios, native/reporting basis, dated cash flows and unavailable reasons; task-6.4 still owns the solver. These changes affect unreleased DTOs; both clients regenerate together and no deployed data requires migration.

## Alignment with task-0.10

Contract version 10 combines D-37–D-43 with the D-44 foundation. D-41 replaces family-lifetime command retention. Durations use UTC and 30/90/400 × 24 hours with exclusive upper boundaries. A reconciled outcome starts retention from resolution; pending commands never expire. Command retains compact future tombstone fields; RequireDetail, InRecent and Recover check authorization, time and recovery without execution. HTTP 410 contains a safe error and optional outcome only after result-resource authorization. An expired tombstone yields not_found without proving absence of an effect. Detail cleanup never removes the compact record or financial audit.

D-43's pure policy lives in connections/domain: pending by default, separate provider/host checks, exact binding equality, failed/revoked blocking, stale-check rejection and evidence reset on rebind. Admission revision starts at 1 and increases on each state/evidence/binding change without resetting on rebind; an exact no-op preserves it and overflow beyond 9007199254740991 fails. RequireResult rejects stale binding/revision even after re-admission with the original binding. The boundary exposes the revision only for an existing admission, with no inverse command converter.

task-1.3 owns the application aggregate/repository in connections/admission, persisted monotonic revision, atomic check+enqueue and commit-time revalidation in the source/posting/outbox transaction. task-3.2/task-3.3 and task-4.x carry the issued binding/revision through jobs/results, recheck before IO and quarantine stale results without source/posting. Cancellation of started IO is best effort. Synthetic domain/DTO tests cover revision transitions, no-ops, overflow, A to B to A, revocation/re-admission and untrusted input. This covers REQ-088/AC-106 domain/DTO behavior, not the full integration/security AC.

D-40 read models distinguish observations with source legs, platform quotes with amount/time/fee/spread coverage, and reasoned unavailable results. D-42 fixes string XIRR to 12 places and no_bracket/numeric_error_unbounded reasons. Task-6.1/task-6.4 implement rate calculations and the solver. These DTOs change before product runtime exists; both generated clients update together and no data migration is needed.
