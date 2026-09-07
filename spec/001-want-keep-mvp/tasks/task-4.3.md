<!-- want-keep-task: task-4.3 -->
# task-4.3 — Реализовать коннектор Ozon Банк / Implement Ozon Bank connector

## RU

Автоматически получать согласованные данные дебетовой карты Ozon и связанного основного счёта без двойного остатка.

**Состояние:** Не начато; задача ожидает собственные зависимости и entry gates.

**Зависимости:** `task-0.3`, `task-3.3`, `task-2.4`, `task-2.5`.

**Тип:** `implementation`.

### Изменение и контракты

Реализовать текущий контракт Ozon по D-32: дебетовая карта и связанный основной счёт, их остатки и операции. Кредитки, накопления и вклады — будущее расширение; их отсутствие не блокирует этот коннектор. Не подменять банковские операции заказами маркетплейса, UI-сводку «Доходы» семейным доходом или округлённый агрегат точной суммой. Использовать синтетическую проекцию готовых HAR из evidence/ozon и реализовать только доказанный read allowlist, mappers и contract fixtures. Source identity следует D-39: accountToken и connection ID не являются account identity; route-specific record ID живёт в product/log namespace. Проверить повторы, поздние изменения, истечение сессии, часовой refresh и историю с выбранной даты. До provider deployment проверить разрешение на session transport, lifecycle сессии, второй аккаунт и полный live readback; collector используется только при подтверждённой необходимости браузера. Два аккаунта участников изолированы; повторное подключение одного реального аккаунта связывается с существующим источником. Старый результат после отключения не применяется. Поля и синтетические проекции: evidence/ozon.md и evidence/ozon.samples.json. Валюта RUR преобразуется в RUB; cents обрабатываются точно. Не использовать меняющийся accountToken или один groupID как ключ дедупликации: перевод и комиссия могут разделять groupID. lastOperationId — кандидат source ID, parentOperationId связывает комиссию; жизненный цикл сверяется отдельно. Учитывать status вместе с типом и meta, не проводить canceled или неизвестное сочетание. Отсутствие available/locked и исходной покупки возврата остаётся явным. Каждый next обрабатывается с сохранением coverage; конец HAR не означает конец истории. Provider evidence публикуется admission service для точного D-43 binding; conformance до admission идёт в quarantine без source record/проводки, а смена binding снова закрывает sync.

### Границы изменений

- `backend/internal/integrations/ozon/`
- `collector/src/providers/ozon/`

Это планируемые пути. Общие контракты: `spec/001-want-keep-mvp/contracts.md`; архитектура и команды: `constraints.md`. Менять только владельца поведения и затронутые тесты; при незакрытом контракте обновить evidence и остановить зависимую реализацию.

### Связанные требования

- **REQ-006:** Перевод между счетами семьи, включая счета разных участников, меняет остатки без дохода или расхода по основной сумме.
- **REQ-007:** Обмен и P2P-конвертация собственных денег сохраняют обе валютные суммы, фактический курс и комиссии.
- **REQ-008:** Повторные импорты, чек и запись чата объединяют доказательства одной операции без повторного учёта.
- **REQ-035:** Торговая аналитика отделяет реализованный результат, нереализованный результат, комиссии и funding.
- **REQ-036:** Вознаграждения майнинга отделены от переводов между собственными кошельками.
- **REQ-040:** Каждый источник обновляется раз в час и по запросу с видимым временем успешного обновления.
- **REQ-041:** История сохраняет границы покрытия, курсоры, пробелы и статусы источника.
- **REQ-044:** Интеграция Ozon Банк автоматически читает дебетовую карту и связанный основной счёт: остатки, операции и доступные сведения в пределах подтверждённого контракта. Другие продукты Ozon отложены до расширения контракта.
- **REQ-048:** Интеграции и браузерный сборщик выполняют только разрешённые операции чтения.
- **REQ-061:** Повторные задания, перезапуски и параллельные изменения не создают двойных финансовых эффектов.
- **REQ-065:** Принадлежность счёта, владелец внешнего аккаунта, автор записи и принадлежность расхода являются отдельными признаками.
- **REQ-073:** Оба управляют подключениями; банковскую авторизацию выполняет владелец внешнего аккаунта без раскрытия секретов партнёру или AI.
- **REQ-076:** Семейная область проверяется для API, файлов, AI, фоновых задач и внешних ID независимо от присланных actor/owner.
- **REQ-088:** Синхронизация провайдера разрешена только актуальным server-side admission, связанным с проверенными версиями адаптера, контракта, allowlist, конфигурации и окружения.

