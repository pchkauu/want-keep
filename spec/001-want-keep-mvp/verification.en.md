# Want Keep MVP readiness verification

Verdict: Not Ready

Date: 2026-09-06. This verdict concerns the complete specification's readiness for application implementation. It neither prevents agreed documentation/backlog delivery nor describes an implemented application.

## Artifact state

Prepared proposal, 87 REQ, 105 AC, architecture, contracts/formulas, glossary, Mermaid flows, integration matrix, operating constraints and 67 RU/EN cards. The catalog/generator preserves the same IDs/links in both languages. Requirements/ACs have task coverage; every implementation task depends on the task-0.10 gate.

`plan.md` is intentionally absent: SDD requires Ready before issuing an implementation plan. Research tasks exist in the full backlog; completing research with a negative result does not unblock an unavailable product.

## Blockers

| ID | Unknown | Closure owner | Required evidence |
| --- | --- | --- | --- |
| BLK-01 | Alfa retail read access/full coverage | task-0.1 | Verified contracts and per-product readback; securely supplied owner access |
| BLK-02 | Raif Russia retail read access/full coverage | task-0.2 | The equivalent for Raif |
| BLK-03 | Automatic Ozon Bank reading | task-0.3 | The equivalent for Ozon, without marketplace substitution |
| BLK-04 | Full Bybit Funding/Spot/Earn/P2P/futures, net/gross and permissions | task-0.4 | Each log/product verified separately |
| BLK-05 | Aifory wallet/payment/card/exchange contracts | task-0.5 | Complete product matrix and readback |
| BLK-06 | EMCD wallet/Coinhold/P2P/card/mining contracts | task-0.6 | Separate accrual/transfers/fees and full product coverage |
| BLK-07 | Free current/historical FX valuation and provider quotes | task-0.7 | All pairs/periods/fees/source policies or a decision on unavailability |
| BLK-08 | OpenAI models, measured quality/cost and request limits | task-0.8 | Financial-invariant evaluation with tokens/errors and selected versions/limits |
| BLK-09 | Concrete VPS/reachability/cost and Mac backup retention | task-0.9 | Dated ≤$40 estimate, reachability, retention/capacity and recovery design |
| BLK-10 | Full structured grace/accrual terms, exact API boundaries and XIRR solver | task-0.10 | Verified input contracts, algorithms/vectors, updated RU/EN and independent Ready review |

BLK-01–BLK-06 need separately and securely supplied owner access. None was supplied during this stage. Marketing pages, mocks or corporate API availability do not close these blockers.

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
