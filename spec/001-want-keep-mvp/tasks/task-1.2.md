<!-- want-keep-task: task-1.2 -->
# task-1.2 — Определить денежные типы и API-контракт / Define money types and API contract

## RU

Закрепить точность денег и типы публичных границ до адаптеров и UI.

**Состояние:** Основа task-1.2 согласована с контрактом версии 10: D-41 retention/recovery и D-43 admission проверяются на уровне domain/DTO. SDD Ready for development; runtime AC остаются у следующих задач.

**Зависимости:** `task-1.1`.

**Тип:** `implementation`.

### Изменение и контракты

Реализовать Money/Asset/Rate/календарные типы, отдельные knownness/coverage/freshness, семейную область и command transitions. apd v3.2.3 закрыт внутри money/domain; десятичные строки до 256 символов без float или display truncation. Материализовать OpenAPI 3.0.3 и воспроизводимую генерацию Go/TypeScript, явные boundary converters, безопасные ошибки версии 1, объяснимые read models и command status/recent. Определить Money/Asset/Rate/Time/coverage и версионированные состояния `source_partial`, `source_ambiguous`, `valuation_unavailable`, `quote_unavailable`, `command_expired`, `provider_not_admitted` по contracts.md. Деньги передавать десятичными строками и валидировать на первой границе; домен не импортирует generated DTO. Добавить explainable read models, command status/recent API и connection `deploymentGate.status=pending|admitted|blocked`. Публичный контракт server-owned admission связывает environment, adapter/collector build digests, contract, allowlist, non-secret config и operator-permission revisions; sync разрешён только при совпадении текущего binding и `admitted`, иначе collector не запускается. Эта задача владеет transport/read model, а aggregate, repository и application transitions реализует task-1.3. Terminal detail хранится 90 дней после исхода, unresolved — до сверки плюс 90 дней; tombstone с `commandId`, scope, key/hash и outcome живёт всё unresolved-состояние и 400 дней после terminal/reconciled outcome. `/commands/recent` отдаёт 30 дней terminal и все unresolved; истёкшая detail возвращает `command_expired`, не разрешая повторный эффект по живому tombstone.

### Границы изменений

- `backend/internal/money/`
- `backend/internal/calendar/`
- `backend/internal/household/`
- `backend/internal/reporting/`
- `backend/internal/commands/`
- `backend/internal/connections/`
- `backend/internal/delivery/http/`
- `api/`
- `scripts/generate-openapi.sh`
- `web/src/api/`
- `spec/001-want-keep-mvp/evidence/task-1.2-domain-api.md`

Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-002:** Учёт поддерживает RUB, USD, USDT, USDC, BTC и ETH; наличные, банковские деньги и платформенные кошельки различаются счетами.
- **REQ-003:** Общую валюту отображения можно переключать между RUB, USD, USDT, USDC, BTC и ETH.
- **REQ-039:** Отсутствующие курсы и неподдерживаемые активы не превращаются в нулевые суммы или условный паритет USD/USDT/USDC.
- **REQ-059:** Денежные расчёты используют точную арифметику и явные правила округления на границах.
- **REQ-062:** Архитектура использует Go/PostgreSQL, React/TypeScript/Vite и отдельный Playwright-сборщик с зависимостями к домену.
- **REQ-063:** Пользователь, семья и членство моделируются отдельно; ограничение двух участников задаётся конфигурацией.
- **REQ-065:** Принадлежность счёта, владелец внешнего аккаунта, автор записи и принадлежность расхода являются отдельными признаками.
- **REQ-076:** Семейная область проверяется для API, файлов, AI, фоновых задач и внешних ID независимо от присланных actor/owner.
- **REQ-088:** Синхронизация провайдера разрешена только актуальным server-side admission, связанным с проверенными версиями адаптера, контракта, allowlist, конфигурации и окружения.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

#### AC-002

- **Дано:** Созданы RUB наличные 1 000, RUB банк 2 000, USD наличные 10, USDT 20, USDC 12.000000000123, BTC 0.001 и ETH 0.001234567891.
- **Когда:** Владелец открывает счета.
- **Тогда:** Показаны семь отдельных счетов с исходными активами и точными остатками; дробные остатки не обрезаются до точности UI или заказа провайдера. RUB суммируется только в соответствующем срезе.
- **Уровень:** `integration`.

#### AC-003

- **Дано:** Для всех необходимых пар есть актуальная оценка.
- **Когда:** Участник переключает RUB на USD, USDT, USDC, BTC и ETH.
- **Тогда:** Меняется эквивалент итогов, исходные суммы операций и счетов сохраняются.
- **Уровень:** `end-to-end`.

#### AC-039

- **Дано:** В источнике есть неподдерживаемый USDC.E; для USDT/USD и USDC/USD отсутствуют курсы.
- **Когда:** Строится общая оценка.
- **Тогда:** Исходные данные сохранены, покрытие оценки обозначено неполным; нет скрытого нуля или автоматического курса 1:1. USDC.E не объединён с USDC по похожему символу.
- **Уровень:** `integration`.

