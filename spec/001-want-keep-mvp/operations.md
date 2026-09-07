# Эксплуатация, стоимость и восстановление

[English](operations.en.md)

SDD Ready for development не является эксплуатационным допуском. До production task-8.x подтверждает инфраструктуру, а каждый task-4.x — отдельный provider deployment gate.

## Лимиты и размещение

D-15: сервер не дороже $40/месяц в Германии, Нидерландах или Болгарии; OpenAI отдельно до $50/месяц. Платные внешние источники не согласованы. Исходная нагрузка — сотни операций/месяц; количество страниц чеков, длина чатов и backfill измеряются отдельно.

task-0.9 завершила [исследование инфраструктуры](evidence/hosting.md) на 2026-09-07. Владелец выбрал немецкий VPS 2 vCPU/4 ГБ/50 ГБ для web/reverse proxy, Go API/worker и одного последовательного collector, а также managed PostgreSQL 1 vCPU/2 ГБ/20 ГБ в той же private VPC без публичного DB IP. Годовая цена с одним VPS IPv4 — 2 520 ₽/месяц; консервативная цена без скидки и с резервом 10% — 3 055.56 ₽/$35.29 при CBR 86.5857 RUB/USD, ниже $40. Цены, налог, IP и курс перечитываются перед заказом.

Текущий исследовательский VPS стоит 800 ₽/месяц со слов владельца. Read-only аудит показал Ubuntu 26.04.1 LTS, 1 vCPU, около 889 MiB RAM, отсутствие swap и файловую систему около 14 GiB; приложение, БД и backup не развёрнуты. Он не допущен для production. До финансовых данных task-8.1 выполняет hardening SSH/firewall/monitoring/secrets, создаёт private DB connection, проверяет invoice и измеряет peak CPU/RAM/disk/collector. С текущего VPS публичные OpenAI/rate endpoints и часть платформ достижимы с оговорёнными статусами; безопасный DNS/TLS route к Альфа-Банку проверяется как runtime gate task-4.1/task-8.1.

Provider deployment выключен по умолчанию. Для D-43 admission нужны provider evidence task-4.x и host/deployment evidence task-8.x на одном binding environment/build/contract/allowlist/config/permission. Application service `backend/internal/connections/admission/` через storage adapter task-1.3 атомарно меняет server-owned state и `admissionRevision`. Admission check и enqueue выполняются в одной транзакции; stale/missing binding возвращает `provider_not_admitted` до нового job/collector IO. Job/result несут exact revision, collector проверяет её перед IO, а storage — при commit source/posting/outbox. Revoke/change инвалидирует не начатую работу; начатый read отменяется best effort, stale result остаётся в quarantine без финансового эффекта. Pre-admission conformance также работает в quarantine. Непройденный gate оставляет конкретный источник отключённым; обычный и ручной учёт продолжают работать. Unknown/partial/ambiguous отражаются в health раздельно.

## OpenAI

Выбор модели и лимиты принадлежат [исследованию task-0.8](evidence/openai.md); [смета](evidence/openai.cost.json) отделена от фактического usage и банковского списания. Срез цен — 2026-09-07. Выбрана gpt-5.6-terra xhigh для всех AI-задач; финальный eval — 206/206 без лишних уточнений, 6/6 PNG/PDF, 3/3 function calling. Контракт SDD закрыт; runtime приложения предстоит.

Foreground Responses, `store=false`, собственная история чата, `prompt_cache_options.mode=explicit` без breakpoints, `detail=high` для страниц. Разрешены только проверяемые предложения application; credentials, SQL, браузер, shell, платежи и hosted tools недоступны модели. Правила OpenAI retention и отсутствие подтверждённых ZDR/EU residency раскрыты в evidence; европейский VPS не обеспечивает европейскую обработку OpenAI.

Бюджет семьи — $50/UTC-месяц: actual + reserved + unknown. Перед каждым вызовом атомарно резервируется максимум input/output и возможных cache writes; reasoning уже входит в output. Подтверждённый usage сверяется с резервом; неизвестный исход и отсутствующий write count не дают молча освободить деньги. Смена месяца/модели/ключа не обнуляет обязательства. Максимум два вызова одновременно, SDK retries выключены; нет автоматического увеличения бюджета. Provider hard limit дополняет этот контроль, но может срабатывать с задержкой.

