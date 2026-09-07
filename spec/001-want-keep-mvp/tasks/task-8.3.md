<!-- want-keep-task: task-8.3 -->
# task-8.3 — Проверить восстановление из локальной копии / Verify recovery from a local backup

## RU

Доказать восстановление согласованного учёта и файлов до четырёх часов.

**Состояние:** Не начато; задача ожидает собственные зависимости и entry gates.

**Зависимости:** `task-8.2`.

**Тип:** `implementation`.

### Изменение и контракты

В изолированном окружении создать чистый managed PostgreSQL совместимой major-версии и восстановить только выбранный полный Mac-набор. Проверить manifest/checksums, версии схемы, необходимые keys и attachments; сравнить ledger balances, audit и незавершённые jobs. Не восстанавливать старые browser/auth sessions как активные без безопасной реавторизации; не повторять платные AI-запросы с неизвестной оплатой. Измерить RTO ≤4 часов на целевом профиле и сформировать RU/EN runbook с действиями при недоступном provider restore.

### Границы изменений

- `ops/restore/`
- `docs/operations/restore.md`

Это планируемые пути. Общие контракты: `spec/001-want-keep-mvp/contracts.md`; архитектура и команды: `constraints.md`. Менять только владельца поведения и затронутые тесты; при незакрытом контракте обновить evidence и остановить зависимую реализацию.

### Связанные требования

- **REQ-004:** Начало учёта задаётся датой; начальные остатки отделены от доходов и расходов.
- **REQ-012:** Исправление учёта сохраняет оригинал, автора, основание, версию и возможность отмены решения.
- **REQ-049:** Каждый участник входит со своими passkey и одноразовыми кодами восстановления; сброс чужого входа партнёром недоступен.
- **REQ-050:** Файлы, ключи источников, сессии и финансовые журналы защищены от постороннего доступа.
- **REQ-057:** Зашифрованная резервная копия выгружается на MacBook ежечасно при его доступности; восстановление проверяется.
- **REQ-058:** Операционные статусы показывают ошибки импорта, AI, курсов, резервирования и расходы без утечки финансового содержимого.
- **REQ-061:** Повторные задания, перезапуски и параллельные изменения не создают двойных финансовых эффектов.
- **REQ-075:** Восстановление данных сохраняет пользователей, членство, принадлежность, роли, историю и общий семейный учёт.
- **REQ-076:** Семейная область проверяется для API, файлов, AI, фоновых задач и внешних ID независимо от присланных actor/owner.

### Критерии приёмки

Связь с критерием задаёт покрытие; исследование или частичная задача не доказывает весь критерий продукта. Точный результат этой задачи указан ниже в проверке.

#### AC-004

- **Дано:** История запрошена с 1 августа; начальный остаток RUB 5 000 подтверждён.
- **Когда:** Импортируется расход RUB 500 от 2 августа.
- **Тогда:** Остаток равен RUB 4 500; доход августа не увеличивается на начальные RUB 5 000; неподтверждённое начало обозначается явно.
- **Уровень:** `integration`.

#### AC-012

- **Дано:** AI ошибочно связал две операции; исходные импортированные записи сохранены.
- **Когда:** Владелец отменяет связь и исправляет категорию.
- **Тогда:** Пересчитаны производные отчёты; видна история; повторный импорт не стирает правку владельца.
- **Уровень:** `integration`.

#### AC-049

- **Дано:** Оба участника зарегистрировали собственные passkey и личные коды восстановления.
- **Когда:** Участник восстанавливает свой вход, повторяет код, пробует чужой origin и сброс входа партнёра.
- **Тогда:** Свой вход восстановлен с отзывом своих старых сессий/подписок; сессии партнёра сохранены; повтор кода, чужой origin и сброс чужого входа отклонены.
- **Уровень:** `end-to-end`.

#### AC-050

- **Дано:** Существует приватный чек и активное подключение источника.
- **Когда:** Проверяются прямой URL файла, экспорт без сессии, логи и отзыв подключения.
- **Тогда:** Без авторизации доступ закрыт; секреты зашифрованы и не журналируются; отзыв подключения прекращает дальнейший сбор.
- **Уровень:** `integration`.

#### AC-057

- **Дано:** Есть база, вложения и MacBook, который временно недоступен.
- **Когда:** Создаются копии, Mac возвращается в сеть, затем проводится восстановление.
- **Тогда:** Показан возраст последней полной копии; после возвращения копирование возобновляется; восстановлены согласованные данные и вложения до четырёх часов; часовой RPO заявляется только при доступном Mac.
- **Уровень:** `integration+manual`.

