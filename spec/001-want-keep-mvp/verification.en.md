# Want Keep MVP readiness verification

Verdict: Not Ready

Date: 2026-09-06. This verdict concerns the complete specification's readiness for application implementation. It neither prevents agreed documentation/backlog delivery nor describes an implemented application.

## Artifact state

Prepared proposal, 87 REQ, 105 AC, architecture, contracts/formulas, glossary, Mermaid flows, integration matrix, operating constraints and 67 RU/EN cards. The catalog/generator preserves the same IDs/links in both languages. Requirements/ACs have task coverage; every implementation task depends on the task-0.10 gate.

`plan.md` is intentionally absent: SDD requires Ready before issuing an implementation plan. Research tasks exist in the full backlog; completing research with a negative result does not unblock an unavailable product.

## Blockers

| ID | Unknown | Closure owner | Required evidence |
| --- | --- | --- | --- |
| BLK-01 | Automatic Alfa retail reading/full coverage | task-0.10, task-0.1 evidence | [Research completed with blockers](evidence/alfa.en.md): UI read, retail API published; needs authorised automatic contract, identity/history/reauth/second account, credit card and exact terms |
| BLK-02 | Raif: current balances, full mapping/history and auth lifecycle | task-0.10, task-0.2 evidence | [Evidence](evidence/raiffeisen.en.md): research completed. RAIF-B01/B05 closed; B02/B03/B04/B06 open. API account from Mac/VPS and two historical statements verified; D-35 — entrepreneur account only. task-4.2 and MVP Not Ready |
| BLK-03 | Operational Ozon debit-card/main-account contract under D-32 | task-0.10, task-0.3 evidence | [Research completed with blockers](evidence/ozon.en.md): read structures obtained; needs history completion, acceptable session operation, second account and unavailable semantics. Other Ozon products not required |
| BLK-04 | Bybit Funding USDT/USDC/ETH/BTC, used Easy Earn and P2P under D-36 | task-0.10, task-0.4 evidence | [Authenticated evidence](evidence/bybit-api.en.md): BYBIT-B03/B04 — cross-log identity, precision reconciliation/history, hourly identity/revisions and Earn forecast basis. BYBIT-B02/B05 read access closed; unused products unnecessary. |
| BLK-05 | Aifory: RUB, USDT, ETH and existing USD card under D-33 | task-0.10, task-0.5 evidence | [Research](evidence/aifory.en.md): AIFORY-B02–B04 — automation permission, structured read contract, identity/history/reauth, card lifecycle/fees/FX. Other products not required |
| BLK-06 | EMCD: USDT wallet, used Grow/crypto cards and P2P history under D-34 | task-0.10, evidence task-0.6 | [Research](evidence/emcd.en.md): EMCD-B02–B04 — structured read/access, identity/history/reauth, balances/Grow/card/P2P. Mining has never been used; other products are unnecessary |
| BLK-07 | Free current/historical FX valuation and provider quotes | task-0.10, task-0.7 evidence | [Research](evidence/fx.en.md): CBR + CoinGecko Demo cover current/≤365d; FX-B02–FX-B04 — >365d crypto history, provider executable quotes and keyed Demo probe/attribution |
| BLK-08 | OpenAI models, measured quality/cost and request limits | task-0.8 | Financial-invariant evaluation with tokens/errors and selected versions/limits |
| BLK-09 | Concrete VPS/reachability/cost and Mac backup retention | task-0.9 | Dated ≤$40 estimate, reachability, retention/capacity and recovery design |
| BLK-10 | Full structured grace/accrual terms, exact API boundaries and XIRR solver | task-0.10 | Verified input contracts, algorithms/vectors, updated RU/EN and independent Ready review |

BLK-01–BLK-06 need separately and securely supplied owner access. None was supplied in the initial 2026-09-06 stage; subsequent research is recorded separately. Marketing pages, mocks or corporate API availability do not close these blockers.

## Alfa-Bank: task-0.1 research completion, 2026-09-07

