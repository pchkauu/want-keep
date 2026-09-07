<!-- want-keep-task: task-0.7 -->
# task-0.7 — Проверить бесплатные источники курсов / Verify free FX and quote sources

## RU

Подтвердить историческую и текущую оценку RUB, USD, USDT, USDC, BTC и ETH и покрытие обменных котировок.

**Состояние:** Исследование завершено 2026-09-07: CBR/CoinGecko/Frankfurter contract, 18 evidence items и шесть blocker decisions записаны; TradingView отвергнут. FX-B02–FX-B04 / BLK-07 переданы task-0.10; task-6.1 и MVP Not Ready.

**Зависимости:** нет.

**Тип:** `research`.

### Изменение и контракты

Evidence/fx.md и .en.md содержат датированные официальные источники, live boundary probes, матрицу кандидатов, формулу USD-кроссов, failure/cache/audit правила и FX-B01–FX-B06. Основной USD/RUB — XML Банка России; Frankfurter v2 только с providers=CBR — fallback/cross-check; CoinGecko Demo — отдельные BTC/USD, ETH/USD, USDT/USD и USDC/USD current/history не старше 365 дней. Default blend и peg USD/USDT/USDC=1 запрещены; похожие токены, включая USDC.E, не объединять без проверенной identity mapping. TradingView отвергнут: библиотеки требуют внешний datafeed, terms запрещают non-display price referencing. Reference valuation не заменяет provider executable buy/sell с обеими native amounts, applicable amount, timestamp, spread/fee. DE/NL/BG snapshot подтверждён, но не является SLA/VPS runtime. BLK-07 сохраняет FX-B02–FX-B04 под task-0.10; исследование не разблокирует task-6.1 до Ready.

### Границы изменений

- `spec/001-want-keep-mvp/evidence/fx.md`
- `spec/001-want-keep-mvp/evidence/fx.en.md`

Это планируемые пути. Общие контракты: `spec/001-want-keep-mvp/contracts.md`; архитектура и команды: `constraints.md`. Менять только владельца поведения и затронутые тесты; при незакрытом контракте обновить evidence и остановить зависимую реализацию.

### Связанные требования

- **REQ-002:** Учёт поддерживает RUB, USD, USDT, USDC, BTC и ETH; наличные, банковские деньги и платформенные кошельки различаются счетами.
- **REQ-003:** Общую валюту отображения можно переключать между RUB, USD, USDT, USDC, BTC и ETH.
- **REQ-037:** Исторические расходы используют зафиксированную оценку на дату операции, текущий капитал — актуальную оценку.
- **REQ-038:** Курсы обмена учитывают направление, сервис, время, сумму применимости и известные комиссии.
- **REQ-039:** Отсутствующие курсы и неподдерживаемые активы не превращаются в нулевые суммы или условный паритет USD/USDT/USDC.
- **REQ-059:** Денежные расчёты используют точную арифметику и явные правила округления на границах.

### Критерии приёмки

Связь с критерием задаёт покрытие; исследование или частичная задача не доказывает весь критерий продукта. Точный результат этой задачи указан ниже в проверке.

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

- **Дано:** В источнике есть неподдерживаемый USDC.E; для USDT/USD и USDC/USD отсутствуют курсы.
- **Когда:** Строится общая оценка.
- **Тогда:** Исходные данные сохранены, покрытие оценки обозначено неполным; нет скрытого нуля или автоматического курса 1:1. USDC.E не объединён с USDC по похожему символу.
- **Уровень:** `integration`.

#### AC-074

- **Дано:** RUB-сумма известна, отсутствует исторический BTC-кросс.
- **Когда:** Владелец выбирает отчёт в BTC.
- **Тогда:** Нативный факт сохранён и доступен; эквивалент/общий итог обозначен неполным; текущий курс не подставлен вместо исторического.
- **Уровень:** `unit+end-to-end`.

### Проверка результата

```sh
make docs-check
```

RU/EN evidence совпадает по FX-E01–FX-E18 и FX-B01–FX-B06; выбранные источники, даты, Decimal-кроссы и unavailable-поведение воспроизводимы. Реальные rates/keys/cookies/IP не опубликованы. Закрытие research не выдаётся за прохождение application AC или BLK-07/Ready.

make docs-check проверяет SDD. Anonymous current/history/plan-limit и DE/NL/BG probes подтверждают только датированный публичный доступ. Demo key, длительный runtime, crypto history >365d и executable provider quotes не проверены и перечислены как открытые решения.

### Передача следующему агенту

