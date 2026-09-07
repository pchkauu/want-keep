<!-- want-keep-task: task-1.4 -->
# task-1.4 — Реализовать passkey и восстановление доступа / Implement passkeys and access recovery

## RU

Обеспечить раздельный безопасный вход участников семьи и личное восстановление доступа.

**Состояние:** Реализована backend/API-основа; продуктовые экраны, приглашения и browser/manual AC остаются профильным задачам.

**Зависимости:** `task-1.3`.

**Тип:** `implementation`.

### Изменение и контракты

Операторский bootstrap однократен. WebAuthn v0.18.0 проверяет origin/RP/challenge/purpose/UV и подпись. Cookie-сессия живёт 12 часов и 30 минут простоя; фоновые запросы не продлевают её. Изменения ключей/кодов требуют собственной auth не старше 5 минут. Recovery выдаёт 10 одноразовых хешированных кодов; после нового passkey атомарно отзывает прежние ключи, сессии, коды и подписки этого пользователя, сохраняя доступ партнёра. PostgreSQL хранит попытки, generation, rate limits и audit. Приглашения — task-1.6, UI — task-7.1.

### Границы изменений

- `backend/internal/identity/`
- `backend/internal/delivery/identity/`
- `backend/internal/storage/`
- `backend/migrations/004_identity.sql`
- `backend/cmd/`
- `api/`
- `backend/test/integration/identity/`

Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-001:** Семейный пилот обслуживает двух участников с раздельным входом и закрытым присоединением; публичной регистрации нет.
- **REQ-049:** Каждый участник входит со своими passkey и одноразовыми кодами восстановления; сброс чужого входа партнёром недоступен.
- **REQ-050:** Файлы, ключи источников, сессии и финансовые журналы защищены от постороннего доступа.
- **REQ-053:** Напоминания и сводки доступны внутри приложения и через разрешённый web-push.
- **REQ-074:** Изменения плана и целей уведомляют второго участника; прочтение и push-подписки принадлежат конкретному пользователю.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

#### AC-001

- **Дано:** Создана семья, первый участник вошёл, лимит активных участников равен 2.
- **Когда:** Второй участник принимает приглашение; посторонний пробует открытый вход, повтор приглашения и присоединение сверх лимита.
- **Тогда:** Приглашение создаёт отдельное членство один раз; посторонний не получает данных, повтор и превышение лимита отклонены.
- **Уровень:** `end-to-end`.

#### AC-049

- **Дано:** Оба участника зарегистрировали собственные passkey и личные коды восстановления.
- **Когда:** Участник восстанавливает свой вход, повторяет код, пробует чужой origin и сброс входа партнёра.
- **Тогда:** Свой вход восстановлен после нового passkey с атомарным отзывом своих старых ключей, recovery-кодов, сессий и подписок; доступ партнёра сохранён. Повтор кода, чужой origin и сброс чужого входа отклонены.
- **Уровень:** `end-to-end`.

#### AC-050

- **Дано:** Существует приватный чек и активное подключение источника.
- **Когда:** Проверяются прямой URL файла, экспорт без сессии, логи и отзыв подключения.
- **Тогда:** Без авторизации доступ закрыт; секреты зашифрованы и не журналируются; отзыв подключения прекращает дальнейший сбор.
- **Уровень:** `integration`.

#### AC-072

- **Дано:** Активны две сессии и push-подписка.
- **Когда:** Владелец восстанавливает доступ и отзывает старое устройство.
- **Тогда:** Старые сессии/привязанные подписки отозваны; ссылка из push требует действующей авторизации; финансовых деталей в push по умолчанию нет.
- **Уровень:** `end-to-end+manual`.

#### AC-088

- **Дано:** Оба имеют уведомления и устройства, A изменяет разрешённую цель.
- **Когда:** B читает уведомление; A восстанавливает вход и отзывает своё устройство.
- **Тогда:** Событие доставлено B один раз; read-state и отзыв устройства одного не меняют подписки другого. Уточнения личного плана направлены его владельцу, остальные доступны обоим.
- **Уровень:** `end-to-end`.

### Проверка результата

```sh
make check
make test-integration AREA=identity
make test-integration AREA=storage
make test-identity-race
make test-storage-race
```

