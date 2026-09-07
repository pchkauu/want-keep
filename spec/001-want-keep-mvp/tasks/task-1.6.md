<!-- want-keep-task: task-1.6 -->
# task-1.6 — Создать семью, членство и права на ресурсы / Implement household membership and resource permissions

## RU

Создать семью, членство и права на ресурсы.

**Состояние:** Реализованы backend/API приглашения и семейные политики; продуктовые UI/AI/файлы/уведомления остаются профильным задачам.

**Зависимости:** `task-1.4`.

**Тип:** `implementation`.

### Изменение и контракты

Закрытое присоединение создаёт отдельный passkey, сессию и личные recovery-коды атомарно с членством. Одно приглашение: 256 бит, hash, 24 часа, revision; выдача/перевыпуск требуют собственной auth до 5 минут, отзыв — активной сессии. Перевыпуск и отзыв запрещают завершение старых попыток. Оба видят семейные данные и исправляют операции; личные ресурсы защищены текущим владельцем. Управление connection отделено от банковской авторизации внешнего владельца. User/Household/Membership и actor/owner/payer остаются отдельными; лимит 2 задаётся конфигурацией.

### Границы изменений

- `backend/internal/household/`
- `backend/internal/identity/`
- `backend/internal/delivery/identity/`
- `backend/internal/storage/`
- `backend/internal/connections/`
- `backend/migrations/005_household_invitations.sql`
- `api/`
- `backend/test/integration/household/`
- `backend/test/integration/identity/`

Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

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

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

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
make check
make test-integration AREA=household
make test-integration AREA=identity
make test-integration AREA=storage
make test-household-race
make test-identity-race
make test-storage-race
```

PostgreSQL-политики и HTTP/WebAuthn приглашения проходят сценарии двух входов, replay/expiry/revoke/reissue, rollback/unknown response, гонок и семейной изоляции. Тесты новых маршрутов находятся в identity suite и используют общий криптографический fixture; household suite проверяет транзакционные права.

Доказательства и границы: evidence/task-1.6-household.md. Отсутствие изолированной БД завершает integration ошибкой. Закрытие задачи не доказывает UI/AI retrieval, файловый доступ, банковский MFA, push delivery или production.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Implement household membership and resource permissions.

**Status:** Invitation backend/API and household policies implemented; product UI/AI/files/notifications remain with their owning tasks.

**Dependencies:** `task-1.4`.

**Kind:** `implementation`.

### Change and contracts

Closed joining creates a separate passkey, session and personal recovery codes atomically with membership. One invitation: 256 bits, hash, 24 hours, revision; issue/reissue requires own authentication within 5 minutes, revocation an active session. Reissue and revocation fence older attempts. Both read family data and correct accounting; current ownership protects personal resources. Connection management is separate from bank authentication by the external owner. User/Household/Membership and actor/owner/payer remain separate; the default cap of 2 is configured.

### Change boundaries

- `backend/internal/household/`
- `backend/internal/identity/`
- `backend/internal/delivery/identity/`
- `backend/internal/storage/`
- `backend/internal/connections/`
- `backend/migrations/005_household_invitations.sql`
- `api/`
- `backend/test/integration/household/`
- `backend/test/integration/identity/`

Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

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

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

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
make check
make test-integration AREA=household
make test-integration AREA=identity
make test-integration AREA=storage
make test-household-race
make test-identity-race
make test-storage-race
```

PostgreSQL policies and HTTP/WebAuthn invitations cover independent sign-ins, replay/expiry/revoke/reissue, rollback/unknown response, races and household isolation. New route tests live in the identity suite and reuse its cryptographic fixture; the household suite verifies transactional permissions.

Evidence and boundaries: evidence/task-1.6-household.en.md. Missing isolated DB fails integration. Closing this task does not prove UI/AI retrieval, file access, bank MFA, push delivery or production.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
