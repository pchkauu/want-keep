<!-- want-keep-task: task-2.1 -->
# task-2.1 — Реализовать счета и начальные остатки / Implement accounts and opening balances

## RU

Учесть собственные счета, наличные и начальную точку истории.

**Состояние:** Не начато; задача ожидает собственные зависимости и entry gates.

**Зависимости:** `task-1.3`, `task-1.6`.

**Тип:** `implementation`.

### Изменение и контракты

Разделить логический аккаунт платформы, счёт актива и карту-доступ к счёту, чтобы карты не дублировали баланс. Поддержать ручные наличные счета, выбранную дату начала, явный начальный остаток и снимки источника с owned/available/locked/debt. Неизвестные значения сохранять неизвестными, неподдерживаемые активы — в покрытии источника.

### Границы изменений

- `backend/internal/accounts/`

Это планируемые пути. Общие контракты: `spec/001-want-keep-mvp/contracts.md`; архитектура и команды: `constraints.md`. Менять только владельца поведения и затронутые тесты; при незакрытом контракте обновить evidence и остановить зависимую реализацию.

### Связанные требования

- **REQ-002:** Учёт поддерживает RUB, USD, USDT, USDC, BTC и ETH; наличные, банковские деньги и платформенные кошельки различаются счетами.
- **REQ-004:** Начало учёта задаётся датой; начальные остатки отделены от доходов и расходов.
- **REQ-005:** Счета показывают собственные, доступные, заблокированные и заёмные средства в пределах данных источника.
- **REQ-039:** Отсутствующие курсы и неподдерживаемые активы не превращаются в нулевые суммы или условный паритет USD/USDT/USDC.
- **REQ-065:** Принадлежность счёта, владелец внешнего аккаунта, автор записи и принадлежность расхода являются отдельными признаками.
- **REQ-066:** Все доходы и доступные средства входят в семейный пул; общий бюджет и личные разрезы используют один финансовый факт.
- **REQ-076:** Семейная область проверяется для API, файлов, AI, фоновых задач и внешних ID независимо от присланных actor/owner.

### Критерии приёмки

Связь с критерием задаёт покрытие; исследование или частичная задача не доказывает весь критерий продукта. Точный результат этой задачи указан ниже в проверке.

#### AC-002

- **Дано:** Созданы RUB наличные 1 000, RUB банк 2 000, USD наличные 10, USDT 20, USDC 12.000000000123, BTC 0.001 и ETH 0.001234567891.
- **Когда:** Владелец открывает счета.
- **Тогда:** Показаны семь отдельных счетов с исходными активами и точными остатками; дробные остатки не обрезаются до точности UI или заказа провайдера. RUB суммируется только в соответствующем срезе.
- **Уровень:** `integration`.

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

#### AC-039

- **Дано:** В источнике есть неподдерживаемый USDC.E; для USDT/USD и USDC/USD отсутствуют курсы.
- **Когда:** Строится общая оценка.
- **Тогда:** Исходные данные сохранены, покрытие оценки обозначено неполным; нет скрытого нуля или автоматического курса 1:1. USDC.E не объединён с USDC по похожему символу.
- **Уровень:** `integration`.

#### AC-079

- **Дано:** A и B имеют разные аккаунты одного провайдера и общий счёт; B заносит покупку A со счёта B.
- **Когда:** Выполняются ввод, импорт обоих аккаунтов и повторное подключение того же внешнего аккаунта.
- **Тогда:** Разные аккаунты не сливаются; повторный источник не удваивает остатки. Плательщик, автор и получатель расхода сохраняются независимо. Неустановленное совпадение блокирует новый учёт до уточнения.
- **Уровень:** `integration`.

#### AC-080

- **Дано:** Зарплата поступила на счёт A, общая аренда оплачена B, у A нет доступного остатка.
- **Когда:** Строятся семейный бюджет, персональные расходы и обеспеченность по валютам.
- **Тогда:** Доход общий с сохранением получателя; аренда учтена в семье один раз и в личных видах по долям. Доступность семьи включает средства обоих без автоматического обмена валют и без кредитного лимита.
- **Уровень:** `integration`.

#### AC-090

- **Дано:** В тестах созданы две изолированные семьи; запрос или задача подменяет householdId/actor/resourceId.
- **Когда:** Проверяются чтение файла, импорт, исправление, поиск AI и дедупликация.
- **Тогда:** Чужие объекты недоступны и не объединяются; сервер берёт principal из сессии или проверенного контекста задания. Отказ не раскрывает чужое содержимое.
- **Уровень:** `integration`.

#### AC-105

- **Дано:** Есть личный счёт A, семейный счёт и запись покупки; B видит их и может исправлять учёт.
- **Когда:** B исправляет покупку и пытается через FORM-03/API изменить владельца или scope счёта A; затем A меняет свой счёт и B меняет семейный.
- **Тогда:** Исправление покупки разрешено, изменение принадлежности личного счёта B отклоняется сервером. Действия текущего владельца и изменение семейного счёта разрешены с revision/audit. Внешний владелец и история операций не изменены, фильтр участника не даёт дополнительных прав.
- **Уровень:** `integration+e2e`.

### Проверка результата

```sh
make test-go PKG=./internal/accounts/...
```

Пять базовых счетов, карточные алиасы, долг и начальные остатки считаются без ложного дохода или двойного капитала.

