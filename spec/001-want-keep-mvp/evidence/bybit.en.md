# Bybit: Funding, Easy Earn and P2P research

[Русский](bybit.md) · [task-0.4 / Issue #4](https://github.com/pchkauu/want-keep/issues/4)

## Outcome and evidence boundary

Research completed on 2026-09-07. Official V5 API is the preferred integration. Public FlexibleSaving product requests for USDT, USDC, ETH and BTC and a USDT APR-history request returned HTTP 200 / `retCode=0` from the research environment. These calls establish public access and response shape, **not authenticated access to the owner's money**. The inspected Chrome account had no API keys. P2P advertiser prerequisites are not met in the inspected portal; no application or key was created.

Reference checkout: `f12f21597a9570846e78238b28d6bfc917ba0f4e`, `docs/want-keep-mvp-sdd`; task-1.1 foundation exists, financial integration/API/storage do not. Browser observations use the owner's existing Google Chrome session on bybit.com. No cookies, private network traffic, hidden state or credentials were extracted. No trade, withdrawal, subscription, redemption, security change, export job or support message was submitted. Public examples are synthetic; personal balances, UID, order IDs and counterparties are omitted.

**task-0.4 is complete as research; BLK-04 and task-4.4 remain Not Ready.** task-0.10 owns closure of BYBIT-B02–B05. Unused products are BYBIT-B06, deferred without blocking. Product ACs are not declared passed by this report.

## Current scope: D-36

| Product | Required coverage | Result |
| --- | --- | --- |
| Funding | USDT, USDC, ETH, BTC balances and all their movements, including conversion, on/off-chain deposits, withdrawals, fees and internal transfers | UI inspected; official balance, ledger and enrichment APIs documented; authenticated mapping unverified |
| Easy Earn | Used savings, principal, subscriptions/redemptions, accrual versus payout and forecasts | Public FlexibleSaving reads pass. Flexible/Fixed distinction remains to be established for actual holdings/history; neither a generic Earn label nor an empty current overview resolves it |
| P2P | Personal completed/cancelled/pending orders, crypto/fiat legs, fees, status and bank linkage | Two completed USDT/RUB sales visible; pending list empty in inspected view. API requires advertiser access, absent in inspected portal |
| Other products | Spot/UTA trading, futures, options, Bybit Card, On-Chain/Advanced Earn and other unused products | Deferred, not a readiness blocker. Movements through included wallets remain in scope |

USDC becomes a distinct accounting and selectable reporting asset alongside RUB, USD, USDT, BTC and ETH. No assumed USD/USDT/USDC parity. Similar symbols such as USDC.E, USDCX or BYUSDT must not be silently mapped to USDC/USDT. Expansion requires an explicit scope decision and verified contracts; no universal speculative adapter is required.

## Observations

`confirmed` is bounded to the stated UI observation, published contract or public call. `inference`, `unverified` and `contradiction` are deliberately separate.

| ID | Evidence / status | Finding and limit |
| --- | --- | --- |
| BYBIT-E01 | confirmed, user instruction | Funding USDT/USDC/ETH/BTC, occasional Easy Earn and P2P; API preferred; other products non-blocking deferred |
| BYBIT-E02 | confirmed, [dashboard](https://www.bybit.com/ru-RU/dashboard) and API management | Signed-in main account, verification indicator; API-key table explicitly empty. No private API run, key creation or reauthorization |
| BYBIT-E03 | confirmed, [Funding](https://www.bybit.com/user/assets/home/fiat) | All four symbols visible after disabling the small-balance filter and searching. Total/available/in-use and display equivalents shown. Zero display is not proof of exact zero; original filter restored |
| BYBIT-E04 | confirmed, [Funding history](https://www.bybit.com/ru-RU/user/assets/records/statements) | Native inflow/outflow, conversions, available-after amounts, date/time, type/description; first page has 20 rows and navigation through 10 pages for the selected month. Residual amounts have more decimals than wallet cards. Full history and API identity/linkage were not tested |
| BYBIT-E05 | confirmed, [P2P orders](https://www.bybit.com/ru-RU/p2p/orderList) | Pending empty; All shows two completed USDT/RUB sales, full order-number field, date/time, price and both amounts. No bank receipt, fee basis, cancelled lifecycle or retention proof |
| BYBIT-E06 | confirmed + inference, [advertiser program](https://www.bybit.com/ru-RU/p2p/identify/GA) | Portal says publication permission is absent and offers an application; general-maker application disabled with unmet requirements. Together with [P2P guide](https://bybit-exchange.github.io/docs/p2p/guide), this does not establish current P2P API eligibility. No role change attempted |
| BYBIT-E07 | confirmed, [Earn](https://www.bybit.com/ru-RU/earn/home/) | Rounded zero overview, Flexible and Fixed offers visible. Clicking the personal total redirected to the app-download page in this session. No application installed; actual positions/order/yield history not verified |
| BYBIT-E08 | confirmed, public GETs at 01:11 UTC | Four `/v5/earn/product` queries each returned one FlexibleSaving product; USDT APR history returned 167 points. Public catalog IDs are discovered, never hardcoded as owner identity. No chosen-VPS reachability proof |
| BYBIT-E09 | contradiction, [APR history](https://bybit-exchange.github.io/docs/v5/finance/earn/easy-onchain/apr-history) versus live response | Field table describes decimal fractions and midnight timestamps; example and live response use percent strings, and adjacent live timestamps are one hour apart. Preserve units/granularity explicitly; do not multiply rates by 100 or aggregate as one daily point blindly |
| BYBIT-E10 | contradiction / unverified, provider documentation | Yield schema says `result.list`, example uses `result.yield`; transfer HTTP example differs from declared route; internal-deposit created-time example uses seconds despite millisecond request filters. These are contract questions, not permission to guess a tolerant parser |

## Official read-contract matrix

All private routes below are **published contracts, not owner-response fixtures**. Authentication: [V5 guide](https://bybit-exchange.github.io/docs/v5/guide); HMAC-SHA256 or RSA, timestamp/receive-window and `X-BAPI-SIGN`. Do not copy the guide's contradictory prose placing a signature in `X-BAPI-API-KEY`; header definitions/examples distinguish them. Select the account's authorized regional service, not a domain inferred solely from server location. Public access here does not prove DE/NL/BG deployment access or private permissions.

| Purpose / source | Method and route | Material contract / gap |
| --- | --- | --- |
| [Key identity](https://bybit-exchange.github.io/docs/v5/user/apikey-info) | GET `/v5/user/query-api` | `userID`, master/parent identity, `readOnly=1`, permissions, IP/expiry. Response also echoes `apiKey`: redact before any evidence/log/AI handling. Credential rotation must preserve external identity |
| [Funding balances](https://bybit-exchange.github.io/docs/v5/asset/balance/all-balance) | GET `/v5/asset/transfer/query-account-coins-balance?accountType=FUND&coin=USDT,USDC,ETH,BTC` | `memberId`, `accountType`, per-coin decimal `walletBalance`, `transferBalance`, bonus. Transferable is not automatically budget-spendable; missing/empty values are not zero |
| [Funding ledger](https://bybit-exchange.github.io/docs/v5/asset/fund-history) | GET `/v5/asset/fundinghistory` | Paired seconds-based `createTimeFrom/To`, ≤7 days, string limit 1–100, cursor. `currcCursor` documented unique for dedup; `ioDirection=I/O`, `txnAmt`, `afterAmt`, seconds `createTime`. Business fields are localization keys/text, not a published exhaustive status/type enum. Retention and cross-log linkage unresolved |
| [On-chain deposits](https://bybit-exchange.github.io/docs/v5/asset/deposit/deposit-record) | GET `/v5/asset/deposit/query-record` | Millisecond filters with second-level query precision; <30-day windows, cursor, ≤50 rows; deposit `id`, chain/txID/txIndex, amount/fee/status/successAt. Transaction hash alone is insufficient identity |
| [Off-chain deposits](https://bybit-exchange.github.io/docs/v5/asset/deposit/internal-deposit-record) | GET `/v5/asset/deposit/query-internal-record` | ≤30 days, cursor, ≤50; `id`, `txID`, `fromMemberId`, amount, statuses 1/2/3. Provider-internal does not mean household-internal. Address may be personal contact data; minimize access/retention |
| [Withdrawals](https://bybit-exchange.github.io/docs/v5/asset/withdraw/withdraw-record) | GET `/v5/asset/withdraw/query-record?withdrawType=2` | Master key; on/off-chain, <30-day windows, ≤50, cursor. `withdrawId`, amount, `withdrawFee`, status, created/updated ms, txID. Establish gross/net and fee asset before posting; no extra fee deduction from a ledger gross debit |
| [Internal transfers](https://bybit-exchange.github.io/docs/v5/asset/transfer/inter-transfer-list) | GET `/v5/asset/transfer/query-inter-transfer-list` | Same UID, `transferId`, native amount, from/to account type, status/time; ≤7-day windows and ≤50 rows/cursor. Movements to an excluded account retain provenance and need counterpart classification |
| [Convert history](https://bybit-exchange.github.io/docs/v5/asset/convert/get-convert-history) | GET `/v5/asset/exchange/query-convert-history` | Include `funding` web/app and `eb_convert_funding` API history as applicable; `exchangeTxId`, both coins/amounts, status/rate/time; index pages ≤100. Web conversions documented from 2025-09-10; older completeness unproven. No stated time filter/terminal cursor; establish completion strategy |
| [Flexible products](https://bybit-exchange.github.io/docs/v5/finance/earn/easy-onchain/product-info) | GET `/v5/earn/product?category=FlexibleSaving` | Public; coin/productId, subscription precision, estimated APR and availability. Product order precision is not ledger scale; estimate excludes some rewards and is not promised income |
| [Flexible positions](https://bybit-exchange.github.io/docs/v5/finance/earn/easy-onchain/position) | GET `/v5/earn/position?category=FlexibleSaving` | Earn permission; amount, productId/coin, claimable yield, redeemable/frozen state. Fully redeemed Flexible positions can remain. `id` is documented for OnChain only: do not require it for Flexible or treat productId globally as owner identity |
| [Flexible orders](https://bybit-exchange.github.io/docs/v5/finance/earn/easy-onchain/order-history) | GET `/v5/earn/order?category=FlexibleSaving` | Earn permission, orderId, Stake/Redeem, amount, status, created/updated ms; ≤7-day windows, ≤100 and cursor. Maximum retention unspecified |
| [Distributed yield](https://bybit-exchange.github.io/docs/v5/finance/earn/easy-onchain/yield-history) | GET `/v5/earn/yield?category=FlexibleSaving` | Three-month history; ≤7-day windows, ≤100/cursor. Per-user unique yield ID, amount/status/distributionMode; manual payout may reference redemption order. Resolve list/yield envelope contradiction with authenticated evidence |
| [Hourly accrual](https://bybit-exchange.github.io/docs/v5/finance/earn/easy-onchain/hourly-yield) | GET `/v5/earn/hourly-yield?category=FlexibleSaving` | Earn permission; per-user ID, amount, effective principal, status/hourlyDate/createdAt. ≤7 days, ≤100/cursor; retention unspecified. Hourly accrual, distributed yield and Funding credit are not three incomes |
| [APR history](https://bybit-exchange.github.io/docs/v5/finance/earn/easy-onchain/apr-history) | GET `/v5/earn/apr-history` | Public; category/productId, ms filters; documentation allows six months / ≤182 days. Live percent format and hourly spacing in E09 govern this observed case; full history was not verified |
| [Fixed products](https://bybit-exchange.github.io/docs/v5/finance/earn/fixed-saving/product), [positions](https://bybit-exchange.github.io/docs/v5/finance/earn/fixed-saving/position), [orders](https://bybit-exchange.github.io/docs/v5/finance/earn/fixed-saving/order) | GET `/v5/earn/fixed-term/product`, `/v5/earn/fixed-term/position`, `/v5/earn/fixed-term/order` | Separate contracts if used: product/category and term, positionId and maturity, orderId and settlement yields. Positions omit settled holdings; order filters use creation time for Stake and settlement for Redeem, cursor ≤50. No authenticated sample; do not use Flexible empty results as Fixed coverage |
| [P2P access](https://bybit-exchange.github.io/docs/p2p/guide), [orders](https://bybit-exchange.github.io/docs/p2p/order/order-list), [detail](https://bybit-exchange.github.io/docs/p2p/order/order-detail) | POST `/v5/p2p/order/simplifyList`, `/v5/p2p/order/info` | Read semantics despite POST. Advertiser required; read-only supported by [P2P introduction](https://www.bybit.com/en/help-center/article/Introduction-to-P2P-Open-API). Page/size ≤30; default 90 days, maximum 180. ID, side 0/1, crypto quantity, fiat amount/price, fee/status. P2P examples use `ret_code`, distinct from V5 `retCode`; confirm actual envelope and timestamp/fee units |

Relevant status rules: [provider enums](https://bybit-exchange.github.io/docs/v5/enum). Deposit success may later be rolled back; keep a revision/correcting effect. Pending, rollback review, unknown, rejected and cancelled do not silently become settled. P2P status 50 is finished, 40 cancelled, 10/20 payment/release pending; it does not independently prove a matching bank entry.

Hourly sync is modest, but backfills must obey each route's window and limit. [Rate limits](https://bybit-exchange.github.io/docs/v5/rate-limit): Funding ledger 30/s, all-balances 5/s, internal transfer 60/min, on-chain deposit 100/min; use response limit/reset headers. Shared IP ceiling is 600/5s; 10006 triggers controlled backoff, frequency-related 403 requires the documented cooldown, not a domain switch. Rate/permission failures never produce empty-success coverage. Actual scopes, throttling and VPS reachability remain untested.

## Target mapping and verification contract

These are requirements for task-0.10/task-4.4, **not implemented behavior**:

1. Isolate household/provider/site/external UID and FUND/asset identity; key or connection changes are provenance. Do not merge spouses by symbol/address. Separate log identity from financial-event identity. Secrets live in protected infrastructure, never AI context. Verify readOnly and per-route capabilities before enabling import; do not grant write scopes to overcome rejection.
2. Read Funding as the monetary ledger and enrich from deposit/withdraw/transfer/convert/Earn/P2P records. Establish status and linkage before classification; do not create another effect for each endpoint. Localization labels never implement business-critical state. Preserve unresolved records with explanation and reconciliation, not guessed income or duplicate posting.
3. Keep exact source decimals, even residuals beyond the displayed/order precision. Bonus, hold, principal and claimable yield are distinct. Compare time-aligned balances and movements; snapshots do not become postings. Null/empty/unsupported differ from exact zero.
4. Subscription/redemption principal is an internal movement, earned yield income once; reinvestment is a separate linked movement. [Easy Earn FAQ](https://www.bybit.com/en/help-center/article/FAQ-Easy-Earn) describes hourly Flexible accrual and daily Funding payouts, with return of undistributed yield on redemption described by the position contract. Verify actual product terms, principal/time basis and adjustments; APR/APY, projected and credited returns remain distinct.
5. P2P exchanges link native crypto and fiat legs to the bank/cash account; missing bank evidence prompts clarification. A platform-internal transfer is family-internal only with ownership evidence. P2P names/contact details are not identity or proof of settlement. Export/receipt buttons do not establish an automated API.
6. Checkpoint each page/window only after durable storage; replay overlap and retain IDs/revisions. An empty result outside retention or on an unsupported product never proves a zero history. Authenticate two independent owners, reconnect/rotate a key, revoke a connection and reject stale jobs.
7. Prefer V5 for all verified coverage. A P2P browser fallback needs a permitted structured read contract, source identity, states, pagination and exact route allowlist. The read POST routes above require explicit allowlisting; order creation, payment acknowledgement, release, ads, subscription/redemption, transfer, withdrawal and key changes remain forbidden. Neither DOM reading nor financial OCR is sufficient proof of reliable automation.

[Synthetic samples](bybit.samples.json) are project-generated projections/scenarios, not private provider responses. They cover FUND query/decimal fields, directed ledger replay, conversion enrichment, withdrawal fee once, yield/accrual/credit once, P2P versus bank linkage, distinct UIDs, APR units and partial history. All real IDs/numbers are replaced. Private envelopes, gross/net linkage and accounting transitions await evidence.

## Remaining blockers and handoff

| ID | Status / owner | Closure evidence |
| --- | --- | --- |
| BYBIT-B01 | CLOSED for research access | Chrome UI and public Earn requests read; not authenticated API readiness |
| BYBIT-B02 | OPEN; task-0.10 with owner, task-0.9 for deployment | Owner securely provisions a read-only key with minimum required Wallet/Earn/Exchange and eligible P2P permissions; prove readOnly/UID, successful allowed calls, free access and authorized regional/VPS reachability. No key in chat/Git/logs |
| BYBIT-B03 | OPEN; task-0.10 | Private fixtures, exact units/envelopes, Funding↔detail linking, gross/net/fee and status/revision rules, historical bounds, page completion/replay, reconcile balances, two owners and rotation/revocation |
| BYBIT-B04 | OPEN; task-0.10 with owner | Identify used Flexible/Fixed products and dated flows; actual positions/orders/yield with principal versus income, payout/reinvestment and verified forecast basis. Resolve E09/E10; unused advanced products are not required |
| BYBIT-B05 | OPEN; task-0.10 with owner/provider | An eligible read-only P2P API path or permitted structured browser-read contract, including 180-day API boundary, status/fee/native legs and bank matching. Do not automatically apply for advertiser status or omit P2P |
| BYBIT-B06 | DEFERRED, NON-BLOCKING | Other products excluded by D-36; future explicit extension with contracts. Included-wallet movements remain required |

Research supplies task-0.7 with USDC coverage needs, task-0.9 with access/limit checks, task-0.10 with the contract matrix and task-4.4 with bounded implementation inputs. It does **not** release the shared Ready gate. REQ-002/REQ-003/REQ-039/REQ-045 and ACs/tasks retain IDs; version 6 is a target-spec change. No financial API schema, migration or application behavior changes in this task.

## Verification

Performed: dated primary-document review; Chrome Funding/history/P2P/Earn/API-management checks; six public GETs total (initial USDT product probe plus four product queries and APR history); synthetic JSON/decimal checks; RU/EN semantic and reference review; repository `make docs-check` and diff review recorded at delivery.

Not performed: private signed calls (no key), P2P API (eligibility gap), complete backfill, actual Earn lifecycle, reauthentication, second-owner or revocation runtime, hourly collector, chosen-VPS tests, live financial writes. No application/adapter code or runtime tests were added. The research browser tab was returned to the original dashboard.
