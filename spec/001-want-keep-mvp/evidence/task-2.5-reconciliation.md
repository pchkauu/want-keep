# Task-2.5 — сверка журнала с балансом источника

Реализован backend/API сверки импортных счетов. База реализации: `c37d9ed255b01c155d92f3163736bcf57193a25b`; ветка: `feat/task-2.5-balance-reconciliation`; task-2.3 включена, task-2.4 не является зависимостью. [Контракт](../contracts.md#task-25--сверка-журнала-с-балансом-источника) фиксирует расчёт на `sourceAsOf`, ограниченный replay и явные корректировки. Production не менялся.

Домен сверки хранит lifecycle, результат, replay и четыре независимых компонента. Application-сервис строит историческую проекцию, дедуплицирует replay, координирует admission и создаёт разрешённые adjustments через публичный контракт ledger. Storage сохраняет неизменяемые source observations и revisions миграции 010. Delivery использует generated DTO, общий session/Origin/CSRF guard, подписанные курсоры и command recovery.

## Матрица проверок

Обязательные команды: `make check`; `make test-integration AREA=reconciliation` и `make test-reconciliation-race`; integration/race для audit, ledger, accounts, storage, identity и household; `make test-integration AREA=privacy`; `git diff --check`. PostgreSQL 17.11 закреплена digest; отсутствие БД завершает reconciliation suite ошибкой. Конкретные результаты локального кандидата и CI фиксируются в PR.

Reconciliation suite проверяет точность RUB, USD, USDT, USDC, BTC и ETH, независимые owned/available/locked/debt, unknown/partial/stale и историческую границу lifecycle. Opening 5000, расход 500 и source 4500 дают balanced без дохода. Source 1000 против ledger 900 создаёт один replay максимум за 90 дней; после completed/unavailable сервер создаёт adjustment 100 без income/expense. Debt корректируется отдельно, available/locked получают явный отказ.

Проверяются новая observation и superseded lifecycle, повторная оценка после revision, исключения и undo, отсутствие события для идентичного результата, admission/reauth failure без фиктивного job, семейная изоляция и неизменность source evidence. Replay сохраняет исходные binding/admission revision/connection generation. Completion выполняется внутри точной admitted `CommitPage`-транзакции, а терминальная ошибка провайдера — через эквивалентный admitted `CommitFailure`; household-only outcome отклоняется. Delayed source posting использует подтверждённый `postedAt`, reversal с неизвестным временем перехода остаётся incomplete, а многопроводочная операция применяется один раз. Resolution проверяет фактический ledger-эффект adjustment на все четыре компонента, включая связанное изменение available. HTTP-проверки охватывают фильтры, session-bound pagination, CSRF, безопасный 404, idempotent command recovery и изменённый payload. Миграционный тест обновляет базу 009 → 010, повторно применяет миграции и подтверждает неизменяемость истории под непривилегированной ролью.

## Границы критериев

| Критерии | Доказуемая часть task-2.5 | Дальнейшая проверка |
| --- | --- | --- |
| AC-004/005 | Точные source/ledger/difference на `sourceAsOf`, историческая проекция и отдельная quality | Полный импорт продуктов и экран счёта |
| AC-013 | Новая observation и revisions переоценивают сверку; идентичный результат не дублируется | Сквозная сверка с реальным источником |
| AC-040/041 | Replay ограничен 90 днями, дедуплицирован и содержит admission/generation; unavailable объясняется | Provider IO, повторная авторизация и UI действий |
| AC-058 | Явный owned/debt adjustment атомарен, не является доходом/расходом и не меняет источник | Пользовательская приёмка SCR-012 в Chrome/Arc |

SDD остаётся **Ready for development**. Реальные адаптеры, provider IO, browser UI и production здесь не проверяются. Полная операционная готовность требует последующих задач и сквозной приёмки.
