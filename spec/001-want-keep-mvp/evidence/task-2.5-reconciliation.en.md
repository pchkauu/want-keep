# Task-2.5 — reconciling the ledger to source balances

The imported-account reconciliation backend/API is implemented. Implementation base: `c37d9ed255b01c155d92f3163736bcf57193a25b`; branch: `feat/task-2.5-balance-reconciliation`; task-2.3 is included and task-2.4 is not a dependency. The [contract](../contracts.en.md#task-25--reconciling-the-ledger-to-source-balances) fixes `sourceAsOf` calculation, bounded replay and explicit adjustments. Production was not changed.

The reconciliation domain retains lifecycle, result, replay and four independent components. The application service builds historical projections, deduplicates replay, coordinates admission and creates allowed adjustments through the ledger public contract. Storage persists immutable source observations and migration-010 revisions. Delivery uses generated DTOs, the shared session/Origin/CSRF guard, signed cursors and command recovery.

## Verification matrix

Required commands: `make check`; `make test-integration AREA=reconciliation` and `make test-reconciliation-race`; integration/race for audit, ledger, accounts, storage, identity and household; `make test-integration AREA=privacy`; `git diff --check`. PostgreSQL 17.11 is digest-pinned; the reconciliation suite fails when the database is unavailable. Exact local-candidate and CI results are recorded in the PR.

The reconciliation suite checks exact RUB, USD, USDT, USDC, BTC and ETH, independent owned/available/locked/debt, unknown/partial/stale and historical lifecycle boundaries. Opening 5000, expense 500 and source 4500 are balanced without income. Source 1000 against ledger 900 creates one replay capped at 90 days; after completed/unavailable the server creates adjustment 100 without income/expense. Debt is adjusted independently, while available/locked receive an explicit rejection.

Coverage includes a new observation and superseded lifecycle, re-evaluation after a revision, exclusion and undo, no event for an identical result, admission/reauth failure without a fake job, household isolation and immutable source evidence. Replay preserves the original binding/admission revision/connection generation. Completion occurs inside the exact admitted `CommitPage` transaction, while a terminal provider failure uses the equivalent admitted `CommitFailure`; household-only outcomes are rejected. A delayed source posting uses confirmed `postedAt`, a reversal with unknown transition time remains incomplete, and one multi-posting operation is applied once. Resolution validates the adjustment's actual ledger effect on all four components, including the correlated available change. HTTP checks cover filters, session-bound pagination, CSRF, safe 404, idempotent command recovery and changed payload. The migration test upgrades 009 → 010, reapplies migrations and proves immutable history under the unprivileged role.

## Acceptance boundaries

| Criteria | Proven by task-2.5 | Downstream verification |
| --- | --- | --- |
| AC-004/005 | Exact source/ledger/difference at `sourceAsOf`, historical projection and separate quality | Complete product import and account screen |
| AC-013 | New observations and revisions re-evaluate; identical results do not duplicate | End-to-end reconciliation against a live source |
| AC-040/041 | Replay is 90-day bounded, deduplicated and carries admission/generation; unavailable is explained | Provider IO, reauthentication and action UI |
| AC-058 | Explicit owned/debt adjustment is atomic, is not income/expense and never changes source evidence | SCR-012 user acceptance in Chrome/Arc |

The SDD remains **Ready for development**. Live adapters, provider IO, browser UI and production are not verified here. Operational readiness requires downstream tasks and end-to-end acceptance.
