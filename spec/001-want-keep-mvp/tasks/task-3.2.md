<!-- want-keep-task: task-3.2 -->
# task-3.2 — Определить входной контракт коннекторов / Define connector ingestion contracts

## RU

Нормализовать данные без утечки моделей платформ в домен.

**Состояние:** Реализованы versioned wire-контракт, exact gateway binding/sync-result cursor fence, строгие Go/TypeScript null-safe boundary, accounts-owned asset normalization, durable evidence ownership/disposition, commit receipt recovery и атомарное применение synthetic ingestion pages с same-page preflight и безопасной частичной обработкой неоднозначности; реальные provider IO, UI и production остаются последующим задачам. Restart reconciliation durable terminal receipts, непустой nextCursor и канонический base64 входят в результат.

**Зависимости:** `task-3.1`, `task-1.2`, `task-2.1`, `task-2.2`.

**Тип:** `implementation`.

### Изменение и контракты

Контракт версии 10 описывает server-issued job, capability только чтения, точный D-43 binding/admissionRevision, cursor/replay, coverage, evidence и типизированные account/balance/transaction/failure records. Server-owned gateway несёт полный immutable binding и должен совпасть с job до manifest/provider IO. Provider, product/log namespace, record kind и read action проверяются по manifest до evidence. Required-поля обязательны и вместе с null/unknown/trailing JSON отклоняются одинаково в Go/TypeScript. Необязательные пустые merchant/note/network/aliases канонизируются как отсутствие, а присутствующие enum, amount/reason и retry delay проверяются по discriminator и диапазону. Строки используют Unicode code points с запретом NUL/lone surrogate и отдельными byte limits; совокупный набор gaps после объединения уникален и ограничен 100 значениями, а maximumLookbackDays либо отсутствует, либо равен 1–36500. Время канонично в UTC с Z. Составная D-39 identity кодируется структурно и не включает page-local evidence ID/locator; payload hash использует нормализованную запись и отдельный raw digest; alias не пропускает PAN/CVV. Fee передаётся отдельной проводкой, а свободный feeId запрещён до типизированной correspondence. Raw evidence сохраняется с server-derived household/job и durable staged disposition до финансового commit; финансовый эффект и immutable receipt миграции 016 коммитятся атомарно. Неизвестный commit подтверждается receipt readback, иначе evidence остаётся staged; отдельный lifecycle context записывает rejected_result только после доказанного отказа и успешного durable retention. Сервер назначает principal, external owner, internal IDs/revisions, evidence reference и fetchedAt. Каждая страница повторяет account descriptor для balance/posting; account-only page не создаёт observation. Accounts-owned policy применяет только подтверждённые mappings, включая Raiffeisen/Ozon RUR → RUB, и сохраняет raw code. Server-side account/source ambiguity предварительно группирует D-39 keys всей страницы, дедуплицирует одинаковые факты, сохраняет omissions, делает coverage partial, не проводит конфликтующую группу и не откатывает независимые записи. Admission/generation/lease/cursor fencing выполняется через CommitPage; stale result остаётся только в quarantine. Provider failure повторяет точный issued cursor; cursor проверяется на sync-result boundary отдельно от lease fence, поэтому запоздалый outcome не меняет job после продвижения checkpoint. Provider failure атомарно связывает evidence с household/job и переводит job в ожидание, ограниченный retry или terminal failure. Generated DTO не входят в domain, суммы остаются decimal-строками, unknown/unavailable и unsupported assets не становятся нулём или паритетом. `nextCursor` либо отсутствует, либо непустой; evidence использует канонический base64 без CR/LF. Миграция 016 хранит terminal receipts для page/provider_outcome/rejected_result/stale_result; restart reconciler завершает staged disposition только по server-owned household/job/evidence reference и не повторяет финансовый эффект.

### Границы изменений

