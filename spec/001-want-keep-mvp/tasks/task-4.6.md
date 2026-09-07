<!-- want-keep-task: task-4.6 -->
# task-4.6 — Реализовать коннектор EMCD / Implement EMCD connector

## RU

Автоматически читать кошелёк USDT, используемые Grow/криптокарты и архив P2P EMCD по D-34.

**Состояние:** Заблокировано зависимостями и проверкой SDD Ready; реализация не начата.

**Зависимости:** `task-0.6`, `task-3.3`, `task-2.4`, `task-2.5`.

**Тип:** `implementation`.

### Изменение и контракты

Покрыть только D-34 по подтверждённому evidence/emcd контракту; EMCD-B02–B04 / BLK-06 и SDD Ready закрываются до реализации. Майнинг, включая исторический, и другие неиспользуемые продукты не блокируют. Получить реальные структурированные fixtures: проектные emcd.samples.json не заменяют их. Не парсить UI-текст в финансовые проводки и не переносить mining auth/scopes на кошелёк. Разделить main aggregate, wallet, Grow, card owned/available/reserve без двойного счёта. Связать reward/capitalization/payout; текущая маркетинговая ставка не подменяет договор. Карты Plus/Light имеют разные условия; decline не списывает principal, fee требует подтверждения, authorization/clearing/refund/reversal связываются. USDT-пополнение и USD-карта — разные валюты; приблизительная USD-оценка покупки в EUR не settlement. P2P owner-side и точные legs/fee определяются контрактом, не порядком валют списка; ордер/wallet/bank не дублируют обмен. Проверить ID/revisions, повторы/поздние изменения, все страницы/coverage, hourly refresh, reauth и два независимых аккаунта. Повторное подключение связывается с тем же источником; lease/version запрещает применение старого результата после отключения. Только разрешённое чтение без PAN/CVV/переписки/секретов для AI.

### Границы изменений

- `backend/internal/integrations/emcd/`
- `collector/src/providers/emcd/`

Это планируемые пути. Общие контракты: `spec/001-want-keep-mvp/contracts.md`; архитектура и команды: `constraints.md`. Менять только владельца поведения и затронутые тесты; при незакрытом контракте обновить evidence и остановить зависимую реализацию.

### Связанные требования

- **REQ-006:** Перевод между счетами семьи, включая счета разных участников, меняет остатки без дохода или расхода по основной сумме.
- **REQ-007:** Обмен и P2P-конвертация собственных денег сохраняют обе валютные суммы, фактический курс и комиссии.
- **REQ-008:** Повторные импорты, чек и запись чата объединяют доказательства одной операции без повторного учёта.
- **REQ-031:** Кредитные карты показывают задолженность, собственные средства, лимит, минимальный платёж и дату по данным источника.
- **REQ-032:** Грейс-период опирается на условия конкретной карты и показывает сумму и срок сохранения льготы.
- **REQ-033:** Накопления показывают фактические начисления и прогноз по ставкам, срокам, капитализации и денежным потокам.
- **REQ-039:** Отсутствующие курсы и неподдерживаемые активы не превращаются в нулевые суммы или условный паритет USDT/USD.
- **REQ-040:** Каждый источник обновляется раз в час и по запросу с видимым временем успешного обновления.
- **REQ-041:** История сохраняет границы покрытия, курсоры, пробелы и статусы источника.
- **REQ-047:** Интеграция EMCD автоматически читает используемые криптокарты, Coinhold/Grow, кошелёк USDT и исторические P2P-ордера по D-34; майнинг никогда не использовался и вместе с другими неиспользуемыми продуктами отложен без блокировки.
- **REQ-048:** Интеграции и браузерный сборщик выполняют только разрешённые операции чтения.
- **REQ-061:** Повторные задания, перезапуски и параллельные изменения не создают двойных финансовых эффектов.
- **REQ-065:** Принадлежность счёта, владелец внешнего аккаунта, автор записи и принадлежность расхода являются отдельными признаками.
- **REQ-073:** Оба управляют подключениями; банковскую авторизацию выполняет владелец внешнего аккаунта без раскрытия секретов партнёру или AI.
- **REQ-076:** Семейная область проверяется для API, файлов, AI, фоновых задач и внешних ID независимо от присланных actor/owner.

