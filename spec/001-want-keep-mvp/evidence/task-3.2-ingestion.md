# Task-3.2 — входной контракт коннекторов

Реализован внутренний ingestion-контракт версии 10. База реализации: `cedbe3ba8384071e0e958167d37899ade557c0bc`; ветка: `feat/task-3.2-ingestion-contracts`. Зависимости task-3.1, task-1.2, task-2.1 и task-2.2 включены. Production и реальные платформы не изменялись.

## Доказанный результат

`collector/contracts/v10/ingestion.openapi.yaml` — единый источник wire-моделей. `make generate-contracts` создаёт Go transport DTO в `backend/internal/integrations/contract/generated` и TypeScript-типы в `collector/src/contracts/generated`; `make check-contracts` воспроизводит оба результата во временном каталоге. Строгие Go/TypeScript codecs проверяют одинаковые Unicode code-point и отдельные byte limits, NUL/lone surrogate, неизвестные поля, trailing JSON, base64/digest, действительные даты, канонические UTC-instant с `Z`, decimal-строки, discriminators и точное эхо server-issued job/binding/admission revision/cursor. Manifest проверяется не только структурно: его provider связывается с job binding, а каждый account/balance/transaction сопоставляется с объявленными product, log namespace, record kind и read action до сохранения evidence.

`integrations/domain` содержит provider-neutral значения; generated и provider DTO остаются в contract boundary. TypeScript boundary разделён на типы, общую валидацию, capabilities, wire codec и synthetic gateway; gateway хранит собственные неизменяемые снимки manifest/result. `integrations/application.Service` вызывает pre-read admission fence, сохраняет raw evidence до финансового commit и применяет страницу только через `admission.CommitPage`. Сервер назначает evidence/fetch/operation/source revision, principal и внутренние account IDs. Application разрешает account descriptor отдельно от observation; balance/posting используют descriptor той же самостоятельной страницы. D-39 source key кодирует части структурно и не использует сумму, время, connection, cursor или локализованный текст. Карточные alias отклоняют PAN/CVV, Unicode-цифры и любые цифры кроме собственных последних четырёх ASCII-цифр.

Synthetic golden page покрывает RUB, USD, USDT, USDC, BTC, ETH и неподдерживаемый USDC.E; разные продукты/аккаунты, карточный alias, known/unknown, partial coverage, точные дроби, pending расход, net trade P&L с included fee/valuation и mining reward. Reconnect/replay не дублирует account, opening, observation, source revision или posting. Ошибка evidence storage не открывает финансовый commit; stale admission сохраняет evidence в quarantine без checkpoint/финансового эффекта. Reauth/MFA/CAPTCHA переводят job в ожидание, rate-limit/temporary failure — в ограниченный повтор с сохранённой задержкой, необратимый отказ — в terminal failure.

## Матрица проверок

Обязательные команды: `make check`; `make check-contracts`; `make test-collector FILTER=contracts`; `make test-integration AREA=all`; race suites ingestion/jobs/storage/accounts/ledger/audit/matching/reconciliation; `git diff --check`. PostgreSQL suite использует изолированную локальную PostgreSQL 17.11 под непривилегированной ролью и завершается ошибкой без `WANT_KEEP_TEST_DATABASE_URL`. Конкретные результаты кандидата и CI фиксируются в PR.

Тесты проверяют account-only page без выдуманного observation, manifest/provider mismatch до evidence, immutable synthetic fixtures, provider failure lifecycle без финансового apply, повтор и конкурентную доставку, точность PostgreSQL NUMERIC, family/job-derived actor, структурные account/source identities, Unicode round trip и quarantine stale result. Варианты PAN с точками, slash, zero-width и Unicode-цифрами отклоняются. Ingestion fixtures удаляют собственную синтетическую БД после каждого теста; повторный suite не заполняет PostgreSQL tmpfs. Golden fixtures проверяют финансовый смысл, а не только JSON shape.

## Границы критериев

| Критерии | Доказуемая часть task-3.2 | Дальнейшая проверка |
| --- | --- | --- |
| AC-004/005/039 | Точные суммы шести активов, независимые knownness/coverage/freshness, отдельные owned/available/locked/debt/credit limit и явный unsupported asset | Полные продуктовые отчёты и валютная оценка |
| AC-008/009/035 | D-39 dedup/revisions и typed pending/posted/cancelled, fees/funding/reward/P&L mappings | Реальные provider lifecycle и matching |
| AC-041 | Cursor/completion/gaps и атомарный checkpoint одной страницы | Реальный многостраничный provider replay |
| AC-048/087 | Только read capability; typed reauth/MFA/CAPTCHA; stale generation/lease не применяется | Live route allowlist, session owner handoff и UI |
| AC-062/079/090/106 | Границы domain/transport, server-owned identity/principal, exact admission binding/revision и commit-time quarantine | Deployment conformance и production admission |

SDD остаётся **Ready for development**. Реальные API/Playwright connectors, разрешённые live routes, UI и production здесь не проверены; их подтверждают task-3.3/task-4.x/task-8.x.