[RU evidence](evidence/alfa.md) / [EN evidence](evidence/alfa.en.md): 20 dated sources/observations, mandatory-product and cashback matrix, published retail schemas, synthetic examples, documentation contradictions and ALFA-B01–ALFA-B06. ALFA-B01 is closed for the live session; ALFA-B02–ALFA-B06 remain open within BLK-01. Research output is complete under the README rule; full AC-042 and AC-041/AC-048/AC-070/AC-079/AC-087 are not claimed as passed.

Research checks are separate from application checks: local spec validation, generator tests, RU/EN IDs/examples/links and diff review. Rendering the card from catalog.json requires RU/EN status support; the default-status test uses a copy without status instead of assuming task-0.1 will always be unstarted. This task's REQ/AC and dependencies are unchanged. Runtime API/collector, full history, reauth and second account were not tested for the evidence-listed reasons. Verdict remains Not Ready; downstream implementation is blocked.

## Rates: task-0.7 research completion, 2026-09-07

[RU evidence](evidence/fx.md) / [EN](evidence/fx.en.md): CBR selected as primary USD/RUB; Frankfurter `providers=CBR` as fallback/cross-check; CoinGecko Demo for separate BTC/USD, ETH/USD, USDT/USD and USDC/USD current and ≤365-day history. USD-cross formula, prohibition on merging USDC.E, effective-date policy, cache/failure/audit rules, 18 evidence items and FX-B01–FX-B06 are recorded in both languages.

TradingView was tested and rejected: libraries require an external datafeed, while its terms prohibit automated price referencing/non-display processing. CBR/CoinGecko returned HTTP 200 from DE/NL/BG; Frankfurter succeeded from DE/BG and then three NL networks after one probe-local DNS failure. This is a dated snapshot, not an SLA or actual VPS runtime.

Research is complete under the README rule. BLK-07 retains FX-B02–FX-B04 under task-0.10: a decision for crypto history older than 365 days, provider-specific executable quotes/fees, and CoinGecko Demo key creation/verification with attribution. task-6.1 and the MVP remain Not Ready. Application AC-037/AC-038/AC-039/AC-074 are not claimed as passed.

## Checks for this stage

Reproducible commands:

```sh
python3 spec/001-want-keep-mvp/tools/spec_tool.py render
python3 spec/001-want-keep-mvp/tools/spec_tool.py check
python3 -m unittest discover -s spec/001-want-keep-mvp/tools -p 'test_*.py'
git diff --check
```

The tool checks IDs, links, coverage, DAG, RU/EN presence, generated consistency and unique GitHub mappings. Independent review separately checks translation meaning, financial rules and future tasks. The final response records the actual latest check/publication results; this document does not treat future ACs as passing tests.

Issue publication uses one `want-keep-task: task-X.Y` marker, a fresh GET before creation, title/body readback and stop/read on ambiguous outcome. Backlog URLs appear only after verified readback. Self-contained RU/EN bodies do not depend on an unpublished documentation commit.

## Not run and why

Application build/unit/integration/E2E, provider live sessions, paid OpenAI evaluations, physical-device testing, provisioning and backup/restore runtime were not run: this stage delivers documentation/Issues, no runtime exists and financial credentials/keys were not connected.

Documentation readiness is not CI or runtime evidence; conditional Mac RPO does not promise an hourly copy while the device is off. All mandatory products remain in scope and backlog.

## Family amendment 2026-09-07

REQ-001–REQ-062, AC-001–AC-076 and all prior task IDs are retained with updated household-access semantics. Added REQ-063–REQ-076, AC-077–AC-093, task-1.6, task-2.8, task-2.9 and task-7.9; other dependencies include the household foundation. Current-stage checks cover documents; new ACs have not run in an application. BLK-01–BLK-06 also require two-account identity and safe reauthorization.

## Desktop refinement and animations 2026-09-07

REQ-055, AC-055/AC-075 and prior UI tasks are updated with unchanged IDs. Added REQ-077–REQ-087, AC-094–AC-104 and task-7.10–task-7.15; SCR-001–SCR-035 retained. Package includes 15 forms, 17 states, design/navigation and MOT-01–MOT-05. Mobile requirements replaced by laptop Chrome/Arc; UI font is Manrope. All runtime/visual/animation and user scenarios remain future work; contracts do not prove they passed.

