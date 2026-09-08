<!-- want-keep-task: task-2.4 -->
# task-2.4 — Связать переводы и исключить дубликаты / Link transfers and prevent duplicates

## RU

Связывать доказательства одной операции без слияния разных покупок.

**Состояние:** Реализован backend/API сопоставления, ожидания без второго эффекта, связывания существующих движений и составной отмены. Проверки и границы фиксируются в evidence/task-2.4-matching.md.

**Зависимости:** `task-2.3`.

**Тип:** `implementation`.

### Изменение и контракты

D-39 остаётся идентичностью источника. Проверенная структурированная связь с namespace, сетью/движением и точными суммами допускает автоматическое сопоставление; сумма, дата и hash файла дают только кандидата. Поиск ID охватывает сохранённую историю; вероятные совпадения — ±7 календарных дней, полнота явная. Возможный дубль сохраняется со статусом банка, но без дополнительного эффекта до решения. Связь содержит одну исходящую и одну входящую сторону, отдельные комиссии и одного носителя каждого эффекта. Собственные стороны без пары не становятся доходом/расходом. Выбор основной записи для показа не назначает носителя эффекта или приоритет правок. Link/resolve, полевые исправления и undo проверяют все revisions; история, проекции, review/outbox и команда атомарны. CommitPage сохраняет checkpoint с matching_unresolved и отвергает устаревшие jobs. Undo сохраняет основание прежнего случая и полноту кандидатов; производный пересчёт не заменяет происхождение связи. Комиссия до principal сохраняется без повторного расхода.

### Границы изменений

- `backend/internal/matching/`
- `backend/internal/ledger/`
- `backend/internal/accounts/application/`
- `backend/internal/storage/`
- `backend/internal/delivery/ledger/`
- `backend/migrations/010_transaction_matching.sql`
- `backend/migrations/011_matching_decision_basis.sql`
- `api/`
- `backend/test/integration/matching/`

### Экранный контракт

### SCR-010 — Карточка операции

`/transactions/:id`

**Вопрос:** Правильно ли учтена эта покупка?

**Главный ответ:** Сумма, назначение, плательщик и доли одной операции.

**Структура сверху вниз:** Результат учёта → счёт/дата/статус → личные/общие доли → чек → исправить/возврат.

**Следующее действие:** Исправить FORM-06/07, вернуть FORM-08, явный долг FORM-09; чек → SCR-011.

**Объяснение и детализация:** История до/после с автором, временем, decisionId и основаниями; отдельные банковское и учётное состояния. Защищённые поля сравниваются с нормализованным источником; review показывает безопасное обоснование и ссылки на evidence. Для выбранного решения видны возможность undo и причина отказа. Группа показывает участников, evidence, носителей эффекта, отдельное ожидание matching_unresolved и конфликт. Список кандидатов сообщает полноту; основная запись не означает приоритет правок. Link/resolve и составные исправления используют версии всех участников; undo сохраняет независимые правки и состояния банка.

**Права:** Оба участника видят и исправляют факты любого счёта семьи; actor из сессии.

Forms: FORM-06, FORM-07, FORM-08, FORM-09, FORM-05.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-14.

#### FORM-05 — Учесть перевод или обмен

**Поля:** Откуда/куда, даты, обе суммы/валюты, комиссии и счёт комиссии, существующие движения.

**Проверки и права:** Оба участника; разные счета семьи; одна исходящая и одна входящая сторона. Principal не входит в доходы/расходы; отдельные комиссии, включая третий актив. Пустой existingTransactions создаёт новое движение. Непустой список содержит ID/revisions всех участников (до 100); суммы, комиссии и дата основной записи проверяются без создания недостающих сторон. Связь переводов/обменов/оплаты подтверждается по версиям, неоднозначность остаётся в matching; отдельная покупка снимает ожидание один раз.

**Результат:** Связано движение денег в журнале; никакой реальной отправки или покупки актива.

#### FORM-06 — Исправление, сопоставление и отмена

**Поля:** Операция, expectedRevision, основание; полный principal и отдельные fees, дата покупки, payer, merchant/note. Пропуск сохраняет поле, пустой текст очищает. Undo: decisionId и expectedRevisions всех участников; исключение — отдельное действие. Сравнение до/после и с источником.

**Проверки и права:** Оба участника исправляют факты. Сервер сохраняет счета/активы principal, проверяет группы сумм, права, версии и происхождение; actor не задаётся формой. Undo сохраняет поздние независимые поля и отвергает пересечение/ABA. Сопоставление, категории и доли активируются профильными задачами.

