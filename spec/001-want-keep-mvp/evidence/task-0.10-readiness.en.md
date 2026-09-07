# task-0.10 — Ready-gate outcome

[Русский](task-0.10-readiness.md)

Date: 2026-09-07. Verdict: **Ready for development**.

This verdict applies to SDD decision completeness. The application, connectors, deployment and product ACs were not tested. Unverified external properties are converted into fail-closed contracts and explicit task-4.x/task-8.x entry/runtime gates.

## Blocker matrix

| BLK | task-0.10 decision | Evidence | Runtime gate |
| --- | --- | --- | --- |
| BLK-01 Alfa | D-37 limits scope to debit/current/savings/deposit/cashback. D-39 forbids UI/time/amount identity; unknown/ambiguous records do not post. | [Alfa](alfa.en.md), signed-in Chrome tab without publishing values | task-4.1: permission, structured fixture, IDs, lifecycle, two accounts, reauthentication, Alfa route |
| BLK-02 Raiffeisen | CAMT 1:N, scoped IDs, cross-report transaction fingerprint without amount/time, collision policy, revisions/reversals; statement ID is provenance; last confirmed CLBD with date, other balances/fees unknown. | [Raiffeisen](raiffeisen.en.md), synthetic JSON/XML | task-4.2: OAuth lifecycle, full history, corrections, second account, conformance |
| BLK-03 Ozon | Completed synthetic HAR projection is accepted for design; accountToken/groupID is not identity. | [Ozon](ozon.en.md), sanitized projections | task-4.3: session permission/lifecycle, history end, second account, reauthentication |
| BLK-04 Bybit | Route namespaces, candidate-only cross-log matches, hourly fallback and collision policy are fixed. | [Bybit](bybit.en.md), [RSA API](bybit-api.en.md) | task-4.4: precision/history, second account, rotation/revocation, conformance |
| BLK-05 Aifory | D-33 scope and fail-closed boundary close design without invented Flutter fields. | [Aifory](aifory.en.md), signed-in Chrome tab | task-4.5: permission, structured fixtures, identity/history/card lifecycle, reauthentication |
| BLK-06 EMCD | D-34 scope, separate namespaces and unknown/collision policy are fixed. | [EMCD](emcd.en.md), signed-in Chrome tab, synthetic scenarios | task-4.6: structured fixtures, balance/card/Grow/P2P lifecycle, second account |
| BLK-07 FX | D-40 accepts `valuation_unavailable` beyond 365 days and `quote_unavailable` without a provider quote. | [FX](fx.en.md) | task-6.1: Demo key/quota/attribution, live rates and cache behavior |
| BLK-08 OpenAI | Model selection, strict schema and cost/failure boundary were resolved by research. | [OpenAI](openai.en.md) | task-5.x/task-8.1: gateway, authz, budget and production health |
| BLK-09 Hosting | Configuration and budget are selected; provisioning/conformance are separate from SDD. | [Hosting](hosting.en.md) | task-8.1–8.3 and provider gates: hardening, load, invoice, backup/restore |
| BLK-10 Formula/API | D-39 identity, D-41 retention, D-42 numeric XIRR and D-43 version-bound admission make decisions deterministic. | [Contracts](../contracts.en.md), this report | task-1.2/1.3/3.3/4.x/6.4/8.1: executable contracts and tests |

## Normalized decisions

- Source key: `householdId + provider + stableExternalAccountId + productOrLogNamespace + providerRecordId`.
- Connection/session/cursor/job is provenance. Amount/time/text is not identity.
- Missing provider ID permits only a documented immutable composite. Collision → `source_ambiguous`, evidence + clarification, no posting.
- Gap → `source_partial`; missing balance/fee/status remains unknown.
- Terminal command detail: 90 days after outcome; unresolved: through reconciliation + 90 days; a tombstone with `commandId` lives throughout unresolved state and 400 days after terminal/reconciled outcome; recent: 30 days terminal + every unresolved; expired detail → HTTP 410 `command_expired`.
- FX gap beyond 365 days → `valuation_unavailable`; incomplete platform quote → `quote_unavailable`.
- XIRR: Actual/365, same-day aggregation, both signs, one sign transition, fractional powers in 50-digit HALF_EVEN decimal with `1e-24` NPV error bound, bisection from `-1 + 1e-12` through `1,000,000`, `1e-12` solver tolerance, 512 iterations.
- Provider admission: server-owned exact environment/build/contract/allowlist/configuration/permission binding; task-4.x provider evidence + task-8.x host evidence; stale/missing binding → `provider_not_admitted` before collector IO.

## Synthetic contract checks

| Scenario | Expected outcome |
| --- | --- |
| Same provider ID and payload is replayed | One source revision, one financial posting |
| Same provider key, different payload | New revision or `source_ambiguous` per provider policy; no second effect until resolved |
| Two facts share amount and second | Never merge without a shared provider ID/proven link |
| History starts after requested date | `source_partial`, coverage gap and opening-balance question; past is not zero |
| Fee/balance is absent | Typed unknown; neither `0` nor spendable |
| Bybit hourly tuple repeats with different payload | Both evidence revisions, `source_ambiguous`, no credit |
| CAMT entry contains two transaction details | One entry, two linked details, postings by proven semantics without duplicating entry amount |
| CAMT correction/reversal | New revision/correction link; original remains |
| One CAMT fact without a stable ID arrives in 052 and 053 | Statement IDs are provenance; a proven transaction fingerprint deduplicates across reports, otherwise `source_ambiguous` prevents a second posting |
| camt.052 is unavailable | Last CLBD with `asOf`; available/locked/fee unknown |
| Crypto history is older than 365 days | Native amount retained, `valuation_unavailable` |
| Platform quote lacks fee/spread coverage | `quote_unavailable`; CBR/CoinGecko never substitute as executable quote |
| `-1000`, then `+1100` after 365 days | XIRR `0.100000000000` |
| `-1000`, then `+1050` after 182 days | XIRR `0.102795595422` under the fixed decimal ln/exp policy |
| Root equals rLow / exceeds rHigh | The boundary root is accepted; a root above `1,000,000` yields `unavailable` |
| No sign transition; `-100,+230,-132`; same-day net zero | Explained `unavailable` |
| Terminal command older than 90, tombstone younger than 400 days | `command_expired`; same key/hash does not execute again, different hash rejected |
| Unresolved command older than 90 days | Retained until reconciliation; included in `/commands/recent` |
| Admission belongs to an old build/allowlist/configuration | `provider_not_admitted`; no job/provider IO/source record/posting occurs before a new combined pass |

## Evidence boundary

Confirmed: catalog structure, paired RU/EN, REQ → AC → task links, D-37–D-43 decisions and synthetic expected outcomes. Signed-in Chrome tabs confirmed selected Alfa/Aifory/EMCD areas, but response bodies were not exported and stable fields are not claimed as proven. Private HAR, keys and responses remain unpublished.

Not confirmed: financial application runtime, production provider permission, second account, reauthentication/revocation, complete history readback, target-host conformance, backup/restore or product/E2E ACs. These checks are listed in [plan.en.md](../plan.en.md) and do not alter SDD Ready.

## Review

This file is part of the reviewed candidate and intentionally does not embed its own fingerprint or result. The publication gate requires an Avida `pass` for the exact committed head; the immutable result with fingerprint/commit SHA is recorded in the PR review/check and Issue #10 comment. Any file change after review requires a new fingerprint and review.
