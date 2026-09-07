# task-0.10 — итог Ready-gate

[English](task-0.10-readiness.en.md)

Дата: 2026-09-07. Verdict: **Ready for development**.

Этот verdict относится к полноте решений SDD. Приложение, коннекторы, deployment и продуктовые AC не проверялись. Неподтверждённые внешние свойства переведены в fail-closed контракты и явные entry/runtime gates владельцев task-4.x/task-8.x.

## Матрица блокеров

| BLK | Решение task-0.10 | Evidence | Runtime gate |
| --- | --- | --- | --- |
| BLK-01 Alfa | D-37 ограничивает scope debit/current/savings/deposit/cashback. D-39 запрещает UI/time/amount identity; unknown/ambiguous не проводятся. | [Alfa](alfa.md), авторизованная Chrome-вкладка без публикации значений | task-4.1: permission, structured fixture, IDs, lifecycle, два аккаунта, reauth, Alfa route |
| BLK-02 Raiffeisen | CAMT 1:N, canonical cross-report fingerprint без amount/time, atomic optional-ID aliases, collision policy, revisions/reversals; statement ID — provenance; last confirmed CLBD с датой, остальные balances/fees unknown. | [Raiffeisen](raiffeisen.md), synthetic JSON/XML | task-4.2: OAuth lifecycle, full history, corrections, second account, conformance |
| BLK-03 Ozon | Готовая synthetic HAR projection принята для дизайна; accountToken/groupID не identity. | [Ozon](ozon.md), sanitized projections | task-4.3: session permission/lifecycle, history end, second account, reauth |
| BLK-04 Bybit | Route namespaces, candidate-only cross-log matches, hourly fallback и collision policy закреплены. | [Bybit](bybit.md), [RSA API](bybit-api.md) | task-4.4: precision/history, second account, rotation/revocation, conformance |
| BLK-05 Aifory | D-33 scope и fail-closed boundary закрывают дизайн без вымышленных Flutter fields. | [Aifory](aifory.md), авторизованная Chrome-вкладка | task-4.5: permission, structured fixtures, identity/history/card lifecycle, reauth |
| BLK-06 EMCD | D-34 scope, раздельные namespaces и unknown/collision policy закреплены. | [EMCD](emcd.md), авторизованная Chrome-вкладка, synthetic scenarios | task-4.6: structured fixtures, balance/card/Grow/P2P lifecycle, second account |
| BLK-07 FX | D-40 принимает `valuation_unavailable` старше 365 дней и `quote_unavailable` без provider quote. | [FX](fx.md) | task-6.1: Demo key/quota/attribution, live rates and cache behavior |
| BLK-08 OpenAI | Выбор модели, strict schema, cost/failure boundary уже закрыты исследованием. | [OpenAI](openai.md) | task-5.x/task-8.1: gateway, authz, budget and production health |
| BLK-09 Hosting | Конфигурация и бюджет выбраны; provisioning/conformance отделены от SDD. | [Hosting](hosting.md) | task-8.1–8.3 и provider gates: hardening, load, invoice, backup/restore |
| BLK-10 Формулы/API | D-39 identity, D-41 retention, D-42 numeric XIRR и D-43 version-bound admission делают решения детерминированными. | [Contracts](../contracts.md), этот отчёт | task-1.2/1.3/3.3/4.x/6.4/8.1: executable contracts and tests |

## Нормализованные решения

- Source key: `householdId + provider + stableExternalAccountId + productOrLogNamespace + providerRecordId`.
- Connection/session/cursor/job — provenance. Amount/time/text не identity.
- Missing provider ID допускает только документированный immutable composite. Collision → `source_ambiguous`, evidence + clarification, без проводки.
- Gap → `source_partial`; missing balance/fee/status остаётся unknown.
- Terminal command detail: 90 дней после исхода; unresolved: до сверки + 90 дней; tombstone с `commandId` живёт всё unresolved-состояние и 400 дней после terminal/reconciled outcome; recent: 30 дней terminal + все unresolved; expired detail → HTTP 410 `command_expired`.
- FX gap старше 365 дней → `valuation_unavailable`; неполная platform quote → `quote_unavailable`.
- XIRR: Actual/365, same-day aggregation, оба знака, одна смена знака, fractional power в 50-digit HALF_EVEN decimal с NPV error bound `1e-24`, bisection от `-1 + 1e-12` до `1 000 000`, solver tolerance `1e-12`, 512 iterations.
- Provider admission: aggregate/repository — application boundary `backend/internal/connections/admission/`, storage adapter — task-1.3; server-owned exact binding environment/build/contract/allowlist/config/permission; atomic task-4.x provider evidence + task-8.x host evidence; stale/missing binding → `provider_not_admitted` до job/collector IO.

