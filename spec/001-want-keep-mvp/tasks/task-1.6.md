<!-- want-keep-task: task-1.6 -->
# task-1.6 — Создать семью, членство и права на ресурсы / Implement household membership and resource permissions

## RU

Создать семью, членство и права на ресурсы.

**Состояние:** Заблокировано зависимостями и проверкой SDD Ready; реализация не начата.

**Зависимости:** `task-1.4`.

**Тип:** `implementation`.

### Изменение и контракты

Создать User/Household/Membership и доменную policy границы по contracts: оба читают/правят операции и подключения, личные планы/цели — владелец. Bootstrap и одноразовое ограниченное приглашение создают отдельные входы; лимит 2 из конфигурации. Не добавлять выход, смену партнёра или конструктор ролей. Проверять принадлежность и ревизию атомарно; API principal не берётся из payload.

### Границы изменений

- `backend/internal/household/`
- `backend/internal/auth/`
- `backend/migrations/`
- `api/openapi.yaml`

Это планируемые пути. Общие контракты: `spec/001-want-keep-mvp/contracts.md`; архитектура и команды: `constraints.md`. Менять только владельца поведения и затронутые тесты; при незакрытом контракте обновить evidence и остановить зависимую реализацию.

### Связанные требования

- **REQ-001:** Семейный пилот обслуживает двух участников с раздельным входом и закрытым присоединением; публичной регистрации нет.
- **REQ-063:** Пользователь, семья и членство моделируются отдельно; ограничение двух участников задаётся конфигурацией.
- **REQ-064:** Оба участника видят все финансовые данные и изменяют операции; личные цели и части плана изменяет только их владелец.
- **REQ-065:** Принадлежность счёта, владелец внешнего аккаунта, автор записи и принадлежность расхода являются отдельными признаками.
- **REQ-072:** Конкурирующие изменения, ответы на уточнения и отмены проверяют версию и текущие права, сохраняя обоих авторов.
- **REQ-073:** Оба управляют подключениями; банковскую авторизацию выполняет владелец внешнего аккаунта без раскрытия секретов партнёру или AI.
- **REQ-074:** Изменения плана и целей уведомляют второго участника; прочтение и push-подписки принадлежат конкретному пользователю.
- **REQ-076:** Семейная область проверяется для API, файлов, AI, фоновых задач и внешних ID независимо от присланных actor/owner.

### Критерии приёмки

Связь с критерием задаёт покрытие; исследование или частичная задача не доказывает весь критерий продукта. Точный результат этой задачи указан ниже в проверке.

#### AC-001

- **Дано:** Создана семья, первый участник вошёл, лимит активных участников равен 2.
- **Когда:** Второй участник принимает приглашение; посторонний пробует открытый вход, повтор приглашения и присоединение сверх лимита.
- **Тогда:** Приглашение создаёт отдельное членство один раз; посторонний не получает данных, повтор и превышение лимита отклонены.
- **Уровень:** `end-to-end`.

#### AC-077

- **Дано:** Два пользователя состоят в одной семье.
- **Когда:** Проверяются схема, авторизация и ограничение членства.
- **Тогда:** Нет полей partner1/partner2 и ветвлений по конкретным пользователям; роли и принадлежность отделены от личности. Выход, замена и новые роли не реализованы.
- **Уровень:** `integration`.

#### AC-078

- **Дано:** У A есть личная цель и статья плана; у семьи общая статья и операции обоих.
- **Когда:** B читает все данные, исправляет операцию A и общий план, затем пытается изменить личную цель/план A через API и AI.
- **Тогда:** Чтение, операции и общее изменение разрешены; личные план/цель A защищены сервером. Одного уполномоченного подтверждения достаточно, второй уведомлён.
- **Уровень:** `end-to-end`.

#### AC-086

- **Дано:** A и B открыли одну версию операции или уточнения.
- **Когда:** Оба отправляют несовместимые изменения и повторяют один запрос.
- **Тогда:** Один результат применяется; второй получает конфликт с необходимостью перечитать состояние. Повтор не дублирует эффект; отмена создаёт новую проверенную revision и не стирает чужую последующую правку.
- **Уровень:** `integration`.

#### AC-087

