<!-- want-keep-task: task-0.3 -->
# task-0.3 — Проверить контракт чтения Ozon Банк / Verify Ozon Bank read contract

## RU

Получить проверяемый контракт чтения дебетовой карты Ozon и связанного основного счёта.

**Состояние:** Исследование Ozon и sanitized HAR projections завершены; task-0.10 закрепила D-39 identity и перенесла session/history/второй аккаунт в gate task-4.3. SDD Ready; connector не реализован.

**Зависимости:** нет.

**Тип:** `research`.

### Изменение и контракты

По решению D-32 проверить официальный API или допустимое чтение авторизованного кабинета для дебетовой карты и связанного основного счёта. Зафиксировать остатки, операции, устойчивые ID, статусы, комиссии, пагинацию, глубину истории, доступные сведения и валюты, требования MFA и границы прав. Хранить только синтетические или обезличенные контракты; секреты подключает владелец вне репозитория. Кредитка, накопительные счета и вклады Ozon — будущее расширение, их отсутствие не блокер текущего MVP. Непроверенный автоматический read-контракт дебетового продукта не подменять ручной выпиской. Проверить независимые внешние аккаунты участников и устойчивую идентичность при повторной авторизации; не считать connectionId идентификатором реального счёта.

### Границы изменений

- `spec/001-want-keep-mvp/evidence/ozon.md`
- `spec/001-want-keep-mvp/evidence/ozon.en.md`
- `spec/001-want-keep-mvp/evidence/ozon.samples.json`

Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-041:** История сохраняет границы покрытия, курсоры, пробелы и статусы источника.
- **REQ-044:** Интеграция Ozon Банк автоматически читает дебетовую карту и связанный основной счёт: остатки, операции и доступные сведения в пределах подтверждённого контракта. Другие продукты Ozon отложены до расширения контракта.
- **REQ-048:** Интеграции и браузерный сборщик выполняют только разрешённые операции чтения.
- **REQ-065:** Принадлежность счёта, владелец внешнего аккаунта, автор записи и принадлежность расхода являются отдельными признаками.
- **REQ-073:** Оба управляют подключениями; банковскую авторизацию выполняет владелец внешнего аккаунта без раскрытия секретов партнёру или AI.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

#### AC-044

- **Дано:** Подключён разрешённый личный аккаунт Ozon Банк с дебетовой картой и связанным основным счётом.
- **Когда:** Запрошены остатки, операции и доступные сведения дебетового продукта; та же карта и счёт встречаются в нескольких представлениях.
- **Тогда:** Данные сопоставимы с источником, свидетельство чтения сохранено, карта не удваивает остаток счёта. Недоступность обязательных полей дебетового продукта отмечена явно. Отсутствие кредитки, накоплений или вкладов Ozon не блокирует MVP: эти продукты вне текущего контракта и не показаны как реализованные.
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

Evidence RU/EN и ozon.samples.json содержат реальные маршруты/структуры с синтетическими значениями и происхождением. Получены счета, карты, баланс, детали покупки/пополнения/возврата, страницы и комиссия. accountToken меняется; groupID общий у перевода и комиссии, lastOperationId различаются. Семь страниц связаны курсорами, но все имеют next: завершение истории не подтверждено. Проверки приложения, прямого вызова сборщиком, обновления сессии и двух внешних аккаунтов не объявлены пройденными; task-4.3 не разблокирована. Иные продукты Ozon вне текущего scope по D-32.

Команды `make` — будущий контракт, создаваемый task-1.1; сейчас они не существуют. Live/paid/manual проверки отдельно фиксируют доступ и фактический результат. Исследования не обходят блокер отсутствующего доступа.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Produce a verifiable read contract for the Ozon debit card and linked main account.

**Status:** Ozon research and sanitized HAR projections are complete; task-0.10 fixed D-39 identity and moved session/history/second-account checks into the task-4.3 gate. The SDD is Ready; the connector is not implemented.

**Dependencies:** none.

**Kind:** `research`.

### Change and contracts

Under D-32, verify the official API or acceptable authenticated-portal reading for the debit card and linked main account. Record balances, transactions, stable IDs, statuses, fees, pagination, history depth, available details/currencies, MFA and permission boundaries. Retain only synthetic or sanitized contracts; the owner supplies secrets outside the repository. Ozon credit cards, savings and deposits are a future extension; their absence does not block the current MVP. Do not replace an unverified automatic debit-product read contract with manual statements. Verify independent member accounts and stable identity across reauthorization; connectionId is not real-account identity.

### Change boundaries

- `spec/001-want-keep-mvp/evidence/ozon.md`
- `spec/001-want-keep-mvp/evidence/ozon.en.md`
- `spec/001-want-keep-mvp/evidence/ozon.samples.json`

Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-041:** History retains coverage boundaries, cursors, gaps and source status.
- **REQ-044:** The Ozon Bank integration automatically reads the debit card and linked main account: balances, transactions and available details under a verified contract. Other Ozon products are deferred until a contract extension.
- **REQ-048:** Integrations and the browser collector perform authorized read operations only.
- **REQ-065:** Account ownership, external-account owner, record author and expense attribution are distinct dimensions.
- **REQ-073:** Both manage connections; the external-account owner performs bank authentication without exposing secrets to the partner or AI.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

#### AC-044

- **Given:** An authorized personal Ozon Bank account with a debit card and linked main account is connected.
- **When:** Debit-product balances, transactions and available details are requested; the same card and account appear in several views.
- **Then:** Data matches the source, read evidence is retained and the card does not duplicate its account balance. Unavailable mandatory debit-product fields are explicit. Missing Ozon credit cards, savings or deposits do not block the MVP: these products are outside the current contract and are not presented as implemented.
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

RU/EN evidence and ozon.samples.json contain actual routes/structures with synthetic values and provenance. Accounts, cards, balance, purchase/top-up/refund details, pages and a commission were obtained. accountToken varies; a transfer and its commission share groupID but have distinct lastOperationId. Seven pages form a cursor chain, yet all have next: history completion is unverified. Application, direct collector replay, session renewal and two-external-account checks are not claimed as passed; task-4.3 remains blocked. Other Ozon products are outside current scope under D-32.

The `make` commands are a future contract established by task-1.1; they do not exist yet. Live/paid/manual checks separately record access and actual outcomes. Research does not bypass missing-access blockers.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