**Результат:** Новое решение и финансовые revisions с историей, либо no_change/conflict без эффекта и потери ввода. Исключение не меняет банковский статус; undo пересчитывает текущий эффект.

#### FORM-07 — Чек и распределение позиций

**Поля:** Фото/PDF, обязательный счёт списания включая наличные; позиции, скидки, категории, personal/shared и доли % или суммы.

**Проверки и права:** Оба member; лимиты файлов по контракту, позиции/скидки/доли точно равны оплате. Неоднозначность уточняется; AI не исполняет инструкции файла.

**Результат:** Создано/связано с существующим/ожидает уточнения/документ не подходит с причиной. Одно подтверждённое списание.

#### FORM-08 — Возврат покупки

**Поля:** Исходная покупка, возвращаемые позиции/доли/сумма, счёт поступления и фактическая дата.

**Проверки и права:** Оба member; совокупный возврат не больше покупки; исходные исторические FX и распределение по возвращённой части сохраняются.

**Результат:** Исходный месяц покупки пересчитан; деньги поступили текущей датой; FX отдельно.

#### FORM-09 — Явный долг и возмещение

**Поля:** Кто кому, сумма/валюта, основание/расход; при погашении существующий семейный перевод и сумма связи.

**Проверки и права:** Только явное действие member; не выводить долг из долей. Нельзя повторно погасить одним переводом сверх его суммы; долг не капитал семьи.

**Результат:** Непогашенный остаток обновлён без нового семейного расхода.

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
- **UISTATE-14 — Ожидание AI:** Отличать очередь, обработку, уточнение и паузу из-за лимита/API; обычный учёт доступен, результат не выдумывать.
- **UISTATE-16 — Подтверждено:** После подтверждённого сервером результата показать что изменилось, ссылку на объект и доступное исправление; не полагаться на исчезающий toast.


Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-006:** Перевод между счетами семьи, включая счета разных участников, меняет остатки без дохода или расхода по основной сумме.
- **REQ-007:** Обмен и P2P-конвертация собственных денег сохраняют обе валютные суммы, фактический курс и комиссии.
- **REQ-008:** Повторные импорты, чек и запись чата объединяют доказательства одной операции без повторного учёта.
- **REQ-012:** Исправление учёта сохраняет оригинал, автора, основание, версию и возможность отмены решения.
- **REQ-017:** Чат создаёт установленную операцию, уточняет недостающие данные и явно объясняет пропуск неподходящего документа.
- **REQ-018:** Каждая новая или содержательно изменённая операция получает AI-проверку своей версии.
- **REQ-019:** AI автоматизирует внутренний учёт через проверяемые команды; неопределённость остаётся явной.
- **REQ-061:** Повторные задания, перезапуски и параллельные изменения не создают двойных финансовых эффектов.
- **REQ-065:** Принадлежность счёта, владелец внешнего аккаунта, автор записи и принадлежность расхода являются отдельными признаками.
- **REQ-067:** Расходы и позиции чеков имеют личное или совместное назначение; общая доля по умолчанию 50/50 с исключениями статьи или покупки.
- **REQ-068:** Взаимный долг учитывается только по явному указанию и не увеличивает активы или расходы семьи.
- **REQ-071:** Один общий чат сохраняет автора сообщения и проверяет полномочия инициатора AI-команды при исполнении.
- **REQ-072:** Конкурирующие изменения, ответы на уточнения и отмены проверяют версию и текущие права, сохраняя обоих авторов.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

#### AC-006

- **Дано:** Два собственных RUB-счёта и перевод RUB 1 000 с комиссией RUB 10.
- **Когда:** Получены обе стороны перевода в любом порядке.
- **Тогда:** Связана одна операция перевода; основная сумма исключена из доходов/расходов, комиссия RUB 10 учтена один раз.
- **Уровень:** `integration`.

#### AC-007

- **Дано:** Собственные RUB 9 000 обменены на USDT 100, комиссия RUB 50.
- **Когда:** Приходят банковская и криптовалютная стороны подтверждённого обмена.
- **Тогда:** Основные суммы не становятся расходом/доходом; курс RUB 90/USDT отделён от комиссии; неоднозначная связь требует уточнения.
- **Уровень:** `integration`.

#### AC-008

- **Дано:** Расход RUB 300 создан из чата с выбранным счётом.
- **Когда:** Поступают соответствующий чек, банковская операция и повтор той же операции.
- **Тогда:** Расход остаётся RUB 300, все источники связаны; две отдельные покупки одной суммы не объединяются лишь из-за равенства суммы.
- **Уровень:** `integration`.