- **Дано:** A владеет внешним аккаунтом, B инициирует повторную авторизацию или отключение.
- **Когда:** Запрашивается MFA; одновременно завершает работу старое задание синхронизации.
- **Тогда:** MFA адресован A; B видит статус, но не пароль/код/сессию. Отключение отзывает lease/version и запрещает применение старого результата; реальные платежи недоступны обоим.
- **Уровень:** `integration`.

#### AC-088

- **Дано:** Оба имеют уведомления и устройства, A изменяет разрешённую цель.
- **Когда:** B читает уведомление; A восстанавливает вход и отзывает своё устройство.
- **Тогда:** Событие доставлено B один раз; read-state и отзыв устройства одного не меняют подписки другого. Уточнения личного плана направлены его владельцу, остальные доступны обоим.
- **Уровень:** `end-to-end`.

#### AC-090

- **Дано:** В тестах созданы две изолированные семьи; запрос или задача подменяет householdId/actor/resourceId.
- **Когда:** Проверяются чтение файла, импорт, исправление, поиск AI и дедупликация.
- **Тогда:** Чужие объекты недоступны и не объединяются; сервер берёт principal из сессии или проверенного контекста задания. Отказ не раскрывает чужое содержимое.
- **Уровень:** `integration`.

#### AC-105

- **Дано:** Есть личный счёт A, семейный счёт и запись покупки; B видит их и может исправлять учёт.
- **Когда:** B исправляет покупку и пытается через FORM-03/API изменить владельца или scope счёта A; затем A меняет свой счёт и B меняет семейный.
- **Тогда:** Исправление покупки разрешено, изменение принадлежности личного счёта B отклоняется сервером. Действия текущего владельца и изменение семейного счёта разрешены с revision/audit. Внешний владелец и история операций не изменены, фильтр участника не даёт дополнительных прав.
- **Уровень:** `integration+e2e`.

### Проверка результата

```sh
make test-go PKG=./internal/household/...
make test-integration AREA=household
make check-contracts
```

Проверены два отдельных входа, повтор/истечение приглашения, лимит, разрешённые/запрещённые действия, подмена actor, чужие family IDs и конкурентная запись.

Команды `make` — будущий контракт, создаваемый task-1.1; сейчас они не существуют. Live/paid/manual проверки отдельно фиксируют доступ и фактический результат. Исследования не обходят блокер отсутствующего доступа.

### Передача следующему агенту

Записать изменённые контракты, команды и результаты, ограничения, незакрытые вопросы и разблокированные зависимости. Обновить обе языковые версии и трассировку. Закрывать задачу только по доказательству её результата; GitHub Closed само по себе не означает Ready MVP.

**Commit boundary:** логическая граница этой задачи; commit/push/deploy не разрешены данной карточкой и требуют действующей авторизации пользователя.

## EN

Implement household membership and resource permissions.

**Status:** Blocked by dependencies and the SDD Ready gate; implementation has not started.

**Dependencies:** `task-1.4`.

**Kind:** `implementation`.

### Change and contracts

Create User/Household/Membership and domain boundary policy per contracts: both read/edit transactions and connections; personal plans/goals are owner-editable. Bootstrap and a restricted single-use invitation create separate sign-ins; limit 2 comes from configuration. Add no exit, partner replacement or role builder. Check scope and revision atomically; API principal never comes from payload.

### Change boundaries

- `backend/internal/household/`
- `backend/internal/auth/`
- `backend/migrations/`
- `api/openapi.yaml`

These are planned paths. Shared contracts: `spec/001-want-keep-mvp/contracts.en.md`; architecture and commands: `constraints.en.md`. Change only the behavior owner and affected tests; an unresolved contract requires updated evidence and stops dependent implementation.

### Linked requirements

- **REQ-001:** The family pilot serves two members with separate sign-in and restricted joining; public registration is unavailable.
- **REQ-063:** User, household and membership are separate models; the two-member limit is configured.
- **REQ-064:** Both members see all financial data and edit transactions; only the owner edits personal goals and plan portions.
- **REQ-065:** Account ownership, external-account owner, record author and expense attribution are distinct dimensions.
- **REQ-072:** Competing edits, clarification answers and reversals check revision and current permissions while retaining both authors.
- **REQ-073:** Both manage connections; the external-account owner performs bank authentication without exposing secrets to the partner or AI.
- **REQ-074:** Plan and goal changes notify the other member; read state and push subscriptions belong to the individual user.
- **REQ-076:** Household scope is checked for APIs, files, AI, jobs and external IDs independently of supplied actor/owner fields.

