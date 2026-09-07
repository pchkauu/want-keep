<!-- want-keep-task: task-5.4 -->
# task-5.4 — Реализовать чат и очередь уточнений / Implement chat and clarification queue

## RU

Сохранять диалог до однозначного результата учёта.

**Состояние:** Не начато; задача ожидает собственные зависимости и entry gates.

**Зависимости:** `task-5.3`, `task-5.2`.

**Тип:** `implementation`.

### Изменение и контракты

Связать сообщения, выбранный счёт, вложения, draft/proposal/operation и уточнения. Полная новая операция создаётся либо связывается с подтверждённым совпадением; недостаточные данные запрашиваются адресно. Ответ на уточнение применяет текущую версию один раз. Сохранять понятные состояния processing/waiting/skipped/recorded, возможность исправления и независимость обычного учёта от AI. Один общий чат, account picker включает личные счета обоих и семейные. Подтверждение принадлежит вошедшему участнику, а не названному в тексте; личные планы/цели подтверждает владелец. Ответы на одно уточнение версионны.

### Границы изменений

- `backend/internal/chat/`
- `backend/internal/clarifications/`

Это планируемые пути. Общие контракты: `spec/001-want-keep-mvp/contracts.md`; архитектура и команды: `constraints.md`. Менять только владельца поведения и затронутые тесты; при незакрытом контракте обновить evidence и остановить зависимую реализацию.

### Связанные требования

- **REQ-008:** Повторные импорты, чек и запись чата объединяют доказательства одной операции без повторного учёта.
- **REQ-015:** Для отправки чека требуется счёт списания; фото/PDF остаётся связанным с результатом обработки.
- **REQ-017:** Чат создаёт установленную операцию, уточняет недостающие данные и явно объясняет пропуск неподходящего документа.
- **REQ-018:** Каждая новая или содержательно изменённая операция получает AI-проверку своей версии.
- **REQ-019:** AI автоматизирует внутренний учёт через проверяемые команды; неопределённость остаётся явной.
- **REQ-021:** AI меняет утверждённый бюджет, прогноз доходов или цели только по явному решению участника с правом на изменение.
- **REQ-051:** AI ограничен бюджетом $50/месяц и деградирует в очередь ожидания без остановки обычного учёта.
- **REQ-064:** Оба участника видят все финансовые данные и изменяют операции; личные цели и части плана изменяет только их владелец.
- **REQ-065:** Принадлежность счёта, владелец внешнего аккаунта, автор записи и принадлежность расхода являются отдельными признаками.
- **REQ-067:** Расходы и позиции чеков имеют личное или совместное назначение; общая доля по умолчанию 50/50 с исключениями статьи или покупки.
- **REQ-071:** Один общий чат сохраняет автора сообщения и проверяет полномочия инициатора AI-команды при исполнении.
- **REQ-072:** Конкурирующие изменения, ответы на уточнения и отмены проверяют версию и текущие права, сохраняя обоих авторов.

### Критерии приёмки

Связь с критерием задаёт покрытие; исследование или частичная задача не доказывает весь критерий продукта. Точный результат этой задачи указан ниже в проверке.

#### AC-008

- **Дано:** Расход RUB 300 создан из чата с выбранным счётом.
- **Когда:** Поступают соответствующий чек, банковская операция и повтор той же операции.
- **Тогда:** Расход остаётся RUB 300, все источники связаны; две отдельные покупки одной суммы не объединяются лишь из-за равенства суммы.
- **Уровень:** `integration`.

#### AC-015

- **Дано:** Загружено фото или PDF чека.
- **Когда:** Владелец отправляет его без счёта, затем выбирает наличный счёт.
- **Тогда:** Без выбора запись не проводится; после выбора обработка использует этот счёт. Оба аутентифицированных участника семьи читают вложение, включая чек партнёра; посторонний и участник другой семьи получают отказ.
- **Уровень:** `end-to-end`.

#### AC-017

- **Дано:** Присланы полный расход, расход без суммы и очевидно нерелевантное фото.
- **Когда:** AI обрабатывает сообщения.
- **Тогда:** Первый расход записан или связан с существующим; второй ждёт вопроса; третье сообщение получает явную причину пропуска; вымышленных сумм нет.
- **Уровень:** `end-to-end`.

#### AC-018

- **Дано:** Есть импортированная, ручная и созданная чатом операции.
- **Когда:** Они создаются, а затем одна финансовая запись исправляется.
- **Тогда:** Каждая актуальная версия поставлена на проверку; повторная доставка задачи не дублирует эффект; устаревший ответ AI не меняет новую версию.
- **Уровень:** `integration`.

#### AC-019

- **Дано:** AI предлагает сумму, противоречащую источнику, и связь с несколькими кандидатами.
- **Когда:** Приложение проверяет предложения.
- **Тогда:** Противоречивое изменение отклонено, неоднозначность поступает в очередь уточнений; категории и подтверждённые связи могут применяться автоматически.
- **Уровень:** `integration`.

#### AC-021