#### AC-019

- **Дано:** AI предлагает сумму, противоречащую источнику, и связь с несколькими кандидатами.
- **Когда:** Приложение проверяет предложения.
- **Тогда:** Противоречивое изменение отклонено, неоднозначность поступает в очередь уточнений; категории и подтверждённые связи могут применяться автоматически.
- **Уровень:** `integration`.

#### AC-063

- **Дано:** Приходящая сторона USDT 100 уже импортирована; исходящая RUB 9 000 и комиссия ещё отсутствуют.
- **Когда:** Приходят поздняя сторона, исправление комиссии и повтор старой страницы.
- **Тогда:** Состояние ожидания связи сменяется проверенным обменом; доход/расход основной суммы не удваивается, устаревшая комиссия не восстанавливается.
- **Уровень:** `integration`.

#### AC-064

- **Дано:** На одном счёте две покупки RUB 300 близко по времени; владелец исправил одну.
- **Когда:** AI получает чек без уникального идентификатора и старый результат классификации.
- **Тогда:** Запрашивается выбор совпадения; покупки не сливаются по вероятности; правка человека не теряется.
- **Уровень:** `integration`.

#### AC-079

- **Дано:** A и B имеют разные аккаунты одного провайдера и общий счёт; B заносит покупку A со счёта B.
- **Когда:** Выполняются ввод, импорт обоих аккаунтов и повторное подключение того же внешнего аккаунта.
- **Тогда:** Разные аккаунты не сливаются; повторный источник не удваивает остатки. Плательщик, автор и получатель расхода сохраняются независимо. Неустановленное совпадение блокирует новый учёт до уточнения.
- **Уровень:** `integration`.

#### AC-082

- **Дано:** A оплачивает общий расход RUB 1000 и явно отмечает возмещение RUB 300 от B.
- **Когда:** B переводит 100, затем 200; обе стороны переводов импортируются повторно.
- **Тогда:** Долг уменьшается 300→200→0 один раз; основная сумма переводов не доход/расход. Обычная покупка без указания долг не создаёт; валютное погашение требует явного соответствия сумм.
- **Уровень:** `integration`.

#### AC-093

- **Дано:** A и B присылают один и тот же чек с одним счётом; затем приходит банковская операция.
- **Когда:** Обрабатываются параллельные сообщения, повтор файла и импорт.
- **Тогда:** Создаётся один денежный эффект и несколько evidence с авторами; при отсутствии доказанного совпадения требуется уточнение, одинаковые суммы разных счетов не сливаются.
- **Уровень:** `integration`.

### Проверка результата

```sh
make test-go PKG=./internal/matching/... && make test-integration AREA=matching && make test-matching-race
```

Шесть активов; ручная оплата, нормализованный чек и банк; разные/вероятные покупки; обе последовательности сторон, комиссии, lifecycle, источники, составные правки/undo, конкуренция, права, replay/rollback/restart и миграция проверяются без двойного эффекта.

Task-2.3 включена в базу; команды существуют. Обязательны make check, matching/audit/ledger/accounts/storage/identity/household integration/race, privacy и git diff --check. Реальные чеки/чат/OpenAI, банковский IO, возвраты, долг, экраны и эксплуатация не подтверждаются.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Link evidence of one transaction without merging different purchases.

**Status:** Matching, waiting without a second effect, existing movement linking and compound undo backend/API are implemented. Verification and boundaries are recorded in evidence/task-2.4-matching.en.md.

**Dependencies:** `task-2.3`.

**Kind:** `implementation`.

### Change and contracts

D-39 remains source identity. Verified structured correspondence with namespace, network/movement and exact amounts permits automatic matching; amount, date and file hash only yield candidates. Identifier search spans retained history; probable matching uses ±7 calendar days with explicit completeness. A possible duplicate retains bank status without an additional effect until resolution. A link has one outgoing and one incoming principal, separate fees and one carrier per effect. A known internal side without its counterpart is not income/expense. Display primary does not select effect ownership or override priority. Link/resolve, field corrections and undo check all revisions; history, projections, review/outbox and command outcome are atomic. CommitPage retains a checkpoint with matching_unresolved and rejects stale jobs. Undo retains the prior case basis and candidate completeness; derived recalculation does not replace association provenance. Fee-before-principal is retained without a repeated expense.

### Change boundaries

