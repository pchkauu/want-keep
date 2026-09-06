<!-- want-keep-task: task-0.9 -->
# task-0.9 — Проверить инфраструктуру и бюджет сервера / Verify infrastructure and server budget

## RU

Подобрать проверяемую конфигурацию в DE/NL/BG до $40 с копиями на Mac.

**Состояние:** Исследование — не начато; live-доступ и платные прогоны требуют безопасно предоставленного доступа владельца.

**Зависимости:** нет.

**Тип:** `research`.

### Изменение и контракты

Сопоставить актуальный тариф с Go, PostgreSQL и одним последовательным браузерным worker; включить диск, IP, налог, валюту расчёта и хранение временного backup-набора. Проверить достижимость платформ и OpenAI из разрешённого региона после появления доступа. Описать исходящее pull-копирование с Mac, хранение recovery-ключа отдельно от единственной копии и условный RPO. Ничего не покупать и не развёртывать в рамках исследования.

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
python3 spec/001-want-keep-mvp/tools/spec_tool.py check
```

Есть датированная смета и схема восстановления; неподтверждённая доступность не объявлена рабочей.

Команды `make` — будущий контракт, создаваемый task-1.1; сейчас они не существуют. Live/paid/manual проверки отдельно фиксируют доступ и фактический результат. Исследования не обходят блокер отсутствующего доступа.

### Передача следующему агенту

Записать изменённые контракты, команды и результаты, ограничения, незакрытые вопросы и разблокированные зависимости. Обновить обе языковые версии и трассировку. Закрывать задачу только по доказательству её результата; GitHub Closed само по себе не означает Ready MVP.

**Commit boundary:** логическая граница этой задачи; commit/push/deploy не разрешены данной карточкой и требуют действующей авторизации пользователя.

## EN

Identify a verifiable DE/NL/BG configuration under $40 with Mac backups.

**Status:** Research — not started; live access and paid runs require securely supplied owner access.

**Dependencies:** none.

**Kind:** `research`.

### Change and contracts

Match current pricing to Go, PostgreSQL and one sequential browser worker; include disk, IP, tax, billing currency and temporary backup-set storage. Verify platform/OpenAI reachability from an allowed region once access exists. Describe outbound Mac pull backups, recovery-key custody separate from the sole backup and conditional RPO. Do not purchase or deploy during research.

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
python3 spec/001-want-keep-mvp/tools/spec_tool.py check
```

A dated estimate and recovery design exist; unverified reachability is not reported as working.

The `make` commands are a future contract established by task-1.1; they do not exist yet. Live/paid/manual checks separately record access and actual outcomes. Research does not bypass missing-access blockers.

### Handoff to the next agent

Record changed contracts, commands/results, limitations, unresolved questions and unblocked dependencies. Update both languages and traceability. Close the task only with evidence of its outcome; GitHub Closed alone does not mean the MVP is Ready.

**Commit boundary:** this task's logical boundary; this card does not authorize commit/push/deploy, which require current user authorization.
