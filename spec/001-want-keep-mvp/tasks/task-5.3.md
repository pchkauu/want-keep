<!-- want-keep-task: task-5.3 -->
# task-5.3 — Обрабатывать чеки и позиции / Process receipts and line items

## RU

Извлекать подтверждаемые данные чека и связывать их с одной оплатой.

**Состояние:** Заблокировано зависимостями и проверкой SDD Ready; реализация не начата.

**Зависимости:** `task-5.2`, `task-1.5`, `task-2.7`.

**Тип:** `implementation`.

### Изменение и контракты

Обработать фото и PDF через ограниченный pipeline с проверкой формата/размера/страниц до AI. Извлекать позиции, суммы, валюту, дату, продавца и скидки с provenance. Проверять итог арифметически, привязывать выбранный счёт, выявлять duplicate attachment и match candidates. Нерелевантное пропускать с причиной; неполное уточнять; не исполнять активный контент или URL документа.

### Границы изменений

- `backend/internal/receipts/`
- `backend/internal/attachments/`

Это планируемые пути. Общие контракты: `spec/001-want-keep-mvp/contracts.md`; архитектура и команды: `constraints.md`. Менять только владельца поведения и затронутые тесты; при незакрытом контракте обновить evidence и остановить зависимую реализацию.

### Связанные требования

- **REQ-008:** Повторные импорты, чек и запись чата объединяют доказательства одной операции без повторного учёта.
- **REQ-010:** Возврат уменьшает расходы исходного месяца покупки, сохраняя дату реального поступления денег.
- **REQ-015:** Для отправки чека требуется счёт списания; фото/PDF остаётся связанным с результатом обработки.
- **REQ-016:** Позиции чека распределяют одну оплаченную сумму по категориям без дублирования итога.
- **REQ-017:** Чат создаёт установленную операцию, уточняет недостающие данные и явно объясняет пропуск неподходящего документа.
- **REQ-022:** Полные операции и чеки могут передаваться OpenAI, секреты доступа и лишние закрытые данные исключаются.
- **REQ-050:** Файлы, ключи источников, сессии и финансовые журналы защищены от постороннего доступа.
- **REQ-060:** Текст чеков, банковских описаний и ответов AI не может расширять полномочия агента.
- **REQ-065:** Принадлежность счёта, владелец внешнего аккаунта, автор записи и принадлежность расхода являются отдельными признаками.
- **REQ-067:** Расходы и позиции чеков имеют личное или совместное назначение; общая доля по умолчанию 50/50 с исключениями статьи или покупки.
- **REQ-071:** Один общий чат сохраняет автора сообщения и проверяет полномочия инициатора AI-команды при исполнении.
- **REQ-072:** Конкурирующие изменения, ответы на уточнения и отмены проверяют версию и текущие права, сохраняя обоих авторов.

### Критерии приёмки

Связь с критерием задаёт покрытие; исследование или частичная задача не доказывает весь критерий продукта. Точный результат этой задачи указан ниже в проверке.

#### AC-015

- **Дано:** Загружено фото или PDF чека.
- **Когда:** Владелец отправляет его без счёта, затем выбирает наличный счёт.
- **Тогда:** Без выбора запись не проводится; после выбора обработка использует этот счёт. Оба аутентифицированных участника семьи читают вложение, включая чек партнёра; посторонний и участник другой семьи получают отказ.
- **Уровень:** `end-to-end`.

#### AC-016

- **Дано:** Оплачено RUB 900 за позиции RUB 600 и RUB 400 со скидкой RUB 100.
- **Когда:** AI извлекает и категоризирует позиции.
- **Тогда:** Сумма распределений точно RUB 900; скидка сохраняется; расхождение суммы направляется на уточнение, а не исправляется выдуманной позицией.
- **Уровень:** `integration`.

#### AC-017

- **Дано:** Присланы полный расход, расход без суммы и очевидно нерелевантное фото.
- **Когда:** AI обрабатывает сообщения.
- **Тогда:** Первый расход записан или связан с существующим; второй ждёт вопроса; третье сообщение получает явную причину пропуска; вымышленных сумм нет.
- **Уровень:** `end-to-end`.

#### AC-022

