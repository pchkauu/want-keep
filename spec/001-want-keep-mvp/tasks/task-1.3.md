<!-- want-keep-task: task-1.3 -->
# task-1.3 — Создать хранилище и транзакционные границы / Create storage and transaction boundaries

## RU

Сделать сохранение учёта, версий и заданий атомарным.

**Состояние:** Хранилище контракта 10 реализовано; PostgreSQL integration и race suite проверяют транзакционные границы. Публикация/review/CI — в issue #13. SDD Ready for development; продуктовая и эксплуатационная приёмка остаются у следующих задач.

**Зависимости:** `task-1.2`.

**Тип:** `implementation`.

### Изменение и контракты

Добавить миграции для семьи/участников, счетов, неизменяемых source records, операций/проводок, revisions, command records, idempotency tombstones и outbox/jobs. Денежные поля NUMERIC сохраняют точность источника. Source uniqueness следует D-39 и включает household, provider, стабильный внешний счёт, product/log namespace и provider record ID; connection/session ID остаётся provenance. Коллизия сохраняет evidence как `source_ambiguous` без проводки. Транзакционно обеспечить uniqueness, version-check и резервы. Реализовать независимые retention jobs: terminal detail 90 дней после исхода; unresolved до сверки плюс 90 дней; tombstone с `commandId` живёт всё unresolved-состояние и 400 дней после terminal/reconciled outcome; финансовый source/audit не удаляется вместе с command detail. Реализовать `ProviderDeploymentAdmission` в `backend/internal/connections/admission/`: aggregate и repository interface принадлежат application boundary, storage adapter сохраняет provider/host evidence и exact binding. Application service атомарно объединяет оба pass, повышает `admissionRevision` при каждой смене state/evidence/binding и в одной транзакции проверяет admission с созданием sync job. Job и result несут неизменяемые binding/revision. Collector повторно проверяет их перед provider IO; storage повторно сверяет current admitted revision в транзакции записи source revision/posting/outbox. Stale result сохраняется только в quarantine, без финансового эффекта. Revoke/change инвалидирует не начатые jobs, запрашивает best-effort cancel уже начатых и всегда блокирует их commit. Покрыть restart, concurrent combine/sync, revoke до/после начала IO и binding-change races. Регистрация предшествует исполнению; эффект и terminal result атомарны. Уникальность household+actor+key; type/hash неизменны, replay до повторного version check. Pending после неизвестного исхода требует сверки; not_found не разрешает новый ключ. Статусы доступны только инициатору; права на результат проверяются отдельно.

### Границы изменений

- `backend/migrations/`
- `backend/internal/storage/`
- `backend/internal/jobs/`
- `backend/internal/connections/admission/`
- `backend/internal/commands/`
- `backend/internal/accounts/`
- `backend/internal/ledger/`
- `backend/internal/goals/`
- `backend/internal/household/application/`
- `backend/cmd/migrate/`
- `backend/cmd/command-retention/`
- `backend/test/integration/storage/`
- `.github/workflows/ci.yml`

Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-012:** Исправление учёта сохраняет оригинал, автора, основание, версию и возможность отмены решения.
- **REQ-029:** Одни средства нельзя одновременно зарезервировать на несколько целей или повторно учесть через выделенный счёт.
- **REQ-059:** Денежные расчёты используют точную арифметику и явные правила округления на границах.
- **REQ-061:** Повторные задания, перезапуски и параллельные изменения не создают двойных финансовых эффектов.
- **REQ-062:** Архитектура использует Go/PostgreSQL, React/TypeScript/Vite и отдельный Playwright-сборщик с зависимостями к домену.
- **REQ-064:** Оба участника видят все финансовые данные и изменяют операции; личные цели и части плана изменяет только их владелец.
- **REQ-069:** Резервы личных и совместных целей задаются явно; совместные цели отображаются отдельным общим блоком без персональных долей.
- **REQ-070:** Сумма индивидуальных дневных лимитов не превышает семейный предел одной валюты; счёт плательщика не меняет долю расходов.
- **REQ-072:** Конкурирующие изменения, ответы на уточнения и отмены проверяют версию и текущие права, сохраняя обоих авторов.
- **REQ-076:** Семейная область проверяется для API, файлов, AI, фоновых задач и внешних ID независимо от присланных actor/owner.
- **REQ-088:** Синхронизация провайдера разрешена только актуальным server-side admission, связанным с проверенными версиями адаптера, контракта, allowlist, конфигурации, разрешения оператора и окружения.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