#### AC-058

- **Дано:** Сломан один коннектор, задержан AI и устарела копия.
- **Когда:** Открывается состояние системы и читаются диагностические логи.
- **Тогда:** Видны отдельные проблемы и действия восстановления; логи содержат идентификаторы/коды, а не чеки, ключи или тексты финансовых сообщений.
- **Уровень:** `integration`.

#### AC-061

- **Дано:** Процесс падает между сохранением записи и подтверждением задания.
- **Когда:** Задание повторяется, одновременно приходит правка владельца.
- **Тогда:** Применён один эффект, правка защищена версией, незавершённое состояние восстанавливается; внешняя неоднозначность не вызывает слепой повтор.
- **Уровень:** `integration`.

#### AC-073

- **Дано:** MacBook недоступен 10 часов; последующая загрузка вложений обрывается.
- **Когда:** Проверяется статус копии и выполняется восстановление последнего полного набора.
- **Тогда:** Не заявляется часовой RPO; неполный набор не помечен успешным; последний полный набор восстановим и имеет проверяемый manifest.
- **Уровень:** `integration+manual`.

#### AC-089

- **Дано:** Копия содержит двух участников, личные/общие цели и операции с разными авторами.
- **Когда:** Полный набор восстанавливается на изолированном сервере.
- **Тогда:** Суммы, связи и права обоих сохранены; входы не объединены, банковские сессии автоматически не оживают. Восстановление укладывается в измеренную цель RTO.
- **Уровень:** `integration+manual`.

#### AC-090

- **Дано:** В тестах созданы две изолированные семьи; запрос или задача подменяет householdId/actor/resourceId.
- **Когда:** Проверяются чтение файла, импорт, исправление, поиск AI и дедупликация.
- **Тогда:** Чужие объекты недоступны и не объединяются; сервер берёт principal из сессии или проверенного контекста задания. Отказ не раскрывает чужое содержимое.
- **Уровень:** `integration`.

### Проверка результата

```sh
make restore-check MODE=synthetic
```

Восстановление укладывается в четыре часа на целевом профиле, данные/файлы согласованы; реальный доступ владельца проверяется отдельно.

Основа task-1.1 уже предоставляет `make restore-check` как fail-fast интерфейс. До реализации task-8.3 suite обязана завершаться понятной ошибкой; RTO ≤4 часов подтверждается только измеренным восстановлением полного набора в чистую среду.

### Передача следующему агенту

Записать изменённые контракты, команды и результаты, ограничения, незакрытые вопросы и разблокированные зависимости. Обновить обе языковые версии и трассировку. Закрывать задачу только по доказательству её результата; GitHub Closed само по себе не означает Ready MVP.

**Commit boundary:** логическая граница этой задачи; commit/push/deploy не разрешены данной карточкой и требуют действующей авторизации пользователя.

## EN

Demonstrate consistent accounting/file recovery within four hours.

**Status:** Not started; the task awaits its own dependencies and entry gates.

**Dependencies:** `task-8.2`.

**Kind:** `implementation`.

### Change and contracts

In an isolated environment create a clean managed PostgreSQL instance on a compatible major and restore only a selected complete Mac set. Verify manifest/checksums, schema versions, required keys and attachments; compare ledger balances, audit and pending jobs. Do not revive old browser/auth sessions without safe reauthorization or replay AI calls with unknown charges. Measure RTO ≤4 hours on the target profile and write an RU/EN runbook including provider-restore unavailability.

### Change boundaries

- `ops/restore/`
- `docs/operations/restore.md`

These are planned paths. Shared contracts: `spec/001-want-keep-mvp/contracts.en.md`; architecture and commands: `constraints.en.md`. Change only the behavior owner and affected tests; an unresolved contract requires updated evidence and stops dependent implementation.

### Linked requirements

- **REQ-004:** Accounting starts on a selected date; opening balances are separate from income and expenses.
- **REQ-012:** Accounting corrections preserve the original, actor, reason, version and ability to undo a decision.
- **REQ-049:** Each member signs in with their own passkeys and one-time recovery codes; partner-assisted reset is unavailable.
- **REQ-050:** Files, source keys, sessions and financial records are protected against unauthorized access.
- **REQ-057:** An encrypted backup is pulled to the MacBook hourly while reachable; recovery is tested.
- **REQ-058:** Operational status exposes import, AI, FX, backup failures and spend without leaking financial content.
- **REQ-061:** Repeated jobs, restarts and concurrent changes cannot create duplicate financial effects.
- **REQ-075:** Data recovery preserves users, memberships, ownership, roles, history and shared household accounting.
- **REQ-076:** Household scope is checked for APIs, files, AI, jobs and external IDs independently of supplied actor/owner fields.