- **Дано:** Операция и чек доступны владельцу; рядом в системе хранятся ключи источника.
- **Когда:** Формируется запрос AI и диагностическая запись.
- **Тогда:** В запросе только разрешённые данные операции/документа; ключи и сессии отсутствуют в запросе и логах; политика хранения OpenAI раскрыта.
- **Уровень:** `contract`.

#### AC-050

- **Дано:** Существует приватный чек и активное подключение источника.
- **Когда:** Проверяются прямой URL файла, экспорт без сессии, логи и отзыв подключения.
- **Тогда:** Без авторизации доступ закрыт; секреты зашифрованы и не журналируются; отзыв подключения прекращает дальнейший сбор.
- **Уровень:** `integration`.

#### AC-060

- **Дано:** В PDF или описании операции есть инструкция раскрыть ключ либо сделать перевод.
- **Когда:** Документ обрабатывается AI.
- **Тогда:** Инструкция считается данными; секреты и платёжные инструменты недоступны; недопустимая команда отклонена и не меняет учёт.
- **Уровень:** `integration`.

#### AC-068

- **Дано:** Загружаются повреждённый PDF, неверно обозначенный тип, чрезмерный файл и чек с вредоносным текстом.
- **Когда:** Срабатывают проверка файла и обработка.
- **Тогда:** Файл с ошибкой не проводится; нет выполнения вложенного кода, произвольного скачивания URL или публичного доступа; понятная причина/уточнение видна в чате.
- **Уровень:** `integration`.

#### AC-081

