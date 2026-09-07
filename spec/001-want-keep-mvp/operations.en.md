# Operations, costs and recovery

[Русский](operations.md)

SDD Ready for development is not operational approval. Before production, task-8.x proves the infrastructure and each task-4.x passes its own provider deployment gate.

## Limits and hosting

D-15: server at most $40/month in Germany, the Netherlands or Bulgaria; OpenAI separately at most $50/month. Paid external data is not authorized. Initial load is hundreds of transactions/month; receipt pages, chat length and backfill are measured separately.

task-0.9 completed the dated [infrastructure research](evidence/hosting.en.md) on 2026-09-07. The owner selected a German 2 vCPU/4 GB/50 GB VPS for web/reverse proxy, Go API/worker and one sequential collector, plus 1 vCPU/2 GB/20 GB managed PostgreSQL in the same private VPC with no public DB IP. Annual pricing with one VPS IPv4 is RUB 2,520/month; conservative no-discount pricing plus 10% reserve is RUB 3,055.56/$35.29 at CBR 86.5857 RUB/USD, below $40. Re-read pricing, tax, IP and FX before purchase.

The current research VPS costs RUB 800/month per the owner. Read-only audit found Ubuntu 26.04.1 LTS, 1 vCPU, about 889 MiB RAM, no swap and a roughly 14 GiB filesystem; application, DB and backup are not deployed. It is not admitted for production. Before financial data, task-8.1 hardens SSH/firewall/monitoring/secrets, creates the private DB connection, reads back the invoice and measures peak CPU/RAM/disk/collector use. From the current VPS, public OpenAI/rate endpoints and several platforms are reachable with stated limits; task-4.1/task-8.1 verify safe Alfa-Bank DNS/TLS routing as a runtime gate.

Provider deployment is disabled by default. D-43 admission requires task-4.x provider evidence and task-8.x host/deployment evidence for one environment/build/contract/allowlist/configuration/permission binding. The `backend/internal/connections/admission/` application service uses the task-1.3 storage adapter to change server-owned state and `admissionRevision` atomically. The admission check and enqueue share a transaction; a stale/missing binding returns `provider_not_admitted` before a new job or collector IO. Jobs/results carry the exact revision, checked by the collector before IO and by storage while committing source/posting/outbox. Revocation/change invalidates unstarted work; cancellation of a started read is best effort and its stale result remains in quarantine without a financial effect. Pre-admission conformance also runs in quarantine. A failed gate leaves that source disabled while ordinary/manual accounting continues. Health reports unknown/partial/ambiguous separately.

## OpenAI

Model choice and limits are owned by [task-0.8 research](evidence/openai.en.md); the [estimate](evidence/openai.cost.json) is distinct from actual usage and bank charges. Pricing date: 2026-09-07. Select gpt-5.6-terra xhigh for all AI tasks; final evaluation is 206/206 without unnecessary clarifications, 6/6 PNG/PDF and 3/3 function calling. The SDD contract is closed; application runtime work remains.

Use foreground Responses, `store=false`, local chat history, `prompt_cache_options.mode=explicit` without breakpoints and `detail=high` for pages. Models only propose validated application commands; credentials, SQL, browser, shell, payments and hosted tools are unavailable. Evidence discloses OpenAI retention and unverified ZDR/EU residency; a European VPS does not establish European OpenAI processing.

The family budget is USD 50 per UTC month: actual + reserved + unknown. Atomically reserve the maximum input/output and possible cache writes before every call; reasoning is already part of output. Reconcile confirmed usage against reservations; unknown outcomes and missing write counts never silently release funds. Month/model/key changes do not erase liabilities. At most two concurrent calls, SDK retries disabled; no automatic budget increases. Provider hard limits supplement this control but can lag.

API failure, unavailable models, exhausted funds or unverified prices leave AI waiting while ordinary accounting continues. Every transaction version remains queued. Evidence owns page/context/retry limits and quality measurement; production gateway, receipt-pipeline and family-permission checks remain task-5.1–task-5.5. Before launch, reconcile project funds and applicable taxes/fees with task-0.9's total estimate.


Planned xhigh profile: USD 47.125/month, stress USD 62.96875 under the shared USD 50 cap; unaffordable jobs wait without downgrade. These are planning assumptions, not measured monthly billing.

## MacBook backups

The Mac initiates outbound pull; Mac reachability is a condition, not an around-the-clock promise. Through a restricted non-root export principal, the VPS streams a logical managed PostgreSQL dump over the private network plus an immutable attachment inventory. The consistent set contains cutoff, schema/version/time and checksums. Success requires the local recipient to verify every part; no inbound Mac access is opened.

Encrypt backups; store private recovery keys separately from the sole backup and failed server. Server/master keys and access recovery codes are never published or printed. An external cloud backup is not part of the agreed MVP.

While the Mac is unreachable show last-complete-set age and the actual potential loss window. Hourly RPO depends on reachability and successful completion; interrupted downloads never update success timestamps. Resume copying after reconnection while retaining the last valid set.

MVP policy retains 48 hourly, 30 daily, 8 weekly and 12 monthly points; the deduplicated repository cap is 20 GiB, with a warning at 15 GiB or less than 25 GiB free. Never delete the last complete set. The inspected Mac had about 46 GiB free, but target-volume encryption is unverified; task-8.2 must preflight capacity/encryption/recovery-key access and measure actual size. If policy cannot fit, report an explicit failure and require more storage instead of removing the last safety point.

An isolated rehearsal restores DB/attachments/schema, checks balances, audit, identities and jobs and measures the four-hour RTO target. Do not automatically revive old bank/web sessions. Check unknown AI charges before retries. This document does not prove runtime recovery or preservation of real data.

## Monitoring and notifications

Source states: connected/reauth_required/syncing/stale/partial/failed/disconnected. Provider admission: pending/admitted/blocked with binding/reasons. AI: pending/running/reviewed/clarification/waiting_budget/failed/superseded. Backup: pending/complete/stale/failed. Never collapse these into one green indicator.

Summaries/reminders have source event ID, time/timezone and dedup key. Push subscriptions bind to device/owner and are invalidated on recovery/device revocation. Default lock-screen payloads omit amounts/merchants. In-app delivery works without push permission. Actual push permission/denial, delivery and revocation are tested in Chrome and Arc on macOS. The contract requires no application installation; unsupported delivery retains in-app and an explicit status.

## Release evidence

Full-MVP acceptance requires all ACs, every product's live readback across six platforms after provider gates, measured AI quality/cost, no duplicate financial effects under retries, privacy/auth, RU/EN/desktop, push in actual Chrome and Arc on macOS, backup/restore rehearsal and actual costs. CI/mocks do not replace these checks. SDD Ready permits development but skips none of these runtime gates. Deployment and resource purchases require current authorization for that separate stage.

## Household operations

The $40/$50 limits and hundreds of transactions apply to the whole household. AI accounting/reservations use the household budget, not $50 per member. Backups include both users, memberships, ownership, allocation revisions, shared chat and authorship. Sign-in recovery revokes only the relevant user’s devices; resetting the other sign-in is unauthorized.
