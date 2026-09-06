<!-- want-keep-task: task-0.8 -->
# task-0.8 — Измерить качество и стоимость OpenAI / Measure OpenAI quality and cost

## RU

Выбрать конфигурацию моделей по финансовым сценариям и лимиту $50.

**Состояние:** Исследование — не начато; live-доступ и платные прогоны требуют безопасно предоставленного доступа владельца.

**Зависимости:** нет.

**Тип:** `research`.

### Изменение и контракты

Сверить текущие модели, модальности, JSON/function-calling контракты, retention и цены по официальным источникам. Подготовить синтетический eval для переводов, чеков, неоднозначности и injection; измерить токены, ошибки, уточнения и стоимость на сотнях операций. Платные прогоны — только с предоставленным владельцем ключом и отдельным лимитом прогона; без него документировать метод и незакрытый quality gate. Начальные кандидаты Luna/Terra не означают доказанного качества.

### Границы изменений

- `spec/001-want-keep-mvp/evidence/openai.md`
- `spec/001-want-keep-mvp/evidence/openai.en.md`

Это планируемые пути. Общие контракты: `spec/001-want-keep-mvp/contracts.md`; архитектура и команды: `constraints.md`. Менять только владельца поведения и затронутые тесты; при незакрытом контракте обновить evidence и остановить зависимую реализацию.

### Связанные требования

- **REQ-018:** Каждая новая или содержательно изменённая операция получает AI-проверку своей версии.
- **REQ-019:** AI автоматизирует внутренний учёт через проверяемые команды; неопределённость остаётся явной.
- **REQ-020:** AI-инсайты по доходам и расходам ссылаются на проверяемые данные и отделяют прогноз от факта.
- **REQ-022:** Полные операции и чеки могут передаваться OpenAI, секреты доступа и лишние закрытые данные исключаются.
- **REQ-051:** AI ограничен бюджетом $50/месяц и деградирует в очередь ожидания без остановки обычного учёта.
- **REQ-055:** Веб-приложение предназначено для ноутбука macOS в Chrome и Arc; изменение окна и масштаба сохраняет доступность ежедневного учёта.
- **REQ-056:** Развёртывание укладывается в $40/месяц на сервер в DE/NL/BG; отдельные платные источники не используются.
- **REQ-060:** Текст чеков, банковских описаний и ответов AI не может расширять полномочия агента.
- **REQ-061:** Повторные задания, перезапуски и параллельные изменения не создают двойных финансовых эффектов.

### Критерии приёмки

Связь с критерием задаёт покрытие; исследование или частичная задача не доказывает весь критерий продукта. Точный результат этой задачи указан ниже в проверке.

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

#### AC-020

- **Дано:** В двух месяцах известны расходы по категориям; один источник устарел.
- **Когда:** AI формирует месячный инсайт.
- **Тогда:** Числа воспроизводятся отчётом; указаны период, связанные операции/срезы и неполнота; прогноз не представлен как полученный доход.
- **Уровень:** `integration`.

#### AC-022

- **Дано:** Операция и чек доступны владельцу; рядом в системе хранятся ключи источника.
- **Когда:** Формируется запрос AI и диагностическая запись.
- **Тогда:** В запросе только разрешённые данные операции/документа; ключи и сессии отсутствуют в запросе и логах; политика хранения OpenAI раскрыта.
- **Уровень:** `contract`.

#### AC-051

- **Дано:** OpenAI недоступен либо израсходован разрешённый бюджет с резервами текущих запросов.
- **Когда:** Поступают новый импорт, ручной расход и запрос AI.
- **Тогда:** Учёт и расчёты доступны; статус AI ожидает; новые платные запросы не запускаются сверх разрешённого резерва; неизвестная стоимость не освобождается молча.
- **Уровень:** `integration`.

#### AC-060

- **Дано:** В PDF или описании операции есть инструкция раскрыть ключ либо сделать перевод.
- **Когда:** Документ обрабатывается AI.
- **Тогда:** Инструкция считается данными; секреты и платёжные инструменты недоступны; недопустимая команда отклонена и не меняет учёт.
- **Уровень:** `integration`.

#### AC-069

- **Дано:** Три ответа AI: отказ, JSON вне схемы и корректный ответ на уже изменённую версию.
- **Когда:** Одновременно исчерпывается резерв месячного бюджета.
- **Тогда:** Ни один ответ не даёт неверного финансового эффекта; статусы различимы, платные повторы ограничены; данные и подтверждённые операции не теряются.
- **Уровень:** `integration`.

#### AC-076

- **Дано:** Подготовлен синтетический набор сотен операций со сложными переводами, чеками и эталонными ответами.
- **Когда:** Выполняются AI-eval, нагрузочная проверка и ручной дневной сценарий.
- **Тогда:** Отчёт показывает ошибки, уточнения, латентность, токены/стоимость и время пользователя; финансовые инварианты проходят, бюджет оценивается по измерению; непроверенное качество не объявлено доказанным.
- **Уровень:** `manual+integration`.

### Проверка результата

```sh
python3 spec/001-want-keep-mvp/tools/spec_tool.py check
```

Зафиксированы конфигурация/снимок цен, измеренный или явно непроверенный бюджет, результаты eval и критерии допуска к автоматическим командам.