- **Дано:** Есть утверждённый бюджет и предложение перераспределения.
- **Когда:** Приходит новый расход, затем уполномоченный участник подтверждает предложенное изменение.
- **Тогда:** До подтверждения план неизменен; подтверждение применяет показанную версию предложения один раз; устаревшее предложение пересогласуется.
- **Уровень:** `integration`.

#### AC-051

- **Дано:** OpenAI недоступен либо израсходован разрешённый бюджет с резервами текущих запросов.
- **Когда:** Поступают новый импорт, ручной расход и запрос AI.
- **Тогда:** Учёт и расчёты доступны; статус AI ожидает; новые платные запросы не запускаются сверх разрешённого резерва; неизвестная стоимость не освобождается молча.
- **Уровень:** `integration`.

#### AC-078

- **Дано:** У A есть личная цель и статья плана; у семьи общая статья и операции обоих.
- **Когда:** B читает все данные, исправляет операцию A и общий план, затем пытается изменить личную цель/план A через API и AI.
- **Тогда:** Чтение, операции и общее изменение разрешены; личные план/цель A защищены сервером. Одного уполномоченного подтверждения достаточно, второй уведомлён.
- **Уровень:** `end-to-end`.

#### AC-085

- **Дано:** Оба видят общий чат; A имеет личную цель, B просит AI изменить её от имени A.
- **Когда:** Модель предлагает действие, а затем A подтверждает новую адресованную ему версию предложения.
- **Тогда:** Сообщение B не выдаёт полномочия A; до разрешённого подтверждения изменения нет. Аудит хранит автора сообщения, подтвердившего и AI-основание; секретов в чате нет.
- **Уровень:** `integration`.

#### AC-086

- **Дано:** A и B открыли одну версию операции или уточнения.
- **Когда:** Оба отправляют несовместимые изменения и повторяют один запрос.
- **Тогда:** Один результат применяется; второй получает конфликт с необходимостью перечитать состояние. Повтор не дублирует эффект; отмена создаёт новую проверенную revision и не стирает чужую последующую правку.
- **Уровень:** `integration`.

#### AC-093

- **Дано:** A и B присылают один и тот же чек с одним счётом; затем приходит банковская операция.
- **Когда:** Обрабатываются параллельные сообщения, повтор файла и импорт.
- **Тогда:** Создаётся один денежный эффект и несколько evidence с авторами; при отсутствии доказанного совпадения требуется уточнение, одинаковые суммы разных счетов не сливаются.
- **Уровень:** `integration`.

### Проверка результата

```sh
make test-integration AREA=chat
```

Повтор отправки/ответа и обновление банковской операции во время диалога не создают дубль; каждое сообщение имеет объяснимый результат.

Команды `make` — будущий контракт, создаваемый task-1.1; сейчас они не существуют. Live/paid/manual проверки отдельно фиксируют доступ и фактический результат. Исследования не обходят блокер отсутствующего доступа.

### Передача следующему агенту

Записать изменённые контракты, команды и результаты, ограничения, незакрытые вопросы и разблокированные зависимости. Обновить обе языковые версии и трассировку. Закрывать задачу только по доказательству её результата; GitHub Closed само по себе не означает Ready MVP.

**Commit boundary:** логическая граница этой задачи; commit/push/deploy не разрешены данной карточкой и требуют действующей авторизации пользователя.

## EN

Retain a conversation until it reaches an unambiguous accounting outcome.

**Status:** Not started; the task awaits its own dependencies and entry gates.

**Dependencies:** `task-5.3`, `task-5.2`.

**Kind:** `implementation`.

### Change and contracts

Link messages, selected account, attachments, draft/proposal/operation and clarifications. A complete new transaction is created or linked to a substantiated match; missing fields trigger targeted questions. Clarification replies apply the current version once. Expose understandable processing/waiting/skipped/recorded states, correction and ordinary-accounting independence from AI. One shared chat; the account picker includes both members’ personal and household accounts. Confirmation belongs to the signed-in member, not a person named in text; personal plans/goals require their owner. Answers to one clarification are versioned.

### Change boundaries

- `backend/internal/chat/`
- `backend/internal/clarifications/`

These are planned paths. Shared contracts: `spec/001-want-keep-mvp/contracts.en.md`; architecture and commands: `constraints.en.md`. Change only the behavior owner and affected tests; an unresolved contract requires updated evidence and stops dependent implementation.

### Linked requirements