- `collector/contracts/v10/`
- `collector/src/contracts/`
- `backend/internal/integrations/`
- `backend/internal/accounts/application/`
- `backend/test/integration/ingestion/`
- `backend/migrations/016_ingestion_result_receipts.sql`
- `scripts/generate-ingestion-contracts.sh`
- `.github/workflows/ci.yml`

Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-004:** Начало учёта задаётся датой; начальные остатки отделены от доходов и расходов.
- **REQ-005:** Счета показывают собственные, доступные, заблокированные и заёмные средства в пределах данных источника.
- **REQ-008:** Повторные импорты, чек и запись чата объединяют доказательства одной операции без повторного учёта.
- **REQ-009:** Статусы ожидающей, проведённой, отменённой и возвращённой операции учитываются явно.
- **REQ-035:** Торговая аналитика отделяет реализованный результат, нереализованный результат, комиссии и funding.
- **REQ-039:** Отсутствующие курсы и неподдерживаемые активы не превращаются в нулевые суммы или условный паритет USD/USDT/USDC.
- **REQ-041:** История сохраняет границы покрытия, курсоры, пробелы и статусы источника.
- **REQ-048:** Интеграции и браузерный сборщик выполняют только разрешённые операции чтения.
- **REQ-062:** Архитектура использует Go/PostgreSQL, React/TypeScript/Vite и отдельный Playwright-сборщик с зависимостями к домену.
- **REQ-065:** Принадлежность счёта, владелец внешнего аккаунта, автор записи и принадлежность расхода являются отдельными признаками.
- **REQ-073:** Оба управляют подключениями; банковскую авторизацию выполняет владелец внешнего аккаунта без раскрытия секретов партнёру или AI.
- **REQ-076:** Семейная область проверяется для API, файлов, AI, фоновых задач и внешних ID независимо от присланных actor/owner.
- **REQ-088:** Синхронизация провайдера разрешена только актуальным server-side admission, связанным с проверенными версиями адаптера, контракта, allowlist, конфигурации, разрешения оператора и окружения.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

#### AC-004

- **Дано:** История запрошена с 1 августа; начальный остаток RUB 5 000 подтверждён.
- **Когда:** Импортируется расход RUB 500 от 2 августа.
- **Тогда:** Остаток равен RUB 4 500; доход августа не увеличивается на начальные RUB 5 000; неподтверждённое начало обозначается явно.
- **Уровень:** `integration`.

#### AC-005

- **Дано:** Источник сообщает собственные RUB 100, долг RUB 300 и кредитный лимит RUB 1 000.
- **Когда:** Строится сводка денег.
- **Тогда:** Кредитный лимит не увеличивает собственный капитал или доступный бюджет; отсутствующее поле отображается как неизвестное.
- **Уровень:** `integration`.

#### AC-008

- **Дано:** Расход RUB 300 создан из чата с выбранным счётом.
- **Когда:** Поступают соответствующий чек, банковская операция и повтор той же операции.
- **Тогда:** Расход остаётся RUB 300, все источники связаны; две отдельные покупки одной суммы не объединяются лишь из-за равенства суммы.
- **Уровень:** `integration`.

#### AC-009

- **Дано:** Карточная авторизация RUB 500 сначала ожидает подтверждения.
- **Когда:** Она проводится либо отменяется.
- **Тогда:** Проведение создаёт один фактический расход; отмена ожидающей операции расхода не создаёт. Заблокированная сумма и статус не скрыты.
- **Уровень:** `integration`.

#### AC-035

- **Дано:** Источник передал реализованный результат USDT 10, комиссию USDT 1 и нереализованный результат USDT 5.
- **Когда:** Обновляется отчёт торгового счёта.
- **Тогда:** Показатели разделены; нереализованные USDT 5 не становятся полученным доходом; чистый результат не дублируется отдельным повторным вычетом уже включённой комиссии.
- **Уровень:** `integration`.

#### AC-039

