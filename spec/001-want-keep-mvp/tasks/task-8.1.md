<!-- want-keep-task: task-8.1 -->
# task-8.1 — Подготовить развёртывание и состояние системы / Prepare deployment and system health

## RU

Запускать сервисы в согласованном бюджете с наблюдаемыми отказами.

**Состояние:** Не начато; задача ожидает собственные зависимости и entry gates.

**Зависимости:** `task-0.9`, `task-3.3`, `task-5.1`, `task-7.8`.

**Тип:** `implementation`.

### Изменение и контракты

По evidence task-0.9 подготовить VPS в Германии 2 vCPU/4 ГБ/50 ГБ и managed PostgreSQL 1 vCPU/2 ГБ/20 ГБ в одной private VPC без публичного DB IP. Docker Compose запускает Go API/worker, web/reverse proxy и изолированный sequential collector; PostgreSQL входит только в локальные/integration окружения. До данных: key-only non-root SSH, provider+host firewall, закрытый/защищённый Zabbix, TLS/private DB endpoint и разделённые DB roles, секреты вне Git, redacted logs, pinned images, cgroups/pids/network allowlist. Миграции выполняются до consumers. Измерить RAM/CPU/disk/collector, health/degradation, deployment/rollback и прочитать итоговую смету с налогом/IP; provisioning требует отдельной авторизации владельца. Каждый provider deployment выключен по умолчанию до успешного read allowlist/permission/structured-fixture/identity/history/reauth conformance gate. Alfa route/DNS/TLS с целевого хоста входит в этот runtime readback. Эти проверки допускают эксплуатацию, но не меняют статус Ready SDD. Host evidence для точного D-43 binding передаётся admission service; только совместный pass provider+host переводит его в `admitted`, а deploy/config/allowlist/permission mismatch атомарно возвращает `pending|blocked` до нового sync.

### Границы изменений

- `deploy/compose.yaml`
- `backend/internal/health/`
- `docs/operations/`

Это планируемые пути. Общие контракты: `spec/001-want-keep-mvp/contracts.md`; архитектура и команды: `constraints.md`. Менять только владельца поведения и затронутые тесты; при незакрытом контракте обновить evidence и остановить зависимую реализацию.

### Связанные требования

- **REQ-022:** Полные операции и чеки могут передаваться OpenAI, секреты доступа и лишние закрытые данные исключаются.
- **REQ-048:** Интеграции и браузерный сборщик выполняют только разрешённые операции чтения.
- **REQ-050:** Файлы, ключи источников, сессии и финансовые журналы защищены от постороннего доступа.
- **REQ-051:** AI ограничен бюджетом $50/месяц и деградирует в очередь ожидания без остановки обычного учёта.
- **REQ-056:** Развёртывание укладывается в $40/месяц на сервер в DE/NL/BG; отдельные платные источники не используются.
- **REQ-058:** Операционные статусы показывают ошибки импорта, AI, курсов, резервирования и расходы без утечки финансового содержимого.
- **REQ-061:** Повторные задания, перезапуски и параллельные изменения не создают двойных финансовых эффектов.
- **REQ-062:** Архитектура использует Go/PostgreSQL, React/TypeScript/Vite и отдельный Playwright-сборщик с зависимостями к домену.
- **REQ-063:** Пользователь, семья и членство моделируются отдельно; ограничение двух участников задаётся конфигурацией.
- **REQ-073:** Оба управляют подключениями; банковскую авторизацию выполняет владелец внешнего аккаунта без раскрытия секретов партнёру или AI.
- **REQ-076:** Семейная область проверяется для API, файлов, AI, фоновых задач и внешних ID независимо от присланных actor/owner.
- **REQ-088:** Синхронизация провайдера разрешена только актуальным server-side admission, связанным с проверенными версиями адаптера, контракта, allowlist, конфигурации и окружения.

### Критерии приёмки

Связь с критерием задаёт покрытие; исследование или частичная задача не доказывает весь критерий продукта. Точный результат этой задачи указан ниже в проверке.

#### AC-022

- **Дано:** Операция и чек доступны владельцу; рядом в системе хранятся ключи источника.
- **Когда:** Формируется запрос AI и диагностическая запись.
- **Тогда:** В запросе только разрешённые данные операции/документа; ключи и сессии отсутствуют в запросе и логах; политика хранения OpenAI раскрыта.
- **Уровень:** `contract`.

#### AC-048

- **Дано:** Сборщик имеет сессию личного кабинета с более широкими внешними правами.
- **Когда:** Возникают запрос на платёж, неподтверждённый маршрут или MFA/CAPTCHA.
- **Тогда:** Платёж и неизвестный маршрут блокируются; MFA/CAPTCHA передаётся владельцу, источник приостанавливается; остальные источники продолжают работать.
- **Уровень:** `integration`.