#### AC-012

- **Дано:** AI ошибочно связал две операции; исходные импортированные записи сохранены.
- **Когда:** Владелец отменяет связь и исправляет категорию.
- **Тогда:** Пересчитаны производные отчёты; видна история; повторный импорт не стирает правку владельца.
- **Уровень:** `integration`.

#### AC-029

- **Дано:** На счёте USD 100 уже зарезервировано USD 80.
- **Когда:** Вторая цель запрашивает USD 30 либо тот же резерв дублируется ссылкой на счёт.
- **Тогда:** Операция превышения отклоняется атомарно; свободно USD 20; параллельные запросы не обходят ограничение.
- **Уровень:** `integration`.

#### AC-059

- **Дано:** Есть дробные BTC, USDT, процентное начисление и распределение чека.
- **Когда:** Данные проходят API, базу и повторный расчёт.
- **Тогда:** Исходная точность не теряется; JSON-суммы не проходят binary float; распределения сходятся точно, округление отображения не меняет журнал.
- **Уровень:** `unit+contract`.

#### AC-061

- **Дано:** Процесс падает между сохранением записи и подтверждением задания.
- **Когда:** Задание повторяется, одновременно приходит правка владельца.
- **Тогда:** Применён один эффект, правка защищена версией, незавершённое состояние восстанавливается; внешняя неоднозначность не вызывает слепой повтор.
- **Уровень:** `integration`.

#### AC-062

- **Дано:** Создана структура приложения и контракты компонентов.
- **Когда:** Проверяются зависимости и публичные интерфейсы.
- **Тогда:** Домен не импортирует HTTP, SQL, UI, OpenAI SDK или браузерные типы; адаптеры маппят внешние модели; сборщик не владеет финансовыми решениями.
- **Уровень:** `static`.

#### AC-086

- **Дано:** A и B открыли одну версию операции или уточнения.
- **Когда:** Оба отправляют несовместимые изменения и повторяют один запрос.
- **Тогда:** Один результат применяется; второй получает конфликт с необходимостью перечитать состояние. Повтор не дублирует эффект; отмена создаёт новую проверенную revision и не стирает чужую последующую правку.
- **Уровень:** `integration`.

#### AC-090

- **Дано:** В тестах созданы две изолированные семьи; запрос или задача подменяет householdId/actor/resourceId.
- **Когда:** Проверяются чтение файла, импорт, исправление, поиск AI и дедупликация.
- **Тогда:** Чужие объекты недоступны и не объединяются; сервер берёт principal из сессии или проверенного контекста задания. Отказ не раскрывает чужое содержимое.
- **Уровень:** `integration`.

#### AC-092

- **Дано:** Свободно RUB 1000; оба пытаются зарезервировать по 800 для разрешённых целей.
- **Когда:** Команды исполняются одновременно.
- **Тогда:** Проверка общего доступного остатка и резерв атомарны: проходит максимум одна команда; отказ не уменьшает другой резерв, оба видят актуальный остаток.
- **Уровень:** `integration`.

#### AC-106

- **Дано:** Подключение авторизовано, но provider/host gate неполон либо прошлый admission относится к другой версии binding.
- **Когда:** Участник или scheduler запрашивает sync, либо меняются build, contract, allowlist, config, permission или environment.
- **Тогда:** Если binding уже неполон или устарел, сервер возвращает `provider_not_admitted` без job, collector IO и проводки. Только admission service ставит `admitted` после provider evidence task-4.x и host evidence task-8.x для точного binding. Job/result несёт неизменяемые binding и `admissionRevision`; смена binding во время read отменяет работу best effort, а обязательная commit-time revalidation сохраняет stale result в quarantine без source record или проводки.
- **Уровень:** `integration+security`.

### Проверка результата

