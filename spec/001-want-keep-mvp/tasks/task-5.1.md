<!-- want-keep-task: task-5.1 -->
# task-5.1 — Создать OpenAI gateway и контроль расходов / Create OpenAI gateway and spend control

## RU

Вызывать OpenAI через ограниченный бюджетом доменный контракт.

**Состояние:** Реализованы Responses API gateway, точный семейный бюджет, долговечные AI-попытки, атомарные provider/job outcomes, возобновление budget-waiting заданий и least-privilege операторская сверка. Применение предложений, чеки, инсайты, UI и production остаются профильным задачам.

**Зависимости:** `task-0.8`, `task-3.1`, `task-1.5`.

**Тип:** `implementation`.

### Изменение и контракты

Использовать официальный Go SDK Responses API через gateway; модели/цены и token limits закрепить по исследованию. Хранить разговор в Want Keep, отправлять store=false, учитывать фактическую retention OpenAI. До запроса атомарно резервировать верхнюю оценку стоимости, после ответа сверять usage; неизвестную оплату держать зарезервированной до сверки. Ограничить параллелизм, повторы и месячный период бюджета. Применить evidence/openai.md: mode=explicit без cache breakpoints, store=false/foreground, Standard short-context, заранее подсчитать полный input, ограничить output с reasoning и резервировать cache-write максимум. Usage.input_tokens_details.cached_tokens/cache_write_tokens сверять раздельно; отсутствие write count — консервативный charge, не ноль: он учитывается в бюджете, не блокирует следующие вызовы и доступен для operator reconciliation. SDK retries=0, family concurrency≤2, модель/цены/права вне пользовательского текста. Невалидный usage/счёт выше резерва/неизвестный outcome останавливают новые вызовы до сверки; без автоматического повышения $50. Исследование task-0.8 завершено: использовать gpt-5.6-terra xhigh и финальную strict-схему из evidence/openai.prompts.json; Luna/Sol/MiniMax/DeepSeek автоматически не подключать. Финальный xhigh eval 206/206 не заменяет runtime/locale проверки. reasoning.effort=xhigh; никаких автоматических downgrade при лимите $50. Статус записи формирует приложение, не объяснение модели. Доверенный resumer возвращает ожидающие budget jobs в ready только после освобождения лимита/слота или начала нового UTC-месяца; транзакционная проверка резерва выполняется повторно. После одной повторной попытки следующий retryable отказ завершает job постоянной ошибкой. После Input Tokens ответа повторно проверять global barrier, семейный слот и бюджет до резерва; нулевой/отсутствующий count запрещает generation. Provider ID, возвращённая модель и непротиворечивый usage обязательны для оплачиваемого результата, иначе outcome неизвестен. Любой известный provider outcome и job transition сохранять атомарно. Сверка известного terminal outcome сверх резерва сохраняет outcome/output/validation и не возобновляет job. Появление рабочего gateway при рестарте возвращает gateway_unavailable jobs в ready. Исторический eval не покрывает runtime-добавления USDC/case envelope, поэтому production остаётся fail-closed до отдельной live-квалификации. В provider payload использовать pseudonymous qualified-case без внутренних household/member/account/operation ID. Maintenance role исполняет только ограниченную сверку истёкшего/unresolved вызова либо terminal over-reservation и не читает таблицы AI/jobs напрямую. После generation структурированные 429 rate_limit_exceeded и insufficient_quota подтверждают отсутствие списания: только rate_limit_exceeded допускает одну новую оплачиваемую попытку с bounded Retry-After, quota завершает job без повтора, неизвестный 429 даёт unknown. Любой другой outcome после external-started без доказательства отсутствия списания, включая 400/401/403/408/5xx и неоднозначный transport, даёт unknown с сохранением резерва. Противоречивые provider usage counters сохранять отдельно как evidence без влияния на расчёт. Input usage допустим в диапазоне count…count+32, output — 1…route cap. Input counting, pre-generation outage и восстановленный резерв без external-started не расходуют лимит generation-попыток. Повторяющийся gateway-resumer обрабатывает все ограниченные пакеты после восстановления; неверная конфигурация OpenAI не останавливает sync/outbox/scheduler. Одинаковый предел длины точной стоимости действует в application и maintenance-сверке. Повторы до generation сохраняют backoff до пяти минут, не расходуют общий attempt и не продлевают исходный 24-часовой deadline; 401/403 при Input Tokens остаются восстановимыми. Сверка и резервирование сериализуются под блокировкой семьи.

