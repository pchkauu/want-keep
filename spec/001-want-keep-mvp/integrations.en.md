# Integrations and research blockers

[Русский](integrations.md)

Public-documentation snapshot: 2026-09-06; research updates are dated in the rows below. Confirmed reading of individual products, history completeness and automation readiness are assessed separately. Public documentation or marketing descriptions do not establish personal-account access.

## Mandatory coverage

| Platform | Products | Established | Open work and task |
| --- | --- | --- | --- |
| Alfa-Bank | Debit/credit cards, current/savings, deposits; cashback | 2026-09-07: [research completed with blockers](evidence/alfa.en.md). Live reading of current/savings accounts, two deposit types, transactions and cashback; published retail accounts/cards/operations/loyalty APIs. | BLK-01 remains open: eligibility and verified automatic read contract, identity/completeness/reauth, second account, missing credit card, exact terms and FX/cashback lifecycle. task-0.1 completes research; task-4.1 and task-0.10 remain blocked. |
| Raiffeisenbank Russia | Individual entrepreneur current account only: balances, incoming/outgoing movements, fees, history (D-35) | 2026-09-07: [research completed](evidence/raiffeisen.en.md) with blockers. Refresh and account GETs from Mac/VPS — 200; two camt.053 reports, seven matching entries, OPBD/CLBD reconciled. DNS/TLS ready. | RAIF-B02/B03/B04/B06: current/available/locked balances, fees, history and auth lifecycle. Intraday 404 no-statements; callback 503. task-4.2 and MVP Not Ready. |
| Ozon Bank | Debit card and linked main account (D-32) | 2026-09-07: [research completed with blockers](evidence/ozon.en.md); five read routes, two HARs, 10 synthetic projections, separate commission and seven-page chain. | BLK-03 remains open: history completion, session operation, second account and unavailable-field semantics. task-0.10 verifies closure; task-4.3 is blocked. Other Ozon products are future extensions, not blockers. |
| Bybit | Funding USDT/USDC/ETH/BTC, used Easy Earn and P2P (D-36) | 2026-09-07: [research](evidence/bybit.en.md), [signed API reads](evidence/bybit-api.en.md) successful, including P2P; read-only RSA. No browser gap demonstrated. | BLK-04: BYBIT-B03/B04 — deterministic linkage, precision/history and hourly identity/Earn basis; task-0.10 → task-4.4. BYBIT-B02/B05 access closed. Other products non-blocking. |
| Aifory Pro | RUB accounts, USDT, ETH and existing USD card (D-33) | 2026-09-07: [research completed](evidence/aifory.en.md); UI reading of accounts, movements and card. | BLK-05: permission/structured read contract, identity/history/reauth and card lifecycle — AIFORY-B02–B04 under task-0.10. task-4.5 remains blocked. Other products deferred without blocking. |
| EMCD | USDT wallet, existing Coinhold/Grow, used crypto cards and historical P2P orders (D-34) | 2026-09-07: [research completed with blockers](evidence/emcd.en.md); four UI areas, 26 sources/observations and six synthetic scenarios. Mining Pool API does not cover selected products. | BLK-06: structured read contract, identity/history/reauth, balances/Grow/card/P2P — EMCD-B02–B04 under task-0.10; task-4.6 blocked. Mining has never been used; its history and other unused products are unnecessary. |

Sources: [Alfa developer portal](https://developers.alfabank.ru/), [Alfa onboarding](https://developers.alfabank.ru/products/alfa-api/documentation/articles/connection/connection), [Raiffeisen API](https://developer.raiffeisen.ru/), [Ozon Bank](https://finance.ozon.ru/), [Bybit wallet balance](https://bybit-exchange.github.io/docs/v5/asset/balance/all-balance), [Bybit transaction log](https://bybit-exchange.github.io/docs/v5/asset/fund-history), [Aifory Pro](https://aifory.pro/), [EMCD wallet](https://help.emcd.io/en/articles/16205516-what-is-emcd-wallet).

Ozon and some Alfa links were unavailable during repeated retrieval through the research tool. This limits research; it does not prove that APIs are absent.

## Required per-source research outcome

For every mandatory product record: product existence/owner availability; read method; required scopes; account identity/card aliases; owned/available/locked/debt balances; events/IDs/revisions/statuses; fees/net-gross; date/timezone; pagination/window/depth; terms/minimum/grace/accrual; quote direction/amount/fee; rate limits; reauth; endpoint allowlist; evidence date; synthetic fixture; live outcome.

A positive result needs actual comparison to an authorized source. An owner-inaccessible product, mandatory payment or missing acceptable automatic path produces a precise blocker and required decision; unsupported is not equivalent to implemented.

Prefer official read APIs with least privilege. Browser collection is allowed with owner consent for agreed read actions only; protect sessions and require human involvement for expiry/MFA. This does not authorize bypassing restrictions or executing external financial actions.

## Rates

[task-0.7 research](evidence/fx.en.md) completed on 2026-09-07. Primary USD/RUB comes from official Bank of Russia XML; Frankfurter v2 is allowed only with `providers=CBR` as fallback/cross-check. BTC/USD, ETH/USD, USDT/USD and USDC/USD use separate CoinGecko Demo observations for current values and history up to 365 days. Crosses use exact Decimal arithmetic through USD; default blended rates and a USD/USDT/USDC=1 assumption are forbidden. Similar tokens, including USDC.E, are not merged without verified identity mapping.

TradingView was rejected: charting libraries do not supply market data, and its terms prohibit automated price referencing/non-display processing. A reference price is not an executable quote. Provider buy/sell needs direction, applicable amount, timestamp and spread/fee from a verified source contract; unknown fields remain unavailable/unknown.

BLK-07 retains FX-B02–FX-B04 until task-0.10: a decision for crypto history older than 365 days, provider executable quotes and a keyed CoinGecko Demo probe/attribution. The research Issue can close, while task-6.1 and the MVP remain Not Ready.

## Closure order

task-0.1–task-0.7 produce specific evidence documents; task-0.8 verifies OpenAI and task-0.9 infrastructure. task-0.10 transfers verified contracts into the specification, updates affected tasks and reviews Ready. Adapter implementation depends on its own research and the common Ready gate.

## Two members and source identity

Each provider must support independent member accounts. One bank/crypto account reauthorized by another member must not create a second set of financial accounts. Research records stable external-account identity and transaction namespace separately from connectionId. Both manage synchronization; the external owner supplies MFA/password in a protected flow. These are requirements, not verified provider capabilities.