## Синтетические проверки контракта

| Сценарий | Ожидаемый результат |
| --- | --- |
| Тот же provider ID и payload пришёл повторно | Одна source revision, одна финансовая проводка |
| Тот же provider key, другой payload | Новая revision либо `source_ambiguous` по provider policy; без второго эффекта до решения |
| Два факта совпали по сумме и секунде | Не объединяются без общего provider ID/доказанной связи |
| История начинается позже requested date | `source_partial`, coverage gap и opening-balance question; не нулевой прошлый период |
| Fee/balance отсутствует | Typed unknown; не `0` и не spendable |
| Bybit hourly tuple повторился с отличным payload | Обе evidence revisions, `source_ambiguous`, нет credit |
| CAMT entry содержит два transaction details | Один entry, два связанных detail, проводки по доказанной семантике без дублирования entry amount |
| CAMT correction/reversal | Новая revision/correction link; оригинал не переписывается |
| Один CAMT-факт пришёл без optional ID в 052 и с NtryRef только в 053 | Один canonical `camtCrossReportFingerprint`; NtryRef добавлен alias к исходной записи, второй проводки нет |
| CAMT alias указывает на другой fingerprint либо fingerprint совпал у разных фактов | Оба evidence сохранены; `source_ambiguous`, новая проводка не создаётся |
| camt.052 недоступен | Последний CLBD с `asOf`; available/locked/fee unknown |
| Crypto history старше 365 дней | Native amount сохранён, `valuation_unavailable` |
| Platform quote без fee/spread coverage | `quote_unavailable`; CBR/CoinGecko не подставляются как executable quote |
| `-1000`, затем `+1100` через 365 дней | XIRR `0.100000000000` |
| `-1000`, затем `+1050` через 182 дня | XIRR `0.102795595422` по fixed decimal ln/exp policy |
| Root равен rLow / выше rHigh | Boundary root принимается; root выше `1 000 000` даёт `unavailable` |
| Нет смены знака; `-100,+230,-132`; same-day net zero | Объяснённый `unavailable` |
| Terminal command старше 90, tombstone моложе 400 дней | `command_expired`; same key/hash не выполняется повторно, другой hash отклонён |
| Unresolved command старше 90 дней | Остаётся доступной до reconciliation; входит в `/commands/recent` |
| Admission относится к старому build/allowlist/config либо revoke гоняется с enqueue | Atomic invalidate/check+enqueue и collector recheck дают `provider_not_admitted`; job/provider IO/source record/проводка не создаются по stale binding |

## Граница доказательств

Подтверждены структура каталога, парность RU/EN, ссылки REQ → AC → task, решения D-37–D-43 и синтетические expected outcomes. Авторизованные Chrome-вкладки подтвердили доступность выбранных разделов Alfa/Aifory/EMCD, но response bodies не экспортировались и stable fields не объявлены доказанными. Приватные HAR/ключи/ответы не публикуются.

Не подтверждены: runtime-код финансового приложения, provider permission для production, второй аккаунт, reauth/revocation, полный history readback, target-host conformance, backup/restore и product/E2E AC. Эти проверки перечислены в [plan.md](../plan.md) и не меняют Ready SDD.

## Review

Этот файл входит в проверяемый кандидат и намеренно не содержит собственного fingerprint или результата. Publication gate требует Avida `pass` для точного committed head; неизменяемый результат с fingerprint/commit SHA фиксируется в PR review/check и комментарии Issue #10. Любое изменение файлов после review требует нового fingerprint и review.