### Границы изменений

- `backend/internal/ai/`
- `backend/internal/gateways/openai/`
- `backend/internal/storage/ai_gateway.go`
- `backend/migrations/016_ai_gateway_budget.sql`
- `backend/cmd/worker/`
- `backend/cmd/ai-reconcile/`

Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-018:** Каждая новая или содержательно изменённая операция получает AI-проверку своей версии.
- **REQ-019:** AI автоматизирует внутренний учёт через проверяемые команды; неопределённость остаётся явной.
- **REQ-020:** AI-инсайты по доходам и расходам ссылаются на проверяемые данные и отделяют прогноз от факта.
- **REQ-022:** Полные операции и чеки могут передаваться OpenAI, секреты доступа и лишние закрытые данные исключаются.
- **REQ-051:** AI ограничен бюджетом $50/месяц и деградирует в очередь ожидания без остановки обычного учёта.
- **REQ-055:** Веб-приложение предназначено для ноутбука macOS в Chrome и Arc; изменение окна и масштаба сохраняет доступность ежедневного учёта.
- **REQ-056:** Развёртывание укладывается в $40/месяц на сервер в DE/NL/BG; отдельные платные источники не используются.
- **REQ-058:** Операционные статусы показывают ошибки импорта, AI, курсов, резервирования и расходы без утечки финансового содержимого.
- **REQ-059:** Денежные расчёты используют точную арифметику и явные правила округления на границах.
- **REQ-060:** Текст чеков, банковских описаний и ответов AI не может расширять полномочия агента.
- **REQ-061:** Повторные задания, перезапуски и параллельные изменения не создают двойных финансовых эффектов.
- **REQ-071:** Один общий чат сохраняет автора сообщения и проверяет полномочия инициатора AI-команды при исполнении.
- **REQ-076:** Семейная область проверяется для API, файлов, AI, фоновых задач и внешних ID независимо от присланных actor/owner.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

#### AC-018

- **Дано:** Есть импортированная, ручная и созданная чатом операции.
- **Когда:** Они создаются, а затем одна финансовая запись исправляется.
- **Тогда:** Каждая актуальная версия поставлена на проверку; повторная доставка задачи не дублирует эффект; устаревший ответ AI не меняет новую версию.
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

#### AC-058

- **Дано:** Сломан один коннектор, задержан AI и устарела копия.
- **Когда:** Открывается состояние системы и читаются диагностические логи.
- **Тогда:** Видны отдельные проблемы и действия восстановления; логи содержат идентификаторы/коды, а не чеки, ключи или тексты финансовых сообщений.
- **Уровень:** `integration`.

#### AC-059

- **Дано:** Есть дробные BTC, USDT, процентное начисление и распределение чека.
- **Когда:** Данные проходят API, базу и повторный расчёт.
- **Тогда:** Исходная точность не теряется; JSON-суммы не проходят binary float; распределения сходятся точно, округление отображения не меняет журнал.
- **Уровень:** `unit+contract`.

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

#### AC-085

- **Дано:** Оба видят общий чат; A имеет личную цель, B просит AI изменить её от имени A.
- **Когда:** Модель предлагает действие, а затем A подтверждает новую адресованную ему версию предложения.
- **Тогда:** Сообщение B не выдаёт полномочия A; до разрешённого подтверждения изменения нет. Аудит хранит автора сообщения, подтвердившего и AI-основание; секретов в чате нет.
- **Уровень:** `integration`.

#### AC-090