При отказе API, отсутствующей модели, исчерпании денег или неподтверждённой цене AI ожидает; обычный учёт работает. Каждая версия операции остаётся в очереди. Лимиты страниц/контекста/повторов и измерение качества находятся в evidence; production-проверки gateway, receipt pipeline и семейных прав остаются task-5.1–task-5.5. Перед запуском сверить проектный баланс и применимые налоги/сборы с общей сметой task-0.9.


Плановый xhigh-профиль: $47.125/месяц, стресс $62.96875 при общем лимите $50; сверх резерва задания ждут, без downgrade. Это смета по допущениям, не измеренный месячный счёт.

## Копии на MacBook

Копирование инициируется локальным Mac по исходящему соединению; доступность Mac — условие, не обещание круглосуточной работы. Через ограниченный non-root export principal VPS потоково получает logical dump managed PostgreSQL по private network и inventory неизменяемых вложений. Согласованный набор содержит cutoff, schema/version/time и checksums. Набор считается успешным только после локальной проверки всех частей; входящий доступ на Mac не открывается.

Копия зашифрована; приватный recovery-ключ хранится отдельно от единственной копии и от аварийного сервера. Server/master keys и recovery-коды доступа не публикуются и не печатаются. Внешняя облачная копия не включена в согласованный MVP.

При недоступном Mac показывать возраст последнего полного набора и реальное окно возможной потери. Часовой RPO условен доступностью и успешным завершением цикла; прерванный download не обновляет success timestamp. После восстановления связи продолжить копирование, сохранив последний исправный набор.

Политика MVP: 48 почасовых, 30 дневных, 8 недельных и 12 месячных точек; deduplicated repository cap 20 GiB, warning при 15 GiB или менее 25 GiB свободного места. Последний complete set не удалять. На проверенном Mac было около 46 GiB свободно, но шифрование целевого volume не подтверждено; task-8.2 обязан выполнить capacity/encryption/recovery-key preflight и измерить фактический размер. Непомещающаяся политика даёт явную ошибку и требует увеличить хранилище, а не удаляет последнюю сохранную точку.

В изолированной rehearsal восстановить DB/attachments/schema, проверить балансы, аудит, identities и jobs, замерить цель RTO до четырёх часов. Старые bank sessions и web sessions не оживлять автоматически. Проверить unknown AI charges перед повторами. Runtime восстановление и сохранность настоящих данных этим документом не доказаны.

## Наблюдение и уведомления

Состояния источников: connected/reauth_required/syncing/stale/partial/failed/disconnected. Provider admission: pending/admitted/blocked с binding/reasons. AI: pending/running/reviewed/clarification/waiting_budget/failed/superseded. Backup: pending/complete/stale/failed. Объединение этих статусов в один зелёный индикатор недопустимо.

Сводки и напоминания имеют source event ID, время/таймзону и dedup key. Push-подписка привязана к устройству/owner, при recovery или отзыве устройства инвалидируется. Payload по умолчанию не показывает суммы/названия продавцов на экране блокировки. In-app канал работает при отсутствии разрешения на push. Реальное разрешение/запрет, доставка и отзыв push проверяются в Chrome и Arc на macOS. Установка приложения не требуется контрактом; неподдерживаемая доставка оставляет in-app канал и явный статус.

## Доказательства запуска

Для допуска полного MVP нужны: все AC; полный readback каждого продукта шести платформ после provider gates; измеренный AI quality/cost; отсутствие двойных финансовых эффектов при retries; приватность/авторизация; RU/EN/desktop; push в реальных Chrome и Arc на macOS; backup/restore rehearsal и фактическая смета. CI и mocks не заменяют эти проверки. SDD Ready разрешает разработку, но не пропускает ни один из этих runtime gates. Развёртывание и любое приобретение ресурсов требуют действующей авторизации отдельного этапа.

## Семейная эксплуатация

Лимиты $40/$50 и сотни операций относятся к семье целиком. AI accounting/reservations привязаны к family budget, а не выделяют $50 каждому. Копии включают обоих пользователей, членство, принадлежность, allocation revisions, общий чат и авторство. Восстановление входа отзывает только устройства соответствующего пользователя; сброс второго входа не разрешён.
