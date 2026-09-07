<!-- want-keep-task: task-0.6 -->
# task-0.6 — Проверить контракт чтения EMCD / Verify EMCD read contract

## RU

Зафиксировать доказательства и ограничения чтения используемых карт, Grow, кошелька USDT и архива P2P EMCD по D-34.

**Состояние:** Исследование EMCD UI завершено; task-0.10 закрепила D-34/D-39 namespaces и unknown/collision behavior. SDD Ready; structured fixtures и conformance остаются gate task-4.6.

**Зависимости:** нет.

**Тип:** `research`.

### Изменение и контракты

Объём D-34: кошелёк USDT, существующие Coinhold/Grow, карты Plus/Light и исторические P2P-ордера. Майнинг не использовался никогда; его история, другие продукты и новые mining API-ключи не нужны. Evidence/emcd.md и .en.md содержат датированные UI-наблюдения, официальные источники, продуктовую матрицу, поля/ограничения и EMCD-B01–B05. Опубликованный Mining Pool API 1.3.0 не доказывает чтение выбранных продуктов. Структурированных request/response этих продуктов нет; emcd.samples.json — проектные сценарии, не provider fixtures. Отделить агрегат и дочерние остатки, accrual/capitalization/payout Grow, declined principal и fee карты, точные стороны P2P и его округлённый UI. Сохранить вопросы identity/revisions/history/reauth/второго аккаунта и чтения без внешних мутаций. D-38/D-39 закрывают SDD-часть; исследование завершает evidence, а structured fixtures и conformance остаются entry/deployment gate task-4.6.

### Границы изменений

- `spec/001-want-keep-mvp/evidence/emcd.md`
- `spec/001-want-keep-mvp/evidence/emcd.en.md`
- `spec/001-want-keep-mvp/evidence/emcd.samples.json`

Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-031:** Кредитные карты показывают задолженность, собственные средства, лимит, минимальный платёж и дату по данным источника.
- **REQ-032:** Грейс-период опирается на условия конкретной карты и показывает сумму и срок сохранения льготы.
- **REQ-033:** Накопления показывают фактические начисления и прогноз по ставкам, срокам, капитализации и денежным потокам.
- **REQ-039:** Отсутствующие курсы и неподдерживаемые активы не превращаются в нулевые суммы или условный паритет USD/USDT/USDC.
- **REQ-041:** История сохраняет границы покрытия, курсоры, пробелы и статусы источника.
- **REQ-047:** Интеграция EMCD автоматически читает используемые криптокарты, Coinhold/Grow, кошелёк USDT и исторические P2P-ордера по D-34; майнинг никогда не использовался и вместе с другими неиспользуемыми продуктами отложен без блокировки.
- **REQ-048:** Интеграции и браузерный сборщик выполняют только разрешённые операции чтения.
- **REQ-065:** Принадлежность счёта, владелец внешнего аккаунта, автор записи и принадлежность расхода являются отдельными признаками.
- **REQ-073:** Оба управляют подключениями; банковскую авторизацию выполняет владелец внешнего аккаунта без раскрытия секретов партнёру или AI.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

#### AC-047

- **Дано:** Подключён разрешённый личный аккаунт EMCD с кошельком USDT, действующими Grow, существующими картами, включая заблокированную, и историей P2P. Майнинг не использовался ни сейчас, ни ранее.
- **Когда:** Запрошены счета, остатки, операции и необходимые условия продуктов.
- **Тогда:** По каждому включённому продукту подтверждены сопоставимые с источником данные и автоматическое чтение: сводки не дублируют дочерние остатки, начисление/капитализация/выплата не утраивают доход, отказ карты не расход по основной сумме, P2P связан с денежными сторонами без дубля. Неизвестные поля/история отмечены явно; отсутствие контракта выбранных продуктов блокирует адаптер. Майнинг и другие неиспользуемые продукты не требуются.
- **Уровень:** `contract+manual`.

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

### Проверка результата

```sh
make docs-check
```

