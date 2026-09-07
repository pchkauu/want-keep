# Free rate sources: research and target contract

[Русский](fx.md)

Date: 2026-09-07, Europe/Moscow. Task: [task-0.7 / Issue #7](https://github.com/pchkauu/want-keep/issues/7). Repository base: `cce4a4b`, branch `docs/want-keep-mvp-sdd`. Public documents and anonymous HTTP responses were inspected; no paid plan, API key or financial account was used.

**Research is complete with precise limitations.** The MVP selection is: Bank of Russia as the primary USD/RUB source; CoinGecko Demo for current and at-most-365-day-old BTC/USD, ETH/USD, USDT/USD and USDC/USD; Frankfurter v2 with `providers=CBR` as a fallback transport/cross-check for CBR, with no blended rates. Older crypto valuation and executable platform quotes remain unavailable pending task-0.10 and provider-specific research. BLK-07 remains open under task-0.10; closing the research Issue does not establish Ready.

## Decision

| Purpose | Source | Usage boundary |
| --- | --- | --- |
| Current and historical USD/RUB | [Bank of Russia XML](https://www.cbr.ru/development/sxml/) | Primary official reference rate. Retain requested date, response effective date, fetchedAt and original decimal. A weekend/holiday uses the latest effective date `≤ D`; it does not create a new quote for every calendar day. |
| USD/RUB fallback | [Frankfurter v2](https://frankfurter.dev/) with `providers=CBR` only | Use when direct CBR is unavailable and for cross-checking. Default blend is forbidden. Retain actual provider `CBR`; Frankfurter is transport/provenance. An older effective date does not replace a newer direct CBR record. |
| Current BTC, ETH, USDT and USDC in USD | [CoinGecko Demo](https://www.coingecko.com/en/api/pricing), `/simple/price` | One batch for IDs `bitcoin,ethereum,tether,usd-coin` with `include_last_updated_at=true`; never use symbol-only identity. USDT and USDC are separate assets with no `1 USD` peg assumption. USDC.E is not merged with USDC without verified provider mapping. |
| Historical BTC, ETH, USDT and USDC in USD | CoinGecko `/coins/{id}/history` | One daily UTC snapshot per asset/date, at most 365 days old under the free plan. Retain requested date, returned timestamp/day, granularity and source. This is reference valuation, not an execution price. |
| Reporting cross-rates | The USD legs above | Calculate with exact Decimal arithmetic. If any leg is missing/stale under the contract, return partial/unavailable; never substitute zero, a current price for history or USD/USDT/USDC=1. |
| Actual exchange, buy/sell or P2P | The operation's provider | Retain both native amounts, direction, applicable amount, provider timestamp and known fees. A reference source cannot invent a spread/fee/executable quote. Missing fields remain unknown. |

This combination is free at the researched scale. CBR publishes no quota in the inspected documentation; that means `unknown`, not unlimited. Frankfurter has no daily/monthly quota but applies anti-abuse rate limiting. As inspected, CoinGecko Demo provides 10,000 calls/month, 100/min, one API key, freshness from 60 seconds, mandatory attribution and up to one year of daily/hourly history. Recheck terms and limits during implementation and expose them in source status.

## Reproducible valuation rule

For source asset `S`, target asset `T` and instant/date `D`, retain `P_USD(X,D)`, USD per one unit of `X`. USD is `1`; RUB is `1 / CBR_USD_RUB(D)`; BTC, ETH, USDT and USDC use separate CoinGecko observations.

`R(S→T,D) = P_USD(S,D) / P_USD(T,D)`

`value_T = amount_S × R(S→T,D)`

Use Decimal and an explicit rounding policy at the presentation boundary only. Retain every input observation and both dates. Different effective dates across legs remain visible: reports expose source/asOf and coverage for each leg.

A historical transaction receives a civil date in the household budget timezone. Its CBR leg is the latest official effective date `≤ D`. Its CoinGecko leg is that date's daily UTC snapshot. A current balance uses the latest successfully fetched current observation; a response without a provider timestamp is not a new observation. A source correction creates an audited valuation revision instead of silently rewriting history.

Synthetic example: if a source reports `BTC/USD = 50000.00`, `USDT/USD = 0.9970`, and CBR reports `USD/RUB = 90.0000`, then `BTC/USDT = 50000.00 / 0.9970`. The denominator cannot become `1`, and the reference result cannot be presented as a price at which Bybit/Aifory/EMCD will execute an exchange.

## Evidence ledger

`confirmed` applies only to the named document or HTTP observation; `inference` is a project conclusion; `unverified` means evidence is absent. Live rates are excluded from examples.

| ID | Status and source | Established | Evidence boundary |
| --- | --- | --- | --- |
| FX-E01 | confirmed, [CBR XML](https://www.cbr.ru/development/sxml/) | `XML_daily.asp` returns the latest registered day or selected date; `XML_dynamic.asp` returns a currency-code range | No published quota/SLA; no crypto price or executable bank quote |
| FX-E02 | confirmed, [CBR about](https://www.cbr.ru/about/) and [user agreement](https://www.cbr.ru/user_agreement/) | Open materials may be reproduced with source attribution | Recheck current terms before production and retain attribution |
| FX-E03 | confirmed, live GET 2026-09-07 | Current XML: HTTP 200, Windows-1251, effective date 2026-09-05, USD `Nominal/Value/VunitRate`; selected historical week: five observations | Point-in-time access from the current network, not an SLA; CBR headers contained DDoS cookies/IP and were not committed |
| FX-E04 | confirmed, [Frankfurter docs](https://frankfurter.dev/) and [OpenAPI](https://api.frankfurter.dev/v2/openapi.json) | No key, historical/range API, provider filter, no daily/monthly quota, anti-abuse limiting; default is blended | Underlying-provider terms still apply; service is not intended for live trading |
| FX-E05 | confirmed, live GET 2026-09-07 | `USD/RUB?providers=CBR` and a historical range returned typed JSON; provider metadata identifies CBR and daily data | Frankfurter was at effective date 2026-09-04 while direct CBR already returned 2026-09-05; transport may lag |
| FX-E06 | confirmed, [CoinGecko pricing](https://www.coingecko.com/en/api/pricing) | Demo: $0, 10k calls/month, 100/min, freshness from 60 sec, one key, attribution, one year daily/hourly history | Plan and limits can change; stable Demo usage needs an owner-created key |
| FX-E07 | confirmed, [simple price](https://docs.coingecko.com/reference/simple-price) and [history](https://docs.coingecko.com/reference/coins-id-history) | Current endpoint supports batching and `last_updated_at`; history returns market data for a specified date | Aggregated reference market price, not a specific platform's execution quote |
| FX-E08 | confirmed, live GET 2026-09-07 | `bitcoin`, `ethereum` and `tether` returned separate USD/RUB values and `last_updated_at`; `usd-coin` separately returned USD + `last_updated_at` and historical USD/RUB/BTC/ETH legs | Keyless public endpoint tested; Demo-key flow not tested; USDC observation does not establish USDC.E identity |
| FX-E09 | confirmed, live boundary probes | A 365-day BTC range returned daily points; an earlier date/366 days returned 401 plan-limit | Free CoinGecko contract beyond 365 days is unproven |
| FX-E10 | confirmed, [TradingView datafeed docs](https://www.tradingview.com/charting-library-docs/latest/connecting_data/) and [libraries](https://www.tradingview.com/free-charting-libraries/) | Advanced Charts/Trading Platform provide no market data; Lightweight Charts is visualization; widget data stays in an iframe display | A visible chart/symbol is neither an API nor a server-valuation permission |
| FX-E11 | confirmed, [TradingView Terms §3](https://www.tradingview.com/policies/) | Market data is licensed for display only; non-display processing and automated price referencing are prohibited | TradingView rejected as a Want Keep source without a separate written data agreement |
| FX-E12 | confirmed, [Coin Metrics Community](https://gitbook-docs.coinmetrics.io/packages/coin-metrics-community-data) and live catalog | No-key community access; the ReferenceRateUSD community contract/catalog is limited to the latest seven observations | May be a short current cross-check after license recheck; cannot close history |
| FX-E13 | confirmed, [Kraken OHLCVT downloads](https://support.kraken.com/hc/en-us/articles/360047124832-downloadable-historical-market-data-time-and-sales-) and [OHLC API](https://docs.kraken.com/api/docs/rest-api/get-ohlc-data/) | Free archives advertise candles/trades; REST OHLC is limited to the latest 720 entries | Required archive pairs and permission for a household service were not verified; not selected |
| FX-E14 | confirmed, [Coinbase Market Data Terms](https://www.coinbase.com/legal/market_data) | A public market API exists, but terms restrict multi-party/redistribution and AI use | Not selected for a household AI application without separate consent |
| FX-E15 | confirmed, [Alpha Vantage limits](https://www.alphavantage.co/support/#api-key) | Free tier is limited to 25 requests/day | USDT, complete depth and shared-service rights remain unverified; not selected |
| FX-E16 | confirmed, [Globalping HTTP probes](https://globalping.io/docs/api.globalping.io) 2026-09-07 | CBR and CoinGecko: HTTP 200 from DE/NL/BG. Frankfurter: DE/BG 200, one NL DNS failure; retry across three NL networks: 3×200 | Measurement IDs `2fsM1iXZRutVZQVSb000215ZM`, `2qCuOHSFPrYPQ99xT000215ZM`, `2ZqixcPO00hNbgB50000215ZM`, `2XSpFloX7zMGNRaNO000215ZN`; snapshot is not uptime/SLA |
| FX-E17 | inference, usage calculation | One hourly CoinGecko batch is about 744 calls over 31 days; at most 124 asset/day history calls for four assets over a 31-day backfill. Both are far below 10k before retries/manual ranges | Implementation must meter actual calls, cap retries and not present this calculation as runtime proof |
| FX-E18 | confirmed gap, provider research | No selected reference source supplies executable buy/sell price with account-specific amount, spread and fee | Only a verified provider contract can supply these; absence means unavailable/unknown |

## TradingView hypothesis result

TradingView is **not suitable** as a server-side rate source for Want Keep. Charting libraries require an external datafeed, Lightweight Charts only renders data, and widgets provide no extraction contract. Public terms separately prohibit automated price referencing and other non-display processing. Therefore no valuation adapter, scraping, webhook or hidden widget bridge is designed. Open-source Lightweight Charts may be evaluated later for UI if a screen needs it; that is outside the rate contract and does not add a dependency in this task.

## Failure, cache and audit

- Retain raw responses as protected source evidence or a content hash; public examples contain no cookies, IP, key or actual financial data.
- Cache keys include provider, provider asset ID, base/quote, observation/effective date and granularity. Current and historical namespaces do not mix.
- 401/403 is a plan/auth/configuration error; 429 is rate-limited with backoff; timeout/5xx is source unavailable. Reconcile an unknown outcome before repeating the idempotent GET.
- The last value may be displayed as stale with source/effective/fetched dates. It does not become a new current observation.
- A cross-check mismatch retains both observations and raises diagnostics. Never silently choose the convenient value or average CBR and market data.
- CoinGecko keys and future provider credentials stay outside Git and are never sent to AI.

## Blockers and handoff

| ID | Status | Required decision and closure owner |
| --- | --- | --- |
| FX-B01 | CLOSED | Current and up-to-365-day history for RUB/USD/BTC/ETH/USDT/USDC are covered by the selected reference contract and cross-rate formula; USDC.E needs separate identity/rate mapping |
| FX-B02 | OPEN | task-0.10: accept `valuation_unavailable` for crypto older than 365 days or separately verify an allowed archive/paid source. Current prices/pegs are forbidden substitutes |
| FX-B03 | OPEN | task-0.1–task-0.6/task-0.10: obtain provider-specific executable buy/sell, amount, fee/spread and timestamp or approve UI `quote unavailable`; reference price cannot replace them |
| FX-B04 | OPEN | owner + task-0.10: create a free CoinGecko Demo key, verify keyed endpoints/usage endpoint and record attribution. Never publish the secret |
| FX-B05 | CLOSED FOR RESEARCH | DE/NL/BG reachability of selected endpoints was established by dated probes; task-0.9 still verifies actual VPS/runtime |
| FX-B06 | CLOSED | TradingView hypothesis tested and rejected for non-display valuation |

BLK-07 retains FX-B02–FX-B04 until task-0.10 decides them. task-0.7 completes research and unblocks contract formalization, while task-6.1 and the MVP remain **Not Ready** pending the common Ready gate.

## Verification

Performed: official documentation and terms review, anonymous endpoint probes, current/history/limit boundaries, separate USDT/USDC observations, CBR/Frankfurter effective-date mismatch, DE/NL/BG reachability and RU/EN contract review. Only public data was used; live rates are not published.

Documentation commands: `make docs-check`, `python3 -m unittest discover -s spec/001-want-keep-mvp/tools -p 'test_*.py'`, `git diff --check`. They validate SDD artifacts, not financial runtime.

Not performed: CoinGecko Demo key flow/usage readback, long soak/SLA, actual VPS, provider buy/sell/fees, crypto history older than 365 days, application/collector/database or complete AC-037/AC-038/AC-039/AC-074. The application, secrets and provider contracts are absent. FX-B02–FX-B04 record these limits instead of presenting them as implemented success.
