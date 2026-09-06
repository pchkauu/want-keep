<!-- want-keep-task: task-6.4 -->
# task-6.4 — Сравнивать доходность денежных потоков / Compare dated cash-flow returns

## RU

Сопоставлять вложения без ложной доходности от пополнения или FX.

**Состояние:** Заблокировано зависимостями и проверкой SDD Ready; реализация не начата.

**Зависимости:** `task-6.3`.

**Тип:** `implementation`.

### Изменение и контракты

Реализовать annualized money-weighted return (XIRR) по dated external contributions/withdrawals и terminal value согласно contracts.md. Показывать исходную/отчётную валюту и фактический доход отдельно. Отдельно обрабатывать отсутствие смены знака, нулевой период, неоднозначные корни и неполную историю; не подставлять 0%. Предположения/метод доступны пользователю.

### Границы изменений

- `backend/internal/returns/`

Это планируемые пути. Общие контракты: `spec/001-want-keep-mvp/contracts.md`; архитектура и команды: `constraints.md`. Менять только владельца поведения и затронутые тесты; при незакрытом контракте обновить evidence и остановить зависимую реализацию.

### Связанные требования

- **REQ-033:** Накопления показывают фактические начисления и прогноз по ставкам, срокам, капитализации и денежным потокам.
- **REQ-034:** Доходность вложений сравнивается с учётом дат денежных потоков и валюты оценки.
- **REQ-037:** Исторические расходы используют зафиксированную оценку на дату операции, текущий капитал — актуальную оценку.
- **REQ-059:** Денежные расчёты используют точную арифметику и явные правила округления на границах.

### Критерии приёмки

Связь с критерием задаёт покрытие; исследование или частичная задача не доказывает весь критерий продукта. Точный результат этой задачи указан ниже в проверке.

#### AC-033

- **Дано:** Есть вклад или Earn с подтверждёнными условиями, пополнением и выводом.
- **Когда:** Рассчитывается доход за период и прогноз.
- **Тогда:** Факт отделён от прогноза и переоценки; смена ставки и капитализация учитываются по условиям; неизвестные условия блокируют точный прогноз.
- **Уровень:** `integration`.

#### AC-034

- **Дано:** Два вложения имеют разные даты пополнений и одинаковый конечный остаток.
- **Когда:** Строится сравнение доходности.
- **Тогда:** Показаны фактический доход и годовая денежно-взвешенная доходность с датами/методом; некорректные или неоднозначные расчёты обозначены недоступными, не нулём.
- **Уровень:** `unit`.

#### AC-037

- **Дано:** Расход USD 10 оценён в RUB 900; текущий курс стал RUB 100/USD.
- **Когда:** Обновляются котировки и дашборд.
- **Тогда:** Исторический расход остаётся RUB 900, USD-остаток переоценивается; видны источник, время курса и отдельное курсовое изменение.
- **Уровень:** `integration`.

#### AC-059

- **Дано:** Есть дробные BTC, USDT, процентное начисление и распределение чека.
- **Когда:** Данные проходят API, базу и повторный расчёт.
- **Тогда:** Исходная точность не теряется; JSON-суммы не проходят binary float; распределения сходятся точно, округление отображения не меняет журнал.
- **Уровень:** `unit+contract`.

### Проверка результата

```sh
make test-go PKG=./internal/returns/...
```

Эталонные денежные потоки дают ожидаемую доходность; неподходящие данные возвращают объяснимую unavailable-оценку.

Команды `make` — будущий контракт, создаваемый task-1.1; сейчас они не существуют. Live/paid/manual проверки отдельно фиксируют доступ и фактический результат. Исследования не обходят блокер отсутствующего доступа.

### Передача следующему агенту

Записать изменённые контракты, команды и результаты, ограничения, незакрытые вопросы и разблокированные зависимости. Обновить обе языковые версии и трассировку. Закрывать задачу только по доказательству её результата; GitHub Closed само по себе не означает Ready MVP.

**Commit boundary:** логическая граница этой задачи; commit/push/deploy не разрешены данной карточкой и требуют действующей авторизации пользователя.

## EN

Compare investments without treating contributions or FX as yield.

**Status:** Blocked by dependencies and the SDD Ready gate; implementation has not started.

**Dependencies:** `task-6.3`.

**Kind:** `implementation`.

### Change and contracts

Implement annualized money-weighted return (XIRR) from dated external contributions/withdrawals and terminal value per contracts.md. Show native/reporting currency and actual income separately. Handle no sign change, zero period, ambiguous roots and incomplete history explicitly; do not substitute 0%. Expose method/assumptions to the owner.

### Change boundaries

- `backend/internal/returns/`

These are planned paths. Shared contracts: `spec/001-want-keep-mvp/contracts.en.md`; architecture and commands: `constraints.en.md`. Change only the behavior owner and affected tests; an unresolved contract requires updated evidence and stops dependent implementation.

### Linked requirements

- **REQ-033:** Savings show actual accruals and forecasts using rates, terms, compounding and cash flows.
- **REQ-034:** Investment returns are compared using dated cash flows and valuation currency.
- **REQ-037:** Historical expenses use a fixed transaction-date valuation; current wealth uses a current valuation.
- **REQ-059:** Money calculations use exact arithmetic and explicit boundary rounding rules.

### Acceptance criteria

A criterion link establishes coverage; research or a partial task does not prove the entire product criterion. This task's exact outcome is specified in verification below.

#### AC-033

- **Given:** A deposit or Earn product has confirmed terms, a top-up and a withdrawal.
- **When:** Period income and forecast are calculated.
- **Then:** Actual income is separate from forecast and revaluation; rate changes and compounding follow the terms; unknown terms prevent an exact forecast.
- **Level:** `integration`.

#### AC-034

- **Given:** Two investments have different top-up dates and the same ending balance.
- **When:** A return comparison is built.
- **Then:** Actual income and annualized money-weighted return show dates/method; invalid or ambiguous calculations are unavailable rather than zero.
- **Level:** `unit`.

#### AC-037

- **Given:** A USD 10 expense was valued at RUB 900; the current rate becomes RUB 100/USD.
- **When:** Quotes and dashboard refresh.
- **Then:** Historical expense remains RUB 900 and the USD balance is revalued; rate source/time and separate FX change are visible.
- **Level:** `integration`.

#### AC-059

- **Given:** Fractional BTC, USDT, interest accrual and receipt allocation exist.
- **When:** Data traverses API, storage and recalculation.
- **Then:** Original precision survives; JSON money never traverses binary floats; allocations reconcile exactly and display rounding does not alter the ledger.
- **Level:** `unit+contract`.

### Verification

```sh
make test-go PKG=./internal/returns/...
```

Reference cash flows give expected returns; unsuitable inputs return an explained unavailable result.

The `make` commands are a future contract established by task-1.1; they do not exist yet. Live/paid/manual checks separately record access and actual outcomes. Research does not bypass missing-access blockers.

### Handoff to the next agent

Record changed contracts, commands/results, limitations, unresolved questions and unblocked dependencies. Update both languages and traceability. Close the task only with evidence of its outcome; GitHub Closed alone does not mean the MVP is Ready.

**Commit boundary:** this task's logical boundary; this card does not authorize commit/push/deploy, which require current user authorization.
