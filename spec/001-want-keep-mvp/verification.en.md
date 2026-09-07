# Want Keep MVP readiness verification

[Русский](verification.md)

**SDD verdict: Ready for development.** Date: 2026-09-07. Gate task: `task-0.10`.

This verdict permits implementation under [plan.en.md](plan.en.md). It does not mean the application is implemented, provider connectors are admitted, production is deployed or product ACs have passed.

## Package completeness

- 88 stable REQ, 106 AC, 67 tasks and SCR-001–SCR-035 are linked through [catalog.json](catalog.json).
- The generator validates the DAG, mandatory `task-0.10` ancestry, RU/EN, links, GitHub mappings, Issue body size and absence of private paths/secret-shaped strings.
- D-37–D-43 resolve Alfa scope, readiness semantics, source identity, FX gaps/quotes, command retention, numeric XIRR and version-bound provider admission.
- [Final evidence](evidence/task-0.10-readiness.en.md) contains `BLK → decision → evidence → runtime gate` and synthetic scenarios.
- The [Ready plan](plan.en.md) defines order, parallelism, entry/exit gates and commands.

## BLK-01–BLK-10 outcome

| BLK | SDD status | Decision | Remaining runtime/acceptance work |
| --- | --- | --- | --- |
| BLK-01 Alfa | Resolved by D-37/D-39 | Debit/current/savings/deposit/cashback scope; Alfa credit card deferred; unknown/ambiguous fails closed | task-4.1 permission/fixture/identity/history/reauth/2 accounts/Alfa route |
| BLK-02 Raiffeisen | Resolved by D-39 | CAMT 1:N, canonical cross-report fingerprint, atomic optional-ID aliases, revisions/reversals, CLBD/unknown balance rules | task-4.2 OAuth/full history/corrections/2 accounts/conformance |
| BLK-03 Ozon | Resolved by D-38/D-39 | Synthetic HAR projection is sufficient for design; rotating token/group is not identity | task-4.3 session permission/lifecycle/history end/2 accounts |
| BLK-04 Bybit | Resolved by D-39 | Route IDs, candidate-only links, hourly tuple collision policy | task-4.4 precision/history/rotation/revocation/2 accounts |
| BLK-05 Aifory | Resolved by D-38/D-39 | Scope and safe boundary are fixed without invented Flutter fields | task-4.5 permission/structured fixtures/card lifecycle/reauth/2 accounts |
| BLK-06 EMCD | Resolved by D-38/D-39 | Scope, namespaces and unknown/collision behavior are fixed | task-4.6 structured fixtures/balance/Grow/card/P2P/reauth/2 accounts |
| BLK-07 FX | Resolved by D-40 | >365d `valuation_unavailable`; incomplete provider quote `quote_unavailable` | task-6.1 Demo key/quota/attribution/live adapters |
| BLK-08 OpenAI | Resolved by research | Model/schema/cost/failure contract selected | task-5.x/task-8.1 runtime gateway/authz/budget |
| BLK-09 Hosting | Resolved by D-38 for SDD | Target profile/budget selected; operational gates separated | task-8.1–8.3 provisioning/hardening/load/backup/restore |
| BLK-10 Formula/API | Resolved by D-39/D-41/D-42/D-43 | Identity, retention, deterministic XIRR and version-bound admission fixed | task-1.2/1.3/3.3/4.x/6.4/8.1 executable contracts/tests |

No separate BLK Issues are created because no fundamental SDD blockers remain. Runtime gates remain explicit in existing task-4.x/task-8.x work.

## task-0.10 checks

Required local set:

```sh
make docs-check
make check
git diff --check
```

Also verify:

- RU/EN semantic parity for D-37–D-43, provider matrices, contracts, operations, evidence and plan;
- complete REQ → AC → implementation/verification task traceability;
- no credentials, raw response bodies, real account details or local paths;
- synthetic identity/replay/gap/unknown/collision/CAMT/FX/XIRR/retention outcomes from evidence;
- diff against fresh `origin/docs/want-keep-mvp-sdd` and independent Avida review of a frozen candidate.

Actual commands are recorded in the PR/Issue. The immutable review outcome with fingerprint/commit SHA is published for the exact committed head in an external PR check/comment and Issue #10; [task-0.10 evidence](evidence/task-0.10-readiness.en.md) remains part of the reviewed candidate and does not make the fingerprint self-referential.

## Evidence boundary

Documentation confirms decisions and expected safe outcomes. Signed-in Chrome observations for Alfa/Aifory/EMCD confirm selected areas are available but do not replace structured response fixtures. Ozon HAR, Bybit API and Raiffeisen API/CAMT are used only through published sanitized/synthetic projections.

Not run as part of task-0.10: financial runtime, provider connector suites, second account, reauthentication/revocation, product E2E, production deployment, real Chrome/Arc application UI and backup/restore. Reason: this task changes the SDD, while task-1.x–task-9.x own those checks. Their absence is not reported as a pass.
