# Task-2.4 — transfer links and one payment

The backend/API implements matching cases, links, effect participation and compound undo. Base: `ff9dcc79eff6c8d35a38aedb39b95532b48672b6`; branch: `feat/task-2.4-transfer-matching`. Task-2.3 is included. Contract version 10 is extended in [contracts.en.md](../contracts.en.md); migrations 011 and 012 are additive. No dependencies are added.

Matching owns search/composition, ledger owns monetary effects/decisions, accounts owns projections. Proven correspondence requires complete namespace/ID and compatible monetary sides; amount/date/file hash alone create clarification. A waiting possible duplicate adds no second effect. Original records remain independent. User confirmation, group corrections and undo use all revisions and the existing authenticated command transaction. Bank facts use the same JournalWriter inside CommitPage. Normalized input contains verified structured data only; this task does not implement real parsers, collectors or OpenAI.

## Verification

Required matrix: `make check`; `make test-integration AREA=matching`; `make test-matching-race`; audit/ledger/accounts/storage/identity/household integration/race; `make test-integration AREA=privacy`; `git diff --check`. Matching integration and race are in CI; missing PostgreSQL fails the suite. Tests use digest-pinned PostgreSQL 17.11, isolated databases and the unprivileged role. Exact results belong to the published SHA and are recorded in the PR/final report.

Matching tests cover HTTP → Go → SQL → JSON for six assets; no second effect, link/separate, history, restart/undo, separate BTC/ETH fees, both arrival orders and missing counterparts. Fenced source tests cover pending → posted, hold removal, late fees, source revision replay, independent reversal, conflicting amounts without a new effect and quarantine after admission revocation. Structured IDs are searched beyond the probable window; different blockchain movements and accounts remain distinct.

HTTP tests cover a manual transfer and two bank cases, compound amount corrections and undo preserving a later independent note, compound exclusion/undo, cursor scope, household isolation, field spoofing and Origin/CSRF. Two members answer concurrently through a barrier: one outcome wins. Rollback retains no partial revisions; after restart the original key recovers the outcome and changed payloads are rejected. A normalized receipt with a synthetic accepted attachment and a bank fact retain authorship and multiple evidence references for one RUB 300 expense. This is not actual file recognition testing.

Additional checks passed for the existing-transfer API: RUB 1000 + fee 10 and RUB 9000 → USDT 100 + fee RUB 50, incomplete composition rejection without changes, same-key replay, truncated searches above 100 candidates, separate-purchase undo retaining candidates, migration over 009 and matching history after D-41 cleanup. The base includes PR #79: 26 local Chromium catalog/token checks passed; this is not product-screen acceptance.

## Acceptance boundaries

| Criteria | Task-2.4 verification | Subsequent tasks |
| --- | --- | --- |
| AC-006/007/063 | Transfer/exchange composition, principal without income/expense, late sides/fees and lifecycle | Live adapters, reports and currency valuation |
| AC-008/093 | One payment from normalized inputs, file/source evidence, authors and replay | Receipt recognition, chat and clarification UI |
| AC-019/064 | Typed link/separate, versions, field protection and persisted review requests | Actual OpenAI execution and UX |
| AC-079/082 | Household accounts/payers and explicit internal-money links without inferred debt | Full reimbursements and UI |

SDD remains **Ready for development**. Production, bank IO, Chrome/Arc, recognition, chat/OpenAI, refunds, debt and reports are not claimed verified. Working-app acceptance and operational readiness remain separate stages.

Review regressions cover per-component carrier/date stability on note edits, date correction and undo, exclusion of a mixed included/excluded group, source conflict projection/review/outbox refresh, recovery of the former amount and coordinated updates of both sides. They also cover rejection of distinct verified payment IDs/blockchain movements and Russian reasons at the 2000-character boundary. These are HTTP and isolated PostgreSQL checks with no external IO.

Additional HTTP/PostgreSQL race regressions cover automatic-link undo after posted/cancelled/reversed, retained postings and holds, restored waiting cases and a separately confirmed transfer followed by its late counterpart. The rejected duplicate persists until explicit undo; source replay adds no effect.

Regressions also cover independent date changes after linking and their later undo, fee-before-principal and replay, a separate fee, restoring two candidates and 100 candidates with incomplete coverage. Migration checks cover pre-funding records, immutable matching decision bases and retention after command cleanup. The derived contribution field cannot become a protected user override.

Additional checks cover unchanged relatedChanges amounts during an independent date edit, late correspondence with a missing side or existing payment/transfer evidence, import replay and retained explicit separate decisions.

After task-3.1 (PR #81), version010 belongs to durable jobs. The undeployed matching migrations are renumbered011–012 without SQL changes; target files001–010 remain byte-identical. No deployed data is migrated. Prior isolated candidate databases remain retained and are not reused to validate the new sequence.

Explicit rejected pairs are checked against every member of an already linked candidate. Late evidence cannot reintroduce a rejected participant through a third record: the confirmed separate purchase retains its effect and new source evidence. Automatic linking also checks the complete expanded composition.
