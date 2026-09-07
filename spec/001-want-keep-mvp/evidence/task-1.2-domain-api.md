# Evidence task-1.2 — домен и API

[English](task-1.2-domain-api.en.md)

## Объём

D-44, 2026-09-07: независимый старт task-1.2 до завершения ресерча ранее обозначался D-37 в её ветке. D-37 теперь сохраняется за решением Alfa из task-0.10. База и цель PR — `docs/want-keep-mvp-sdd`. SDD Ready for development; эксплуатационная готовность приложения ещё не подтверждена. Пользователь принял D-41 вместо прежнего хранения команд весь срок жизни семьи.

Реализованы точные Money/Asset/Rate, явное округление и largest-remainder allocation; календарные типы; User/Household/Membership и проверка семейной области/личного редактирования; состояния известности, покрытия и свежести; неизменяемые переходы команд и проверка replay/версий. Money закрывает apd v3.2.3 внутри домена. Лимит строки — 256 символов; display scale не ограничивает точность источника. Переполнение возвращает ошибку, не обрезает сумму. Allocation принимает до 1000 уникальных весов и scale 0–254; итог должен точно выражаться в выбранном quantum. Floor и half-even задаются явно.

OpenAPI 3.0.3 описывает согласованные группы API, формы, безопасные ошибки, семейную область и объяснимые отчёты. Источник — `api/openapi.yaml`; oapi-codegen 2.8.0 создаёт Go models/strict interfaces, openapi-typescript 7.13.0 — TypeScript. Генератор TypeScript изолирован в пакете api с TS 5.9.3 из-за peer requirement `^5.x`; приложение сохраняет TS 6.0.3. Обновления иных зависимостей приложения не требуются.

## Команды и доверие

UUIDv4 Idempotency-Key создаётся до отправки и используется для status lookup. Уникальность — household+actor, тип и payload hash неизменны. Replay проверяется до повторной проверки старой expectedRevision: уже успешная команда возвращает существующий результат. Неизвестный исход остаётся pending до сверки, а не становится failed или разрешением нового ключа. По D-41 detail хранится 90 дней после terminal/reconciled outcome, unresolved до сверки; tombstone — до сверки и 400 дней после исхода; исходные сообщения, файлы, passkey и recovery-коды туда не копируются.

Principal создаётся из загруженного проверенного membership. Это чистая доменная policy, не аутентификация. Нет runtime-проверки реальной cookie, CSRF, DB-транзакции, outbox или HTTP-обработчиков. Они обязательны в task-1.3/task-1.4 и продуктовых задачах. Успешный schema test не доказывает реальную изоляцию серверной сессии или atomic exactly-once effect.

Схемы различают actorId (User), payer.memberId/распределение (Membership), personalOwnerId и externalAccountOwnerId (User). Внешние asset code/network и identity не подменяют Money.Asset. Provider DTO, секреты и неподтверждённые endpoint не реализованы.

## Проверка и передача

Исполняемые проверки: `make check`, `make check-contracts`, профильные Go-тесты money/calendar/household/reporting/commands/delivery и web API fixtures. Проверяются синтетические round trips шести активов, некорректные суммы, распределение, семейная policy, конфликт версии, replay, unknown/partial/stale, schema guards и воспроизводимость Go/TypeScript. Отсутствующий generated artifact должен завершать проверку ошибкой. Текущий результат локальных проверок и CI фиксируется в отчёте доставки [Issue #12](https://github.com/pchkauu/want-keep/issues/12).

Task-1.3 получает атомарную регистрацию команд и сохранение эффекта+итогового статуса, NUMERIC без потери точности, durable idempotency и outbox. Task-1.4 реализует доверенную сессию, семейные права и CSRF; продуктовые use cases проверяют текущую принадлежность и ревизию. `currentRevision` в ошибке возвращается только после разрешённого чтения ресурса.

Не выполнялись проверки PostgreSQL, банков, платежей, browser E2E или deploy: эти реализации за пределами task-1.2. Полные продуктовые AC-002/003/039/059/077/079/090 этим результатом не объявляются пройденными.

Уточнение task-1.2 после review: payer — явное known/memberId, unknown или not_applicable; плательщик вводится и исправляется отдельно от actor и долей. Связь существующих движений требует ID/expectedRevision каждого участника и атомарной проверки. Preview плана различает create/update/delete и lineId; expectedRevision относится к Budget aggregate, который меняется при каждом изменении статьи/подтверждении. ReturnsReport передаёт dimensionless XIRR ratio строкой, native/reporting basis, dated cash flows и unavailable reason; solver остаётся в task-6.4. Эти поправки затрагивают ещё не выпущенные DTO; оба клиента регенерируются вместе, действующих данных для миграции нет.

## Сверка с task-0.10

Контракт версии 10 объединяет D-37–D-43 с основой D-44. D-41 заменяет прежнее хранение команд весь срок жизни семьи. Длительности считаются по UTC как 30/90/400 × 24 часа; правая граница исключена. Исход после сверки запускает сроки от момента разрешения, pending не истекает. Command хранит компактные поля будущего tombstone; RequireDetail, InRecent и Recover проверяют права, время и восстановление без исполнения. HTTP 410 содержит безопасную ошибку и optional outcome только после проверки доступа к результату. Истечение tombstone даёт not_found без доказательства отсутствия эффекта. Очистка detail не удаляет компактную запись и финансовый аудит.

D-43 реализован как чистая policy в connections/domain: default pending, отдельные provider/host checks, точное совпадение binding, блокировка failed/revoked, отклонение старого check и сброс evidence при rebind. Ревизия admission начинается с 1 и растёт при каждом изменении состояния/evidence/binding без сброса при rebind; точный no-op сохраняет её, переполнение выше 9007199254740991 отклоняется. RequireResult запрещает старые binding/revision даже после повторного допуска с прежним binding. Boundary публикует ревизию только существующего admission; обратного command converter нет.

task-1.3 владеет application aggregate/repository в connections/admission, сохранением монотонной ревизии, атомарным check+enqueue и commit-time revalidation в транзакции source/posting/outbox. task-3.2/task-3.3 и task-4.x передают выданные binding/revision через jobs/results, проверяют их до IO и сохраняют устаревшие результаты в карантине без source/posting. Отмена начатого IO — best effort. Синтетические domain/DTO тесты покрывают переходы ревизии, no-op, переполнение, A → B → A, отзыв/повторный допуск и недоверенный ввод. Это domain/DTO coverage REQ-088/AC-106, не пройденный integration/security AC.

D-40 read models различают наблюдение с исходными legs, platform quote с amount/time/fee/spread coverage и unavailable с причиной. D-42 фиксирует строковый XIRR до 12 знаков и причины no_bracket/numeric_error_unbounded. Расчёт курсов и solver реализуют task-6.1/task-6.4. DTO изменены до появления product runtime; оба generated клиента обновляются вместе, миграция данных не требуется.
