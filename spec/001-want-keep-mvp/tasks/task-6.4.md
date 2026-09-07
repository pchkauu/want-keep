<!-- want-keep-task: task-6.4 -->
# task-6.4 — Сравнивать доходность денежных потоков / Compare dated cash-flow returns

## RU

Сопоставлять вложения без ложной доходности от пополнения или FX.

**Состояние:** Не начато; задача ожидает собственные зависимости и entry gates.

**Зависимости:** `task-6.3`.

**Тип:** `implementation`.

### Изменение и контракты

Реализовать XIRR Actual/365 по объединённым потокам одной календарной даты. Cash-flow decimal остаются точными; fractional power считать как `exp((days/365) × ln(1+r))` в decimal context минимум 50 значащих цифр, ROUND_HALF_EVEN, с доказанной общей погрешностью NPV `≤ 1e-24 × max(1, Σ|CF|)`. Требуются положительный и отрицательный потоки и ровно одна смена знака. Решать NPV=0 bisection от `rLow=-1+1e-12` до `rHigh=1 000 000`; bracket требует разные знаки boundary NPV либо границу в tolerance. Остановка: `|NPV| ≤ 1e-12 × max(1, Σ|CF|)` или ширина `≤ 1e-12 × max(1, |rMid|)`, максимум 512 итераций; результат — rMid, HALF_EVEN до 12 знаков после запятой. Если error bound/знак не доказан, нет bracket, несколько смен знака, нулевой период, missing valuation или convergence — объяснённый `unavailable`, не 0%. Проверить irregular 182-day и boundary vectors; показывать native/reporting currency, доход и метод отдельно от FX.

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

`-1000/+1100` за 365 дней даёт 10%; `-1000/+1050` за 182 дня даёт `0.102795595422`. rLow/root-above-rHigh, отсутствие или несколько смен знака, same-day net zero, missing valuation, недоказанный numeric error и no convergence дают заданный boundary outcome либо `unavailable`.

Команды `make` — будущий контракт, создаваемый task-1.1; сейчас они не существуют. Live/paid/manual проверки отдельно фиксируют доступ и фактический результат. Исследования не обходят блокер отсутствующего доступа.

### Передача следующему агенту

Записать изменённые контракты, команды и результаты, ограничения, незакрытые вопросы и разблокированные зависимости. Обновить обе языковые версии и трассировку. Закрывать задачу только по доказательству её результата; GitHub Closed само по себе не означает Ready MVP.

**Commit boundary:** логическая граница этой задачи; commit/push/deploy не разрешены данной карточкой и требуют действующей авторизации пользователя.

## EN

Compare investments without treating contributions or FX as yield.

**Status:** Not started; the task awaits its own dependencies and entry gates.

**Dependencies:** `task-6.3`.

**Kind:** `implementation`.

### Change and contracts

Implement Actual/365 XIRR after aggregating flows on the same calendar date. Cash-flow decimals remain exact; evaluate fractional powers as `exp((days/365) × ln(1+r))` in a decimal context of at least 50 significant digits, ROUND_HALF_EVEN, with a proven total NPV error `≤ 1e-24 × max(1, Σ|CF|)`. Inputs require positive and negative flows and exactly one sign transition. Solve NPV=0 by bisection from `rLow=-1+1e-12` through `rHigh=1,000,000`; a bracket requires opposite boundary NPV signs or a boundary within tolerance. Stop at `|NPV| ≤ 1e-12 × max(1, Σ|CF|)` or interval width `≤ 1e-12 × max(1, |rMid|)`, with at most 512 iterations; return rMid rounded HALF_EVEN to 12 decimal places. If the error bound/sign is not established, no bracket exists, signs change more than once, the period is zero, valuation is missing or convergence fails, return explained `unavailable`, never 0%. Test irregular 182-day and boundary vectors; show native/reporting currency, income and method separately from FX.

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

`-1000/+1100` over 365 days returns 10%; `-1000/+1050` over 182 days returns `0.102795595422`. rLow/root-above-rHigh, no or multiple sign transitions, same-day net zero, missing valuation, unproven numeric error and non-convergence yield the specified boundary outcome or `unavailable`.

The `make` commands are a future contract established by task-1.1; they do not exist yet. Live/paid/manual checks separately record access and actual outcomes. Research does not bypass missing-access blockers.

### Handoff to the next agent

Record changed contracts, commands/results, limitations, unresolved questions and unblocked dependencies. Update both languages and traceability. Close the task only with evidence of its outcome; GitHub Closed alone does not mean the MVP is Ready.

**Commit boundary:** this task's logical boundary; this card does not authorize commit/push/deploy, which require current user authorization.
