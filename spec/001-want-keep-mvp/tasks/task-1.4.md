<!-- want-keep-task: task-1.4 -->
# task-1.4 — Реализовать passkey и восстановление доступа / Implement passkeys and access recovery

## RU

Обеспечить единственный безопасный вход владельца.

**Состояние:** Заблокировано зависимостями и проверкой SDD Ready; реализация не начата.

**Зависимости:** `task-1.3`.

**Тип:** `implementation`.

### Изменение и контракты

Реализовать одноразовую первичную регистрацию владельца через операторский bootstrap, WebAuthn challenge/origin/RP проверки, защищённые cookie-сессии, logout и одноразовые хешированные recovery-коды. Восстановление отзывает старые сессии и подписки устройств. Исключить публичную регистрацию и повтор bootstrap после инициализации.

### Границы изменений

- `backend/internal/identity/`
- `backend/internal/delivery/identity/`

Это планируемые пути. Общие контракты: `spec/001-want-keep-mvp/contracts.md`; архитектура и команды: `constraints.md`. Менять только владельца поведения и затронутые тесты; при незакрытом контракте обновить evidence и остановить зависимую реализацию.

### Связанные требования

- **REQ-001:** Семейный пилот обслуживает двух участников с раздельным входом и закрытым присоединением; публичной регистрации нет.
- **REQ-049:** Каждый участник входит со своими passkey и одноразовыми кодами восстановления; сброс чужого входа партнёром недоступен.
- **REQ-050:** Файлы, ключи источников, сессии и финансовые журналы защищены от постороннего доступа.
- **REQ-053:** Напоминания и сводки доступны внутри приложения и через разрешённый web-push.
- **REQ-074:** Изменения плана и целей уведомляют второго участника; прочтение и push-подписки принадлежат конкретному пользователю.

### Критерии приёмки

Связь с критерием задаёт покрытие; исследование или частичная задача не доказывает весь критерий продукта. Точный результат этой задачи указан ниже в проверке.

#### AC-001

- **Дано:** Создана семья, первый участник вошёл, лимит активных участников равен 2.
- **Когда:** Второй участник принимает приглашение; посторонний пробует открытый вход, повтор приглашения и присоединение сверх лимита.
- **Тогда:** Приглашение создаёт отдельное членство один раз; посторонний не получает данных, повтор и превышение лимита отклонены.
- **Уровень:** `end-to-end`.

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
make test-integration AREA=identity
```

Вход и восстановление проходят; чужой origin, повтор challenge/recovery-кода и неавторизованный доступ отклонены.

Команды `make` — будущий контракт, создаваемый task-1.1; сейчас они не существуют. Live/paid/manual проверки отдельно фиксируют доступ и фактический результат. Исследования не обходят блокер отсутствующего доступа.

### Передача следующему агенту

Записать изменённые контракты, команды и результаты, ограничения, незакрытые вопросы и разблокированные зависимости. Обновить обе языковые версии и трассировку. Закрывать задачу только по доказательству её результата; GitHub Closed само по себе не означает Ready MVP.

**Commit boundary:** логическая граница этой задачи; commit/push/deploy не разрешены данной карточкой и требуют действующей авторизации пользователя.

## EN

Provide a secure single-owner entry point.

**Status:** Blocked by dependencies and the SDD Ready gate; implementation has not started.

**Dependencies:** `task-1.3`.

**Kind:** `implementation`.

### Change and contracts

Implement one-time owner enrollment through operator bootstrap, WebAuthn challenge/origin/RP checks, secure cookie sessions, logout and one-time hashed recovery codes. Recovery revokes old sessions and device subscriptions. Prevent public registration and repeated bootstrap after initialization.

### Change boundaries

- `backend/internal/identity/`
- `backend/internal/delivery/identity/`

These are planned paths. Shared contracts: `spec/001-want-keep-mvp/contracts.en.md`; architecture and commands: `constraints.en.md`. Change only the behavior owner and affected tests; an unresolved contract requires updated evidence and stops dependent implementation.

### Linked requirements

- **REQ-001:** The family pilot serves two members with separate sign-in and restricted joining; public registration is unavailable.
- **REQ-049:** Each member signs in with their own passkeys and one-time recovery codes; partner-assisted reset is unavailable.
- **REQ-050:** Files, source keys, sessions and financial records are protected against unauthorized access.
- **REQ-053:** Reminders and summaries are available in-app and through authorized web push.
- **REQ-074:** Plan and goal changes notify the other member; read state and push subscriptions belong to the individual user.

### Acceptance criteria

A criterion link establishes coverage; research or a partial task does not prove the entire product criterion. This task's exact outcome is specified in verification below.

#### AC-001

- **Given:** A household exists, the first member is signed in and the active-member limit is 2.
- **When:** The second member accepts an invitation; an outsider attempts open registration, invitation replay and joining beyond the limit.
- **Then:** The invitation creates one separate membership; outsiders receive no data and replay or exceeding the limit is rejected.
- **Level:** `end-to-end`.

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
make test-integration AREA=identity
```

Sign-in and recovery pass; alien origin, challenge/recovery-code replay and unauthorized access fail.

The `make` commands are a future contract established by task-1.1; they do not exist yet. Live/paid/manual checks separately record access and actual outcomes. Research does not bypass missing-access blockers.

### Handoff to the next agent

Record changed contracts, commands/results, limitations, unresolved questions and unblocked dependencies. Update both languages and traceability. Close the task only with evidence of its outcome; GitHub Closed alone does not mean the MVP is Ready.

**Commit boundary:** this task's logical boundary; this card does not authorize commit/push/deploy, which require current user authorization.