### Acceptance criteria

A criterion link establishes coverage; research or a partial task does not prove the entire product criterion. This task's exact outcome is specified in verification below.

#### AC-004

- **Given:** History is requested from August 1; an opening RUB 5,000 balance is confirmed.
- **When:** A RUB 500 expense dated August 2 is imported.
- **Then:** Balance is RUB 4,500; August income excludes the opening RUB 5,000; an unverified opening is explicit.
- **Level:** `integration`.

#### AC-012

- **Given:** AI incorrectly linked two transactions; original imports are retained.
- **When:** The owner unlinks them and corrects the category.
- **Then:** Derived reports are recalculated, history is visible and reimport does not overwrite the owner's correction.
- **Level:** `integration`.

#### AC-049

- **Given:** Both members enrolled their own passkeys and personal recovery codes.
- **When:** A member recovers their sign-in, reuses a code, tries an alien origin and attempts to reset their partner’s sign-in.
- **Then:** Own access is restored with own old sessions/subscriptions revoked; the partner’s sessions remain; code reuse, alien origins and resetting the partner’s sign-in fail.
- **Level:** `end-to-end`.

#### AC-050

- **Given:** A private receipt and an active source connection exist.
- **When:** A direct file URL, unauthenticated export, logs and disconnection are checked.
- **Then:** Unauthenticated access fails; secrets are encrypted and not logged; disconnecting stops further collection.
- **Level:** `integration`.

#### AC-057

- **Given:** The database, attachments and a temporarily unreachable MacBook exist.
- **When:** Backups are attempted, the Mac reconnects and recovery is rehearsed.
- **Then:** Last complete backup age is visible; copying resumes after reconnection; consistent data and attachments restore within four hours; hourly RPO is claimed only while the Mac is reachable.
- **Level:** `integration+manual`.

#### AC-058

- **Given:** A connector fails, AI is delayed and a backup is stale.
- **When:** System health and diagnostic logs are inspected.
- **Then:** Separate failures and recovery actions are visible; logs contain identifiers/codes, not receipts, keys or financial message text.
- **Level:** `integration`.

#### AC-061

- **Given:** A process crashes between persisting a record and acknowledging its job.
- **When:** The job is retried while the owner submits a correction.
- **Then:** One effect is applied, the correction is version-protected and incomplete state recovers; an ambiguous external outcome is not blindly retried.
- **Level:** `integration`.

#### AC-073

- **Given:** The MacBook is unreachable for 10 hours and the subsequent attachment download is interrupted.
- **When:** Backup status is inspected and the last complete set is restored.
- **Then:** Hourly RPO is not claimed; the partial set is not marked successful; the last complete set is recoverable with a verifiable manifest.
- **Level:** `integration+manual`.

#### AC-089

- **Given:** A backup contains two members, personal/joint goals and transactions by different authors.
- **When:** The complete set is restored onto an isolated server.
- **Then:** Amounts, relationships and both users’ permissions are preserved; sign-ins are not merged and bank sessions do not revive automatically. Recovery meets the measured RTO target.
- **Level:** `integration+manual`.

#### AC-090

- **Given:** Tests contain two isolated households; a request or job forges householdId/actor/resourceId.
- **When:** File reads, import, correction, AI retrieval and deduplication are exercised.
- **Then:** Foreign objects are inaccessible and never merged; the server takes principal from the session or validated job context. Denial reveals no foreign content.
- **Level:** `integration`.

### Verification

```sh
make restore-check MODE=synthetic
```

Recovery completes within four hours on the target profile with consistent data/files; real owner access is checked separately.

The task-1.1 foundation already provides `make restore-check` as a fail-fast interface. Until task-8.3 implements it, the suite must fail clearly; RTO ≤4 hours is established only by a measured restore of a complete set into a clean environment.

### Handoff to the next agent

Record changed contracts, commands/results, limitations, unresolved questions and unblocked dependencies. Update both languages and traceability. Close the task only with evidence of its outcome; GitHub Closed alone does not mean the MVP is Ready.

**Commit boundary:** this task's logical boundary; this card does not authorize commit/push/deploy, which require current user authorization.
