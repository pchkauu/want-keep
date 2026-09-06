# Contracts and financial rules

[Русский](contracts.md)

Target project contract version 3; family and desktop amendment 2026-09-07. No actual API or database schema exists yet. These are shared rules; task-0.1–task-0.10 resolve provider-specific fields/terms before implementation. REQ/AC take precedence over adapter assumptions.

## Domain entities

| Entity | Minimum contract |
| --- | --- |
| Money | Exact decimal-string `amount` and `asset` code; explicit posting sign. No NaN/Infinity/exponent float coercion. |
| Account | ID, householdId, personal/household scope, personalOwnerId for personal scope, asset, purpose, imported connection/product reference, card aliases, state and opening point. |
| SourceRecord | Household/provider/external-account/product/log namespace, connection as provenance, source ID or documented identity strategy, revision/hash, fetchedAt, occurredAt/status and raw evidence reference. |
| Transaction | ID/revision, economic type/state, native postings, cash date, expense attribution date, fees, links, evidence, actorId and human overrides. |
| BalanceSnapshot | Account, sourceAsOf, fetchedAt, owned/available/locked/debt with individual knownness and coverage. |
| Valuation | Amount pair, direction, rate, asOf, source, method/reference versus executed quote, fee coverage and revision. |
| Budget / Obligation / ExpectedIncome | Month/timezone, currency, category allocation, dated occurrences, planned amount, matched actual amount and approval revision. |
| Goal / Reservation | Amount/asset/deadline, virtual/dedicated mode, funding account and unique allocation; money cannot be reserved twice. |
| Receipt / Item | Attachment, selected account, extraction revision, total/discount/items, merchant, currency/date, validity and matching outcome. |
| AIReview / Proposal / Clarification | Subject revision, allowed action, evidence, validated payload, pending/applied/rejected/superseded state, question/answer and usage. |

## Monetary ledger

Every posted transaction atomically stores all monetary legs. Same-currency transfers preserve principal across owned accounts; exchanges retain distinct native amounts and actual rate. Do not add different assets to check balance. Fees are separate economic effects with provenance; source net P&L is not reduced again by already-included fees.

Opening balances, deposits/withdrawals to owned accounts, principal card repayments, movements into Earn/Coinhold and goal reservations are not income/consumer expenses. Interest, funding, trading results and mining rewards use verified source semantics and separate metrics. Unrealized P&L does not increase received income.

Pending affects availability through holds; actual expense arises on posting. Unknown status is not mapped to posted. Retain source evidence even when classification is unavailable.

Corrections preserve originals: a new revision and correcting/reversing effect with actor/reason/evidence, expected revision and previous-result link. Replayed commands have one effect. Human corrections take precedence over reimporting the same fact but retain evidence of source disagreement.

## Refunds and currencies

Purchases have actual cash dates and expense-budget months. Refunds have their own cash dates and original-expense links; analytics reduces the original month/category. Partial refunds are capped by the remaining unrefunded value. Unknown purchase/item attribution requires clarification; guesses do not recalculate history.

Refunding USD 4 of a USD 10 purchase originally valued at RUB 900 reduces historical expense by RUB 360. Actual receipt/conversion amounts and FX differences remain separate. Item/discount allocations exactly equal payment; allocate rounding remainders deterministically by largest fractional remainder, breaking ties by stable item ID.

Historical rate snapshots are fixed to transaction dates. Current quote updates do not alter them; correcting an erroneous historical price creates an audited valuation revision. Missing prices produce unavailable/partial, never zero, a current price substituted for history or USDT/USD=1. Native amounts stay accessible.

## Daily allowances

Calculate independently for each currency/calendar month in the budget timezone. `N` is remaining calendar days including today. Closed months have no daily allowance; never divide by zero.

- `S`: available owned money in eligible spending accounts, already adjusted for posted movements and holds. Exclude credit limits, debt, locked funds and dedicated goal accounts. Time-align source snapshots and ledger; expose incompleteness.
- `G`: virtual goal reservations within `S`. Do not deduct already-excluded dedicated-account money again.
- `O`: outstanding month obligations and planned debt repayments. Do not duplicate a payment as both minimum and repayment; do not reserve a matched hold already deducted from `S` again.
- `R = max(0, Σ plannedFlexible − Σ actualFlexible − U)`, where `U` is confirmed but uncategorized spending. One category's overspend reduces the total remaining plan; summing positive category remainders alone must not erase overspend.
- `K = max(0, min(R, S − G − O))`; overall available allowance is `K / N`. Show negative remaining amounts/shortfalls separately; a zero limit does not hide the issue.

