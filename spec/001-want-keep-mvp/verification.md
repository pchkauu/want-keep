# Проверка готовности Want Keep MVP

[English](verification.en.md)

**Verdict SDD: Ready for development.** Дата: 2026-09-07. Gate task: `task-0.10`.

Этот verdict разрешает реализацию по [plan.md](plan.md). Он не означает, что приложение реализовано, provider connectors допущены, production развёрнут или продуктовые AC пройдены.

## Полнота пакета

- 88 стабильных REQ, 106 AC, 67 task и SCR-001–SCR-035 связаны через [catalog.json](catalog.json).
- Генератор проверяет DAG, обязательный ancestry `task-0.10`, RU/EN, ссылки, GitHub mappings, размер Issue body и отсутствие private paths/secret-shaped strings.
- D-37–D-43 закрывают Alfa scope, readiness semantics, source identity, FX gaps/quotes, command retention, numeric XIRR и version-bound provider admission.
- [Итоговый evidence](evidence/task-0.10-readiness.md) содержит `BLK → решение → evidence → runtime gate` и синтетические сценарии.
- [Ready plan](plan.md) задаёт порядок, параллелизм, entry/exit gates и команды.

## Итог BLK-01–BLK-10

| BLK | Статус SDD | Решение | Что остаётся до runtime/приёмки |
| --- | --- | --- | --- |
| BLK-01 Alfa | Закрыт D-37/D-39 | Scope debit/current/savings/deposit/cashback; кредитка Alfa отложена; unknown/ambiguous fail closed | task-4.1 permission/fixture/identity/history/reauth/2 accounts/Alfa route |
| BLK-02 Raiffeisen | Закрыт D-39 | CAMT 1:N, scoped ID/fallback, revisions/reversals, CLBD/unknown balance rules | task-4.2 OAuth/full history/corrections/2 accounts/conformance |
| BLK-03 Ozon | Закрыт D-38/D-39 | Synthetic HAR projection достаточна для дизайна; rotating token/group не identity | task-4.3 session permission/lifecycle/history end/2 accounts |
| BLK-04 Bybit | Закрыт D-39 | Route IDs, candidate-only links, hourly tuple collision policy | task-4.4 precision/history/rotation/revocation/2 accounts |
| BLK-05 Aifory | Закрыт D-38/D-39 | Scope и safe boundary фиксированы без вымышленных Flutter fields | task-4.5 permission/structured fixtures/card lifecycle/reauth/2 accounts |
| BLK-06 EMCD | Закрыт D-38/D-39 | Scope, namespaces и unknown/collision behavior фиксированы | task-4.6 structured fixtures/balance/Grow/card/P2P/reauth/2 accounts |
| BLK-07 FX | Закрыт D-40 | >365d `valuation_unavailable`; incomplete provider quote `quote_unavailable` | task-6.1 Demo key/quota/attribution/live adapters |
| BLK-08 OpenAI | Закрыт research | Model/schema/cost/failure contract выбран | task-5.x/task-8.1 runtime gateway/authz/budget |
| BLK-09 Hosting | Закрыт D-38 для SDD | Target profile/budget выбраны; operations gates отделены | task-8.1–8.3 provisioning/hardening/load/backup/restore |
| BLK-10 Формулы/API | Закрыт D-39/D-41/D-42/D-43 | Identity, retention, deterministic XIRR и version-bound admission закреплены | task-1.2/1.3/3.3/4.x/6.4/8.1 executable contracts/tests |

Отдельные BLK Issues не создаются: фундаментальных SDD-блокеров не осталось. Runtime gates остаются в существующих task-4.x/task-8.x и не скрыты.

## Проверки task-0.10

Обязательный локальный набор:

```sh
make docs-check
make check
git diff --check
```

Проверяются также:

- RU/EN semantic parity для D-37–D-43, provider matrices, contracts, operations, evidence и plan;
- полная трассировка REQ → AC → implementation/verification task;
- отсутствие credentials, raw response bodies, реальных реквизитов и local paths;
- synthetic identity/replay/gap/unknown/collision/CAMT/FX/XIRR/retention outcomes из evidence;
- diff против свежего `origin/docs/want-keep-mvp-sdd` и независимый Avida review frozen candidate.

Фактические команды фиксируются в PR/Issue. Неизменяемый review outcome с fingerprint/commit SHA публикуется для точного committed head во внешнем PR check/comment и Issue #10; [task-0.10 evidence](evidence/task-0.10-readiness.md) остаётся частью проверенного кандидата и не делает fingerprint самоссылочным.

## Граница доказательств

Документационно подтверждены решения и ожидаемые безопасные результаты. Авторизованные Chrome-наблюдения Alfa/Aifory/EMCD подтверждают доступность выбранных областей, но не заменяют structured response fixtures. Ozon HAR, Bybit API и Raiffeisen API/CAMT используются только через опубликованные sanitized/synthetic projections.

Не запускались как часть task-0.10: финансовый runtime, provider connector suites, второй аккаунт, reauth/revocation, product E2E, production deployment, real Chrome/Arc application UI и backup/restore. Основание: задача меняет SDD, а владельцы этих проверок — task-1.x–task-9.x. Их отсутствие не выдано за pass.
