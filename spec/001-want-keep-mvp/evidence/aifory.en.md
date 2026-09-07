# Aifory Pro: read-contract research

[Русский](aifory.md)

Date: 2026-09-07, Europe/Moscow. Task: [task-0.5 / Issue #5](https://github.com/pchkauu/want-keep/issues/5). Repository baseline: `7189149`, branch `docs/want-keep-mvp-sdd`. Environment: the owner-provided authenticated portal in Google Chrome. Application and server API versions were not established.

**Research completed with blocking automation findings.** RUB accounts, USDT, ETH and the existing virtual card were read through the interface. No structured provider contract or automatic import was verified. BLK-05 remains open under task-0.10; task-4.5 and the MVP remain **Not Ready**. Research may finish with these findings under the [README](../README.en.md) rule.

## Current scope: D-33

Explicit owner decision dated 2026-09-07: RUB accounts, USDT, ETH and the existing crypto card are required. Other Aifory products are outside the current research scope and **do not block the MVP**. This changes requirements; it does not claim the platform has no other products.

| Included | Boundary |
| --- | --- |
| RUB | Individual accounts and groups; balances and movements, including exchanges, cash deposits/withdrawals and fees |
| USDT | Wallet, observed TRC-20 network, balance and all movements of the selected wallet |
| ETH | Ethereum wallet, exact amounts, withdrawals, exchanges and fees; ETH added to Want Keep's supported assets |
| Existing virtual card | Actual USD balance, funding, payments, fees and statuses; a crypto card is not assumed to be a credit card |
| Future extension | Other currency wallets, other card types, P2P as a separate product, referral product and service catalogs. No requirement to research/connect them now |

A deferred product does not mean skipping its movement through an included wallet. When history contains a service payment, referral-account transfer or certificate operation, retain source-provided amounts, status, fee and provenance. Unknown semantics need clarification, not invented income. Do not collect active certificate numbers: they may grant access to funds.

## Evidence ledger

Below, `confirmed` means only the stated observation or published document content, `inference` means a deduction, and `unverified` means no proof. Amounts, addresses, login, identifiers and personal names are excluded from public documents. Portal images and original financial data were not saved to Git.

| ID | Status and source | Established | Evidence limit |
| --- | --- | --- | --- |
| AIFORY-E01 | confirmed, user decision | RUB, USDT, ETH and existing card; other products deferred without blocking | Selected products still require reliable automatic reading |
| AIFORY-E02 | confirmed, [portal home](https://app.aifory.pro/home) | USDT/TRC-20, ETH/Ethereum, two RUB groups, USD card; separate referral block and other currencies visible | Composition of this portal, not the global platform catalog |
| AIFORY-E03 | confirmed, both RUB groups and child accounts | Each group contains one child account; the same office name appears under different groups. One account has history, another an explicit empty state. Zero balances are visible | Office name, currency or balance cannot establish identity; a group total is not another asset |
| AIFORY-E04 | confirmed, RUB cash-withdrawal detail | Amount, status «Подтверждено», truncated request ID, withdrawal account, date/time and fee | Request status alone does not prove physical cash receipt or define posted without a contract |
| AIFORY-E05 | confirmed, USDT wallet | Balance, network and truncated address; deposits, withdrawals, exchanges and card funding in history | No separate owned/available/hold or sourceAsOf fields in the inspected view |
| AIFORY-E06 | confirmed, completed RUB → USDT exchange | Both native amounts, debit/credit accounts, «Исполнено», truncated order ID and date/time; corresponding row also appears in RUB history | No shared full ID obtained from structured responses; separate fee/quote not shown |
| AIFORY-E07 | confirmed, incoming USDT | Amount, «Исполнен», truncated order ID, account, address and date/time | Transfer origin not classified as owned/external; network confirmations and revisions unknown |
| AIFORY-E08 | confirmed, on-chain USDT withdrawal | Recipient amount, separate fee, account/network, truncated address, ID and hash, date/time and explorer link. History row includes principal and fee | Explorer not opened; no independent blockchain verification |
| AIFORY-E09 | confirmed, ETH withdrawal and two ETH history rows | Exchange from USDT, ETH withdrawal with separate fee; displayed receipt minus withdrawal and fee matched the visible balance. Display contains 8 decimal places | Arithmetic of this UI sample only; API precision, opening point and history completeness unproven |
| AIFORY-E10 | confirmed, two card-funding entries in USDT history | Plus sign in row/detail despite a named debit wallet; funding amount, fee percentage, card, «Завершен» and date/time | Sign is not posting direction. Header/funding difference is not simple multiplication by the displayed percentage; basis/rate/rounding unknown |
| AIFORY-E11 | confirmed, card and its history | USD balance and card mask, separate payments, fees and USD funding. Details show date with year, time, card, status and purchase merchant | Full PAN/CVV not revealed; inspected card details expose no ID, MCC or authorization/clearing link |
| AIFORY-E12 | confirmed, similar payments and fee | Equal amount/time, different merchant descriptions: one «В процессе», another «Подтверждено». Separate fee confirmed; zero-value «В процессе» payment also inspected | inference: possibly different stages of one payment, not an established link. Declines, cancellations and refunds untested |
| AIFORY-E13 | confirmed + inference, USD card funding | «Подтверждено», USD amount, date with year and time. A plausible USDT-funding candidate for the same card is close in time | No shared ID or established rate; equal USDT/USD numeric values do not establish parity |
| AIFORY-E14 | confirmed, wallet row labelled «Вывод» | Details reveal an external-service payment: purpose, confirmed status, USD/USDT amounts, fee and total debit | Service catalog/execution not researched; generic row title is insufficient for classification |
| AIFORY-E15 | confirmed, reading and scrolling | USDT history reveals older entries and the list's bottom edge; wallet groups/details omit the year. Card history scrolls separately | No proven retention, cursor, page size, server ordering, start-date filter or completeness marker; all operations were not exported |
| AIFORY-E16 | confirmed, UI and routes | `Enable accessibility` did not expose readable financial DOM fields; images were used. USDT and ETH share `/home/wallet` without query; operation modal leaves URL unchanged; card opens inside `/home` | Coordinates/common URL are neither an automation contract nor ID. Hidden state, cookies, tokens and network traffic were not extracted |
| AIFORY-E17 | confirmed, settings | Login/security sections visible; no separate developer/API connection on the inspected screen | Security «Ключи доступа» was not identified as API keys. No settings/secrets changed; initial session was already authenticated |
| AIFORY-E18 | confirmed, [Aifory terms](https://aifory.pro/terms-of-use), §§5, 9, 12, 17.2 | Document distinguishes wallets from bank accounts, allows deposit-address changes, assigns cards to third-party providers and requires operator consent for automated access. Exchange rates may include spread | Published terms do not prove permission for this collector, API access, current individual tariffs or runtime |
| AIFORY-E19 | unverified, [official site](https://aifory.pro/) and public documentation search | No personal read-only API with schemas, scopes, auth and pagination found in inspected sources | Does not assert such an API cannot exist. Similarly named services' documentation is not an Aifory contract |

## AS IS and target adapter

AS IS: active session → home → selected group/account or card → list → modal details. The platform owns balances/statuses; card processing is an external boundary. Want Keep currently receives no automatic import result.

There is no confirmed Aifory HTTP request/response in this research. `/home` and `/home/wallet` are UI routes, not API endpoints. Do not publish invented curl examples, transfer cookies into scripts or present the JSON below as provider responses.

Target rules for task-0.10/task-4.5, **not implemented**:

| Boundary | Required verified contract |
| --- | --- |
| Access | Operator consent/permitted reading method, free access, auth/scopes, session policy and limits. Prefer a verified official personal API; otherwise a permitted structured portal read contract |
| Account identity | Household/provider/external-account/product/asset/network namespace and permanent product ID. connectionId is provenance; names, addresses and routes are not identity. RUB groups do not duplicate child balances |
| Money/balances | Exact strings and provider scale; do not round ETH to fiat cents. Preserve USD/USDT separately. Missing available/locked/debt/asOf remain unknown; do not label a RUB platform wallet as a bank deposit |
| Operations | Full IDs/revisions per operation family, linked source legs, native amounts, fees, status mapping, time with year/zone and provenance. Do not build postings from header signs or OCR text |
| Card lifecycle | Proven authorization/clearing linkage, separate fee/refund/reversal and hold/release. Clarify E12's similar pair until linkage is established; no double debit, merging distinct purchases or subtracting a hold twice |
| Card funding | Link USDT debit to USD credit; gross/net, fee currency/basis, rate and rounding. Internal movement with separate fee, not consumption expense |
| History | Coverage start, year/zone, ordering, cursor/offset, limits and end marker. Persist confirmed pages before advancing, overlapping replay and reconciliation; unknown completeness is not a successful full import |
| Household access | Do not merge different external accounts; reauthorization links to the same source. Owner handles MFA; disconnect/version fence rejects stale results |
| Read-only | Exact origin/method/path/body allowlist; deny product opening/funding/closure, repeat-payment, exchange, withdrawal and security changes. A button's presence is not collector permission |

Later product expansion requires updating D-33/REQ-046/AC-046, capability matrix, evidence and corresponding fixtures. No universal adapter for every service needs to be built in advance.

## Synthetic verification examples

These are **project scenarios** based on observed behavior classes, with wholly invented values. They are not provider fixtures, real identifiers or API proof.

```json
{
  "kind": "synthetic_mapping_cases",
  "eth_withdrawal": {
    "asset": "ETH",
    "received_before": "0.05000000",
    "recipient_amount": "0.04000000",
    "fee": "0.00300000",
    "history_debit": "0.04300000",
    "remaining": "0.00700000"
  },
  "card_funding": {
    "wallet_asset": "USDT",
    "card_asset": "USD",
    "wallet_header_sign": "+",
    "debit_wallet_ref": "wallet-example",
    "credit_card_ref": "card-example",
    "gross_debit": null,
    "fee": null,
    "executed_rate": null,
    "link_state": "needs_evidence"
  },
  "similar_card_rows": [
    {"amount": "-12.00", "asset": "USD", "source_status": "В процессе", "source_id": null},
    {"amount": "-12.00", "asset": "USD", "source_status": "Подтверждено", "source_id": null}
  ]
}
```

Expected: `0.04000000 + 0.00300000 = 0.04300000`, remainder `0.00700000`; do not subtract the fee twice. Unknown card-funding fields remain null. Similar rows prove neither USD 24 expense nor an automatically merged USD 12 expense: establish IDs/link/statuses before one confirmed posting.

## Blockers and handoff

| ID | Status | Needed evidence and closure owner |
| --- | --- | --- |
| AIFORY-B01 | CLOSED | Access to the supplied tab confirmed; this does not test reauthorization |
| AIFORY-B02 | OPEN | task-0.10: permitted free automatic access, operator consent, documented read contract, auth/limits/allowlist. No client-side fix substitutes for access permission |
| AIFORY-B03 | OPEN | task-0.10: structured IDs/statuses/precision/time/coverage, full repeat/resume case, reauthorization and second external account; no human-name identity shortcut |
| AIFORY-B04 | OPEN | task-0.10: card lifecycle linkage, funding gross/net/conversion, fees, refunds and holds; E10–E13 do not prove automatic matching |
| AIFORY-B05 | DEFERRED, NON-BLOCKING | Other D-33 products; investigate only after a future explicit scope extension |

BLK-05 contains only AIFORY-B02–B04 for included products. task-0.5 finishes evidence collection and hands questions to task-0.10; task-4.5 receives concrete rules/scenarios but remains blocked. ETH valuation is handed to task-0.7 and the valuation implementation task; an unverified rate is not zero.

## Verification

Performed: read-only Chrome UI observations E02–E17, official-source inspection E18–E19, comparison of displayed ETH native amounts and analysis of ambiguous card rows. SDD validation passed: 87 REQ, 105 AC, 67 tasks, 35 screens and 67 GitHub mappings. All 13 generator tests passed. RU/EN ID/source/JSON parity, exact Decimal arithmetic, diff and absence of private data in public examples were checked. Validation also ran on the isolated commit candidate; another provider's unfinished changes were excluded.

Commands: `python3 spec/001-want-keep-mvp/tools/spec_tool.py check`, `python3 -m unittest discover -s spec/001-want-keep-mvp/tools -p 'test_*.py'`, `git diff --check`. These validate documentation/generation, not the application.

Not performed: API/collector runtime, HAR capture, full history export, DE/NL/BG cold start, hourly sync, reauth/MFA/expiry, second account, replay/429/5xx, refund/reversal and full-card money reconciliation. Reason: no permitted verified structured contract or required runtime samples. Full AC-046, AC-041, AC-048, AC-079, AC-087 are not claimed as passed. Aifory credit/savings terms are not researched and do not block current scope.