```sh
make test-integration AREA=storage
```

Миграции применяются к пустой БД; crash/retry и конкуренция не дают частичных проводок или двойных эффектов. Admission переживает restart; неполный/устаревший binding не создаёт job, а revoke/change до IO не допускает provider IO, а после начала IO commit-time fence не допускает source record или проводку.

Suite реализована на PostgreSQL 17.11 с закреплённым digest и pgx 5.10.0. Обязательны make check, make test-integration AREA=storage, make test-storage-race и git diff --check; отсутствие БД — ошибка. Доказательства и ограничения: evidence/task-1.3-storage.md. Live banking, HTTP/auth, browser E2E и production deploy не заявляются пройденными.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Make accounting, revision and job persistence atomic.

**Status:** Contract 10 storage is implemented; PostgreSQL integration and race suites verify transaction boundaries. Publication/review/CI are recorded in issue #13. The SDD is Ready for development; product and operational acceptance remain with subsequent tasks.

**Dependencies:** `task-1.2`.

**Kind:** `implementation`.

### Change and contracts

Add migrations for households/members, accounts, immutable source records, transactions/postings, revisions, command records, idempotency tombstones and outbox/jobs. NUMERIC money fields preserve source precision. Source uniqueness follows D-39 and includes household, provider, stable external account, product/log namespace and provider record ID; connection/session ID remains provenance. A collision retains evidence as `source_ambiguous` without posting. Enforce uniqueness, version checks and reservations transactionally. Implement independent retention jobs: terminal detail for 90 days after outcome; unresolved commands through reconciliation plus 90 days; a tombstone with `commandId` lives throughout unresolved state and for 400 days after terminal/reconciled outcome; financial source/audit data is not removed with command detail. Implement `ProviderDeploymentAdmission` in `backend/internal/connections/admission/`: the aggregate and repository interface belong to the application boundary, while the storage adapter persists provider/host evidence and the exact binding. The application service atomically combines both passes, increments `admissionRevision` on every state/evidence/binding change and checks admission in the same transaction that creates a sync job. Jobs and results carry immutable binding/revision. The collector rechecks them before provider IO; storage rechecks the current admitted revision in the transaction that persists source revision/posting/outbox. A stale result is retained only in quarantine with no financial effect. Revocation/change invalidates unstarted jobs, requests best-effort cancellation of started work and always blocks its commit. Cover restart, concurrent combine/sync, revocation before/after IO starts and binding-change races. Registration precedes execution; effect and terminal result are atomic. Uniqueness is household+actor+key; type/hash are immutable and replay precedes a fresh version check. Pending after an unknown outcome requires reconciliation; not_found does not permit a new key. Status is visible only to its originator; result permissions are checked separately.

### Change boundaries

- `backend/migrations/`
- `backend/internal/storage/`
- `backend/internal/jobs/`
- `backend/internal/connections/admission/`
- `backend/internal/commands/`
- `backend/internal/accounts/`
- `backend/internal/ledger/`
- `backend/internal/goals/`
- `backend/internal/household/application/`
- `backend/cmd/migrate/`
- `backend/cmd/command-retention/`
- `backend/test/integration/storage/`
- `.github/workflows/ci.yml`

Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-012:** Accounting corrections preserve the original, actor, reason, version and ability to undo a decision.
- **REQ-029:** The same money cannot be reserved for multiple goals or counted again through a dedicated account.
- **REQ-059:** Money calculations use exact arithmetic and explicit boundary rounding rules.
- **REQ-061:** Repeated jobs, restarts and concurrent changes cannot create duplicate financial effects.
- **REQ-062:** Architecture uses Go/PostgreSQL, React/TypeScript/Vite and a separate Playwright collector with dependencies pointing toward the domain.
- **REQ-064:** Both members see all financial data and edit transactions; only the owner edits personal goals and plan portions.
- **REQ-069:** Personal and joint goal reservations are explicit; joint goals appear in a separate shared block without personal shares.
- **REQ-070:** Individual daily allowances sum to no more than the household ceiling in one currency; the payer’s account does not change expense shares.
- **REQ-072:** Competing edits, clarification answers and reversals check revision and current permissions while retaining both authors.
- **REQ-076:** Household scope is checked for APIs, files, AI, jobs and external IDs independently of supplied actor/owner fields.
- **REQ-088:** Provider sync is allowed only by a current server-side admission bound to verified adapter, contract, allowlist, configuration, operator-permission and environment revisions.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

