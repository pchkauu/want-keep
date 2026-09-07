<!-- want-keep-task: task-0.5 -->
# task-0.5 — Проверить контракт чтения Aifory Pro / Verify Aifory Pro read contract

## RU

Зафиксировать проверяемый read-контракт и ограничения RUB, USDT, ETH и используемой криптокарты Aifory по D-33.

**Состояние:** Исследование завершено 2026-09-07: [RU evidence](https://github.com/pchkauu/want-keep/blob/docs/want-keep-mvp-sdd/spec/001-want-keep-mvp/evidence/aifory.md), [EN evidence](https://github.com/pchkauu/want-keep/blob/docs/want-keep-mvp-sdd/spec/001-want-keep-mvp/evidence/aifory.en.md). UI-чтение выполнено; AIFORY-B02–B04 / BLK-05 остаются открыты под контролем task-0.10. Другие продукты отложены без блокировки. task-4.5 и MVP — Not Ready.

**Зависимости:** нет.

**Тип:** `research`.

### Изменение и контракты

Исследовать только RUB-счета, USDT, ETH и существующую криптокарту USD (D-33), включая их движения/комиссии. Другие продукты и их условия — будущее расширение, не блокер; движение по включённому кошельку сохраняется даже при отложенном связанном сервисе. Проверить официальный личный API либо разрешённое чтение кабинета, согласие оператора на автоматизацию, бесплатность, auth и allowlist. Зафиксировать поля, IDs/revisions, gross/net, card authorization/clearing и funding USD/USDT, время/полноту истории, reauth и два аккаунта семьи. UI/OCR/общий маршрут/маска не заменяют структурированный контракт и identity. Результат — датированный RU/EN evidence с синтетическими примерами и явными ограничениями. AIFORY-B02–B04 закрывает task-0.10 до task-4.5; отсутствие других продуктов не переносить в блокеры.

### Границы изменений

- `spec/001-want-keep-mvp/evidence/aifory.md`
- `spec/001-want-keep-mvp/evidence/aifory.en.md`

Это планируемые пути. Общие контракты: `spec/001-want-keep-mvp/contracts.md`; архитектура и команды: `constraints.md`. Менять только владельца поведения и затронутые тесты; при незакрытом контракте обновить evidence и остановить зависимую реализацию.

### Связанные требования

- **REQ-039:** Отсутствующие курсы и неподдерживаемые активы не превращаются в нулевые суммы или условный паритет USD/USDT/USDC.
- **REQ-041:** История сохраняет границы покрытия, курсоры, пробелы и статусы источника.
- **REQ-046:** Aifory Pro автоматически читает RUB-счета, USDT, ETH и используемую криптокарту, включая движения и комиссии этих продуктов. Остальные продукты отложены и не блокируют MVP.
- **REQ-048:** Интеграции и браузерный сборщик выполняют только разрешённые операции чтения.
- **REQ-065:** Принадлежность счёта, владелец внешнего аккаунта, автор записи и принадлежность расхода являются отдельными признаками.
- **REQ-073:** Оба управляют подключениями; банковскую авторизацию выполняет владелец внешнего аккаунта без раскрытия секретов партнёру или AI.

### Критерии приёмки

Связь с критерием задаёт покрытие; исследование или частичная задача не доказывает весь критерий продукта. Точный результат этой задачи указан ниже в проверке.

#### AC-046

- **Дано:** Подключён разрешённый личный аккаунт Aifory с RUB-счетами, USDT/TRC-20, ETH/Ethereum и используемой картой USD (D-33); другие продукты не подключены.
- **Когда:** Повторно прочитаны остатки и история, обмен RUB/USDT, ETH-вывод с комиссией, пополнение карты USDT/USD и похожие pending/confirmed оплаты с отдельной fee.
- **Тогда:** Каждый включённый продукт имеет структурированное доказательство чтения; RUB-группа не дублирует дочерние счета, ETH точен. Движения и комиссии учтены один раз по подтверждённым ID/связям/статусам; знак UI и паритет USDT/USD не предполагаются. Неизвестная связь требует уточнения. Пробелы включённых продуктов блокируют адаптер; остальные продукты не требуются, но их движения по выбранным кошелькам не пропускаются.
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
python3 spec/001-want-keep-mvp/tools/spec_tool.py check
```

RU/EN evidence содержит 19 наблюдений/источников, продуктовую матрицу D-33, проверенные и неизвестные поля, синтетический сценарий и AIFORY-B01–B05. Реальные HTTP request/response не получены и не выдумываются; отсутствие API и полного runtime не подменяется UI-прогоном. Отложенные продукты не блокируют, открытые вопросы выбранных продуктов переданы task-0.10.

Команды `make` — будущий контракт, создаваемый task-1.1; сейчас они не существуют. Live/paid/manual проверки отдельно фиксируют доступ и фактический результат. Исследования не обходят блокер отсутствующего доступа.

### Передача следующему агенту

Записать изменённые контракты, команды и результаты, ограничения, незакрытые вопросы и разблокированные зависимости. Обновить обе языковые версии и трассировку. Закрывать задачу только по доказательству её результата; GitHub Closed само по себе не означает Ready MVP.

**Commit boundary:** логическая граница этой задачи; commit/push/deploy не разрешены данной карточкой и требуют действующей авторизации пользователя.

## EN

Record a verifiable read contract and limitations for Aifory RUB, USDT, ETH and the existing crypto card under D-33.

**Status:** Research completed 2026-09-07: [RU evidence](https://github.com/pchkauu/want-keep/blob/docs/want-keep-mvp-sdd/spec/001-want-keep-mvp/evidence/aifory.md), [EN evidence](https://github.com/pchkauu/want-keep/blob/docs/want-keep-mvp-sdd/spec/001-want-keep-mvp/evidence/aifory.en.md). UI reading performed; AIFORY-B02–B04 / BLK-05 remain open under task-0.10. Other products are deferred without blocking. task-4.5 and MVP are Not Ready.

**Dependencies:** none.

**Kind:** `research`.

### Change and contracts

Research only RUB accounts, USDT, ETH and the existing USD crypto card (D-33), including their movements/fees. Other products/terms are a future extension, not a blocker; retain movements through included wallets even when the related service is deferred. Verify an official personal API or permitted portal reading, operator consent for automation, free access, auth and allowlist. Record fields, IDs/revisions, gross/net, card authorization/clearing, USD/USDT funding, time/history completeness, reauth and two household accounts. UI/OCR/common route/mask do not replace a structured contract or identity. Deliver dated RU/EN evidence with synthetic examples and explicit limits. task-0.10 closes AIFORY-B02–B04 before task-4.5; absent other products must not become blockers.

### Change boundaries

- `spec/001-want-keep-mvp/evidence/aifory.md`
- `spec/001-want-keep-mvp/evidence/aifory.en.md`

These are planned paths. Shared contracts: `spec/001-want-keep-mvp/contracts.en.md`; architecture and commands: `constraints.en.md`. Change only the behavior owner and affected tests; an unresolved contract requires updated evidence and stops dependent implementation.

### Linked requirements

- **REQ-039:** Missing rates and unsupported assets never become zero amounts or assumed USD/USDT/USDC parity.
- **REQ-041:** History retains coverage boundaries, cursors, gaps and source status.
- **REQ-046:** Aifory Pro automatically reads RUB accounts, USDT, ETH and the existing crypto card, including these products’ movements and fees. Other products are deferred and do not block the MVP.
- **REQ-048:** Integrations and the browser collector perform authorized read operations only.
- **REQ-065:** Account ownership, external-account owner, record author and expense attribution are distinct dimensions.
- **REQ-073:** Both manage connections; the external-account owner performs bank authentication without exposing secrets to the partner or AI.

### Acceptance criteria

A criterion link establishes coverage; research or a partial task does not prove the entire product criterion. This task's exact outcome is specified in verification below.

#### AC-046

- **Given:** An authorized personal Aifory account provides RUB accounts, USDT/TRC-20, ETH/Ethereum and the existing USD card (D-33); other products are not connected.
- **When:** Balances/history are read again, including RUB/USDT exchange, ETH withdrawal with a fee, USDT/USD card funding and similar pending/confirmed payments with a separate fee.
- **Then:** Each included product has structured read evidence; RUB groups do not duplicate child accounts and ETH remains exact. Movements/fees count once using verified IDs/links/statuses; neither UI signs nor USDT/USD parity are assumed. Unknown linkage requires clarification. Gaps in included products block the adapter; other products are not required, but their movements through selected wallets are retained.
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
python3 spec/001-want-keep-mvp/tools/spec_tool.py check
```

RU/EN evidence contains 19 observations/sources, the D-33 product matrix, established/unknown fields, a synthetic scenario and AIFORY-B01–B05. Real HTTP requests/responses were not obtained and must not be invented; UI reading does not substitute for API/full runtime. Deferred products do not block; open questions for selected products are handed to task-0.10.

The `make` commands are a future contract established by task-1.1; they do not exist yet. Live/paid/manual checks separately record access and actual outcomes. Research does not bypass missing-access blockers.

### Handoff to the next agent

Record changed contracts, commands/results, limitations, unresolved questions and unblocked dependencies. Update both languages and traceability. Close the task only with evidence of its outcome; GitHub Closed alone does not mean the MVP is Ready.

**Commit boundary:** this task's logical boundary; this card does not authorize commit/push/deploy, which require current user authorization.
