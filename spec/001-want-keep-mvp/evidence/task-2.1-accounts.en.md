# Task-2.1 — accounts and opening balances

Implemented cash creation, versioned opening corrections, ownership changes, admitted product/alias/snapshot import, exact projections and the accounts/command recovery HTTP API. Base: `ce25bf1515f8564e6493676fca060451d324986a`; branch: `feat/task-2.1-accounts-opening-balances`. Dependencies task-1.3/1.4/1.5/1.6 are included in the base. The [contract](../contracts.en.md#task-21--accounts-and-opening-balances) records the financial and authorization boundaries. No new dependencies, production deployment or changes to applied migrations.

Migration 007 retains exact NUMERIC, nanosecond instants, independent amount knownness/coverage/freshness, composite household FKs and append-only opening/source/audit rows under the application role. Raw snapshots are separate from the journal. Creation and corrections use separate pending registration followed by atomic ledger/projection/audit/outbox/result, rechecking session and membership. Source import runs in CommitPage after admission, connection generation and lease checks; ambiguity/unsupported symbols retain evidence in quarantine without a financial effect. Source and cash use one funding policy shared with reserves.

## Verification matrix

Required local commands: `make check`, `make test-integration AREA=accounts`, `make test-accounts-race`, storage/identity/household integration and race targets, `make test-integration AREA=privacy`, `git diff --check`. The isolated PostgreSQL 17.11 image is digest-pinned. Missing DB fails accounts integration. CI preserves all base gates and adds accounts integration/race. Exact local and CI outcomes are recorded against the published candidate in the PR/delivery evidence; real product/browser/provider tests are separate.

New accounts tests exercise seven accounts/all six assets through HTTP → Go → PostgreSQL → JSON, fractional USDC/BTC/ETH, opening 5000 minus expense 500, corrections and date movement preserving history, current/reversed effects and timezone nanosecond boundaries. They cover same-key replay/changed payload, concurrent revisions, rollback, lost-response recovery via the existing key, actor-only recent/expired outcome reads, session/CSRF, two families in one DB and partner opening corrections. Import tests exercise reconnect, distinct products/external owners, aliases, debt/credit separation, delayed/repeated/conflicting observations, partial history, unsupported symbols, quarantine, stale admission and source/journal separation. Domain tests check unknown/stale/partial funding and native totals without parity. Session fixtures use stored synthetic credentials; actual WebAuthn verification remains covered by the unchanged identity suite.

## Acceptance boundaries

| Criteria | Evidence here | Remaining work |
| --- | --- | --- |
| AC-002 | Seven-account set, six exact assets, cards not double counted | Account screens and live providers |
| AC-004/005 | Versioned opening, explicit source/ledger quality, late/conflicting snapshots, debt separated | Full reconciliation, FX valuation, product-specific debt/hold lifecycle |
| AC-039 | Partial history and unsupported raw code preserve evidence/unknown | Provider history acquisition and complete reconciliation |
| AC-079/080 | Distinct external accounts/products, partner corrections, aliases and opening provenance | Full financial allocation/budget flows |
| AC-090/105 | Active household scope, trusted actor, current-owner change policy, foreign-resource non-disclosure | UI, future AI commands and all downstream resource consumers |

Task-2.2 extends financial operation semantics, including specialized debt/hold changes. Task-3.2 proves coverage and aligns source/journal instead of adding balances blindly. Import adapters provide stable identity/alias/observation IDs, proven own-availability mapping, page coverage and immutable job binding; they call this contract inside CommitPage. Notifications/motion consume committed events and own per-user acknowledgement; backfill is not celebration-eligible. Task-7.1 implements FORM-03 and SCR-007/008. SDD remains **Ready for development**; operational readiness, live bank IO, browser screens, full budgets and FX are not claimed.