#### AC-012

- **Given:** AI incorrectly linked two transactions; original imports are retained.
- **When:** The owner unlinks them and corrects the category.
- **Then:** Derived reports are recalculated, history is visible and reimport does not overwrite the owner's correction.
- **Level:** `integration`.

#### AC-029

- **Given:** USD 80 of an account's USD 100 is already reserved.
- **When:** A second goal requests USD 30 or an account link duplicates the reservation.
- **Then:** The over-allocation is rejected atomically; USD 20 remains free; concurrent requests cannot bypass the limit.
- **Level:** `integration`.

#### AC-059

- **Given:** Fractional BTC, USDT, interest accrual and receipt allocation exist.
- **When:** Data traverses API, storage and recalculation.
- **Then:** Original precision survives; JSON money never traverses binary floats; allocations reconcile exactly and display rounding does not alter the ledger.
- **Level:** `unit+contract`.

#### AC-061

- **Given:** A process crashes between persisting a record and acknowledging its job.
- **When:** The job is retried while the owner submits a correction.
- **Then:** One effect is applied, the correction is version-protected and incomplete state recovers; an ambiguous external outcome is not blindly retried.
- **Level:** `integration`.

#### AC-062

- **Given:** Application structure and component contracts exist.
- **When:** Dependencies and public interfaces are checked.
- **Then:** Domain imports no HTTP, SQL, UI, OpenAI SDK or browser types; adapters map external models; the collector owns no financial decisions.
- **Level:** `static`.

#### AC-086

- **Given:** A and B opened the same transaction or clarification revision.
- **When:** Both submit conflicting edits and replay one request.
- **Then:** One result applies; the other receives a conflict requiring refresh. Replay does not duplicate effects; reversal creates a checked new revision without erasing the other member’s later edit.
- **Level:** `integration`.

#### AC-090

- **Given:** Tests contain two isolated households; a request or job forges householdId/actor/resourceId.
- **When:** File reads, import, correction, AI retrieval and deduplication are exercised.
- **Then:** Foreign objects are inaccessible and never merged; the server takes principal from the session or validated job context. Denial reveals no foreign content.
- **Level:** `integration`.

#### AC-092

- **Given:** RUB 1,000 is free; both attempt to reserve 800 for authorized goals.
- **When:** Commands execute concurrently.
- **Then:** Checking household availability and reserving are atomic: at most one command succeeds; rejection does not reduce another reserve and both see current availability.
- **Level:** `integration`.

#### AC-106

- **Given:** A connection is authenticated, but the provider/host gate is incomplete or the prior admission belongs to a different binding revision.
- **When:** A member or scheduler requests sync, or the build, contract, allowlist, configuration, permission or environment changes.
- **Then:** If the binding is already incomplete or stale, the server returns `provider_not_admitted` with no job, collector IO or posting. Only the admission service sets `admitted` after task-4.x provider evidence and task-8.x host evidence for the exact binding. Each job/result carries immutable binding and `admissionRevision`; a binding change during a read cancels work best effort, while mandatory commit-time revalidation retains a stale result in quarantine without a source record or posting.
- **Level:** `integration+security`.

### Verification

```sh
make test-integration AREA=storage
```

Migrations apply to an empty DB; crash/retry and concurrency create neither partial postings nor duplicate effects. Admission survives restart; an incomplete/stale binding creates no job, and revocation/change before IO prevents provider IO, while after IO starts the commit-time fence prevents a source record or posting.

The suite uses PostgreSQL 17.11 at a pinned digest and pgx 5.10.0. Required: make check, make test-integration AREA=storage, make test-storage-race and git diff --check; a missing DB fails. Evidence and limits: evidence/task-1.3-storage.en.md. Live banking, HTTP/auth, browser E2E and production deployment are not claimed as passed.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