- `backend/internal/matching/`
- `backend/internal/ledger/`
- `backend/internal/accounts/application/`
- `backend/internal/storage/`
- `backend/internal/delivery/ledger/`
- `backend/migrations/010_transaction_matching.sql`
- `backend/migrations/011_matching_decision_basis.sql`
- `api/`
- `backend/test/integration/matching/`

### Screen contract

### SCR-010 — Transaction details

`/transactions/:id`

**Question:** Is this purchase accounted for correctly?

**Primary answer:** Amount, purpose, payer and shares of one transaction.

**Top-down structure:** Accounting outcome → account/date/status → personal/shared shares → receipt → correction/refund.

**Next action:** Correct FORM-06/07, refund FORM-08, explicit debt FORM-09; receipt → SCR-011.

**Explanation and details:** Before/after history with actor, time, decisionId and reasons; separate bank and accounting states. Protected fields can be compared with normalized source values; review shows a safe rationale and evidence references. Each decision exposes undo availability and rejection reason. A group exposes participants, evidence, effect carriers, matching_unresolved waiting and conflicts. Candidate completeness is explicit; primary does not imply override priority. Link/resolve and compound corrections use all participant revisions; undo preserves independent edits and bank states.

**Permissions:** Both members read/correct facts for any household account; actor from session.

Forms: FORM-06, FORM-07, FORM-08, FORM-09, FORM-05.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-14.

#### FORM-05 — Record transfer or exchange

**Fields:** From/to accounts, dates, both amounts/currencies, fees/fee account, existing movements.

**Validation and permissions:** Either member; distinct household accounts; one outgoing and one incoming side. Principal is excluded from income/expenses; separate fees may use a third asset. Empty existingTransactions creates new movement. A nonempty list supplies all participant IDs/revisions (up to 100); amounts, fees and primary date are checked without creating missing sides. Transfer/exchange/payment links require version checks; ambiguity remains in matching; separate-purchase confirmation releases waiting once.

**Outcome:** Ledger movements linked; no actual transfer or asset purchase.

#### FORM-06 — Correction, matching and undo

**Fields:** Transaction, expectedRevision and reason; complete principal and separate fees, purchase time, payer, merchant/note. Omission retains a field; empty text clears it. Undo: decisionId and all participant expectedRevisions; exclusion is a separate action. Compare before/after and source values.

**Validation and permissions:** Both members correct facts. The server preserves principal accounts/assets and validates monetary groups, rights, versions and provenance; the form cannot assign actor. Undo preserves later independent fields and rejects overlaps/ABA. Matching, categories and shares are activated by their owning tasks.

**Outcome:** New decision and financial revisions with history, or no_change/conflict without effect or lost input. Exclusion does not change bank state; undo recomputes the current effect.

#### FORM-07 — Receipt and item allocation

**Fields:** Photo/PDF, required debit account including cash; items, discounts, categories, personal/shared and percentage or amount shares.

**Validation and permissions:** Either member; file limits from contract, items/discounts/shares exactly equal payment. Ambiguity requires clarification; AI never executes file instructions.

**Outcome:** Created/linked to existing/awaiting clarification/document unsuitable with reason. One confirmed debit.

#### FORM-08 — Purchase refund

**Fields:** Original purchase, returned items/shares/amount, receiving account and actual date.

**Validation and permissions:** Either member; cumulative refund cannot exceed purchase; original historical FX and refunded-part allocation are retained.

**Outcome:** Original purchase month recalculated; cash arrives on actual date; FX separate.

#### FORM-09 — Explicit debt and reimbursement

**Fields:** Debtor/creditor, amount/currency, reason/expense; for settlement an existing household transfer and linked amount.

**Validation and permissions:** Explicit member action only; never infer debt from shares. One transfer cannot settle beyond its amount; debt is not household wealth.

