# Интеграции и исследовательские блокеры

[English](integrations.en.md)

Срез публичной документации: 2026-09-06; обновления исследований датированы в строках ниже. Подтверждённое чтение отдельных продуктов, полнота истории и готовность автоматизации оцениваются отдельно. Наличие публичной документации или маркетингового описания не означает доступ к личному аккаунту.

## Обязательное покрытие

| Платформа | Продукты | Подтверждено | Открыто и задача |
| --- | --- | --- | --- |
| Альфа-Банк | Debit/credit cards, current/savings, deposits; кэшбэк | 2026-09-07: [исследование завершено с блокерами](evidence/alfa.md). Live-чтение текущего/накопительного счетов, двух типов вкладов, операций и кэшбэка; опубликованные retail API accounts/cards/operations/loyalty. | BLK-01 открыт: eligibility и проверенный автоматический read-контракт, identity/полнота/reauth, второй аккаунт, отсутствующая кредитка, точные условия и FX/cashback lifecycle. task-0.1 завершает research; task-4.1 и task-0.10 остаются заблокированы. |
| Райффайзенбанк РФ | Debit/credit cards, current/savings, deposits | Есть портал API и сценарии выписок. | Применимость к личному retail аккаунту, договор/права и все продукты; task-0.2 → task-4.2. |
| Ozon Банк | Дебетовая карта и связанный основной счёт (D-32) | 2026-09-07: [исследование завершено с блокерами](evidence/ozon.md); пять read-маршрутов, два HAR, 10 синтетических проекций, отдельная комиссия и цепочка семи страниц. | BLK-03 открыт: конец истории, эксплуатация сессии, второй аккаунт и семантика недоступных полей. task-0.10 проверяет закрытие; task-4.3 заблокирована. Другие продукты Ozon — будущее расширение, не блокер. |
| Bybit | Funding USDT/USDC/ETH/BTC, используемый Easy Earn и P2P (D-36) | 2026-09-07: [исследование завершено](evidence/bybit.md); Chrome и публичный Flexible API прочитаны. Официальный API приоритетен. | BLK-04: BYBIT-B02–B05 — private key/scopes, структурные связи/история, actual Earn и P2P eligibility/способ чтения; task-0.10 → task-4.4. Прочие продукты отложены без блокировки. |
| Aifory Pro | RUB-счета, USDT, ETH и используемая карта USD (D-33) | 2026-09-07: [исследование завершено](evidence/aifory.md); UI-чтение счетов, движений и карты. | BLK-05: право/структурированный read-контракт, identity/history/reauth и card lifecycle — AIFORY-B02–B04 под task-0.10. task-4.5 не разблокирована. Другие продукты отложены без блокировки. |
| EMCD | Кошелёк USDT, существующие Coinhold/Grow, используемые криптокарты и исторические P2P-ордера (D-34) | 2026-09-07: [исследование завершено с блокерами](evidence/emcd.md); четыре области UI, 26 источников/наблюдений, шесть синтетических сценариев. Mining Pool API не покрывает выбранные продукты. | BLK-06: структурированный read-контракт, identity/history/reauth, балансы/Grow/card/P2P — EMCD-B02–B04 под task-0.10; task-4.6 заблокирована. Майнинг никогда не использовался, его история и другие неиспользуемые продукты не требуются. |

Источники: [Alfa developer portal](https://developers.alfabank.ru/), [Alfa onboarding](https://developers.alfabank.ru/products/alfa-api/documentation/articles/connection/connection), [Raiffeisen API](https://developer.raiffeisen.ru/), [Ozon Bank](https://finance.ozon.ru/), [Bybit wallet balance](https://bybit-exchange.github.io/docs/v5/asset/balance/all-balance), [Bybit transaction log](https://bybit-exchange.github.io/docs/v5/asset/fund-history), [Aifory Pro](https://aifory.pro/), [EMCD wallet](https://help.emcd.io/en/articles/16205516-what-is-emcd-wallet).

Ссылки Ozon и отдельные страницы Alfa при повторном чтении через исследовательский инструмент были недоступны. Это ограничение исследования, не доказательство отсутствия API.

## Результат исследования каждого источника

Для каждого обязательного продукта заполнить матрицу: продукт существует/доступен владельцу; read method; необходимые scopes; account identity/card alias; balances owned/available/locked/debt; events/IDs/revisions/statuses; fees/net-gross; date/timezone; pagination/window/depth; terms/minimum/grace/accrual; quote direction/amount/fee; rate limits; reauth; endpoint allowlist; evidence date; synthetic fixture; live result.

Положительный результат требует фактического сопоставления с авторизованным источником. Если продукт не предоставляется владельцу, недоступен без оплаты или нет допустимого автоматического пути, записать конкретный blocker и необходимое решение; не объявлять unsupported эквивалентом реализации.

Приоритет — официальный read API с минимальными правами. Браузерный сбор допустим по согласию владельца, только для согласованных действий чтения; сессии защищены, истечение и MFA требуют участия человека. Это не разрешение обходить ограничения или совершать внешние финансовые действия.

## Курсы

[Исследование task-0.7](evidence/fx.md) завершено 2026-09-07. Основной USD/RUB — официальный XML Банка России; Frankfurter v2 допустим только с `providers=CBR` как fallback/cross-check. BTC/USD, ETH/USD, USDT/USD и USDC/USD — отдельные CoinGecko Demo observations: current и история до 365 дней. Кроссы вычисляются через USD точной Decimal-арифметикой; default blended rates и предположение USD/USDT/USDC=1 запрещены. Похожие токены, включая USDC.E, не объединяются без проверенной identity mapping.

TradingView отвергнут: charting libraries не поставляют market data, а terms запрещают automated price referencing/non-display processing. Справочная цена не является исполнимой котировкой. Provider buy/sell требует direction, applicable amount, timestamp, spread/fee из подтверждённого source contract; неизвестные поля остаются unavailable/unknown.

BLK-07 сохраняет FX-B02–FX-B04 до task-0.10: решение для crypto history старше 365 дней, provider executable quotes и keyed CoinGecko Demo probe/attribution. Research Issue может быть закрыт, но task-6.1 и MVP остаются Not Ready.

## Порядок закрытия

task-0.1–task-0.7 формируют конкретные evidence документы, task-0.8 проверяет OpenAI, task-0.9 — инфраструктуру. task-0.10 переносит подтверждённые contracts в спецификацию, уточняет affected tasks и проверяет Ready. Реализация адаптеров зависит от собственного исследования и общего Ready-барьера.

## Два участника и идентичность источника

Каждый провайдер должен поддержать независимые аккаунты участников. Один банковский/криптоаккаунт, повторно авторизованный другим участником, не создаёт второй набор финансовых счетов. Исследование фиксирует устойчивую external-account identity и namespace операций отдельно от connectionId. Оба управляют синхронизацией; ввод MFA/пароля выполняет внешний владелец в защищённом потоке. Это требования, а не подтверждённая возможность сервисов.
