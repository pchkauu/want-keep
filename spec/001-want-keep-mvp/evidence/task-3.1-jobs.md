# task-3.1 — долговечные задания

[English](task-3.1-jobs.en.md)

## Результат и границы

Реализованы PostgreSQL-расписание, очереди sync/outbox/ai, worker, lease/heartbeat, ограниченные повторы, ожидание зависимостей, подтверждение локального эффекта и сверка неизвестного внешнего результата. SDD остаётся Ready for development. Это инфраструктура task-3.1; реальные provider/collector/OpenAI handlers, денежные резервы AI, HTTP подключения и UI остаются task-3.2/3.3/4.x/5.x/7.x/8.x.

`transaction.changed` создаёт ровно одно AI-задание на household/operation/revision через существующий ledger review request. Другие события сохраняются в outbox в состоянии ожидания consumer, не теряются и не считаются доставленными. При отсутствии sync/AI обработчиков worker сохраняет `waiting/handler_unavailable`; бухгалтерский журнал продолжает работать. Платные вызовы и production provisioning не выполнялись.

## Контракты и совместимость

- `jobs/domain` владеет состояниями, причинами, retry policy и прогрессом. `jobs/application` — worker, scheduler, executor, outbox routing и trusted reconciliation port. `storage` скрывает SQL/pgx; `cmd/worker` соединяет реализации. Новых HTTP маршрутов, generated DTO и зависимостей нет.
- По умолчанию один исполнитель каждой очереди, polling 1 секунда, lease 60 секунд, heartbeat 20 секунд, пять попыток. Backoff: 5 секунд с удвоением до 5 минут и уменьшением на случайные 0–20%. Scheduler проверяет сроки каждые 30 секунд; ручной запрос и hourly job переносят следующий запуск на час. Пропущенные часы объединяются в один запуск.
- Deadline выполнения ограничен 24 часами. `deadline` исходного задания остаётся неизменяемым; `run_deadline` возобновления задаётся сервером после ожидания зависимости или доказанного отсутствия внешнего эффекта. Waiting не расходует execution attempts; unresolved не возобновляется автоматически.
- Worker получает только свободное число jobs, восстанавливает principal из сохранённого actor и актуального membership, обновляет heartbeat и отменяет handler при потере lease. Handler обязан соблюдать context cancellation. Для outbox/AI `Prepare` не изменяет финансовые записи; `Result.Apply` выполняет только локальную транзакционную DB-работу. Executor проверяет family/actor/job/attempt/token/target и lease до effect и при terminal commit. Effect, receipt и acknowledgement атомарны.
- Sync-handler внутри `Prepare` управляет страницами: вызывает `BeforeRead` перед каждым внешним чтением вне транзакции, затем сохраняет страницу исключительно через admission `CommitPage`. Это явное исключение из общего разделения Prepare/Apply; handler не открывает внешнюю household-транзакцию вокруг `CommitPage`. Его callback выполняет только локальную DB-работу, сохраняя порядок admission → household. Только подтверждённая последняя страница разрешает вернуть `Succeeded` с `Apply=nil`; executor читает уже сохранённый receipt. `CommitPage` с `applied=false` не означает успех. Промежуточные страницы сохраняют checkpoint для продолжения при сбое. Generic executor отклоняет попытку sync-записи через `Result.Apply`, сохраняя source fence. Синтетический тест `TestSyncWorkerCommitsAdmittedPages` проверяет этот протокол через `Worker.Step`, включая финансовые эффекты и повторный запуск.
- Перед неповторяемым внешним действием handler обязан вызвать `Execution.BeginExternal`. Маркер записывается до IO; неизвестный ответ или падение после маркера даёт unresolved даже при последней попытке. Trusted `ReconcileJob` принимает exact attempt/token, evidence reference и confirmed/absent. Повтор той же сверки идемпотентен, конфликт отклоняется. Только absent разрешает ограниченный повтор; local apply и confirmed outcome атомарны.
- `CompleteReview` привязывает ответ к operation/revision AI-задания и вызывает существующий ledger service. Старый ответ не меняет новую revision; самостоятельной AI-интерпретации денег worker не выполняет.
- Admission lock предшествует household lock. Source page, posting/outbox и checkpoint сохраняются вместе; новая попытка и новое задание после terminal failure используют незавершённый прогресс. Coverage gaps сохраняются. `last_success_at` меняется только при завершении; partial coverage остаётся явно частичным.
- Отключение или смена admission отменяет старую попытку; неизвестный внешний эффект остаётся unresolved. Историческая uncertainty не разрешает новое автоматическое выполнение. MFA остаётся действием владельца аккаунта.
- Причины ошибок — закрытые коды; logs содержат queue, persisted job/source/transaction ID, stage, duration и закрытый код причины. Secret, DSN, курсор, payload и финансовые сообщения не печатаются.