Команды `make` — будущий контракт, создаваемый task-1.1; сейчас они не существуют. Live/paid/manual проверки отдельно фиксируют доступ и фактический результат. Исследования не обходят блокер отсутствующего доступа.

### Передача следующему агенту

Записать изменённые контракты, команды и результаты, ограничения, незакрытые вопросы и разблокированные зависимости. Обновить обе языковые версии и трассировку. Закрывать задачу только по доказательству её результата; GitHub Closed само по себе не означает Ready MVP.

**Commit boundary:** логическая граница этой задачи; commit/push/deploy не разрешены данной карточкой и требуют действующей авторизации пользователя.

## EN

Account for owned accounts, cash and the history opening point.

**Status:** Not started; the task awaits its own dependencies and entry gates.

**Dependencies:** `task-1.3`, `task-1.6`.

**Kind:** `implementation`.

### Change and contracts

Separate platform identity, asset account and card-as-access-to-account so cards do not duplicate balances. Support manual cash accounts, selected start date, explicit opening balance and source snapshots with owned/available/locked/debt. Preserve unknown values and track unsupported assets in source coverage.

### Change boundaries

- `backend/internal/accounts/`

These are planned paths. Shared contracts: `spec/001-want-keep-mvp/contracts.en.md`; architecture and commands: `constraints.en.md`. Change only the behavior owner and affected tests; an unresolved contract requires updated evidence and stops dependent implementation.

### Linked requirements

- **REQ-002:** Accounting supports RUB, USD, USDT, USDC, BTC and ETH; cash, bank money and platform wallets are separate accounts.
- **REQ-004:** Accounting starts on a selected date; opening balances are separate from income and expenses.
- **REQ-005:** Accounts distinguish owned, available, locked and borrowed amounts where the source provides them.
- **REQ-039:** Missing rates and unsupported assets never become zero amounts or assumed USD/USDT/USDC parity.
- **REQ-065:** Account ownership, external-account owner, record author and expense attribution are distinct dimensions.
- **REQ-066:** All income and available funds enter the household pool; household and individual budget views share one financial fact.
- **REQ-076:** Household scope is checked for APIs, files, AI, jobs and external IDs independently of supplied actor/owner fields.

### Acceptance criteria

A criterion link establishes coverage; research or a partial task does not prove the entire product criterion. This task's exact outcome is specified in verification below.

#### AC-002

- **Given:** Accounts contain RUB cash 1,000, RUB bank 2,000, USD cash 10, USDT 20, USDC 12.000000000123, BTC 0.001 and ETH 0.001234567891.
- **When:** The owner opens accounts.
- **Then:** Seven distinct accounts show original assets and exact balances; residuals are not truncated to UI or provider order precision. RUB is combined only in the relevant aggregate.
- **Level:** `integration`.

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

#### AC-039

- **Given:** A source contains unsupported USDC.E; USDT/USD and USDC/USD rates are unavailable.
- **When:** A total valuation is built.
- **Then:** Raw data is retained and valuation coverage is incomplete; no hidden zero or automatic 1:1 rate is used. USDC.E is not merged into USDC by symbol similarity.
- **Level:** `integration`.

#### AC-079

- **Given:** A and B have separate accounts at one provider and a joint account; B enters A’s purchase paid from B’s account.
- **When:** Entry, import of both accounts and reconnection of the same external account run.
- **Then:** Distinct accounts are not merged; a repeated source does not double balances. Payer, author and expense beneficiary remain independent. Unresolved source identity blocks new posting pending clarification.
- **Level:** `integration`.

#### AC-080

- **Given:** Salary arrived in A’s account, B paid joint rent and A has no available balance.
- **When:** The household budget, individual expenses and currency funding are calculated.
- **Then:** Income is pooled with recipient retained; rent appears once for the household and by shares in individual views. Household availability includes both members’ funds without automatic currency exchange or credit limits.
- **Level:** `integration`.

#### AC-090

- **Given:** Tests contain two isolated households; a request or job forges householdId/actor/resourceId.
- **When:** File reads, import, correction, AI retrieval and deduplication are exercised.
- **Then:** Foreign objects are inaccessible and never merged; the server takes principal from the session or validated job context. Denial reveals no foreign content.
- **Level:** `integration`.

#### AC-105

- **Given:** There is A’s personal account, a household account and a purchase; B can read them and correct accounting.
- **When:** B corrects the purchase and uses FORM-03/API to change A’s account owner/scope; A then changes their own account and B changes the household account.
- **Then:** Purchase correction succeeds; B’s personal-account ownership change is rejected server-side. Current-owner actions and household-account changes succeed with revision/audit. External ownership and transaction history remain unchanged; member filter grants no extra authority.
- **Level:** `integration+e2e`.

### Verification

```sh
make test-go PKG=./internal/accounts/...
```

Five baseline accounts, card aliases, debt and opening balances produce no false income or duplicated wealth.

The `make` commands are a future contract established by task-1.1; they do not exist yet. Live/paid/manual checks separately record access and actual outcomes. Research does not bypass missing-access blockers.

### Handoff to the next agent

Record changed contracts, commands/results, limitations, unresolved questions and unblocked dependencies. Update both languages and traceability. Close the task only with evidence of its outcome; GitHub Closed alone does not mean the MVP is Ready.

**Commit boundary:** this task's logical boundary; this card does not authorize commit/push/deploy, which require current user authorization.
