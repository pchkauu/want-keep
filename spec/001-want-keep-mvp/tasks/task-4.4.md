<!-- want-keep-task: task-4.4 -->
# task-4.4 — Реализовать коннектор Bybit / Implement Bybit connector

## RU

Автоматически получать Funding USDT/USDC/ETH/BTC, используемый Easy Earn и P2P по D-36 без повторного финансового эффекта.

**Состояние:** Заблокировано зависимостями и проверкой SDD Ready; реализация не начата.

**Зависимости:** `task-0.4`, `task-3.3`, `task-2.4`, `task-2.5`.

**Тип:** `implementation`.

### Изменение и контракты

После закрытия BYBIT-B03/B04 и Ready реализовать официальный read-only API по evidence/bybit и evidence/bybit-api. API-доступ владельца к Funding, Flexible и P2P доказан; read POST list/detail включить в явный allowlist, финансовые create/pay/release/ads/transfer/stake/redeem запрещены. FUND не подменять UTA. Связывать журнал с деталями детерминированно: совпадение суммы/времени лишь кандидат; неоднозначность уточнять. Не классифицировать финансовое состояние по localization labels. Сохранять 18-значные native decimals и расхождения snapshot/ledger, не списывать остаточную сумму автоматически. Учесть yield result.list, P2P ret_code/items, seconds internal-deposit/Funding против ms фильтров/деталей, отсутствие hourly id и политику коллизий. Principal, accrual/distribution/Funding и fee учитывать один раз; нулевая выплата не создаёт credit. Сохранять исходные P2P fiat amount/quantity/quote независимо, связывать банк отдельно; empty fee не ноль. Следовать cursor на коротких страницах, сохранять durable checkpoints/revisions/coverage. Проверить RSA readOnly и изоляцию/ротацию/отзыв двух владельцев. Collector добавлять только при доказанном новом пробеле API. Fixed и остальные неиспользуемые продукты не блокируют; движения включённых кошельков сохраняются.

### Границы изменений

- `backend/internal/integrations/bybit/`
- `collector/src/providers/bybit/`

Это планируемые пути. Общие контракты: `spec/001-want-keep-mvp/contracts.md`; архитектура и команды: `constraints.md`. Менять только владельца поведения и затронутые тесты; при незакрытом контракте обновить evidence и остановить зависимую реализацию.

### Связанные требования

- **REQ-006:** Перевод между счетами семьи, включая счета разных участников, меняет остатки без дохода или расхода по основной сумме.
- **REQ-007:** Обмен и P2P-конвертация собственных денег сохраняют обе валютные суммы, фактический курс и комиссии.
- **REQ-008:** Повторные импорты, чек и запись чата объединяют доказательства одной операции без повторного учёта.
- **REQ-033:** Накопления показывают фактические начисления и прогноз по ставкам, срокам, капитализации и денежным потокам.
- **REQ-039:** Отсутствующие курсы и неподдерживаемые активы не превращаются в нулевые суммы или условный паритет USD/USDT/USDC.
- **REQ-040:** Каждый источник обновляется раз в час и по запросу с видимым временем успешного обновления.
- **REQ-041:** История сохраняет границы покрытия, курсоры, пробелы и статусы источника.
- **REQ-045:** Bybit автоматически читает Funding USDT/USDC/ETH/BTC, используемый Easy Earn и P2P; официальный API приоритетен. Остальные продукты отложены без блокировки по D-36.
- **REQ-048:** Интеграции и браузерный сборщик выполняют только разрешённые операции чтения.
- **REQ-061:** Повторные задания, перезапуски и параллельные изменения не создают двойных финансовых эффектов.
- **REQ-065:** Принадлежность счёта, владелец внешнего аккаунта, автор записи и принадлежность расхода являются отдельными признаками.
- **REQ-073:** Оба управляют подключениями; банковскую авторизацию выполняет владелец внешнего аккаунта без раскрытия секретов партнёру или AI.
- **REQ-076:** Семейная область проверяется для API, файлов, AI, фоновых задач и внешних ID независимо от присланных actor/owner.

### Критерии приёмки

Связь с критерием задаёт покрытие; исследование или частичная задача не доказывает весь критерий продукта. Точный результат этой задачи указан ниже в проверке.