Allocate household `K` once across positive member×category cell remainders; individual and category totals are margins of one matrix. These ceilings are not independent wallets; cross-currency equivalents are informational. Uncategorized spending remains in actual totals and `U`, not outside the budget. Unknown obligation matching may cause conservative additional reservation with an explicit explanation until clarified.

Forecast by date: `F(d) = S − G + expectedReceipts(≤d) − outstandingPayments(≤d)`. Uniform forecast allowance is `q = max(0, min(R/N, min_d F(d)/elapsedDays(d)))`. Negative `F(d)` shows a cash shortfall even with zero flexible spending. Expected receipts never increase available allowance; moving an income date recalculates forecast while approved plans change only on an authorized member decision. Fulfilling a payment changes both `S` and remaining `O` without a second deduction.

Calculations are exact; rounding displayed safe allowance downward does not change the ledger. Unused fractions remain in the monthly balance and are redistributed on recalculation. Goal reservation is not expense; a prior-month refund increases current liquidity without inventing current-month income.

## Credit cards, savings and returns

Grace, minimum/due and eligibility use structured provider fields or an owner-confirmed structured terms model. Do not turn marketing/free text into critical business automation. Missing statement cycles, exclusions, accrual basis or repayment order block exact conclusions, not viewing the account.

Savings forecasts use effective rate schedules, day-count/basis, compounding/payout schedules, term/lock, top-ups/withdrawals and explicit early-exit terms. Promised-rate income is forecast; actual accrual is a separate transaction. Contributions are not returns.

Comparison uses XIRR: `Σ CF_i / (1+r)^((date_i−date_0)/365) = 0`, with owner contributions negative and distributions/terminal value positive; `r > −1`. Native and reporting-currency results are separate, historical flows use their own dates and terminal value the comparison date. No sign change, zero period, missing valuation, incomplete history or no uniquely substantiated root returns explained unavailable, not 0%. task-0.10 resolves the solver and reference vectors; provider APR is not XIRR.

## Web API

Target prefix `/api/v1`, JSON, money strings, UTC RFC3339 timestamps plus explicit budget timezone/date. Sessions determine actor; server-validated membership determines household access and stored resource ownership determines edit rights. Lists use cursor pagination. Mutations have `Idempotency-Key`; corrections/confirmation have `expectedRevision`. Errors contain `code`, safe message, field violations, retryable and correlation ID; no raw provider payload or secrets. Unknown/partial is explicit, not hidden null/zero.

| Group | Intended interface |
| --- | --- |
| Auth | POST login options/verify; enrollment options/verify only with bootstrap, a valid invitation or fresh own auth; POST recovery/logout. |
| Accounts | GET/POST `/accounts`, GET `/accounts/{id}`; manual owned cash-account creation/opening point through validated commands. |
| Transactions | GET `/transactions`, POST manual entry, GET `/{id}`; POST `/{id}/corrections` and `/{id}/links` with revision/evidence. |
| Connections | GET/POST `/connections`, POST `/{id}/sync`, DELETE `/{id}`; separate protected credential/session enrollment, with no secret fields returned. |
| Files/chat | POST `/attachments`, family-authorized GET `/{id}`; GET/POST threads/messages, POST clarification answer. Receipt messages require accountId. |
| Plans/goals | GET/POST budgets/goals and versioned changes; POST `/proposals/{id}/apply` after an explicit authorized-member decision on the current revision. |
| Reports | GET dashboard, valuation, daily-limit, credit, savings, returns and insights with filters/date/currency/coverage. Reads do not trigger hidden mutations. |
| Notifications | GET in-app notifications, POST read acknowledgment, POST/DELETE push subscriptions. Delivery receipt does not mean read. |

task-1.2 materializes OpenAPI only after Ready; exact form fields follow the models above and verified provider contracts. Client/generated types never become domain types.

## Collector and AI