#### AC-050

- **Дано:** Существует приватный чек и активное подключение источника.
- **Когда:** Проверяются прямой URL файла, экспорт без сессии, логи и отзыв подключения.
- **Тогда:** Без авторизации доступ закрыт; секреты зашифрованы и не журналируются; отзыв подключения прекращает дальнейший сбор.
- **Уровень:** `integration`.

#### AC-051

- **Дано:** OpenAI недоступен либо израсходован разрешённый бюджет с резервами текущих запросов.
- **Когда:** Поступают новый импорт, ручной расход и запрос AI.
- **Тогда:** Учёт и расчёты доступны; статус AI ожидает; новые платные запросы не запускаются сверх разрешённого резерва; неизвестная стоимость не освобождается молча.
- **Уровень:** `integration`.

#### AC-056

- **Дано:** Выбран конкретный тариф, регион, валюта счёта и налоги.
- **Когда:** Проверяется эксплуатационная смета для сотен операций в месяц.
- **Тогда:** Зафиксирована датированная полная смета сервера не выше $40; OpenAI учитывается отдельно до $50; обязательный платный источник остаётся блокером.
- **Уровень:** `manual`.

#### AC-058

- **Дано:** Сломан один коннектор, задержан AI и устарела копия.
- **Когда:** Открывается состояние системы и читаются диагностические логи.
- **Тогда:** Видны отдельные проблемы и действия восстановления; логи содержат идентификаторы/коды, а не чеки, ключи или тексты финансовых сообщений.
- **Уровень:** `integration`.

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

#### AC-077

- **Дано:** Два пользователя состоят в одной семье.
- **Когда:** Проверяются схема, авторизация и ограничение членства.
- **Тогда:** Нет полей partner1/partner2 и ветвлений по конкретным пользователям; роли и принадлежность отделены от личности. Выход, замена и новые роли не реализованы.
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
- **Тогда:** Сервер возвращает `provider_not_admitted`, collector не запускается и проводок нет. Только admission service ставит `admitted` после provider evidence task-4.x и host evidence task-8.x для точного binding; любое расхождение снова закрывает sync.
- **Уровень:** `integration+security`.

### Проверка результата

```sh
make check-deploy && make test-integration AREA=health
```

Контейнеры и ограничения проверены; отказ коннектора/AI не останавливает учёт; нет публичной БД или секретов в логах.

Основа task-1.1 уже предоставляет `make check-deploy` и `make test-integration` как fail-fast интерфейсы. До реализации task-8.1 эти suites обязаны завершаться понятной ошибкой; наличие команды не доказывает deployment, hardening, managed DB или runtime health.

### Передача следующему агенту

Записать изменённые контракты, команды и результаты, ограничения, незакрытые вопросы и разблокированные зависимости. Обновить обе языковые версии и трассировку. Закрывать задачу только по доказательству её результата; GitHub Closed само по себе не означает Ready MVP.

**Commit boundary:** логическая граница этой задачи; commit/push/deploy не разрешены данной карточкой и требуют действующей авторизации пользователя.

## EN

Run services within the agreed budget with observable failures.

**Status:** Not started; the task awaits its own dependencies and entry gates.

**Dependencies:** `task-0.9`, `task-3.3`, `task-5.1`, `task-7.8`.

**Kind:** `implementation`.

### Change and contracts

Using task-0.9 evidence, prepare a German 2 vCPU/4 GB/50 GB VPS and 1 vCPU/2 GB/20 GB managed PostgreSQL in one private VPC with no public DB IP. Docker Compose runs Go API/worker, web/reverse proxy and an isolated sequential collector; PostgreSQL remains containerized only in development/integration environments. Before data: key-only non-root SSH, provider and host firewalls, closed/protected Zabbix, TLS/private DB endpoint with separate DB roles, secrets outside Git, redacted logs, pinned images and cgroup/pid/network allowlists. Run migrations before consumers. Measure RAM/CPU/disk/collector, health/degradation and deployment/rollback, and read back the tax/IP-inclusive cost; provisioning requires separate owner authorization. Every provider deployment is disabled by default until its read allowlist, permission, structured-fixture, identity, history and reauthentication conformance gate passes. Alfa route/DNS/TLS from the target host belongs to this runtime readback. These checks authorize operation but do not change SDD Ready status. Host evidence for the exact D-43 binding is supplied to the admission service; only a combined provider+host pass sets `admitted`, while a deployment/configuration/allowlist/permission mismatch atomically returns it to `pending|blocked` before another sync.

### Change boundaries

- `deploy/compose.yaml`
- `backend/internal/health/`
- `docs/operations/`

