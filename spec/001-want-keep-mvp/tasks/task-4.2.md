<!-- want-keep-task: task-4.2 -->
# task-4.2 — Реализовать коннектор Райффайзенбанк РФ / Implement Raiffeisenbank Russia connector

## RU

Автоматически получать согласованные остатки и движения расчётного счёта ИП через RBO API по D-35.

**Состояние:** Заблокировано зависимостями и проверкой SDD Ready; реализация не начата.

**Зависимости:** `task-0.2`, `task-3.3`, `task-2.4`, `task-2.5`.

**Тип:** `implementation`.

### Изменение и контракты

Реализовать RBO API-коннектор только расчётного счёта ИП по D-35 после закрытия оставшихся RAIF-B02/B03/B04/B06 в task-0.10. Начальные evidence и синтетические JSON/XML находятся в evidence/raiffeisen*. Обрабатывать lower-camel-case поля accounts, явный алиас RUR→RUB, отдельные UUID id и 20-значный number/accountKeys, Decimal, CAMT.053.001.08 и вложенные записи без двойного учёта. Сохранять source status/code и только подтверждённые преобразования completed/COMPLETED, no-statements/NO_STATEMENTS. no-statements не означает нулевой остаток. NtryRef подтверждён на двух перекрывающихся отчётах, но reconnect/коррекции требуют проверки. Коды FCHG/NTRF не заменяют доказательство типа расхода или внутреннего перевода. Реализовать защищённый Code Flow по contracts, атомарную ротацию, allowlist чтения, почасовой импорт и полноту истории. Два внешних аккаунта участников изолированы; повторное подключение связывается с прежним источником, старое задание после отключения не применяется. Прочие продукты Raif отложены; Playwright возможен только при доказанном API-пробеле.

### Границы изменений

- `backend/internal/integrations/raiffeisen/`
- `collector/src/providers/raiffeisen/`

Это планируемые пути. Общие контракты: `spec/001-want-keep-mvp/contracts.md`; архитектура и команды: `constraints.md`. Менять только владельца поведения и затронутые тесты; при незакрытом контракте обновить evidence и остановить зависимую реализацию.

### Связанные требования

- **REQ-006:** Перевод между счетами семьи, включая счета разных участников, меняет остатки без дохода или расхода по основной сумме.
- **REQ-007:** Обмен и P2P-конвертация собственных денег сохраняют обе валютные суммы, фактический курс и комиссии.
- **REQ-008:** Повторные импорты, чек и запись чата объединяют доказательства одной операции без повторного учёта.
- **REQ-035:** Торговая аналитика отделяет реализованный результат, нереализованный результат, комиссии и funding.
- **REQ-036:** Вознаграждения майнинга отделены от переводов между собственными кошельками.
- **REQ-040:** Каждый источник обновляется раз в час и по запросу с видимым временем успешного обновления.
- **REQ-041:** История сохраняет границы покрытия, курсоры, пробелы и статусы источника.
- **REQ-043:** Raiffeisen через RBO API читает только расчётный счёт ИП: остатки, поступления, списания, комиссии и историю (D-35).
- **REQ-048:** Интеграции и браузерный сборщик выполняют только разрешённые операции чтения.
- **REQ-061:** Повторные задания, перезапуски и параллельные изменения не создают двойных финансовых эффектов.
- **REQ-065:** Принадлежность счёта, владелец внешнего аккаунта, автор записи и принадлежность расхода являются отдельными признаками.
- **REQ-073:** Оба управляют подключениями; банковскую авторизацию выполняет владелец внешнего аккаунта без раскрытия секретов партнёру или AI.
- **REQ-076:** Семейная область проверяется для API, файлов, AI, фоновых задач и внешних ID независимо от присланных actor/owner.

### Критерии приёмки

Связь с критерием задаёт покрытие; исследование или частичная задача не доказывает весь критерий продукта. Точный результат этой задачи указан ниже в проверке.

#### AC-043

- **Дано:** Подключён разрешённый расчётный счёт ИП в RBO API.
- **Когда:** Запрошены остатки и движения, повторный импорт и intraday no-statements.
- **Тогда:** Данные совпадают с источником; дублей нет, комиссии учтены отдельно. Неизвестный текущий остаток не равен нулю: видны последний подтверждённый остаток, его дата и пробел покрытия.
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

#### AC-071

