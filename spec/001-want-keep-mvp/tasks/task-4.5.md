<!-- want-keep-task: task-4.5 -->
# task-4.5 — Реализовать коннектор Aifory Pro / Implement Aifory Pro connector

## RU

Автоматически получать согласованные данные RUB-счетов, USDT, ETH и используемой карты Aifory по D-33.

**Состояние:** Заблокировано зависимостями и проверкой SDD Ready; реализация не начата.

**Зависимости:** `task-0.5`, `task-3.3`, `task-2.4`, `task-2.5`.

**Тип:** `implementation`.

### Изменение и контракты

Реализовать только разрешённый и структурированно подтверждённый в evidence/aifory read-контракт после закрытия AIFORY-B02–B04 в task-0.10. D-33 исключает остальные продукты из блокеров. RUB-группы отделены от счетов; ID не строится из имени офиса, адреса или /home/wallet. ETH хранит точные native amounts/network/fee. USD-карта отделена от USDT: нужны funding legs, gross/net, курс/база комиссии и lifecycle authorization/clearing/refund. Похожие pending/confirmed строки не объединять без доказанной связи и не списывать дважды. Сохранять все движения выбранных кошельков, включая операции отложенных сервисов. Проверить повтор, revisions, неполную историю/resume, session expiry, hourly refresh, выбранную дату, два внешних аккаунта и reauth; stale job после disconnect не применяется. Не использовать OCR/человеческие подписи как финансовый контракт. Нет открытия продуктов или платежей.

### Границы изменений

- `backend/internal/integrations/aifory/`
- `collector/src/providers/aifory/`

Это планируемые пути. Общие контракты: `spec/001-want-keep-mvp/contracts.md`; архитектура и команды: `constraints.md`. Менять только владельца поведения и затронутые тесты; при незакрытом контракте обновить evidence и остановить зависимую реализацию.

### Связанные требования

- **REQ-006:** Перевод между счетами семьи, включая счета разных участников, меняет остатки без дохода или расхода по основной сумме.
- **REQ-007:** Обмен и P2P-конвертация собственных денег сохраняют обе валютные суммы, фактический курс и комиссии.
- **REQ-008:** Повторные импорты, чек и запись чата объединяют доказательства одной операции без повторного учёта.
- **REQ-039:** Отсутствующие курсы и неподдерживаемые активы не превращаются в нулевые суммы или условный паритет USDT/USD.
- **REQ-040:** Каждый источник обновляется раз в час и по запросу с видимым временем успешного обновления.
- **REQ-041:** История сохраняет границы покрытия, курсоры, пробелы и статусы источника.
- **REQ-046:** Aifory Pro автоматически читает RUB-счета, USDT, ETH и используемую криптокарту, включая движения и комиссии этих продуктов. Остальные продукты отложены и не блокируют MVP.
- **REQ-048:** Интеграции и браузерный сборщик выполняют только разрешённые операции чтения.
- **REQ-061:** Повторные задания, перезапуски и параллельные изменения не создают двойных финансовых эффектов.
- **REQ-065:** Принадлежность счёта, владелец внешнего аккаунта, автор записи и принадлежность расхода являются отдельными признаками.
- **REQ-073:** Оба управляют подключениями; банковскую авторизацию выполняет владелец внешнего аккаунта без раскрытия секретов партнёру или AI.
- **REQ-076:** Семейная область проверяется для API, файлов, AI, фоновых задач и внешних ID независимо от присланных actor/owner.

### Критерии приёмки

Связь с критерием задаёт покрытие; исследование или частичная задача не доказывает весь критерий продукта. Точный результат этой задачи указан ниже в проверке.

#### AC-046

- **Дано:** Подключён разрешённый личный аккаунт Aifory с RUB-счетами, USDT/TRC-20, ETH/Ethereum и используемой картой USD (D-33); другие продукты не подключены.
- **Когда:** Повторно прочитаны остатки и история, обмен RUB/USDT, ETH-вывод с комиссией, пополнение карты USDT/USD и похожие pending/confirmed оплаты с отдельной fee.
- **Тогда:** Каждый включённый продукт имеет структурированное доказательство чтения; RUB-группа не дублирует дочерние счета, ETH точен. Движения и комиссии учтены один раз по подтверждённым ID/связям/статусам; знак UI и паритет USDT/USD не предполагаются. Неизвестная связь требует уточнения. Пробелы включённых продуктов блокируют адаптер; остальные продукты не требуются, но их движения по выбранным кошелькам не пропускаются.
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

#### AC-039