#### AC-045

- **Дано:** Безопасно подключён read-only аккаунт Bybit с подтверждёнными контрактами Funding, используемого Easy Earn и P2P.
- **Когда:** Читаются остатки, история с выбранной даты, повторные страницы, Convert, начисление/выплата Earn и P2P; моделируется отказ прав выбранного продукта.
- **Тогда:** Точные native amounts, ID, статусы, комиссии и coverage сопоставимы с источником; Funding и детали не удваивают обмен, комиссию или доход. P2P связывает crypto/fiat с банком либо требует уточнения. Отказ доступа/неполная история явно блокируют соответствующее покрытие; отсутствие Spot/UTA trading, futures, карты, On-Chain/Advanced Earn и иных неиспользуемых продуктов не блокирует.
- **Уровень:** `contract+manual`.

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

#### AC-048

- **Дано:** Сборщик имеет сессию личного кабинета с более широкими внешними правами.
- **Когда:** Возникают запрос на платёж, неподтверждённый маршрут или MFA/CAPTCHA.
- **Тогда:** Платёж и неизвестный маршрут блокируются; MFA/CAPTCHA передаётся владельцу, источник приостанавливается; остальные источники продолжают работать.
- **Уровень:** `integration`.

#### AC-063

- **Дано:** Приходящая сторона USDT 100 уже импортирована; исходящая RUB 9 000 и комиссия ещё отсутствуют.
- **Когда:** Приходят поздняя сторона, исправление комиссии и повтор старой страницы.
- **Тогда:** Состояние ожидания связи сменяется проверенным обменом; доход/расход основной суммы не удваивается, устаревшая комиссия не восстанавливается.
- **Уровень:** `integration`.

#### AC-033

- **Дано:** Есть вклад или Earn с подтверждёнными условиями, пополнением и выводом.
- **Когда:** Рассчитывается доход за период и прогноз.
- **Тогда:** Факт отделён от прогноза и переоценки; смена ставки и капитализация учитываются по условиям; неизвестные условия блокируют точный прогноз.
- **Уровень:** `integration`.

#### AC-039

- **Дано:** В источнике есть неподдерживаемый USDC.E; для USDT/USD и USDC/USD отсутствуют курсы.
- **Когда:** Строится общая оценка.
- **Тогда:** Исходные данные сохранены, покрытие оценки обозначено неполным; нет скрытого нуля или автоматического курса 1:1. USDC.E не объединён с USDC по похожему символу.
- **Уровень:** `integration`.

#### AC-079

- **Дано:** A и B имеют разные аккаунты одного провайдера и общий счёт; B заносит покупку A со счёта B.
- **Когда:** Выполняются ввод, импорт обоих аккаунтов и повторное подключение того же внешнего аккаунта.
- **Тогда:** Разные аккаунты не сливаются; повторный источник не удваивает остатки. Плательщик, автор и получатель расхода сохраняются независимо. Неустановленное совпадение блокирует новый учёт до уточнения.
- **Уровень:** `integration`.

#### AC-087

- **Дано:** A владеет внешним аккаунтом, B инициирует повторную авторизацию или отключение.
- **Когда:** Запрашивается MFA; одновременно завершает работу старое задание синхронизации.
- **Тогда:** MFA адресован A; B видит статус, но не пароль/код/сессию. Отключение отзывает lease/version и запрещает применение старого результата; реальные платежи недоступны обоим.
- **Уровень:** `integration`.

#### AC-090

- **Дано:** В тестах созданы две изолированные семьи; запрос или задача подменяет householdId/actor/resourceId.
- **Когда:** Проверяются чтение файла, импорт, исправление, поиск AI и дедупликация.
- **Тогда:** Чужие объекты недоступны и не объединяются; сервер берёт principal из сессии или проверенного контекста задания. Отказ не раскрывает чужое содержимое.
- **Уровень:** `integration`.

### Проверка результата

```sh
make test-contract PROVIDER=bybit && make test-integration AREA=bybit
```

