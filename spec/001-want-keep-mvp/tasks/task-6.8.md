<!-- want-keep-task: task-6.8 -->
# task-6.8 — Считать дневные лимиты и прогноз ликвидности / Calculate daily allowances and liquidity forecast

## RU

Дать объяснимую доступную и прогнозную сумму по каждой валюте.

**Состояние:** Не начато; задача ожидает собственные зависимости и entry gates.

**Зависимости:** `task-6.6`, `task-6.7`, `task-6.3`, `task-2.5`.

**Тип:** `implementation`.

### Изменение и контракты

Применить формулы и day-by-day forecast из contracts.md: native spendable owned funds, обязательства/минимальные платежи, unique goals reserve, flexible remaining plan и даты доходов. Не включать чужую валюту, кредитный лимит, lock и неполученный доход в available. Категорийные лимиты делят общий предел; no-AI/uncategorized и incomplete sources явно влияют на покрытие. Рассчитывать K один раз для семьи в валюте; распределять его по положительным остаткам ячеек участник×категория. Персональные и категорийные итоги — маргинальные суммы одной матрицы, не независимое размножение K; общий доход и цели не удваиваются.

### Границы изменений

- `backend/internal/forecast/`

Это планируемые пути. Общие контракты: `spec/001-want-keep-mvp/contracts.md`; архитектура и команды: `constraints.md`. Менять только владельца поведения и затронутые тесты; при незакрытом контракте обновить evidence и остановить зависимую реализацию.

### Связанные требования

- **REQ-005:** Счета показывают собственные, доступные, заблокированные и заёмные средства в пределах данных источника.
- **REQ-024:** План поддерживает обязательные расходы по датам и повторяемые платежи.
- **REQ-025:** Гибкие категории ограничивают траты за месяц и показывают остаток и перерасход.
- **REQ-026:** Прогнозируемые доходы имеют сумму, валюту, дату и отдельное состояние исполнения.
- **REQ-028:** Личная или совместная цель содержит сумму, валюту, срок и способ накопления: явный резерв либо выделенный счёт.
- **REQ-029:** Одни средства нельзя одновременно зарезервировать на несколько целей или повторно учесть через выделенный счёт.
- **REQ-030:** Дневные лимиты показывают семейный и индивидуальный доступный/прогнозный остаток, по категориям и с отдельным обеспечением каждой валютой.
- **REQ-031:** Кредитные карты показывают задолженность, собственные средства, лимит, минимальный платёж и дату по данным источника.
- **REQ-039:** Отсутствующие курсы и неподдерживаемые активы не превращаются в нулевые суммы или условный паритет USD/USDT/USDC.
- **REQ-064:** Оба участника видят все финансовые данные и изменяют операции; личные цели и части плана изменяет только их владелец.
- **REQ-066:** Все доходы и доступные средства входят в семейный пул; общий бюджет и личные разрезы используют один финансовый факт.
- **REQ-069:** Резервы личных и совместных целей задаются явно; совместные цели отображаются отдельным общим блоком без персональных долей.
- **REQ-070:** Сумма индивидуальных дневных лимитов не превышает семейный предел одной валюты; счёт плательщика не меняет долю расходов.
- **REQ-072:** Конкурирующие изменения, ответы на уточнения и отмены проверяют версию и текущие права, сохраняя обоих авторов.

### Критерии приёмки

Связь с критерием задаёт покрытие; исследование или частичная задача не доказывает весь критерий продукта. Точный результат этой задачи указан ниже в проверке.

#### AC-005

- **Дано:** Источник сообщает собственные RUB 100, долг RUB 300 и кредитный лимит RUB 1 000.
- **Когда:** Строится сводка денег.
- **Тогда:** Кредитный лимит не увеличивает собственный капитал или доступный бюджет; отсутствующее поле отображается как неизвестное.
- **Уровень:** `integration`.

#### AC-024

- **Дано:** Аренда запланирована на 5-е число, подписка повторяется ежемесячно.
- **Когда:** Наступает дата платежа и импортируется фактическое списание.
- **Тогда:** Плановая строка сама не создаёт расход; сопоставленный факт погашает обязательство без двойного резервирования.
- **Уровень:** `integration`.

#### AC-025

