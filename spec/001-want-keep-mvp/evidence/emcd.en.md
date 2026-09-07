# EMCD: read-contract research

[Русский](emcd.md)

Date: 2026-09-07, Europe/Moscow. Task: [task-0.6 / Issue #6](https://github.com/pchkauu/want-keep/issues/6). The owner-provided Google Chrome session was inspected. Final verification base: `f12f215`; research started on `05ca027`, branch `docs/want-keep-mvp-sdd`.

**Research completed with blocking findings for automation.** The USDT wallet, existing Grow deposits, existing cards and P2P archive are readable in the UI. Their structured responses were not obtained; automatic import is not implemented. BLK-06 remains open under task-0.10; task-4.6 and the MVP remain **Not Ready**. This completes research under the [README](../README.en.md) rule, not adapter acceptance.

## Current scope: D-34

The owner explicitly included used crypto cards, Coinhold/Grow, the USDT wallet and previously used P2P. Mining has never been used; neither its current data nor history is required. Mining, the trading terminal, staking, Portfolio, Borrow, referral programs and other unused products/wallets are deferred. Their research **does not block EMCD/the MVP**; future expansion requires a new verified contract. Shared Want Keep features and other providers remain unchanged.

| Product | Included now | Outcome |
| --- | --- | --- |
| USDT | Main wallet, balance, all its movements and fees | UI reading confirmed; separate available/locked fields and stable wallet ID not established |
| Coinhold / Grow | Existing USDT deposits, terms, maturity, accruals, capitalization, payouts and transfers | Three listed deposits; fixed-term and indefinite details; first and last UI history pages for one deposit |
| Crypto cards | Existing Plus and Light, including the blocked card and its history; USD balances | Top-up and declined purchases inspected; successful clearing/refund/reversal and holds untested |
| P2P | Historical orders, amounts/statuses and links to wallet and bank legs | Archive and completed USDT sale for RUB accessible; complete structured log not obtained |
| Deferred products | No mandatory research or new mining API keys | No claim that EMCD lacks these products |

Included-wallet movements are retained even when the related product is deferred. Preserve source amount, status and provenance; unknown semantics require clarification. A crypto card is not a credit card without an established credit agreement. EUR in a card purchase is its original currency, not a new requirement for an EUR wallet or selectable reporting currency.

## Evidence

`confirmed` establishes only the stated observation; `inference` identifies a conclusion; `unverified` identifies a gap. Real amounts, addresses, card numbers, IDs, usernames and conversations are excluded below. UI routes with placeholders are not API endpoints. Public materials were inspected on the research date.

| ID | Source | Established facts and evidence limits |
| --- | --- | --- |
| EMCD-E01 | confirmed, owner decision | D-34; unused products do not block research or the MVP |
| EMCD-E02 | confirmed, [initial tab](https://emcd.io/pool/dashboard/) | Active session and navigation to wallet, cards, Grow and P2P. Mining was not connected; initial sign-in/MFA were not observed |
| EMCD-E03 | confirmed, [wallets](https://emcd.io/wallets/) and `/wallets/main-account` | Main aggregate, wallet, Grow, P2P, pending withdrawals and separate card account. Native USDT has eight displayed decimal places; USD valuation is rounded. The aggregate contains child products and is not an additional asset |
| EMCD-E04 | confirmed, [wallet history](https://emcd.io/wallets/history/main-account) | Coin/type/date filters, export, date/coin/type/amount/status; Show more increased the list from 10 to 25 operations. UI loading does not prove API page size or completeness |
| EMCD-E05 | confirmed, expanded network withdrawal | TxID, address and separate USDT fee; completed status. Address shape alone does not establish network; gross/net semantics remain unverified. Explorer was not opened |
| EMCD-E06 | confirmed, wallet history | Grow interest payouts, withdrawals and negative P2P operation rows. Two withdrawals share displayed date/amount: they are not proven duplicates. Labels and signs cannot replace identity/economic role |
| EMCD-E07 | confirmed, [Grow](https://emcd.io/deposits/) | Three USDT deposits: two with maturity and one without an end date. Separate amount and earnings; total earnings are not current balance |
| EMCD-E08 | confirmed, fixed-term `/deposits/deposit/{growId}` | Route UUID, annual rate, maturity, distinct accrued/earned figures, capitalization enabled and renewal disabled; daily timestamped rewards with successful status. Provider scale, day-count, rounding and balance composition unverified |
| EMCD-E09 | confirmed, first and last pages of that Grow | UI: 10 first-page rows, 5 last-page rows, 135 total; opening event and disabled Next on the last page. Intermediate pages were not exported. This is the end of one available UI list, not an API terminator, retention or complete-import proof |
| EMCD-E10 | confirmed, indefinite Grow and history filter | Capitalization disabled; interest goes to the wallet. Filter lists opening, top-up, withdrawal, closure, reward, capitalization, payout, bonus, unlock and auto-deposit. A filter option does not prove an actual event |
| EMCD-E11 | confirmed, [cards](https://emcd.io/cards/) | Plus and blocked Light, USD balances, top-up limit, combined history and card/month filters. A card mask is not stable identity. Plus balance is not split into available and reserve |
| EMCD-E12 | confirmed, declined Light purchase in USD | Negative nonzero list amount; details reveal decline and a separate fee. Fee posting and hold linkage are unverified; declined principal is not an expense |
| EMCD-E13 | confirmed, declined Light purchase in EUR | Original EUR amount, approximate USD valuation, timestamp including year, USD fee and decline reason. Approximation is not exact settlement or execution rate |
| EMCD-E14 | confirmed, Plus `/cards/{cardId}` | Route UUID, masked credentials, balance, monthly summary; one timestamped USD top-up. No exposed source transaction ID, USDT leg, rate or fee. PAN/CVV were not revealed |
| EMCD-E15 | confirmed, [P2P archive](https://emcd.io/p2p/history) | Counter of 89, first 10 rows, operation/status/period filters, loading, shortened IDs, two currencies and status. Not 89 verified orders; list amounts are rounded |
| EMCD-E16 | confirmed, `/p2p/history/{orderId}` | Completed USDT sale for RUB; route UUID, start/end, price, quantity, RUB total and payment method. List currency order under Sell → Buy conflicts with the Sold direction in details. Full precision/fee and wallet-log linkage unknown; conversation is not evidence of bank receipt |
| EMCD-E17 | confirmed, [Mining Pool API 1.3.0](https://emcd.io/pool/api/) | Published HMAC-SHA256, X-API-Key/X-Timestamp/X-Signature, master_readonly/master_readwrite, subaccount context; hashrate/workers/earnings/payouts/subaccounts/watcher/addresses/time. No USDT wallet, Grow, card or P2P contracts in this API; no keys created |
| EMCD-E18 | confirmed, [API settings](https://help.emcd.io/en/articles/14461030-api-settings) | Watcher monitors equipment; documentation links to E17. It does not establish financial coverage of selected products |
| EMCD-E19 | confirmed, [Wallet](https://help.emcd.io/en/articles/16205516-what-is-emcd-wallet) | Custodial wallet, coin balances, multiple networks and account purposes. An internal EMCD transfer may go to another user rather than stay within the household |
| EMCD-E20 | confirmed, [Grow overview](https://help.emcd.io/en/articles/5744364-grow-your-savings-crypto-wallet) | Fixed, partial-withdrawal and indefinite types have different exit restrictions. Marketing cannot replace an existing deposit's terms |
| EMCD-E21 | confirmed, [Grow yield](https://help.emcd.io/en/articles/8936321-grow-yield) | Tables for rates, terms, capitalization and exit. Observed existing-deposit rates differ from current tables; the individual contract needs an effective schedule |
| EMCD-E22 | confirmed, [Grow FAQ](https://help.emcd.io/en/articles/8936354-grow-emcd-faq) | Daily accrual at 00:00 UTC, 30-day capitalization/payout, 24-hour fund eligibility and setting-dependent renewal. Standard and fast withdrawals differ. Exact API calculation basis/rounding are not established |
| EMCD-E23 | confirmed, [Plus/Premium](https://help.emcd.io/en/articles/12087413-payment-card-plus-premium) | USDT funding converts to USD; distinct fees, issuance reserve, transactions and settlement-date conversion. Do not treat the entire shown balance as available or apply current tariffs automatically to old events |
| EMCD-E24 | confirmed, [Light](https://help.emcd.io/en/articles/15908034-payment-card-light) | Legacy product with separate rules, including decline fees. Light is no longer issued, but the existing blocked card and its history remain in scope |
| EMCD-E25 | confirmed, [P2P statuses](https://help.emcd.io/en/articles/14085058-emcd-p2p-order-statuses-what-each-stage-means) | Waiting for payment/confirmation, dispute, cancellation and completion; escrow and release differ. EMCD status does not replace the bank-side record |
| EMCD-E26 | confirmed, [ToS, 2026-07-21](https://s3.emcd.io/strapi/2026_07_21_Tos_Blockchain_Tech_e77b7f4aa0.pdf), definitions and §3 | Shared sign-in joins products of different operators, not their contracts. Access must be protected; the document does not establish scopes, terms or supported operation of our automated reader |

## Adapter contract and unresolved fields

AS IS: active session → selected product → history → details. Separate frontend modules and a Mining API version do not establish a shared financial API. This iteration obtained no HAR or real JSON request/response for selected products. Do not present a UI route as an HTTP read endpoint or implement financial import by parsing labels.

| Boundary | Required from task-0.10 before task-4.6 |
| --- | --- |
| Access | Verified free personal API or permitted structured portal reading; scopes, authentication, session lifetime/refresh, request limits and hourly operation. Mining permissions do not cover other products |
| Identity | Verified external owner, persistent wallet/deposit/card and transaction IDs in household/provider/external-account/product/log namespace. A UI UUID is a candidate, not proof of reauth stability. Mask, address, currency, connectionId, text and amount are not identity |
| Balances | Native asset/amount/scale, sourceAsOf, independent knownness for owned/available/locked. Establish aggregate composition, Grow principal/accrued/capitalized amounts, card reserve and pending withdrawals. Entire Grow balances are not spendable; unknown holds do not become zero |
| Wallet | Type, direction, status, source ID/revision, timestamp/timezone, fee and gross/net, network/hash for blockchain movement. Some internal movements lack TxID; it is not universal identity |
| Grow | Initial principal, flows, effective rate schedule, day-count, rounding, cutoff, term/exit mode, capitalization/payout and renewal. Recognize rewards once; later capitalization or transfer of recognized income does not create it again |
| Card | Separate authorization/clearing/hold/release/decline/refund/reversal and fees; USDT → USD funding linkage. Declined principal is not an expense; fees need confirmed effects. Authorization and clearing cannot be added together |
| P2P | Verified owner-side buy/sell, exact crypto/fiat/gross/net/fee and order ID, wallet-leg and bank-fact linkage. List currency order or rounded prices cannot determine postings. Selling owned USDT for RUB is an exchange, not income equal to all RUB received |
| History | Date filters/timezone, ordering, cursor/offset, terminal marker, retention and late revisions per log; repeat/resume without gaps. One Grow UI ending does not establish coverage for other logs |
| Household/security | Second independent account, reconnection of one source, owner reauth, version/lease and rejection of stale results after disconnect. Never extract PAN/CVV, keys, cookies or conversations for AI |
| Read-only | Exact origin/method/path/body allowlist after routes are proven. Deny transfers, trades, Grow/renewal changes, card opening/funding/freeze/closure, P2P confirm/release/cancel and messaging. MFA/CAPTCHA goes to the owner; unknown routes are not permitted |

Target-contract version 5 narrows EMCD only: D-34/REQ-047/AC-047. Links from task-4.6 and REQ-047 to trading/mining AC-071 are removed; task-6.5 no longer depends on EMCD. AC-070 remains for unknown Grow accrual basis. Shared mining, credit-card and trading requirements remain. The task-1.1 foundation exists; financial runtime/database are not implemented and no migration is needed.

## Synthetic scenarios

[emcd.samples.json](emcd.samples.json) contains six **designed verification scenarios**, not EMCD responses. All values are fictional; `source_request`/`source_response` are null because the contract was not obtained. They define expected decisions for future fixtures:

1. Parent summary and child wallet/Grow are not added twice; card reserve differs from available, and unknown available remains null.
2. Reward accrual, capitalization and payout yield one income event; moving its storage location is not new income.
3. A declined purchase has zero principal expense; a separately confirmed fee remains, while an unknown fee awaits evidence.
4. USDT → USD card funding links as an exchange/internal movement; fees remain separate and parity is not assumed.
5. A P2P order, wallet movement and bank receipt yield one exchange; rounded UI figures do not recalculate exact source amounts.
6. One Grow's last UI page does not establish completeness of all logs or a verified API terminal marker.

## Blockers and handoff

| ID | Status | Resolution |
| --- | --- | --- |
| EMCD-B01 | CLOSED | UI access to all four included areas confirmed by E02–E16 |
| EMCD-B02 | OPEN | task-0.10: structured read contract for selected products, automation terms, free access, auth/scopes/limits and allowlist. Mining API is not a substitute |
| EMCD-B03 | OPEN | task-0.10: identity/revisions, precision/dates/statuses, full history and repeat/resume, reauthorization and second external account |
| EMCD-B04 | OPEN | task-0.10: balance/reserve composition; Grow accrual/capitalization/payout linkage and exact forecast terms; card lifecycle/fees/FX; owner-side P2P and log linkage |
| EMCD-B05 | DEFERRED, NON-BLOCKING | Products outside D-34; extend only following a future owner decision |

The task-0.6 research outcome is complete. task-0.10 receives E01–E26, B02–B04 and verification scenarios; resolving them and the shared Ready gate enables task-4.6. UI access alone does not unblock the adapter. For a structured sample, the next step is a safe HAR recording of selected read screens/pages or supplied official schemas for the relevant personal API; originals stay outside Git and only checked synthetic projections are published. Authentication recording and revealing payment credentials are unnecessary.

## Verification

Read-only UI observations and official-source checks were performed. Final-candidate commands: `make docs-check` (spec check and generator tests), JSON/Decimal scenario checks, RU/EN IDs/links/meaning and `git diff --check`. Execution and delivery results are recorded in Issue #6 after verification.

Not performed: credentialed HTTP replay/API calls, complete page import, reauth/second account, hourly runtime, allowlist enforcement, forecasts, application tests/build/E2E. Reasons: selected API contracts unavailable and financial workflows and EMCD adapter absent; the task-1.1 foundation exists. No runtime pass is claimed for AC-047/AC-041/AC-048/AC-070/AC-079/AC-087. The earlier Avida pass does not cover this research; current self review checks documentation.

Final-candidate checks on base `f12f215`: `make docs-check` — pass, 87 REQ / 105 AC / 67 tasks / 35 screens / 67 GitHub mappings; 14 generator tests — pass. Six Decimal scenarios, 26 paired observations/IDs/RU/EN links, JSON, scoped EMCD catalog changes and absence of private IDs — pass. Publication content was checked separately; unfinished changes from another research task were preserved.
