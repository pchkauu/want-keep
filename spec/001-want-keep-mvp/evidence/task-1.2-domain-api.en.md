# Evidence task-1.2 — domain and API

[Русский](task-1.2-domain-api.md)

## Scope

D-37, 2026-09-07: the user authorized independent task-1.2 implementation before parallel research completes. The branch base and PR target are `docs/want-keep-mvp-sdd`. Overall MVP status remains Not Ready.

Implemented exact Money/Asset/Rate, explicit rounding and largest-remainder allocation; calendar types; User/Household/Membership and household/personal-edit policies; knowledge, coverage and freshness states; immutable command transitions and replay/revision checks. Money encapsulates apd v3.2.3. Input strings are limited to 256 characters; display scale never limits source precision. Overflow returns an error instead of truncating. Allocation accepts up to 1000 unique weights and scale 0–254; the total must be exactly representable in that quantum. Floor and half-even are explicit choices.

OpenAPI 3.0.3 describes the agreed API groups, forms, safe errors, household scope and explainable reports. `api/openapi.yaml` is the source; oapi-codegen 2.8.0 creates Go models/strict interfaces and openapi-typescript 7.13.0 creates TypeScript. The latter is isolated in the api package with TS 5.9.3 because its peer requirement is `^5.x`; the application remains on TS 6.0.3. Other application dependencies need no updates.

## Commands and trust

The UUIDv4 Idempotency-Key is created before submission and used for status lookup. Uniqueness is household+actor; operation type and payload hash are immutable. Replay is checked before rechecking an old expectedRevision: an already successful command returns its existing result. An unknown outcome stays pending until reconciliation, never becomes failed or permission to create a new key. Compact status/key/hash/result records live as long as the family; source messages, files, passkeys and recovery codes are not copied into them.

Principal derives from a loaded, verified membership. This is a pure domain policy, not authentication. Real cookie/CSRF checks, database transactions, outbox and HTTP handlers are absent. They are mandatory in task-1.3/task-1.4 and product tasks. A passing schema test does not prove runtime session isolation or atomic exactly-once effects.

Schemas distinguish actorId (User), payerMemberId/allocation (Membership), personalOwnerId and externalAccountOwnerId (User). External asset codes/networks and identity do not replace Money.Asset. Provider DTOs, secrets and unconfirmed endpoints are not implemented.

## Verification and handoff

Executable checks: `make check`, `make check-contracts`, focused Go money/calendar/household/reporting/commands/delivery tests and web API fixtures. Synthetic checks cover six-asset round trips, invalid amounts, allocation, household policy, revision conflict, replay, unknown/partial/stale, schema guards and reproducible Go/TypeScript output. A missing generated artifact must fail validation. Current local and CI results are recorded in the [Issue #12](https://github.com/pchkauu/want-keep/issues/12) delivery report.

Task-1.3 owns atomic command registration and effect+final-status persistence, precision-preserving NUMERIC, durable idempotency and outbox. Task-1.4 implements trusted sessions, family authorization and CSRF; product use cases check current ownership and revision. Errors expose currentRevision only after authorized resource access.

PostgreSQL, banking, payment, browser E2E and deployment checks were not run: those implementations are outside task-1.2. This result does not claim full product AC-002/003/039/059/077/079/090 acceptance.
