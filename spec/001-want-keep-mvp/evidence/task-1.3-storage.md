# Evidence task-1.3 — хранилище и транзакции

[English](task-1.3-storage.en.md)

## Результат и совместимость

Реализована основа хранения контракта 10: pgx v5.10.0, последовательные SQL-миграции, application-сервисы команд, журнала, резервов и admission. PostgreSQL suite использует 17.11 с закреплённым digest. Новые публичные HTTP-маршруты и generated DTO не менялись. Это первые миграции ещё не развёрнутого продукта; production-данные не изменялись. После применения checksum защищает миграцию от редактирования, исправления схемы добавляются следующими файлами.

SDD остаётся **Ready for development**. Результат task-1.3 не подтверждает эксплуатационную готовность приложения. Статус публикации, review и CI фиксируется в [issue #13](https://github.com/pchkauu/want-keep/issues/13).

## Владельцы и внутренние контракты

| Область | Владелец и правило |
|---|---|
| Транзакции | Application interfaces; storage скрывает pgx и transaction context. `WithinHousehold` — READ COMMITTED, блокировка семьи и актуальное членство. Вложенная транзакция использует тот же Store и principal; переход на другой Store отклоняется. Admission lock всегда раньше семейного. |
| Команды | `commands/application.Executor`: регистрация pending отдельной короткой транзакцией; replay до expectedRevision; effect, revision, audit, outbox и terminal outcome сохраняются одной транзакцией. Callback выполняет только DB-работу. Инфраструктурная ошибка оставляет pending для сверки; подтверждённый business rejection фиксируется после rollback. |
| Журнал | `ledger/application.Writer`: неизменяемая revision, previous revision, actor и command reference; delta между версиями обновляет известные остатки. Unknown сохраняется unknown. Подтверждённый расход не отклоняется из-за ранее созданного резерва. |
| Цели | `goals/application.Service`: владелец личной цели, текущая revision, валюта и funding account проверяются под семейной блокировкой. Virtual/dedicated и освобождение старого резерва меняются атомарно. |
| Источники | `ledger/application.Sources`: D-39 identity включает household/provider/stable account/product/log/record ID; connection/job — provenance. Digest ускоряет поиск, полный ключ и внешний владелец сравниваются обязательно. Явное решение нормализатора определяет correction/ambiguity; hash не назначает исправление. Manual override переживает повторный импорт. |
| Admission | `connections/admission.Service`: точные binding/provider/host evidence и монотонная revision сохраняются атомарно. No-op не меняет revision; rebind не сбрасывает счётчик. Snapshot восстанавливается без переходов. |
| Jobs | Бounded batch, SKIP LOCKED, token каждой lease, deadline, максимум 5 попыток. Предыдущая попытка не завершает задание после истечения/смены lease. Неопределённый внешний эффект становится unresolved и не повторяется автоматически. |

Пользовательские/AI обработчики не получают admission write API. Будущий composition root выдаёт этот application-сервис доверенному операторскому процессу; обычные команды используют проверенный серверный principal. SQL-права не заменяют проверку principal на входе приложения.

## Точность и хранение

`NUMERIC` без scale сохраняет все шесть активов и десятичные строки до 256 символов. На границе используются строки, без float; NaN/Infinity и неподдерживаемые активы отклоняются. Известность, coverage и freshness — отдельные поля. UTC Instant хранится как TIMESTAMPTZ, усечённый до микросекунд, плюс остаток наносекунд 0–999; граничные сравнения используют обе части. Date, Month и IANA timezone сохраняются отдельно.

Составные foreign keys сохраняют семейную область. Source revisions, postings, balance snapshots, audit, outbox и quarantine защищены от изменения/удаления. Идентичность команды и выданные binding/revision/generation задания недоступны для UPDATE application-роли. Записи команд не содержат копий документов, сообщений или секретов; evidence — ссылка, жизненным циклом файлов владеют последующие задачи.

## D-41 и восстановление

Detail доступен 90 × 24 часа после terminal/reconciled outcome, tombstone — 400 × 24 часа; unresolved не истекает. Recent возвращает terminal младше 30 × 24 часов и все pending; курсор — UUID последней записи в порядке registration timestamp/ns/ID. Права на статус ограничены actor, ссылка на результат дополнительно проверяется `ResultAuthorizer`.

Очистка detail и tombstone независима и ограничена batch 1–1000. Пока detail ещё физически не очищен, tombstone не удаляется. Detail сейчас является маркером; компактные status/hash/outcome принадлежат tombstone. После истечения detail query возвращает `command_expired` с разрешённой ссылкой. Финансовый аудит не зависит от retention: сохранившаяся command reference блокирует повтор эффекта даже после удаления tombstone. `not_found` и timeout не разрешают новый ключ.

## D-43 и передача collector

Проверка admission и enqueue атомарны. Выданный Job несёт неизменяемые binding, admissionRevision и connection generation. `BeforeRead` вызывается непосредственно перед IO; IO выполняется вне транзакции. `CommitPage` проверяет актуальное admitted, поколение подключения, lease и входной cursor. Запись source/page, ledger/outbox и нового checkpoint атомарна. Revoke/change отменяет ожидающие задания и запрашивает отмену выполняющихся; поздний результат сохраняется в quarantine без финансовой записи или продвижения checkpoint.

Task-3.3 подключает настоящий collector к этим вызовам, передаёт issued Job без подмены и сохраняет raw evidence до CommitPage. Task-3.2 реализует полноценную сверку балансов; task-4.x — нормализацию конкретных платформ. Сейчас запись известного остатка и ledger delta проверена синтетически, полная политика авторитетности snapshot/истории остаётся задачей сверки.

## Проверки и границы доказательства

- `make check`: Go unit/architecture, web/collector lint/typecheck/tests/build, документация и воспроизводимость OpenAPI.
- `make test-integration AREA=storage`: реальная изолированная PostgreSQL, непривилегированная application-роль; отсутствие БД — ошибка, test cache отключён.
- `make test-storage-race`: те же реальные транзакционные сценарии с Go race detector.
- `git diff --check`.

Suite проверяет пустую БД, повтор/конкуренцию миграций, rollback ошибочного SQL, checksum/неизвестную версию; шесть активов и предельную точность, наносекунды/даты/unknown; семейные права; concurrent duplicate commands и версии; потерю подтверждения, принудительный разрыв DB-соединения до commit и восстановление одного эффекта; 800+800/1000 и 80+30/100, валюты и режимы резервов; D-39 reconnect/correction/collision/checkpoint/manual override; границы 30/90/400 дней; lease/restart/outbox; concurrent evidence, A→B→A, revoke до/после IO, quarantine.

REQ-012/029/059/061/062/064/069/070/072/076/088 → AC-012/029/059/061/062/086/090/092/106 → task-1.3. AC-012 ещё требует продуктового исправления AI-связей и пересчёта отчётов. AC-106 ещё требует настоящий collector и provider/host evidence deployment. HTTP/passkey/CSRF, live banking, browser E2E, production deployment и весь финансовый lifecycle не проверялись и принадлежат следующим задачам.

## Локальный запуск и эксплуатационная передача

Из корня репозитория:

```sh
docker compose -f backend/test/integration/storage/compose.yaml up -d --wait
export WANT_KEEP_TEST_DATABASE_URL='postgres://postgres:synthetic-admin@127.0.0.1:55432/want_keep_test?sslmode=disable'
make test-integration AREA=storage
make test-storage-race
```

Compose использует только synthetic credentials, loopback и tmpfs. Suite создаёт изолированную БД для каждого теста и не удаляет существующие базы. После проверки `docker compose -f backend/test/integration/storage/compose.yaml down` освобождает только этот тестовый сервис; tmpfs-данные теряются.

Task-8.1 создаёт отдельные migration, application (`want_keep_app`) и maintenance (`want_keep_maintenance`) роли с собственными секретами; две последние должны существовать до миграции, без SUPERUSER/CREATEDB/CREATEROLE/BYPASSRLS и без членства в migration-роли. Migration-роль владеет БД/схемой; application получает SELECT/INSERT и ограниченный UPDATE, maintenance — SELECT/DELETE только команд. Скрипты не создают production-пароли и не повышают права автоматически. Production подключение требует проверяемый TLS без plaintext fallback; development/test допускают только loopback.

```sh
cd backend
# DSN поступают через защищённое окружение, значения не печатаются.
WANT_KEEP_ENV=production go run ./cmd/migrate
WANT_KEEP_ENV=production go run ./cmd/command-retention -mode details -batch 100
WANT_KEEP_ENV=production go run ./cmd/command-retention -mode tombstones -batch 100
```

Соответствующие переменные: `WANT_KEEP_MIGRATION_DATABASE_URL`, `WANT_KEEP_MAINTENANCE_DATABASE_URL`. Retention CLI выполняет один ограниченный batch; scheduler task-8.x запускает режимы независимо, наблюдает ошибки/lag и повторяет до исчерпания очереди с ограничением работы. Rollback приложения не удаляет схему/историю. Production major, секреты, расписание и мониторинг закрепляются в task-8.1/8.3; здесь production не изменён.

Review: подтверждённая correction с текущей source revision снимает неоднозначность даже при том же hash; повтор не создаёт эффект. Любой stale lease/result/cursor сохраняется в quarantine после отката страницы; обычные ошибки хранения остаются ошибками и не превращаются в успешную обработку. Барьерный PostgreSQL-тест проверяет истечение lease во время apply.
