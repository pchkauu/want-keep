# Dahl: MiniMax M2.7 и DeepSeek V4 Flash

[English](dahl.en.md) · [Выбор OpenAI](openai.md) · [Issue #8](https://github.com/pchkauu/want-keep/issues/8)

Источники и прогоны: 2026-09-07. Исследование использует только синтетические финансовые случаи. По последнему решению владельца выбран **Terra Extra High**; Dahl не включается в production routing, fallback или обработку семейных данных. Остальные продукты Want Keep не меняются.

## DAHL-E01–E04: доступ и границы

Проверены точные пользовательские IDs `MiniMaxAI/MiniMax-M2.7` и `deepseek-ai/DeepSeek-V4-Flash-0731` через `https://inference.dahl.global/v1/chat/completions`. `/v1/models` вернул оба ID; успешные completions вернули соответствующий model. Это доказательство доступности маршрута в момент проверки, а не неизменяемости модели или гарантированной доступности.

DAHL-E01 [API](https://inference.dahl.global/docs/api/), DAHL-E02 [Models](https://inference.dahl.global/docs/models/): документированы Chat Completions, tools и streaming; изображения для этих предлагаемых моделей не поддерживаются. Фото/PDF не отправлялись, vision/OCR не подменялся извлечённым текстом. Без дополнительной квалифицированной vision-цепочки эти маршруты не заменяют все обязательные функции MVP.

DAHL-E03 [Tokens](https://inference.dahl.global/docs/tokens/), DAHL-E04 [Authentication](https://inference.dahl.global/docs/authentication/): используется заранее выделенный пул; расход — input + output. Документация приводит ориентир около $0.03/1M, но точный оплаченный тариф ключа не подтверждён. Billing packages требует account session и вернул 401; ключ не заменялся банковскими/браузерными секретами. Покупки, пополнения или перераспределение токенов не выполнялись. В публичных материалах нет ключа, начального остатка или закрытого ответа аккаунта.

## DAHL-E05–E06: что означает high

DAHL-E05 [DeepSeek thinking mode](https://api-docs.deepseek.com/guides/thinking_mode/) описывает прямой API: `thinking.type=enabled` и `reasoning_effort=high`, который также является default. Оба поля отправлены через Dahl в фазе high_effort. Dahl не вернул подтверждение применённого effort: HTTP 200 доказывает приём запроса, но не передачу настройки upstream. Это не доказанный A/B low→high.

DAHL-E06 [MiniMax OpenAI-compatible API](https://platform.minimax.io/docs/api-reference/text-openai-api): у M2.x thinking постоянно включён, `reasoning_split` меняет формат. Управление `reasoning_effort` для M2.7 не документировано. По просьбе владельца поле high отправлено; API принял его, подтверждения применения нет. Рост времени/токенов в одном прогоне не доказывает причинного эффекта настройки.

## DAHL-E07: измерение

[Машинный отчёт](dahl.results.json), [замороженный prompt/schema](dahl.contract.json), [runner](../tools/dahl_eval.py), [reporter](../tools/dahl_report.py), [offline tests](../tools/test_dahl_eval.py).

| Model / phase | Text exact | Function | Usage tokens | Reserved tokens |
| --- | --- | --- | --- | --- |
| MiniMaxAI/MiniMax-M2.7 / baseline | 0/0 | 0/0 | 3244 | 0 |
| MiniMaxAI/MiniMax-M2.7 / explicit_schema | 15/20 | 0/0 | 7492 | 0 |
| MiniMaxAI/MiniMax-M2.7 / high_effort | 10/20 | 0/0 | 8421 | 0 |
| deepseek-ai/DeepSeek-V4-Flash-0731 / baseline | 0/0 | 0/0 | 0 | 20259 |
| deepseek-ai/DeepSeek-V4-Flash-0731 / explicit_schema | 0/0 | 0/0 | 0 | 21928 |
| deepseek-ai/DeepSeek-V4-Flash-0731 / streaming_schema | 15/20 | 0/0 | 5589 | 19985 |
| deepseek-ai/DeepSeek-V4-Flash-0731 / high_effort | 9/10 | 0/0 | 2662 | 22208 |

Ноль graded cases означает отсутствие оценённого результата, а не нулевое качество или успешный тест. Начальный screen использует первые 20 случаев: все 20 архетипов по одному, два batch по десять. Полные 200 вариантов запускаются лишь после чистого screen. MiniMax и DeepSeek screen не прошли; последующие 180 вариантов не запускались. High DeepSeek оценён только на 10 случаях; второй запрос оборвался. Отдельная function-проба DeepSeek получила 429, корректность tools не доказана; MiniMax function-проба не запускалась.

В MiniMax default/schema — 15/20, high requested — 10/20. Среди ошибок high: create вместо link для внутреннего движения, потеря данных duplicate, финансовые поля при clarify, отсутствие target отчёта. DeepSeek default/stream — 15/20; high requested — 9/10, с непустыми финансовыми полями при неизвестном счёте. Остальные ошибки и ID приведены в JSON. Финансовые команды не исполнялись; валидный JSON не равен корректному учёту.

Схема и prompt одинаковы между Dahl default/high после explicit_schema и соответствуют сохранённой редакции Terra high_receipts. Более поздняя итоговая Terra использует уточнения self-check/report target; результаты разных редакций не являются контролируемым сравнением качества самих моделей. Gold/Decimal-grader одинаковы; эталонные ответы не отправляются. В MiniMax переход к high одновременно использовал streaming, что дополнительно ограничивает сравнение скорости.

Baseline MiniMax вернул `<think>…</think>` перед JSON при response_format strict. Адаптер убирает только один полный наблюдаемый envelope, затем строго разбирает JSON и проверяет все финансовые поля. Это не доказательство enforced schema на сервере. Stream собирается до finish_reason, usage и `[DONE]`; обрыв никогда не становится предложением. Raw reasoning не публикуется. Требования upstream сохранять reasoning для продолжения tool-диалога не проверялись: здесь только одношаговые предложения без исполнения.

## DAHL-E08: расходы и сбои

Наблюдаемый usage — **27408 токенов**, неразрешённые/консервативные резервы — **84380 токенов**. По 250000 токенов на модель суммарно через все фазы, без сброса прежних вызовов. Резервы за 429 оставлены консервативно; HTTP 524 и curl timeout при HTTP 200 имеют неизвестный итог расходов. Поле HTTP 200 при оборванном SSE не означает законченный ответ. Никаких автоматических повторов.

Первый Python HTTP-клиент получил Cloudflare 403; обычный curl по пользовательскому примеру работал без обхода защиты. Не-stream DeepSeek получил 524 примерно через 120 s; документированный SSE позволил получить два ответа (27.144/28.186 s), но поздний high SSE оборвался через 180 s. MiniMax high — 154.070/150.552 s на десять случаев. Это latency вызова, не UI или production SLA.

Общий разрешённый бюджет тестов повышен владельцем до $7. OpenAI учитывается отдельно по точному usage и прайсу; Dahl использует существующий token pool без новой денежной покупки. Точный совокупный инвойс не заявляется: цену фактически выделенного пула и окончательные списания незавершённых запросов не удалось подтвердить. Эти ограничения сохранены как свойства невыбранного маршрута и не блокируют выбор Terra.

Заявление Dahl о zero retention на [сайте](https://inference.dahl.global/) само по себе не подтверждает полный договор обработки данных или условия всех upstream-операторов. Перед возможным будущим использованием семейных данных нужны отдельное решение владельца, проверка retention/стоимости, полномочий, устойчивости и полный повторный допуск. Новые runtime-контракты провайдера в этом исследовании не добавляются.
