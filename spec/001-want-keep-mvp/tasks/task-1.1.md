<!-- want-keep-task: task-1.1 -->
# task-1.1 — Создать структуру проекта и команды проверки / Create project structure and verification commands

## RU

Поднять воспроизводимую основу Go, React и сборщика без продуктовых заглушек, выдаваемых за функции.

**Состояние:** Техническая основа реализована; task-0.10 завершила общий SDD gate. Следующая задача по плану — task-1.2.

**Зависимости:** `task-0.10`.

**Тип:** `implementation`.

### Изменение и контракты

Создать выбранную модульную структуру и закрепить совместимые версии инструментов/зависимостей. Реализовать контракт Makefile из constraints для Go, web, collector, интеграционных и E2E-проверок; CI запускает доступные проверки. Зафиксировать OpenAPI-source → generation порядок и отдельные окружения; runtime-команды документируются только после появления реализации.

### Границы изменений

- `.nvmrc`
- `backend/go.mod`
- `web/package.json`
- `collector/package.json`
- `Makefile`
- `.github/workflows/ci.yml`
- `api/`
- `config/environments/`
- `scripts/check-openapi-state.sh`
- `spec/001-want-keep-mvp/evidence/task-1.1-foundation.md`

Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-054:** Интерфейс, чат и документация поддерживают RU/EN без изменения финансовой семантики.
- **REQ-056:** Развёртывание укладывается в $40/месяц на сервер в DE/NL/BG; отдельные платные источники не используются.
- **REQ-062:** Архитектура использует Go/PostgreSQL, React/TypeScript/Vite и отдельный Playwright-сборщик с зависимостями к домену.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

#### AC-054

- **Дано:** Есть русская и английская версии одной операции, бюджета и ошибки.
- **Когда:** Переключается язык.
- **Тогда:** Суммы, даты, валюты и смысл совпадают; форматирование локализовано, идентификаторы и категории пользователя не переводятся с потерей данных.
- **Уровень:** `end-to-end+static`.

#### AC-056

- **Дано:** Выбран конкретный тариф, регион, валюта счёта и налоги.
- **Когда:** Проверяется эксплуатационная смета для сотен операций в месяц.
- **Тогда:** Зафиксирована датированная полная смета сервера не выше $40; OpenAI учитывается отдельно до $50; обязательный платный источник остаётся блокером.
- **Уровень:** `manual`.

#### AC-062

- **Дано:** Создана структура приложения и контракты компонентов.
- **Когда:** Проверяются зависимости и публичные интерфейсы.
- **Тогда:** Домен не импортирует HTTP, SQL, UI, OpenAI SDK или браузерные типы; адаптеры маппят внешние модели; сборщик не владеет финансовыми решениями.
- **Уровень:** `static`.

### Проверка результата

```sh
make check
```

Локальный clean bootstrap и CI выполняют документированные команды; архитектурные границы и lockfiles согласованы.

Интерфейс `make` создан task-1.1. Отсутствующая suite завершается ошибкой; live/paid/manual проверки отдельно фиксируют доступ и фактический результат. Исследования не обходят блокер отсутствующего доступа.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Establish reproducible Go, React and collector foundations without presenting stubs as features.

**Status:** The technical foundation is implemented; task-0.10 completed the shared SDD gate. The next planned task is task-1.2.

**Dependencies:** `task-0.10`.

**Kind:** `implementation`.

### Change and contracts

Create the chosen modular layout and pin compatible tool/dependency versions. Implement the constraints Makefile contract for Go, web, collector, integration and E2E checks; CI runs available checks. Establish OpenAPI-source → generation order and separate environments; document runtime commands only once implemented.

### Change boundaries

- `.nvmrc`
- `backend/go.mod`
- `web/package.json`
- `collector/package.json`
- `Makefile`
- `.github/workflows/ci.yml`
- `api/`
- `config/environments/`
- `scripts/check-openapi-state.sh`
- `spec/001-want-keep-mvp/evidence/task-1.1-foundation.md`

Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-054:** UI, chat and documentation support RU/EN without changing financial semantics.
- **REQ-056:** Deployment fits $40/month for a server in DE/NL/BG; no separately paid data sources are used.
- **REQ-062:** Architecture uses Go/PostgreSQL, React/TypeScript/Vite and a separate Playwright collector with dependencies pointing toward the domain.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

#### AC-054

- **Given:** Russian and English versions of the same transaction, budget and error exist.
- **When:** The language is switched.
- **Then:** Amounts, dates, currencies and meaning agree; formatting is localized while IDs and owner categories are not destructively translated.
- **Level:** `end-to-end+static`.

#### AC-056

- **Given:** A concrete plan, region, billing currency and taxes are selected.
- **When:** Operating costs are checked for hundreds of monthly transactions.
- **Then:** A dated all-in server estimate is at most $40; OpenAI has a separate $50 cap; a mandatory paid data source remains a blocker.
- **Level:** `manual`.

#### AC-062

- **Given:** Application structure and component contracts exist.
- **When:** Dependencies and public interfaces are checked.
- **Then:** Domain imports no HTTP, SQL, UI, OpenAI SDK or browser types; adapters map external models; the collector owns no financial decisions.
- **Level:** `static`.

### Verification

```sh
make check
```

Clean local bootstrap and CI execute documented commands; architectural boundaries and lockfiles agree.

The `make` interface is provided by task-1.1. A missing suite fails explicitly; live/paid/manual checks separately record access and actual outcomes. Research does not bypass missing-access blockers.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