- **Дано:** На еду выделено RUB 6 000, потрачено RUB 6 200.
- **Когда:** Открывается бюджет.
- **Тогда:** Показан перерасход RUB 200; факт не скрыт и лимит другой категории не изменён автоматически.
- **Уровень:** `unit`.

#### AC-026

- **Дано:** Зарплата RUB 50 000 ожидается 20-го числа.
- **Когда:** Доход ещё не поступил, затем приходит RUB 45 000.
- **Тогда:** До поступления это только прогноз; факт RUB 45 000 сопоставляется отдельно, разница RUB 5 000 не становится доходом.
- **Уровень:** `integration`.

#### AC-029

- **Дано:** На счёте USD 100 уже зарезервировано USD 80.
- **Когда:** Вторая цель запрашивает USD 30 либо тот же резерв дублируется ссылкой на счёт.
- **Тогда:** Операция превышения отклоняется атомарно; свободно USD 20; параллельные запросы не обходят ограничение.
- **Уровень:** `integration`.

#### AC-030

- **Дано:** Есть RUB-бюджет, будущая зарплата, обязательный платёж, резерв цели и USDT на другом счёте.
- **Когда:** Рассчитываются лимиты на оставшиеся дни месяца.
- **Тогда:** Доступный RUB-лимит исключает будущую зарплату, USDT, долг и резервы; прогноз учитывает даты поступлений и показывает кассовые разрывы; общий предел не размножается по категориям.
- **Уровень:** `integration`.

#### AC-039

- **Дано:** В источнике есть неподдерживаемый USDC.E; для USDT/USD и USDC/USD отсутствуют курсы.
- **Когда:** Строится общая оценка.
- **Тогда:** Исходные данные сохранены, покрытие оценки обозначено неполным; нет скрытого нуля или автоматического курса 1:1. USDC.E не объединён с USDC по похожему символу.
- **Уровень:** `integration`.

#### AC-066

- **Дано:** До аренды нет достаточных RUB; будущая зарплата запланирована после даты аренды; есть кредитный лимит.
- **Когда:** Рассчитывается прогноз и доход переносится на более позднюю дату.
- **Тогда:** Показан кассовый разрыв на дату аренды; кредитный лимит не выдаётся за доступные собственные деньги; основной план без решения владельца не меняется.
- **Уровень:** `unit+end-to-end`.

#### AC-067

- **Дано:** USD 100 размещены на выделенном счёте цели; ещё USD 50 на расходном.
- **Когда:** Строятся капитал, прогресс и дневной лимит.
- **Тогда:** Капитал USD 150, прогресс USD 100; доступно к тратам не более USD 50, резерв не вычтен второй раз.
- **Уровень:** `unit`.

#### AC-080

- **Дано:** Зарплата поступила на счёт A, общая аренда оплачена B, у A нет доступного остатка.
- **Когда:** Строятся семейный бюджет, персональные расходы и обеспеченность по валютам.
- **Тогда:** Доход общий с сохранением получателя; аренда учтена в семье один раз и в личных видах по долям. Доступность семьи включает средства обоих без автоматического обмена валют и без кредитного лимита.
- **Уровень:** `integration`.

#### AC-083

- **Дано:** Есть личные цели A и B, общая цель RUB 600000 и виртуальный резерв RUB 10000.
- **Когда:** Оба открывают личные и семейный виды, увеличивают разрешённый резерв и связывают выделенный счёт.
- **Тогда:** Общая цель показана целиком в общем блоке; персональные половины не создаются. Резерв уменьшает семейную доступность один раз; нет автоматического распределения свободных средств на цели.
- **Уровень:** `end-to-end`.

#### AC-084

- **Дано:** Осталось 10 дней, K=RUB 1000, положительные персональные остатки A=3000 и B=1000.
- **Когда:** Рассчитаны доступные лимиты; затем меняется плательщик общей покупки или дата ожидаемого дохода.
- **Тогда:** Семейный лимит 100/день, A 75, B 25; суммы не дублируют K. Плательщик не меняет доли; перенос дохода меняет прогноз, не доступный остаток. Неизвестное назначение не скрывает факт расхода.
- **Уровень:** `integration`.

#### AC-092

- **Дано:** Свободно RUB 1000; оба пытаются зарезервировать по 800 для разрешённых целей.
- **Когда:** Команды исполняются одновременно.
- **Тогда:** Проверка общего доступного остатка и резерв атомарны: проходит максимум одна команда; отказ не уменьшает другой резерв, оба видят актуальный остаток.
- **Уровень:** `integration`.

