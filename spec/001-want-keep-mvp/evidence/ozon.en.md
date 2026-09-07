# Ozon Bank: read-contract research

[Русский](ozon.md)

Research date: 2026-09-07. Task: [task-0.3 / Issue #3](https://github.com/pchkauu/want-keep/issues/3). Status: **research completed with blocking findings; structured responses obtained**. BLK-03 remains open for task-0.10 verification; task-4.3 and the shared Ready gate remain blocked. The HAR section below is current; subsequent UI observations describe the preceding stage.

## Structured contract from HAR

OZON-E17 is the owner-supplied recording: 291 requests and 290 response bodies. OZON-E18 is the additional pagination recording: 28 requests. All recorded HTTP responses have status 200; this does not test 401/403/429 handling. Originals remain local outside Git. Their SHA-256 hashes, request indices and [response projections with synthetic values](ozon.samples.json) establish sample provenance. Published JSON retains selected fields only; these are neither full responses nor authentication data.

### Read routes

All five routes were observed on `https://finance.ozon.ru`, using **POST** with JSON requests/responses. This is an internal web-portal interface, not an established public retail API. Publishing a route neither authorizes payments nor proves autonomous server access.

| Route | Request | Response fields and sample |
| --- | --- | --- |
| `/apps/pfm/api/operations/modalAccounts` | `{}` | `data.me.client.accounts[]`: accountNumber, accountToken, balance, productTypeV2/V3, status; `mainAccount.accountToken`. OZON-S01 |
| `/api/v4/cards/all` | `{}` | `data.me.client.cards[]`: accountNumberV2, cardId, cardMode, status, product. OZON-S02 |
| `/api/v4/userBalance` | `{}` | `data.ozonUser.balance.cents`. OZON-S03 |
| `/apps/pfm/api/operations/groupOperationsV3` | `filter`, `cursorPagination: {cursor, perPage}` | `data.me.client.groupOperationsV3`: items, cursors.next/prev. OZON-S07–S10 |
| `/apps/pfm/api/operations/operationById` | `{id, timeZone}`; id matches lastOperationId | `data.me.client.groupOperationsV3.items[]`; details additionally contain accountNumber and direction. OZON-S04–S06 |

Filter structure: `effect="EFFECT_UNKNOWN"`, `coopAccountIDs=[]`, `accountTokens=[]`, `pfmTags=[]`, `timeZone="Europe/Moscow"`. The historical request additionally sent `dateRange={from:"2026-08-01",to:"2026-08-31"}`. Nonempty account/category filters were not tested. Observed `perPage=30` is not an established API maximum.

Only these verified read operations are candidates for the financial allowlist. Do not permit all of `/api/`, all POSTs or every page request: the recording includes background `/apps/promo/api/banners/setAction`, cache warming, contexts, advertising and other unrelated routes. UI navigation does not prove future collector allowlist enforcement. Cookies/session and opaque references remain inside the protected adapter and never reach AI.

### Identity and exact amounts

| Field | Established | Adapter rule |
| --- | --- | --- |
| accountNumber / accountNumberV2 | modalAccounts number matched the card and three transaction details; UI number matched after a new login (OZON-E16) | Link the card to its account without creating a second balance. Final namespace includes provider and verified account identity, not connectionId or the action author |
| accountToken | Five different values for the same verified account number within one HAR | Opaque context reference, not a stable ID; exclude from the financial fingerprint. Never decode or publish its original value. Resolve a list row through details when no other verified account mapping exists |
| lastOperationId | 210 distinct values across seven OZON-E18 pages; the details request uses this ID | Candidate source ID within the verified account namespace. Repeated ID does not create a second fact; financial-field changes require a revision. ID changes during future lifecycle transitions still require reconciliation |
| groupID | 209 distinct groupIDs in 210 rows: transfer and commission share groupID but have different lastOperationId | Never use groupID as the sole deduplication key. Retain it for linking and investigating revisions |
| accountAmount / originalAmount | `{amountAbs:{cents:integer,currencyName:"RUR"},sign:"POSITIVE"\|"NEGATIVE"}` | Map RUR → RUB at the boundary; integer kopecks → exact decimal amount /100 with a separate sign. No floating point or UI aggregates |
| balance | modalAccounts.balance matched userBalance.cents in the observed snapshot; no separate owned/available/locked fields | Retain the reported balance and provenance. Do not classify its full amount as available for daily limits or substitute zero for an absent hold |
| time / lastOperationTime | Timestamp strings with `Z`; request supplies Europe/Moscow | Preserve source time and separate fetchedAt. Do not infer cash/booking/value-date semantics from a field name; retain limitations where calendar meaning is unverified |

Account-number equality across two logins was verified for one owner. A second independent external account was not connected. Two-owner isolation, lease/version and importer recovery remain mandatory checks; one HAR does not establish them.

### Statuses, refunds and commissions

- OZON-E18 contains 206 `GROUP_OPERATION_CONFIRMED` rows and 4 `GROUP_OPERATION_CANCELED` rows. A canceled row may retain a negative amount; never count it as a confirmed expense. Unknown status does not become posted.
- `groupOperationType="OZON"` appears for both purchases and refunds. `meta.__typename` distinguishes `OzonClearingOperationMeta` from `OzonRefundOperationMeta`; sign or merchant name alone does not establish economic type.
- `AUTHORIZATION` appears with both `ClearingOperationMeta` and `AuthorizationOperationMeta`. Never classify lifecycle from groupOperationType alone. Unknown status/type/meta combinations require explicit review rather than automatic posting.
- In OZON-S10, `TRANSFER_BY_CARD_NUMBER_COMMISSION` has its own lastOperationId and a parentOperationId equal to the linked `TRANSFER_BY_CARD_NUMBER_OUTGOING` lastOperationId. Both amounts are negative and the commission has `meta=null`, an observed valid variant. Retain both records separately; do not add the commission again from prose/calculation. Net/gross generalization to other commission types is unverified.
- The inspected Ozon refund has `parentOperationId=null`. A name or shared type does not establish its original purchase. The existing Want Keep receipt/purchase matching or user clarification flow resolves the link; only a confirmed link adjusts the original month.
- `INTERNAL_TRANSFER_*` is a provider type name, not proof of movement between the family's own accounts. A household transfer requires verified identity of both sides.

### Pagination and evidence limits

OZON-E18 contains seven consecutive pages of 30 rows. Each next request.cursor exactly matched the previous response.cursors.next, with the filter retained. No groupID was repeated across pages, but one page contains the transfer/commission pair described above. All 210 lastOperationIds differ. Every response has a nonempty next: **history completion was not reached**. A HAR ending without further requests is not an end-of-data signal.

In OZON-E17, three repeated first pages had identical checked financial fields and IDs. All 30 rows in the August request fall within the requested Europe/Moscow dates, including a last-day record. The first day, interval completion, retention and recovery after errors remain unverified. The earliest displayed day does not establish a retention contract.

The collector must advance its cursor only after durable page storage, replay an interrupted page without a second effect and retain coverage with filter/bounds. These are future implementation requirements, not passing runtime tests. A separate HAR reaching the end of the August interval has been requested.

### Session and outstanding questions

The UI requested sign-in (OZON-E15), and the owner restored access (OZON-E16). HAR also contains `POST /api/v4/auth/check` → `data.authV2.bankAuth.tokenExpiresAfter`. Its change corresponds to roughly 1000 units per second across two observed pairs; milliseconds are an inference from measurement, not a published guarantee. Full session lifetime, supported refresh and autonomous hourly operation remain unverified. Never invent a refresh endpoint or bypass owner sign-in.

Direct HTTP replay by a separate collector, server frequency limits, authentication errors, two external accounts and late revisions were not tested. This HAR has no original authorization headers; it establishes read shapes without providing a portable bank session.

**Current outcome:** task-0.3 research output is prepared: actual routes/fields, 10 synthetic projections and an RU/EN matrix of established facts and limitations. The [README](../README.en.md) allows research to finish with a documented blocker. Issue #3 completion means this report is complete, not that the automatic integration is ready. OZON-B02/B04/B05 remain within BLK-03 for task-0.10; the complete MVP remains Not Ready.

## Scope and preceding observations

Under D-32, the current contract covers the **debit card and linked main RUB account**. Observed codes are account `OPENED`, product `EDS_FULL_UPGRADE`, card `ACTIVE`, `DEBIT`, `MIR_VIRTUAL`. These are observations, not a universal product list or legal classification. The card creates no second monetary balance. Ozon credit cards, savings and deposits are future extensions; their absence does not block current scope.

The table retains OZON-E01–OZON-E16 provenance. “HAR not received” describes that observation's point in time; OZON-E17/E18 results above supersede it. Public offers are not the owner's individual terms; a search result does not prove the absence of a retail API.

| Source | Obtained | What it establishes and does not establish |
| --- | --- | --- |
| OZON-E01: Issue #3, direct GitHub read, 2026-09-07 | Complete RU/EN card; OPEN, no comments | Research scope, criteria and dependencies |
| OZON-E02: owner-supplied screenshot in this interview, 2026-09-07 | `finance.ozon.ru/lk`: main account, card alias, recent transactions, “All transactions” navigation and a credit-card offer | Visible banking data in the supplied image; not agent live access, an API contract or complete history |
| OZON-E03: Chrome connection, 2026-09-07 | Intended profile initially unavailable; after the owner connected the extension, the specified profile and `/lk` tab were found | Browser access obstacle resolved; OZON-B01 closed |
| OZON-E04: [bank website](https://finance.ozon.ru/) | Indexed official-site content; direct retrieval by the research tool returned HTTP 403 | Public descriptions of personal banking products; not current authenticated portal contents |
| OZON-E05: [savings account](https://finance.ozon.ru/promo/savings/landing) | Indexed official description: RUB, minimum balance during the calculation period, monthly payout | A publicly described savings variant. Individual terms, exact calendar basis and API remain unverified |
| OZON-E06: [daily-income account](https://finance.ozon.ru/apps/landing/saving-daily) | Indexed official description: RUB and daily payout; direct retrieval returned HTTP 403 | A separately described savings variant; owner eligibility remains unverified |
| OZON-E07: [deposits](https://finance.ozon.ru/promo/deposit/landing) | Indexed official description of a term deposit; direct retrieval returned HTTP 403 | A public offering, not an existing owner deposit or its agreement |
| OZON-E08: [credit card](https://finance.ozon.ru/promo/cards/credit/bez-procentov) | Indexed official offering; direct retrieval returned HTTP 403 | Product existence; advertised grace does not define the owner's payment schedule |
| OZON-E09: live main-account and history reading, 2026-09-07 | Opened `/lk/account`, `/lk/operations`, purchase, top-up and refund details | UI can be read automatically in an existing session; does not establish network API suitability or background import |
| OZON-E10: live filters and repeated reading, 2026-09-07 | One main account in the filter; August 1–31 applied; the same purchase reopened | Repeated purchase URL and visible details matched; continuous complete history unverified |
| OZON-E11: owner clarification, 2026-09-07 | Only a debit card is open | No credit card, deposits or savings accounts in this account; the subsequent D-32 decision defers them to a future extension |
| OZON-E12: owner-authorized account-details check, 2026-09-07 | Read the labeled recipient account number; it matched on reopening and in the limits-section account link | One account's identity matched across UI views within a session; new login/second member untested. The value is not retained in Git |
| OZON-E13: owner scope decision, 2026-09-07 | Accept currently available product types; extend contracts if needed later | D-32: debit card/main account form current Ozon scope; missing other products is not a blocker |
| OZON-E14: repeated account DOM check, 2026-09-07 | Session accessible; `/lk` → `/lk/account` works. Exactly one `span[data-testid="account-balance-visible"]`; its only attribute is `data-testid`, with no descendant attributes, `data-amount` or `datetime` | The selector locates the displayed balance. The inspected node has no machine value, snapshot time or owned/available/locked breakdown; this does not establish balance semantics or a network contract |
| OZON-E15: walkthrough after the owner enabled Network, 2026-09-07 | Reopened account, account details, linked card, history, refund and the previous purchase. Returning to history eventually reached `/apps/auth/signin` | Hidden card details were not revealed. A transition to sign-in during reading was observed, but its cause, session TTL and HTTP code remain unknown. Sign-in was handed to the owner with recording stopped for secret entry; HAR not yet examined |
| OZON-E16: walkthrough after owner reauthorization, 2026-09-07 | Tab back at `/lk`; account number matched the earlier reading. The purchase opened at the same URL and showed a successful status. Top-up and refund read; August 1–31 applied again | UI identity of one account and one purchase checked across logins. This does not verify a second account namespace, importer idempotency or network pagination. The owner confirmed recording enabled; a local HAR has not yet been supplied |

## Safety and reproduction

Research is limited to reading. Payments, money transfers, card issuance/activation and product changes were not performed. The owner handles sign-in and MFA in Chrome. Reading account details to compare IDs was separately authorized; values are not published. The bank session has broader permissions than the future collector, so allowlist enforcement requires implementation verification.

The owner exports Network → **Save all [listed] as HAR (sanitized)** after normal browsing. As described in the [Chrome documentation](https://developer.chrome.com/docs/devtools/network/reference/#save-as-har), excluding standard sensitive headers does not remove every personal value from URLs and bodies. Original HARs remain in local `tmp/ozon-read-contract/`, with owner-only directory access; Git excludes `tmp/` and `*.har`. Never copy HARs, cookies or session contexts into the Issue.

Public OZON-S01–S10 retain selected source field names/types and relationships. Names, amounts, dates, numbers, IDs and opaque references are replaced with synthetic values; actual account-number formatting is deliberately not reproduced. Fields outside the projection are omitted. Samples support design and future contract tests, not bank authorization or requests.

## Current blockers and handoff

| ID | Status and evidence | Remaining work |
| --- | --- | --- |
| OZON-B01 | Closed: OZON-E03/E09, access to the named profile and live reading | None |
| OZON-B02 | Structural gap resolved: OZON-E17/E18, five read routes, OZON-S01–S10 | Verify an acceptable operational automation method and authenticated-browser operation; HAR does not establish server session portability |
| OZON-B03 | Removed by D-32 | Other Ozon products are future extensions, not blockers |
| OZON-B04 | Account number across logins, card mapping, repeated records and seven-page chain verified | Terminal page of the selected interval, retention and ID lifecycle; no independent second external account supplied |
| OZON-B05 | Integer kopecks/RUR, two statuses, purchase/refund types and a separate commission verified; sign-in redirect and owner recovery observed | Separate owned/available/locked and date semantics unestablished. Full TTL, supported refresh, frequency limits and autonomous hourly operation untested; uncertainty must not become an assumption |

The next external-contract step is a completed page chain for a selected interval, followed by an acceptable session-maintenance method. A second account is checked only with separately supplied access. Never create transactions or open products to manufacture test data.

Collection infrastructure and task-4.3 test allowlist enforcement, importer idempotency, conflicts/lease/version, retry after failure and process isolation. These runtime tests are not required from a nonexistent application in a research task; external-contract gaps remain within BLK-03. task-4.3 and task-0.10 remain blocked.

## Traceability and verification

| Criterion | Requirements | Evidence obtained | Unverified |
| --- | --- | --- | --- |
| AC-044 | REQ-044 | OZON-E09–E18: debit product, read routes and structured fields | Complete automatic contract and available funds |
| AC-041 | REQ-041 | OZON-E17/E18: filter, repeated records and seven linked pages | History completion and importer recovery after failure |
| AC-048 | REQ-048 | Five allowlist candidates and broader session permissions established | Future collector route enforcement |
| AC-079 | REQ-065 | OZON-E16–E18: identity across logins, card/account, operation ID and separate commission | Second external account, ID lifecycle and importer behavior |
| AC-087 | REQ-073 | OZON-E15/E16: sign-in redirect and owner recovery | Lease/version, stale results and application MFA routing |

Checks: `spec_tool.py check`, 13 documentation-tool tests, JSON/links, semantic RU/EN review, synthetic-value and cursor/commission relationship checks, `git diff --check`. No complete application AC is claimed as passed. Exact results and publication are recorded in the [readiness report](../verification.en.md) and Issue #3.
