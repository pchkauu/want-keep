<!-- want-keep-task: task-0.8 -->
# task-0.8 — Измерить качество и стоимость OpenAI / Measure OpenAI quality and cost

## RU

Выбрать конфигурацию моделей по финансовым сценариям и лимиту $50.

**Состояние:** Исследование OpenAI завершено: модель, strict schema, качество, стоимость и failure boundary выбраны. SDD Ready; gateway, авторизация и runtime budget остаются task-5.x/task-8.1.

**Зависимости:** нет.

**Тип:** `research`.

### Изменение и контракты

Владелец выбрал gpt-5.6-terra, reasoning.effort=xhigh для всех AI-задач. OAI-E01–E15, evidence/openai.md/en.md и results/prompts/eval/cost JSON фиксируют цены, retention, Standard foreground Responses, strict proposal, store=false, explicit cache без breakpoints, detail=high и предварительный input count. Финальный отдельный 206+3 eval: 206/206, visual 6/6, function 3/3, 0 критических ошибок; цена $0.445556. 301 OpenAI requests всего: $3.2926944 в общем лимите $7 с сохранением ранних неудач и оплаченного incomplete Luna. Калибровка на известных синтетических случаях, не holdout и не app acceptance. Dahl MiniMax/DeepSeek: отдельные ограниченные screen, high принят без подтверждения применения; unknown расходы остаются резервами, тариф пула не подтверждён. Альтернативы и автоматический downgrade в runtime не включать. Xhigh месячная смета $47.125, стресс $62.96875; при $50 задания ждут, обычный учёт работает. Допущения объёма/налоги/прочий расход сверить перед запуском. Изменение модели/effort/prompt/schema/pricing требует повторного допуска. Контракты task-5.1/5.2/5.3/5.5 разблокированы; task-0.10 завершила SDD gate. Серверные validators/права/runtime очередей/locale и полный AC остаются последующими задачами.

### Границы изменений

- `spec/001-want-keep-mvp/evidence/openai.md`
- `spec/001-want-keep-mvp/evidence/openai.en.md`
- `spec/001-want-keep-mvp/evidence/openai.eval.json`
- `spec/001-want-keep-mvp/evidence/openai.cost.json`
- `spec/001-want-keep-mvp/assets/openai-eval/`
- `spec/001-want-keep-mvp/tools/openai_eval.py`
- `spec/001-want-keep-mvp/tools/openai_cases.py`
- `spec/001-want-keep-mvp/tools/test_openai_eval.py`
- `spec/001-want-keep-mvp/evidence/openai.results.json`
- `spec/001-want-keep-mvp/evidence/openai.prompts.json`
- `spec/001-want-keep-mvp/tools/openai_report.py`
- `spec/001-want-keep-mvp/evidence/dahl.md`
- `spec/001-want-keep-mvp/evidence/dahl.en.md`
- `spec/001-want-keep-mvp/evidence/dahl.contract.json`
- `spec/001-want-keep-mvp/evidence/dahl.results.json`
- `spec/001-want-keep-mvp/tools/dahl_eval.py`
- `spec/001-want-keep-mvp/tools/dahl_report.py`
- `spec/001-want-keep-mvp/tools/test_dahl_eval.py`

Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

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

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

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
make docs-check
```

Официальный срез OAI-E01–E13, фактический отчёт eval, выбранный маршрут и ограничения, смета/неизмеренные части и критерии допуска; закрытие BLK-08 отдельно от полной приёмки приложения.

Основа task-1.1 уже предоставляет make. make docs-check проверяет документацию и исследовательский инструмент; production AI integration/E2E suites ещё не реализованы. Модельный eval и приёмка приложения фиксируются раздельно.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Choose model configuration using financial cases and the $50 cap.

**Status:** OpenAI research is complete: model, strict schema, quality, cost and failure boundary are selected. The SDD is Ready; gateway, authorization and runtime budget remain task-5.x/task-8.1 work.

**Dependencies:** none.

**Kind:** `research`.

### Change and contracts

The owner selected gpt-5.6-terra, reasoning.effort=xhigh for all AI tasks. OAI-E01–E15, evidence/openai.md/en.md and results/prompts/eval/cost JSON record pricing, retention, Standard foreground Responses, strict proposals, store=false, explicit cache without breakpoints, detail=high and input pre-counting. Final separate 206+3 evaluation: 206/206, visual 6/6, function 3/3, zero critical errors; cost USD 0.445556. All 301 OpenAI requests cost USD 3.2926944 within the shared USD 7 cap, retaining earlier failures and paid Luna incomplete output. Calibration on known synthetic cases, not holdout or app acceptance. Dahl MiniMax/DeepSeek: separate limited screens, requested high accepted without applied-setting confirmation; unknown costs remain reserved and pool pricing unverified. No runtime alternative or automatic downgrade. Xhigh monthly estimate USD 47.125, stress USD 62.96875; unaffordable work waits at USD 50 while ordinary accounting continues. Reconcile workload assumptions/taxes/other usage before launch. Model/effort/prompt/schema/pricing changes require requalification. The task-5.1/5.2/5.3/5.5 contracts are unblocked and task-0.10 completed the SDD gate. Server validation/permissions/runtime queues/locale and full acceptance remain subsequent work.

### Change boundaries

- `spec/001-want-keep-mvp/evidence/openai.md`
- `spec/001-want-keep-mvp/evidence/openai.en.md`
- `spec/001-want-keep-mvp/evidence/openai.eval.json`
- `spec/001-want-keep-mvp/evidence/openai.cost.json`
- `spec/001-want-keep-mvp/assets/openai-eval/`
- `spec/001-want-keep-mvp/tools/openai_eval.py`
- `spec/001-want-keep-mvp/tools/openai_cases.py`
- `spec/001-want-keep-mvp/tools/test_openai_eval.py`
- `spec/001-want-keep-mvp/evidence/openai.results.json`
- `spec/001-want-keep-mvp/evidence/openai.prompts.json`
- `spec/001-want-keep-mvp/tools/openai_report.py`
- `spec/001-want-keep-mvp/evidence/dahl.md`
- `spec/001-want-keep-mvp/evidence/dahl.en.md`
- `spec/001-want-keep-mvp/evidence/dahl.contract.json`
- `spec/001-want-keep-mvp/evidence/dahl.results.json`
- `spec/001-want-keep-mvp/tools/dahl_eval.py`
- `spec/001-want-keep-mvp/tools/dahl_report.py`
- `spec/001-want-keep-mvp/tools/test_dahl_eval.py`

Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

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

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

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
make docs-check
```

Official OAI-E01–E13 snapshot, actual evaluation report, selected route/limits, estimate/unmeasured portions and qualification gates; BLK-08 closure is distinct from full application acceptance.

The task-1.1 foundation already provides make. make docs-check validates documentation and research tooling; production AI integration/E2E suites are not implemented. Model evaluation and application acceptance are recorded separately.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
