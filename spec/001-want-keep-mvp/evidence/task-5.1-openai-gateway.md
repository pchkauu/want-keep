# task-5.1 — OpenAI gateway и контроль расходов

[English](task-5.1-openai-gateway.en.md)

## Результат и границы

Реализованы gateway Responses API, точный семейный лимит $50 за UTC-месяц, долговечные AI-попытки, интеграция с worker и операторская сверка неизвестной оплаты. Выбранный `gpt-5.6-terra` с `reasoning.effort=xhigh` и strict schema материализуется из квалифицированного `terra_xhigh` в проверяемый runtime-contract. Локальные и CI-тесты используют поддельный HTTP transport; реальный OpenAI API и production не вызывались.

Task-5.1 сохраняет результат модели со статусом `pending_validation`. Он не изменяет финансовый журнал: проверка полномочий и применение предложения остаются task-5.2, разбор чеков — task-5.3, инсайты — task-5.5. Публичный OpenAPI не изменён.

## Исполняемый контракт

- `ai/domain` хранит точные decimal USD, usage с различием отсутствующего и нулевого `cache_write_tokens`, тариф `$2/$0.20/$2.50/$12` и состояния попытки. Reservation включает `counted input + 32`, максимальный output и консервативный cache-write. Reasoning входит в output один раз.
- `gateways/openai` закрепляет `openai-go/v3` v3.56.0, отключает SDK retries и отправляет только согласованные поля Input Tokens и Responses API. Generation всегда foreground, `store=false`, default service tier, explicit cache, disabled truncation и parallel tools. API key читается только из приватного абсолютного файла.
- `scripts/generate-openai-runtime.py` создаёт runtime-contract из `evidence/openai.prompts.json`; `make check-ai-runtime-contract` сравнивает генерацию во временном файле и запрещает drift prompt/schema/config.
- Миграция 016 хранит неизменяемую identity попытки и append-only состояния с request/prompt/schema/config fingerprints, разрешённым input, provider ID, usage, резервом, стоимостью, output и сверкой. Application role имеет SELECT/INSERT без UPDATE/DELETE; maintenance role ограничена сверкой.
- Перед платным вызовом storage под household lock проверяет глобальный unresolved-barrier, сумму `actual + reserved + unknown`, два слота и месячный лимит. После этого отдельно сохраняются reserve и external-started marker. Provider IO не выполняется в транзакции.
- Known retryable отказ допускает одну новую попытку; следующий такой отказ завершает job постоянной ошибкой. Таймаут либо неопределённый transport outcome после marker становится `unknown`, сохраняет резерв и job `unresolved`. Новый месяц, смена ключа или проекта не снимают barrier. Оператор подтверждает точную стоимость либо отсутствие списания по request ID и безопасной evidence-ссылке. Доверенный resumer возвращает budget-waiting job в `ready`, когда сохранённый резерв снова помещается в лимит либо начинается новый UTC-месяц; финальная проверка резерва остаётся атомарной.
- Успешные provider result, usage, output, validation status и job receipt сохраняются атомарно. Повтор worker после commit не вызывает provider снова. Проекция в OpenAI ограничена конкретной ledger revision и исключает credentials, sessions, raw source payloads и данные других семей.

## Проверенное покрытие

| Связь | Доказано task-5.1 | Остаётся владельцу продукта |
| --- | --- | --- |
| AC-018 | Каждая существующая review job получает версионированную AI-попытку; повтор terminal job не вызывает gateway | Применение и stale-result UX в task-5.2 |
| AC-022 | Разрешённая ledger-проекция, приватный key file, безопасные persistence/diagnostics и раскрытая retention | Чеки в task-5.3 и production review |
| AC-051 | $50, резервы, unknown barrier, waiting и независимость финансовых очередей | Системный экран в task-7.14 |
| AC-058 | Закрытые provider/job codes без финансового текста | Общая operational UI/наблюдаемость |
| AC-059 | Точные decimal reserve/usage, без float | Остальные финансовые вычисления |
| AC-069 | refusal/incomplete/schema/known rejection/unknown различены; повторы ограничены | Применение предложения task-5.2 |
| AC-085/090 | Principal и household берутся из job и повторно проверяются storage; семейные бюджеты изолированы | Chat authority и UI |

Синтетические проверки покрывают точный предел `$49.9 + $0.099999`, ожидание и автоматическое возобновление после UTC rollover, три конкурентных вызова, отдельную семью, неизвестный outcome, ручную сверку, completion/retry и exact HTTP shape. Обязательная suite завершается ошибкой без изолированного PostgreSQL.

## Эксплуатационная передача

Worker использует `WANT_KEEP_OPENAI_API_KEY_FILE`; необязательные `WANT_KEEP_OPENAI_PROJECT_ID` и test/development `WANT_KEEP_OPENAI_BASE_URL` не меняют семейный учёт. Production запрещает custom base URL. Сверка запускается maintenance DSN через `go run ./cmd/ai-reconcile --request-id ... --outcome charged|not_charged --actual-usd ... --evidence-ref ...`; plaintext key и финансовый текст в параметры не передаются.

OpenAI может хранить abuse-monitoring данные до 30 дней по действующим API data controls. `store=false` отключает обычное application-state хранение ответа, но не отменяет эту политику. Перед production task-8.1 проверяет secret provisioning, egress, тарифы, retention и operational runbook. Live-вызов требует отдельного разрешения и расхода.

SDD остаётся **Ready for development**. Task-5.1 не подтверждает готовность task-5.2, UI или production.