### Проверка результата

```sh
make test-go PKG=./internal/forecast/...
```

Дефицит до зарплаты, несколько валют, дата оплаты, goal reserve и последняя дата месяца дают объяснимые воспроизводимые лимиты.

Команды `make` — будущий контракт, создаваемый task-1.1; сейчас они не существуют. Live/paid/manual проверки отдельно фиксируют доступ и фактический результат. Исследования не обходят блокер отсутствующего доступа.

### Передача следующему агенту

Записать изменённые контракты, команды и результаты, ограничения, незакрытые вопросы и разблокированные зависимости. Обновить обе языковые версии и трассировку. Закрывать задачу только по доказательству её результата; GitHub Closed само по себе не означает Ready MVP.

**Commit boundary:** логическая граница этой задачи; commit/push/deploy не разрешены данной карточкой и требуют действующей авторизации пользователя.

## EN

Provide explainable available/forecast allowance for each currency.

**Status:** Not started; the task awaits its own dependencies and entry gates.

**Dependencies:** `task-6.6`, `task-6.7`, `task-6.3`, `task-2.5`.

**Kind:** `implementation`.

### Change and contracts

Apply contracts.md formulas/day-by-day forecast using native spendable owned funds, obligations/minimum payments, unique goal reservation, flexible remaining plan and income dates. Available excludes other currencies, credit limits, locks and unreceived income. Category limits share the overall cap; no-AI/uncategorized and incomplete sources affect coverage explicitly. Compute K once per household and currency; allocate it across positive member×category cell remainders. Individual and category totals are margins of one matrix, never independent duplication of K; pooled income and goals are not doubled.

### Change boundaries

- `backend/internal/forecast/`

These are planned paths. Shared contracts: `spec/001-want-keep-mvp/contracts.en.md`; architecture and commands: `constraints.en.md`. Change only the behavior owner and affected tests; an unresolved contract requires updated evidence and stops dependent implementation.

### Linked requirements

- **REQ-005:** Accounts distinguish owned, available, locked and borrowed amounts where the source provides them.
- **REQ-024:** The plan supports dated obligations and recurring payments.
- **REQ-025:** Flexible categories limit monthly spending and show remaining allowance and overspend.
- **REQ-026:** Forecast income has an amount, currency, date and separate fulfillment state.
- **REQ-028:** A personal or joint goal has an amount, currency, deadline and funding mode: an explicit reservation or dedicated account.
- **REQ-029:** The same money cannot be reserved for multiple goals or counted again through a dedicated account.
- **REQ-030:** Daily limits show household and individual available/forecast allowances, by category and with separate funding in each currency.
- **REQ-031:** Credit cards show debt, own funds, credit limit, minimum payment and due date from source data.
- **REQ-039:** Missing rates and unsupported assets never become zero amounts or assumed USD/USDT/USDC parity.
- **REQ-064:** Both members see all financial data and edit transactions; only the owner edits personal goals and plan portions.
- **REQ-066:** All income and available funds enter the household pool; household and individual budget views share one financial fact.
- **REQ-069:** Personal and joint goal reservations are explicit; joint goals appear in a separate shared block without personal shares.
- **REQ-070:** Individual daily allowances sum to no more than the household ceiling in one currency; the payer’s account does not change expense shares.
- **REQ-072:** Competing edits, clarification answers and reversals check revision and current permissions while retaining both authors.

### Acceptance criteria

A criterion link establishes coverage; research or a partial task does not prove the entire product criterion. This task's exact outcome is specified in verification below.

#### AC-005

- **Given:** The source reports RUB 100 owned, RUB 300 debt and a RUB 1,000 credit limit.
- **When:** A money summary is built.
- **Then:** The credit limit does not increase net worth or the spendable budget; missing fields are shown as unknown.
- **Level:** `integration`.

#### AC-024

- **Given:** Rent is planned for the 5th and a subscription recurs monthly.
- **When:** The due date arrives and the actual charge is imported.
- **Then:** The planned row does not itself create an expense; matched actual payment settles the obligation without double reservation.
- **Level:** `integration`.

#### AC-025

- **Given:** Food has a RUB 6,000 allocation and RUB 6,200 spent.
- **When:** The budget is opened.
- **Then:** RUB 200 overspend is visible; actual spending is not hidden and another category's limit is not changed automatically.
- **Level:** `unit`.

