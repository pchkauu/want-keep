<!-- want-keep-task: task-0.9 -->
# task-0.9 — Проверить инфраструктуру и бюджет сервера / Verify infrastructure and server budget

## RU

Подобрать проверяемую конфигурацию в DE/NL/BG до $40 с копиями на Mac.

**Состояние:** Исследование завершено 2026-09-07 с эксплуатационными блокерами. Владелец выбрал сервер приложения в Германии 2 vCPU/4 ГБ/50 ГБ и managed PostgreSQL 1 vCPU/2 ГБ/20 ГБ в той же private VPC. Смета: 2 520 ₽/мес. по годовому тарифу; консервативно без скидки и с резервом 10% — 3 055.56 ₽/$35.29 при зафиксированном курсе, ниже $40. Текущий VPS 1 vCPU/<1 ГБ исследован read-only и не допущен как production. HOST-B01–HOST-B06 переданы task-8.1–task-8.3 и task-0.10/task-4.1; безопасная достижимость Alfa, hardening, нагрузка и backup/restore runtime не подтверждены. task-0.9 завершена, но AC приложения и MVP остаются Not Ready.

**Зависимости:** нет.

**Тип:** `research`.

### Изменение и контракты

Evidence/hosting.md и .en.md фиксируют выбранные владельцем VPS 2 vCPU/4 ГБ/50 ГБ и managed PostgreSQL 1 vCPU/2 ГБ/20 ГБ в одной немецкой private VPC, датированные цены/НДС/IP/курс и смету с резервом. Текущий VPS проверен read-only по ресурсам, ОС, TLS, listeners, SSH/firewall/update и публичной достижимости OpenAI/rates/платформ без секретов. Safe Alfa DNS/TLS остаётся блокером. Backup contract: Mac-initiated hourly pull, streaming logical dump и immutable attachments, manifest/checksums, отдельный recovery key, retention 48 hourly/30 daily/8 weekly/12 monthly и лимит 20 GiB с безопасным отказом. HOST-B01–HOST-B06 назначают provisioning, hardening, load, Alfa route и backup/restore runtime следующим задачам. Исследование ничего не покупает и не меняет на сервере; его завершение не доказывает AC приложения или Ready.

### Границы изменений

- `spec/001-want-keep-mvp/evidence/hosting.md`
- `spec/001-want-keep-mvp/evidence/hosting.en.md`

Это планируемые пути. Общие контракты: `spec/001-want-keep-mvp/contracts.md`; архитектура и команды: `constraints.md`. Менять только владельца поведения и затронутые тесты; при незакрытом контракте обновить evidence и остановить зависимую реализацию.

### Связанные требования

- **REQ-048:** Интеграции и браузерный сборщик выполняют только разрешённые операции чтения.
- **REQ-056:** Развёртывание укладывается в $40/месяц на сервер в DE/NL/BG; отдельные платные источники не используются.
- **REQ-057:** Зашифрованная резервная копия выгружается на MacBook ежечасно при его доступности; восстановление проверяется.
- **REQ-058:** Операционные статусы показывают ошибки импорта, AI, курсов, резервирования и расходы без утечки финансового содержимого.

### Критерии приёмки

Связь с критерием задаёт покрытие; исследование или частичная задача не доказывает весь критерий продукта. Точный результат этой задачи указан ниже в проверке.

#### AC-048

- **Дано:** Сборщик имеет сессию личного кабинета с более широкими внешними правами.
- **Когда:** Возникают запрос на платёж, неподтверждённый маршрут или MFA/CAPTCHA.
- **Тогда:** Платёж и неизвестный маршрут блокируются; MFA/CAPTCHA передаётся владельцу, источник приостанавливается; остальные источники продолжают работать.
- **Уровень:** `integration`.

#### AC-056

- **Дано:** Выбран конкретный тариф, регион, валюта счёта и налоги.
- **Когда:** Проверяется эксплуатационная смета для сотен операций в месяц.
- **Тогда:** Зафиксирована датированная полная смета сервера не выше $40; OpenAI учитывается отдельно до $50; обязательный платный источник остаётся блокером.
- **Уровень:** `manual`.

#### AC-057

- **Дано:** Есть база, вложения и MacBook, который временно недоступен.
- **Когда:** Создаются копии, Mac возвращается в сеть, затем проводится восстановление.
- **Тогда:** Показан возраст последней полной копии; после возвращения копирование возобновляется; восстановлены согласованные данные и вложения до четырёх часов; часовой RPO заявляется только при доступном Mac.
- **Уровень:** `integration+manual`.

#### AC-058

- **Дано:** Сломан один коннектор, задержан AI и устарела копия.
- **Когда:** Открывается состояние системы и читаются диагностические логи.
- **Тогда:** Видны отдельные проблемы и действия восстановления; логи содержат идентификаторы/коды, а не чеки, ключи или тексты финансовых сообщений.
- **Уровень:** `integration`.

### Проверка результата

```sh
make docs-check
```

RU/EN evidence совпадают по HOST-E01–HOST-E15 и HOST-B01–HOST-B06; выбранные компоненты, смета, security/backup gates и ограничения reachability воспроизводимы. IP, секреты и финансовые данные не опубликованы; provisioning/runtime не выданы за пройденные AC.

Основа task-1.1 уже предоставляет make-интерфейс. `make docs-check` существует и проверяет этот исследовательский результат; live/manual server evidence отделено от CI. Команды будущих deploy/integration/backup/restore suites намеренно завершаются ошибкой до реализации соответствующих задач, а не сообщают пустой успех.

