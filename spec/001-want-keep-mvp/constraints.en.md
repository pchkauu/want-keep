# Architecture constraints

[Русский](constraints.md)

The user selected the stack (D-17); runtime code/manifests do not exist yet. This is target architecture, not a description of existing code.

## Behavior ownership and layout

A modular Go monolith with API/worker processes, PostgreSQL, React/TypeScript/Vite and a separate Playwright TypeScript collector. One VPS and Docker Compose; no mandatory Redis, Kafka, Kubernetes, vector database or Python service.

| Planned area | Responsibility |
| --- | --- |
| `backend/internal/<feature>/` | Money/accounts/ledger/budget/goals/forecast/credit/savings/returns/trading/mining and application contracts. Organize by owner, not global helpers/utils. |
| `backend/internal/delivery/`, `api/openapi.yaml` | HTTP, auth boundary, schema validation and transport mapping. OpenAPI is source; do not hand-edit generated bindings. |
| `backend/internal/integrations/<provider>/` | Verified provider contract, DTOs, gateway/mapper, replay and error classification. No UI imports. |
| `backend/internal/storage/`, `backend/migrations/` | Transactions, persistence mapping and migrations; no business decisions hidden in SQL. |
| `backend/internal/ai/`, `gateways/openai/` | Application commands and bounded orchestration / external OpenAI SDK. Domain does not depend on the SDK. |
| `web/src/features/<feature>/` | Local screens/forms/state; cross-feature access through public APIs and narrow query subscriptions. |
| `collector/src/providers/`, `collector/contracts/` | Isolated portal-read workflows and collection transport contract. No classification or accounting ownership. |
| `deploy/`, `ops/` | Deployment, backup and recovery; keys/data stay outside Git. |

Money is a small independent domain contract. Add other shared modules only with proven shared ownership. Coordinate modules in application services through public contracts, never through another feature's private/data/presentation implementation.

## Flow and persistence

Source/chat → retained evidence → mapping and domain validation → atomic transactions/postings → revision-specific AI review → validated command or clarification → derived reports.

Original source records are immutable; provider changes create revisions. A transaction links multiple pieces of evidence. Financial posting and outbox/job share a transaction. Source IDs are unique within household + stable external-account identity + product/log namespace. Connection/session is provenance, not permanent account identity. Exact idempotency does not replace economic cross-source matching.

Persist exact decimals and transport strings; retain source precision. Separate native ledger and historical rate snapshots from formatting. One-sided/partial transfers, unsupported assets, unknown balances and pending events stay explicitly incomplete.

## Security and trust

- Separate users with household membership; one-time operator-controlled first-member bootstrap. Passkeys validate origin/RP/challenge/purpose/expiry. Recovery codes are hashed and single-use; recovery revokes only the recovering user’s old sessions/subscriptions.
- Browser sessions use Secure HttpOnly SameSite cookies; mutating HTTP requests have CSRF protection. Actor comes from the authenticated server context; ownership is a stored resource property and membership is checked independently of client JSON.
- Encrypt provider keys and bank sessions separately from data; keep master/recovery keys outside DB/logs/Git. Authorize every attachment download/preview.
- Isolate the collector by process, profiles, network and verified action/route/payload allowlists. A GET/POST allowlist alone is insufficient: validate action semantics. Block unknown actions and hand MFA/CAPTCHA to the owner. AI does not control the browser.
- Transaction descriptions, files and model outputs are untrusted. AI receives authorized data without secrets and invokes typed application commands only; no SQL, raw HTTP, shell, payment or trading tools.
- Validate files before AI: MIME/signature, size/page limits and safe parsing. Never execute active content or arbitrary links. Financial texts/documents do not enter telemetry.
- Corrections/links carry expectedRevision, evidence and audit. Changes to approved plans/goals require confirmation of the same proposal version.

## Failure and observability

Source and AI work use independent queues. Sync has leases, bounded retries, checkpoints and last-success time. One failed source does not stop others. Recovery replays idempotent read jobs; reconcile ambiguous paid/external effects first.

