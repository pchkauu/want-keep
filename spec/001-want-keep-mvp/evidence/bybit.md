# Bybit: исследование Funding, Easy Earn и P2P

[English](bybit.en.md) · [task-0.4 / Issue #4](https://github.com/pchkauu/want-keep/issues/4)

## Результат и граница доказательств

Исследование завершено и дополнено авторизованной API-проверкой 2026-09-07. Владелец разрешил RSA-ключ только для чтения с ограничением IP и прошёл MFA. Официальное чтение Funding, Flexible Easy Earn и P2P успешно; **API-доступ подтверждён для этого аккаунта**, включая оба маршрута ордеров P2P. Прежнее предположение об отсутствии P2P-доступа по кабинету уточнено BYBIT-E11–E18. [Авторизованные проверки](bybit-api.md) фиксируют точность, связи и границы истории; для проверенного чтения браузерный сборщик не требуется.

Опорный checkout первичного UI: `f12f21597a9570846e78238b28d6bfc917ba0f4e`; обновление по приватному API основано на `2412b773ad449d87fa846126907daeb237f39bd1`, ветка `docs/want-keep-mvp-sdd`. Основа приложения есть, коннектора Bybit нет. Google Chrome использован для UI и разрешённого выпуска ключа; подписанные запросы отправлены в официальный API. Ключи прочитаны локально для подписи, без вывода и включения в публичные доказательства. Сделки, выводы, подписки, погашения, заявки рекламодателя и сообщения поддержке не отправлялись. Личные остатки, UID, номера ордеров и контрагенты заменены в примерах синтетическими данными.

**task-0.4 завершена как исследование; автоматический коннектор не реализован.** BYBIT-B01/B02/B05 закрыты для наблюдаемого read access. Исходные BYBIT-B03/B04 переданы task-0.10; их решение D-39 и runtime gates task-4.4 записаны в итоговом разделе ниже. Product AC не объявлены пройденными.

## Текущий объём: D-36

| Продукт | Обязательное покрытие | Результат |
| --- | --- | --- |
| Funding | Остатки USDT, USDC, ETH, BTC и все их движения, включая конвертации, on/off-chain поступления, выводы, комиссии и внутренние переводы | Прочитаны четыре остатка, 357 записей за 89 дней, поступления/выводы/Convert; точность и ограничения связей — E12/E13 |
| Easy Earn | Используемые накопления, основной капитал, подписки/погашения, начисление отдельно от выплаты и прогноз | Прочитаны Flexible-позиции USDT/USDC/ETH, 10 ордеров, 141 выплата и 48 почасовых начислений. Fixed-запросы пусты в своих границах; неиспользуемые продукты не блокируют |
| P2P | Личные завершённые/отменённые/ожидающие ордера, крипто/фиатные движения, комиссии, статус и связь с банком | Официальный API успешно вернул список и детали обеих завершённых продаж USDT/RUB. Отменённые/ожидающие состояния и банковское поступление не проверены |
| Остальные продукты | Торговля Spot/UTA, фьючерсы, опционы, Bybit Card, On-Chain/Advanced Earn и другие неиспользуемые продукты | Отложены, не блокируют готовность. Движения через включённые кошельки остаются обязательными |

USDC добавляется как отдельный актив учёта и выбираемой валютной оценки наряду с RUB, USD, USDT, BTC и ETH. Паритет USD/USDT/USDC не предполагается. Похожие символы USDC.E, USDCX, BYUSDT нельзя молча объединять с USDC/USDT. Расширение требует явного решения и проверенных контрактов; универсальный адаптер для гипотетических продуктов не нужен.

## Наблюдения

Таблица ниже сохраняет **первичный этап UI/публичного API**. Отрицательные утверждения о доступе относятся только к тому моменту; BYBIT-E11–E18 в [отчёте авторизованных проверок](bybit-api.md) описывают текущий результат приватных запросов. `confirmed`, `inference`, `unverified` и `contradiction` различаются.

| ID | Доказательство / статус | Вывод и ограничение |
| --- | --- | --- |
| BYBIT-E01 | confirmed, указание пользователя | Funding USDT/USDC/ETH/BTC, периодически Easy Earn и P2P; API приоритетен; остальные продукты отложены без блокировки |
| BYBIT-E02 | confirmed, [dashboard](https://www.bybit.com/ru-RU/dashboard) и управление API | Авторизованный основной аккаунт, индикатор верификации; таблица API-ключей явно пуста. Приватный API, создание ключей и повторная авторизация не проверялись |
| BYBIT-E03 | confirmed, [Funding](https://www.bybit.com/user/assets/home/fiat) | Все четыре символа найдены после отключения фильтра малых остатков и поиска. Есть всего/доступно/в использовании и эквиваленты. Ноль на экране не доказывает точный ноль; исходный фильтр восстановлен |
| BYBIT-E04 | confirmed, [история Funding](https://www.bybit.com/ru-RU/user/assets/records/statements) | Native приход/расход, конвертации, доступный остаток после движения, дата/время, тип/описание; первая страница содержит 20 строк и навигацию по 10 страницам за выбранный месяц. Остаточные суммы точнее карточек кошелька. Полная история и API identity/linkage не проверены |
| BYBIT-E05 | confirmed, [P2P-ордера](https://www.bybit.com/ru-RU/p2p/orderList) | Очередь ожидания пуста; «Все» показывает две завершённые продажи USDT/RUB, полный номер ордера, дату/время, цену и обе суммы. Нет доказательства банковского поступления, базы комиссии, отменённого lifecycle или глубины хранения |
| BYBIT-E06 | confirmed + inference, [программа рекламодателя](https://www.bybit.com/ru-RU/p2p/identify/GA) | Кабинет сообщает об отсутствии разрешения публикации и предлагает заявку; кнопка обычного мейкера недоступна при невыполненных условиях. Вместе с [P2P guide](https://bybit-exchange.github.io/docs/p2p/guide) это не подтверждает текущий доступ к P2P API. Роль не менялась |
| BYBIT-E07 | confirmed, [Earn](https://www.bybit.com/ru-RU/earn/home/) | Округлённая нулевая сводка, предложения Flexible и Fixed. Переход по личному итогу в этой сессии привёл на страницу загрузки приложения. Приложение не устанавливалось; реальные позиции, ордера и начисления не проверены |
| BYBIT-E08 | confirmed, публичные GET в 01:11 UTC | Четыре запроса `/v5/earn/product` вернули по одному продукту FlexibleSaving; история APR USDT — 167 точек. Публичные product ID читаются из каталога, не задаются как константы identity владельца. Доступность с выбранного VPS не доказана |
| BYBIT-E09 | contradiction, [история APR](https://bybit-exchange.github.io/docs/v5/finance/earn/easy-onchain/apr-history) и live-ответ | Таблица полей описывает десятичные доли и полночь; пример и реальный ответ используют строки с %, соседние live timestamps разделены часом. Единицы и частоту сохранять явно; не умножать ставку на 100 и не считать все точки суточными |
| BYBIT-E10 | contradiction / unverified, документация провайдера | Схема yield называет `result.list`, пример — `result.yield`; HTTP-пример перевода отличается от объявленного маршрута; пример created-time внутреннего депозита использует секунды при миллисекундных фильтрах. Это вопросы контракта, а не основание угадывать универсальный parser |

## Матрица официального read-контракта

Матрица объединяет опубликованные контракты и проверенные приватные ответы; [E11–E18](bybit-api.md) задают точный объём live-проверки. Авторизация: [V5 guide](https://bybit-exchange.github.io/docs/v5/guide), проверены RSA-SHA256/base64, `X-BAPI-SIGN` отдельно от `X-BAPI-API-KEY`, timestamp и receive window. Аккаунт принял `api.bybit.com` с разрешённого IP; это не доказывает доступность выбранного VPS и не разрешает обход региональных ограничений.

| Назначение / источник | Метод и маршрут | Существенный контракт / пробел |
| --- | --- | --- |
| [Identity ключа](https://bybit-exchange.github.io/docs/v5/user/apikey-info) | GET `/v5/user/query-api` | `userID`, master/parent identity, `readOnly=1`, permissions, IP/expiry. Ответ также содержит `apiKey`: исключить до evidence/log/AI. Ротация ключа сохраняет внешнюю identity |
| [Остатки Funding](https://bybit-exchange.github.io/docs/v5/asset/balance/all-balance) | GET `/v5/asset/transfer/query-account-coins-balance?accountType=FUND&coin=USDT,USDC,ETH,BTC` | `memberId`, `accountType`, десятичные `walletBalance`, `transferBalance`, bonus по активам. Transferable не означает автоматически доступность для бюджета; пустое/отсутствующее не равно нулю |
| [Журнал Funding](https://bybit-exchange.github.io/docs/v5/asset/fund-history) | GET `/v5/asset/fundinghistory` | Парные `createTimeFrom/To` в секундах, ≤7 дней, строковый limit 1–100, cursor. `currcCursor` документирован как уникальный для dedup; `ioDirection=I/O`, `txnAmt`, `afterAmt`, секундный `createTime`. Business-поля — ключи локализации/текст, не полный enum типов/статусов. Retention и связи журналов не установлены |
| [On-chain поступления](https://bybit-exchange.github.io/docs/v5/asset/deposit/deposit-record) | GET `/v5/asset/deposit/query-record` | Фильтры ms с секундной точностью запроса; окна <30 дней, cursor, ≤50 строк; `id`, chain/txID/txIndex, amount/fee/status/successAt. Одного хеша транзакции недостаточно для identity |
| [Off-chain поступления](https://bybit-exchange.github.io/docs/v5/asset/deposit/internal-deposit-record) | GET `/v5/asset/deposit/query-internal-record` | ≤30 дней, cursor, ≤50; `id`, `txID`, `fromMemberId`, amount, статусы 1/2/3. Внутренний перевод платформы не обязательно внутрисемейный. Address может содержать личный контакт: минимизировать доступ/хранение |
| [Выводы](https://bybit-exchange.github.io/docs/v5/asset/withdraw/withdraw-record) | GET `/v5/asset/withdraw/query-record?withdrawType=2` | Master key; on/off-chain, <30 дней, ≤50, cursor. `withdrawId`, amount, `withdrawFee`, status, created/updated ms, txID. До проведения установить gross/net и актив комиссии; не вычитать комиссию повторно из полного списания журнала |
| [Внутренние переводы](https://bybit-exchange.github.io/docs/v5/asset/transfer/inter-transfer-list) | GET `/v5/asset/transfer/query-inter-transfer-list` | Один UID, `transferId`, native amount, типы счетов from/to, status/time; ≤7 дней, ≤50 строк/cursor. Движения в исключённый счёт сохраняют provenance и требуют классификации второй стороны |
| [История Convert](https://bybit-exchange.github.io/docs/v5/asset/convert/get-convert-history) | GET `/v5/asset/exchange/query-convert-history` | По необходимости `funding` для web/app и `eb_convert_funding` для API; `exchangeTxId`, оба актива/суммы, status/rate/time; index pages ≤100. Web-конвертации документированы с 2025-09-10; полнота старых не доказана. Time filter/terminal cursor не описаны; установить признак завершения |
| [Продукты Flexible](https://bybit-exchange.github.io/docs/v5/finance/earn/easy-onchain/product-info) | GET `/v5/earn/product?category=FlexibleSaving` | Публичный; coin/productId, точность суммы подписки, оценочный APR и доступность. Точность заказа не равна scale журнала; оценка не включает часть вознаграждений и не обещает доход |
| [Позиции Flexible](https://bybit-exchange.github.io/docs/v5/finance/earn/easy-onchain/position) | GET `/v5/earn/position?category=FlexibleSaving` | У Flexible-позиций наблюдается непустой `id`, вопреки примечанию документации только об OnChain; identity привязан к владельцу/category/product, не к глобальному product ID. Нулевой principal совместим с историческим доходом. |
| [Ордера Flexible](https://bybit-exchange.github.io/docs/v5/finance/earn/easy-onchain/order-history) | GET `/v5/earn/order?category=FlexibleSaving` | Earn permission, orderId, Stake/Redeem, amount, status, created/updated ms; ≤7 дней, ≤100 и cursor. Максимальный retention не указан |
| [Выплаченный доход](https://bybit-exchange.github.io/docs/v5/finance/earn/easy-onchain/yield-history) | GET `/v5/earn/yield?category=FlexibleSaving` | Подтверждены `result.list`, уникальный yield `id`, decimal amount и Auto/Manual; пять ручных записей ссылаются на ID погашения. Окна ≤7 дней, ≤100/cursor, документированы три месяца истории. Нулевой yield не требует Funding-зачисления. |
| [Почасовое начисление](https://bybit-exchange.github.io/docs/v5/finance/earn/easy-onchain/hourly-yield) | GET `/v5/earn/hourly-yield?category=FlexibleSaving` | У 48 строк отсутствует обещанный документацией `id`; coin/productId/hourlyDate уникальны и стабильны при повторе выборки. Гарантированный межзапусковой контракт identity не доказан. Начисление не добавляет доход поверх выплаты/Funding. ≤7 дней, ≤100/cursor. |
| [История APR](https://bybit-exchange.github.io/docs/v5/finance/earn/easy-onchain/apr-history) | GET `/v5/earn/apr-history` | Публичный; category/productId, фильтры ms; документация допускает шесть месяцев / ≤182 дня. Для наблюдаемого случая действуют формат % и часовые интервалы E09; полная история не проверена |
| [Продукты Fixed](https://bybit-exchange.github.io/docs/v5/finance/earn/fixed-saving/product), [позиции](https://bybit-exchange.github.io/docs/v5/finance/earn/fixed-saving/position), [ордера](https://bybit-exchange.github.io/docs/v5/finance/earn/fixed-saving/order) | GET `/v5/earn/fixed-term/product`, `/v5/earn/fixed-term/position`, `/v5/earn/fixed-term/order` | Отдельные контракты, если используются: product/category и срок, positionId и погашение, orderId и выплаченные доходы. Позиции не включают закрытые вложения; фильтр ордеров учитывает дату создания Stake и расчёта Redeem, cursor ≤50. Чтение positions/orders успешно с пустыми результатами в своих границах (E14); непустая запись Fixed не получена. Lifetime-отсутствие не предполагается |
| [Доступ P2P](https://bybit-exchange.github.io/docs/p2p/guide), [ордера](https://bybit-exchange.github.io/docs/p2p/order/order-list), [деталь](https://bybit-exchange.github.io/docs/p2p/order/order-detail) | POST `/v5/p2p/order/simplifyList`, `/v5/p2p/order/info` | Оба read POST успешны с readOnly + FiatP2POrder, несмотря на прежнее предположение о рекламодателе. Фактические `ret_code`, строковые ID/quantity/amount/price и даты в ms; side/status — integer. Page/size ≤30, документированы default 90/max 180 дней. Исходный fiat amount приоритетен: quantity × price отличается при наблюдаемом квантовании криптосуммы; банковское поступление проверяется отдельно. |

Правила статусов: [enum провайдера](https://bybit-exchange.github.io/docs/v5/enum). Успешный депозит может впоследствии откатиться; сохранять revision/корректировку. Pending, проверка rollback, unknown, rejected и cancelled не становятся проведёнными молча. P2P 50 — завершён, 40 — отменён, 10/20 — ожидание платежа/выпуска; это само по себе не доказывает соответствующую банковскую запись.

Часовая синхронизация невелика, но backfill соблюдает окна и лимиты каждого маршрута. [Rate limits](https://bybit-exchange.github.io/docs/v5/rate-limit): Funding journal 30/s, все остатки 5/s, внутренние переводы 60/min, on-chain поступления 100/min; использовать заголовки лимита/reset. Общий IP-предел 600/5s; 10006 запускает контролируемый backoff, частотный 403 — документированное ожидание, а не смену домена. Ошибка лимита/прав не превращается в успешное пустое покрытие. Scopes и разрешённое чтение подтверждены E11; throttling и доступность выбранного VPS не проверены.

## Целевые mapping и проверки

Это требования к task-0.10/task-4.4, **не реализованное поведение**:

1. Изолировать household/provider/site/external UID и FUND/asset identity; смена ключа/подключения — provenance. Не объединять супругов по символу/адресу. Разделить identity исходной записи и финансового события. Секреты хранятся в защищённой инфраструктуре, не в AI-контексте. Проверять readOnly и права маршрутов перед импортом; не добавлять write-права ради обхода отказа.
2. Читать Funding как денежный журнал, дополнять deposit/withdraw/transfer/convert/Earn/P2P. До классификации установить статус и связь; не создавать новый эффект на каждый endpoint. Ключи локализации не реализуют критические бизнес-состояния. Неразрешённые записи сохранять с объяснением и сверкой, без угаданного дохода или повторного проведения.
3. Сохранять точные десятичные суммы, включая остатки точнее UI/точности заказа. Bonus, hold, principal и claimable yield различаются. Сверять остатки и движения на согласованный момент; snapshot не создаёт проводку. Null/empty/unsupported отличаются от точного нуля.
4. Основной капитал подписки/погашения — внутреннее движение, заработанный yield — доход один раз; реинвестирование — отдельное связанное движение. [FAQ Easy Earn](https://www.bybit.com/en/help-center/article/FAQ-Easy-Earn) описывает почасовое начисление Flexible и ежедневные выплаты Funding; контракт позиции описывает возврат невыплаченного дохода при погашении. Проверить реальные условия, principal/time basis и корректировки; APR/APY, прогноз и выплата различаются.
5. P2P-обмен связывает native crypto/fiat legs с банковским/наличным счётом; отсутствие банковского доказательства требует уточнения. Внутренний перевод платформы становится семейным только при доказанной принадлежности. Имена/контакты P2P не identity и не доказательство расчёта. Кнопки экспорта/квитанции не устанавливают автоматический API.
6. Checkpoint страницы/окна сохранять только после устойчивой записи; повторять перекрытие, сохранять ID/revisions. Пустой результат за пределами retention/неподдерживаемого продукта не доказывает отсутствие истории. Проверить двух независимых владельцев, переподключение/ротацию ключа, отзыв подключения и отклонение старых заданий.
7. Для подтверждённого покрытия предпочитать V5. Браузерный fallback P2P требует разрешённого структурированного read-контракта, identity, состояний, пагинации и точного allowlist. Read POST выше разрешаются явно; создание ордера, подтверждение оплаты, release, объявления, подписки/погашения, переводы, выводы и изменение ключей запрещены. DOM/OCR финансового экрана не доказывают надёжную автоматизацию.

[Синтетические примеры](bybit.samples.json) содержат двенадцать проектных сценариев, включая проекции проверенных форм с заменой всех личных данных. Это не сырые приватные ответы и не пройденные тесты адаптера. S09–S12 покрывают отсутствие hourly ID, точность остатков, квантование P2P и завершение пагинации.

## Оставшиеся блокеры и передача

| ID | Статус / ответственный | Доказательство закрытия |
| --- | --- | --- |
| BYBIT-B01 | Закрыт для исследовательского доступа | Проверены Chrome, публичное и приватное официальное чтение |
| BYBIT-B02 | Закрыт для этого владельца/ключа | Проверены RSA readOnly, UID, IP-ограничение и нужные права маршрутов. task-0.9 проверяет разрешённый VPS; task-4.4 — истечение/отзыв ключа и изоляцию второго владельца |
| BYBIT-B03 | SDD RESOLVED; RUNTIME GATE task-4.4 | Закрепить детерминированные правила cross-log identity/неоднозначности, изменений статусов, сверки с учётом точности и покрытия выбранной истории. E12/E13/E16/E17 дают конкретные примеры; полная история и все lifecycle не доказаны |
| BYBIT-B04 | SDD RESOLVED; RUNTIME GATE task-4.4 | Закрыть hourly identity/revision, расхождение lifetime totalPnl USDT с доступной историей yield и базу прогноза principal/rate/time. Использование Flexible подтверждено. Fixed/прочие неиспользуемые продукты не блокируют |
| BYBIT-B05 | Закрыт для чтения P2P | Список и обе детали успешно прочитаны read-only API. Заявка рекламодателя и Playwright для проверенного покрытия не нужны. Комиссии/квантование, смена статусов и связь с банком остаются B03/приёмкой реализации |
| BYBIT-B06 | Отложен, не блокирует | Другие продукты исключены D-36; только явное будущее расширение; движения включённых кошельков обязательны |

Исследование передало task-0.7 USDC, task-0.9 access/limits, task-0.10 contract decisions и task-4.4 bounded implementation inputs. task-0.10 завершила общий SDD gate; task-4.4 остаётся владельцем executable conformance. REQ/AC/task IDs сохранены; финансовый runtime и схема БД этим research не менялись.

## Проверка

Выполнено: исследование первичных документов/UI, шесть публичных GET, разрешённый выпуск RSA-ключа и подписанные официальные read-запросы E11–E18, проверки Decimal/replay, двенадцать синтетических сценариев, RU/EN и diff review, `make docs-check` при поставке.

Не выполнено: полная lifetime-история, все статусы/исправления, runtime второго владельца/отзыва, часовой сборщик, выбранный VPS, сопоставление с банковским поступлением, production-коннектор и E2E приложения. Финансовых изменений нет. Успех приватного API — доказательство исследования, не приёмка реализации.

## Решение task-0.10, 2026-09-07

BYBIT-B03/B04 выше закрыты как SDD-решения D-39. Provider IDs остаются route-specific; amount/time cross-log matches — только кандидаты. Для hourly без ID разрешён fallback `(coin, productId, hourlyDate)` внутри hourly namespace. Differing payload сохраняет обе evidence revisions как `source_ambiguous` без financial credit. Gaps и lifetime mismatch дают `source_partial`.

Precision/history, principal/yield reconciliation, второй аккаунт, rotation/revocation и live conformance остаются deployment gate task-4.4. Official RSA read-only API сохраняет приоритет; Playwright допустим только при новом доказанном API-пробеле. Общая SDD Ready, работающий коннектор не заявлен.
