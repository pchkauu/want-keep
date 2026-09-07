# Contracts and financial rules

[Русский](contracts.md)

Project contract version 9, D-37 dated 2026-09-07. Task-1.2 implements OpenAPI and basic domain types; HTTP handlers and a database schema do not exist yet. These are shared rules; task-0.1–task-0.10 resolve provider-specific fields/terms before implementation. REQ/AC take precedence over adapter assumptions.

Task-1.2 review clarification: payer is explicit known/memberId, unknown or not_applicable, entered/corrected independently from actor and shares. Existing movement links require each ID/expectedRevision and atomic validation. Plan preview distinguishes create/update/delete and lineId; expectedRevision identifies the Budget aggregate, advanced by every line change/approval. ReturnsReport carries decimal-string dimensionless XIRR ratios, native/reporting basis, dated cash flows and unavailable reasons; task-0.10/task-6.4 still own the solver. These changes affect unreleased DTOs; both clients regenerate together and no deployed data requires migration.

## Domain entities

### Executable foundation D-37

Decimal strings are limited to 256 characters, reject exponent/float input and retain fractional precision for all six assets. RUB does not imply two stored decimal places, nor BTC eight. Rounding is a separate action with scale and floor/half-even; allocation preserves the exact total and distributes remainders by stable ID. Totals not representable in the chosen quantum and overflow are rejected. Rate is a positive finite decimal quote/base observation; valuation owns cross calculations and retains source legs.

A known amount contains value; unknown/unavailable contains reason without value. Complete coverage has an empty reason list, partial/unavailable a non-empty list. Fresh/stale/unknown freshness is independent of completeness. UTC timestamps accept RFC3339 with Z and up to nine fractional-second digits; Date/Month contain no time, timezone is UTC or a validated IANA zone. Revision is an integer from 1 through 9007199254740991, exactly representable in JavaScript.

OpenAPI 3.0.3 and Go/TypeScript models/strict interfaces derive from one source. Schema validation and explicit boundary converters do not replace permissions, transactions or use-case invariants. DTOs distinguish actorId/User, payer.memberId/Membership, personalOwnerId/User and externalAccountOwnerId/User. Command input cannot assign actor/household. Lists and result references require current household scope and resource authorization.

Command ID is a client-created UUIDv4 Idempotency-Key, unique within household+actor and bound to immutable type/hash. Pending is persisted before execution; effect and succeeded/result commit atomically. Replay precedes old expectedRevision checks and returns the original outcome. Timeout never changes a command to failed; not_found permits only the original key under the registration protocol. Status/key/hash/result metadata lives for the family lifetime. Auth/recovery/enrollment, private upload bytes and push credentials use separate protected flows and are not copied into financial-command records; preview stores no financial change.

This is a new foundation with no running product API or database, so data migration is unnecessary. Task-1.3/task-1.4 own durable storage and server authorization. [Checks and limitations](evidence/task-1.2-domain-api.en.md).

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

Historical rate snapshots are fixed to transaction dates. Current quote updates do not alter them; correcting an erroneous historical price creates an audited valuation revision. Missing prices produce unavailable/partial, never zero, a current price substituted for history or USD/USDT/USDC=1. Native amounts stay accessible.

## Reference-rate contract

[task-0.7 evidence](evidence/fx.en.md) selects Bank of Russia as primary USD/RUB, Frankfurter v2 only with `providers=CBR` as fallback/cross-check, and CoinGecko Demo for separate BTC/USD, ETH/USD, USDT/USD and USDC/USD observations at most 365 days old. Default blends are forbidden. TradingView is not a data source. Crypto history older than 365 days remains unavailable pending task-0.10.

For `P_USD(X,D)`, USD per one asset unit, calculate `R(S→T,D) = P_USD(S,D) / P_USD(T,D)`. `P_USD(USD,D)=1`; `P_USD(RUB,D)=1/CBR_USD_RUB(D)`. Each leg retains provider asset ID, requested date, observed/effective time, fetchedAt, granularity, source/transport and revision. Calculation and inversion use Decimal; rounding happens only at an explicit presentation boundary.