- **Дано:** В тестах созданы две изолированные семьи; запрос или задача подменяет householdId/actor/resourceId.
- **Когда:** Проверяются чтение файла, импорт, исправление, поиск AI и дедупликация.
- **Тогда:** Чужие объекты недоступны и не объединяются; сервер берёт principal из сессии или проверенного контекста задания. Отказ не раскрывает чужое содержимое.
- **Уровень:** `integration`.

### Проверка результата

```sh
make test-integration AREA=ai-budget
```

Конкурентные запросы, неизвестный outcome, смена месяца, недоступность и неверная usage не позволяют незаметно превысить разрешённый бюджет.

Зависимости task-0.8, task-3.1 и task-1.5 включены в базу. Реализацию и точные границы подтверждает evidence/task-5.1-openai-gateway.md; обязательны make check, AI-budget/jobs/audit/ledger/storage/privacy integration и race suites. Live OpenAI и production не проверялись.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Call OpenAI through a budget-controlled domain contract.

**Status:** The Responses API gateway, exact household budget, durable AI attempts, atomic provider/job outcomes, budget-waiting resumption and least-privilege operator reconciliation are implemented. Proposal application, receipts, insights, UI and production remain with their owning tasks.

**Dependencies:** `task-0.8`, `task-3.1`, `task-1.5`.

**Kind:** `implementation`.

### Change and contracts

Use the official Go Responses API SDK behind a gateway; pin models/prices and token limits from research. Store conversations in Want Keep, request store=false and account for actual OpenAI retention. Atomically reserve a conservative cost ceiling before calls, reconcile usage afterward and keep unknown charges reserved until reconciliation. Bound concurrency, retries and the monthly spending period. Apply evidence/openai.en.md: explicit mode with no cache breakpoints, store=false/foreground, Standard short context, pre-count full input, bound output including reasoning and reserve cache-write ceilings. Reconcile usage.input_tokens_details.cached_tokens/cache_write_tokens separately; absent write count retains a conservative charge, not zero: it counts against the budget, does not block subsequent calls and remains operator-reconcilable. SDK retries=0, family concurrency≤2; models/prices/authority are outside user text. Invalid usage, over-reservation cost and unknown outcomes stop new calls pending reconciliation; never raise USD 50 automatically. task-0.8 research is complete: use gpt-5.6-terra xhigh and the final strict schema in evidence/openai.prompts.json; do not automatically enable Luna/Sol/MiniMax/DeepSeek. The final xhigh evaluation 206/206 does not replace runtime/locale checks. reasoning.effort=xhigh; no automatic downgrade at the USD 50 cap. The application supplies persistence status, not the model explanation. A trusted resumer returns budget-waiting jobs to ready only after budget/slot capacity becomes available or a new UTC month starts; the transactional reservation check runs again. After one retry, the next retryable rejection fails the job permanently. After the Input Tokens response, recheck the global barrier, household slot and budget before reservation; a missing/zero count prevents generation. Provider ID, returned model and consistent usage are mandatory for a chargeable result, otherwise the outcome is unknown. Commit every known provider outcome atomically with its job transition. Reconciliation of a known terminal outcome above reservation preserves its outcome/output/validation and never resumes its job. A working gateway appearing on restart returns gateway_unavailable jobs to ready. The historical evaluation does not cover runtime USDC/case-envelope adaptations, so production remains fail-closed until a separate live qualification. Use a pseudonymous qualified-case provider payload without internal household/member/account/operation IDs. The maintenance role executes only constrained reconciliation for an expired/unresolved call or terminal over-reservation and cannot read AI/job tables directly. After generation, structured 429 rate_limit_exceeded and insufficient_quota prove no charge: only rate_limit_exceeded permits one new billable attempt after a bounded Retry-After, quota terminalizes without retry, and an unknown 429 becomes unknown. Any other outcome after external-started without proof of no charge, including 400/401/403/408/5xx and ambiguous transport, becomes unknown while retaining the reservation. Persist contradictory provider usage counters separately as evidence without accounting effects. Input usage must fall within count through count+32 and output within 1 through the route cap. Input counting, pre-generation outage and a recovered reservation without external-started do not consume the generation-attempt allowance. A recurring gateway resumer drains every bounded batch after recovery; invalid OpenAI configuration does not stop sync, outbox or the scheduler. The exact-cost length bound is identical in application and maintenance reconciliation. Pre-generation retries persist backoff up to five minutes, do not consume the generic attempt count and never extend the original 24-hour deadline; Input Tokens 401/403 remain recoverable. Reconciliation and reservation serialize under the household lock.