- **Дано:** В источнике есть неподдерживаемый USDC.E; для USDT/USD и USDC/USD отсутствуют курсы.
- **Когда:** Строится общая оценка.
- **Тогда:** Исходные данные сохранены, покрытие оценки обозначено неполным; нет скрытого нуля или автоматического курса 1:1. USDC.E не объединён с USDC по похожему символу.
- **Уровень:** `integration`.

#### AC-041

- **Дано:** Источник выдаёт несколько страниц с ограничением глубины; второй запрос завершился ошибкой.
- **Когда:** Импорт возобновляется.
- **Тогда:** Подтверждённые страницы сохранены без дублей; курсор не перескакивает пропуск; неполная история и её границы видны.
- **Уровень:** `integration`.

#### AC-048

- **Дано:** Сборщик имеет сессию личного кабинета с более широкими внешними правами.
- **Когда:** Возникают запрос на платёж, неподтверждённый маршрут или MFA/CAPTCHA.
- **Тогда:** Платёж и неизвестный маршрут блокируются; MFA/CAPTCHA передаётся владельцу, источник приостанавливается; остальные источники продолжают работать.
- **Уровень:** `integration`.

#### AC-062

- **Дано:** Создана структура приложения и контракты компонентов.
- **Когда:** Проверяются зависимости и публичные интерфейсы.
- **Тогда:** Домен не импортирует HTTP, SQL, UI, OpenAI SDK или браузерные типы; адаптеры маппят внешние модели; сборщик не владеет финансовыми решениями.
- **Уровень:** `static`.

#### AC-079

- **Дано:** A и B имеют разные аккаунты одного провайдера и общий счёт; B заносит покупку A со счёта B.
- **Когда:** Выполняются ввод, импорт обоих аккаунтов и повторное подключение того же внешнего аккаунта.
- **Тогда:** Разные аккаунты не сливаются; повторный источник не удваивает остатки. Плательщик, автор и получатель расхода сохраняются независимо. Неустановленное совпадение блокирует новый учёт до уточнения.
- **Уровень:** `integration`.

#### AC-087

- **Дано:** A владеет внешним аккаунтом, B инициирует повторную авторизацию или отключение.
- **Когда:** Запрашивается MFA; одновременно завершает работу старое задание синхронизации.
- **Тогда:** MFA адресован A; B видит статус, но не пароль/код/сессию. Отключение отзывает lease/version и запрещает применение старого результата; реальные платежи недоступны обоим.
- **Уровень:** `integration`.

#### AC-090

- **Дано:** В тестах созданы две изолированные семьи; запрос или задача подменяет householdId/actor/resourceId.
- **Когда:** Проверяются чтение файла, импорт, исправление, поиск AI и дедупликация.
- **Тогда:** Чужие объекты недоступны и не объединяются; сервер берёт principal из сессии или проверенного контекста задания. Отказ не раскрывает чужое содержимое.
- **Уровень:** `integration`.

#### AC-106

- **Дано:** Подключение авторизовано, но provider/host gate неполон либо прошлый admission относится к другой версии binding.
- **Когда:** Участник или scheduler запрашивает sync, либо меняются build, contract, allowlist, config, permission или environment.
- **Тогда:** Если binding уже неполон или устарел, сервер возвращает `provider_not_admitted` без job, collector IO и проводки. Только admission service ставит `admitted` после provider evidence task-4.x и host evidence task-8.x для точного binding. Job/result несёт неизменяемые binding и `admissionRevision`; смена binding во время read отменяет работу best effort, а обязательная commit-time revalidation сохраняет stale result в quarantine без source record или проводки.
- **Уровень:** `integration+security`.

### Проверка результата

```sh
make check-contracts && make test-collector FILTER=contracts && make test-integration AREA=ingestion && make test-ingestion-race
```

