<!-- want-keep-task: task-6.2 -->
# task-6.2 — Учитывать кредитки и грейс-период / Account for credit cards and grace periods

## RU

Показывать долг и условия сохранения льготы по подтверждённым данным.

**Состояние:** Заблокировано зависимостями и проверкой SDD Ready; реализация не начата.

**Зависимости:** `task-4.1`, `task-4.2`, `task-4.3`, `task-2.2`.

**Тип:** `implementation`.

### Изменение и контракты

Маппить statement cycle, задолженность, own funds, minimum/due date и grace eligibility из проверенных контрактов конкретных карт. Доменные правила должны учитывать исключения, частичные платежи и изменение условий. Не строить сроки из рекламных текстов или универсального graceDays. Missing terms дают unknown; purchase/repayment/interest/fee имеют различную семантику.

### Границы изменений

- `backend/internal/credit/`

Это планируемые пути. Общие контракты: `spec/001-want-keep-mvp/contracts.md`; архитектура и команды: `constraints.md`. Менять только владельца поведения и затронутые тесты; при незакрытом контракте обновить evidence и остановить зависимую реализацию.

### Связанные требования

- **REQ-005:** Счета показывают собственные, доступные, заблокированные и заёмные средства в пределах данных источника.
- **REQ-031:** Кредитные карты показывают задолженность, собственные средства, лимит, минимальный платёж и дату по данным источника.
- **REQ-032:** Грейс-период опирается на условия конкретной карты и показывает сумму и срок сохранения льготы.
- **REQ-033:** Накопления показывают фактические начисления и прогноз по ставкам, срокам, капитализации и денежным потокам.
- **REQ-039:** Отсутствующие курсы и неподдерживаемые активы не превращаются в нулевые суммы или условный паритет USDT/USD.
- **REQ-048:** Интеграции и браузерный сборщик выполняют только разрешённые операции чтения.

### Критерии приёмки

Связь с критерием задаёт покрытие; исследование или частичная задача не доказывает весь критерий продукта. Точный результат этой задачи указан ниже в проверке.

#### AC-005

- **Дано:** Источник сообщает собственные RUB 100, долг RUB 300 и кредитный лимит RUB 1 000.
- **Когда:** Строится сводка денег.
- **Тогда:** Кредитный лимит не увеличивает собственный капитал или доступный бюджет; отсутствующее поле отображается как неизвестное.
- **Уровень:** `integration`.

#### AC-031

- **Дано:** Покупка RUB 1 000 сделана с кредитки, затем долг погашен с собственного счёта.
- **Когда:** Формируется бюджет и сводка кредитки.
- **Тогда:** Покупка учтена один раз; погашение не второй расход; проценты и комиссии — отдельные расходы; неизвестный минимальный платёж не вычисляется догадкой.
- **Уровень:** `integration`.

#### AC-032

- **Дано:** Для карты подтверждены условия, выписка, исключения и крайняя дата.
- **Когда:** Совершаются покупка, частичное погашение и операция, исключённая из льготы.
- **Тогда:** Сумма и срок согласованы с подтверждёнными условиями; при нехватке условий отображается неизвестность, а не обещание сохранения льготы.
- **Уровень:** `contract`.

#### AC-048

- **Дано:** Сборщик имеет сессию личного кабинета с более широкими внешними правами.
- **Когда:** Возникают запрос на платёж, неподтверждённый маршрут или MFA/CAPTCHA.
- **Тогда:** Платёж и неизвестный маршрут блокируются; MFA/CAPTCHA передаётся владельцу, источник приостанавливается; остальные источники продолжают работать.
- **Уровень:** `integration`.

#### AC-070

- **Дано:** Банк передаёт баланс, но не условия грейса; ставка Earn имеет неизвестную базу начисления.
- **Когда:** Открываются прогнозы.
- **Тогда:** Баланс отображается; льгота и точный прогноз имеют причину недоступности; AI не извлекает гарантированную бизнес-логику из рекламной формулировки.
- **Уровень:** `contract+end-to-end`.

### Проверка результата

```sh
make test-go PKG=./internal/credit/... && make test-contract PROVIDER=credit
```

Синтетические контракты каждого банка покрывают грейс/исключения/частичное погашение; нет обещания льготы при неизвестных условиях.

Команды `make` — будущий контракт, создаваемый task-1.1; сейчас они не существуют. Live/paid/manual проверки отдельно фиксируют доступ и фактический результат. Исследования не обходят блокер отсутствующего доступа.

