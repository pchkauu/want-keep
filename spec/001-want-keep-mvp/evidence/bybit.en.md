# Bybit: Funding, Easy Earn and P2P research

[Русский](bybit.md) · [task-0.4 / Issue #4](https://github.com/pchkauu/want-keep/issues/4)

## Outcome and evidence boundary

Research completed and extended with authenticated API evidence on 2026-09-07. The owner authorized an IP-restricted RSA read-only key and completed MFA. Official Funding, Flexible Easy Earn and P2P reads succeeded; **API access is confirmed for this account**, including both P2P order routes. The earlier portal-based P2P eligibility concern is superseded by BYBIT-E11–E18. [Authenticated findings](bybit-api.en.md) record precision, linkage and history limits; no browser collector is needed for the verified read coverage.

Initial UI reference: `f12f21597a9570846e78238b28d6bfc917ba0f4e`; authenticated-document update based on `2412b773ad449d87fa846126907daeb237f39bd1`, branch `docs/want-keep-mvp-sdd`. The foundation exists; the Bybit connector does not. Google Chrome was used for UI and authorized key provisioning; signed requests used the official API. Credentials were read locally for signing, never printed or placed in public evidence. No trade, withdrawal, subscription, redemption, advertiser application or support message was submitted. Samples replace personal balances, UID, order IDs and counterparties with synthetic data.

**task-0.4 is complete as research; the automatic connector is not implemented.** BYBIT-B01/B02/B05 are closed for observed read access. The original BYBIT-B03/B04 were handed to task-0.10; their D-39 decision and task-4.4 runtime gates are recorded in the final section below. Product ACs are not claimed as passed.

## Current scope: D-36

| Product | Required coverage | Result |
| --- | --- | --- |
| Funding | USDT, USDC, ETH, BTC balances and all their movements, including conversion, on/off-chain deposits, withdrawals, fees and internal transfers | Four balances, 357 ledger rows across 89 days, deposits/withdrawals/Convert read; see E12/E13 for precision and linkage limits |
| Easy Earn | Used savings, principal, subscriptions/redemptions, accrual versus payout and forecasts | Flexible positions for USDT/USDC/ETH, 10 orders, 141 yield rows and 48 hourly accruals read. Fixed queries empty within requested bounds; unused products do not block |
| P2P | Personal completed/cancelled/pending orders, crypto/fiat legs, fees, status and bank linkage | Official list and details of both completed USDT/RUB sales read successfully. Cancelled/pending lifecycle and bank settlement remain unverified |
| Other products | Spot/UTA trading, futures, options, Bybit Card, On-Chain/Advanced Earn and other unused products | Deferred, not a readiness blocker. Movements through included wallets remain in scope |

USDC becomes a distinct accounting and selectable reporting asset alongside RUB, USD, USDT, BTC and ETH. No assumed USD/USDT/USDC parity. Similar symbols such as USDC.E, USDCX or BYUSDT must not be silently mapped to USDC/USDT. Expansion requires an explicit scope decision and verified contracts; no universal speculative adapter is required.

## Observations

The table below preserves the **initial UI/public phase**. Its negative access statements apply to that time only; BYBIT-E11–E18 in the [authenticated report](bybit-api.en.md) establish the current private-call results. `confirmed`, `inference`, `unverified` and `contradiction` retain separate meanings.

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

The matrix combines published contracts with observed private responses; [E11–E18](bybit-api.en.md) identify exactly what was exercised. Authentication: [V5 guide](https://bybit-exchange.github.io/docs/v5/guide), verified RSA-SHA256/base64, `X-BAPI-SIGN` separate from `X-BAPI-API-KEY`, timestamp and receive window. The account accepted `api.bybit.com` from the IP allowlist; this does not prove chosen-VPS access or authorize regional bypass.

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
| [Flexible positions](https://bybit-exchange.github.io/docs/v5/finance/earn/easy-onchain/position) | GET `/v5/earn/position?category=FlexibleSaving` | Observed Flexible positions include a nonempty `id`, despite the OnChain-only documentation note; bind identity to owner/category/product, never a global product ID. Zero principal can coexist with historical yield. |
| [Flexible orders](https://bybit-exchange.github.io/docs/v5/finance/earn/easy-onchain/order-history) | GET `/v5/earn/order?category=FlexibleSaving` | Earn permission, orderId, Stake/Redeem, amount, status, created/updated ms; ≤7-day windows, ≤100 and cursor. Maximum retention unspecified |
| [Distributed yield](https://bybit-exchange.github.io/docs/v5/finance/earn/easy-onchain/yield-history) | GET `/v5/earn/yield?category=FlexibleSaving` | Observed `result.list`, unique yield `id`, decimal amount and Auto/Manual distribution; five manual rows reference redemption order IDs. ≤7-day windows, ≤100/cursor, documented three-month history. Zero yield does not require a Funding credit. |
| [Hourly accrual](https://bybit-exchange.github.io/docs/v5/finance/earn/easy-onchain/hourly-yield) | GET `/v5/earn/hourly-yield?category=FlexibleSaving` | Observed 48 rows have no `id`, although the published contract promises one; coin/productId/hourlyDate tuples are unique and replay-stable in the sample. This is not a guaranteed cross-run identity contract. Accrual is not additional income over distribution/Funding. ≤7 days, ≤100/cursor. |
| [APR history](https://bybit-exchange.github.io/docs/v5/finance/earn/easy-onchain/apr-history) | GET `/v5/earn/apr-history` | Public; category/productId, ms filters; documentation allows six months / ≤182 days. Live percent format and hourly spacing in E09 govern this observed case; full history was not verified |
| [Fixed products](https://bybit-exchange.github.io/docs/v5/finance/earn/fixed-saving/product), [positions](https://bybit-exchange.github.io/docs/v5/finance/earn/fixed-saving/position), [orders](https://bybit-exchange.github.io/docs/v5/finance/earn/fixed-saving/order) | GET `/v5/earn/fixed-term/product`, `/v5/earn/fixed-term/position`, `/v5/earn/fixed-term/order` | Separate contracts if used: product/category and term, positionId and maturity, orderId and settlement yields. Positions omit settled holdings; order filters use creation time for Stake and settlement for Redeem, cursor ≤50. Position/order reads succeeded with empty results in their requested bounds (E14); no populated Fixed record was established. Do not infer lifetime absence |
| [P2P access](https://bybit-exchange.github.io/docs/p2p/guide), [orders](https://bybit-exchange.github.io/docs/p2p/order/order-list), [detail](https://bybit-exchange.github.io/docs/p2p/order/order-detail) | POST `/v5/p2p/order/simplifyList`, `/v5/p2p/order/info` | Both read POSTs succeeded with readOnly + FiatP2POrder despite the earlier advertiser concern. Actual `ret_code`, string ID/quantity/amount/price and millisecond date strings; side/status are integers. Page/size ≤30, documented default 90/max 180 days. Native fiat amount is authoritative: quantity × price differs, consistent with observed crypto quantization; bank settlement is separate. |

Relevant status rules: [provider enums](https://bybit-exchange.github.io/docs/v5/enum). Deposit success may later be rolled back; keep a revision/correcting effect. Pending, rollback review, unknown, rejected and cancelled do not silently become settled. P2P status 50 is finished, 40 cancelled, 10/20 payment/release pending; it does not independently prove a matching bank entry.

Hourly sync is modest, but backfills must obey each route's window and limit. [Rate limits](https://bybit-exchange.github.io/docs/v5/rate-limit): Funding ledger 30/s, all-balances 5/s, internal transfer 60/min, on-chain deposit 100/min; use response limit/reset headers. Shared IP ceiling is 600/5s; 10006 triggers controlled backoff, frequency-related 403 requires the documented cooldown, not a domain switch. Rate/permission failures never produce empty-success coverage. Scopes and allowed calls are confirmed in E11; throttling and chosen-VPS reachability remain untested.

## Target mapping and verification contract

These are requirements for task-0.10/task-4.4, **not implemented behavior**:

1. Isolate household/provider/site/external UID and FUND/asset identity; key or connection changes are provenance. Do not merge spouses by symbol/address. Separate log identity from financial-event identity. Secrets live in protected infrastructure, never AI context. Verify readOnly and per-route capabilities before enabling import; do not grant write scopes to overcome rejection.
2. Read Funding as the monetary ledger and enrich from deposit/withdraw/transfer/convert/Earn/P2P records. Establish status and linkage before classification; do not create another effect for each endpoint. Localization labels never implement business-critical state. Preserve unresolved records with explanation and reconciliation, not guessed income or duplicate posting.
3. Keep exact source decimals, even residuals beyond the displayed/order precision. Bonus, hold, principal and claimable yield are distinct. Compare time-aligned balances and movements; snapshots do not become postings. Null/empty/unsupported differ from exact zero.
4. Subscription/redemption principal is an internal movement, earned yield income once; reinvestment is a separate linked movement. [Easy Earn FAQ](https://www.bybit.com/en/help-center/article/FAQ-Easy-Earn) describes hourly Flexible accrual and daily Funding payouts, with return of undistributed yield on redemption described by the position contract. Verify actual product terms, principal/time basis and adjustments; APR/APY, projected and credited returns remain distinct.
5. P2P exchanges link native crypto and fiat legs to the bank/cash account; missing bank evidence prompts clarification. A platform-internal transfer is family-internal only with ownership evidence. P2P names/contact details are not identity or proof of settlement. Export/receipt buttons do not establish an automated API.
6. Checkpoint each page/window only after durable storage; replay overlap and retain IDs/revisions. An empty result outside retention or on an unsupported product never proves a zero history. Authenticate two independent owners, reconnect/rotate a key, revoke a connection and reject stale jobs.
7. Prefer V5 for all verified coverage. A P2P browser fallback needs a permitted structured read contract, source identity, states, pagination and exact route allowlist. The read POST routes above require explicit allowlisting; order creation, payment acknowledgement, release, ads, subscription/redemption, transfer, withdrawal and key changes remain forbidden. Neither DOM reading nor financial OCR is sufficient proof of reliable automation.

[Synthetic samples](bybit.samples.json) contain twelve project-generated scenarios, including observed-shape projections with all personal data replaced. They are not raw private responses or passing adapter tests. S09–S12 cover missing hourly IDs, balance precision, P2P quantization and terminal pagination.

## Remaining blockers and handoff

| ID | Status / owner | Closure evidence |
| --- | --- | --- |
| BYBIT-B01 | CLOSED for research access | Chrome, public and private official reads observed |
| BYBIT-B02 | CLOSED for this owner/key | RSA readOnly, UID, allowlist and required route permissions verified. task-0.9 still checks authorized VPS reachability; task-4.4 tests expired/revoked keys and second-owner isolation |
| BYBIT-B03 | SDD RESOLVED; RUNTIME GATE task-4.4 | Specify deterministic cross-log identity/ambiguity rules, source status revisions, precision-aware reconciliation and requested-history coverage. E12/E13/E16/E17 provide concrete samples; full history and every lifecycle are not established |
| BYBIT-B04 | SDD RESOLVED; RUNTIME GATE task-4.4 | Finalize hourly identity/revision policy, USDT lifetime totalPnl versus available yield history, and forecast principal/rate/time basis. Flexible usage is confirmed. Fixed/other unused products do not block |
| BYBIT-B05 | CLOSED for P2P read access | List and both details succeed with read-only API. No advertiser application or Playwright required for observed coverage. Fee/quantization, status transitions and bank matching remain B03/implementation acceptance |
| BYBIT-B06 | DEFERRED, NON-BLOCKING | Other products excluded by D-36; explicit future extension only; included-wallet movements remain required |

Research handed task-0.7 USDC, task-0.9 access/limits, task-0.10 contract decisions and task-4.4 bounded implementation inputs. task-0.10 completed the shared SDD gate; task-4.4 remains the executable-conformance owner. REQ/AC/task IDs remain; this research changed no financial runtime or database schema.

## Verification

Performed: primary-document/UI research, six public GETs, authorized RSA provisioning and signed official read calls documented in E11–E18, decimal/replay checks, twelve synthetic scenarios, RU/EN and diff review, `make docs-check` at delivery.

Not performed: complete lifetime history, full status/revision lifecycle, second-owner/revocation runtime, hourly collector, chosen-VPS tests, bank settlement matching, production connector or application E2E. No financial mutations. Private API success is research evidence, not implementation acceptance.

## task-0.10 decision, 2026-09-07

BYBIT-B03/B04 above are closed as D-39 SDD decisions. Provider IDs remain route-specific; amount/time cross-log matches are candidates only. Hourly data without ID may use `(coin, productId, hourlyDate)` within the hourly namespace. A differing payload retains both evidence revisions as `source_ambiguous` without financial credit. Gaps and lifetime mismatch yield `source_partial`.

Precision/history, principal/yield reconciliation, a second account, rotation/revocation and live conformance remain the task-4.4 deployment gate. Official RSA read-only APIs retain priority; Playwright is allowed only for a newly proven API gap. The SDD is Ready; no running connector is claimed.