Collector read jobs contain job ID, connection reference, authorized action/product, range/cursor, deadline and short-lived authorization binding. Calls stay on an authenticated private network; credentials come from isolated secret stores/profiles, never AI payloads. Results contain source records, account references, balance snapshots, next cursor, coverage and typed status/errors. Never return/log full secrets/sessions. Unknown products/fields remain unsupported/unknown.

Allowed AI commands: classify transaction/items; propose/link a verified match; propose/create a known transaction; request clarification; skip irrelevant documents with a reason; propose budget/goal changes; explain reports. The application, not the model, decides execution eligibility. AI cannot set actor/owner or bypass revision/approval/idempotency. No payment/trade/SQL/browser/shell tools.

Receipt pipeline: uploaded → validating → processing → clarification / skipped / linked / recorded; failure and waiting-AI are separate. Retain originals. Initial technical limits: JPEG/PNG/WebP/PDF, 10 MiB/file, 10 PDF pages; excess/unsupported inputs receive explicit errors without losing the message. task-0.8 checks these limits against cost/load; changes require contract updates, not silent changes.

Important failure codes: unauthorized, version_conflict, duplicate_command, invalid_money, unsupported_asset, source_reauth_required, source_partial, valuation_unavailable, clarification_required, ai_waiting, ai_budget_exhausted, invalid_attachment, backup_stale. Each status maps to a clear UI state and AC scenario.

## Household entities, API and actions

| Entity | Contract |
| --- | --- |
| User / Household / Membership | Independent IDs; membership links user, household, member role and status. `max_active_members=2` is the MVP setting. |
| Resource scope | Mandatory householdId; personal/household plus personalOwnerId for personal accounts, plan lines and goals. Actor and externalAccountOwnerId are separate fields. |
| ExpenseAllocation | Transaction/item revision, personal/joint attribution, memberId→amount/share map; assigned shares equal the item amount. Unresolved attribution has its own state. |
| BudgetLine | One Budget per household/month; personal or joint line, allocation snapshot and approval revision. Household income retains actual recipient without restricting ownership of pooled money. |
| Reimbursement | Explicit creditor/debtor member IDs, asset/amount, optional expense link, settlements and revision. Internal claims do not enter household wealth. |
| SharedThread / Message | One household thread, message actorId, attachments, proposal/clarification revision. Shared visibility does not grant authority for every command. |

Minimum `/api/v1` additions: GET `/me` and `/household`; POST `/household/invitations`, POST `/invitations/accept`; ownership in accounts/transactions/budgets/goals; versioned allocation/reimbursement commands; `view=household|member` and memberId for reports. These filters do not change principal. Unauthorized/forbidden/scope mismatch, invitation_expired/used, member_limit_reached and version_conflict are distinct safe errors. Forms materialize in OpenAPI after Ready.

Engineering defaults: the first user uses restricted single-use bootstrap; the second accepts a signed-in member’s single-use random invitation with a 24-hour expiry, stored hashed. It binds to a separate new sign-in; replay and the limit are checked atomically. It cannot reset another user’s passkeys. The system does not send external invitation messages itself. Editing a personal goal or plan line, including deletion, changing owner/personal scope or applying an AI proposal, requires its current owner. Converting another user’s goal to joint cannot bypass this. Members create personal goals/lines for themselves and joint ones freely. Both can read/create/correct all accounting transactions. Personal account ownership changes require its owner, household account changes either member; these cannot change the verified external owner or transaction history.

Both manage connections: create/sync/disconnect/reauth-request. Secret/MFA submission accepts only a verified externalAccountOwnerId session, outside chat. Management initiator and external-account owner are retained. Background AI review uses a restricted system principal for its household; it cannot approve plans/goals. A proposal executes as its approving user with renewed permission/revision checks. Either member can resolve a transaction-fact clarification; personal plan/goal clarifications require their owner.

Mixed receipts allocate by item. Precedence: explicit item allocation, then purchase, then plan line, then equal shares for the two current members when joint attribution is established. Unknown attribution does not imply joint. Validate amounts/shares; round by largest remainder with memberId tie-break, preserving total. New rules never rewrite historical allocation snapshots. Refunds use the current audited allocation of the original purchase/items and its historical valuation; a later explicit purchase correction consistently recalculates linked refunds in the same calculation revision, while new monthly defaults never affect history.