AI-reviewed does not mean posted. During AI outage confirmed source/manual transactions persist and calculations work; unknown classification is separate. Unknown amount/account/currency stays draft. Stale AI results never apply.

Logs contain codes, correlation IDs, source/job/transaction IDs, durations and error categories. Metrics cover sync freshness/completeness, AI backlog, clarifications, amount/match errors, usage/reserved spend and backup age. No keys, receipts or full messages.

## Future command contract

Only the documentation Python commands in this package's README exist today. task-1.1 creates the following root Makefile interface; this does not claim existing tests.

| Command | Required behavior |
| --- | --- |
| `make check` | Check-mode formatting, lint/typecheck, Go/web/collector unit checks, docs and contract-generation checks. |
| `make test-go PKG=./internal/<feature>/...` | Run the package tests from `backend/` with repo-pinned Go. |
| `make test-web FILTER=<feature>` | Run the matching headless web tests. |
| `make test-collector FILTER=<suite>` | Run bounded synthetic collector scenarios without external accounts. |
| `make test-integration AREA=<suite|all>` | Isolated PostgreSQL/services with fault/concurrency scenarios; never production. |
| `make test-contract PROVIDER=<name>` | Check versioned synthetic provider fixtures and normalization. |
| `make check-contracts` | Check OpenAPI/schema and reproducible generated output. |
| `make e2e SCENARIO=<name|all>` | Browser flows using test data; real-device evidence is separate. |
| `make eval-ai SUITE=<name|all>` | Offline fixtures by default; live runs require a key, explicit run cap and usage report. |
| `make check-deploy` | Validate Compose/config/resources/security without provisioning. |
| `make backup-check MODE=synthetic`, `make restore-check MODE=synthetic` | Isolated integrity/recovery checks with measured duration. |

Suite names in task cards form part of the runner contract. A missing suite fails rather than returning successful empty execution.

## Readiness, migrations and rollout

Before binding generation or adapter code, task-0.10 resolves fields/sources/terms, updates these documents and publishes a Ready plan. Never fill unknown external contracts with invented endpoints.

Add schema changes through new migrations; do not rewrite applied migrations. Check compatibility against existing APIs/data; use expand → backfill → switch → contract only where needed. Rollback never discards the ledger, files or owner corrections.

CI, synthetic integration, live-source readback, physical-device push and restore rehearsal are separate evidence. Only all mandatory ACs complete the MVP. Final review uses read-only Avida with independent fact-checking; the SDD owner fixes confirmed findings.

## Household scope and permissions

`backend/internal/household/` owns User/Household/Membership and a public membership/permission policy. Budget/goals/ledger/integrations owners apply it within their use cases alongside domain invariants. Current participants have the member role; personal/household scope and personalOwnerId are not replaced by a role. Add no arbitrary-role editor or first/second-partner fields.

All financial objects, files, chat retrieval, jobs, idempotency and matching are scoped to householdId. Principal comes from a session or validated job context; a report person filter cannot change it. Check household scope, permission, expectedRevision and mutation within one transactional boundary. UI permissions do not replace backend checks. A second test household verifies isolation, not a new public-SaaS feature.

Both edit transactions and connections; only the owner edits a personal goal or personal plan portion; any member edits joint resources. Correcting an expense fact grants no right to edit the other member’s plan. Command replay and revision conflicts differ. Disconnect changes connector generation/lease and prevents previous-generation results from applying. Bank-secret entry is isolated from the shared chat and other member.

Financial source identity includes household, provider and a verified real external account/product/log. Recreating a connection never resets deduplication; equal source IDs from distinct external accounts do not merge. When identity is insufficient, retain evidence without blindly creating another financial account.

## Desktop and presentation

Design, screens and navigation are UI contracts supplementing the API: [design](design.en.md), [screens](screens.en.md), [navigation](navigation.en.md). Design system narrowly owns tokens/primitives/motion; features own user tasks. Server returns amounts, explanations and statuses; client never repeats financial formulas. States and animation events do not control the ledger. Base UI/shadcn and font versions are pinned during implementation; no runtime dependencies are added at this stage.