Записать изменённые контракты, команды и результаты, ограничения, незакрытые вопросы и разблокированные зависимости. Обновить обе языковые версии и трассировку. Закрывать задачу только по доказательству её результата; GitHub Closed само по себе не означает Ready MVP.

**Commit boundary:** логическая граница этой задачи; commit/push/deploy не разрешены данной карточкой и требуют действующей авторизации пользователя.

## EN

Verify historical/current RUB, USD, USDT, USDC, BTC and ETH valuation and exchange-quote coverage.

**Status:** Research completed on 2026-09-07: CBR/CoinGecko/Frankfurter contract, 18 evidence items and six blocker decisions recorded; TradingView rejected. FX-B02–FX-B04 / BLK-07 handed to task-0.10; task-6.1 and MVP Not Ready.

**Dependencies:** none.

**Kind:** `research`.

### Change and contracts

Evidence/fx.md and .en.md contain dated official sources, live boundary probes, a candidate matrix, USD-cross formula, failure/cache/audit rules and FX-B01–FX-B06. Primary USD/RUB is Bank of Russia XML; Frankfurter v2 with providers=CBR only is fallback/cross-check; CoinGecko Demo supplies separate BTC/USD, ETH/USD, USDT/USD and USDC/USD current/history no older than 365 days. Default blends and a USD/USDT/USDC=1 peg are forbidden; similar tokens, including USDC.E, are not merged without verified identity mapping. TradingView is rejected: libraries need an external datafeed and terms prohibit non-display price referencing. Reference valuation cannot replace provider-executable buy/sell with both native amounts, applicable amount, timestamp and spread/fee. DE/NL/BG snapshot reachability is established but is not SLA/VPS runtime. BLK-07 retains FX-B02–FX-B04 under task-0.10; research does not unblock task-6.1 before Ready.

### Change boundaries

- `spec/001-want-keep-mvp/evidence/fx.md`
- `spec/001-want-keep-mvp/evidence/fx.en.md`

These are planned paths. Shared contracts: `spec/001-want-keep-mvp/contracts.en.md`; architecture and commands: `constraints.en.md`. Change only the behavior owner and affected tests; an unresolved contract requires updated evidence and stops dependent implementation.

### Linked requirements

- **REQ-002:** Accounting supports RUB, USD, USDT, USDC, BTC and ETH; cash, bank money and platform wallets are separate accounts.
- **REQ-003:** The reporting currency can switch among RUB, USD, USDT, USDC, BTC and ETH.
- **REQ-037:** Historical expenses use a fixed transaction-date valuation; current wealth uses a current valuation.
- **REQ-038:** Exchange quotes include direction, provider, timestamp, applicable amount and known fees.
- **REQ-039:** Missing rates and unsupported assets never become zero amounts or assumed USD/USDT/USDC parity.
- **REQ-059:** Money calculations use exact arithmetic and explicit boundary rounding rules.

### Acceptance criteria

A criterion link establishes coverage; research or a partial task does not prove the entire product criterion. This task's exact outcome is specified in verification below.

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

- **Given:** A source contains unsupported USDC.E; USDT/USD and USDC/USD rates are unavailable.
- **When:** A total valuation is built.
- **Then:** Raw data is retained and valuation coverage is incomplete; no hidden zero or automatic 1:1 rate is used. USDC.E is not merged into USDC by symbol similarity.
- **Level:** `integration`.

#### AC-074

- **Given:** A RUB amount is known but its historical BTC cross-rate is missing.
- **When:** The owner selects BTC reporting.
- **Then:** Native actuals remain available; equivalent/total is marked incomplete; a current rate is not substituted for the historical rate.
- **Level:** `unit+end-to-end`.

### Verification

```sh
make docs-check
```

RU/EN evidence aligns on FX-E01–FX-E18 and FX-B01–FX-B06; selected sources, dates, Decimal crosses and unavailable behavior are reproducible. No actual rates/keys/cookies/IP are published. Research closure is not presented as passing application ACs or BLK-07/Ready.

make docs-check validates SDD. Anonymous current/history/plan-limit and DE/NL/BG probes establish dated public access only. Demo key, long-running runtime, >365-day crypto history and executable provider quotes remain untested and are listed as open decisions.

### Handoff to the next agent

Record changed contracts, commands/results, limitations, unresolved questions and unblocked dependencies. Update both languages and traceability. Close the task only with evidence of its outcome; GitHub Closed alone does not mean the MVP is Ready.

**Commit boundary:** this task's logical boundary; this card does not authorize commit/push/deploy, which require current user authorization.
