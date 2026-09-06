<!-- want-keep-task: task-6.1 -->
# task-6.1 — Реализовать курсы и валютную оценку / Implement rates and currency valuation

## RU

Сохранять историческую оценку и показывать текущие эквиваленты/котировки.

**Состояние:** Заблокировано зависимостями и проверкой SDD Ready; реализация не начата.

**Зависимости:** `task-0.7`, `task-2.2`, `task-3.2`.

**Тип:** `implementation`.

### Изменение и контракты

Реализовать подтверждённые источники исторических/текущих курсов и сервисных bid/ask по контракту. Фиксировать source/time/direction/amount/fee coverage. Историческую оценку не менять при обновлении текущей цены; исправление курса аудировать отдельно. Missing/stale/unsupported не считать нулём или peg; native amount остаётся доступен. Перевод/обмен использует фактические legs.

### Границы изменений

- `backend/internal/valuation/`
- `backend/internal/integrations/rates/`

Это планируемые пути. Общие контракты: `spec/001-want-keep-mvp/contracts.md`; архитектура и команды: `constraints.md`. Менять только владельца поведения и затронутые тесты; при незакрытом контракте обновить evidence и остановить зависимую реализацию.

### Связанные требования

- **REQ-002:** Учёт поддерживает RUB, USD, USDT и BTC; наличные и банковские деньги различаются счетами.
- **REQ-003:** Общую валюту отображения можно переключать между четырьмя поддерживаемыми активами.
- **REQ-007:** Обмен и P2P-конвертация собственных денег сохраняют обе валютные суммы, фактический курс и комиссии.
- **REQ-010:** Возврат уменьшает расходы исходного месяца покупки, сохраняя дату реального поступления денег.
- **REQ-016:** Позиции чека распределяют одну оплаченную сумму по категориям без дублирования итога.
- **REQ-037:** Исторические расходы используют зафиксированную оценку на дату операции, текущий капитал — актуальную оценку.
- **REQ-038:** Курсы обмена учитывают направление, сервис, время, сумму применимости и известные комиссии.
- **REQ-039:** Отсутствующие курсы и неподдерживаемые активы не превращаются в нулевые суммы или условный паритет USDT/USD.
- **REQ-059:** Денежные расчёты используют точную арифметику и явные правила округления на границах.

### Критерии приёмки

Связь с критерием задаёт покрытие; исследование или частичная задача не доказывает весь критерий продукта. Точный результат этой задачи указан ниже в проверке.

#### AC-003

- **Дано:** Для всех необходимых пар есть актуальная оценка.
- **Когда:** Владелец переключает RUB на USD, USDT и BTC.
- **Тогда:** Меняется эквивалент итогов, исходные суммы операций и счетов сохраняются.
- **Уровень:** `end-to-end`.

#### AC-007

- **Дано:** Собственные RUB 9 000 обменены на USDT 100, комиссия RUB 50.
- **Когда:** Приходят банковская и криптовалютная стороны подтверждённого обмена.
- **Тогда:** Основные суммы не становятся расходом/доходом; курс RUB 90/USDT отделён от комиссии; неоднозначная связь требует уточнения.
- **Уровень:** `integration`.

#### AC-010

- **Дано:** В августе оплачен расход RUB 1 000.
- **Когда:** В сентябре получен связанный частичный возврат RUB 400.
- **Тогда:** Расход августа становится RUB 600; движение RUB +400 остаётся в сентябре; пересчёт и связь доступны в истории.
- **Уровень:** `integration`.

#### AC-037

- **Дано:** Расход USD 10 оценён в RUB 900; текущий курс стал RUB 100/USD.
- **Когда:** Обновляются котировки и дашборд.
- **Тогда:** Исторический расход остаётся RUB 900, USD-остаток переоценивается; видны источник, время курса и отдельное курсовое изменение.
- **Уровень:** `integration`.

#### AC-038

