<!-- want-keep-task: task-0.2 -->
# task-0.2 — Проверить контракт чтения Райффайзенбанк РФ / Verify Raiffeisenbank Russia read contract

## RU

Получить проверяемый контракт чтения расчётного счёта ИП через RBO API.

**Состояние:** Исследование Raiffeisen RBO/CAMT завершено; task-0.10 закрепила CAMT 1:N, identity/fallback, revision и unknown-balance правила. SDD Ready; OAuth/history/второй аккаунт/conformance остаются gate task-4.2.

**Зависимости:** нет.

**Тип:** `research`.

### Изменение и контракты

По D-35 исследовать официальный RBO API расчётного счёта ИП: остатки, поступления, списания, комиссии и историю. Личные карты, кредиты, накопления и вклады Raif не входят в MVP. Сохранить исходные JSON/CAMT только приватно; в репозитории — синтетические примеры. Зафиксировать RUR→RUB, accounts.id отдельно от number/accountKeys, CAMT.053.001.08, NtryRef и вложенные TxDtls, фактический completed против OpenAPI COMPLETED и no-statements против NO_STATEMENTS. Подтвердить границы истории, текущие/доступные/заблокированные остатки, точное значение кодов комиссий, ротацию/reauth, внешний аккаунт и независимость участников. Частичное подтверждение или документированный блокер завершает исследование по README, но не снимает Ready gate task-0.10 и не заменяет автоматический импорт ручной выпиской.

### Границы изменений

- `spec/001-want-keep-mvp/evidence/raiffeisen.md`
- `spec/001-want-keep-mvp/evidence/raiffeisen.en.md`
- `spec/001-want-keep-mvp/evidence/raiffeisen.samples.json`
- `spec/001-want-keep-mvp/evidence/raiffeisen.camt053.sample.xml`
- `deploy/raiffeisen-research/`

Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-041:** История сохраняет границы покрытия, курсоры, пробелы и статусы источника.
- **REQ-043:** Raiffeisen через RBO API читает только расчётный счёт ИП: остатки, поступления, списания, комиссии и историю (D-35).
- **REQ-048:** Интеграции и браузерный сборщик выполняют только разрешённые операции чтения.
- **REQ-065:** Принадлежность счёта, владелец внешнего аккаунта, автор записи и принадлежность расхода являются отдельными признаками.
- **REQ-073:** Оба управляют подключениями; банковскую авторизацию выполняет владелец внешнего аккаунта без раскрытия секретов партнёру или AI.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

#### AC-043

- **Дано:** Подключён разрешённый расчётный счёт ИП в RBO API.
- **Когда:** Запрошены остатки и движения, повторный импорт и intraday no-statements.
- **Тогда:** Данные совпадают с источником; дублей нет, комиссии учтены отдельно. Неизвестный текущий остаток не равен нулю: видны последний подтверждённый остаток, его дата и пробел покрытия.
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

Документ на двух языках содержит источник и дату, проверенные и непроверенные поля, примеры запросов/ответов без секретов и итог по каждому продукту. Реальный read-only прогон выполняется только после безопасного предоставления доступа владельцем; его отсутствие явно записано.

Основа task-1.1 уже содержит make-команды; наличие команды не означает реализацию адаптера или прохождение live-проверок. Исследовательские проверки Raif: python3 -m unittest discover -s deploy/raiffeisen-research -p 'test_*.py'. Токены и исходные ответы подключаются только приватно; незакрытые RAIF-B02/B03/B04/B06 остаются барьером реализации.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Produce a verifiable read contract for the individual entrepreneur current account through RBO API.

**Status:** Raiffeisen RBO/CAMT research is complete; task-0.10 fixed CAMT 1:N, identity/fallback, revision and unknown-balance rules. The SDD is Ready; OAuth/history/second-account/conformance remains the task-4.2 gate.

**Dependencies:** none.

**Kind:** `research`.

### Change and contracts

Under D-35 research the official RBO API for the individual entrepreneur current account: balances, incoming/outgoing movements, fees and history. Raif personal cards, credit, savings and deposits are outside the MVP. Keep original JSON/CAMT private; repository examples are synthetic. Record RUR→RUB, accounts.id separate from number/accountKeys, CAMT.053.001.08, NtryRef and nested TxDtls, actual completed versus OpenAPI COMPLETED and no-statements versus NO_STATEMENTS. Verify history boundaries, current/available/locked balances, exact fee-code semantics, rotation/reauth, external-account identity and member isolation. Partial evidence or documented blockers can complete research under README, but do not remove the task-0.10 Ready gate or replace automatic import with manual statements.

### Change boundaries

- `spec/001-want-keep-mvp/evidence/raiffeisen.md`
- `spec/001-want-keep-mvp/evidence/raiffeisen.en.md`
- `spec/001-want-keep-mvp/evidence/raiffeisen.samples.json`
- `spec/001-want-keep-mvp/evidence/raiffeisen.camt053.sample.xml`
- `deploy/raiffeisen-research/`

Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-041:** History retains coverage boundaries, cursors, gaps and source status.
- **REQ-043:** Raiffeisen RBO API reads only the entrepreneur current account: balances, receipts, debits, fees and history (D-35).
- **REQ-048:** Integrations and the browser collector perform authorized read operations only.
- **REQ-065:** Account ownership, external-account owner, record author and expense attribution are distinct dimensions.
- **REQ-073:** Both manage connections; the external-account owner performs bank authentication without exposing secrets to the partner or AI.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

#### AC-043

- **Given:** An authorized entrepreneur current account is connected through RBO API.
- **When:** Request balances, movements, repeated import and intraday no-statements.
- **Then:** Data matches the source; no duplicates, fees recorded separately. Unknown current balance is not zero: show the last verified balance, its date and the coverage gap.
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

The bilingual document includes dated sources, verified/unverified fields, secret-free request/response examples and per-product outcomes. A live read-only check runs only after the owner securely supplies access; absence of access is explicit.

The task-1.1 foundation already contains make commands; a command existing does not establish an implemented adapter or passing live checks. Raif research checks: python3 -m unittest discover -s deploy/raiffeisen-research -p 'test_*.py'. Tokens and original responses stay private; unresolved RAIF-B02/B03/B04/B06 remain an implementation gate.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