**Outcome:** Outstanding balance updated without another household expense.

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
- **UISTATE-14 — AI waiting:** Distinguish queued, processing, clarification and budget/API pause; ordinary accounting remains available and results are not invented.
- **UISTATE-16 — Confirmed:** After server-confirmed outcome show what changed, an object link and available correction; do not rely on a disappearing toast.


Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-006:** Transfers between household accounts, including different members’ accounts, change balances without principal income or expense.
- **REQ-007:** Exchange and P2P conversion of owned money preserve both currency amounts, the actual rate and fees.
- **REQ-008:** Repeated imports, receipts and chat entries combine evidence of one transaction without double counting.
- **REQ-012:** Accounting corrections preserve the original, actor, reason, version and ability to undo a decision.
- **REQ-017:** Chat records an established transaction, clarifies missing data and explicitly explains skipped irrelevant documents.
- **REQ-018:** Every new or materially changed transaction receives AI review of its version.
- **REQ-019:** AI automates internal accounting through validated commands; uncertainty remains explicit.
- **REQ-061:** Repeated jobs, restarts and concurrent changes cannot create duplicate financial effects.
- **REQ-065:** Account ownership, external-account owner, record author and expense attribution are distinct dimensions.
- **REQ-067:** Expenses and receipt items have personal or joint attribution; joint shares default to 50/50 with line or purchase overrides.
- **REQ-068:** An inter-member debt is recorded only explicitly and does not increase household assets or expenses.
- **REQ-071:** One shared chat retains message authors and checks the AI command initiator’s authority at execution.
- **REQ-072:** Competing edits, clarification answers and reversals check revision and current permissions while retaining both authors.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

#### AC-006

- **Given:** Two owned RUB accounts and a RUB 1,000 transfer with a RUB 10 fee.
- **When:** Both transfer legs arrive in either order.
- **Then:** One transfer is linked; principal is excluded from income/expenses and the RUB 10 fee is counted once.
- **Level:** `integration`.

#### AC-007

- **Given:** Owned RUB 9,000 is exchanged for USDT 100 with a RUB 50 fee.
- **When:** The bank and crypto legs of a confirmed exchange arrive.
- **Then:** Principal is not income/expense; the RUB 90/USDT rate is separate from the fee; an ambiguous match requires clarification.
- **Level:** `integration`.

#### AC-008

- **Given:** A RUB 300 expense was created from chat for a selected account.
- **When:** The matching receipt, bank transaction and duplicate bank delivery arrive.
- **Then:** Expense remains RUB 300 and all evidence is linked; separate equal-amount purchases are not merged merely by amount.
- **Level:** `integration`.

#### AC-019

- **Given:** AI proposes a source-conflicting amount and a match with several candidates.
- **When:** The application validates the proposals.
- **Then:** The conflicting change is rejected and ambiguity enters the clarification queue; categories and substantiated links may be applied automatically.
- **Level:** `integration`.

#### AC-063

- **Given:** The incoming USDT 100 leg is imported; outgoing RUB 9,000 and fee are missing.
- **When:** The late leg, fee correction and replayed old page arrive.
- **Then:** Pending matching becomes a verified exchange; principal is not double-counted and the stale fee is not restored.
- **Level:** `integration`.

#### AC-064

- **Given:** One account has two RUB 300 purchases close in time; the owner corrected one.
- **When:** AI receives a receipt without a unique identifier and a stale classification result.
- **Then:** A match selection is requested; purchases are not merged merely by likelihood and the owner's correction survives.
- **Level:** `integration`.

#### AC-079

- **Given:** A and B have separate accounts at one provider and a joint account; B enters A’s purchase paid from B’s account.
- **When:** Entry, import of both accounts and reconnection of the same external account run.
- **Then:** Distinct accounts are not merged; a repeated source does not double balances. Payer, author and expense beneficiary remain independent. Unresolved source identity blocks new posting pending clarification.
- **Level:** `integration`.

#### AC-082

- **Given:** A pays a RUB 1,000 joint expense and explicitly records RUB 300 reimbursement due from B.
- **When:** B transfers 100 and then 200; both legs of the transfers are imported again.
- **Then:** Debt falls 300→200→0 once; transfer principal is not income/expense. An ordinary purchase creates no debt without instruction; cross-currency settlement requires explicit amount mapping.
- **Level:** `integration`.

#### AC-093

- **Given:** A and B submit the same receipt for one account; the bank transaction arrives later.
- **When:** Parallel messages, a file replay and import are processed.
- **Then:** One financial effect and multiple authored evidence records result; an unproven match requires clarification and equal amounts from different accounts are not merged.
- **Level:** `integration`.

### Verification

```sh
make test-go PKG=./internal/matching/... && make test-integration AREA=matching && make test-matching-race
```

Six assets; manual payment, normalized receipt and bank; distinct/probable purchases; both side arrival orders, fees, lifecycle, sources, compound corrections/undo, concurrency, rights, replay/rollback/restart and migration are checked without duplicate effects.

Task-2.3 is included in the base; commands exist. Require make check, matching/audit/ledger/accounts/storage/identity/household integration/race, privacy and git diff --check. Real receipts/chat/OpenAI, bank IO, refunds, debt, screens and operations are not verified.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
