# Task-1.4 — passkey и восстановление доступа

Реализована backend/API-основа раздельного входа участников: WebAuthn, операторский bootstrap, cookie-сессии и recovery. База — `2ff2dcd2e7a10732e49b4db5df4c92f824167e1e`; ветка — `feat/task-1.4-passkeys-recovery`. Правила и сроки определяет D-45 в [контракте](../contracts.md).

## Реализация

WebAuthn v0.18.0 проверяет ES256/RS256, UV, challenge, RP/origin и browser/purpose binding. Библиотека скрыта во внешнем адаптере; domain/application не импортируют её типы. PostgreSQL migration 004 хранит profiles, credentials, attempts/grants, sessions, recovery hashes, subscription bindings, rate limits и неизменяемый audit. Миграции 001–003 сохранены.

Bootstrap/recovery завершаются атомарно. Generation и проверка отозванного ключа исключают применение прежнего доступа. Savepoint откатывает частичный эффект отказавшей церемонии. Сессии имеют предел 12 часов и 30 минут простоя; обновление ключей/кодов требует собственной auth до 5 минут. Добавление запасного ключа не перевыпускает коды. Recovery отзывает только ресурсы восстанавливаемого пользователя.

## Запуск и передача эксплуатации

Из каталога backend: `go run ./cmd/migrate`, `go run ./cmd/identity-bootstrap --output /private/path/bootstrap-token`, затем `go run ./cmd/api`. Это локальные entry points, не факт deployment. Файл токена создаётся O_EXCL/0600 и не перезаписывается. При неизвестном исходе выдачи оператор сверяет приватный файл и БД перед повтором. API не применяет DDL и использует `want_keep_app`; оператор — отдельный migration DSN.

Конфигурация: `WANT_KEEP_ENV`, `WANT_KEEP_DATABASE_URL`, `WANT_KEEP_ORIGIN`, `WANT_KEEP_LISTEN_ADDR` (default `127.0.0.1:8080`), `WANT_KEEP_MAX_MEMBERS` (default 2), `WANT_KEEP_TRUSTED_PROXIES` (CIDR-list, default empty). Production origin строго `https://want-keep.tech`; local/test — отдельный localhost RP. Production DB требует проверенный TLS. Оператор использует `WANT_KEEP_MIGRATION_DATABASE_URL`, maintenance — `WANT_KEEP_MAINTENANCE_DATABASE_URL`.

Task-8.1 настраивает proxy, перезаписывающий X-Forwarded-For одним проверенным IP и X-Forwarded-Proto. От остальных peers forwarding-заголовки игнорируются. Research nginx и зарегистрированный callback Raiffeisen не изменены. `go run ./cmd/identity-retention` удаляет до 1000 истёкших записей каждого transient-типа; task-8.1 запускает maintenance каждые 5 минут отдельной ролью. Audit и финансовая история сохраняются.

## Проверки и границы

Обязательны `make check`, `make test-integration AREA=identity`, `make test-integration AREA=storage`, `make test-identity-race`, `make test-storage-race`, `git diff --check`. Отсутствие изолированной PostgreSQL — ошибка, не skip. Проверки используют настоящий WebAuthn verifier, синтетические ES256/RS256 ключи, HTTP handlers и PostgreSQL 17.11 с закреплённым digest. Журнал проверок и CI относятся к точному публикуемому SHA и сохраняются отдельно.

Сценарии: bootstrap/replay/rollback/lost response, два входа и чужие ресурсы, подпись/UV/origin/RP/handle/challenge, CSRF/сроки/fresh auth, конкурентный recovery/generation, сохранение доступа партнёра, отзыв bindings, запасной ключ, pagination, rate limits и restart.

AC-001 подтверждается в части bootstrap; приглашения — task-1.6. AC-049: backend recovery/isolation; реальные Chrome/Arc/Touch ID — task-7.1 и итоговая приёмка. AC-050: auth boundary без файлов и provider secrets task-1.5. AC-072/088: сохранённый отзыв своих bindings без фактической доставки push/уведомлений. Notification task проверяет binding при регистрации и непосредственно перед отправкой. Task-7.1 реализует UI, memory-only CSRF и foreground activity максимум раз в минуту, без фонового продления.

Live banking, browser UI E2E и production не заявляются проверенными. SDD остаётся Ready for development. Источники: [WebAuthn](https://www.w3.org/TR/webauthn-3/), [go-webauthn v0.18.0](https://github.com/go-webauthn/webauthn/releases/tag/v0.18.0).
