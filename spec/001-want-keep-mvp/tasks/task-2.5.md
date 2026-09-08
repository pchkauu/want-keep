<!-- want-keep-task: task-2.5 -->
# task-2.5 — Сверять журнал с балансом источника / Reconcile the ledger to source balances

## RU

Показывать полноту истории и объяснимые расхождения.

**Состояние:** Реализованы backend/API сверки, ограниченный replay-контракт и явные owned/debt adjustments; provider IO, UI и production остаются последующим задачам.

**Зависимости:** `task-2.3`.

**Тип:** `implementation`.

### Изменение и контракты

Для каждого импортного счёта owned, available, locked и debt сравниваются отдельно на точный sourceAsOf; difference = source − ledger сохраняется точной native-суммой. Историческая проекция использует только доказанные к этому моменту effects, а неопределимый lifecycle делает компонент unknown. Новое наблюдение supersede предыдущую активную сверку; revision операции или открытия переоценивает текущую без события при идентичном результате. Discrepant/incomplete атомарно сохраняет durable outbox intent; worker после commit создаёт один дедуплицированный replay только для точной revision не более чем за 90 дней от последней подтверждённой точки или opening, с точными admission binding/revision и generation. Недоступность admission/авторизации сохраняется как unavailable без фиктивного задания. После completed/unavailable replay пользователь может атомарно разрешить только ненулевые известные owned/debt differences: сервер вычисляет adjustment, ledger сохраняет его без income/expense вместе с audit/outbox/review/command outcome. Available/locked напрямую не корректируются, source observation не изменяется.

### Границы изменений

- `backend/internal/reconciliation/`
- `backend/internal/storage/`
- `backend/internal/delivery/reconciliation/`
- `backend/migrations/011_reconciliation.sql`
- `api/`
- `backend/test/integration/reconciliation/`
- `backend/internal/jobs/application/`
- `backend/cmd/worker/`

### Экранный контракт

### SCR-012 — Сверка остатка

`/accounts/:id/reconciliation`

**Вопрос:** Почему остаток отличается?

**Главный ответ:** Owned, available, locked и debt источника и журнала на sourceAsOf, их точная разница, качество данных и доказанные объяснения.

**Структура сверху вниз:** Lifecycle/result и свежесть → четыре компонента source/ledger/difference → объяснения и связанные операции → replay → resolution.

**Следующее действие:** Запустить/дождаться повторной загрузки, повторно войти в источник, открыть связанные движения SCR-010 либо после completed/unavailable replay явно скорректировать owned/debt.

**Объяснение и детализация:** Unknown не равен нулю; balanced stale не становится fresh. Available/locked напрямую не корректируются. Явный adjustment не является доходом/расходом, не меняет снимок источника и заранее показывает рассчитанный эффект.

**Права:** Оба участника видят и исправляют факты любого счёта семьи; actor из сессии.

Forms: FORM-06.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-15.

#### FORM-06 — Исправление, сопоставление и отмена

**Поля:** Операция, expectedRevision, основание; полный principal и отдельные fees, дата покупки, payer, merchant/note. Пропуск сохраняет поле, пустой текст очищает. Undo: decisionId и expectedRevisions всех участников; исключение — отдельное действие. Сравнение до/после и с источником.

**Проверки и права:** Оба участника исправляют факты. Сервер сохраняет счета/активы principal, проверяет группы сумм, права, версии и происхождение; actor не задаётся формой. Undo сохраняет поздние независимые поля и отвергает пересечение/ABA. Сопоставление, категории и доли активируются профильными задачами.

**Результат:** Новое решение и финансовые revisions с историей, либо no_change/conflict без эффекта и потери ввода. Исключение не меняет банковский статус; undo пересчитывает текущий эффект.