Точные шесть активов, подтверждённое RUR → RUB с сохранением raw code, строгий JSON/required/null/evidence, нормализация необязательных пустых значений, discriminator/range validation, commit receipt readback, evidence-neutral payload hash, exact gateway/job binding, каноническое время, совокупный лимит gaps, единые Unicode/lookback limits, структурная D-39 identity, безопасные alias, отдельные fee postings без свободного feeId, partial coverage при account/source ambiguity, provider failure sync-result cursor fence, durable staged/rejected-result disposition и same-page D-39 preflight, replay/cursor и stale-admission quarantine проходят без потери точности, подмены principal или повторного финансового эффекта. После рестарта staged evidence завершается по durable terminal receipt без повторного финансового применения; без receipt остаётся staged. Пустой nextCursor и base64 с CR/LF отклоняются на обеих границах.

Зависимости включены в базу. Доказательства и границы: evidence/task-3.2-ingestion.md. Обязательны make check, full integration matrix и ingestion/jobs/storage/accounts/ledger/audit/matching/reconciliation race suites. Ingestion fixtures удаляют свои синтетические БД. Live provider IO и эксплуатационная готовность не подтверждаются.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Normalize data without leaking provider models into the domain.

**Status:** The versioned wire contract, exact gateway binding/sync-result cursor fence, strict null-safe Go/TypeScript boundaries, accounts-owned asset normalization, durable evidence ownership/disposition, commit-receipt recovery and atomic synthetic-page application with same-page preflight and safe partial ambiguity handling are implemented; live provider IO, UI and production remain downstream. Restart reconciliation through durable terminal receipts, non-empty nextCursor and canonical base64 are included.

**Dependencies:** `task-3.1`, `task-1.2`, `task-2.1`, `task-2.2`.

**Kind:** `implementation`.

### Change and contracts

Contract version 10 defines a server-issued job, read-only capabilities, exact D-43 binding/admissionRevision, cursor/replay, coverage, evidence and typed account/balance/transaction/failure records. The server-owned gateway carries the full immutable binding and must match the job before manifest/provider IO. Provider, product/log namespace, record kind and read action are checked against the manifest before evidence. Required fields are mandatory and missing/null/unknown/trailing JSON is rejected consistently in Go and TypeScript. Optional empty merchant/note/network/aliases values canonicalize as absent, while present enums, amount/reason and retry delays are validated against their discriminator and range. Strings use Unicode code points with NUL/lone-surrogate rejection and separate byte limits; the cumulative gap set after merging is unique and capped at 100 values, while maximumLookbackDays is either omitted or 1–36500. Instants use canonical UTC Z. Composite D-39 identity is structurally encoded without page-local evidence ID/locator; the payload hash uses the normalized record and separate raw digest. Aliases cannot carry PAN/CVV. Fees use separate postings, while free-form feeId is forbidden until typed correspondence exists. Raw evidence is durable with server-derived household/job ownership and a staged disposition before the financial commit; the financial effect and immutable migration-016 receipt commit atomically. An unknown commit is proven by receipt readback or evidence remains staged; a separate lifecycle context records rejected_result only after a proven rejection and successful durable retention. The server assigns principal, external owner, internal IDs/revisions, evidence reference and fetchedAt. Every page repeats an account descriptor for each balance/posting; an account-only page creates no observation. An accounts-owned policy applies only confirmed mappings, including Raiffeisen/Ozon RUR → RUB, and preserves the raw code. Server-side account/source ambiguity preflights D-39 keys across the page, deduplicates identical facts, retains omissions, makes coverage partial, posts no conflicting group and does not roll back independent records. Admission/generation/lease/cursor fencing uses CommitPage; stale results remain only in quarantine. A provider failure echoes the exact issued cursor; the cursor is checked at the sync-result boundary separately from the lease fence, so a delayed outcome cannot change the job after checkpoint advancement. A provider failure atomically associates evidence with household/job and moves the job to waiting, bounded retry or terminal failure. Generated DTOs never enter the domain, money remains decimal strings, and unknown/unavailable or unsupported assets never become zero or parity. `nextCursor` is either absent or non-empty, and evidence uses canonical base64 without CR/LF. Migration 016 stores terminal receipts for page/provider_outcome/rejected_result/stale_result; the restart reconciler finalizes staged disposition only by server-owned household/job/evidence reference and does not replay financial effects.