### Acceptance criteria

A criterion link establishes coverage; research or a partial task does not prove the entire product criterion. This task's exact outcome is specified in verification below.

#### AC-001

- **Given:** A household exists, the first member is signed in and the active-member limit is 2.
- **When:** The second member accepts an invitation; an outsider attempts open registration, invitation replay and joining beyond the limit.
- **Then:** The invitation creates one separate membership; outsiders receive no data and replay or exceeding the limit is rejected.
- **Level:** `end-to-end`.

#### AC-077

- **Given:** Two users belong to one household.
- **When:** Schema, authorization and the membership limit are inspected.
- **Then:** There are no partner1/partner2 fields or specific-user branches; roles and ownership are separate from identity. Exit, replacement and new roles are not implemented.
- **Level:** `integration`.

#### AC-078

- **Given:** A has a personal goal and plan line; the household has a joint line and both members’ transactions.
- **When:** B reads all data, edits A’s transaction and the joint plan, then attempts to change A’s personal goal/plan through API and AI.
- **Then:** Reads, transaction edits and joint changes succeed; A’s personal plan/goal are protected server-side. One authorized confirmation suffices and the other member is notified.
- **Level:** `end-to-end`.

#### AC-086

- **Given:** A and B opened the same transaction or clarification revision.
- **When:** Both submit conflicting edits and replay one request.
- **Then:** One result applies; the other receives a conflict requiring refresh. Replay does not duplicate effects; reversal creates a checked new revision without erasing the other member’s later edit.
- **Level:** `integration`.

#### AC-087

- **Given:** A owns the external account and B initiates reauthorization or disconnect.
- **When:** MFA is requested while an old sync job completes.
- **Then:** MFA is addressed to A; B sees status but no password/code/session. Disconnect revokes lease/version and prevents stale-result application; actual payments are unavailable to both.
- **Level:** `integration`.

#### AC-088

- **Given:** Both have notifications and devices and A changes an authorized goal.
- **When:** B reads the notification; A recovers sign-in and revokes their device.
- **Then:** B receives the event once; one member’s read state and device revocation do not alter the other’s subscriptions. Personal-plan clarifications target its owner; other clarifications are open to both.
- **Level:** `end-to-end`.

#### AC-090

- **Given:** Tests contain two isolated households; a request or job forges householdId/actor/resourceId.
- **When:** File reads, import, correction, AI retrieval and deduplication are exercised.
- **Then:** Foreign objects are inaccessible and never merged; the server takes principal from the session or validated job context. Denial reveals no foreign content.
- **Level:** `integration`.

#### AC-105

- **Given:** There is A’s personal account, a household account and a purchase; B can read them and correct accounting.
- **When:** B corrects the purchase and uses FORM-03/API to change A’s account owner/scope; A then changes their own account and B changes the household account.
- **Then:** Purchase correction succeeds; B’s personal-account ownership change is rejected server-side. Current-owner actions and household-account changes succeed with revision/audit. External ownership and transaction history remain unchanged; member filter grants no extra authority.
- **Level:** `integration+e2e`.

### Verification

```sh
make test-go PKG=./internal/household/...
make test-integration AREA=household
make check-contracts
```

Two separate sign-ins, invitation replay/expiry, limit, allowed/denied actions, actor spoofing, foreign household IDs and concurrent writes are covered.

The `make` commands are a future contract established by task-1.1; they do not exist yet. Live/paid/manual checks separately record access and actual outcomes. Research does not bypass missing-access blockers.

### Handoff to the next agent

Record changed contracts, commands/results, limitations, unresolved questions and unblocked dependencies. Update both languages and traceability. Close the task only with evidence of its outcome; GitHub Closed alone does not mean the MVP is Ready.

**Commit boundary:** this task's logical boundary; this card does not authorize commit/push/deploy, which require current user authorization.