Self review corrected AC-015’s conflict with shared receipt visibility and aligned FORM-03 with the existing account-ownership rule. AC-105 checks current owner, household account, API and retained history. The complete package is reviewed again before publication.

## Delivery of this stage

Complete RU/EN SDD package prepared: product, 87 requirements, 105 criteria, architecture/API/formulas, glossary/flows, six integrations, operations, design/MOT-01–MOT-05, 35 screens/15 forms/17 states, navigation and 67 tasks.

[67 GitHub Issues](https://github.com/pchkauu/want-keep/issues) created and read back: unique task IDs, exact RU/EN titles/bodies and another complete duplicate scan. All URLs are in the [backlog](backlog.en.md); last individual issue readback: `2026-09-06T22:29:07.841238+00:00`. Task bodies remain byte-identical to reviewed versions after links were added.

Avida self review: completed, pass, 0 remaining findings/questions, all 106 changed paths covered. Used 4 of 5 rounds: first interrupted by scope refinement; then corrected receipt access, account-form permission and English AI wording. Pre-publication content fingerprint: `sha256:b20d4f9fca9b5eb6e16a38a261e50d181aa5d1a3c9cd335eceb4cda206185c11`. Only GitHub mappings, rendered backlog links and this delivery report changed afterward; these were checked separately without changing task cards.

Checks: spec_tool check — pass; 10 documentation-tool regression tests — pass; REQ/AC/task/SCR graph, RU/EN presence, generated consistency, links and whitespace — pass. Independent reviewer roles checked semantic RU/EN parity. No commits, push or application implementation.

Application verdict remains Not Ready: BLK-01–BLK-10 above remain open. External contracts, six-platform live checks, AI eval, server costs/retention and the Ready plan remain future work. Chrome/Arc, user/animation acceptance and backup/restore runtime have not run because the app does not exist. Next executable stage: task-0.1–task-0.9 research and task-0.10 readiness review.

## Raiffeisen: task-0.2 research completion, 2026-09-07

[RU evidence](evidence/raiffeisen.md) / [EN](evidence/raiffeisen.en.md), [JSON](evidence/raiffeisen.samples.json), [XML](evidence/raiffeisen.camt053.sample.xml): RAIF-E01–E24 and RAIF-B01–B06. D-35 limits the integration to the entrepreneur current account; REQ-043/AC-043 and task-0.2/task-4.2 refined with IDs preserved. AC-070 on credit/savings terms removed only from Raif.

The owner completed registration and initial Refresh-token issuance. Direct Mac refresh grant returned HTTP 200 and changed the Refresh token; the set was atomically saved locally with 0600 in a 0700 directory. Two Mac account GETs returned HTTP 200, one stable UUID id and separate number. After separate owner approval, one VPS GET returned HTTP 200 and the same account; access/id tokens were passed through SSH into memory without installing credentials on the server. Tokens and original responses are not printed or committed.

Historical August flow: dryRun 200 → generation 202 → status completed → XML 200. An independent August 1–30 statement returned the same seven NtryRefs and Ntry elements. camt.053.001.08 contains two CRDT/five DBIT, all BOOK; Decimal reconciliation OPBD + movements = CLBD passed. JSON RUR explicitly maps to XML RUB. Actual completed/no-statements differ from OpenAPI COMPLETED/NO_STATEMENTS; original values retained. Initial strict status validation stopped download until the actual value was established; after the fix, the existing report was read without another generation.

Intraday for 2026-09-07 returned HTTP 404 with code=no-statements. Current/available/locked balance, exact FCHG semantics, full archive, late changes, reauth/second account and refresh lifecycle 30/180 days remain open. RAIF-B01/B05 closed; structural B02 part verified; B02/B03/B04/B06 handed to task-0.10. Research completed under README; task-4.2 and MVP remain Not Ready.

Previous-step infrastructure: DNS/TLS, nginx -t, HTTPS 204/503, HTTP 308 without query, unknown-path 404, unknown-SNI rejection and Certbot renewal simulation passed. Application callback still 503. Diagnostic Python scripts are research tools, not the Go connector; full Code Flow/household authorization and bank payments were not implemented.

Checks: 18 synthetic diagnostic-client tests, 14 spec_tool tests, render/check (87 REQ, 105 AC, 67 tasks, 35 screens, 67 GitHub mappings), RU/EN IDs/examples/links, JSON/XML and Decimal reconciliation of the synthetic sample — pass. Scanning artifacts for actual credentials/account details/owner name and git diff --check — pass. Shortening REQ-043/AC-043 preserved conditions and brought task-9.1 within the Issue length limit: 59,811 characters; generator limits unchanged. API checks above are live; guard/allowlist tests are synthetic. XSD, sandbox and full application end-to-end were not run; local review completed. Commit/push and GitHub readback are recorded in the delivery result in [Issue #2](https://github.com/pchkauu/want-keep/issues/2). Related task-0.10/task-4.2/task-9.1 are updated while retaining Not Ready; concurrent changes for other tasks are excluded from this result.

## Ozon: task-0.3 research completion, 2026-09-07

[RU evidence](evidence/ozon.md) / [EN](evidence/ozon.en.md): OZON-E01–E18, current debit-product matrix, five observed JSON read routes and [10 projections with synthetic values](evidence/ozon.samples.json) are retained. Two owner HARs were analyzed locally; originals are excluded from Git. History includes seven linked pages, 210 distinct lastOperationIds, 206 confirmed and four canceled rows. A transfer and commission share groupID but have distinct lastOperationId: groupID-only deduplication loses the commission. accountToken changes and is not a stable ID; the account number links the card and transaction details.

The account number and one purchase were read again after owner sign-in. All seven pages have a continuation: history completion and retention are unproven. Authenticated-session operation, autonomous hourly reads and a second external account were not tested. Separate available/locked and the refund's original purchase are absent from observed responses; these remain unknown, not zero or an invented link.

Research output is complete under the README rule; OZON-B01 is closed, the structural part of OZON-B02 is resolved, and OZON-B03 is removed by D-32. OZON-B02/B04/B05 remain within BLK-03, with closure verified by task-0.10 against this evidence. task-4.3 and the complete MVP remain Not Ready. Completing the GitHub Issue does not remove this gate or claim application ACs passed.

REQ-044/AC-044, task-0.3/task-4.3, integrations and traceability reflect D-32; IDs are unchanged. The Ozon link to AC-070 is removed: credit terms remain for other providers but are outside Ozon's current contract. Other research in the shared checkout is outside this task's publication.

Checks: spec_tool check — pass (87 REQ, 105 AC, 67 tasks, 35 screens, 67 GitHub mappings); 13 documentation-tool tests — pass; JSON, RU/EN IDs/meaning, cursor and commission relationships, projection privacy and git diff --check — pass. Local self review performed; the previous Avida pass applies to the previous package. HTTP replay, allowlist/importer, complete ACs and application runtime were not tested: the collector and application are not implemented. Commit/push and GitHub readback are recorded separately as delivery results in Issue #3.

## Aifory: task-0.5 research completion, 2026-09-07

[RU evidence](evidence/aifory.md) / [EN evidence](evidence/aifory.en.md): 19 observations/sources, D-33, selected RUB/USDT/ETH/USD card, synthetic scenarios and AIFORY-B01–B05. B01 closed, B02–B04 remain in BLK-05 under task-0.10, B05 deferred without blocking. Aifory credit/savings and other unused products are not required. ETH is included in REQ-002/REQ-003, valuation and rate-source research; REQ-046/AC-046 and tasks are updated without changing IDs.

UI reading does not prove structured API, automatic import, completeness, identity after reauth/second account or card lifecycle. Operator consent for automation is not confirmed. Research is complete under the README rule; task-4.5 and Ready remain blocked. Full AC-046/AC-041/AC-048/AC-079/AC-087 are not claimed as passed. Local spec, RU/EN/JSON and diff checks are separate from runtime.

Current changes received a local self review. Aifory links to AC-070/AC-071 credit, savings, trading and mining products were removed; AC-039 retains missing-rate checks. The previously recorded Avida pass concerns the earlier package.

## EMCD: task-0.6 research completion, 2026-09-07

[RU evidence](evidence/emcd.md) / [EN](evidence/emcd.en.md): D-34, EMCD-E01–E26, four included UI areas and [six synthetic scenarios](evidence/emcd.samples.json). USDT wallet, existing Grow, Plus/Light and P2P archive inspected. Mining has never been used; its history and other unused products do not block readiness.

Aggregates may include child balances; Grow distinguishes accrued/earned figures and capitalization/payout modes. A negative card row may be a decline with a separate fee. P2P lists round amounts; details establish direction rather than list currency order. Mining Pool API 1.3.0 does not establish reading these products. One Grow last UI page was reached, but full API traversal and other logs remain unverified.

EMCD-B01 is closed; EMCD-B02–B04 remain in BLK-06 under task-0.10. Research is complete under README; task-4.6 and MVP Not Ready. REQ-047/AC-047 and tasks changed with IDs preserved; AC-071 removed from EMCD, AC-070 retained for Grow. Shared credit, trading and mining features remain.

Current self review checks documents, RU/EN IDs/meaning and synthetic Decimal scenarios; the earlier Avida pass does not cover this research. Spec check, generator tests and diff review run on the final candidate before publication; outcomes and GitHub readback are recorded in Issue #6. API replay, application/collector, reauth/second account, hourly/allowlist and full ACs were not run: selected-product structured contracts and financial runtime are absent; the task-1.1 foundation exists.

Final-candidate checks on base `f12f215`: `make docs-check` — pass, 87 REQ / 105 AC / 67 tasks / 35 screens / 67 GitHub mappings; 14 generator tests — pass. Six Decimal scenarios, 26 paired observations/IDs/RU/EN links, JSON, scoped EMCD catalog changes and absence of private IDs — pass. Publication content was checked separately; unfinished changes from another research task were preserved.

## Bybit: task-0.4 research completion, 2026-09-07

[RU evidence](evidence/bybit.md) / [EN](evidence/bybit.en.md), [authenticated supplement](evidence/bybit-api.en.md): D-36, BYBIT-E01–E18 and twelve [synthetic scenarios](evidence/bybit.samples.json). Authorized RSA readOnly key, Funding balances and 357 ledger records over 89 days, Flexible positions/orders/yield, P2P list and both details read successfully. Ledger/replay checks pass within the sampled scope; no production connector exists.

BYBIT-B01/B02/B05 are closed for observed access. BYBIT-B03/B04 remain with task-0.10; VPS access with task-0.9, second-owner/revocation and production conformance with task-4.4. BYBIT-B06 and unused products do not block. Research is complete under README; task-4.4 and MVP remain Not Ready. Stable REQ/AC/task IDs and D-36 asset/product scope are unchanged by this evidence update.

Local self review covers the final Bybit diff, RU/EN meaning/IDs, read-only authority, exact decimals, source/enrichment separation, partial history and private-data exclusion. Corrected stale no-key/P2P/Fixed claims, missing hourly ID assumptions and the P2P quantization scenario. The previous Avida pass does not cover this research. Private Funding/Earn/P2P calls and sampled replay/arithmetic were verified; complete lifetime history, all status transitions, second-owner/revocation, hourly collector, bank matching, chosen VPS and production E2E were not run.

Authenticated-evidence candidate on base `2412b77`: `make docs-check` — pass, 14 generator tests — pass; 87 REQ / 105 AC / 67 tasks / 35 screens / 67 GitHub mappings. Twelve synthetic Decimal/identity/coverage scenarios, RU/EN IDs/routes, unchanged requirements/ACs, exactly three task updates and secret/private-identifier exclusion — pass. Local self review completed. Only Bybit evidence/contracts and associated documentation/tasks are changed; Raif and FX work is preserved. Publication and Issue readback are verified during delivery.