- **Дано:** Два сервиса дают разные bid/ask и один не раскрывает комиссию.
- **Когда:** Владелец сравнивает RUB → USDT.
- **Тогда:** Показаны сопоставимые направления и свежесть; неизвестная комиссия не считается нулевой; недоступная котировка не заменяется обещанием рыночного курса.
- **Уровень:** `integration`.

#### AC-039

- **Дано:** В источнике есть неподдерживаемый актив, для USDT/USD отсутствует курс.
- **Когда:** Строится общая оценка.
- **Тогда:** Исходные данные сохранены, покрытие оценки обозначено неполным; нет скрытого нуля или автоматического курса 1:1.
- **Уровень:** `integration`.

#### AC-059

- **Дано:** Есть дробные BTC, USDT, процентное начисление и распределение чека.
- **Когда:** Данные проходят API, базу и повторный расчёт.
- **Тогда:** Исходная точность не теряется; JSON-суммы не проходят binary float; распределения сходятся точно, округление отображения не меняет журнал.
- **Уровень:** `unit+contract`.

#### AC-065

- **Дано:** Покупка USD 10 распределена по двум категориям и оценена в RUB 900.
- **Когда:** Позже возвращено USD 4 за известную позицию; текущий курс иной.
- **Тогда:** Историческая категория уменьшается на исходную стоимость возвращённой части RUB 360; реальные поступления и валютная разница сохраняются отдельно; превышение суммы возвратов блокируется.
- **Уровень:** `unit+integration`.

#### AC-074

- **Дано:** RUB-сумма известна, отсутствует исторический BTC-кросс.
- **Когда:** Владелец выбирает отчёт в BTC.
- **Тогда:** Нативный факт сохранён и доступен; эквивалент/общий итог обозначен неполным; текущий курс не подставлен вместо исторического.
- **Уровень:** `unit+end-to-end`.

### Проверка результата

```sh
make test-go PKG=./internal/valuation/... && make test-contract PROVIDER=rates
```

Все четыре валюты, cross-rates, partial refund и отсутствие исторической цены проверены без изменения native ledger.

Команды `make` — будущий контракт, создаваемый task-1.1; сейчас они не существуют. Live/paid/manual проверки отдельно фиксируют доступ и фактический результат. Исследования не обходят блокер отсутствующего доступа.

### Передача следующему агенту

Записать изменённые контракты, команды и результаты, ограничения, незакрытые вопросы и разблокированные зависимости. Обновить обе языковые версии и трассировку. Закрывать задачу только по доказательству её результата; GitHub Closed само по себе не означает Ready MVP.

**Commit boundary:** логическая граница этой задачи; commit/push/deploy не разрешены данной карточкой и требуют действующей авторизации пользователя.

## EN

Preserve historical valuation and show current equivalents/quotes.

**Status:** Blocked by dependencies and the SDD Ready gate; implementation has not started.

**Dependencies:** `task-0.7`, `task-2.2`, `task-3.2`.

**Kind:** `implementation`.

### Change and contracts

Implement verified historical/current rate sources and provider bid/ask contracts. Retain source/time/direction/amount/fee coverage. Current-price refresh does not change historical valuation; audit rate corrections separately. Missing/stale/unsupported is neither zero nor a peg; native amounts remain available. Transfers/exchanges use actual legs.

### Change boundaries

- `backend/internal/valuation/`
- `backend/internal/integrations/rates/`

These are planned paths. Shared contracts: `spec/001-want-keep-mvp/contracts.en.md`; architecture and commands: `constraints.en.md`. Change only the behavior owner and affected tests; an unresolved contract requires updated evidence and stops dependent implementation.

### Linked requirements

- **REQ-002:** Accounting supports RUB, USD, USDT and BTC; cash and bank money are separate accounts.
- **REQ-003:** The reporting currency can switch among the four supported assets.
- **REQ-007:** Exchange and P2P conversion of owned money preserve both currency amounts, the actual rate and fees.
- **REQ-010:** A refund reduces expenses in the purchase month while preserving the actual cash receipt date.
- **REQ-016:** Receipt items allocate one paid amount across categories without duplicating the total.
- **REQ-037:** Historical expenses use a fixed transaction-date valuation; current wealth uses a current valuation.
- **REQ-038:** Exchange quotes include direction, provider, timestamp, applicable amount and known fees.
- **REQ-039:** Missing rates and unsupported assets never become zero amounts or an assumed USDT/USD peg.
- **REQ-059:** Money calculations use exact arithmetic and explicit boundary rounding rules.

