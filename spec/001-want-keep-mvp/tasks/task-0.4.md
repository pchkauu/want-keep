<!-- want-keep-task: task-0.4 -->
# task-0.4 — Проверить контракт чтения Bybit / Verify Bybit read contract

## RU

Зафиксировать проверяемый read-контракт Bybit для Funding USDT/USDC/ETH/BTC, Easy Earn и P2P и ограничения доступа.

**Состояние:** Исследование завершено 2026-09-07: [RU evidence](https://github.com/pchkauu/want-keep/blob/docs/want-keep-mvp-sdd/spec/001-want-keep-mvp/evidence/bybit-api.md). RSA readOnly API успешно читает Funding USDT/USDC/ETH/BTC, Flexible Easy Earn и P2P. BYBIT-B01/B02/B05 закрыты для проверенного доступа, BYBIT-B03/B04 открыты под task-0.10, BYBIT-B06 отложен без блокировки. 18 наблюдений/источников, 12 синтетических сценариев; продуктовые AC не объявлены пройденными. task-4.4 и BLK-04 Not Ready.

**Зависимости:** нет.

**Тип:** `research`.

### Изменение и контракты

По D-36 использовать официальный API для Funding USDT/USDC/ETH/BTC, используемого Easy Earn и P2P. Разделить первичные Chrome/публичные наблюдения и подписанные приватные ответы. Владелец отдельно разрешил выпуск IP-ограниченного RSA readOnly ключа и ввёл MFA; публиковать только синтетические проекции. Проверить UID/scopes, Funding/history, deposit/withdraw/transfer/Convert, Flexible positions/orders/yield/hourly и P2P list/detail read POST. Зафиксировать типы/единицы, source identity, gross/net/fee, связи, precision, окна/cursors/retention и противоречия документации. Не выполнять финансовые операции и не подавать заявку рекламодателя. BYBIT-B02/B05 закрываются только доказанным чтением; BYBIT-B03/B04 передаются task-0.10, VPS — task-0.9, USDC — task-0.7, контрактные сценарии — task-4.4. Остальные продукты не исследуются и не блокируют.

### Границы изменений

- `spec/001-want-keep-mvp/evidence/bybit.md`
- `spec/001-want-keep-mvp/evidence/bybit.en.md`
- `spec/001-want-keep-mvp/evidence/bybit.samples.json`
- `spec/001-want-keep-mvp/evidence/bybit-api.md`
- `spec/001-want-keep-mvp/evidence/bybit-api.en.md`

Это планируемые пути. Общие контракты: `spec/001-want-keep-mvp/contracts.md`; архитектура и команды: `constraints.md`. Менять только владельца поведения и затронутые тесты; при незакрытом контракте обновить evidence и остановить зависимую реализацию.

### Связанные требования

- **REQ-033:** Накопления показывают фактические начисления и прогноз по ставкам, срокам, капитализации и денежным потокам.
- **REQ-039:** Отсутствующие курсы и неподдерживаемые активы не превращаются в нулевые суммы или условный паритет USD/USDT/USDC.
- **REQ-041:** История сохраняет границы покрытия, курсоры, пробелы и статусы источника.
- **REQ-045:** Bybit автоматически читает Funding USDT/USDC/ETH/BTC, используемый Easy Earn и P2P; официальный API приоритетен. Остальные продукты отложены без блокировки по D-36.
- **REQ-048:** Интеграции и браузерный сборщик выполняют только разрешённые операции чтения.
- **REQ-065:** Принадлежность счёта, владелец внешнего аккаунта, автор записи и принадлежность расхода являются отдельными признаками.
- **REQ-073:** Оба управляют подключениями; банковскую авторизацию выполняет владелец внешнего аккаунта без раскрытия секретов партнёру или AI.

### Критерии приёмки

Связь с критерием задаёт покрытие; исследование или частичная задача не доказывает весь критерий продукта. Точный результат этой задачи указан ниже в проверке.

#### AC-045

- **Дано:** Безопасно подключён read-only аккаунт Bybit с подтверждёнными контрактами Funding, используемого Easy Earn и P2P.
- **Когда:** Читаются остатки, история с выбранной даты, повторные страницы, Convert, начисление/выплата Earn и P2P; моделируется отказ прав выбранного продукта.
- **Тогда:** Точные native amounts, ID, статусы, комиссии и coverage сопоставимы с источником; Funding и детали не удваивают обмен, комиссию или доход. P2P связывает crypto/fiat с банком либо требует уточнения. Отказ доступа/неполная история явно блокируют соответствующее покрытие; отсутствие Spot/UTA trading, futures, карты, On-Chain/Advanced Earn и иных неиспользуемых продуктов не блокирует.
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

### Проверка результата

```sh
make docs-check
```

RU/EN содержат датированные BYBIT-E01–E18, матрицу read-маршрутов и 12 синтетических сценариев. Подтверждены readOnly/права, 357 Funding-записей за 89 дней, Flexible и P2P list/detail, арифметика и replay. Неполная история, расхождения точности/totalPnl, hourly identity, отсутствие второго владельца/отзыва/VPS и production-коннектора выделены явно. Исследование может завершиться с блокерами по README.

Команды make созданы основой task-1.1; финансовые provider/integration/E2E suites ещё не реализованы. Для документации используется make docs-check. Live/paid/manual проверки отдельно фиксируют доступ и фактический результат; наличие команды или UI-доступа не доказывает runtime.

### Передача следующему агенту

Записать изменённые контракты, команды и результаты, ограничения, незакрытые вопросы и разблокированные зависимости. Обновить обе языковые версии и трассировку. Закрывать задачу только по доказательству её результата; GitHub Closed само по себе не означает Ready MVP.

**Commit boundary:** логическая граница этой задачи; commit/push/deploy не разрешены данной карточкой и требуют действующей авторизации пользователя.

## EN

Record a verifiable Bybit read contract for Funding USDT/USDC/ETH/BTC, Easy Earn and P2P and its access limits.

**Status:** Research completed on 2026-09-07: [EN evidence](https://github.com/pchkauu/want-keep/blob/docs/want-keep-mvp-sdd/spec/001-want-keep-mvp/evidence/bybit-api.en.md). RSA readOnly API successfully reads Funding USDT/USDC/ETH/BTC, Flexible Easy Earn and P2P. BYBIT-B01/B02/B05 closed for verified access, BYBIT-B03/B04 open under task-0.10, BYBIT-B06 non-blocking deferred. 18 observations/sources, 12 synthetic scenarios; product ACs are not declared passed. task-4.4 and BLK-04 remain Not Ready.

**Dependencies:** none.

**Kind:** `research`.

### Change and contracts

Under D-36 use official APIs for Funding USDT/USDC/ETH/BTC, used Easy Earn and P2P. Separate initial Chrome/public observations from signed private responses. The owner separately authorized an IP-restricted RSA readOnly key and entered MFA; publish synthetic projections only. Check UID/scopes, Funding/history, deposit/withdraw/transfer/Convert, Flexible positions/orders/yield/hourly and P2P list/detail read POSTs. Record types/units, source identity, gross/net/fees, links, precision, windows/cursors/retention and documentation contradictions. Do not perform financial operations or apply for advertiser status. Close BYBIT-B02/B05 only with verified reads; hand BYBIT-B03/B04 to task-0.10, VPS to task-0.9, USDC to task-0.7 and contract scenarios to task-4.4. Other products are not researched and do not block.

### Change boundaries

- `spec/001-want-keep-mvp/evidence/bybit.md`
- `spec/001-want-keep-mvp/evidence/bybit.en.md`
- `spec/001-want-keep-mvp/evidence/bybit.samples.json`
- `spec/001-want-keep-mvp/evidence/bybit-api.md`
- `spec/001-want-keep-mvp/evidence/bybit-api.en.md`

These are planned paths. Shared contracts: `spec/001-want-keep-mvp/contracts.en.md`; architecture and commands: `constraints.en.md`. Change only the behavior owner and affected tests; an unresolved contract requires updated evidence and stops dependent implementation.

### Linked requirements

- **REQ-033:** Savings show actual accruals and forecasts using rates, terms, compounding and cash flows.
- **REQ-039:** Missing rates and unsupported assets never become zero amounts or assumed USD/USDT/USDC parity.
- **REQ-041:** History retains coverage boundaries, cursors, gaps and source status.
- **REQ-045:** Bybit automatically reads Funding USDT/USDC/ETH/BTC, used Easy Earn and P2P; the official API is preferred. Other products are deferred without blocking under D-36.
- **REQ-048:** Integrations and the browser collector perform authorized read operations only.
- **REQ-065:** Account ownership, external-account owner, record author and expense attribution are distinct dimensions.
- **REQ-073:** Both manage connections; the external-account owner performs bank authentication without exposing secrets to the partner or AI.

### Acceptance criteria

A criterion link establishes coverage; research or a partial task does not prove the entire product criterion. This task's exact outcome is specified in verification below.

#### AC-045

- **Given:** A read-only Bybit account is securely connected with verified Funding, used Easy Earn and P2P contracts.
- **When:** Balances, history from the selected date, replayed pages, Convert, Earn accrual/payout and P2P are read; a selected-product permission failure is simulated.
- **Then:** Exact native amounts, IDs, statuses, fees and coverage match the source; Funding and details do not duplicate an exchange, fee or income. P2P links crypto/fiat with the bank or requires clarification. Access failure/incomplete history explicitly blocks the corresponding coverage; absent Spot/UTA trading, futures, card, On-Chain/Advanced Earn and other unused products do not block.
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

### Verification

```sh
make docs-check
```

RU/EN include dated BYBIT-E01–E18, the read-route matrix and 12 synthetic scenarios. Confirm readOnly/scopes, 357 Funding records over 89 days, Flexible and P2P list/detail, arithmetic and replay. Explicitly distinguish incomplete history, precision/totalPnl differences, hourly identity and absent second-owner/revocation/VPS/production-connector evidence. Research can complete with blockers under README.

The task-1.1 foundation provides make commands; financial provider/integration/E2E suites are not implemented yet. Use make docs-check for documentation. Live/paid/manual checks separately record access and outcomes; an existing command or UI access is not runtime proof.

### Handoff to the next agent

Record changed contracts, commands/results, limitations, unresolved questions and unblocked dependencies. Update both languages and traceability. Close the task only with evidence of its outcome; GitHub Closed alone does not mean the MVP is Ready.

**Commit boundary:** this task's logical boundary; this card does not authorize commit/push/deploy, which require current user authorization.