### Критерии приёмки

Связь с критерием задаёт покрытие; исследование или частичная задача не доказывает весь критерий продукта. Точный результат этой задачи указан ниже в проверке.

#### AC-044

- **Дано:** Подключён разрешённый личный аккаунт Ozon Банк с дебетовой картой и связанным основным счётом.
- **Когда:** Запрошены остатки, операции и доступные сведения дебетового продукта; та же карта и счёт встречаются в нескольких представлениях.
- **Тогда:** Данные сопоставимы с источником, свидетельство чтения сохранено, карта не удваивает остаток счёта. Недоступность обязательных полей дебетового продукта отмечена явно. Отсутствие кредитки, накоплений или вкладов Ozon не блокирует MVP: эти продукты вне текущего контракта и не показаны как реализованные.
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

#### AC-106

- **Дано:** Подключение авторизовано, но provider/host gate неполон либо прошлый admission относится к другой версии binding.
- **Когда:** Участник или scheduler запрашивает sync, либо меняются build, contract, allowlist, config, permission или environment.
- **Тогда:** Сервер возвращает `provider_not_admitted`, collector не запускается и проводок нет. Только admission service ставит `admitted` после provider evidence task-4.x и host evidence task-8.x для точного binding; любое расхождение снова закрывает sync.
- **Уровень:** `integration+security`.

### Проверка результата

```sh
make test-contract PROVIDER=ozon && make test-integration AREA=ozon
```

Дебетовый контракт имеет пройденные синтетические сценарии и отдельный read-only live readback с безопасно подключённым аккаунтом. Карта не удваивает баланс, переводы/возвраты не становятся ложным доходом. Отсутствие других продуктов Ozon не блокирует приёмку по D-32; неизвестные обязательные поля и непроверенная полнота дебетовых данных не скрываются.

Команды `make` — будущий контракт, создаваемый task-1.1; сейчас они не существуют. Live/paid/manual проверки отдельно фиксируют доступ и фактический результат. Исследования не обходят блокер отсутствующего доступа.

### Передача следующему агенту

Записать изменённые контракты, команды и результаты, ограничения, незакрытые вопросы и разблокированные зависимости. Обновить обе языковые версии и трассировку. Закрывать задачу только по доказательству её результата; GitHub Closed само по себе не означает Ready MVP.

**Commit boundary:** логическая граница этой задачи; commit/push/deploy не разрешены данной карточкой и требуют действующей авторизации пользователя.

## EN

Automatically retrieve consistent Ozon debit-card and linked main-account data without duplicating the balance.

**Status:** Not started; the task awaits its own dependencies and entry gates.

**Dependencies:** `task-0.3`, `task-3.3`, `task-2.4`, `task-2.5`.

**Kind:** `implementation`.

### Change and contracts

Implement the current Ozon contract under D-32: debit card and linked main account, their balances and transactions. Credit cards, savings and deposits are a future extension; their absence does not block this connector. Do not substitute marketplace orders for bank transactions, the UI Income summary for household income, or rounded aggregates for exact amounts. Use the synthetic projection of the completed HAR evidence and implement only the proven read allowlist, mappers and contract fixtures. Source identity follows D-39: accountToken and connection ID are not account identity; a route-specific record ID lives in the product/log namespace. Verify replay, late revisions, session expiry, hourly refresh and history from the selected date. Before provider deployment verify session-transport permission, session lifecycle, a second account and complete live readback; use the collector only when browser access is proven necessary. Member accounts remain isolated; reconnection of one real account links to the existing source. Stale results cannot apply after disconnect. Field shapes and synthetic projections: evidence/ozon.en.md and evidence/ozon.samples.json. Map RUR to RUB and process cents exactly. Never use rotating accountToken or groupID alone for deduplication: a transfer and its commission may share groupID. lastOperationId is a source-ID candidate and parentOperationId links the commission; reconcile lifecycle separately. Combine status with type/meta; never post canceled or unknown combinations. Missing available/locked fields and original refund purchase remain explicit. Process each next with persisted coverage; HAR completion is not history completion. Provider evidence is supplied to the admission service for the exact D-43 binding; pre-admission conformance runs in quarantine without source records/postings, and any binding change closes sync again.