### Передача следующему агенту

Записать изменённые контракты, команды и результаты, ограничения, незакрытые вопросы и разблокированные зависимости. Обновить обе языковые версии и трассировку. Закрывать задачу только по доказательству её результата; GitHub Closed само по себе не означает Ready MVP.

**Commit boundary:** логическая граница этой задачи; commit/push/deploy не разрешены данной карточкой и требуют действующей авторизации пользователя.

## EN

Show debt and grace eligibility from confirmed data.

**Status:** Blocked by dependencies and the SDD Ready gate; implementation has not started.

**Dependencies:** `task-4.1`, `task-4.2`, `task-4.3`, `task-2.2`.

**Kind:** `implementation`.

### Change and contracts

Map statement cycles, debt, own funds, minimum/due dates and grace eligibility from verified card-specific contracts. Domain rules handle exclusions, partial payments and term changes. Do not derive deadlines from marketing text or a universal graceDays value. Missing terms yield unknown; purchases/repayments/interest/fees retain distinct semantics.

### Change boundaries

- `backend/internal/credit/`

These are planned paths. Shared contracts: `spec/001-want-keep-mvp/contracts.en.md`; architecture and commands: `constraints.en.md`. Change only the behavior owner and affected tests; an unresolved contract requires updated evidence and stops dependent implementation.

### Linked requirements

- **REQ-005:** Accounts distinguish owned, available, locked and borrowed amounts where the source provides them.
- **REQ-031:** Credit cards show debt, own funds, credit limit, minimum payment and due date from source data.
- **REQ-032:** Grace-period tracking uses the specific card's terms and shows the amount and deadline needed to preserve the benefit.
- **REQ-033:** Savings show actual accruals and forecasts using rates, terms, compounding and cash flows.
- **REQ-039:** Missing rates and unsupported assets never become zero amounts or an assumed USDT/USD peg.
- **REQ-048:** Integrations and the browser collector perform authorized read operations only.

### Acceptance criteria

A criterion link establishes coverage; research or a partial task does not prove the entire product criterion. This task's exact outcome is specified in verification below.

#### AC-005

- **Given:** The source reports RUB 100 owned, RUB 300 debt and a RUB 1,000 credit limit.
- **When:** A money summary is built.
- **Then:** The credit limit does not increase net worth or the spendable budget; missing fields are shown as unknown.
- **Level:** `integration`.

#### AC-031

- **Given:** A RUB 1,000 credit-card purchase is followed by repayment from an owned account.
- **When:** The budget and card summary are built.
- **Then:** The purchase is counted once; repayment is not another expense; interest and fees are separate expenses; an unknown minimum payment is not guessed.
- **Level:** `integration`.

#### AC-032

- **Given:** Card terms, statement, exclusions and deadline are confirmed.
- **When:** A purchase, partial repayment and grace-excluded transaction occur.
- **Then:** Amount and deadline follow confirmed terms; missing terms produce an unknown state rather than a promise of grace eligibility.
- **Level:** `contract`.

#### AC-048

- **Given:** The collector has a personal-account session with broader provider permissions.
- **When:** A payment request, unapproved route or MFA/CAPTCHA appears.
- **Then:** Payments and unknown routes are blocked; MFA/CAPTCHA is handed to the owner and that source pauses; other sources continue.
- **Level:** `integration`.

#### AC-070

- **Given:** A bank exposes balance but no grace terms; an Earn rate has an unknown accrual basis.
- **When:** Forecasts are opened.
- **Then:** Balance is shown; grace eligibility and exact forecasts explain unavailability; AI does not turn marketing wording into guaranteed business rules.
- **Level:** `contract+end-to-end`.

### Verification

```sh
make test-go PKG=./internal/credit/... && make test-contract PROVIDER=credit
```

Each bank's synthetic contracts cover grace/exclusions/partial repayment; unknown terms never promise eligibility.

The `make` commands are a future contract established by task-1.1; they do not exist yet. Live/paid/manual checks separately record access and actual outcomes. Research does not bypass missing-access blockers.

### Handoff to the next agent

Record changed contracts, commands/results, limitations, unresolved questions and unblocked dependencies. Update both languages and traceability. Close the task only with evidence of its outcome; GitHub Closed alone does not mean the MVP is Ready.

**Commit boundary:** this task's logical boundary; this card does not authorize commit/push/deploy, which require current user authorization.
