<!-- want-keep-task: task-2.5 -->
# task-2.5 — Сверять журнал с балансом источника / Reconcile the ledger to source balances

## RU

Показывать полноту истории и объяснимые расхождения.

**Состояние:** Заблокировано зависимостями и проверкой SDD Ready; реализация не начата.

**Зависимости:** `task-2.3`.

**Тип:** `implementation`.

### Изменение и контракты

Сопоставлять баланс и операции на согласованный момент с учётом pending/locked и покрытого периода. Разницу хранить как состояние сверки, запускать ограниченный повтор истории и предлагать объяснение; не создавать молчаливую balancing-запись. Корректировка требует доказанной причины или решения владельца и проходит обычный аудит.

### Границы изменений

- `backend/internal/reconciliation/`

Это планируемые пути. Общие контракты: `spec/001-want-keep-mvp/contracts.md`; архитектура и команды: `constraints.md`. Менять только владельца поведения и затронутые тесты; при незакрытом контракте обновить evidence и остановить зависимую реализацию.

### Связанные требования

- **REQ-004:** Начало учёта задаётся датой; начальные остатки отделены от доходов и расходов.
- **REQ-005:** Счета показывают собственные, доступные, заблокированные и заёмные средства в пределах данных источника.
- **REQ-013:** Расхождение журнала и баланса источника расследуется без скрытого автоматического выравнивания.
- **REQ-040:** Каждый источник обновляется раз в час и по запросу с видимым временем успешного обновления.
- **REQ-041:** История сохраняет границы покрытия, курсоры, пробелы и статусы источника.
- **REQ-058:** Операционные статусы показывают ошибки импорта, AI, курсов, резервирования и расходы без утечки финансового содержимого.

### Критерии приёмки

Связь с критерием задаёт покрытие; исследование или частичная задача не доказывает весь критерий продукта. Точный результат этой задачи указан ниже в проверке.

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

#### AC-013

- **Дано:** Учёт показывает RUB 900, источник RUB 1 000 на сопоставимый момент.
- **Когда:** Завершается сверка.
- **Тогда:** Показаны оба остатка и разница RUB 100; повторно проверяется история; корректирующая запись требует установленной причины или решения владельца.
- **Уровень:** `integration`.

#### AC-040

- **Дано:** Два источника доступны, третий требует повторного входа.
- **Когда:** Срабатывает расписание и одновременно нажата кнопка обновления.
- **Тогда:** Нет параллельного дублирования одного задания; доступные источники обновлены, проблемный имеет отдельный статус и старый timestamp.
- **Уровень:** `integration`.

#### AC-041

- **Дано:** Источник выдаёт несколько страниц с ограничением глубины; второй запрос завершился ошибкой.
- **Когда:** Импорт возобновляется.
- **Тогда:** Подтверждённые страницы сохранены без дублей; курсор не перескакивает пропуск; неполная история и её границы видны.
- **Уровень:** `integration`.

#### AC-058

- **Дано:** Сломан один коннектор, задержан AI и устарела копия.
- **Когда:** Открывается состояние системы и читаются диагностические логи.
- **Тогда:** Видны отдельные проблемы и действия восстановления; логи содержат идентификаторы/коды, а не чеки, ключи или тексты финансовых сообщений.
- **Уровень:** `integration`.

### Проверка результата

```sh
make test-integration AREA=reconciliation
```

Смещение времени снимка не создаёт ложный доход; пробелы и настоящая разница видны; корректировка не происходит без основания.

Команды `make` — будущий контракт, создаваемый task-1.1; сейчас они не существуют. Live/paid/manual проверки отдельно фиксируют доступ и фактический результат. Исследования не обходят блокер отсутствующего доступа.

### Передача следующему агенту

Записать изменённые контракты, команды и результаты, ограничения, незакрытые вопросы и разблокированные зависимости. Обновить обе языковые версии и трассировку. Закрывать задачу только по доказательству её результата; GitHub Closed само по себе не означает Ready MVP.

**Commit boundary:** логическая граница этой задачи; commit/push/deploy не разрешены данной карточкой и требуют действующей авторизации пользователя.

## EN

Expose history completeness and explainable discrepancies.

**Status:** Blocked by dependencies and the SDD Ready gate; implementation has not started.

**Dependencies:** `task-2.3`.

**Kind:** `implementation`.

### Change and contracts

Compare balances and transactions at a consistent instant considering pending/locked amounts and covered periods. Store discrepancies as reconciliation state, request bounded history replay and propose explanations; never silently create balancing entries. Adjustments require an established cause or owner decision and normal audit.

### Change boundaries

- `backend/internal/reconciliation/`

These are planned paths. Shared contracts: `spec/001-want-keep-mvp/contracts.en.md`; architecture and commands: `constraints.en.md`. Change only the behavior owner and affected tests; an unresolved contract requires updated evidence and stops dependent implementation.

### Linked requirements

- **REQ-004:** Accounting starts on a selected date; opening balances are separate from income and expenses.
- **REQ-005:** Accounts distinguish owned, available, locked and borrowed amounts where the source provides them.
- **REQ-013:** Ledger/source balance discrepancies are investigated without hidden automatic balancing.
- **REQ-040:** Each source refreshes hourly and on demand with a visible last-success timestamp.
- **REQ-041:** History retains coverage boundaries, cursors, gaps and source status.
- **REQ-058:** Operational status exposes import, AI, FX, backup failures and spend without leaking financial content.

### Acceptance criteria

A criterion link establishes coverage; research or a partial task does not prove the entire product criterion. This task's exact outcome is specified in verification below.

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

#### AC-013

- **Given:** The ledger shows RUB 900 and the source RUB 1,000 at a comparable instant.
- **When:** Reconciliation completes.
- **Then:** Both balances and the RUB 100 discrepancy are shown; history is rechecked; an adjustment requires an established cause or the owner's decision.
- **Level:** `integration`.

#### AC-040

- **Given:** Two sources are available and a third requires sign-in again.
- **When:** The schedule fires while the refresh button is pressed.
- **Then:** The same job is not duplicated concurrently; available sources refresh and the failing source has its own status and old timestamp.
- **Level:** `integration`.

#### AC-041

- **Given:** A source provides paginated history with a retention limit; the second request fails.
- **When:** Import resumes.
- **Then:** Confirmed pages remain without duplicates; the cursor does not skip the gap; incomplete history and its boundaries are visible.
- **Level:** `integration`.

#### AC-058

- **Given:** A connector fails, AI is delayed and a backup is stale.
- **When:** System health and diagnostic logs are inspected.
- **Then:** Separate failures and recovery actions are visible; logs contain identifiers/codes, not receipts, keys or financial message text.
- **Level:** `integration`.

### Verification

```sh
make test-integration AREA=reconciliation
```

Snapshot timing does not create false income; gaps and real differences are visible; no unsupported adjustment occurs.

The `make` commands are a future contract established by task-1.1; they do not exist yet. Live/paid/manual checks separately record access and actual outcomes. Research does not bypass missing-access blockers.

### Handoff to the next agent

Record changed contracts, commands/results, limitations, unresolved questions and unblocked dependencies. Update both languages and traceability. Close the task only with evidence of its outcome; GitHub Closed alone does not mean the MVP is Ready.

**Commit boundary:** this task's logical boundary; this card does not authorize commit/push/deploy, which require current user authorization.