### Change boundaries

- `backend/internal/integrations/ozon/`
- `collector/src/providers/ozon/`

These are planned paths. Shared contracts: `spec/001-want-keep-mvp/contracts.en.md`; architecture and commands: `constraints.en.md`. Change only the behavior owner and affected tests; an unresolved contract requires updated evidence and stops dependent implementation.

### Linked requirements

- **REQ-006:** Transfers between household accounts, including different members’ accounts, change balances without principal income or expense.
- **REQ-007:** Exchange and P2P conversion of owned money preserve both currency amounts, the actual rate and fees.
- **REQ-008:** Repeated imports, receipts and chat entries combine evidence of one transaction without double counting.
- **REQ-035:** Trading analytics separates realized P&L, unrealized P&L, fees and funding.
- **REQ-036:** Mining rewards are separate from transfers between owned wallets.
- **REQ-040:** Each source refreshes hourly and on demand with a visible last-success timestamp.
- **REQ-041:** History retains coverage boundaries, cursors, gaps and source status.
- **REQ-044:** The Ozon Bank integration automatically reads the debit card and linked main account: balances, transactions and available details under a verified contract. Other Ozon products are deferred until a contract extension.
- **REQ-048:** Integrations and the browser collector perform authorized read operations only.
- **REQ-061:** Repeated jobs, restarts and concurrent changes cannot create duplicate financial effects.
- **REQ-065:** Account ownership, external-account owner, record author and expense attribution are distinct dimensions.
- **REQ-073:** Both manage connections; the external-account owner performs bank authentication without exposing secrets to the partner or AI.
- **REQ-076:** Household scope is checked for APIs, files, AI, jobs and external IDs independently of supplied actor/owner fields.
- **REQ-088:** Provider sync is allowed only by a current server-side admission bound to verified adapter, contract, allowlist, configuration and environment revisions.

### Acceptance criteria

A criterion link establishes coverage; research or a partial task does not prove the entire product criterion. This task's exact outcome is specified in verification below.

#### AC-044

- **Given:** An authorized personal Ozon Bank account with a debit card and linked main account is connected.
- **When:** Debit-product balances, transactions and available details are requested; the same card and account appear in several views.
- **Then:** Data matches the source, read evidence is retained and the card does not duplicate its account balance. Unavailable mandatory debit-product fields are explicit. Missing Ozon credit cards, savings or deposits do not block the MVP: these products are outside the current contract and are not presented as implemented.
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

#### AC-106

- **Given:** A connection is authenticated, but the provider/host gate is incomplete or the prior admission belongs to a different binding revision.
- **When:** A member or scheduler requests sync, or the build, contract, allowlist, configuration, permission or environment changes.
- **Then:** The server returns `provider_not_admitted`, never starts the collector and creates no posting. Only the admission service sets `admitted` after task-4.x provider evidence and task-8.x host evidence for the exact binding; any mismatch closes sync again.
- **Level:** `integration+security`.

### Verification

```sh
make test-contract PROVIDER=ozon && make test-integration AREA=ozon
```

The debit contract has passing synthetic scenarios and separate read-only live readback with a securely connected account. The card does not duplicate balance; transfers/refunds do not become false income. Missing other Ozon products does not block acceptance under D-32; unknown mandatory fields and unverified debit-data completeness remain explicit.

The `make` commands are a future contract established by task-1.1; they do not exist yet. Live/paid/manual checks separately record access and actual outcomes. Research does not bypass missing-access blockers.

### Handoff to the next agent

Record changed contracts, commands/results, limitations, unresolved questions and unblocked dependencies. Update both languages and traceability. Close the task only with evidence of its outcome; GitHub Closed alone does not mean the MVP is Ready.

**Commit boundary:** this task's logical boundary; this card does not authorize commit/push/deploy, which require current user authorization.