### Acceptance criteria

A criterion link establishes coverage; research or a partial task does not prove the entire product criterion. This task's exact outcome is specified in verification below.

#### AC-003

- **Given:** A current valuation exists for every required pair.
- **When:** The owner switches RUB to USD, USDT and BTC.
- **Then:** Equivalent totals change while original account and transaction amounts remain unchanged.
- **Level:** `end-to-end`.

#### AC-007

- **Given:** Owned RUB 9,000 is exchanged for USDT 100 with a RUB 50 fee.
- **When:** The bank and crypto legs of a confirmed exchange arrive.
- **Then:** Principal is not income/expense; the RUB 90/USDT rate is separate from the fee; an ambiguous match requires clarification.
- **Level:** `integration`.

#### AC-010

- **Given:** A RUB 1,000 expense was paid in August.
- **When:** A linked RUB 400 partial refund is received in September.
- **Then:** August expense becomes RUB 600; the RUB +400 cash movement stays in September; the recalculation and link are auditable.
- **Level:** `integration`.

#### AC-037

- **Given:** A USD 10 expense was valued at RUB 900; the current rate becomes RUB 100/USD.
- **When:** Quotes and dashboard refresh.
- **Then:** Historical expense remains RUB 900 and the USD balance is revalued; rate source/time and separate FX change are visible.
- **Level:** `integration`.

#### AC-038

- **Given:** Two providers have different bid/ask quotes and one omits its fee.
- **When:** The owner compares RUB → USDT.
- **Then:** Directions and freshness are comparable; an unknown fee is not treated as zero; an unavailable quote is not replaced by a promise of market execution.
- **Level:** `integration`.

#### AC-039

- **Given:** A source contains an unsupported asset and no USDT/USD rate is available.
- **When:** A total valuation is built.
- **Then:** Raw data is retained and valuation coverage is incomplete; no hidden zero or automatic 1:1 rate is used.
- **Level:** `integration`.

#### AC-059

- **Given:** Fractional BTC, USDT, interest accrual and receipt allocation exist.
- **When:** Data traverses API, storage and recalculation.
- **Then:** Original precision survives; JSON money never traverses binary floats; allocations reconcile exactly and display rounding does not alter the ledger.
- **Level:** `unit+contract`.

#### AC-065

- **Given:** A USD 10 purchase is split across two categories and valued at RUB 900.
- **When:** USD 4 is later refunded for a known item at a different current rate.
- **Then:** The historical category decreases by the refunded original value RUB 360; actual cash receipts and FX difference stay separate; excess cumulative refunds are rejected.
- **Level:** `unit+integration`.

#### AC-074

- **Given:** A RUB amount is known but its historical BTC cross-rate is missing.
- **When:** The owner selects BTC reporting.
- **Then:** Native actuals remain available; equivalent/total is marked incomplete; a current rate is not substituted for the historical rate.
- **Level:** `unit+end-to-end`.

### Verification

```sh
make test-go PKG=./internal/valuation/... && make test-contract PROVIDER=rates
```

All four currencies, cross-rates, partial refund and missing historical price pass without changing the native ledger.

The `make` commands are a future contract established by task-1.1; they do not exist yet. Live/paid/manual checks separately record access and actual outcomes. Research does not bypass missing-access blockers.

### Handoff to the next agent

Record changed contracts, commands/results, limitations, unresolved questions and unblocked dependencies. Update both languages and traceability. Close the task only with evidence of its outcome; GitHub Closed alone does not mean the MVP is Ready.

**Commit boundary:** this task's logical boundary; this card does not authorize commit/push/deploy, which require current user authorization.