### Передача следующему агенту

Записать изменённые контракты, команды и результаты, ограничения, незакрытые вопросы и разблокированные зависимости. Обновить обе языковые версии и трассировку. Закрывать задачу только по доказательству её результата; GitHub Closed само по себе не означает Ready MVP.

**Commit boundary:** логическая граница этой задачи; commit/push/deploy не разрешены данной карточкой и требуют действующей авторизации пользователя.

## EN

Identify a verifiable DE/NL/BG configuration under $40 with Mac backups.

**Status:** Research completed on 2026-09-07 with operational blockers. The owner selected a German 2 vCPU/4 GB/50 GB application server and 1 vCPU/2 GB/20 GB managed PostgreSQL in the same private VPC. Estimate: RUB 2,520/month under annual pricing; conservative no-discount total plus 10% reserve is RUB 3,055.56/$35.29 at the recorded FX rate, below $40. The current 1 vCPU/<1 GB VPS was inspected read-only and is not admitted for production. HOST-B01–HOST-B06 are assigned to task-8.1–task-8.3 and task-0.10/task-4.1; safe Alfa reachability, hardening, load and backup/restore runtime remain unverified. task-0.9 is complete, but application ACs and the MVP remain Not Ready.

**Dependencies:** none.

**Kind:** `research`.

### Change and contracts

Evidence/hosting.md and .en.md record the owner-selected 2 vCPU/4 GB/50 GB VPS and 1 vCPU/2 GB/20 GB managed PostgreSQL in one German private VPC, dated pricing/VAT/IP/FX and a reserved estimate. The current VPS was inspected read-only for resources, OS, TLS, listeners, SSH/firewall/update state and public OpenAI/rate/platform reachability without secrets. Safe Alfa DNS/TLS remains blocked. Backup contract: Mac-initiated hourly pull, streaming logical dump plus immutable attachments, manifest/checksums, separate recovery key, retention of 48 hourly/30 daily/8 weekly/12 monthly points and a 20 GiB cap with safe failure. HOST-B01–HOST-B06 assign provisioning, hardening, load, Alfa route and backup/restore runtime to later tasks. Research purchases or changes nothing on the server; completion does not prove application ACs or Ready.

### Change boundaries

- `spec/001-want-keep-mvp/evidence/hosting.md`
- `spec/001-want-keep-mvp/evidence/hosting.en.md`

These are planned paths. Shared contracts: `spec/001-want-keep-mvp/contracts.en.md`; architecture and commands: `constraints.en.md`. Change only the behavior owner and affected tests; an unresolved contract requires updated evidence and stops dependent implementation.

### Linked requirements

- **REQ-048:** Integrations and the browser collector perform authorized read operations only.
- **REQ-056:** Deployment fits $40/month for a server in DE/NL/BG; no separately paid data sources are used.
- **REQ-057:** An encrypted backup is pulled to the MacBook hourly while reachable; recovery is tested.
- **REQ-058:** Operational status exposes import, AI, FX, backup failures and spend without leaking financial content.

### Acceptance criteria

A criterion link establishes coverage; research or a partial task does not prove the entire product criterion. This task's exact outcome is specified in verification below.

#### AC-048

- **Given:** The collector has a personal-account session with broader provider permissions.
- **When:** A payment request, unapproved route or MFA/CAPTCHA appears.
- **Then:** Payments and unknown routes are blocked; MFA/CAPTCHA is handed to the owner and that source pauses; other sources continue.
- **Level:** `integration`.

#### AC-056

- **Given:** A concrete plan, region, billing currency and taxes are selected.
- **When:** Operating costs are checked for hundreds of monthly transactions.
- **Then:** A dated all-in server estimate is at most $40; OpenAI has a separate $50 cap; a mandatory paid data source remains a blocker.
- **Level:** `manual`.

#### AC-057

- **Given:** The database, attachments and a temporarily unreachable MacBook exist.
- **When:** Backups are attempted, the Mac reconnects and recovery is rehearsed.
- **Then:** Last complete backup age is visible; copying resumes after reconnection; consistent data and attachments restore within four hours; hourly RPO is claimed only while the Mac is reachable.
- **Level:** `integration+manual`.

#### AC-058

- **Given:** A connector fails, AI is delayed and a backup is stale.
- **When:** System health and diagnostic logs are inspected.
- **Then:** Separate failures and recovery actions are visible; logs contain identifiers/codes, not receipts, keys or financial message text.
- **Level:** `integration`.

### Verification

```sh
make docs-check
```

RU/EN evidence align on HOST-E01–HOST-E15 and HOST-B01–HOST-B06; selected components, estimate, security/backup gates and reachability limits are reproducible. No IPs, secrets or financial data are published, and provisioning/runtime are not presented as passing ACs.

The task-1.1 foundation already provides the make interface. `make docs-check` exists and verifies this research output; live/manual server evidence is separate from CI. Commands for future deploy/integration/backup/restore suites intentionally fail until their owning tasks implement them rather than reporting an empty success.

### Handoff to the next agent

Record changed contracts, commands/results, limitations, unresolved questions and unblocked dependencies. Update both languages and traceability. Close the task only with evidence of its outcome; GitHub Closed alone does not mean the MVP is Ready.

**Commit boundary:** this task's logical boundary; this card does not authorize commit/push/deploy, which require current user authorization.