- **UISTATE-01 — Загрузка:** Скелетон структуры и подпись загрузки; суммы не подменяются нулями.
- **UISTATE-02 — Обновление:** Сохранить предыдущие данные и контекст, показать время последнего успеха; блокировать только конфликтующие действия.
- **UISTATE-03 — Пусто:** Объяснить полезный результат и предложить первое действие: счёт, чек, план или цель.
- **UISTATE-05 — Частичные данные:** Назвать отсутствующий источник/период и последствия для суммы; доступные блоки работают; неизвестное обозначить отдельно.
- **UISTATE-06 — Устаревшие данные:** Показать дату последнего успеха и влияние на решение; дать обновить или перейти к подключению.
- **UISTATE-07 — Ошибка:** Понятная причина и следующий шаг у проблемного блока; ввод и исправные данные сохранить, диагностику раскрывать отдельно.
- **UISTATE-08 — Offline:** Показать отсутствие связи; не обещать сохранение. Чувствительные черновики только в памяти текущей вкладки, без новой offline-очереди.
- **UISTATE-09 — Сохранение:** Немедленно показать прогресс текущего действия и не допускать дублирующую отправку команды.
- **UISTATE-10 — Исход неизвестен:** Сохранить ID команды/ввод, запросить её результат; не создавать новую финансовую команду вслепую. После перезагрузки сверять серверный список недавних команд.
- **UISTATE-11 — Конфликт версии:** Показать авторов и различия, сохранить мой ввод; загрузить актуальную версию и дать повторно применить выбранные изменения после проверки.
- **UISTATE-12 — Недостаточно прав:** Финансовые данные доступны семье; запрещённое изменение объясняет владельца. Сервер отклоняет команду независимо от видимости кнопки.
- **UISTATE-13 — Сессия истекла:** Закрыть защищённое содержимое; вход для того же участника, безопасный возврат по внутреннему маршруту. Чужой вход не получает прежний черновик.
- **UISTATE-15 — Нужен банковский вход:** Назвать подключение и владельца, дать ему безопасно войти; партнёру показать ожидание без доступа к секрету.
- **UISTATE-16 — Подтверждено:** После подтверждённого сервером результата показать что изменилось, ссылку на объект и доступное исправление; не полагаться на исчезающий toast.


Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-004:** Начало учёта задаётся датой; начальные остатки отделены от доходов и расходов.
- **REQ-005:** Счета показывают собственные, доступные, заблокированные и заёмные средства в пределах данных источника.
- **REQ-013:** Расхождение журнала и баланса источника расследуется без скрытого автоматического выравнивания.
- **REQ-040:** Каждый источник обновляется раз в час и по запросу с видимым временем успешного обновления.
- **REQ-041:** История сохраняет границы покрытия, курсоры, пробелы и статусы источника.
- **REQ-058:** Операционные статусы показывают ошибки импорта, AI, курсов, резервирования и расходы без утечки финансового содержимого.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

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
make test-go PKG=./internal/reconciliation/... && make test-integration AREA=reconciliation && make test-reconciliation-race
```

Шесть активов, четыре компонента, sourceAsOf/lifecycle, unknown/partial/stale, durable post-commit replay 90 дней, admission/reauth, переоценка, adjustment, rollback/replay, права, пагинация и миграция проходят без ложного дохода или повторного эффекта.

Команды `make` реализованы и обязательны для локальной и CI-проверки task-2.5. Live provider IO, браузерная приёмка и production остаются последующим задачам и не подтверждаются этими suites.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Expose history completeness and explainable discrepancies.

**Status:** Reconciliation backend/API, bounded replay contract and explicit owned/debt adjustments are implemented; provider IO, UI and production remain downstream.

**Dependencies:** `task-2.3`.

**Kind:** `implementation`.

### Change and contracts

For each imported account, owned, available, locked and debt are compared independently at the exact sourceAsOf; difference = source − ledger is retained as an exact native amount. The historical projection uses only effects proven by that instant, while indeterminate lifecycle timing makes the affected component unknown. A new observation supersedes the previous active reconciliation; an operation or opening revision re-evaluates the current one without an event for an identical result. Discrepant/incomplete atomically stores a durable outbox intent; after commit the worker creates one deduplicated replay only for the exact revision, at most 90 days from the last confirmed point or opening, with exact admission binding/revision and generation. Missing admission or authorization records unavailable without a fake job. After completed/unavailable replay, a user may atomically resolve only nonzero known owned/debt differences: the server derives the adjustment and the ledger stores it without income/expense together with audit/outbox/review/command outcome. Available/locked cannot be adjusted directly and source observations remain immutable.

### Change boundaries

- `backend/internal/reconciliation/`
- `backend/internal/storage/`
- `backend/internal/delivery/reconciliation/`
- `backend/migrations/011_reconciliation.sql`
- `api/`
- `backend/test/integration/reconciliation/`
- `backend/internal/jobs/application/`
- `backend/cmd/worker/`

### Screen contract

### SCR-012 — Balance reconciliation

`/accounts/:id/reconciliation`

**Question:** Why does the balance differ?

**Primary answer:** Source and ledger owned, available, locked and debt at sourceAsOf, their exact differences, data quality and evidence-backed explanations.

**Top-down structure:** Lifecycle/result and freshness → four source/ledger/difference components → explanations and related transactions → replay → resolution.

**Next action:** Start/wait for bounded replay, reauthenticate the source, open related movements in SCR-010 or, after completed/unavailable replay, explicitly adjust owned/debt.

**Explanation and details:** Unknown is not zero; balanced stale does not become fresh. Available/locked cannot be adjusted directly. An explicit adjustment is not income/expense, never changes the source observation and previews the server-derived effect.

**Permissions:** Both members read/correct facts for any household account; actor from session.

Forms: FORM-06.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-15.

#### FORM-06 — Correction, matching and undo

**Fields:** Transaction, expectedRevision and reason; complete principal and separate fees, purchase time, payer, merchant/note. Omission retains a field; empty text clears it. Undo: decisionId and all participant expectedRevisions; exclusion is a separate action. Compare before/after and source values.

**Validation and permissions:** Both members correct facts. The server preserves principal accounts/assets and validates monetary groups, rights, versions and provenance; the form cannot assign actor. Undo preserves later independent fields and rejects overlaps/ABA. Matching, categories and shares are activated by their owning tasks.

**Outcome:** New decision and financial revisions with history, or no_change/conflict without effect or lost input. Exclusion does not change bank state; undo recomputes the current effect.

- **UISTATE-01 — Loading:** Structural skeleton and loading label; amounts are never replaced by zero.
- **UISTATE-02 — Refreshing:** Keep previous data/context and last-success time; block only conflicting actions.
- **UISTATE-03 — Empty:** Explain the useful outcome and offer a first account, receipt, plan or goal action.
- **UISTATE-05 — Partial data:** Name the missing source/period and its effect on the amount; available sections work and unknowns stay explicit.
- **UISTATE-06 — Stale data:** Show last-success date and impact on the decision; offer refresh or connection details.
- **UISTATE-07 — Error:** Plain cause and next step beside the affected section; preserve input/healthy data and expand diagnostics separately.
- **UISTATE-08 — Offline:** Show missing connectivity and do not promise saved data. Sensitive drafts remain only in current-tab memory, without a new offline queue.
- **UISTATE-09 — Saving:** Immediately show current-action progress and prevent duplicate command submission.
- **UISTATE-10 — Unknown outcome:** Keep command ID/input and query its result; never blindly create another financial command. After reload reconcile the server list of recent commands.
- **UISTATE-11 — Version conflict:** Show authors/differences and keep my input; load current version and allow chosen changes to be reapplied after validation.
- **UISTATE-12 — Insufficient permission:** Household can read financial data; forbidden edits explain ownership. Server rejects the command regardless of button visibility.
- **UISTATE-13 — Session expired:** Hide protected contents; require the same member to sign in and return through a safe internal route. Another identity never receives the prior draft.
- **UISTATE-15 — Bank sign-in needed:** Name connection and owner, offer safe owner sign-in; partner sees waiting without secret access.
- **UISTATE-16 — Confirmed:** After server-confirmed outcome show what changed, an object link and available correction; do not rely on a disappearing toast.


Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-004:** Accounting starts on a selected date; opening balances are separate from income and expenses.
- **REQ-005:** Accounts distinguish owned, available, locked and borrowed amounts where the source provides them.
- **REQ-013:** Ledger/source balance discrepancies are investigated without hidden automatic balancing.
- **REQ-040:** Each source refreshes hourly and on demand with a visible last-success timestamp.
- **REQ-041:** History retains coverage boundaries, cursors, gaps and source status.
- **REQ-058:** Operational status exposes import, AI, FX, backup failures and spend without leaking financial content.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

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
make test-go PKG=./internal/reconciliation/... && make test-integration AREA=reconciliation && make test-reconciliation-race
```

Six assets, four components, sourceAsOf/lifecycle, unknown/partial/stale, durable post-commit 90-day replay, admission/reauth, re-evaluation, adjustment, rollback/replay, permissions, pagination and migration pass without false income or duplicate effects.

The `make` commands are implemented and required for local and CI validation of task-2.5. Live provider IO, browser acceptance and production remain downstream and are not proven by these suites.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
