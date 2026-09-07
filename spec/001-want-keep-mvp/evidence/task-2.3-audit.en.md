# Task-2.3 — corrections, undo and audit

Implemented correction, exclusion, selective undo and history backend/API. Base: `ee48b278510c93386cb1fd26387a71d22bd77bda`; branch: `feat/task-2.3-corrections-audit`; task-2.2 is included. The [contract](../contracts.en.md#task-23--corrections-selective-undo-and-audit) preserves financial rules, independent participant roles and transaction boundaries. No new dependencies; migrations 001–008 and production remain unchanged.

Ledger owns decisions, field protection and undo; accounts recomputes projections; storage persists immutable migration-009 data; delivery uses generated DTOs. User fields merge with normalized source facts; whole-record HumanOverride remains only as conservative legacy protection. Exclusion cannot assign bank lifecycle. Compound decision/undo and CompleteReview are internal application contracts. API exposes source values for comparison, before/after and safe review rationales.

## Verification matrix

Required commands: `make check`; `make test-integration AREA=audit` and `make test-audit-race`; ledger/accounts/storage/identity/household integration/race; `make test-integration AREA=privacy`; `git diff --check`. PostgreSQL 17.11 is digest-pinned, the temporary DB is isolated, and financial scenarios use the unprivileged role. Missing DB fails the audit suite. Audit integration/race are in CI. Exact local and CI results belong to the published candidate and are recorded in the PR/report.

The audit suite checks six assets through HTTP/SQL, 5000 − 500 → 5000 − 700 and undo; transfer/exchange principal and fees including a third asset; moving a purchase across months and opening date while retaining postedAt. It checks a later independent partner edit, ABA conflict, no-op, exclusion/restoration, pending → posted with a protected merchant, latest source amount when removing an override, and reversed without resurrecting effects.

It checks atomic compound decisions and undo, rollback before commit, restart/replay after a lost acknowledgement, competing revisions with a barrier, changed payload/target, household isolation, field spoofing, Origin/CSRF, exact pagination and history. Review is tested without OpenAI: one request per revision, result replay, stale responses, rejection of protected/monetary proposals, evidence, rollback and no loop from recording a review. Migration over 008 preserves legacy_all and unknown time; roles and triggers prevent history mutation. D-41 cleanup removes command data while retaining financial decisions and audit.

Regression coverage includes retained pending/posted conflict resolution on undo, independent later review suggestions, identical omitted-default fees, 101 source versions and a 51-operation decision with 102 evidence references. Contract tests cover live/expired failure outcomes and closed reimbursement/link schemas.

When undo reapplies source data, independent effective review decisions are compared against their immutable decision evidence, regardless of whether they preceded or followed the selected decision. An unchanged source field preserves the decision; a confirmed new value updates an unprotected field. Source links later appended to a financial revision are not used as that baseline.

## Acceptance boundaries

| Criteria | Task-2.3 evidence | Subsequent verification |
| --- | --- | --- |
| AC-012 | Selective/compound undo and preserved edits during import | Matching task-2.4, categories and reports |
| AC-018/019/064 | One review per revision, stale/protected/monetary proposals and safe rationale | Chat/receipts, matching clarifications and real OpenAI |
| AC-021 | New routes do not change the approved budget; the budget scenario itself is not implemented here | Budget proposal confirmation at its owner |
| AC-061 | Exact revisions, atomicity, restart/replay and competing correction | Real workers and complete lifecycle |
| AC-078/086 | Both members correct facts, rights/actor, concurrency and independent later fields | Complete personal/shared budgets and clarification UI |
| AC-093 | Authored evidence and correction protection on reimport; automatic receipt merging is not implemented | Receipt/chat/import matching, FORM-06/SCR-010 in Chrome/Arc |

SDD remains **Ready for development**. Live adapters, matching, categories, shares, refunds, budgets/reports, OpenAI, browser screens and production are not verified here. Historical import without normalized versions receives no invented source values; legacy requires explicit subsequent investigation.