Historical CBR uses the latest effective date `≤ D`; a weekend creates no new observation. CoinGecko history is a daily UTC snapshot for the operation date in the budget timezone. If a leg is missing, outside available depth or a current response lacks the required timestamp, the cross and dependent conclusions are partial/unavailable. The latest cache may be displayed as stale with dates but cannot rewrite history.

Reference valuation does not replace an actual exchange. An executed quote needs both native amounts, direction, applicable amount, provider timestamp and known spread/fees from the source operation. Missing fields display as unknown/quote unavailable. A primary/cross-check mismatch retains both observations and creates diagnostics without hidden averaging.

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

Under D-37 task-1.2 materializes shared OpenAPI independently of remaining research. Provider-specific forms and handlers stay with their owning tasks. Client/generated types never become domain types.

### Raiffeisen authorization callback

RAIF-E15 registered `https://want-keep.tech/api/v1/connections/raiffeisen/callback`: future `GET /connections/raiffeisen/callback` under the shared API prefix. This extends the protected Connections enrollment flow; it is not an existing application endpoint. Compatibility: external registration fixes the path; changes require preparing the new handler and updating bank registration first. RAIF-E16 verifies DNS/HTTPS; the server OAuth handler is not implemented and the placeholder returns 503. Do not start Code Flow until the handler is verified. Initial RBO Refresh-token issuance for research does not replace this contract. See [evidence](evidence/raiffeisen.en.md).

The callback accepts `state` and `code`, or safely handles provider denial. The browser OAuth redirect is an exception to the general JSON/Idempotency-Key rules: it is a GET protected by single-use state, not a financial-ledger command. Protected authorization initiation creates a time-limited attempt with state, nonce, PKCE S256/verifier and bindings to household, connection, its version, current principal and externalAccountOwnerId. Lifetime is configurable; the callback requires the same authenticated owner session. URL parameters cannot assign a user, household or owner; disconnection, attempt expiry and version changes prevent result application.

At callback, the server checks state and bindings, then atomically claims a valid attempt once and exchanges code using the exact registered redirect_uri and verifier. Client secret/verifier/tokens stay server-side; ID token validation includes signature, issuer, audience, lifetimes and nonce under the verified OIDC contract. An unknown exchange outcome does not automatically retry the single-use code; the attempt gets an explicit status, and a subsequent login creates a new attempt without duplicating the connection. Code, state and tokens are excluded from request/error logs, traces and analytics; the callback loads no third-party resources and returns `Cache-Control: no-store`, `Referrer-Policy: no-referrer`. A 303 leads to a safe connection screen without secrets in the URL. Persisted results let repeated redirects display status without another exchange.

task-4.2 checks success, bank denial, missing/foreign/expired state, repeated callbacks, another user's or an expired session, connection disconnection/version changes, invalid nonce/ID token, unknown exchange outcome and secret-free logs. This refines REQ-048/REQ-073 and AC-048/AC-087; implementation and runtime checks remain future work.

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

Minimum `/api/v1` additions: GET `/me` and `/household`; POST `/household/invitations`, POST `/invitations/accept`; ownership in accounts/transactions/budgets/goals; versioned allocation/reimbursement commands; `view=household|member` and memberId for reports. These filters do not change principal. Unauthorized/forbidden/scope mismatch, invitation_expired/used, member_limit_reached and version_conflict are distinct safe errors. Shared forms are materialized in task-1.2 OpenAPI under D-37; runtime permissions are implemented separately.

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

SCR-001–SCR-035 UI routes are not API endpoints. The [screen catalog](screens.en.md) defines FORM-01–FORM-15 fields and error flows; task-1.2 materializes shared OpenAPI under D-37. Reports return native amounts, separately known reporting amounts, asOf/coverage, actual/forecast/reserved type, calculation inputs and explanatory transaction links. Client formats and expands these data without repeating financial formulas.