Команды `make` — будущий контракт, создаваемый task-1.1; сейчас они не существуют. Live/paid/manual проверки отдельно фиксируют доступ и фактический результат. Исследования не обходят блокер отсутствующего доступа.

### Передача следующему агенту

Записать изменённые контракты, команды и результаты, ограничения, незакрытые вопросы и разблокированные зависимости. Обновить обе языковые версии и трассировку. Закрывать задачу только по доказательству её результата; GitHub Closed само по себе не означает Ready MVP.

**Commit boundary:** логическая граница этой задачи; commit/push/deploy не разрешены данной карточкой и требуют действующей авторизации пользователя.

## EN

Choose model configuration using financial cases and the $50 cap.

**Status:** Research — not started; live access and paid runs require securely supplied owner access.

**Dependencies:** none.

**Kind:** `research`.

### Change and contracts

Check current models, modalities, JSON/function-calling contracts, retention and prices against official sources. Prepare a synthetic evaluation for transfers, receipts, ambiguity and injection; measure tokens, errors, clarifications and cost at hundreds-of-transactions scale. Paid runs require an owner-supplied key and an explicit run cap; otherwise document the method and outstanding quality gate. Initial Luna/Terra candidates do not establish quality.

### Change boundaries

- `spec/001-want-keep-mvp/evidence/openai.md`
- `spec/001-want-keep-mvp/evidence/openai.en.md`

These are planned paths. Shared contracts: `spec/001-want-keep-mvp/contracts.en.md`; architecture and commands: `constraints.en.md`. Change only the behavior owner and affected tests; an unresolved contract requires updated evidence and stops dependent implementation.

### Linked requirements

- **REQ-018:** Every new or materially changed transaction receives AI review of its version.
- **REQ-019:** AI automates internal accounting through validated commands; uncertainty remains explicit.
- **REQ-020:** AI income/expense insights reference verifiable data and separate forecasts from facts.
- **REQ-022:** Full transactions and receipts may be sent to OpenAI; access secrets and unrelated private data are excluded.
- **REQ-051:** AI is limited to $50/month and degrades to a waiting queue without stopping ordinary accounting.
- **REQ-055:** The web app targets macOS laptops in Chrome and Arc; window resizing and zoom preserve daily accounting access.
- **REQ-056:** Deployment fits $40/month for a server in DE/NL/BG; no separately paid data sources are used.
- **REQ-060:** Receipt text, bank descriptions and AI outputs cannot expand agent authority.
- **REQ-061:** Repeated jobs, restarts and concurrent changes cannot create duplicate financial effects.

### Acceptance criteria

A criterion link establishes coverage; research or a partial task does not prove the entire product criterion. This task's exact outcome is specified in verification below.

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

#### AC-020

- **Given:** Category expenses are known for two months and one source is stale.
- **When:** AI generates a monthly insight.
- **Then:** Figures are reproducible from reports; period, linked transactions/aggregates and incompleteness are shown; forecast income is not presented as received.
- **Level:** `integration`.

#### AC-022

- **Given:** A transaction and receipt are available to the owner; source credentials are stored elsewhere.
- **When:** An AI request and diagnostic record are produced.
- **Then:** The request contains only permitted transaction/document data; keys and sessions appear in neither request nor logs; OpenAI retention policy is disclosed.
- **Level:** `contract`.

#### AC-051

- **Given:** OpenAI is unavailable or the allowed budget including in-flight reservations is exhausted.
- **When:** A new import, manual expense and AI request arrive.
- **Then:** Accounting and calculations remain available; AI status is waiting; no new paid calls exceed the allowed reservation; unknown cost is not silently released.
- **Level:** `integration`.

#### AC-060

- **Given:** A PDF or transaction description instructs the agent to reveal a key or transfer funds.
- **When:** AI processes the document.
- **Then:** The instruction is treated as data; secrets and payment tools are unavailable; an invalid command is rejected without changing accounting.
- **Level:** `integration`.

#### AC-069

- **Given:** Three AI outputs are a refusal, off-schema JSON and a valid response to an obsolete version.
- **When:** The monthly spend reservation is exhausted concurrently.
- **Then:** No response produces an incorrect financial effect; states are distinguishable, paid retries are bounded and data/confirmed transactions are retained.
- **Level:** `integration`.

#### AC-076

- **Given:** A synthetic hundreds-of-transactions set includes difficult transfers, receipts and reference answers.
- **When:** AI evaluation, load checks and a manual daily flow run.
- **Then:** The report shows errors, clarifications, latency, tokens/cost and user time; financial invariants pass and cost uses measurements; untested quality is not claimed as proven.
- **Level:** `manual+integration`.

### Verification

```sh
python3 spec/001-want-keep-mvp/tools/spec_tool.py check
```

Record configuration/pricing snapshot, measured or explicitly unverified cost, evaluation results and gates for automatic commands.

The `make` commands are a future contract established by task-1.1; they do not exist yet. Live/paid/manual checks separately record access and actual outcomes. Research does not bypass missing-access blockers.

### Handoff to the next agent

Record changed contracts, commands/results, limitations, unresolved questions and unblocked dependencies. Update both languages and traceability. Close the task only with evidence of its outcome; GitHub Closed alone does not mean the MVP is Ready.

**Commit boundary:** this task's logical boundary; this card does not authorize commit/push/deploy, which require current user authorization.