#### AC-059

- **Дано:** Есть дробные BTC, USDT, процентное начисление и распределение чека.
- **Когда:** Данные проходят API, базу и повторный расчёт.
- **Тогда:** Исходная точность не теряется; JSON-суммы не проходят binary float; распределения сходятся точно, округление отображения не меняет журнал.
- **Уровень:** `unit+contract`.

#### AC-062

- **Дано:** Создана структура приложения и контракты компонентов.
- **Когда:** Проверяются зависимости и публичные интерфейсы.
- **Тогда:** Домен не импортирует HTTP, SQL, UI, OpenAI SDK или браузерные типы; адаптеры маппят внешние модели; сборщик не владеет финансовыми решениями.
- **Уровень:** `static`.

#### AC-077

- **Дано:** Два пользователя состоят в одной семье.
- **Когда:** Проверяются схема, авторизация и ограничение членства.
- **Тогда:** Нет полей partner1/partner2 и ветвлений по конкретным пользователям; роли и принадлежность отделены от личности. Выход, замена и новые роли не реализованы.
- **Уровень:** `integration`.

#### AC-079

- **Дано:** A и B имеют разные аккаунты одного провайдера и общий счёт; B заносит покупку A со счёта B.
- **Когда:** Выполняются ввод, импорт обоих аккаунтов и повторное подключение того же внешнего аккаунта.
- **Тогда:** Разные аккаунты не сливаются; повторный источник не удваивает остатки. Плательщик, автор и получатель расхода сохраняются независимо. Неустановленное совпадение блокирует новый учёт до уточнения.
- **Уровень:** `integration`.

#### AC-090

- **Дано:** В тестах созданы две изолированные семьи; запрос или задача подменяет householdId/actor/resourceId.
- **Когда:** Проверяются чтение файла, импорт, исправление, поиск AI и дедупликация.
- **Тогда:** Чужие объекты недоступны и не объединяются; сервер берёт principal из сессии или проверенного контекста задания. Отказ не раскрывает чужое содержимое.
- **Уровень:** `integration`.

#### AC-106

- **Дано:** Подключение авторизовано, но provider/host gate неполон либо прошлый admission относится к другой версии binding.
- **Когда:** Участник или scheduler запрашивает sync, либо меняются build, contract, allowlist, config, permission или environment.
- **Тогда:** Сервер возвращает `provider_not_admitted`, collector не запускается и проводок нет. Только admission service ставит `admitted` после provider evidence task-4.x и host evidence task-8.x для точного binding; любое расхождение снова закрывает sync.
- **Уровень:** `integration+security`.

### Проверка результата

```sh
make check
```

Точные synthetic round trips и распределения, отклонение неверных значений, проверяемые семейные/command инварианты; OpenAPI валиден, Go/TypeScript output воспроизводим. Это не доказательство auth/storage/product runtime.

Команды make реализованы. Проверяются Go 1.26.5 и Node 24.19.0; OpenAPI tooling использует изолированный TS 5.9.3, web — TS 6.0.3. Live banking, БД, deploy и browser E2E вне этой задачи.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Establish money precision and public boundary types before adapters and UI.

**Status:** The task-1.2 foundation aligns with contract version 10: D-41 retention/recovery and D-43 admission are checked at domain/DTO level. The SDD is Ready for development; runtime ACs remain with subsequent tasks.

**Dependencies:** `task-1.1`.

**Kind:** `implementation`.

### Change and contracts

Implement Money/Asset/Rate/calendar types, separate knowledge/coverage/freshness, household scope and command transitions. Encapsulate apd v3.2.3 in money/domain; decimal strings up to 256 characters without float or display truncation. Materialize OpenAPI 3.0.3 and reproducible Go/TypeScript generation, explicit boundary converters, safe version-1 errors, explainable read models and command status/recent. Define Money/Asset/Rate/Time/coverage and versioned `source_partial`, `source_ambiguous`, `valuation_unavailable`, `quote_unavailable`, `command_expired` and `provider_not_admitted` states from contracts.en.md. Transport money as decimal strings and validate at the first boundary; the domain must not import generated DTOs. Add explainable read models, command status/recent APIs and connection `deploymentGate.status=pending|admitted|blocked`. The public server-owned admission contract binds environment, adapter/collector build digests, contract, allowlist, non-secret configuration and operator-permission revisions; sync is permitted only when the current binding matches `admitted`, otherwise the collector never starts. This task owns the transport/read model; task-1.3 implements the aggregate, repository and application transitions. Terminal detail remains for 90 days after outcome, unresolved commands through reconciliation plus 90 days; a tombstone with `commandId`, scope, key/hash and outcome lives throughout unresolved state and for 400 days after terminal/reconciled outcome. `/commands/recent` returns 30 days of terminal commands and all unresolved commands; expired detail returns `command_expired` without permitting a repeated effect while the tombstone is live.

### Change boundaries