For a mutating command, server binds Idempotency-Key to householdId, actorId, type and payload hash; same key with a different payload is rejected. Result and financial effect are atomic. Target GET `/api/v1/commands/{id}` and `/api/v1/commands/recent` return only the current member’s own authorized commands with `pending|succeeded|failed`, outcome reference and safe error. `unknown` describes client knowledge, not permission to create a new command. After timeout/reload client checks command status; `not_found` cannot prove no effect without the server registration contract. Keep input in tab memory, never secrets in URLs/localStorage. Under D-37 compact command records and idempotency keys live as long as the family, without copies of source documents/messages or secrets. Task-1.3 implements durable storage.

UIState derives from typed errors/coverage/result. `version_conflict` includes an authorized current revision for comparison; server rechecks permission on reapply. Session expiry hides protected screens; another principal never receives a draft. Personal preferences include locale, reporting currency, notification options and decorativeEffectsEnabled; preferences change neither household fact nor authority.

Motion subscribes to confirmed domain events and never owns a posting. Event stores ID, householdId, subject/revision, kind, occurredAt, origin (`interactive|live_sync|historical_backfill`), eligibility and correction consequences. Accounts/goals/budget determine business event/eligibility through application/outbox; notifications owns delivery/personal presentation ack. Initial historical import has eligibility false; top-up/goal achievement never derive from balance changes on read. Server atomic claim/ack by event+user prevents competing-tab replays; lost presentation confirmation prefers static result over repeat celebration. Effect ack differs from notification read. Reduced motion/off produces static outcome; unavailable animation support never affects accounting.

Synthetic overview response example: “Available today RUB 400; USD 100 expected in 5 days; RUB 3,000 reserved for a goal.” These fields are never summed without explicit valuation; clicking RUB 400 expands K/N and its inputs. Example figures are neither prices nor personal data.

## Aifory and ETH: D-33

RUB, USD, USDT, USDC (D-36), BTC and ETH are available in Money and valuation. Network is a separate source/operation attribute; verify provider scale at the boundary. Do not round ETH to fiat cents or equate USD/USDT/USDC. Unknown rate/available/locked does not become zero or spendable money.

Aifory reads only RUB accounts, USDT, ETH and the existing USD card with their movements/fees. A RUB-group total does not create another balance; matching office names do not merge accounts. A platform RUB wallet preserves product kind and is not a bank deposit. Card funding links distinct native legs under an established contract, with fees separate. Authorization/clearing need IDs/linkage; UI sign, mask, shared URL and similar merchant are not identity.

Other products are deferred without blocking. Retain their movements through included wallets with provenance and clarification of unknown semantics. task-0.10 resolves structured provider mapping, automation permission, history/reauth and card lifecycle under AIFORY-B02–B04 before task-4.5; no OCR-accounting bypass. [Evidence](evidence/aifory.en.md).

## EMCD: D-34

The current contract covers the USDT wallet, existing Coinhold/Grow, used Plus/Light cards and historical P2P orders. Mining has never been used; its history and other unused products are unnecessary. Crypto cards do not acquire credit characteristics without an established agreement. Grow identity persists across Coinhold/Grow names; one product does not create two accounts.

The main aggregate and child wallet/Grow balances are not added twice. Link accrued, capitalized and paid rewards; capitalization/movement of already recognized income creates no additional income. Unknown balance composition or card reserves cannot become spendable funds. Purchase decline and a posted fee are separate effects; legacy Light and Plus use their own terms. Card funding links USDT and USD; original EUR purchase amounts remain separate from approximate USD valuation and settlement. Exact owner-side fields link P2P to wallet/bank facts, not currency order, conversations or rounded UI amounts.

[Evidence and gaps](evidence/emcd.en.md), [synthetic scenarios](evidence/emcd.samples.json). Real provider request/response pairs were not obtained; scenarios are not an API schema. task-0.10 closes BLK-06 before task-4.6 implementation. This updates target-contract version 5; financial runtime/database are absent and no migration is needed; the task-1.1 foundation is already implemented.

## Bybit: D-36

