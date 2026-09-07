# Task-1.6 — семья, приглашения и права

Реализованы закрытое приглашение второго участника, атомарная регистрация его passkey/сессии/recovery-кодов и семейные политики. База: `fc0f27d2a8dad4ac0db9672aff8108673e8a8c58`; ветка: `feat/task-1.6-household-permissions`. [Контракт](../contracts.md#task-16-приглашения-и-семейные-права) фиксирует согласованные сроки, отзыв, revision и неизвестные результаты.

## Реализация

Household domain/application владеет жизненным циклом приглашения и членством; identity application координирует настоящую WebAuthn-проверку и выдачу доступа. HTTP использует существующую границу delivery/identity и отдельный household-файл без дублирования session/Origin/CSRF guards. Domain/application не импортируют SQL, generated DTO или WebAuthn SDK.

Миграция 005 добавляет hash-токены, исходы приглашения, семейную revision, неизменяемый audit и привязку попытки. Миграции 001–004 сохранены. Сессия проверяется внутри исполнения; identity lock предшествует household/invitation lock. Регистрация, коды, сессия и outbox сохраняются атомарно. Повтор, отзыв, истечение, перевыпуск, неверный browser/purpose и заполненный состав не дают нового доступа. Чтение и создание финансовых записей не выполняются по одному invitation token.

Ownership проверяет текущего владельца до изменения scope; операции сохраняют отдельные actor/payer. ExternalOwnership позволяет обоим управлять connection, но банковскую авторизацию — внешнему владельцу. Реальные финансовые обработчики новых действий и secret storage здесь не добавлены.

## Проверки

Обязательные команды: `make check`, `make test-integration AREA=household`, `make test-integration AREA=identity`, `make test-integration AREA=storage`, `make test-household-race`, `make test-identity-race`, `make test-storage-race`, `git diff --check`. CI запускает все три integration/race suites. Изолированная PostgreSQL 17.11 закреплена digest; её отсутствие — ошибка, не skip. Версии Go/Node и API-генераторов сохраняются.

Новые HTTP-сценарии находятся в identity suite, повторно используя её ES256/RS256 fixture: полный путь второго входа и восстановления, сроки, одноразовость, ошибки токена, конфликт revision, отдельные cookies/коды, смена browser/purpose, reissue/revoke, конкурирующее принятие, гонки с отзывом/перевыпуском, рестарт, rollback перед session commit и потеря успешного ответа. Household suite проверяет реальные PostgreSQL-права на личные ресурсы, исправления партнёра, replay/конкурентную revision, изоляцию семей, актуальное членство, лимит и порядок блокировок. Синхронизация гонок использует барьеры, сроки — управляемые часы.

Все обязательные локальные команды прошли перед первой публикацией, включая отказ без БД. Review и CI фиксируются отдельно и связываются с точным опубликованным SHA в PR и отчёте доставки. Локальные fixtures не доказывают браузерную или эксплуатационную готовность.

## Границы AC и передача

| AC | Подтверждаемый результат task-1.6 | Следующая проверка |
| --- | --- | --- |
| AC-001 | Backend bootstrap/invitation, два входа, replay/expiry/лимит | SCR-003/004/005 и реальные Chrome/Arc |
| AC-077 | Отдельные сущности, конфигурируемый лимит, отсутствие partner1/partner2 | Новые роли/выход/замена не входят в MVP |
| AC-078 | Чтение семьи, личные/общие policies, существующий reserve service | Полный бюджет/цели, AI approval, уведомления |
| AC-086 | Конкурентные учётные исправления, actor, revision, replay | Уточнения AI и полный undo lifecycle |
| AC-087 | Разделение connection management/external-owner auth; regression admission | Secret/MFA IO и реальные адаптеры |
| AC-088 | Изоляция входа/recovery; один membership outbox event | Доставка, read-state и продуктовые уведомления |
| AC-090 | Изоляция семей и trusted principal в реализованных путях | Файлы, AI retrieval и полные фоновые процессы |
| AC-105 | Policy текущего владельца и запрет scope bypass | Account ownership handler и FORM-03 |

Task-1.5 использует ExternalOwnership и trusted session/membership для файлов и банковских секретов; task-2.1 — Ownership.RequireChange перед сохранением нового scope. Задачи бюджетов/целей/AI повторно проверяют текущую принадлежность, членство и revision; изменение payload не назначает actor. Task-7.1 реализует invitation UI, безопасную передачу токена и восстановление неизвестного ответа. Consumer outbox адресует member_joined другому активному участнику, без повторной доставки.

Production, live banking/MFA, браузерный UI E2E, файлы, AI retrieval и доставка push не заявляются проверенными. SDD остаётся **Ready for development**; готовность к эксплуатации не подтверждается.