- `backend/internal/money/`
- `backend/internal/calendar/`
- `backend/internal/household/`
- `backend/internal/reporting/`
- `backend/internal/commands/`
- `backend/internal/connections/`
- `backend/internal/delivery/http/`
- `api/`
- `scripts/generate-openapi.sh`
- `web/src/api/`
- `spec/001-want-keep-mvp/evidence/task-1.2-domain-api.md`

Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-002:** Accounting supports RUB, USD, USDT, USDC, BTC and ETH; cash, bank money and platform wallets are separate accounts.
- **REQ-003:** The reporting currency can switch among RUB, USD, USDT, USDC, BTC and ETH.
- **REQ-039:** Missing rates and unsupported assets never become zero amounts or assumed USD/USDT/USDC parity.
- **REQ-059:** Money calculations use exact arithmetic and explicit boundary rounding rules.
- **REQ-062:** Architecture uses Go/PostgreSQL, React/TypeScript/Vite and a separate Playwright collector with dependencies pointing toward the domain.
- **REQ-063:** User, household and membership are separate models; the two-member limit is configured.
- **REQ-065:** Account ownership, external-account owner, record author and expense attribution are distinct dimensions.
- **REQ-076:** Household scope is checked for APIs, files, AI, jobs and external IDs independently of supplied actor/owner fields.
- **REQ-088:** Provider sync is allowed only by a current server-side admission bound to verified adapter, contract, allowlist, configuration and environment revisions.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

#### AC-002

- **Given:** Accounts contain RUB cash 1,000, RUB bank 2,000, USD cash 10, USDT 20, USDC 12.000000000123, BTC 0.001 and ETH 0.001234567891.
- **When:** The owner opens accounts.
- **Then:** Seven distinct accounts show original assets and exact balances; residuals are not truncated to UI or provider order precision. RUB is combined only in the relevant aggregate.
- **Level:** `integration`.

#### AC-003

- **Given:** A current valuation exists for every required pair.
- **When:** The member switches RUB to USD, USDT, USDC, BTC and ETH.
- **Then:** Equivalent totals change while original account and transaction amounts remain unchanged.
- **Level:** `end-to-end`.

#### AC-039

- **Given:** A source contains unsupported USDC.E; USDT/USD and USDC/USD rates are unavailable.
- **When:** A total valuation is built.
- **Then:** Raw data is retained and valuation coverage is incomplete; no hidden zero or automatic 1:1 rate is used. USDC.E is not merged into USDC by symbol similarity.
- **Level:** `integration`.

#### AC-059

- **Given:** Fractional BTC, USDT, interest accrual and receipt allocation exist.
- **When:** Data traverses API, storage and recalculation.
- **Then:** Original precision survives; JSON money never traverses binary floats; allocations reconcile exactly and display rounding does not alter the ledger.
- **Level:** `unit+contract`.

#### AC-062

- **Given:** Application structure and component contracts exist.
- **When:** Dependencies and public interfaces are checked.
- **Then:** Domain imports no HTTP, SQL, UI, OpenAI SDK or browser types; adapters map external models; the collector owns no financial decisions.
- **Level:** `static`.

#### AC-077

- **Given:** Two users belong to one household.
- **When:** Schema, authorization and the membership limit are inspected.
- **Then:** There are no partner1/partner2 fields or specific-user branches; roles and ownership are separate from identity. Exit, replacement and new roles are not implemented.
- **Level:** `integration`.

#### AC-079

- **Given:** A and B have separate accounts at one provider and a joint account; B enters A’s purchase paid from B’s account.
- **When:** Entry, import of both accounts and reconnection of the same external account run.
- **Then:** Distinct accounts are not merged; a repeated source does not double balances. Payer, author and expense beneficiary remain independent. Unresolved source identity blocks new posting pending clarification.
- **Level:** `integration`.

#### AC-090

- **Given:** Tests contain two isolated households; a request or job forges householdId/actor/resourceId.
- **When:** File reads, import, correction, AI retrieval and deduplication are exercised.
- **Then:** Foreign objects are inaccessible and never merged; the server takes principal from the session or validated job context. Denial reveals no foreign content.
- **Level:** `integration`.

#### AC-106

- **Given:** A connection is authenticated, but the provider/host gate is incomplete or the prior admission belongs to a different binding revision.
- **When:** A member or scheduler requests sync, or the build, contract, allowlist, configuration, permission or environment changes.
- **Then:** The server returns `provider_not_admitted`, never starts the collector and creates no posting. Only the admission service sets `admitted` after task-4.x provider evidence and task-8.x host evidence for the exact binding; any mismatch closes sync again.
- **Level:** `integration+security`.

### Verification

```sh
make check
```

Exact synthetic round trips and allocations, invalid inputs rejected, tested household/command invariants; valid OpenAPI and reproducible Go/TypeScript output. This is not auth/storage/product runtime proof.

Make commands are implemented. Checks use Go 1.26.5 and Node 24.19.0; OpenAPI tooling has isolated TS 5.9.3, web uses TS 6.0.3. Live banking, DB, deploy and browser E2E are outside this task.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
