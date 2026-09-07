# Task-2.2 — transaction ledger and states

Implemented financial states and monetary effects, manual income/expense API and new transfers/exchanges. Base: `fbbf070337131ad2128332d0c39577a6cb2a46f5`; branch: `feat/task-2.2-transaction-ledger`; task-1.2 and task-2.1 are included. The [contract](../contracts.en.md#task-22--transaction-ledger-and-states) defines the exact boundaries. No dependencies were added, migrations 001–007 are unchanged and production is untouched.

Changes belong to ledger/domain/application, accounts projections, storage and delivery/ledger. Migration 008 retains existing states/postings and adds immutable details with nanosecond precision. Immutable operation revisions with actor/reason/command/previous revision provide financial audit. Independent pending registration and commands.Authenticated session revalidation precede atomic revision, projection, outbox and terminal outcome persistence. Internal import uses CommitPage/admission and D-39, retaining source evidence and human overrides. The transaction.changed event carries ID/revision for the subsequent AI executor; reads emit no events.

## Verification matrix

Required commands: `make check`, `make test-integration AREA=ledger`, `make test-ledger-race`, accounts/storage/identity/household integration/race, `make test-integration AREA=privacy`, `git diff --check`. Isolated PostgreSQL 17.11 is digest-pinned; financial scenarios use the unprivileged application role. Missing DB fails the suite. CI includes ledger integration/race. Exact local and CI results belong to the published candidate and are recorded in the PR/report.

The ledger suite checks six assets through HTTP → Go → PostgreSQL → JSON; 5000 − 500 = 4500; pending → posted/cancelled, posted → reversed, replay and invalid transitions; August purchases posted in September; annual subscriptions and unresolved household facts. It checks transfer 1000 plus fee 10, exchange 9000 RUB → 100 USDT plus fee 50 RUB, a BTC fee; credit purchase/repayment, unknown credit split, net/gross PnL, included fees, unrealized valuation, funding/rewards and subsequent transfers. It checks durable pending, rollback, storage restart, replay after commit, changed payloads, competing revisions and session revocation between registration and execution.

HTTP scenarios cover two households, both members' access to facts, independent payer/actor, CSRF/Origin, unknown fields, exact filters and keyset pagination, cursor binding and actor-only command outcomes. Accepted-receipt references use real persisted metadata/lease transitions; document contents and processor isolation are independently tested by the privacy suite. Import scenarios cover reconnect, states, no repeated event, human override, unknown states/incomplete legs with partial/quarantine, stale admission rejection and preserved bank observations. Migration runs over synthetic version-007 data without invented postedAt, preserving nanoseconds and immutable history.

## Acceptance boundaries

| Criteria | Task-2.2 coverage | Subsequent work |
| --- | --- | --- |
| AC-006/007 | New complete legs, principal and separate fees, exact exchange basis | Task-2.4: matching existing legs and existingTransactions |
| AC-009/011 | States/holds and confirmed purchase month; full subscription recognition | Screens and actual platform states |
| AC-031/035/036 | Owned money/debt, repayment, PnL/funding/rewards without duplicate effects | Card terms, reports, live adapters |
| AC-059/061 | Exact monetary round trips, immutable revisions, rollback/replay/restart, admission fence | Complete lifecycle, allocations and real workers |
| AC-080/082/086 | One household fact, independent actor/payer, no automatic inter-member debt, competing revisions | Shares/budgets, reimbursements, user corrections/undo, AI clarifications |

Task-2.3/2.4/2.6/2.7/2.8 deliver full corrections, linking, categories, refunds and allocations; task-3.2 proves source/projection coverage. Any unreconciled effect prevents confirmed reserve funding from the source, including backdated entries. Task-5.2 runs AI review; task-7.1 delivers FORM-04/05 and screens. SDD remains **Ready for development**. This does not verify live adapters, browser E2E, Touch ID, complete budgets/reports, AI or production readiness.
