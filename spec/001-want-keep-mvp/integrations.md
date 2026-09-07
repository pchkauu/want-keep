# Интеграции и исследовательские блокеры

[English](integrations.en.md)

Срез публичной документации: 2026-09-06; обновления исследований датированы в строках ниже. Подтверждённое чтение отдельных продуктов, полнота истории и готовность автоматизации оцениваются отдельно. Наличие публичной документации или маркетингового описания не означает доступ к личному аккаунту.

## Обязательное покрытие

| Платформа | Продукты | Подтверждено | Открыто и задача |
| --- | --- | --- | --- |
| Альфа-Банк | Debit/credit cards, current/savings, deposits; кэшбэк | 2026-09-07: [исследование завершено с блокерами](evidence/alfa.md). Live-чтение текущего/накопительного счетов, двух типов вкладов, операций и кэшбэка; опубликованные retail API accounts/cards/operations/loyalty. | BLK-01 открыт: eligibility и проверенный автоматический read-контракт, identity/полнота/reauth, второй аккаунт, отсутствующая кредитка, точные условия и FX/cashback lifecycle. task-0.1 завершает research; task-4.1 и task-0.10 остаются заблокированы. |
| Райффайзенбанк РФ | Debit/credit cards, current/savings, deposits | Есть портал API и сценарии выписок. | Применимость к личному retail аккаунту, договор/права и все продукты; task-0.2 → task-4.2. |
| Ozon Банк | Debit/credit cards, current/savings, deposits | Идентичность сервиса подтверждена решением владельца; публичный личный API данным исследованием не подтверждён. | Проверить именно банк, способы автоматического доступа, все продукты; task-0.3 → task-4.3. |
| Bybit | Funding, Spot, Earn, P2P, futures | V5 документирует wallet balance и журнал Unified account. Журнал UTA имеет ограничения периода/пагинацию и не доказывает покрытие Funding/Earn/P2P. | Полный набор logs/endpoints, права ключа, регион, историю и net/gross semantics; task-0.4 → task-4.4. |
| Aifory Pro | RUB/crypto wallet, exchange, payments, card | Официальный сайт описывает кошелёк, обмены, платежи, карты и веб-приложение. | Контракт чтения личных данных/карты, комиссий, history и quotes; task-0.5 → task-4.5. |
| EMCD | Wallet, Coinhold, P2P, card, mining | Help Center описывает несколько account purposes и историю кошелька. Это не доказательство доступного read API всех продуктов. | Контракты wallet/mining/Coinhold/card/P2P, accrual vs transfer и доступ; task-0.6 → task-4.6. |

Источники: [Alfa developer portal](https://developers.alfabank.ru/), [Alfa onboarding](https://developers.alfabank.ru/products/alfa-api/documentation/articles/connection/connection), [Raiffeisen API](https://developer.raiffeisen.ru/), [Ozon Bank](https://finance.ozon.ru/), [Bybit wallet balance](https://bybit-exchange.github.io/docs/v5/account/wallet-balance), [Bybit transaction log](https://bybit-exchange.github.io/docs/v5/account/transaction-log), [Aifory Pro](https://aifory.pro/), [EMCD wallet](https://help.emcd.io/en/articles/16205516-what-is-emcd-wallet).

Ссылки Ozon и отдельные страницы Alfa при повторном чтении через исследовательский инструмент были недоступны. Это ограничение исследования, не доказательство отсутствия API.

## Результат исследования каждого источника

Для каждого обязательного продукта заполнить матрицу: продукт существует/доступен владельцу; read method; необходимые scopes; account identity/card alias; balances owned/available/locked/debt; events/IDs/revisions/statuses; fees/net-gross; date/timezone; pagination/window/depth; terms/minimum/grace/accrual; quote direction/amount/fee; rate limits; reauth; endpoint allowlist; evidence date; synthetic fixture; live result.

Положительный результат требует фактического сопоставления с авторизованным источником. Если продукт не предоставляется владельцу, недоступен без оплаты или нет допустимого автоматического пути, записать конкретный blocker и необходимое решение; не объявлять unsupported эквивалентом реализации.

Приоритет — официальный read API с минимальными правами. Браузерный сбор допустим по согласию владельца, только для согласованных действий чтения; сессии защищены, истечение и MFA требуют участия человека. Это не разрешение обходить ограничения или совершать внешние финансовые действия.

## Курсы

task-0.7 должен доказать бесплатное покрытие USD/RUB, BTC, USDT и всех необходимых исторических кроссов. Справочная цена не является исполнимой котировкой. Для сервисных buy/sell требуется направление, сумма применимости, timestamp и fee coverage. Неизвестная комиссия не 0; USDT не фиксируется к USD.

В этом пакете конкретный rate provider не выбран: нельзя обещать полную историю или исполнимую цену без подтверждения. Отсутствие бесплатного обязательного источника остаётся блокером.

## Порядок закрытия

task-0.1–task-0.7 формируют конкретные evidence документы, task-0.8 проверяет OpenAI, task-0.9 — инфраструктуру. task-0.10 переносит подтверждённые contracts в спецификацию, уточняет affected tasks и проверяет Ready. Реализация адаптеров зависит от собственного исследования и общего Ready-барьера.

## Два участника и идентичность источника

Каждый провайдер должен поддержать независимые аккаунты участников. Один банковский/криптоаккаунт, повторно авторизованный другим участником, не создаёт второй набор финансовых счетов. Исследование фиксирует устойчивую external-account identity и namespace операций отдельно от connectionId. Оба управляют синхронизацией; ввод MFA/пароля выполняет внешний владелец в защищённом потоке. Это требования, а не подтверждённая возможность сервисов.