RU/EN evidence разделяет подтверждённое UI-чтение, опубликованные условия, выводы и неизвестные API-поля; источники/ID/JSON совпадают между языками. Шесть синтетических сценариев сохраняют Decimal-инварианты и unknown. Публикация не содержит реальных сумм/ID/адресов/реквизитов/переписки/сессий. Непроверенные API/runtime/полная история/reauth явно перечислены; research Closed не означает BLK-06 Closed.

Команды make созданы основой task-1.1; финансовые provider/integration/E2E suites ещё не реализованы. Для документации используется make docs-check. Live/paid/manual проверки отдельно фиксируют доступ и фактический результат; наличие команды или UI-доступа не доказывает runtime.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Record reading evidence and limitations for used EMCD cards, Grow, USDT wallet and P2P archive under D-34.

**Status:** EMCD UI research is complete; task-0.10 fixed D-34/D-39 namespaces and unknown/collision behavior. The SDD is Ready; structured fixtures and conformance remain the task-4.6 gate.

**Dependencies:** none.

**Kind:** `research`.

### Change and contracts

D-34 scope: USDT wallet, existing Coinhold/Grow, Plus/Light cards and historical P2P orders. Mining has never been used; its history, other products and new mining API keys are unnecessary. Evidence/emcd.md and .en.md contain dated UI observations, official sources, product matrix, fields/limitations and EMCD-B01–B05. Published Mining Pool API 1.3.0 does not establish selected-product reading. Their structured request/response pairs are unavailable; emcd.samples.json contains designed scenarios, not provider fixtures. Separate aggregates/child balances, Grow accrual/capitalization/payout, declined card principal/fees and exact P2P legs/rounded UI. Retain identity/revisions/history/reauth/second-account and read-only questions. D-38/D-39 resolve the SDD portion; research completes evidence, while structured fixtures and conformance remain the task-4.6 entry/deployment gate.

### Change boundaries

- `spec/001-want-keep-mvp/evidence/emcd.md`
- `spec/001-want-keep-mvp/evidence/emcd.en.md`
- `spec/001-want-keep-mvp/evidence/emcd.samples.json`

Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-031:** Credit cards show debt, own funds, credit limit, minimum payment and due date from source data.
- **REQ-032:** Grace-period tracking uses the specific card's terms and shows the amount and deadline needed to preserve the benefit.
- **REQ-033:** Savings show actual accruals and forecasts using rates, terms, compounding and cash flows.
- **REQ-039:** Missing rates and unsupported assets never become zero amounts or assumed USD/USDT/USDC parity.
- **REQ-041:** History retains coverage boundaries, cursors, gaps and source status.
- **REQ-047:** The EMCD integration automatically reads used crypto cards, Coinhold/Grow, the USDT wallet and historical P2P orders under D-34; mining has never been used and is deferred with other unused products without blocking readiness.
- **REQ-048:** Integrations and the browser collector perform authorized read operations only.
- **REQ-065:** Account ownership, external-account owner, record author and expense attribution are distinct dimensions.
- **REQ-073:** Both manage connections; the external-account owner performs bank authentication without exposing secrets to the partner or AI.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

#### AC-047

- **Given:** An authorized personal EMCD account has a USDT wallet, existing Grow deposits, existing cards including a blocked card, and P2P history. Mining has never been used.
- **When:** Accounts, balances, transactions and required product terms are requested.
- **Then:** Each included product has source-matching data and verified automatic reading: aggregates do not duplicate child balances; accrual/capitalization/payout do not triple income; declined card principal is not an expense; P2P links to monetary legs without duplicates. Unknown fields/history are explicit; missing selected-product contracts block the adapter. Mining and other unused products are not required.
- **Level:** `contract+manual`.

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

### Verification

```sh
make docs-check
```

RU/EN evidence distinguishes observed UI reading, published terms, inferences and unknown API fields; sources/IDs/JSON align. Six synthetic scenarios preserve Decimal invariants and unknown values. Publication excludes real amounts/IDs/addresses/payment credentials/conversations/sessions. Unverified API/runtime/full history/reauth are explicit; research Closed does not mean BLK-06 Closed.

The task-1.1 foundation provides make commands; financial provider/integration/E2E suites are not implemented yet. Use make docs-check for documentation. Live/paid/manual checks separately record access and outcomes; an existing command or UI access is not runtime proof.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
