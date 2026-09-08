# Task-2.8 — семейное распределение расходов

Реализован backend/API распределения семейных расходов. Исходная база: `cedbe3ba8384071e0e958167d37899ade557c0bc`; ветка: `feat/task-2.8-family-allocation`; task-1.6 и task-2.6 включены. [Контракт](../contracts.md#task-28--семейное-распределение-расходов) фиксирует один семейный финансовый факт и точный персональный разрез. Новых зависимостей нет; миграции 001–015 и production не менялись.

Ledger сохраняет immutable allocation snapshot по `MembershipID`, отдельно от payer, account owner и actor. Поддержаны `personal|shared`, точные суммы, доли и equal. Largest remainder работает на исходной точности каждого актива и использует `MembershipID` как стабильный tie-break. Распределение покрывает позиции, fallback principal, комиссии и проценты в native-активах; principal внутреннего перевода или обмена не становится расходом. Unknown сохраняется в unallocated и не превращается в ноль или 50/50.

`allocation/domain/application` владеет merchant/category rules. Условия одного правила используют AND, меньшее priority важнее. Совместимые равноприоритетные rules сохраняют все ссылки, разные результаты дают `rule_conflict`. Preview не пишет данные. Новая revision правила влияет только на новые факты. Source import применяет merchant rule только через подтверждённый alias; provider ID не подменяет внутренний merchant/category ID. Category rule ждёт доверенной классификации. Correction, compound decision, selective undo, field protection и source merge используют самостоятельное поле `allocation`; source не стирает пользовательское решение. Hash review-запроса включает типизированное распределение. Matching сохраняет одного носителя семейного и персонального эффекта и переводит waiting/non-carrier/internal principal в `not_applicable`.

Миграция 016 добавляет rules/revisions/shares и immutable transaction/item snapshots с составными household FK, точными `NUMERIC`, immutable triggers и минимальными grants. Старые операции получают только доказуемое `unresolved` или `not_applicable`, без выдуманных получателей. OpenAPI активирует transaction allocations и rule CRUD/preview; Go/TypeScript генерируются из одного источника. Mutation-маршруты используют session actor, Origin/CSRF, idempotency, expected revision и command recovery.

## Матрица проверок

Обязательные команды: `make check`; `make test-integration AREA=all`; `make test-family-allocation-race`; затронутые race suites; `make test-integration AREA=privacy`; `git diff --check`. PostgreSQL 17.11 закреплена digest; отсутствие БД завершает suite ошибкой. Family-allocation integration/race включены в CI. Точные результаты опубликованного кандидата фиксируются в PR и Issue.

Suite проверяет RUB, USD, USDT, USDC, BTC и ETH с произвольной точностью; 50/50, 60/40, exact amounts, deterministic remainder и mixed receipt `1000 → A 400 + B 600`. Проверяются item override, полностью unresolved item split и сохранение его явной причины при correction/matching refresh, перерасчёт share-based и amount-based purchase fallback без превращения производных item snapshots в явные overrides, fee в третьем активе и повтор одного `MembershipID` в разных native-активах. Rule precedence/conflict/preview, сохранение `rule_conflict` в ручном и импортированном факте и архивация правила после архивации его merchant/category также покрыты. Дополнительно проверяются подтверждённый merchant alias без доверия provider ID, игнорирование allocation из provider payload, `not_applicable` для импортированного дохода и rule allocation только расходной комиссии перевода, отсутствие ретроактивного изменения, correction/undo/source protection, полный hash review и один эффект при matching нескольких evidence.

PostgreSQL-сценарии проверяют concurrent revisions двух участников, replay, rollback, потерю ответа после commit, семейную изоляцию, session-bound cursor, CSRF, command visibility, migration-over-015, неизменяемость истории и grants непривилегированной роли. Распределение не меняет проводки или account projection.

## Границы критериев

| Критерии | Доказуемая часть task-2.8 | Дальнейшая проверка |
| --- | --- | --- |
| AC-078/079 | Оба активных участника меняют семейный расход; actor/payer/owner/beneficiary разделены, чужая семья скрыта | Личные цели/план, банковский MFA и UI |
| AC-080/081 | Один семейный факт, точные персональные суммы, unallocated и смешанный чек 400/600 | Бюджетные отчёты и AI-уточнение |
| AC-086/093 | Revision/replay, compound undo и matching evidence не дублируют персональный эффект | Чат/receipt lifecycle и браузерная приёмка |
| AC-065/091 | Snapshot и исторические rule references готовы для пропорционального возврата | Полный refund lifecycle выполняет task-2.7 |

SDD остаётся **Ready for development**. UI, бюджеты, реальные чеки/OpenAI, банковский IO, возвраты и production этой задачей не проверяются.