### Change boundaries

- `collector/contracts/v10/`
- `collector/src/contracts/`
- `backend/internal/integrations/`
- `backend/internal/accounts/application/`
- `backend/test/integration/ingestion/`
- `backend/migrations/016_ingestion_result_receipts.sql`
- `scripts/generate-ingestion-contracts.sh`
- `.github/workflows/ci.yml`

Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-004:** Accounting starts on a selected date; opening balances are separate from income and expenses.
- **REQ-005:** Accounts distinguish owned, available, locked and borrowed amounts where the source provides them.
- **REQ-008:** Repeated imports, receipts and chat entries combine evidence of one transaction without double counting.
- **REQ-009:** Pending, posted, cancelled and refunded transaction states are explicit.
- **REQ-035:** Trading analytics separates realized P&L, unrealized P&L, fees and funding.
- **REQ-039:** Missing rates and unsupported assets never become zero amounts or assumed USD/USDT/USDC parity.
- **REQ-041:** History retains coverage boundaries, cursors, gaps and source status.
- **REQ-048:** Integrations and the browser collector perform authorized read operations only.
- **REQ-062:** Architecture uses Go/PostgreSQL, React/TypeScript/Vite and a separate Playwright collector with dependencies pointing toward the domain.
- **REQ-065:** Account ownership, external-account owner, record author and expense attribution are distinct dimensions.
- **REQ-073:** Both manage connections; the external-account owner performs bank authentication without exposing secrets to the partner or AI.
- **REQ-076:** Household scope is checked for APIs, files, AI, jobs and external IDs independently of supplied actor/owner fields.
- **REQ-088:** Provider sync is allowed only by a current server-side admission bound to verified adapter, contract, allowlist, configuration, operator-permission and environment revisions.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

#### AC-004

- **Given:** History is requested from August 1; an opening RUB 5,000 balance is confirmed.
- **When:** A RUB 500 expense dated August 2 is imported.
- **Then:** Balance is RUB 4,500; August income excludes the opening RUB 5,000; an unverified opening is explicit.
- **Level:** `integration`.

#### AC-005

- **Given:** The source reports RUB 100 owned, RUB 300 debt and a RUB 1,000 credit limit.
- **When:** A money summary is built.
- **Then:** The credit limit does not increase net worth or the spendable budget; missing fields are shown as unknown.
- **Level:** `integration`.

#### AC-008

- **Given:** A RUB 300 expense was created from chat for a selected account.
- **When:** The matching receipt, bank transaction and duplicate bank delivery arrive.
- **Then:** Expense remains RUB 300 and all evidence is linked; separate equal-amount purchases are not merged merely by amount.
- **Level:** `integration`.

#### AC-009

- **Given:** A RUB 500 card authorization is initially pending.
- **When:** It posts or is cancelled.
- **Then:** Posting creates one actual expense; cancelling a pending authorization creates none. The hold and status remain visible.
- **Level:** `integration`.

#### AC-035

- **Given:** A source reports USDT 10 realized P&L, USDT 1 fee and USDT 5 unrealized P&L.
- **When:** The trading account report updates.
- **Then:** Metrics are separate; unrealized USDT 5 is not received income; net results do not suffer a second deduction for already-included fees.
- **Level:** `integration`.

#### AC-039

- **Given:** A source contains unsupported USDC.E; USDT/USD and USDC/USD rates are unavailable.
- **When:** A total valuation is built.
- **Then:** Raw data is retained and valuation coverage is incomplete; no hidden zero or automatic 1:1 rate is used. USDC.E is not merged into USDC by symbol similarity.
- **Level:** `integration`.

