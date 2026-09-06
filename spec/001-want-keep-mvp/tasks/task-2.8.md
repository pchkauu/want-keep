<!-- want-keep-task: task-2.8 -->
# task-2.8 — Распределять семейные расходы и позиции по участникам / Allocate household expenses and items to members

## RU

Распределять семейные расходы и позиции по участникам.

**Состояние:** Заблокировано зависимостями и проверкой SDD Ready; реализация не начата.

**Зависимости:** `task-2.6`, `task-1.6`.

**Тип:** `implementation`.

### Изменение и контракты

Разделить payer/account owner/actor/beneficiary; распределения позиций личные или общие. Приоритет: явное распределение позиции/покупки, правило статьи, 50/50; неизвестное назначение требует уточнения. Хранить revision распределения, точную сумму и детерминированное округление. Правила по продавцу/категории применяются к новым фактам, не переписывают историю.

### Границы изменений

- `backend/internal/ledger/`
- `backend/internal/categories/`
- `backend/internal/household/`
- `api/openapi.yaml`

Это планируемые пути. Общие контракты: `spec/001-want-keep-mvp/contracts.md`; архитектура и команды: `constraints.md`. Менять только владельца поведения и затронутые тесты; при незакрытом контракте обновить evidence и остановить зависимую реализацию.

### Связанные требования

- **REQ-008:** Повторные импорты, чек и запись чата объединяют доказательства одной операции без повторного учёта.
- **REQ-010:** Возврат уменьшает расходы исходного месяца покупки, сохраняя дату реального поступления денег.
- **REQ-016:** Позиции чека распределяют одну оплаченную сумму по категориям без дублирования итога.
- **REQ-037:** Исторические расходы используют зафиксированную оценку на дату операции, текущий капитал — актуальную оценку.
- **REQ-064:** Оба участника видят все финансовые данные и изменяют операции; личные цели и части плана изменяет только их владелец.
- **REQ-065:** Принадлежность счёта, владелец внешнего аккаунта, автор записи и принадлежность расхода являются отдельными признаками.
- **REQ-066:** Все доходы и доступные средства входят в семейный пул; общий бюджет и личные разрезы используют один финансовый факт.
- **REQ-067:** Расходы и позиции чеков имеют личное или совместное назначение; общая доля по умолчанию 50/50 с исключениями статьи или покупки.
- **REQ-071:** Один общий чат сохраняет автора сообщения и проверяет полномочия инициатора AI-команды при исполнении.
- **REQ-072:** Конкурирующие изменения, ответы на уточнения и отмены проверяют версию и текущие права, сохраняя обоих авторов.

### Критерии приёмки

Связь с критерием задаёт покрытие; исследование или частичная задача не доказывает весь критерий продукта. Точный результат этой задачи указан ниже в проверке.

#### AC-065

- **Дано:** Покупка USD 10 распределена по двум категориям и оценена в RUB 900.
- **Когда:** Позже возвращено USD 4 за известную позицию; текущий курс иной.
- **Тогда:** Историческая категория уменьшается на исходную стоимость возвращённой части RUB 360; реальные поступления и валютная разница сохраняются отдельно; превышение суммы возвратов блокируется.
- **Уровень:** `unit+integration`.

#### AC-078

- **Дано:** У A есть личная цель и статья плана; у семьи общая статья и операции обоих.
- **Когда:** B читает все данные, исправляет операцию A и общий план, затем пытается изменить личную цель/план A через API и AI.
- **Тогда:** Чтение, операции и общее изменение разрешены; личные план/цель A защищены сервером. Одного уполномоченного подтверждения достаточно, второй уведомлён.
- **Уровень:** `end-to-end`.

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

#### AC-081

- **Дано:** Чек RUB 1000 содержит общие продукты 600 и личные покупки A 100 и B 300.
- **Когда:** Чек заносит любой участник; AI применяет правила или уточняет неизвестное назначение.
- **Тогда:** Факт семьи 1000, A 400, B 600; доли суммируются точно. Исключение покупки приоритетнее статьи, затем 50/50; неоднозначная трата сохранена без вымышленной принадлежности.
- **Уровень:** `integration`.

#### AC-086

- **Дано:** A и B открыли одну версию операции или уточнения.
- **Когда:** Оба отправляют несовместимые изменения и повторяют один запрос.
- **Тогда:** Один результат применяется; второй получает конфликт с необходимостью перечитать состояние. Повтор не дублирует эффект; отмена создаёт новую проверенную revision и не стирает чужую последующую правку.
- **Уровень:** `integration`.

#### AC-091

- **Дано:** Январский чек: общие товары 600 (50/50), личные A 100, B 300; в феврале доли статьи стали 60/40.
- **Когда:** В феврале возвращаются общие товары на 200.
- **Тогда:** Январь уменьшается семье на 200, каждому на 100 с исходной исторической оценкой; февральская пропорция не меняет январь, cash date возврата остаётся февральской.
- **Уровень:** `integration`.

#### AC-093

- **Дано:** A и B присылают один и тот же чек с одним счётом; затем приходит банковская операция.
- **Когда:** Обрабатываются параллельные сообщения, повтор файла и импорт.
- **Тогда:** Создаётся один денежный эффект и несколько evidence с авторами; при отсутствии доказанного совпадения требуется уточнение, одинаковые суммы разных счетов не сливаются.
- **Уровень:** `integration`.

### Проверка результата

```sh
make test-go PKG=./internal/ledger/...
make test-integration AREA=family-allocation
```

Смешанный чек 1000 даёт 400/600 без дублей; перераспределение версионно, дробные доли сохраняют итог; контракт исторического возврата передан task-2.7.