- **REQ-008:** Repeated imports, receipts and chat entries combine evidence of one transaction without double counting.
- **REQ-015:** Receipt submission requires a debit account; the photo/PDF stays linked to the processing result.
- **REQ-017:** Chat records an established transaction, clarifies missing data and explicitly explains skipped irrelevant documents.
- **REQ-018:** Every new or materially changed transaction receives AI review of its version.
- **REQ-019:** AI automates internal accounting through validated commands; uncertainty remains explicit.
- **REQ-021:** AI changes an approved budget, income forecast or goals only on an explicit decision by a member authorized for the change.
- **REQ-051:** AI is limited to $50/month and degrades to a waiting queue without stopping ordinary accounting.
- **REQ-064:** Both members see all financial data and edit transactions; only the owner edits personal goals and plan portions.
- **REQ-065:** Account ownership, external-account owner, record author and expense attribution are distinct dimensions.
- **REQ-067:** Expenses and receipt items have personal or joint attribution; joint shares default to 50/50 with line or purchase overrides.
- **REQ-071:** One shared chat retains message authors and checks the AI command initiator’s authority at execution.
- **REQ-072:** Competing edits, clarification answers and reversals check revision and current permissions while retaining both authors.

### Acceptance criteria

A criterion link establishes coverage; research or a partial task does not prove the entire product criterion. This task's exact outcome is specified in verification below.

#### AC-008

- **Given:** A RUB 300 expense was created from chat for a selected account.
- **When:** The matching receipt, bank transaction and duplicate bank delivery arrive.
- **Then:** Expense remains RUB 300 and all evidence is linked; separate equal-amount purchases are not merged merely by amount.
- **Level:** `integration`.

#### AC-015

- **Given:** A receipt photo or PDF is uploaded.
- **When:** The owner submits it without an account, then selects cash.
- **Then:** No entry is posted without selection; processing then uses the selected account. Both authenticated household members can read the attachment, including a partner’s receipt; unauthenticated and foreign-household access is denied.
- **Level:** `end-to-end`.

#### AC-017

- **Given:** A complete expense, an expense missing its amount and an obviously irrelevant image are submitted.
- **When:** AI processes the messages.
- **Then:** The first expense is recorded or matched; the second awaits clarification; the third receives an explicit skip reason; no amounts are invented.
- **Level:** `end-to-end`.

#### AC-018

- **Given:** Imported, manual and chat-created transactions exist.
- **When:** They are created and one financial record is then corrected.
- **Then:** Every current version is queued for review; repeated job delivery does not duplicate effects; a stale AI response cannot alter a newer version.
- **Level:** `integration`.

#### AC-019

- **Given:** AI proposes a source-conflicting amount and a match with several candidates.
- **When:** The application validates the proposals.
- **Then:** The conflicting change is rejected and ambiguity enters the clarification queue; categories and substantiated links may be applied automatically.
- **Level:** `integration`.

#### AC-021

- **Given:** An approved budget and a reallocation proposal exist.
- **When:** A new expense arrives and an authorized member later confirms the proposal.
- **Then:** The plan stays unchanged until confirmation; confirmation applies the displayed proposal version once; a stale proposal must be reconfirmed.
- **Level:** `integration`.

#### AC-051

- **Given:** OpenAI is unavailable or the allowed budget including in-flight reservations is exhausted.
- **When:** A new import, manual expense and AI request arrive.
- **Then:** Accounting and calculations remain available; AI status is waiting; no new paid calls exceed the allowed reservation; unknown cost is not silently released.
- **Level:** `integration`.

#### AC-078

- **Given:** A has a personal goal and plan line; the household has a joint line and both members’ transactions.
- **When:** B reads all data, edits A’s transaction and the joint plan, then attempts to change A’s personal goal/plan through API and AI.
- **Then:** Reads, transaction edits and joint changes succeed; A’s personal plan/goal are protected server-side. One authorized confirmation suffices and the other member is notified.
- **Level:** `end-to-end`.

#### AC-085

- **Given:** Both see the shared chat; A has a personal goal and B asks AI to change it as A.
- **When:** The model proposes an action and A later confirms a new proposal version addressed to A.
- **Then:** B’s message grants no authority of A; nothing changes before authorized confirmation. Audit records message author, approver and AI rationale; chat contains no secrets.
- **Level:** `integration`.

#### AC-086

- **Given:** A and B opened the same transaction or clarification revision.
- **When:** Both submit conflicting edits and replay one request.
- **Then:** One result applies; the other receives a conflict requiring refresh. Replay does not duplicate effects; reversal creates a checked new revision without erasing the other member’s later edit.
- **Level:** `integration`.

#### AC-093

- **Given:** A and B submit the same receipt for one account; the bank transaction arrives later.
- **When:** Parallel messages, a file replay and import are processed.
- **Then:** One financial effect and multiple authored evidence records result; an unproven match requires clarification and equal amounts from different accounts are not merged.
- **Level:** `integration`.

### Verification

```sh
make test-integration AREA=chat
```

Resubmitting/replying and bank updates during a conversation create no duplicate; every message has an explainable outcome.

The `make` commands are a future contract established by task-1.1; they do not exist yet. Live/paid/manual checks separately record access and actual outcomes. Research does not bypass missing-access blockers.

### Handoff to the next agent

Record changed contracts, commands/results, limitations, unresolved questions and unblocked dependencies. Update both languages and traceability. Close the task only with evidence of its outcome; GitHub Closed alone does not mean the MVP is Ready.

**Commit boundary:** this task's logical boundary; this card does not authorize commit/push/deploy, which require current user authorization.
