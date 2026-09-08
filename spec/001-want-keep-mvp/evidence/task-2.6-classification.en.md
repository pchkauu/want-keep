# Task-2.6 — categories, merchants and receipt items

Implemented the backend contract for independent expense classification. Original base: `c37d9ed255b01c155d92f3163736bcf57193a25b`; target `553ab977522a7e080d88c1e8f49bb758b8743fdc` was integrated before final review; branch: `feat/task-2.6-categories-merchants-items`; task-2.3, task-3.1 and task-2.5 are included. The [contract](../contracts.en.md#task-26--categories-merchants-and-receipt-items) preserves exact financial amounts, decision history and household permissions. No new dependencies; migrations 001–009 and production remain unchanged.

The catalog belongs to `categories/domain/application`; ledger owns revision classification and selective undo; storage persists migration 012 after durable jobs migration 010 and reconciliation migration 011; delivery uses generated DTOs. Starter RU/EN categories have stable keys. User names are never translated. Merchants and confirmed aliases are separate from categories. Review can retain a category, merchant, alias or item-set proposal but cannot apply it or mutate the catalog.

## Verification matrix

Required commands: `make check`; `make test-integration AREA=categories` and `make test-categories-race`; audit/ledger/accounts/storage/identity/household integration/race; `make test-integration AREA=privacy`; `git diff --check`. PostgreSQL 17.11 is digest-pinned; missing DB fails the categories suite. Categories integration/race are in CI. Exact local and CI results belong to the published candidate and are recorded in the PR/Issue.

The categories suite checks the starter catalog for existing and new households, RU/EN and customName; two levels, cycles, names, archive/restore and revisions. Merchant coverage includes confirmed aliases, collisions, matching release after archival and collision revalidation on restore. Every action uses the unprivileged role and hides foreign resources consistently.

The ledger scenario checks a RUB 900 expense and receipt `600 + 400 − 100 = 900`, stable 60/40 discount allocation, no top-level category beside items and no duplicate expense. The contract uses the same exact decimal types for RUB, USD, USDT, USDC, BTC and ETH. Mismatched totals, mixed assets and incomplete discounts create no revision. Correction and selective undo atomically change category, merchant and the complete item set; import retains protected classification.

API checks cover cookie/Origin/CSRF, command replay and outcome recovery, keyset pagination, category/merchant/item filters, history and generated Go/TypeScript. A review scenario stores a proposal separately from the effective transaction and creates neither catalog data nor a new financial revision. Migration-over-009 seeds existing households; immutable triggers and grants protect history.

## Acceptance boundaries

| Criterion | Task-2.6 evidence | Subsequent verification |
| --- | --- | --- |
| AC-014 | Separate category/merchant/item, filters, confirmed aliases and protected corrections | SCR-009/010/034, live imports and reports |
| AC-016 | Exact item/discount total, deterministic remainder and clarification without invented items | OCR/PDF, receipt matching and member allocation |
| AC-054 | Stable RU/EN starter labels and retained customName/ID/money semantics | Complete UI and browser language switching |

SDD remains **Ready for development**. This task does not verify UI, OCR/PDF, real OpenAI, budgets, refunds, household shares, reports or production.