Funding USDT/USDC/ETH/BTC, used Easy Earn and P2P remain required; unused products do not block. [Route matrix](evidence/bybit.en.md), [authenticated evidence](evidence/bybit-api.en.md) and [synthetic projections](evidence/bybit.samples.json) define task-0.10 inputs. RSA readOnly with Wallet/AccountTransfer, Exchange/ExchangeHistory, Earn/Earn and FiatP2P/FiatP2POrder succeeded for this owner. Read POST `/v5/p2p/order/simplifyList` and `/v5/p2p/order/info` are explicitly allowlisted; financial POSTs remain forbidden. P2P API is confirmed, so no Playwright collector is required for observed coverage. `apiKey` echoed by query-api is filtered before logs/AI/evidence; credentials stay in protected infrastructure.

Identity separates household/provider/site/UID, FUND accountType/asset and each source-log ID. Rotate credentials without creating an account; isolate different owners. Funding `currcCursor` and detail IDs do not automatically identify different economic events. Observed amount/time matches for 49 Convert pairs, 45 withdrawal gross debits, 140 nonzero yield credits and two P2P debits are candidates, not proven cross-log foreign keys. Define deterministic matching and ambiguity handling before automated posting; localized business labels cannot drive critical classification. Fees, exchange legs and accrued/distributed/Funding yield affect money once; zero yield needs no credit, principal is not income.

Preserve exact decimals and distinguish USD/USDT/USDC. Balance snapshots can hide ledger residuals; neither single-coin nor all-coin reads solved the observed difference. Do not discard precision or create a balancing expense. P2P native fiat amount and crypto quantity are independent authoritative legs; quoted price does not necessarily reconstruct fiat cents. Empty maker/taker fee is unknown, not zero. Bank settlement is a separate link. Historical/forecast valuation remains separate from these native amounts.

Observed boundaries: Funding and internal-deposit event times are seconds; most detail/query times are ms. Flexible yield uses `result.list`, P2P uses `ret_code` and list `items`. Flexible positions have IDs in this sample; hourly accruals do not. The tuple coin/productId/hourlyDate repeats uniquely in one query/replay, but its revision/collision policy remains unresolved. Current Flexible principal is zero with prior orders/yield; USDT lifetime totalPnl differs from the available yield sum. Fixed queries are empty within their bounds and do not become an unused-product blocker. APR percent/hourly units remain as documented by live evidence.

Follow supplied cursors even on short pages. Convert index and P2P page completion are source-specific. Store pages durably before checkpoints, replay overlaps and retain revisions. An 89-day sampled backfill does not extend published yield retention (three months), P2P maximum 180 days or Convert web history from 2025-09-10. Partial history requires explicit coverage/opening-balance evidence. BYBIT-B02/B05 read access is closed; task-0.10 closes BYBIT-B03/B04 before task-4.4. This evidence supplement does not change the shared target contract version, financial runtime or database schema.

## OpenAI: selection contract and execution boundary

[task-0.8 evidence](evidence/openai.en.md) selects gpt-5.6-terra xhigh and owns pricing, strict proposal schema, limits, retention and qualification. Luna/Sol/MiniMax/DeepSeek are not used automatically. reasoning.effort=xhigh is the owner’s decision. Research JSON is not a public application API command: the gateway maps it into AIReview/Proposal, and the application revalidates source, actor/household, expected revision, Money, allocation and authority. Unknown fees differ from confirmed zero; kind, amount and refund month cannot be guessed. Response/refusal/incomplete/schema-error/unknown have distinct states.

The server configures model, reasoning, permitted tools and pricing, outside chat input. Every attempt records model/prompt/schema/pricing revision, input count, output cap, reservation, actual usage and validated outcome. Do not mix cache_write_tokens with cached input or bill reasoning twice on top of output. Budget months are UTC; unresolved reservations survive rollover and recovery. Model/contract changes require evaluation before qualification; never switch to a more expensive model to fix network failures.

The 10 MiB/10-page upload limits remain. Page splitting preserves source evidence and cannot create separate expenses without matching. Research does not replace server regression/authorization/retry checks. This clarifies the target AI contract; no existing AI API/store requires data migration.