Вход и восстановление проходят; чужой origin, повтор challenge/recovery-кода и неавторизованный доступ отклонены.

HTTP/PostgreSQL и криптографические fixtures проверяются в изоляции; отсутствие БД — ошибка. Доказательства и ограничения: evidence/task-1.4-identity.md. Реальные Chrome/Arc, Touch ID, push delivery, приглашения, banking и production не заявляются пройденными.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Provide independent secure sign-in and personal access recovery for household members.

**Status:** Backend/API foundation implemented; product screens, invitations and browser/manual AC remain with their owning tasks.

**Dependencies:** `task-1.3`.

**Kind:** `implementation`.

### Change and contracts

Operator bootstrap is one-time. WebAuthn v0.18.0 verifies origin/RP/challenge/purpose/UV and signatures. Cookie sessions expire after 12 hours or 30 idle minutes; background requests do not extend them. Key/code changes require own authentication within 5 minutes. Recovery issues 10 hashed one-use codes; after a new passkey it atomically revokes that user’s old keys, sessions, codes and subscriptions while preserving the partner’s access. PostgreSQL persists attempts, generation, rate limits and audit. Invitations belong to task-1.6; UI belongs to task-7.1.

### Change boundaries

- `backend/internal/identity/`
- `backend/internal/delivery/identity/`
- `backend/internal/storage/`
- `backend/migrations/004_identity.sql`
- `backend/cmd/`
- `api/`
- `backend/test/integration/identity/`

Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-001:** The family pilot serves two members with separate sign-in and restricted joining; public registration is unavailable.
- **REQ-049:** Each member signs in with their own passkeys and one-time recovery codes; partner-assisted reset is unavailable.
- **REQ-050:** Files, source keys, sessions and financial records are protected against unauthorized access.
- **REQ-053:** Reminders and summaries are available in-app and through authorized web push.
- **REQ-074:** Plan and goal changes notify the other member; read state and push subscriptions belong to the individual user.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

#### AC-001

- **Given:** A household exists, the first member is signed in and the active-member limit is 2.
- **When:** The second member accepts an invitation; an outsider attempts open registration, invitation replay and joining beyond the limit.
- **Then:** The invitation creates one separate membership; outsiders receive no data and replay or exceeding the limit is rejected.
- **Level:** `end-to-end`.

#### AC-049

- **Given:** Both members enrolled their own passkeys and personal recovery codes.
- **When:** A member recovers their sign-in, reuses a code, tries an alien origin and attempts to reset their partner’s sign-in.
- **Then:** Own access is restored after a new passkey with atomic revocation of own old keys, recovery codes, sessions and subscriptions; the partner’s access remains. Code reuse, alien origin and resetting the partner’s sign-in fail.
- **Level:** `end-to-end`.

#### AC-050

- **Given:** A private receipt and an active source connection exist.
- **When:** A direct file URL, unauthenticated export, logs and disconnection are checked.
- **Then:** Unauthenticated access fails; secrets are encrypted and not logged; disconnecting stops further collection.
- **Level:** `integration`.

#### AC-072

- **Given:** Two sessions and a push subscription are active.
- **When:** The owner recovers access and revokes an old device.
- **Then:** Old sessions/associated subscriptions are revoked; push links require current authorization; push contains no financial details by default.
- **Level:** `end-to-end+manual`.

#### AC-088

- **Given:** Both have notifications and devices and A changes an authorized goal.
- **When:** B reads the notification; A recovers sign-in and revokes their device.
- **Then:** B receives the event once; one member’s read state and device revocation do not alter the other’s subscriptions. Personal-plan clarifications target its owner; other clarifications are open to both.
- **Level:** `end-to-end`.

### Verification

```sh
make check
make test-integration AREA=identity
make test-integration AREA=storage
make test-identity-race
make test-storage-race
```

Sign-in and recovery pass; alien origin, challenge/recovery-code replay and unauthorized access fail.

HTTP/PostgreSQL and cryptographic fixtures run in isolation; missing DB fails. Evidence and limits: evidence/task-1.4-identity.en.md. Real Chrome/Arc, Touch ID, push delivery, invitations, banking and production are not claimed as passed.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
