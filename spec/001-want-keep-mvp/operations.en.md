# Operations, costs and recovery

[Русский](operations.md)

## Limits and hosting

D-15: server at most $40/month in Germany, the Netherlands or Bulgaria; OpenAI separately at most $50/month. Paid external data is not authorized. Initial load is hundreds of transactions/month; receipt pages, chat length and backfill are measured separately.

task-0.9 completed the dated [infrastructure research](evidence/hosting.en.md) on 2026-09-07. The owner selected a German 2 vCPU/4 GB/50 GB VPS for web/reverse proxy, Go API/worker and one sequential collector, plus 1 vCPU/2 GB/20 GB managed PostgreSQL in the same private VPC with no public DB IP. Annual pricing with one VPS IPv4 is RUB 2,520/month; conservative no-discount pricing plus 10% reserve is RUB 3,055.56/$35.29 at CBR 86.5857 RUB/USD, below $40. Re-read pricing, tax, IP and FX before purchase.

The current research VPS costs RUB 800/month per the owner. Read-only audit found Ubuntu 26.04.1 LTS, 1 vCPU, about 889 MiB RAM, no swap and a roughly 14 GiB filesystem; application, DB and backup are not deployed. It is not admitted for production. Before financial data, task-8.1 hardens SSH/firewall/monitoring/secrets, creates the private DB connection, reads back the invoice and measures peak CPU/RAM/disk/collector use. From the current VPS, public OpenAI/rate endpoints and several platforms are reachable with stated limits; safe Alfa-Bank DNS/TLS routing is not established and remains HOST-B04 for task-0.10/task-4.1.

## OpenAI

Official Standard short-context pricing snapshot, 2026-09-06, USD per 1M tokens:

| Candidate | Input | Output |
| --- | --- | --- |
| GPT-5.6 Luna | 0.20 | 1.20 |
| GPT-5.6 Terra | 2.00 | 12.00 |

These are evaluation candidates, not approved configuration. Vision, reasoning/output, retries, cache writes, long context, tier and paid tools require separate accounting. Recheck pricing before selection/execution; account model availability is unverified. [Official pricing](https://developers.openai.com/api/docs/pricing), [models](https://developers.openai.com/api/docs/models).

Scale-only example: 1,000 Luna calls with 2,000 uncached input and 500 output tokens each cost $1.00 under the table. This is text-token arithmetic, not a full-application cost promise. task-0.8 measures real calls/tokens/pages/retries, then fixes routing, size limits and cost reservations.

The application budget includes actual + reserved in-flight + unknown-outcome costs. Before a call, transactionally reserve a conservative ceiling with bounded output/context/tools; models cannot choose unlimited compute. Unknown charges do not free reservations. AI-cost periods are calendar UTC months; incomplete/uncertain charges survive month rollover and reconcile with actual billing months. Insufficient budget queues work while ordinary accounting continues. The user has not authorized automatic increases above $50.

Full transactions/receipts excluding secrets are permitted. API data is not used for training by default without opt-in; standard abuse monitoring may retain content for up to 30 days subject to documented exceptions. Responses application state has separate rules: request `store=false`, keep chat history locally and do not interpret this as Zero Data Retention. ZDR requires separate eligibility/approval and is not claimed for the pilot. Avoid unnecessary long-lived provider file stores; delete temporary provider file objects after use under their contract. [Data controls](https://developers.openai.com/api/docs/guides/your-data).

## MacBook backups

The Mac initiates outbound pull; Mac reachability is a condition, not an around-the-clock promise. Through a restricted non-root export principal, the VPS streams a logical managed PostgreSQL dump over the private network plus an immutable attachment inventory. The consistent set contains cutoff, schema/version/time and checksums. Success requires the local recipient to verify every part; no inbound Mac access is opened.

Encrypt backups; store private recovery keys separately from the sole backup and failed server. Server/master keys and access recovery codes are never published or printed. An external cloud backup is not part of the agreed MVP.

While the Mac is unreachable show last-complete-set age and the actual potential loss window. Hourly RPO depends on reachability and successful completion; interrupted downloads never update success timestamps. Resume copying after reconnection while retaining the last valid set.

MVP policy retains 48 hourly, 30 daily, 8 weekly and 12 monthly points; the deduplicated repository cap is 20 GiB, with a warning at 15 GiB or less than 25 GiB free. Never delete the last complete set. The inspected Mac had about 46 GiB free, but target-volume encryption is unverified; task-8.2 must preflight capacity/encryption/recovery-key access and measure actual size. If policy cannot fit, report an explicit failure and require more storage instead of removing the last safety point.

An isolated rehearsal restores DB/attachments/schema, checks balances, audit, identities and jobs and measures the four-hour RTO target. Do not automatically revive old bank/web sessions. Check unknown AI charges before retries. This document does not prove runtime recovery or preservation of real data.

## Monitoring and notifications

Source states: connected/reauth_required/syncing/stale/partial/failed/disconnected. AI: pending/running/reviewed/clarification/waiting_budget/failed/superseded. Backup: pending/complete/stale/failed. Never collapse these into one green indicator.

Summaries/reminders have source event ID, time/timezone and dedup key. Push subscriptions bind to device/owner and are invalidated on recovery/device revocation. Default lock-screen payloads omit amounts/merchants. In-app delivery works without push permission. Actual push permission/denial, delivery and revocation are tested in Chrome and Arc on macOS. The contract requires no application installation; unsupported delivery retains in-app and an explicit status.

## Release evidence

Full-MVP acceptance requires all ACs, every product's live readback across six platforms, measured AI quality/cost, no duplicate financial effects under retries, privacy/auth, RU/EN/desktop, push in actual Chrome and Arc on macOS, backup/restore rehearsal and actual costs. CI/mocks do not replace these checks. Deployment and resource purchases require current authorization for that separate stage.

## Household operations

The $40/$50 limits and hundreds of transactions apply to the whole household. AI accounting/reservations use the household budget, not $50 per member. Backups include both users, memberships, ownership, allocation revisions, shared chat and authorship. Sign-in recovery revokes only the relevant user’s devices; resetting the other sign-in is unauthorized.