These are planned paths. Shared contracts: `spec/001-want-keep-mvp/contracts.en.md`; architecture and commands: `constraints.en.md`. Change only the behavior owner and affected tests; an unresolved contract requires updated evidence and stops dependent implementation.

### Linked requirements

- **REQ-022:** Full transactions and receipts may be sent to OpenAI; access secrets and unrelated private data are excluded.
- **REQ-048:** Integrations and the browser collector perform authorized read operations only.
- **REQ-050:** Files, source keys, sessions and financial records are protected against unauthorized access.
- **REQ-051:** AI is limited to $50/month and degrades to a waiting queue without stopping ordinary accounting.
- **REQ-056:** Deployment fits $40/month for a server in DE/NL/BG; no separately paid data sources are used.
- **REQ-058:** Operational status exposes import, AI, FX, backup failures and spend without leaking financial content.
- **REQ-061:** Repeated jobs, restarts and concurrent changes cannot create duplicate financial effects.
- **REQ-062:** Architecture uses Go/PostgreSQL, React/TypeScript/Vite and a separate Playwright collector with dependencies pointing toward the domain.
- **REQ-063:** User, household and membership are separate models; the two-member limit is configured.
- **REQ-073:** Both manage connections; the external-account owner performs bank authentication without exposing secrets to the partner or AI.
- **REQ-076:** Household scope is checked for APIs, files, AI, jobs and external IDs independently of supplied actor/owner fields.
- **REQ-088:** Provider sync is allowed only by a current server-side admission bound to verified adapter, contract, allowlist, configuration and environment revisions.

### Acceptance criteria

A criterion link establishes coverage; research or a partial task does not prove the entire product criterion. This task's exact outcome is specified in verification below.

#### AC-022

- **Given:** A transaction and receipt are available to the owner; source credentials are stored elsewhere.
- **When:** An AI request and diagnostic record are produced.
- **Then:** The request contains only permitted transaction/document data; keys and sessions appear in neither request nor logs; OpenAI retention policy is disclosed.
- **Level:** `contract`.

#### AC-048

- **Given:** The collector has a personal-account session with broader provider permissions.
- **When:** A payment request, unapproved route or MFA/CAPTCHA appears.
- **Then:** Payments and unknown routes are blocked; MFA/CAPTCHA is handed to the owner and that source pauses; other sources continue.
- **Level:** `integration`.

#### AC-050

- **Given:** A private receipt and an active source connection exist.
- **When:** A direct file URL, unauthenticated export, logs and disconnection are checked.
- **Then:** Unauthenticated access fails; secrets are encrypted and not logged; disconnecting stops further collection.
- **Level:** `integration`.

#### AC-051

- **Given:** OpenAI is unavailable or the allowed budget including in-flight reservations is exhausted.
- **When:** A new import, manual expense and AI request arrive.
- **Then:** Accounting and calculations remain available; AI status is waiting; no new paid calls exceed the allowed reservation; unknown cost is not silently released.
- **Level:** `integration`.

#### AC-056

- **Given:** A concrete plan, region, billing currency and taxes are selected.
- **When:** Operating costs are checked for hundreds of monthly transactions.
- **Then:** A dated all-in server estimate is at most $40; OpenAI has a separate $50 cap; a mandatory paid data source remains a blocker.
- **Level:** `manual`.

#### AC-058

- **Given:** A connector fails, AI is delayed and a backup is stale.
- **When:** System health and diagnostic logs are inspected.
- **Then:** Separate failures and recovery actions are visible; logs contain identifiers/codes, not receipts, keys or financial message text.
- **Level:** `integration`.

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

#### AC-077

- **Given:** Two users belong to one household.
- **When:** Schema, authorization and the membership limit are inspected.
- **Then:** There are no partner1/partner2 fields or specific-user branches; roles and ownership are separate from identity. Exit, replacement and new roles are not implemented.
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
- **Then:** The server returns `provider_not_admitted`, never starts the collector and creates no posting. Only the admission service sets `admitted` after task-4.x provider evidence and task-8.x host evidence for the exact binding; any mismatch closes sync again.
- **Level:** `integration+security`.

### Verification

```sh
make check-deploy && make test-integration AREA=health
```

Containers/limits are verified; connector/AI failure does not stop accounting; no public DB or logged secrets.

The task-1.1 foundation already provides `make check-deploy` and `make test-integration` as fail-fast interfaces. Until task-8.1 implements them, these suites must fail clearly; command presence does not establish deployment, hardening, managed DB or runtime health.

### Handoff to the next agent

Record changed contracts, commands/results, limitations, unresolved questions and unblocked dependencies. Update both languages and traceability. Close the task only with evidence of its outcome; GitHub Closed alone does not mean the MVP is Ready.

**Commit boundary:** this task's logical boundary; this card does not authorize commit/push/deploy, which require current user authorization.