12 сценариев evidence/bybit.samples.json превращены в исполняемые contract fixtures с учётом BYBIT-E11–E18. Проверены replay/18-decimal/precision mismatch, два Convert движения и fee/yield один раз, нулевой yield, P2P rounding/bank linkage, missing hourly ID, cursor на короткой странице, partial/expired/forbidden, два владельца и stale jobs после отключения. Read-only live readback выбранных продуктов отделён от локальных тестов; исследовательский probe не заменяет работающий коннектор.

Команды make созданы основой task-1.1; финансовые provider/integration/E2E suites ещё не реализованы. Для документации используется make docs-check. Live/paid/manual проверки отдельно фиксируют доступ и фактический результат; наличие команды или UI-доступа не доказывает runtime.

### Передача следующему агенту

Записать изменённые контракты, команды и результаты, ограничения, незакрытые вопросы и разблокированные зависимости. Обновить обе языковые версии и трассировку. Закрывать задачу только по доказательству её результата; GitHub Closed само по себе не означает Ready MVP.

**Commit boundary:** логическая граница этой задачи; commit/push/deploy не разрешены данной карточкой и требуют действующей авторизации пользователя.

## EN

Automatically read Funding USDT/USDC/ETH/BTC, used Easy Earn and P2P under D-36 without duplicate financial effects.

**Status:** Blocked by dependencies and the SDD Ready gate; implementation has not started.

**Dependencies:** `task-0.4`, `task-3.3`, `task-2.4`, `task-2.5`.

**Kind:** `implementation`.

### Change and contracts

After BYBIT-B03/B04 closure and Ready, implement official read-only APIs from evidence/bybit and evidence/bybit-api. Owner access to Funding, Flexible and P2P is proven; explicitly allowlist read POST list/detail, forbid financial create/pay/release/ads/transfer/stake/redeem. Do not substitute UTA for FUND. Link ledger/details deterministically: amount/time equality is only a candidate; clarify ambiguity. Do not classify financial state from localization labels. Preserve 18-place native decimals and snapshot/ledger differences without automatically spending residuals. Handle yield result.list, P2P ret_code/items, seconds for internal-deposit/Funding versus ms filters/details, missing hourly id and collision policy. Account for principal, accrual/distribution/Funding and fee once; zero yield creates no credit. Preserve P2P native fiat amount/quantity/quote independently and match the bank separately; empty fee is not zero. Follow cursors on short pages; retain durable checkpoints/revisions/coverage. Verify RSA readOnly and two-owner isolation/rotation/revocation. Add collector code only for a newly demonstrated API gap. Fixed and other unused products do not block; included-wallet movements remain.

### Change boundaries

- `backend/internal/integrations/bybit/`
- `collector/src/providers/bybit/`

These are planned paths. Shared contracts: `spec/001-want-keep-mvp/contracts.en.md`; architecture and commands: `constraints.en.md`. Change only the behavior owner and affected tests; an unresolved contract requires updated evidence and stops dependent implementation.

### Linked requirements

- **REQ-006:** Transfers between household accounts, including different members’ accounts, change balances without principal income or expense.
- **REQ-007:** Exchange and P2P conversion of owned money preserve both currency amounts, the actual rate and fees.
- **REQ-008:** Repeated imports, receipts and chat entries combine evidence of one transaction without double counting.
- **REQ-033:** Savings show actual accruals and forecasts using rates, terms, compounding and cash flows.
- **REQ-039:** Missing rates and unsupported assets never become zero amounts or assumed USD/USDT/USDC parity.
- **REQ-040:** Each source refreshes hourly and on demand with a visible last-success timestamp.
- **REQ-041:** History retains coverage boundaries, cursors, gaps and source status.
- **REQ-045:** Bybit automatically reads Funding USDT/USDC/ETH/BTC, used Easy Earn and P2P; the official API is preferred. Other products are deferred without blocking under D-36.
- **REQ-048:** Integrations and the browser collector perform authorized read operations only.
- **REQ-061:** Repeated jobs, restarts and concurrent changes cannot create duplicate financial effects.
- **REQ-065:** Account ownership, external-account owner, record author and expense attribution are distinct dimensions.
- **REQ-073:** Both manage connections; the external-account owner performs bank authentication without exposing secrets to the partner or AI.
- **REQ-076:** Household scope is checked for APIs, files, AI, jobs and external IDs independently of supplied actor/owner fields.

### Acceptance criteria