Для confirmed sync reconciliation обязательна `Page` с тем же evidence reference. После проверки актуального admission и поколения доверенная транзакция даёт доступ существующим source/account application-контрактам и атомарно сохраняет страницу, omissions, checkpoint и reconciliation. Последняя страница добавляет receipt и время успеха; промежуточная продолжает с новым курсором. Обычный running-attempt fence не ослабляется. Неизвестный исход сохраняется при invalidation/disconnect даже у legacy-заданий без external marker.

Успешный `CommitPage` подтверждает текущую маркированную внешнюю операцию и очищает её marker атомарно с checkpoint; следующая страница ставит новый marker. При legacy-сочетании unresolved и активной замены подтверждённое отсутствие эффекта завершает старую попытку, сохраняя замену. Любая подтверждённая страница сохраняет новый checkpoint и отзывает старую замену, включая последнюю страницу и исчерпание попыток; собственный неизвестный эффект замены, если есть, сохраняет отдельный unresolved-барьер. Регрессионные сценарии проверяют отсутствие повторного запуска после завершения и продолжение новым заданием с подтверждённого checkpoint после исчерпания попыток.

## Запуск и миграция

Миграция `010_durable_jobs.sql` добавляется после 009; сохраняет outbox/историю/курсоры и существующие задания. Перед миграцией остановить старые workers; применить миграцию операторской ролью, затем запустить новый binary прикладной ролью. Startup не выполняет миграции и не удаляет данные; откат схемы автоматически не предлагается. `run_deadline` не ослабляет SQL-защиту исходного `deadline`.

Backfill выбирает единственное активное задание текущего поколения. Если активного нет и terminal-история неоднозначна, курсор не угадывается по deadline: сохраняется gap `legacy_checkpoint_ambiguous`, следующий импорт консервативно перечитывает историю с начала через действующую дедупликацию. Старые job rows и их курсоры остаются доступными для расследования.

Для сохранённых `transaction.changed`, созданных до появления review requests в 009, миграция 010 добавляет недостающие заявки по существующим household/operation/revision. Уже имеющиеся заявки, outbox, проводки и revisions не изменяются. Тест `TestUpgradeDeliversLegacyTransactionEvents` создаёт две версии операции на схеме 008, проходит 009/010 и повтор миграции, затем доставляет оба события через прикладную роль ровно один раз; сохранённая финансовая история сравнивается до и после обновления.

```sh
cd backend
go run ./cmd/migrate
go run ./cmd/worker
```

Окружение: `WANT_KEEP_ENV`, `WANT_KEEP_DATABASE_URL`; миграции используют отдельный `WANT_KEEP_MIGRATION_DATABASE_URL`. DSN поступают из защищённого окружения. Опциональный `WANT_KEEP_JOB_BINDINGS_FILE` — операторский JSON-массив точных `connections.Binding` текущего deployment без секретов; его environment обязан совпадать с процессом. Пустой список выключает scheduler для providers. Файл не устанавливает admission: необходим сохранённый совместный provider/host pass. Sync и AI обработчики подключаются в composition root своими задачами; после подключения handler автоматически возобновляет только `handler_unavailable`. Остальные причины возобновляет их доверенный владелец через `ResumeWaiting` после устранения причины.

## Проверка и передача

```sh
make bootstrap
make check
make test-integration AREA=jobs
make test-jobs-race
```

Jobs suite использует штатный изолированный PostgreSQL 17.11 и прикладную роль, отдельную БД на тест. Покрывает конкурентные workers/schedulers, manual coalescing, lease replacement, bounded retries, неизвестный результат и reconciliation, outbox→AI, revision binding, waiting, checkpoint после failure, cancellation и migration. Дочерний процесс принудительно завершается до commit, после commit и после external marker. Это локальные synthetic integration tests; они не подтверждают доступ к платформам или их идемпотентность.

Локальная проверка на Go 1.26.5, Node.js 24.19.0 и PostgreSQL 17.11: `make bootstrap`, `make check`, jobs integration/race — успешно. Затронутые storage, accounts, identity, household, ledger и audit integration/race, а также privacy isolation suite — успешно. Проверка SDD включает RU/EN, трассировку и воспроизводимость карточек; OpenAPI generation consistency и `git diff --check` проходят. Переполнение tmpfs тестового PostgreSQL при последовательных полных запусках устранено пересозданием только изолированного тестового контейнера; проверки повторены успешно.

Фактический итог проверок, publication/review/CI и merge SHA фиксируются в Issue #26 и PR. Задача разблокирует task-3.2 и зависимость task-5.1. Перед внешним IO handler обязан соблюдать read admission, explicit external marker и отдельный бюджетный контракт task-5.1; неподтверждённые gateway возможности не считаются реализованными.