- **Дано:** В источнике есть неподдерживаемый актив, для USDT/USD отсутствует курс.
- **Когда:** Строится общая оценка.
- **Тогда:** Исходные данные сохранены, покрытие оценки обозначено неполным; нет скрытого нуля или автоматического курса 1:1.
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
make test-contract PROVIDER=aifory && make test-integration AREA=aifory
```

RUB, USDT, ETH и используемая карта имеют синтетические contract scenarios и отдельный live readback; пройдены AIFORY-E03/E08–E14 edge cases после подтверждения структурированного контракта. Остальные продукты не требуются. Перед реализацией закрыты AIFORY-B02–B04 и Ready.

Команды `make` — будущий контракт, создаваемый task-1.1; сейчас они не существуют. Live/paid/manual проверки отдельно фиксируют доступ и фактический результат. Исследования не обходят блокер отсутствующего доступа.

### Передача следующему агенту

Записать изменённые контракты, команды и результаты, ограничения, незакрытые вопросы и разблокированные зависимости. Обновить обе языковые версии и трассировку. Закрывать задачу только по доказательству её результата; GitHub Closed само по себе не означает Ready MVP.

**Commit boundary:** логическая граница этой задачи; commit/push/deploy не разрешены данной карточкой и требуют действующей авторизации пользователя.

## EN

Automatically retrieve consistent Aifory RUB-account, USDT, ETH and existing-card data under D-33.

**Status:** Blocked by dependencies and the SDD Ready gate; implementation has not started.

**Dependencies:** `task-0.5`, `task-3.3`, `task-2.4`, `task-2.5`.

**Kind:** `implementation`.

### Change and contracts

Implement only the permitted structured read contract established in evidence/aifory after task-0.10 closes AIFORY-B02–B04. D-33 excludes other products from blockers. Separate RUB groups/accounts; do not derive identity from office names, addresses or /home/wallet. ETH retains exact native amounts/network/fees. Separate USD card from USDT: establish funding legs, gross/net, rate/fee basis and authorization/clearing/refund lifecycle. Do not merge similar pending/confirmed rows without proven linkage or debit them twice. Retain all selected-wallet movements, including deferred-service operations. Verify replay, revisions, incomplete history/resume, session expiry, hourly refresh, selected start date, two external accounts and reauth; reject stale jobs after disconnect. Do not use OCR/human labels as financial contracts. No product opening or payments.

### Change boundaries

- `backend/internal/integrations/aifory/`
- `collector/src/providers/aifory/`

These are planned paths. Shared contracts: `spec/001-want-keep-mvp/contracts.en.md`; architecture and commands: `constraints.en.md`. Change only the behavior owner and affected tests; an unresolved contract requires updated evidence and stops dependent implementation.

### Linked requirements

- **REQ-006:** Transfers between household accounts, including different members’ accounts, change balances without principal income or expense.
- **REQ-007:** Exchange and P2P conversion of owned money preserve both currency amounts, the actual rate and fees.
- **REQ-008:** Repeated imports, receipts and chat entries combine evidence of one transaction without double counting.
- **REQ-039:** Missing rates and unsupported assets never become zero amounts or an assumed USDT/USD peg.
- **REQ-040:** Each source refreshes hourly and on demand with a visible last-success timestamp.
- **REQ-041:** History retains coverage boundaries, cursors, gaps and source status.
- **REQ-046:** Aifory Pro automatically reads RUB accounts, USDT, ETH and the existing crypto card, including these products’ movements and fees. Other products are deferred and do not block the MVP.
- **REQ-048:** Integrations and the browser collector perform authorized read operations only.
- **REQ-061:** Repeated jobs, restarts and concurrent changes cannot create duplicate financial effects.
- **REQ-065:** Account ownership, external-account owner, record author and expense attribution are distinct dimensions.
- **REQ-073:** Both manage connections; the external-account owner performs bank authentication without exposing secrets to the partner or AI.
- **REQ-076:** Household scope is checked for APIs, files, AI, jobs and external IDs independently of supplied actor/owner fields.

### Acceptance criteria

A criterion link establishes coverage; research or a partial task does not prove the entire product criterion. This task's exact outcome is specified in verification below.

#### AC-046

- **Given:** An authorized personal Aifory account provides RUB accounts, USDT/TRC-20, ETH/Ethereum and the existing USD card (D-33); other products are not connected.
- **When:** Balances/history are read again, including RUB/USDT exchange, ETH withdrawal with a fee, USDT/USD card funding and similar pending/confirmed payments with a separate fee.
- **Then:** Each included product has structured read evidence; RUB groups do not duplicate child accounts and ETH remains exact. Movements/fees count once using verified IDs/links/statuses; neither UI signs nor USDT/USD parity are assumed. Unknown linkage requires clarification. Gaps in included products block the adapter; other products are not required, but their movements through selected wallets are retained.
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

#### AC-039

- **Given:** A source contains an unsupported asset and no USDT/USD rate is available.
- **When:** A total valuation is built.
- **Then:** Raw data is retained and valuation coverage is incomplete; no hidden zero or automatic 1:1 rate is used.
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
make test-contract PROVIDER=aifory && make test-integration AREA=aifory
```

RUB, USDT, ETH and the existing card have synthetic contract scenarios and separate live readback; AIFORY-E03/E08–E14 edge cases pass after structured-contract verification. Other products are not required. AIFORY-B02–B04 and Ready are resolved before implementation.

The `make` commands are a future contract established by task-1.1; they do not exist yet. Live/paid/manual checks separately record access and actual outcomes. Research does not bypass missing-access blockers.

### Handoff to the next agent

Record changed contracts, commands/results, limitations, unresolved questions and unblocked dependencies. Update both languages and traceability. Close the task only with evidence of its outcome; GitHub Closed alone does not mean the MVP is Ready.

**Commit boundary:** this task's logical boundary; this card does not authorize commit/push/deploy, which require current user authorization.