A criterion link establishes coverage; research or a partial task does not prove the entire product criterion. This task's exact outcome is specified in verification below.

#### AC-045

- **Given:** A read-only Bybit account is securely connected with verified Funding, used Easy Earn and P2P contracts.
- **When:** Balances, history from the selected date, replayed pages, Convert, Earn accrual/payout and P2P are read; a selected-product permission failure is simulated.
- **Then:** Exact native amounts, IDs, statuses, fees and coverage match the source; Funding and details do not duplicate an exchange, fee or income. P2P links crypto/fiat with the bank or requires clarification. Access failure/incomplete history explicitly blocks the corresponding coverage; absent Spot/UTA trading, futures, card, On-Chain/Advanced Earn and other unused products do not block.
- **Level:** `contract+manual`.

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

#### AC-048

- **Given:** The collector has a personal-account session with broader provider permissions.
- **When:** A payment request, unapproved route or MFA/CAPTCHA appears.
- **Then:** Payments and unknown routes are blocked; MFA/CAPTCHA is handed to the owner and that source pauses; other sources continue.
- **Level:** `integration`.

#### AC-063

- **Given:** The incoming USDT 100 leg is imported; outgoing RUB 9,000 and fee are missing.
- **When:** The late leg, fee correction and replayed old page arrive.
- **Then:** Pending matching becomes a verified exchange; principal is not double-counted and the stale fee is not restored.
- **Level:** `integration`.

#### AC-033

- **Given:** A deposit or Earn product has confirmed terms, a top-up and a withdrawal.
- **When:** Period income and forecast are calculated.
- **Then:** Actual income is separate from forecast and revaluation; rate changes and compounding follow the terms; unknown terms prevent an exact forecast.
- **Level:** `integration`.

#### AC-039

- **Given:** A source contains unsupported USDC.E; USDT/USD and USDC/USD rates are unavailable.
- **When:** A total valuation is built.
- **Then:** Raw data is retained and valuation coverage is incomplete; no hidden zero or automatic 1:1 rate is used. USDC.E is not merged into USDC by symbol similarity.
- **Level:** `integration`.

#### AC-079

- **Given:** A and B have separate accounts at one provider and a joint account; B enters A’s purchase paid from B’s account.
- **When:** Entry, import of both accounts and reconnection of the same external account run.
- **Then:** Distinct accounts are not merged; a repeated source does not double balances. Payer, author and expense beneficiary remain independent. Unresolved source identity blocks new posting pending clarification.
- **Level:** `integration`.

#### AC-087

- **Given:** A owns the external account and B initiates reauthorization or disconnect.
- **When:** MFA is requested while an old sync job completes.
- **Then:** MFA is addressed to A; B sees status but no password/code/session. Disconnect revokes lease/version and prevents stale-result application; actual payments are unavailable to both.
- **Level:** `integration`.

#### AC-090

- **Given:** Tests contain two isolated households; a request or job forges householdId/actor/resourceId.
- **When:** File reads, import, correction, AI retrieval and deduplication are exercised.
- **Then:** Foreign objects are inaccessible and never merged; the server takes principal from the session or validated job context. Denial reveals no foreign content.
- **Level:** `integration`.

### Verification

```sh
make test-contract PROVIDER=bybit && make test-integration AREA=bybit
```

Turn the 12 evidence/bybit.samples.json scenarios into executable contract fixtures using BYBIT-E11–E18. Check replay/18-decimal/precision mismatch, both Convert legs and fee/yield once, zero yield, P2P rounding/bank linkage, missing hourly ID, short-page cursor, partial/expired/forbidden, two owners and stale jobs after disconnect. Separate selected-product read-only live readback from local tests; the research probe is not a running connector.

The task-1.1 foundation provides make commands; financial provider/integration/E2E suites are not implemented yet. Use make docs-check for documentation. Live/paid/manual checks separately record access and outcomes; an existing command or UI access is not runtime proof.

### Handoff to the next agent

Record changed contracts, commands/results, limitations, unresolved questions and unblocked dependencies. Update both languages and traceability. Close the task only with evidence of its outcome; GitHub Closed alone does not mean the MVP is Ready.

**Commit boundary:** this task's logical boundary; this card does not authorize commit/push/deploy, which require current user authorization.