#### AC-041

- **Given:** A source provides paginated history with a retention limit; the second request fails.
- **When:** Import resumes.
- **Then:** Confirmed pages remain without duplicates; the cursor does not skip the gap; incomplete history and its boundaries are visible.
- **Level:** `integration`.

#### AC-048

- **Given:** The collector has a personal-account session with broader provider permissions.
- **When:** A payment request, unapproved route or MFA/CAPTCHA appears.
- **Then:** Payments and unknown routes are blocked; MFA/CAPTCHA is handed to the owner and that source pauses; other sources continue.
- **Level:** `integration`.

#### AC-062

- **Given:** Application structure and component contracts exist.
- **When:** Dependencies and public interfaces are checked.
- **Then:** Domain imports no HTTP, SQL, UI, OpenAI SDK or browser types; adapters map external models; the collector owns no financial decisions.
- **Level:** `static`.

#### AC-079

- **Given:** A and B have separate accounts at one provider and a joint account; B enters A’s purchase paid from B’s account.
- **When:** Entry, import of both accounts and reconnection of the same external account run.
- **Then:** Distinct accounts are not merged; a repeated source does not double balances. Payer, author and expense beneficiary remain independent. Unresolved source identity blocks new posting pending clarification.
- **Level:** `integration`.

#### AC-087

- **Given:** A owns the external account and B initiates reauthorization or disconnect.
- **When:** MFA is requested while an old sync job completes.
- **Then:** MFA is addressed to A; B sees status but no password/code/session. Disconnect revokes lease/version and prevents stale-result application; actual payments are unavailable to both.
- **Level:** `integration`.

#### AC-090

- **Given:** Tests contain two isolated households; a request or job forges householdId/actor/resourceId.
- **When:** File reads, import, correction, AI retrieval and deduplication are exercised.
- **Then:** Foreign objects are inaccessible and never merged; the server takes principal from the session or validated job context. Denial reveals no foreign content.
- **Level:** `integration`.

#### AC-106

- **Given:** A connection is authenticated, but the provider/host gate is incomplete or the prior admission belongs to a different binding revision.
- **When:** A member or scheduler requests sync, or the build, contract, allowlist, configuration, permission or environment changes.
- **Then:** If the binding is already incomplete or stale, the server returns `provider_not_admitted` with no job, collector IO or posting. Only the admission service sets `admitted` after task-4.x provider evidence and task-8.x host evidence for the exact binding. Each job/result carries immutable binding and `admissionRevision`; a binding change during a read cancels work best effort, while mandatory commit-time revalidation retains a stale result in quarantine without a source record or posting.
- **Level:** `integration+security`.

### Verification

```sh
make check-contracts && make test-collector FILTER=contracts && make test-integration AREA=ingestion && make test-ingestion-race
```

Six exact assets, confirmed RUR → RUB with the raw code retained, strict JSON/required/null/evidence, optional-empty normalization, discriminator/range validation, commit-receipt readback, evidence-neutral payload hashing, exact gateway/job binding, canonical time, cumulative gap cap, aligned Unicode/lookback limits, structural D-39 identity, safe aliases, separate fee postings without free-form feeId, partial coverage on account/source ambiguity, provider-failure sync-result cursor fencing, durable staged/rejected-result disposition and same-page D-39 preflight, replay/cursor and stale-admission quarantine pass without precision loss, principal spoofing or duplicate financial effects. After restart, staged evidence is finalized from a durable terminal receipt without reapplying financial effects; without a receipt it remains staged. Empty nextCursor and base64 with CR/LF are rejected at both boundaries.

Dependencies are included in the base. Evidence and boundaries: evidence/task-3.2-ingestion.en.md. Require make check, the full integration matrix and ingestion/jobs/storage/accounts/ledger/audit/matching/reconciliation race suites. Ingestion fixtures drop their synthetic databases. Live provider IO and operational readiness are not proven.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
