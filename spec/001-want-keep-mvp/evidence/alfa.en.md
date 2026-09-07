# Alfa-Bank: read-contract research

[Русский](alfa.md)

Date: 2026-09-07. Task: [task-0.1 / Issue #1](https://github.com/pchkauu/want-keep/issues/1). Research is complete: the product matrix, portal observations, published API contracts and synthetic examples were collected. Automatic import is not implemented. The original ALFA-B02–B06 were handed to the Ready gate; their final split into SDD decisions and runtime gates is recorded in the task-0.10 section below.

## Scope and evidence

This research concerns an individual banking portal. D-37 defines mandatory REQ-042 coverage as debit card, current/savings accounts, deposits and cashback. The owner has no Alfa credit card; opening one for testing is not required, while shared manual credit-card accounting remains.

| ID / source | Observation | Evidence boundary |
| --- | --- | --- |
| ALFA-E01: [Issue #1](https://github.com/pchkauu/want-keep/issues/1), initial GitHub readback | OPEN at research start; task-0.1, its criteria and boundaries were read | Research scope; final delivery status is checked separately in GitHub |
| ALFA-E02: owner message | Debit card, deposit and cashback present; no credit card | Declared products; supplemented by live reading below |
| ALFA-E03: browser discovery | The owner identified the initial session as Arc; a different Chrome profile was then selected. Discovery subsequently located the specified Google Chrome profile and dashboard tab | The earlier missing-tab conclusion is superseded. Profile files and session stores were not read |
| ALFA-E04: [public Alfa API page](https://alfabank.ru/corporate/guide-alfaapi/) | Initially only URL/title; opening in Chrome returned ERR_NAME_NOT_RESOLVED | Initial text-retrieval limitation. Published retail contracts were subsequently found: ALFA-E14–ALFA-E20 |
| ALFA-E05: owner screenshot | Signed-in Google Chrome dashboard with card, current account and savings | An image is not an API contract; it remains in the conversation only |
| ALFA-E06: live dashboard DOM | Total balance, current account with card alias, savings account and three deposits; RUB, deposit rates and deadlines | Reading the overview in the current session is confirmed |
| ALFA-E07: opening the existing current account | The next read was stopped by automatic review at private.auth.alfabank.ru. The owner entered the code themselves; reading resumed at web.alfabank.ru after their message | Owner authentication handoff was observed; redirect cause and session lifetime remain unknown. The agent did not receive the code |
| ALFA-E08: current account and purchase | Available amount, card alias, history and filters; purchase amount/time, MCC, debit card, cashback and processing status; Show more added 20 rows | Partial reading; no raw enums, complete ledger or pagination contract |
| ALFA-E09: deposits | Maximum and one New Money deposit opened: overview, About, schedule. Maximum history and one interest payment read twice with identical details | Two terms samples and one repeat read; the second New Money deposit was read in the overview only |
| ALFA-E10: savings account | Overview, About, history, internal-transfer and separate cashback-credit details | Existing movements shown; full identities of both sides and raw statuses unproven |
| ALFA-E11: benefits and cashback | Balance/future accruals, monthly overview, history and an accrual detail with purchase date/amount, category, card mask and future payout date | UI linkage without stable original-purchase or payout-batch identity |
| ALFA-E12: history and repeated navigation | Selected 01.08.2026–31.08.2026; August groups and Show more appeared. Returning to the current account produced the same URL | One interval and route identity within one session; history continuity and identity across reauth unproven |
| ALFA-E13: purchase-row DOM attributes | The button has role=button and data-test-id=operation-cell; its two nearest containers have no attributes except styling | No unique operation ID in this inspected fragment. This is not a complete DOM or network-response analysis |

All observations are dated 2026-09-07. Git retains no personal amounts, names, card/account numbers, contracts, raw DOM, screenshots, cookies, codes, tokens or browser IDs. Product/field names remain; examples below are entirely synthetic. Evidence consists of visible UI observations, not saved network responses.

## Live product-reading matrix

| Product | Confirmed | Unverified / limitation |
| --- | --- | --- |
| Debit card | Owner confirmed type; card alias accompanies current account; purchase detail shows debit card | Card status/replacement, full card-to-account identity, holds and separate limits; alias is not a permanent ID or second balance |
| Current account | Label, RUB, available amount with minor units, account mask, card alias, history and route | Separate owned/locked/debt and sourceAsOf; fees; stability after fresh sign-in; full-ledger reconciliation |
| Savings account | Daily-balance Alfa Account, RUB, zero available balance, lifetime income, rate/welcome uplift, forecast, opening date; history of interest, transfers and cashback payment | Rate/uplift history and eligibility, exact day-count/rounding, full history and structured machine-readable terms |
| Maximum deposit | Balance, principal, opening/closing dates, term, annual rate, compounding, return/payout account, restrictions, renewal, received/future income; schedule and history | Early-closure rules, day-count/rounding, term updates, structured accrual identity/status |
| New Money deposit | One of two deposits: the same field groups, with renewal date and automatic renewal enabled | Second instance: overview only. Complete terms/log contract unconfirmed; equal names do not merge deposits |
| Credit card | Absent according to owner | Deferred by D-37; does not block Alfa. Shared manual credit-card accounting remains |
| Cashback | Ruble balance, expected monthly accruals, accrual-type history, displayed purchase linkage and separate bank credit | Accrual→payout→correction, stable IDs, allocation of payouts to purchases, caps/expiry, net/gross and rule history |
| Currencies and exchange | RUB for observed accounts/deposits; RUR in cashback-filter query | Other currencies, buy/sell quotes, exchange rate and fee not live-tested; REQ-039 requires explicit unavailable |

Exact minor-unit addition of five displayed balances matched the dashboard total. This validates one snapshot, not complete asset coverage, balance semantics or ledger reconciliation.

## Routes and read contract

These are **UI routes**, not HTTP API endpoints. Portal network methods/schemas and raw responses were not obtained; the published external API is assessed separately below. The documented Chrome tool exposes DOM/actions, not network request/response interception; hidden application state was not extracted.

| UI / action | Observed result | Limitation |
| --- | --- | --- |
| /dashboard, /dashboard/ link | Products and total balance | Total balance is not spendable household money |
| Current/savings card → /accounts/{account-reference} | Balance, history, account mask, actions/information | Current-account reference is a 20-digit URL value: a private identifier, not published; requisites were not opened separately |
| Deposit card → /deposits/{deposit-reference} | Balance, deadlines, income, schedule/history | Private reference does not prove identity across sessions |
| Transaction row → dialog | Exact amount, type/description, date/time, available details and sometimes explicit status | Tested interest-payment URL remained the deposit route; no standalone operation ID established in the dialog |
| /history/ → calendar | Selected range and history | Date selection did not add a query to this UI URL; the URL alone cannot reproduce the filter |
| /marketplace/ → expected accrual → /profit/cashback | Balance, pending amount, monthly summary; loyaltyType/month query | Query values are not banking transaction IDs |
| All history → /partner-offers/accruals-history | Accrual types, currencies, dates and cashback rows; currencies/groups/limit/from/to query | UI limit does not prove server page size; RUR needs explicit boundary mapping to RUB after schema verification |
| Cashback row → dialog | Future accrual date, purchase date/amount, category and card mask | No proven common identity with banking purchase/payout |

SourceRecord must follow the [shared contract](../contracts.en.md): household/provider/stable external account/product/log namespace, source ID or evidenced identity strategy, revision and provenance. ConnectionId, card mask, label/amount/time and row order do not prove uniqueness. The data-test-id=operation-cell attribute in ALFA-E13 identifies a UI component, not a unique record. Until identity is resolved, repeated DOM reading must not be presented as import idempotency.

## Published retail API: separate evidence layer

ALFA-E14–ALFA-E20 were obtained on 2026-09-07 from indexed official bank pages (search-source snapshots aged 2–5 months). Direct retrieval of several pages returned timeout/502. These are published documents, **not live API responses, not web.alfabank.ru network contracts, and not proof of current Want Keep access**. The failed earlier text retrieval in ALFA-E04 does not imply that retail APIs are absent.

| ID / source | Confirmed published contract | Open boundary |
| --- | --- | --- |
| ALFA-E14: [connection](https://developers.alfabank.ru/products/alfa-api/documentation/articles/connection/connection), [signing](https://developers.alfabank.ru/products/alfa-api/documentation/articles/connection/articles/onboarding/onboarding) | Agreement through Alfa-Business; test integration, then production with TLS certificate and credentials. Instructions state no API usage/connection fee. | This does not establish private-project eligibility, free required banking services or access to all retail scopes. A personal portal session alone does not grant API access. |
| ALFA-E15: [accounts](https://developers.alfabank.ru/products/alfa-api/documentation/articles/accounts/articles/list/v1/list), [account](https://developers.alfabank.ru/products/alfa-api/documentation/articles/accounts/articles/details/v1/details) | GET /pp/v1/accounts and /pp/v1/accounts/{accountNumber}; accounts scope. String number, type, ACTIVE/INACTIVE status, dateCreated; balance includes currency, minorUnits, holds, amount. | Full savings/deposit coverage, amount versus holds semantics and identity across reauth need a live check. |
| ALFA-E16: [cards](https://developers.alfabank.ru/products/alfa-api/documentation/articles/cards/articles/list/v1/list) | GET /pp/v1/cards; cards scope. cardId, maskedNumber, account.number, isCredit, status/state and dates. | isCredit does not establish debt or grace. cardFilter is typed boolean but described with named options: clarify the contract. |
| ALFA-E17: [history](https://developers.alfabank.ru/products/alfa-api/documentation/articles/operations-history/articles/operations-list/v1/operations-list) | GET /pp/v1/operations; operations-history scope; inclusive YYYY-MM-DD dateFrom/dateTo; accounts/cards, operationDirection; offset ≥ 0 and multiple of limit > 0, defaults 0/20. operations entries: id, dateTime, amount(value/currency/minorUnits), direction, fee, reference, status. | status: SUCCESS/HOLD/FAILED or null/unknown. INCOME/EXPENSE describe direction. Retention, ordering, revisions, snapshot consistency, max limit, timezone and fee units remain unverified. |
| ALFA-E18: [operation details](https://developers.alfabank.ru/products/alfa-api/documentation/articles/operations-history/articles/operation-details/v1/operation-details) | GET /pp/v1/operations/{id}; operations-history scope; ID from list. Additional sender/recipient, mcc, loyalty, terminal, isAnotherClient, cashout. | cURL duplicates /api/api whereas the endpoint heading has one /api. Do not treat the example as tested or derive financial ownership from isAnotherClient. |
| ALFA-E19: [bonus accounts](https://developers.alfabank.ru/products/alfa-api/documentation/articles/loyalty/loyalty), [accruals](https://developers.alfabank.ru/products/alfa-api/documentation/articles/loyalty/articles/transactions/v1/transactions) | loyalty scope. GET /pp/v1/bonus-accounts returns loyaltyAccounts, accountNumber/typeId, optional balanceAmount. POST /pp/v1/bonus-accounts-transactions reads history using references/startDate/endDate; transactions include optional reference and cashbackAmount. | cURL uses bonus-account-transactions, differing from the bonus-accounts-transactions heading. Cash payout, its status, reversals and unique purchase linkage remain unproven. |
| ALFA-E20: [statement](https://developers.alfabank.ru/products/alfa-api/documentation/articles/operations-history/articles/account-report/v1/account-report), [method registry](https://developers.alfabank.ru/release-notes) | POST /pp/v1/reports/type/account, debit-statements scope, orders a document using dateBegin/dateEnd/accountIds. The registry lists credit/statement, deposit, GET reports and reports/{reportId}/body, cards/{cardId}/tariffs and /rates/gd/v1/offices-rates. | An order is not a finished statement. A listed method does not prove structured savings/grace terms or a personalised executable quote. Corporate /jp APIs do not replace /pp. |

The documented /pp methods above use Bearer through Authorization Code Flow and Accept: application/json; API base is https://baas.alfabank.ru/api, sandbox is https://sandbox.alfabank.ru/api. No API requests were made; bank browser tokens were neither extracted nor reused for API access.

### Synthetic API examples and mapping

All values below are invented. These are minimal examples using the published shape, **not successful requests/responses or complete schema fixtures**; account-example is a placeholder that must not be sent to the bank.

~~~http
GET /api/pp/v1/accounts/account-example
Accept: application/json
~~~

~~~json
{"mnemonic":"Example","number":"account-example","type":"EE","status":"ACTIVE","dateCreated":"2026-01-01","balance":{"currency":"RUR","minorUnits":100,"holds":2500,"amount":125000}}
~~~

~~~http
GET /api/pp/v1/operations/operation-example
Accept: application/json
~~~

~~~json
{"id":"operation-example","dateTime":"2026-08-15T09:30:00Z","title":"Example shop","amount":{"value":125040,"currency":"RUR","minorUnits":100},"direction":"EXPENSE","status":"HOLD","fee":0,"isAnotherClient":false,"cashout":false}
~~~

Authorization is intentionally omitted. The proposed Want Keep mapping preserves string provider IDs without float conversion; amount.value=125040 with a confirmed scale of 100 becomes 1250.40 RUB. RUR→RUB is an explicit boundary mapping that retains the original code. Do not subtract holds again or add fee until semantics are established. Null/unknown status retains uncertainty. INCOME does not turn internal transfers into income; HOLD does not establish final posting. These are future mapping constraints, not implemented code.

### Automation and recovery decision

Prefer the official API **after** access to required products is confirmed. The browser alternative has evidence of interactive reads only; its background Playwright collector contract remains open. Scraping amount/name/time strings does not establish stable financial-ledger identity.

The future allowlist must enumerate origin, HTTP method, path, query/body and response for each read. A scope does not replace that allowlist. Exclude card actions, payments, limit changes, product opening/closure and cashback issuance. Loyalty-history POST is assessed separately from financial-action POST; document orders require separate checks for cost, side effects and unknown outcomes before retries.

The task-4.1 test plan after access approval: fixed interval/page size; persist each page before advancing offset; overlap/replay and reread changed transactions; compare to source and resume after a second-page failure. Offset pagination does not guarantee completeness when earlier rows change. Do not claim full coverage before verifying that behaviour. A 401 waits for owner access recovery; 403 exposes unavailable scope; 429/5xx use bounded backoff retries. Numeric quotas and refresh/session lifetime remain unknown; hourly sync is a Want Keep requirement, not a bank guarantee.

### Required owner and bank input

To resolve ALFA-B02, the bank must confirm personal-pilot eligibility without mandatory paid external sources, available scopes/products, limits and test/production access provisioning. The owner supplies access securely outside Git/chat. A browser collector needs an authorised way to obtain structured read contracts and identifiers, not session export in conversation. This research did not send bank messages or sign agreements.

ALFA-B03 needs a second independent external account and entity comparison across reauth; ALFA-B04 needs an authorised credit-product sample or explicit owner scope decision. ALFA-B05/B06 require clarified terms, fee/status and accrual–payout–correction linkage. Synthetic responses and absence of a product from the current owner's account cannot resolve these items.

## Money and terms semantics

- A purchase shows processing text; an internal transfer shows completed text. The inspected interest payment has no explicit status. Raw enum and occurredAt/postedAt/timezone are unverified; absence of pending text does not mean posted. A UI status is evidence, but mapping requires a verified contract.
- Deposits also label balances Available although both inspected samples disallow top-ups and partial withdrawals. Such a balance must not automatically enter the spending pool; term/lock and withdrawal conditions belong to the financial domain.
- Deposits show received income, next payment, whole-term income and a schedule spanning past/future dates. History separately shows payments and opening funding. Opening funding is capital movement; a payment and its schedule row do not produce two incomes. Closing date and Renewal date differ; automatic renewal was respectively off/on in the inspected samples.
- About deposit structures principal, term, dates, annual rate, compounding, return/payout account and restrictions. Text about payment into the deposit and the return account does not define the entire accrual/redemption algorithm. Day-count, rounding, early closure, term history and exact meaning of every rate are still required.
- About savings contains a FAQ describing opening-of-day balances, month-end payment, possible rate/uplift changes and no interest for a month with early closure. This is source text, not an executable contract. Personal eligibility, complete rate history, calculation basis and rounding are unverified; AI must not turn the FAQ into a guaranteed model.
- The bank excludes the internal transfer from its analytics. Its detail shows the debit account but did not establish the credit account. Want Keep verifies ownership and both sides independently; a name or bank category does not prove household-pool membership.
- Purchase cashback and accrual-history entries may await payment. A separate credit was found in savings-account history. Expected accrual does not increase the available budget; payment is accounted for once after status and accrual linkage are established. Cashback/reverse-compensation classification needs an explicit domain-contract decision: its label does not make every credit ordinary income or a refund.
- Separate fee fields were not established in the inspected details; absence is not zero. Analytics-category display precision may differ from the main amount; Money requires the exact transaction amount.

## History, repeats and failures

The current account initially showed 20 history rows; one Show more added another 20 and remained available. This is a UI observation, not proof of constant page size, cursor, full coverage or end marker. Selecting 01.08.2026–31.08.2026 in overall history produced August transactions; not all interval pages were retrieved. Retention depth and boundary inclusivity remain unknown. Older savings records establish the existence of history, not its continuity.

Reopening one interest payment produced identical dialog text and the same deposit route. Returning to the current account from other sections produced its earlier URL. These are same-session checks, requiring no simulated bank transaction. Rapid purchase-dialog closing/reopening did not yield a verified repeat and is not counted as a passing test. Two click timeouts on Benefits and a cashback row were followed by snapshots showing completed navigation; no repeat click was issued.

The owner entered the code after the auth-origin stop. No pre-auth identity baseline had been collected, so identity across reauth is unproven. A second household member's external account, session expiry, hourly runs, late revisions, refunds, resume after network failure and stale-job rejection after disconnect were not tested.

## Synthetic examples

These are **UI-read results** based on observed field labels. All values are invented; these are neither Alfa HTTP requests/responses nor accepted provider fixtures. Source UI labels remain in Russian.

~~~text
Action: open an existing purchase row.
Dialog: −1 250,40 ₽; Магазин-пример; 6 сентября, 12:34;
        Операция в обработке; Карта списания: Основная ****1234;
        Код торговой точки MCC 5411; Кэшбэк / Рубли: +12 ₽.
Unknown: source operation ID, raw status, postedAt, timezone, fee.

Action: open About deposit.
Fields: Начальная сумма 10 000,00 ₽; Срок 92 дня;
        Дата открытия 01.06.2026; Дата закрытия 01.09.2026;
        Ваша ставка 10.00% годовых; Капитализация Да;
        Пополнение Нет; Частичное снятие Нет; Автопродление Выключено.
These fields alone do not establish the income formula.

Action: open a cashback accrual row.
Dialog: +12 ₽; Магазин-пример; Категории на месяц;
        Начислим 10 сентября; Дата покупки 25 августа 2026;
        Сумма покупки 1 250,40 ₽; Категория Продукты; Карта списания *1234.
The navigation is not a bank payout.
~~~

## Blockers, traceability and continuation

| Blocker | Status / closure evidence | REQ / AC |
| --- | --- | --- |
| ALFA-B01: correct browser and viewing access | Closed for this run: ALFA-E03/E06/E07; a future session may need the owner again | REQ-073; observation for AC-087, not a complete test |
| ALFA-B02: automation method and allowed read contract | RUNTIME GATE task-4.1 — retail API exists (ALFA-E14–ALFA-E20), while Want Keep eligibility/provisioning is unconfirmed; verify exact read allowlist, limits, MFA handoff and documented inconsistencies. Browser access remains an alternative until structured identity is proven | REQ-042, REQ-048; AC-042, AC-048 |
| ALFA-B03: source identity and history | RUNTIME GATE task-4.1 — account/card/operation IDs and offset/limit are published; runtime verifies stability, revisions, retention/resume/coverage, time/fee semantics, reauthentication and an independent second-member account | REQ-041, REQ-065, REQ-073; AC-041, AC-079, AC-087 |
| ALFA-B04: missing credit card | CLOSED D-37 — the Alfa credit card is deferred; shared manual credit-card behavior remains | REQ-031, REQ-032, REQ-042; AC-042, AC-070 |
| ALFA-B05: exact savings terms | RUNTIME GATE task-4.1 — verify structured rate schedule/day-count/rounding/cash flows, accrual statuses, early closure and updates for selected savings products | REQ-033; AC-070 |
| ALFA-B06: FX and cashback lifecycle | RUNTIME GATE task-4.1 — verify quote capabilities and cashback accrual/payout/correction/purchase linkage without duplication | REQ-039, REQ-042; AC-042 |

Research handed ALFA-B02/B03 and remaining runtime questions to task-4.1. Structured samples are obtained only under owner-controlled authentication; cookies/tokens are never exported. D-37 closes scope and D-39 closes safe identity/unknown/collision behavior; see the task-0.10 section below.

task-0.1 is complete as research. task-0.10 accepted D-37/D-39 and issued SDD Ready; task-4.1 must still implement and prove collector/API runtime. GitHub Closed for research is not integration acceptance.

Document checks on 2026-09-07: 13 generator tests (unittest discover) — pass; self review clarified the UI/API evidence distinction and BLK-01 closure owner; `python3 spec/001-want-keep-mvp/tools/spec_tool.py check` — pass; `git diff --check` — pass; RU/EN REQ/AC/task/ALFA-ID sets match and local links resolve. The pair was also checked for long numeric identifiers, private paths and tested secret patterns. These checks do not replace live acceptance criteria or future application-security tests.

## task-0.10 decision, 2026-09-07

ALFA-B02–B06 above remain research-evidence boundaries but no longer form a global SDD blocker. D-37 removes the unavailable Alfa credit card from current provider scope while preserving shared manual credit-card support. D-39 defines identity/unknown/collision behavior without invented fields.

The signed-in Chrome tab additionally confirmed debit/current/savings/deposit/cashback/history sections and stable UI category markers without reading or publishing amounts. DOM/UI is not a financial schema. Before deployment enablement, task-4.1 obtains an authorized structured fixture, allowlist, stable IDs, pagination/coverage, revisions/statuses/fees/cashback lifecycle, reauthentication, two accounts and target-host route. A failed gate keeps Alfa disabled; the SDD remains Ready for development.