#### AC-026

- **Given:** RUB 50,000 salary is expected on the 20th.
- **When:** Income has not arrived, then RUB 45,000 is received.
- **Then:** Before receipt it remains a forecast; RUB 45,000 actual income is matched separately and the RUB 5,000 difference is not booked as income.
- **Level:** `integration`.

#### AC-029

- **Given:** USD 80 of an account's USD 100 is already reserved.
- **When:** A second goal requests USD 30 or an account link duplicates the reservation.
- **Then:** The over-allocation is rejected atomically; USD 20 remains free; concurrent requests cannot bypass the limit.
- **Level:** `integration`.

#### AC-030

- **Given:** There is a RUB budget, future salary, an obligation, a goal reservation and USDT in another account.
- **When:** Limits are calculated for the remaining days of the month.
- **Then:** Available RUB allowance excludes future salary, USDT, debt and reservations; the forecast uses receipt dates and shows cash shortfalls; the overall ceiling is not duplicated across categories.
- **Level:** `integration`.

#### AC-039

- **Given:** A source contains unsupported USDC.E; USDT/USD and USDC/USD rates are unavailable.
- **When:** A total valuation is built.
- **Then:** Raw data is retained and valuation coverage is incomplete; no hidden zero or automatic 1:1 rate is used. USDC.E is not merged into USDC by symbol similarity.
- **Level:** `integration`.

#### AC-066

- **Given:** Available RUB cannot cover rent; salary is scheduled after rent; a credit limit exists.
- **When:** The forecast is calculated and the income date is moved later.
- **Then:** A rent-date cash shortfall is shown; credit is not presented as owned cash and the plan is not changed without an owner decision.
- **Level:** `unit+end-to-end`.

#### AC-067

- **Given:** USD 100 is in a dedicated goal account and USD 50 in a spending account.
- **When:** Wealth, progress and the daily limit are built.
- **Then:** Wealth is USD 150 and progress USD 100; spendable cash is at most USD 50 and the reservation is not deducted twice.
- **Level:** `unit`.

#### AC-080

- **Given:** Salary arrived in A’s account, B paid joint rent and A has no available balance.
- **When:** The household budget, individual expenses and currency funding are calculated.
- **Then:** Income is pooled with recipient retained; rent appears once for the household and by shares in individual views. Household availability includes both members’ funds without automatic currency exchange or credit limits.
- **Level:** `integration`.

#### AC-083

- **Given:** There are personal goals of A and B, a joint RUB 600,000 goal and a RUB 10,000 virtual reserve.
- **When:** Both open individual and household views, increase an authorized reserve and link a dedicated account.
- **Then:** The joint goal appears whole in the shared block; personal halves are not created. The reserve reduces household availability once; free funds are not automatically allocated to goals.
- **Level:** `end-to-end`.

#### AC-084

- **Given:** 10 days remain, K=RUB 1,000 and positive individual remainders are A=3,000 and B=1,000.
- **When:** Available allowances are calculated; then a joint purchase payer or expected-income date changes.
- **Then:** Household allowance is 100/day, A 75, B 25; totals do not duplicate K. Payer does not change shares; rescheduling income changes forecast, not available funds. Unknown attribution does not hide actual expense.
- **Level:** `integration`.

#### AC-092

- **Given:** RUB 1,000 is free; both attempt to reserve 800 for authorized goals.
- **When:** Commands execute concurrently.
- **Then:** Checking household availability and reserving are atomic: at most one command succeeds; rejection does not reduce another reserve and both see current availability.
- **Level:** `integration`.

### Verification

```sh
make test-go PKG=./internal/forecast/...
```

Pre-salary shortfall, multiple currencies, payment dates, goal reservations and month end yield explainable reproducible limits.

The `make` commands are a future contract established by task-1.1; they do not exist yet. Live/paid/manual checks separately record access and actual outcomes. Research does not bypass missing-access blockers.

### Handoff to the next agent

Record changed contracts, commands/results, limitations, unresolved questions and unblocked dependencies. Update both languages and traceability. Close the task only with evidence of its outcome; GitHub Closed alone does not mean the MVP is Ready.

**Commit boundary:** this task's logical boundary; this card does not authorize commit/push/deploy, which require current user authorization.