- **Дано:** Чек RUB 1000 содержит общие продукты 600 и личные покупки A 100 и B 300.
- **Когда:** Чек заносит любой участник; AI применяет правила или уточняет неизвестное назначение.
- **Тогда:** Факт семьи 1000, A 400, B 600; доли суммируются точно. Исключение покупки приоритетнее статьи, затем 50/50; неоднозначная трата сохранена без вымышленной принадлежности.
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
make test-integration AREA=receipts && make eval-ai SUITE=receipts
```

Валидные, частичные, повторные и вредоносные документы проходят требуемые состояния; итог распределения равен оплате.

Команды `make` — будущий контракт, создаваемый task-1.1; сейчас они не существуют. Live/paid/manual проверки отдельно фиксируют доступ и фактический результат. Исследования не обходят блокер отсутствующего доступа.

### Передача следующему агенту

Записать изменённые контракты, команды и результаты, ограничения, незакрытые вопросы и разблокированные зависимости. Обновить обе языковые версии и трассировку. Закрывать задачу только по доказательству её результата; GitHub Closed само по себе не означает Ready MVP.

**Commit boundary:** логическая граница этой задачи; commit/push/deploy не разрешены данной карточкой и требуют действующей авторизации пользователя.

## EN

Extract verifiable receipt data and link it to one payment.

**Status:** Blocked by dependencies and the SDD Ready gate; implementation has not started.

**Dependencies:** `task-5.2`, `task-1.5`, `task-2.7`.

**Kind:** `implementation`.

### Change and contracts

Process photos and PDFs through a bounded pipeline validating format/size/pages before AI. Extract items, amounts, currency, date, merchant and discounts with provenance. Validate totals arithmetically, bind the selected account and detect duplicate attachments/match candidates. Skip irrelevant content with a reason, clarify incomplete data and never execute document content or URLs.

### Change boundaries

- `backend/internal/receipts/`
- `backend/internal/attachments/`

These are planned paths. Shared contracts: `spec/001-want-keep-mvp/contracts.en.md`; architecture and commands: `constraints.en.md`. Change only the behavior owner and affected tests; an unresolved contract requires updated evidence and stops dependent implementation.

### Linked requirements

- **REQ-008:** Repeated imports, receipts and chat entries combine evidence of one transaction without double counting.
- **REQ-010:** A refund reduces expenses in the purchase month while preserving the actual cash receipt date.
- **REQ-015:** Receipt submission requires a debit account; the photo/PDF stays linked to the processing result.
- **REQ-016:** Receipt items allocate one paid amount across categories without duplicating the total.
- **REQ-017:** Chat records an established transaction, clarifies missing data and explicitly explains skipped irrelevant documents.
- **REQ-022:** Full transactions and receipts may be sent to OpenAI; access secrets and unrelated private data are excluded.
- **REQ-050:** Files, source keys, sessions and financial records are protected against unauthorized access.
- **REQ-060:** Receipt text, bank descriptions and AI outputs cannot expand agent authority.
- **REQ-065:** Account ownership, external-account owner, record author and expense attribution are distinct dimensions.
- **REQ-067:** Expenses and receipt items have personal or joint attribution; joint shares default to 50/50 with line or purchase overrides.
- **REQ-071:** One shared chat retains message authors and checks the AI command initiator’s authority at execution.
- **REQ-072:** Competing edits, clarification answers and reversals check revision and current permissions while retaining both authors.

### Acceptance criteria

A criterion link establishes coverage; research or a partial task does not prove the entire product criterion. This task's exact outcome is specified in verification below.

#### AC-015

- **Given:** A receipt photo or PDF is uploaded.
- **When:** The owner submits it without an account, then selects cash.
- **Then:** No entry is posted without selection; processing then uses the selected account. Both authenticated household members can read the attachment, including a partner’s receipt; unauthenticated and foreign-household access is denied.
- **Level:** `end-to-end`.

#### AC-016

- **Given:** RUB 900 was paid for RUB 600 and RUB 400 items with a RUB 100 discount.
- **When:** AI extracts and categorizes the items.
- **Then:** Allocations total exactly RUB 900 and retain the discount; a mismatch is clarified rather than patched with an invented item.
- **Level:** `integration`.

#### AC-017

- **Given:** A complete expense, an expense missing its amount and an obviously irrelevant image are submitted.
- **When:** AI processes the messages.
- **Then:** The first expense is recorded or matched; the second awaits clarification; the third receives an explicit skip reason; no amounts are invented.
- **Level:** `end-to-end`.

#### AC-022

- **Given:** A transaction and receipt are available to the owner; source credentials are stored elsewhere.
- **When:** An AI request and diagnostic record are produced.
- **Then:** The request contains only permitted transaction/document data; keys and sessions appear in neither request nor logs; OpenAI retention policy is disclosed.
- **Level:** `contract`.

#### AC-050

- **Given:** A private receipt and an active source connection exist.
- **When:** A direct file URL, unauthenticated export, logs and disconnection are checked.
- **Then:** Unauthenticated access fails; secrets are encrypted and not logged; disconnecting stops further collection.
- **Level:** `integration`.

#### AC-060

- **Given:** A PDF or transaction description instructs the agent to reveal a key or transfer funds.
- **When:** AI processes the document.
- **Then:** The instruction is treated as data; secrets and payment tools are unavailable; an invalid command is rejected without changing accounting.
- **Level:** `integration`.

#### AC-068

- **Given:** A corrupt PDF, mislabeled type, oversized file and prompt-injected receipt are uploaded.
- **When:** File validation and processing run.
- **Then:** Invalid files do not post; embedded code, arbitrary URL fetching and public access are unavailable; chat shows an understandable reason or clarification.
- **Level:** `integration`.

#### AC-081

- **Given:** A RUB 1,000 receipt contains joint groceries of 600 and personal purchases of A 100 and B 300.
- **When:** Either member enters the receipt; AI applies rules or clarifies unknown attribution.
- **Then:** Household actual is 1,000, A 400, B 600; shares sum exactly. Purchase override takes precedence over plan line, then 50/50; ambiguous spending persists without invented attribution.
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
make test-integration AREA=receipts && make eval-ai SUITE=receipts
```

Valid, partial, repeated and malicious documents follow required states; allocation total equals payment.

The `make` commands are a future contract established by task-1.1; they do not exist yet. Live/paid/manual checks separately record access and actual outcomes. Research does not bypass missing-access blockers.

### Handoff to the next agent

Record changed contracts, commands/results, limitations, unresolved questions and unblocked dependencies. Update both languages and traceability. Close the task only with evidence of its outcome; GitHub Closed alone does not mean the MVP is Ready.

**Commit boundary:** this task's logical boundary; this card does not authorize commit/push/deploy, which require current user authorization.