### Критерии приёмки

Связь с критерием задаёт покрытие; исследование или частичная задача не доказывает весь критерий продукта. Точный результат этой задачи указан ниже в проверке.

#### AC-047

- **Дано:** Подключён разрешённый личный аккаунт EMCD с кошельком USDT, действующими Grow, существующими картами, включая заблокированную, и историей P2P. Майнинг не использовался ни сейчас, ни ранее.
- **Когда:** Запрошены счета, остатки, операции и необходимые условия продуктов.
- **Тогда:** По каждому включённому продукту подтверждены сопоставимые с источником данные и автоматическое чтение: сводки не дублируют дочерние остатки, начисление/капитализация/выплата не утраивают доход, отказ карты не расход по основной сумме, P2P связан с денежными сторонами без дубля. Неизвестные поля/история отмечены явно; отсутствие контракта выбранных продуктов блокирует адаптер. Майнинг и другие неиспользуемые продукты не требуются.
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

#### AC-070

- **Дано:** Банк передаёт баланс, но не условия грейса; ставка Earn имеет неизвестную базу начисления.
- **Когда:** Открываются прогнозы.
- **Тогда:** Баланс отображается; льгота и точный прогноз имеют причину недоступности; AI не извлекает гарантированную бизнес-логику из рекламной формулировки.
- **Уровень:** `contract+end-to-end`.

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
make test-contract PROVIDER=emcd && make test-integration AREA=emcd
```

Все продукты D-34 проходят реальные contract fixtures и отдельный live readback. Проверить EMCD-S01–S06 на структурированных данных; неизвестные поля остаются unknown. Доказать отдельный полный проход и повторы журналов wallet/Grow/card/P2P, lifecycle/fee и семейную identity. Mining и неиспользуемые продукты не требуются. Снятие task-0.6 не заменяет закрытие BLK-06 и Ready.

Команды make созданы основой task-1.1; финансовые provider/integration/E2E suites ещё не реализованы. Для документации используется make docs-check. Live/paid/manual проверки отдельно фиксируют доступ и фактический результат; наличие команды или UI-доступа не доказывает runtime.

### Передача следующему агенту

Записать изменённые контракты, команды и результаты, ограничения, незакрытые вопросы и разблокированные зависимости. Обновить обе языковые версии и трассировку. Закрывать задачу только по доказательству её результата; GitHub Closed само по себе не означает Ready MVP.

**Commit boundary:** логическая граница этой задачи; commit/push/deploy не разрешены данной карточкой и требуют действующей авторизации пользователя.

## EN

Automatically read the EMCD USDT wallet, used Grow/crypto cards and P2P archive under D-34.

**Status:** Blocked by dependencies and the SDD Ready gate; implementation has not started.

**Dependencies:** `task-0.6`, `task-3.3`, `task-2.4`, `task-2.5`.

**Kind:** `implementation`.

### Change and contracts

Implement only D-34 using the verified evidence/emcd contract; resolve EMCD-B02–B04 / BLK-06 and SDD Ready first. Mining, including historical mining, and other unused products do not block readiness. Obtain real structured fixtures: designed emcd.samples.json scenarios are not substitutes. Do not parse UI prose into financial postings or apply mining auth/scopes to wallets. Separate main aggregate, wallet, Grow and card owned/available/reserve without duplication. Link reward/capitalization/payout; current marketing rates cannot replace deposit terms. Plus/Light terms differ; decline does not post principal, fees need evidence, and authorization/clearing/refund/reversal must link. USDT funding and USD card are different currencies; approximate USD valuation of an EUR purchase is not settlement. Contract fields determine P2P owner-side and exact legs/fees, not list currency order; order/wallet/bank do not duplicate an exchange. Verify identity/revisions, replay/late changes, full pagination/coverage, hourly refresh, reauth and two independent accounts. Reconnection links the same source; lease/version rejects stale results after disconnect. Authorized reads only; no PAN/CVV/conversations/secrets for AI.

### Change boundaries

- `backend/internal/integrations/emcd/`
- `collector/src/providers/emcd/`

These are planned paths. Shared contracts: `spec/001-want-keep-mvp/contracts.en.md`; architecture and commands: `constraints.en.md`. Change only the behavior owner and affected tests; an unresolved contract requires updated evidence and stops dependent implementation.

### Linked requirements

- **REQ-006:** Transfers between household accounts, including different members’ accounts, change balances without principal income or expense.
- **REQ-007:** Exchange and P2P conversion of owned money preserve both currency amounts, the actual rate and fees.
- **REQ-008:** Repeated imports, receipts and chat entries combine evidence of one transaction without double counting.
- **REQ-031:** Credit cards show debt, own funds, credit limit, minimum payment and due date from source data.
- **REQ-032:** Grace-period tracking uses the specific card's terms and shows the amount and deadline needed to preserve the benefit.
- **REQ-033:** Savings show actual accruals and forecasts using rates, terms, compounding and cash flows.
- **REQ-039:** Missing rates and unsupported assets never become zero amounts or an assumed USDT/USD peg.
- **REQ-040:** Each source refreshes hourly and on demand with a visible last-success timestamp.
- **REQ-041:** History retains coverage boundaries, cursors, gaps and source status.
- **REQ-047:** The EMCD integration automatically reads used crypto cards, Coinhold/Grow, the USDT wallet and historical P2P orders under D-34; mining has never been used and is deferred with other unused products without blocking readiness.
- **REQ-048:** Integrations and the browser collector perform authorized read operations only.
- **REQ-061:** Repeated jobs, restarts and concurrent changes cannot create duplicate financial effects.
- **REQ-065:** Account ownership, external-account owner, record author and expense attribution are distinct dimensions.
- **REQ-073:** Both manage connections; the external-account owner performs bank authentication without exposing secrets to the partner or AI.
- **REQ-076:** Household scope is checked for APIs, files, AI, jobs and external IDs independently of supplied actor/owner fields.

### Acceptance criteria

A criterion link establishes coverage; research or a partial task does not prove the entire product criterion. This task's exact outcome is specified in verification below.

#### AC-047

- **Given:** An authorized personal EMCD account has a USDT wallet, existing Grow deposits, existing cards including a blocked card, and P2P history. Mining has never been used.
- **When:** Accounts, balances, transactions and required product terms are requested.
- **Then:** Each included product has source-matching data and verified automatic reading: aggregates do not duplicate child balances; accrual/capitalization/payout do not triple income; declined card principal is not an expense; P2P links to monetary legs without duplicates. Unknown fields/history are explicit; missing selected-product contracts block the adapter. Mining and other unused products are not required.
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

#### AC-070

- **Given:** A bank exposes balance but no grace terms; an Earn rate has an unknown accrual basis.
- **When:** Forecasts are opened.
- **Then:** Balance is shown; grace eligibility and exact forecasts explain unavailability; AI does not turn marketing wording into guaranteed business rules.
- **Level:** `contract+end-to-end`.

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
make test-contract PROVIDER=emcd && make test-integration AREA=emcd
```

All D-34 products pass real contract fixtures and separate live readback. Verify EMCD-S01–S06 on structured data; unknown fields stay unknown. Prove complete traversal and replay separately for wallet/Grow/card/P2P logs, lifecycle/fees and household identity. Mining and unused products are unnecessary. Completing task-0.6 does not replace BLK-06 closure and Ready.

The task-1.1 foundation provides make commands; financial provider/integration/E2E suites are not implemented yet. Use make docs-check for documentation. Live/paid/manual checks separately record access and outcomes; an existing command or UI access is not runtime proof.

### Handoff to the next agent

Record changed contracts, commands/results, limitations, unresolved questions and unblocked dependencies. Update both languages and traceability. Close the task only with evidence of its outcome; GitHub Closed alone does not mean the MVP is Ready.

**Commit boundary:** this task's logical boundary; this card does not authorize commit/push/deploy, which require current user authorization.
