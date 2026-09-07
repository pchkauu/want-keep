<!-- want-keep-task: task-2.4 -->
# task-2.4 — Связать переводы и исключить дубликаты / Link transfers and prevent duplicates

## RU

Связывать доказательства одной операции без слияния разных покупок.

**Состояние:** Не начато; задача ожидает собственные зависимости и entry gates.

**Зависимости:** `task-2.3`.

**Тип:** `implementation`.

### Изменение и контракты

Разделить точную идемпотентность источника и экономическое сопоставление между источниками/чатом. Использовать подтверждённые ID/счета/суммы/валюты и фактические комиссии; совпадение суммы/времени только формирует кандидатов. Подтверждённые однозначные связи применяются автоматически, конкурирующие кандидаты идут в уточнения; частичные переводы остаются явными.

### Границы изменений

- `backend/internal/matching/`

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
make test-integration AREA=matching
```

Ранний чек, поздняя/обратная сторона, P2P-обмен, две одинаковые покупки и изменение комиссии не дают двойного учёта.

Команды `make` — будущий контракт, создаваемый task-1.1; сейчас они не существуют. Live/paid/manual проверки отдельно фиксируют доступ и фактический результат. Исследования не обходят блокер отсутствующего доступа.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Link evidence of one transaction without merging different purchases.

**Status:** Not started; the task awaits its own dependencies and entry gates.

**Dependencies:** `task-2.3`.

**Kind:** `implementation`.

### Change and contracts

Separate exact source idempotency from economic matching across sources/chat. Use verified IDs/accounts/amounts/currencies and actual fees; equal amount/time only yields candidates. Substantiated unambiguous links apply automatically; competing candidates require clarification; partial transfers remain explicit.

### Change boundaries

- `backend/internal/matching/`

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
make test-integration AREA=matching
```

Early receipts, late/reversed legs, P2P conversion, equal purchases and fee changes do not double count.

The `make` commands are a future contract established by task-1.1; they do not exist yet. Live/paid/manual checks separately record access and actual outcomes. Research does not bypass missing-access blockers.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
