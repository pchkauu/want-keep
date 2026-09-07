# Task-1.4 — passkeys and access recovery

The backend/API foundation implements independent member sign-in: WebAuthn, operator bootstrap, cookie sessions and recovery. Base: `2ff2dcd2e7a10732e49b4db5df4c92f824167e1e`; branch: `feat/task-1.4-passkeys-recovery`. D-45 in the [contract](../contracts.en.md) defines policy and lifetimes.

## Implementation

WebAuthn v0.18.0 verifies ES256/RS256, UV, challenge, RP/origin and browser/purpose binding. The library stays in an outer adapter; domain/application do not import its types. PostgreSQL migration 004 persists profiles, credentials, attempts/grants, sessions, recovery hashes, subscription bindings, rate limits and immutable audit. Migrations 001–003 are unchanged.

Bootstrap/recovery complete atomically. Generation and revoked-key checks prevent old access from being applied. Savepoints roll back partial effects of rejected ceremonies. Sessions have 12-hour absolute and 30-minute idle limits; key/code changes require own authentication within 5 minutes. Adding a backup key preserves recovery codes. Recovery revokes only the recovering user’s resources.

## Entry points and operational handoff

From backend: `go run ./cmd/migrate`, `go run ./cmd/identity-bootstrap --output /private/path/bootstrap-token`, then `go run ./cmd/api`. These are local entry points, not deployment proof. The token file uses O_EXCL/0600 and is not overwritten. Unknown issuance outcome requires reconciling the private file and DB before retrying. API does no DDL and uses `want_keep_app`; the operator uses a separate migration DSN.

Configuration: `WANT_KEEP_ENV`, `WANT_KEEP_DATABASE_URL`, `WANT_KEEP_ORIGIN`, `WANT_KEEP_LISTEN_ADDR` (default `127.0.0.1:8080`), `WANT_KEEP_MAX_MEMBERS` (default 2), `WANT_KEEP_TRUSTED_PROXIES` (CIDR list, empty by default). Production origin is exactly `https://want-keep.tech`; local/test use a separate localhost RP. Production DB requires verified TLS. The operator uses `WANT_KEEP_MIGRATION_DATABASE_URL`; maintenance uses `WANT_KEEP_MAINTENANCE_DATABASE_URL`.

Task-8.1 configures a proxy overwriting X-Forwarded-For with one verified IP and X-Forwarded-Proto. Forwarding headers from other peers are ignored. Research nginx and the registered Raiffeisen callback remain unchanged. `go run ./cmd/identity-retention` removes up to 1000 expired rows of each transient type; task-8.1 schedules maintenance every 5 minutes with a separate role. Audit and financial history remain.

## Verification and boundaries

Required: `make check`, `make test-integration AREA=identity`, `make test-integration AREA=storage`, `make test-identity-race`, `make test-storage-race`, `git diff --check`. Missing isolated PostgreSQL fails rather than skipping. Tests use the real WebAuthn verifier, synthetic ES256/RS256 keys, HTTP handlers and PostgreSQL 17.11 at the pinned digest. The verification ledger and CI bind the exact published SHA and are retained separately.

Scenarios: bootstrap/replay/rollback/lost response, two sign-ins and foreign resources, signature/UV/origin/RP/handle/challenge, CSRF/lifetimes/fresh authentication, concurrent recovery/generation, partner preservation, binding revocation, backup keys, pagination, rate limits and restart.

AC-001 covers bootstrap; invitations remain task-1.6. AC-049 covers backend recovery/isolation; real Chrome/Arc/Touch ID remain task-7.1 and final acceptance. AC-050 covers auth without task-1.5 files/provider secrets. AC-072/088 cover persisted own-binding revocation without actual push delivery/notifications. The notification task checks bindings at registration and immediately before sending. Task-7.1 owns UI, memory-only CSRF and foreground activity at most once per minute, without background extension.

Live banking, browser UI E2E and production are not claimed as verified. SDD remains Ready for development. Sources: [WebAuthn](https://www.w3.org/TR/webauthn-3/), [go-webauthn v0.18.0](https://github.com/go-webauthn/webauthn/releases/tag/v0.18.0).