Explicit debt supports partial settlement through linked real transfers/cash. Cross-currency settlement retains explicitly confirmed sent and settled amounts; a current quote never settles debt by itself. Settlement beyond the remaining debt is rejected; any extra actually transferred money remains a separate movement. Reversing the original expense does not silently delete debt; it flags the need for a confirmed correction.

## Individual allowances and reservations

`S`, `G`, `O`, `R`, `K` and `F(d)` above apply to the whole household in one currency. `S` includes eligible money from both members’ personal and household accounts; ownership requires no actual transfer for household funding. Joint goals appear in a separate block without personal halves and reduce household availability once. Goal contributions are not income/expense.

For each `(member, category)` cell define `r_mc = plannedFlexible_mc − attributedActualFlexible_mc`, `w_mc = max(0,r_mc)`. `U` contains only confirmed spending not yet included in cells and reduces overall `R` once. Unknown attribution or category remains an explicit household-actual unallocated bucket; individual actuals plus this bucket equal household actuals. Allocate joint spending before summation instead of repeating it whole for each member.

For `W=Σw_mc > 0`: `k_mc = K×w_mc/W`, individual daily allowance `Σ_c k_mc/N`, category allowance `Σ_m k_mc/N`. If `W=0`, individual/category allowances are zero. Distribute forecast `q` by the same weight matrix. Round displayed allowances downward; show any unallocated fraction separately without increasing K. Example K=1,000, N=10, A weight=3,000 and B=1,000 yields 75 and 25 per day; household allowance is 100.

In `F(d)`, outstandingPayments is the same unpaid obligation portion not covered by already-deducted holds as O, scheduled by date. All expectedReceipts are pooled with dates and sources retained. Goal reservation and household free-funds validation are atomic; a financial spending fact is never rejected for exceeding a reserve, instead record the shortfall without silently editing the goal. Publishing the joint plan never approves another member’s personal drafts: owners include their personal lines, either member includes joint lines and the other is notified.

## Presentation, command and event contracts

SCR-001–SCR-035 UI routes are not API endpoints. The [screen catalog](screens.en.md) defines FORM-01–FORM-15 fields and error flows; task-1.2 refines OpenAPI after Ready. Reports return native amounts, separately known reporting amounts, asOf/coverage, actual/forecast/reserved type, calculation inputs and explanatory transaction links. Client formats and expands these data without repeating financial formulas.

For a mutating command, server binds Idempotency-Key to householdId, actorId, type and payload hash; same key with a different payload is rejected. Result and financial effect are atomic. Target GET `/api/v1/commands/{id}` and `/api/v1/commands/recent` return only the current member’s own authorized commands with `pending|succeeded|failed`, outcome reference and safe error. `unknown` describes client knowledge, not permission to create a new command. After timeout/reload client checks command status; `not_found` cannot prove no effect without the server registration contract. Keep input in tab memory, never secrets in URLs/localStorage. Command-record/idempotency retention must cover retry/recovery windows and is resolved in task-0.10/task-1.2.

UIState derives from typed errors/coverage/result. `version_conflict` includes an authorized current revision for comparison; server rechecks permission on reapply. Session expiry hides protected screens; another principal never receives a draft. Personal preferences include locale, reporting currency, notification options and decorativeEffectsEnabled; preferences change neither household fact nor authority.

Motion subscribes to confirmed domain events and never owns a posting. Event stores ID, householdId, subject/revision, kind, occurredAt, origin (`interactive|live_sync|historical_backfill`), eligibility and correction consequences. Accounts/goals/budget determine business event/eligibility through application/outbox; notifications owns delivery/personal presentation ack. Initial historical import has eligibility false; top-up/goal achievement never derive from balance changes on read. Server atomic claim/ack by event+user prevents competing-tab replays; lost presentation confirmation prefers static result over repeat celebration. Effect ack differs from notification read. Reduced motion/off produces static outcome; unavailable animation support never affects accounting.

Synthetic overview response example: “Available today RUB 400; USD 100 expected in 5 days; RUB 3,000 reserved for a goal.” These fields are never summed without explicit valuation; clicking RUB 400 expands K/N and its inputs. Example figures are neither prices nor personal data.