Команды `make` — будущий контракт, создаваемый task-1.1; сейчас они не существуют. Live/paid/manual проверки отдельно фиксируют доступ и фактический результат. Исследования не обходят блокер отсутствующего доступа.

### Передача следующему агенту

Записать изменённые контракты, команды и результаты, ограничения, незакрытые вопросы и разблокированные зависимости. Обновить обе языковые версии и трассировку. Закрывать задачу только по доказательству её результата; GitHub Closed само по себе не означает Ready MVP.

**Commit boundary:** логическая граница этой задачи; commit/push/deploy не разрешены данной карточкой и требуют действующей авторизации пользователя.

## EN

Allocate household expenses and items to members.

**Status:** Blocked by dependencies and the SDD Ready gate; implementation has not started.

**Dependencies:** `task-2.6`, `task-1.6`.

**Kind:** `implementation`.

### Change and contracts

Separate payer/account owner/actor/beneficiary; item allocations are personal or joint. Precedence: explicit item/purchase allocation, plan-line rule, 50/50; unknown attribution requires clarification. Retain allocation revision, exact total and deterministic rounding. Merchant/category rules apply to new facts and do not rewrite history.

### Change boundaries

- `backend/internal/ledger/`
- `backend/internal/categories/`
- `backend/internal/household/`
- `api/openapi.yaml`

These are planned paths. Shared contracts: `spec/001-want-keep-mvp/contracts.en.md`; architecture and commands: `constraints.en.md`. Change only the behavior owner and affected tests; an unresolved contract requires updated evidence and stops dependent implementation.

### Linked requirements

- **REQ-008:** Repeated imports, receipts and chat entries combine evidence of one transaction without double counting.
- **REQ-010:** A refund reduces expenses in the purchase month while preserving the actual cash receipt date.
- **REQ-016:** Receipt items allocate one paid amount across categories without duplicating the total.
- **REQ-037:** Historical expenses use a fixed transaction-date valuation; current wealth uses a current valuation.
- **REQ-064:** Both members see all financial data and edit transactions; only the owner edits personal goals and plan portions.
- **REQ-065:** Account ownership, external-account owner, record author and expense attribution are distinct dimensions.
- **REQ-066:** All income and available funds enter the household pool; household and individual budget views share one financial fact.
- **REQ-067:** Expenses and receipt items have personal or joint attribution; joint shares default to 50/50 with line or purchase overrides.
- **REQ-071:** One shared chat retains message authors and checks the AI command initiator’s authority at execution.
- **REQ-072:** Competing edits, clarification answers and reversals check revision and current permissions while retaining both authors.

### Acceptance criteria

A criterion link establishes coverage; research or a partial task does not prove the entire product criterion. This task's exact outcome is specified in verification below.

#### AC-065

- **Given:** A USD 10 purchase is split across two categories and valued at RUB 900.
- **When:** USD 4 is later refunded for a known item at a different current rate.
- **Then:** The historical category decreases by the refunded original value RUB 360; actual cash receipts and FX difference stay separate; excess cumulative refunds are rejected.
- **Level:** `unit+integration`.

#### AC-078

- **Given:** A has a personal goal and plan line; the household has a joint line and both members’ transactions.
- **When:** B reads all data, edits A’s transaction and the joint plan, then attempts to change A’s personal goal/plan through API and AI.
- **Then:** Reads, transaction edits and joint changes succeed; A’s personal plan/goal are protected server-side. One authorized confirmation suffices and the other member is notified.
- **Level:** `end-to-end`.

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

#### AC-081

- **Given:** A RUB 1,000 receipt contains joint groceries of 600 and personal purchases of A 100 and B 300.
- **When:** Either member enters the receipt; AI applies rules or clarifies unknown attribution.
- **Then:** Household actual is 1,000, A 400, B 600; shares sum exactly. Purchase override takes precedence over plan line, then 50/50; ambiguous spending persists without invented attribution.
- **Level:** `integration`.

#### AC-086

- **Given:** A and B opened the same transaction or clarification revision.
- **When:** Both submit conflicting edits and replay one request.
- **Then:** One result applies; the other receives a conflict requiring refresh. Replay does not duplicate effects; reversal creates a checked new revision without erasing the other member’s later edit.
- **Level:** `integration`.

#### AC-091

- **Given:** January receipt: joint items 600 (50/50), personal A 100, B 300; February plan shares became 60/40.
- **When:** Joint items worth 200 are returned in February.
- **Then:** January falls by 200 for the household and 100 for each member using original historical valuation; February shares do not rewrite January and refund cash date stays in February.
- **Level:** `integration`.

#### AC-093

- **Given:** A and B submit the same receipt for one account; the bank transaction arrives later.
- **When:** Parallel messages, a file replay and import are processed.
- **Then:** One financial effect and multiple authored evidence records result; an unproven match requires clarification and equal amounts from different accounts are not merged.
- **Level:** `integration`.

### Verification

```sh
make test-go PKG=./internal/ledger/...
make test-integration AREA=family-allocation
```

A mixed 1,000 receipt yields 400/600 without duplicates; reallocation is versioned, fractional shares preserve total and the historical-refund contract is handed to task-2.7.

The `make` commands are a future contract established by task-1.1; they do not exist yet. Live/paid/manual checks separately record access and actual outcomes. Research does not bypass missing-access blockers.

### Handoff to the next agent

Record changed contracts, commands/results, limitations, unresolved questions and unblocked dependencies. Update both languages and traceability. Close the task only with evidence of its outcome; GitHub Closed alone does not mean the MVP is Ready.

**Commit boundary:** this task's logical boundary; this card does not authorize commit/push/deploy, which require current user authorization.
