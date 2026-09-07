# Raiffeisen Business Online: read-contract research

[Русский](raiffeisen.md)

Date: 2026-09-07. Task: [task-0.2 / Issue #2](https://github.com/pchkauu/want-keep/issues/2). **Research is complete; production account and historical statement reads are verified, while the connector is not implemented.** The original RAIF-B02/B03/B04/B06 were handed to task-0.10; D-39 and the task-4.2 runtime gate are recorded in the final section below.

## Scope and selected approach

The owner reports using an individual entrepreneur's current account in RBO for contract income, with no credit or deposit products. **The official API is preferred over Playwright.** Research and future import use the API first; a browser collector is considered only for a demonstrated API gap. This preference is not an absolute ban on a fallback.

The owner uses Google Chrome for integration setup and authorization. Agent access to Chrome is not a prerequisite for public API research or subsequent server import. The previously inspected sign-in page was in Arc, as corrected by the owner; it does not establish that the Chrome session expired. The requested Chrome profile was later found and its integration section read (RAIF-E12); transactions, account details and balances were not researched.

D-35: the owner confirmed the entrepreneur current account only. Raif cards, credit, savings accounts and deposits are excluded; other providers and shared features remain. REQ-043/AC-043 updated with IDs preserved.

## Dated sources

All public sources below were read on 2026-09-07. A public specification, simulator and production run are distinct evidence levels.

| ID | Source | Evidence and limits |
| --- | --- | --- |
| RAIF-E01 | Issue #2, direct GitHub read | Initial read before research: RU/EN card, OPEN, no comments; research scope. Delivery is tracked in Issue #2 |
| RAIF-E02 | Owner messages | Individual entrepreneur current account, contract income, no credit/deposits; API preferred over Playwright |
| RAIF-E03 | [API portal](https://developer.raiffeisen.ru/) | RBO integration and free use according to the public description; owner connection unverified |
| RAIF-E04 | [API help](https://developer.raiffeisen.ru/docs/support/help) | Authorization errors and end-of-day statement availability; not a guaranteed latency for every data source |
| RAIF-E05 | [API catalog](https://developer.raiffeisen.ru/docs/api) | Statements and Corporate Cards found. Both catalog pages were read through the public portal API; a separate current accounts/balances specification was not found there |
| RAIF-E06 | [RBO agreement dated 2026-03-23](https://www.raiffeisen.ru/static/common/RBO_docs/Soglashenie_RBO/23-03-2026/23-03-2026_RBO_publ-all.pdf) | Historical description of RBO and user rights. Does not establish the owner's current terms or current permission for automation |
| RAIF-E07 | [Getting started](https://developer.raiffeisen.ru/docs/howToStart), [connection](https://developer.raiffeisen.ru/docs/howToStart/howToConnect) | Signed application, system registration, Redirect URL and Client secret; user-accessible accounts, including the Operator role |
| RAIF-E08 | [Tokens](https://developer.raiffeisen.ru/docs/howToStart/tokens/howToGetTokens), [Code Flow](https://developer.raiffeisen.ru/docs/howToStart/tokens/howToGetTokensByCodeFlow), [RBO issuance](https://developer.raiffeisen.ru/docs/howToStart/tokens/howToGetTokensInOnlineBank) | Token lifetimes/rotation, PKCE, manual initial issuance. Does not prove read-only token permissions |
| RAIF-E09 | [First API request](https://developer.raiffeisen.ru/docs/howToStart/howToTestApiConnection) | Production account-list URL, field projection, two token headers, HTTP 200 with an array of accessible accounts |
| RAIF-E10 | [Statements specification](https://developer.raiffeisen.ru/docs/api/adddd098-b2cc-4363-a01c-3453b48998c9), [public OpenAPI file](https://developer.raiffeisen.ru/api/v1/public-apis/adddd098-b2cc-4363-a01c-3453b48998c9/variants/public/spec) | Direct HTTP GET: OpenAPI 3.0.3, info.version 1.0.0, server, requests, JSON responses and statuses. Official contract; no token-authenticated banking endpoint called |
| RAIF-E11 | [Statement sandbox](https://developer.raiffeisen.ru/docs/sandbox/statements), [test values](https://developer.raiffeisen.ru/docs/sandbox/test-values) | Public simulator description. Sandbox tokens remain static across repeated issuance, unlike production rotation. No sandbox run performed |
| RAIF-E12 | Authenticated RBO in the owner's specified Google Chrome | At the owner's explicit request, a Want Keep application was created with their contact details and offer consent. After the owner completed the SMS signature, the bank confirmed document acceptance and application submission, then requested system registration. The connection card says use is free. The later registration outcome is RAIF-E15; personal fields and the application number are excluded from the repository |
| RAIF-E13 | [Offer linked by the RBO form](https://www.raiffeisen.ru/static/common/corporate/contracts/cifroviye_kanaly/api_orchestrator/API-orchestrator-rules.pdf) | Individual entrepreneurs are eligible clients; IP restrictions and registration-change email OTP are described. Refresh token: 180 days, conflicting with RAIF-E08. Tariffs determine fees; the free-use card does not guarantee an unchanged price |
| RAIF-E14 | Registration form reached through the accepted application's private link; [Redirect URI changes](https://developer.raiffeisen.ru/docs/howToStart/changes/howToUpdateRedirectUri) | Redirect URI field has an `https://` placeholder; Client secret requires 30–36 characters, a digit, an uppercase and a lowercase Latin letter. Client ID follows registration. Documentation describes an address accessible from the client's infrastructure; no explicit localhost/loopback support confirmation was found. The private link is not published |
| RAIF-E15 | Registration in the owner-specified Chrome and RBO readback; domain HTTPS check | The owner acquired want-keep.tech and entered Client secret themselves. Registered Redirect URI: `https://want-keep.tech/api/v1/connections/raiffeisen/callback`. The form confirmed registration and Client ID issuance; refreshed RBO application status is “Completed”. Client secret was not extracted; Client ID is not published. Direct HTTPS HEAD could not execute because DNS resolution failed, including outside the sandbox; callback operation is unconfirmed |
| RAIF-E16 | Owner-provided VPS, authoritative DNS, public HTTP/TLS requests and Certbot | After DNS propagation the domain resolves; HTTPS with a valid certificate works, including a Mac request without overriding DNS. nginx/Certbot installed; `nginx -t`, simulated renewal and deploy hook passed. `/_health` returns 204 for infrastructure only; callback returns 503 without processing a code. [Configuration and runbook](../../../deploy/raiffeisen-research/README.en.md); application not deployed |
| RAIF-E17 | RBO Integrations section and New refresh token form | Integration Connected, registered URI visible, no IP restrictions configured. The form requires URI selection, bank password and Client secret. URI selected; Client secret filled from the owner-designated file without printing its value. The owner must enter the bank password in Chrome; token issuance is pending. IP settings were not changed |
| RAIF-E18 | Refresh grant using owner files | HTTP 200, token set rotated and saved privately; Refresh token changed |
| RAIF-E19 | Two account GETs from Mac | HTTP 200, one account, UUID id and number stable, currency RUR |
| RAIF-E20 | camt.053 for 2026-08-01–2026-08-31 | dryRun 200, generation 202, completed, XML 200; seven entries and OPBD/CLBD reconciliation |
| RAIF-E21 | Independent camt.053 for 2026-08-01–2026-08-30 | Seven NtryRefs and Ntry elements matched the first report |
| RAIF-E22 | camt.052 for 2026-09-07 | HTTP 404, code=no-statements; current balance unverified |
| RAIF-E23 | Explicit owner decision | Entrepreneur current account only under D-35; other Raif products outside current MVP |
| RAIF-E24 | One VPS GET after separate approval | HTTP 200, same account as Mac; tokens in process memory only, response private on Mac |

SHA-256 of the retrieved RAIF-E10 file: `e9b5511901c748d51c8d0f70c7b1fa23febda65de8c33ee0b453bc19e776c703`. This fingerprints the public specification, not owner data. The full third-party file is not committed; its link and fingerprint identify the version.

## API connection and authorization

The owner completed system registration and initial token issuance in RBO; actual rotation and reads are RAIF-E18–E24 below. Registered URI: https://want-keep.tech/api/v1/connections/raiffeisen/callback; DNS/HTTPS verified, callback placeholder still 503. The initial RBO token enabled API research before application Code Flow implementation.

Future Code Flow under the [contract](../contracts.en.md) uses https://sso.rbo.raiffeisen.ru/authorize, exact redirect_uri, single-use code, state/nonce, PKCE S256 and owner/household/connection-version bindings. Documented scope: openid profile email phone; a read-only banking scope is not established. The client independently restricts requests to reads. No payments performed.

Exchange/refresh endpoint: POST https://sso.rbo.raiffeisen.ru/token. Refresh uses Basic Base64(client_id:client_secret), form-urlencoded grant_type=refresh_token, client_id, refresh_token. Code Flow also requires code, redirect_uri, code_verifier; it was not run. Secrets/tokens are not printed.

RAIF-E08 documents access/id at 24 hours and refresh at 30 days; the RAIF-E13 offer specifies 180 days for refresh. Response has no expires_in. One rotation verified; lifetime/logout/revocation/concurrent refresh, second account and OIDC checks remain open. Unknown outcomes block retry until reconciliation.

## Accounts and statements contract

| Operation | Confirmed contract | Limit |
| --- | --- | --- |
| Account list | `GET https://api.openapi.raiffeisen.ru/api/v1/accounts?fields=Id,Number,Name,OrganizationName,Currency`; `Accept: application/json`, `Authorization: Bearer {access_token}`, `ID-Token: {id_token}`; HTTP 200, array of user-accessible accounts (RAIF-E09) | Projection, types and one account's stability verified in RAIF-E19–E21; current balances, pagination and identity across reauth need verification |
| Historical/end-of-day statement | `POST https://api.raiffeisen.ru/bank-statements/v1/reports/camt-053`; JSON `accountKeys, from, to`; 202 with `reportId` (RAIF-E10) | `to` cannot be today's date; the ID belongs to a job, not a transaction |
| Today's statement | Same host/prefix, `POST /v1/reports/camt-052`; `from = to = today` | Bank timezone and intraday entry detail require verification |
| Job status | `GET /v1/reports/{reportId}/status`; 200, `reportId` and `status` | Enum: CREATED, STARTED, FAILED, STOPPED, RESTARTED, COMPLETED, CANCELLED. These are generation states, not financial transaction statuses |
| File retrieval | `GET /v1/reports/{reportId}/file`, HEAD for metadata; Range, ETag, Content-Length documented | OpenAPI describes binary/XML/ZIP and other formats but contains no bank transaction XSD/structure |
| Parameter validation | `dryRun=true` on generation POST; 200 on successful validation without starting a job | Does not establish statement contents/completeness; false starts generation |

All statement calls require Authorization and Id-Token. Accounts and statements have **different published base URLs**; do not transfer paths between them automatically. `accountKeys` contains a 20-digit account number or `number:CNUM`; do not assume equality with account API `Id`. In RAIF-E20–E21 number without CNUM was accepted and matched the XML account number; CNUM applicability to other accounts needs verification.

Synthetic example based on RAIF-E10; values are invented and the request was not executed:

```http
POST /bank-statements/v1/reports/camt-053 HTTP/1.1
Host: api.raiffeisen.ru
Authorization: Bearer {access_token}
Id-Token: {id_token}
Content-Type: application/json

{"accountKeys":["00000000000000000000:123456"],"from":"2026-08-01","to":"2026-08-31","zero":true}

HTTP/1.1 202 Accepted
Content-Type: application/json

{"reportId":"00000000-0000-4000-8000-000000000001"}
```

```json
{"reportId":"00000000-0000-4000-8000-000000000001","status":"COMPLETED"}
```

The last JSON is a status-response example using the documented schema. Only COMPLETED permits file retrieval; coverage is confirmed after content validation. Statement generation creates a report, not a bank payment: POST is allowlisted for that purpose.

### History, errors and limits

- Candidate hourly import: camt.052 for today, followed by camt.053 reconciliation of completed days. RAIF-E04: a working day's final statement becomes available the next day after 04:00 Moscow time; first request is recommended after 06:00, Saturday on Monday or Sunday after 14:00. This does not prove that source data itself refreshes hourly.
- `zero=true` includes statements without movements. Missing statements or 404 do not establish a zero balance/complete history. Exact from/to inclusivity, retention, maximum range, account/transaction counts and quotas are absent from the inspected specification.
- 400, 401, 404 and 500 are documented; ErrorType uses UPPER_SNAKE_CASE while JSON examples use kebab-case (`NO_STATEMENTS` / `no-statements`). This is a documentation conflict. Verify actual codes and mappings; unknown errors must not advance coverage. Do not classify using free-form message text.
- TOO_MANY_TRANSACTIONS_ERROR exists without a numerical threshold. Period splitting must retain unverified intervals. CAMT methods have no documented transaction pagination: they produce reports, not transaction pages.
- No generation idempotency key is described. Another job after a timeout must not post money again. Recovery uses a known reportId; after file retrieval, deduplication requires a stable transaction ID. Report ID or file hash is not a substitute.
- Range/If-None-Match are present but 206/304 responses are not listed. Byte-range download resume is not yet confirmed. Generated-file retention, polling interval and STOPPED/RESTARTED behaviour need verification.

## Production run and mapping, 2026-09-07

Research completed with documented blockers under the README rule. After the owner issued the Refresh token, refresh grant returned HTTP 200; a new token set was atomically saved with mode 0600 in a private local 0700 directory and the Refresh token changed. Basic Base64 is verified; the response contains access_token/id_token/refresh_token/token_type, no expires_in. The original file is bootstrap input; subsequent commands use tokens.json. Scripts do not print credentials. The JWT is used as an opaque API credential obtained over bank TLS, not as verified application-user OIDC login.

Two account GETs from Mac returned the same account. After separate owner approval, one GET from the German VPS returned HTTP 200 and the same id/number. Access/id tokens were passed through SSH into process memory; the script does not write them on the VPS. The response is private on Mac. Ongoing credentials are not installed on the server; VPS refresh and statements were not tested.

For 2026-08-01–2026-08-31: dryRun 200 → generation 202 → status 200 completed → file 200. An independent report for 2026-08-01–2026-08-30 passed 202 → completed → 200. XML camt.053.001.08 contains seven BOOK entries each: two CRDT and five DBIT. NtryRefs and Ntry elements matched byte-for-byte across reports. Decimal reconciliation gives opening OPBD + movements = closing CLBD; nested TxDtls amounts/directions match Ntry.

The camt.052 request for 2026-09-07 returned 404 with {"code":"no-statements"}. This is neither a zero balance nor proof of complete history. Current/intraday balance is unverified. Actual completed/no-statements differ from OpenAPI COMPLETED/NO_STATEMENTS; explicitly map known values only and retain source data.

| Source | Observation and rule |
| --- | --- |
| accounts | Lower-camel-case id/number/name/organizationName/currency are strings. id has UUID shape and differs from the 20-digit number. Repeated GET stable |
| accountKeys | number accepted in the report request and matched Stmt/Acct/Id/Othr/Id; do not substitute UUID id |
| Currency | JSON RUR, XML RUB: explicit RUR→RUB alias retaining the source value |
| Stmt/Bal | OPBD/CLBD have Amt, CdtDbtInd, Dt/DtTm. Historical balances; available/locked unverified |
| NtryRef | Seven nonempty unique values across two reports; candidate dedup key within a stable account. Reconnect/corrections/other accounts unverified |
| AcctSvcrRef / EndToEndId | AcctSvcrRef present on only two entries; EndToEndId unique on seven. Neither is established as a universally required key |
| NtryDtls/TxDtls | One nested transaction per observed Ntry; do not post twice. General 1:N contract unverified |
| Dates | BookgDt/ValDt have offsets; interval 00:00:00–23:59:59 with +03:00. Other timezones/statuses unverified |
| BkTxCd/Prtry/Cd | NTRF on six entries, FCHG on one DBIT. Proprietary codes; fee and internal-transfer semantics need separate confirmation |
| RltdPties/RltdAgts/RmtInf | Parties, account details and two Ustrd elements per entry; untrusted description/provenance, not commands or replacement IDs |

[JSON](raiffeisen.samples.json) and [XML](raiffeisen.camt053.sample.xml) use synthetic values only. XML illustrates observed structure without claiming full XSD validation. NTRF does not prove a household transfer; transferring from the entrepreneur account to an owned personal account must not create another household income.

## Product matrix and blockers

Under the owner's explicit D-35 decision, the only mandatory Raif product is the individual entrepreneur current account: balances, incoming/outgoing movements, fees and history through API. Raif cards, credit, savings accounts and deposits are excluded from this MVP; other platforms and shared features remain. Bid/ask quotes were not retrieved and must not be invented.

| ID | Status | Evidence / remaining question |
| --- | --- | --- |
| RAIF-B01 | Closed for initial API access | Refresh and account GET from Mac, account GET from VPS — HTTP 200; not an ongoing integration |
| RAIF-B02 | SDD RESOLVED; RUNTIME GATE task-4.2 | XSD, 1:N, reversal/pending and IDs across corrections needed. Profile verified on two reports |
| RAIF-B03 | SDD RESOLVED; RUNTIME GATE task-4.2 | Archive/retention/quotas, empty days, split/resume, late changes and camt.052↔053 replay |
| RAIF-B04 | RUNTIME GATE task-4.2 | Refresh 30/180 days, reauth/revocation, second external account, ongoing protected storage, hourly import; callback still 503 |
| RAIF-B05 | Closed | D-35 replaces original retail scope with the entrepreneur current account only |
| RAIF-B06 | SDD RESOLVED; RUNTIME GATE task-4.2 | Current/available/locked balance and exact fee semantics. 404 is not zero; unused products do not block |

task-4.2 verifies intraday on an available banking day, current balance after no-statements, fee semantics, history depth, reauthentication and a second account. Tests move no money; unknown outcomes are reconciled first. D-39 closes SDD rules without replacing these runtime tests.

## Traceability and verification

AC-043/REQ-043: account and historical statement live reads completed; full application criterion not passed. AC-041/REQ-041: overlap checked, full archive/resume unverified. AC-048/REQ-048: diagnostic allowlist tested synthetically; no financial commands executed. AC-079/REQ-065: one account stable across requests/hosts; reconnect and two members unverified. AC-087/REQ-073: one rotation and unknown-outcome guard verified; household authorization/stale-job revocation not implemented. AC-070 removed from Raif under D-35; other platforms' credit/savings requirements remain.

[Verification](../verification.en.md) separates tests, API and infrastructure. Originals and credentials remain private; public examples contain no real amounts, personal data, account details, tokens or local credential paths. Code Flow, sandbox and full application runtime were not tested. Commit/push results and research-closure readback are recorded in [Issue #2](https://github.com/pchkauu/want-keep/issues/2); BLK-02 remains assigned to task-0.10.

## task-0.10 decision, 2026-09-07

RAIF-B02/B03/B04/B06 above move from a global SDD blocker into the executable task-4.2 gate. The target contract permits CAMT entry to 1:N transaction details. For every postable detail, a sufficient versioned `camtCrossReportFingerprint` built from fields proven invariant across overlapping camt.052/camt.053 is always the canonical providerRecordId. NtryRef/AcctSvcrRef/EndToEndId and statement/report ID are atomically registered as aliases/provenance and do not select an alternative key; amount/time are excluded. An insufficient fingerprint or ambiguous alias mapping yields `source_ambiguous`, retains evidence and creates no new posting. Corrections/reversals are retained as revisions.

`no-statements` is not a zero balance. When camt.052 is unavailable, use the last confirmed CLBD with `asOf`/coverage; available/locked/balance/fee without evidence stay unknown. OAuth lifecycle, complete archive, corrections, a second account and live conformance are mandatory before provider deployment but do not block SDD development.
