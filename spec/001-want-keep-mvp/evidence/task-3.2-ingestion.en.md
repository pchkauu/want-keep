# Task-3.2 — connector ingestion contract

Internal ingestion contract version 10 is implemented. Implementation base: `cedbe3ba8384071e0e958167d37899ade557c0bc`; branch: `feat/task-3.2-ingestion-contracts`. Dependencies task-3.1, task-1.2, task-2.1 and task-2.2 are included. Production and live platforms were not changed.

## Proven result

`collector/contracts/v10/ingestion.openapi.yaml` is the single wire-model source. `make generate-contracts` creates Go transport DTOs in `backend/internal/integrations/contract/generated` and TypeScript types in `collector/src/contracts/generated`; `make check-contracts` reproduces both outputs in a temporary directory. Strict Go/TypeScript codecs enforce limits, unknown-field rejection, trailing JSON, base64/digest, decimal strings, discriminators, capability manifests and exact server-issued job/binding/admission-revision echoes.

`integrations/domain` contains provider-neutral values; generated and provider DTOs remain at the contract boundary. `integrations/application.Service` runs the pre-read admission fence, persists raw evidence before the financial commit and applies a page only through `admission.CommitPage`. The server assigns evidence/fetch/operation/source revision, principal and internal account IDs. The application resolves an account descriptor independently from an observation; balances/postings use a descriptor from the same self-contained page. A D-39 source key excludes amount, time, connection, cursor and localized text.

The synthetic golden page covers RUB, USD, USDT, USDC, BTC, ETH and unsupported USDC.E; distinct products/accounts, a card alias, known/unknown, partial coverage, exact fractions, a pending expense, net trade P&L with an included fee/valuation and a mining reward. Reconnect/replay duplicates no account, opening, observation, source revision or posting. Evidence-storage failure never opens a financial commit; stale admission retains evidence in quarantine without a checkpoint or financial effect.

## Verification matrix

Required commands are `make check`; `make check-contracts`; `make test-collector FILTER=contracts`; `make test-integration AREA=all`; ingestion/jobs/storage/accounts/ledger/audit/matching/reconciliation race suites; and `git diff --check`. The PostgreSQL suite uses isolated local PostgreSQL 17.11 under the unprivileged role and fails without `WANT_KEEP_TEST_DATABASE_URL`. Exact candidate and CI outcomes are recorded in the PR.

Tests cover an account-only page without an invented observation, evidence-before-commit, a typed MFA failure with no financial apply, replay and concurrent delivery, exact PostgreSQL NUMERIC, family/job-derived actor, stable account/source identities and stale-result quarantine. Golden fixtures assert financial meaning instead of JSON shape alone.

## Acceptance boundaries

| Criteria | Proven by task-3.2 | Downstream verification |
| --- | --- | --- |
| AC-004/005/039 | Exact values for six assets, independent knownness/coverage/freshness, separate owned/available/locked/debt/credit limit and explicit unsupported assets | Complete product reports and FX valuation |
| AC-008/009/035 | D-39 dedup/revisions and typed pending/posted/cancelled, fee/funding/reward/P&L mappings | Live provider lifecycle and matching |
| AC-041 | Cursor/completion/gaps and one-page atomic checkpoint | Live multi-page provider replay |
| AC-048/087 | Read capabilities only; typed reauth/MFA/CAPTCHA; stale generation/lease cannot apply | Live route allowlist, session-owner handoff and UI |
| AC-062/079/090/106 | Domain/transport boundaries, server-owned identity/principal, exact admission binding/revision and commit-time quarantine | Deployment conformance and production admission |

The SDD remains **Ready for development**. Live API/Playwright connectors, authorized live routes, UI and production are not verified here; task-3.3/task-4.x/task-8.x own that evidence.
