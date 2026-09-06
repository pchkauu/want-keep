<!-- want-keep-task: task-2.9 -->
# task-2.9 — Учитывать явные долги и возмещения внутри семьи / Track explicit inter-member debts and reimbursements

## RU

Учитывать явные долги и возмещения внутри семьи.

**Состояние:** Заблокировано зависимостями и проверкой SDD Ready; реализация не начата.

**Зависимости:** `task-2.4`, `task-2.8`.

**Тип:** `implementation`.

### Изменение и контракты

Вести явную запись creditor/debtor/asset/amount/revision и связь с расходом при наличии. Оплата общей доли долг автоматически не создаёт. Погашение связывается с реальным переводом/наличными один раз, допускает частичное погашение; другая валюта требует явно согласованного соответствия сумм. Внутренние требования исключить из семейных активов, доходов и расходов.

### Границы изменений

- `backend/internal/ledger/`
- `backend/internal/household/`
- `api/openapi.yaml`

Это планируемые пути. Общие контракты: `spec/001-want-keep-mvp/contracts.md`; архитектура и команды: `constraints.md`. Менять только владельца поведения и затронутые тесты; при незакрытом контракте обновить evidence и остановить зависимую реализацию.

### Связанные требования

- **REQ-006:** Перевод между счетами семьи, включая счета разных участников, меняет остатки без дохода или расхода по основной сумме.
- **REQ-068:** Взаимный долг учитывается только по явному указанию и не увеличивает активы или расходы семьи.
- **REQ-072:** Конкурирующие изменения, ответы на уточнения и отмены проверяют версию и текущие права, сохраняя обоих авторов.

### Критерии приёмки

Связь с критерием задаёт покрытие; исследование или частичная задача не доказывает весь критерий продукта. Точный результат этой задачи указан ниже в проверке.

#### AC-006

- **Дано:** Два собственных RUB-счёта и перевод RUB 1 000 с комиссией RUB 10.
- **Когда:** Получены обе стороны перевода в любом порядке.
- **Тогда:** Связана одна операция перевода; основная сумма исключена из доходов/расходов, комиссия RUB 10 учтена один раз.
- **Уровень:** `integration`.

#### AC-082

- **Дано:** A оплачивает общий расход RUB 1000 и явно отмечает возмещение RUB 300 от B.
- **Когда:** B переводит 100, затем 200; обе стороны переводов импортируются повторно.
- **Тогда:** Долг уменьшается 300→200→0 один раз; основная сумма переводов не доход/расход. Обычная покупка без указания долг не создаёт; валютное погашение требует явного соответствия сумм.
- **Уровень:** `integration`.

#### AC-086

- **Дано:** A и B открыли одну версию операции или уточнения.
- **Когда:** Оба отправляют несовместимые изменения и повторяют один запрос.
- **Тогда:** Один результат применяется; второй получает конфликт с необходимостью перечитать состояние. Повтор не дублирует эффект; отмена создаёт новую проверенную revision и не стирает чужую последующую правку.
- **Уровень:** `integration`.

### Проверка результата

```sh
make test-go PKG=./internal/ledger/...
make test-integration AREA=family-reimbursements
```

Долг 300 погашается 100+200; повтор импорта не уменьшает его снова, неизвестное соответствие не списывает долг, семейный денежный факт не дублируется.

Команды `make` — будущий контракт, создаваемый task-1.1; сейчас они не существуют. Live/paid/manual проверки отдельно фиксируют доступ и фактический результат. Исследования не обходят блокер отсутствующего доступа.

### Передача следующему агенту

Записать изменённые контракты, команды и результаты, ограничения, незакрытые вопросы и разблокированные зависимости. Обновить обе языковые версии и трассировку. Закрывать задачу только по доказательству её результата; GitHub Closed само по себе не означает Ready MVP.

**Commit boundary:** логическая граница этой задачи; commit/push/deploy не разрешены данной карточкой и требуют действующей авторизации пользователя.

## EN

Track explicit inter-member debts and reimbursements.

**Status:** Blocked by dependencies and the SDD Ready gate; implementation has not started.

**Dependencies:** `task-2.4`, `task-2.8`.

**Kind:** `implementation`.

### Change and contracts

Keep an explicit creditor/debtor/asset/amount/revision record and an expense link when present. Paying a joint share does not automatically create debt. Settlement links once to an actual transfer/cash movement, supports partial settlement and requires explicitly agreed amount mapping for another currency. Exclude internal claims from household assets, income and expenses.

### Change boundaries

- `backend/internal/ledger/`
- `backend/internal/household/`
- `api/openapi.yaml`

These are planned paths. Shared contracts: `spec/001-want-keep-mvp/contracts.en.md`; architecture and commands: `constraints.en.md`. Change only the behavior owner and affected tests; an unresolved contract requires updated evidence and stops dependent implementation.

### Linked requirements

- **REQ-006:** Transfers between household accounts, including different members’ accounts, change balances without principal income or expense.
- **REQ-068:** An inter-member debt is recorded only explicitly and does not increase household assets or expenses.
- **REQ-072:** Competing edits, clarification answers and reversals check revision and current permissions while retaining both authors.

### Acceptance criteria

A criterion link establishes coverage; research or a partial task does not prove the entire product criterion. This task's exact outcome is specified in verification below.

#### AC-006

- **Given:** Two owned RUB accounts and a RUB 1,000 transfer with a RUB 10 fee.
- **When:** Both transfer legs arrive in either order.
- **Then:** One transfer is linked; principal is excluded from income/expenses and the RUB 10 fee is counted once.
- **Level:** `integration`.

#### AC-082

- **Given:** A pays a RUB 1,000 joint expense and explicitly records RUB 300 reimbursement due from B.
- **When:** B transfers 100 and then 200; both legs of the transfers are imported again.
- **Then:** Debt falls 300→200→0 once; transfer principal is not income/expense. An ordinary purchase creates no debt without instruction; cross-currency settlement requires explicit amount mapping.
- **Level:** `integration`.

#### AC-086

- **Given:** A and B opened the same transaction or clarification revision.
- **When:** Both submit conflicting edits and replay one request.
- **Then:** One result applies; the other receives a conflict requiring refresh. Replay does not duplicate effects; reversal creates a checked new revision without erasing the other member’s later edit.
- **Level:** `integration`.

### Verification

```sh
make test-go PKG=./internal/ledger/...
make test-integration AREA=family-reimbursements
```

Debt 300 settles through 100+200; replay does not reduce it again, uncertain matching does not settle debt and household cash facts are not duplicated.

The `make` commands are a future contract established by task-1.1; they do not exist yet. Live/paid/manual checks separately record access and actual outcomes. Research does not bypass missing-access blockers.

### Handoff to the next agent

Record changed contracts, commands/results, limitations, unresolved questions and unblocked dependencies. Update both languages and traceability. Close the task only with evidence of its outcome; GitHub Closed alone does not mean the MVP is Ready.

**Commit boundary:** this task's logical boundary; this card does not authorize commit/push/deploy, which require current user authorization.