- **Дано:** Источник различает gross P&L, net P&L, fee, funding и reward/transfer.
- **Когда:** Одна экономическая операция встречается в нескольких журналах.
- **Тогда:** Происхождение показателей сохранено; комиссия и доход не удваиваются; выбор net/gross подтверждён контрактом.
- **Уровень:** `contract+integration`.

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
make test-contract PROVIDER=raiffeisen && make test-integration AREA=raiffeisen
```

Все продукты имеют пройденные синтетические контрактные сценарии и отдельный read-only live readback с безопасно подключённым аккаунтом; доступность только части продуктов не считается полным результатом.

Основа task-1.1 уже содержит make-команды; наличие команды не означает реализацию адаптера или прохождение live-проверок. Исследовательские проверки Raif: python3 -m unittest discover -s deploy/raiffeisen-research -p 'test_*.py'. Токены и исходные ответы подключаются только приватно; незакрытые RAIF-B02/B03/B04/B06 остаются барьером реализации.

### Передача следующему агенту

Записать изменённые контракты, команды и результаты, ограничения, незакрытые вопросы и разблокированные зависимости. Обновить обе языковые версии и трассировку. Закрывать задачу только по доказательству её результата; GitHub Closed само по себе не означает Ready MVP.

**Commit boundary:** логическая граница этой задачи; commit/push/deploy не разрешены данной карточкой и требуют действующей авторизации пользователя.

## EN

Automatically retrieve consistent balances and movements of the individual entrepreneur current account through RBO API under D-35.

**Status:** Blocked by dependencies and the SDD Ready gate; implementation has not started.

**Dependencies:** `task-0.2`, `task-3.3`, `task-2.4`, `task-2.5`.

**Kind:** `implementation`.

### Change and contracts

Implement the RBO API connector for the individual entrepreneur current account only under D-35, after task-0.10 resolves RAIF-B02/B03/B04/B06. Initial evidence and synthetic JSON/XML are in evidence/raiffeisen*. Handle lower-camel-case account fields, explicit RUR→RUB alias, separate UUID id and 20-digit number/accountKeys, Decimal, CAMT.053.001.08 and nested entries without double posting. Preserve source status/code and only verified mappings for completed/COMPLETED and no-statements/NO_STATEMENTS. no-statements is not a zero balance. NtryRef was verified across two overlapping reports, but reconnect/correction behaviour needs testing. FCHG/NTRF codes do not establish expense or internal-transfer classification. Implement protected Code Flow under contracts, atomic rotation, a read allowlist, hourly imports and history completeness. Isolate members’ external accounts; reconnect links the existing source and stale jobs cannot apply after disconnect. Other Raif products are deferred; Playwright requires a proven API gap.

### Change boundaries

- `backend/internal/integrations/raiffeisen/`
- `collector/src/providers/raiffeisen/`

These are planned paths. Shared contracts: `spec/001-want-keep-mvp/contracts.en.md`; architecture and commands: `constraints.en.md`. Change only the behavior owner and affected tests; an unresolved contract requires updated evidence and stops dependent implementation.

### Linked requirements

- **REQ-006:** Transfers between household accounts, including different members’ accounts, change balances without principal income or expense.
- **REQ-007:** Exchange and P2P conversion of owned money preserve both currency amounts, the actual rate and fees.
- **REQ-008:** Repeated imports, receipts and chat entries combine evidence of one transaction without double counting.
- **REQ-035:** Trading analytics separates realized P&L, unrealized P&L, fees and funding.
- **REQ-036:** Mining rewards are separate from transfers between owned wallets.
- **REQ-040:** Each source refreshes hourly and on demand with a visible last-success timestamp.
- **REQ-041:** History retains coverage boundaries, cursors, gaps and source status.
- **REQ-043:** Raiffeisen RBO API reads only the entrepreneur current account: balances, receipts, debits, fees and history (D-35).
- **REQ-048:** Integrations and the browser collector perform authorized read operations only.
- **REQ-061:** Repeated jobs, restarts and concurrent changes cannot create duplicate financial effects.
- **REQ-065:** Account ownership, external-account owner, record author and expense attribution are distinct dimensions.
- **REQ-073:** Both manage connections; the external-account owner performs bank authentication without exposing secrets to the partner or AI.
- **REQ-076:** Household scope is checked for APIs, files, AI, jobs and external IDs independently of supplied actor/owner fields.

### Acceptance criteria

A criterion link establishes coverage; research or a partial task does not prove the entire product criterion. This task's exact outcome is specified in verification below.

#### AC-043

- **Given:** An authorized entrepreneur current account is connected through RBO API.
- **When:** Request balances, movements, repeated import and intraday no-statements.
- **Then:** Data matches the source; no duplicates, fees recorded separately. Unknown current balance is not zero: show the last verified balance, its date and the coverage gap.
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

#### AC-071

- **Given:** A source distinguishes gross P&L, net P&L, fee, funding and reward/transfer.
- **When:** One economic event appears in several logs.
- **Then:** Metric provenance is preserved; fees and income are not doubled; net/gross semantics are contract-verified.
- **Level:** `contract+integration`.

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
make test-contract PROVIDER=raiffeisen && make test-integration AREA=raiffeisen
```

All products have passing synthetic contract scenarios and separate read-only live readback using a securely connected account; partial product access is not a complete result.

The task-1.1 foundation already contains make commands; a command existing does not establish an implemented adapter or passing live checks. Raif research checks: python3 -m unittest discover -s deploy/raiffeisen-research -p 'test_*.py'. Tokens and original responses stay private; unresolved RAIF-B02/B03/B04/B06 remain an implementation gate.

### Handoff to the next agent

Record changed contracts, commands/results, limitations, unresolved questions and unblocked dependencies. Update both languages and traceability. Close the task only with evidence of its outcome; GitHub Closed alone does not mean the MVP is Ready.

**Commit boundary:** this task's logical boundary; this card does not authorize commit/push/deploy, which require current user authorization.