### Change boundaries

- `backend/internal/ai/`
- `backend/internal/gateways/openai/`
- `backend/internal/storage/ai_gateway.go`
- `backend/migrations/016_ai_gateway_budget.sql`
- `backend/cmd/worker/`
- `backend/cmd/ai-reconcile/`

Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-018:** Every new or materially changed transaction receives AI review of its version.
- **REQ-019:** AI automates internal accounting through validated commands; uncertainty remains explicit.
- **REQ-020:** AI income/expense insights reference verifiable data and separate forecasts from facts.
- **REQ-022:** Full transactions and receipts may be sent to OpenAI; access secrets and unrelated private data are excluded.
- **REQ-051:** AI is limited to $50/month and degrades to a waiting queue without stopping ordinary accounting.
- **REQ-055:** The web app targets macOS laptops in Chrome and Arc; window resizing and zoom preserve daily accounting access.
- **REQ-056:** Deployment fits $40/month for a server in DE/NL/BG; no separately paid data sources are used.
- **REQ-058:** Operational status exposes import, AI, FX, backup failures and spend without leaking financial content.
- **REQ-059:** Money calculations use exact arithmetic and explicit boundary rounding rules.
- **REQ-060:** Receipt text, bank descriptions and AI outputs cannot expand agent authority.
- **REQ-061:** Repeated jobs, restarts and concurrent changes cannot create duplicate financial effects.
- **REQ-071:** One shared chat retains message authors and checks the AI command initiator’s authority at execution.
- **REQ-076:** Household scope is checked for APIs, files, AI, jobs and external IDs independently of supplied actor/owner fields.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

#### AC-018

- **Given:** Imported, manual and chat-created transactions exist.
- **When:** They are created and one financial record is then corrected.
- **Then:** Every current version is queued for review; repeated job delivery does not duplicate effects; a stale AI response cannot alter a newer version.
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

#### AC-058

- **Given:** A connector fails, AI is delayed and a backup is stale.
- **When:** System health and diagnostic logs are inspected.
- **Then:** Separate failures and recovery actions are visible; logs contain identifiers/codes, not receipts, keys or financial message text.
- **Level:** `integration`.

#### AC-059

- **Given:** Fractional BTC, USDT, interest accrual and receipt allocation exist.
- **When:** Data traverses API, storage and recalculation.
- **Then:** Original precision survives; JSON money never traverses binary floats; allocations reconcile exactly and display rounding does not alter the ledger.
- **Level:** `unit+contract`.

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

#### AC-085

- **Given:** Both see the shared chat; A has a personal goal and B asks AI to change it as A.
- **When:** The model proposes an action and A later confirms a new proposal version addressed to A.
- **Then:** B’s message grants no authority of A; nothing changes before authorized confirmation. Audit records message author, approver and AI rationale; chat contains no secrets.
- **Level:** `integration`.

#### AC-090

- **Given:** Tests contain two isolated households; a request or job forges householdId/actor/resourceId.
- **When:** File reads, import, correction, AI retrieval and deduplication are exercised.
- **Then:** Foreign objects are inaccessible and never merged; the server takes principal from the session or validated job context. Denial reveals no foreign content.
- **Level:** `integration`.

### Verification

```sh
make test-integration AREA=ai-budget
```

Concurrent calls, unknown outcomes, month rollover, outage and invalid usage cannot silently exceed the authorized budget.

Dependencies task-0.8, task-3.1 and task-1.5 are included in the base. Evidence and exact boundaries are in evidence/task-5.1-openai-gateway.en.md; make check plus AI-budget/jobs/audit/ledger/storage/privacy integration and race suites are required. Live OpenAI and production were not tested.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
