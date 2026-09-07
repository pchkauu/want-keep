# Бесплатные источники курсов: исследование и целевой контракт

[English](fx.en.md)

Дата: 2026-09-07, Europe/Moscow. Задача: [task-0.7 / Issue #7](https://github.com/pchkauu/want-keep/issues/7). База репозитория: `cce4a4b`, ветка `docs/want-keep-mvp-sdd`. Проверены публичные документы и анонимные HTTP-ответы; платные планы, API-ключи и финансовые аккаунты не использовались.

**Исследование завершено с точными ограничениями.** Для MVP выбран состав: Банк России — основной USD/RUB; CoinGecko Demo — текущие и не старше 365 дней BTC/USD, ETH/USD, USDT/USD и USDC/USD; Frankfurter v2 с `providers=CBR` — резервный transport/cross-check для CBR, без blended rates. Более старая crypto-оценка и исполнимые котировки платформ остаются unavailable до решения task-0.10 и provider-specific исследований. BLK-07 остаётся открыт под контролем task-0.10; закрытие research Issue не означает Ready.

## Решение

| Назначение | Источник | Граница использования |
| --- | --- | --- |
| Текущий и исторический USD/RUB | [XML Банка России](https://www.cbr.ru/development/sxml/) | Основной официальный reference rate. Хранить дату запроса, effective date из ответа, fetchedAt и исходный decimal. Выходной/праздничный день использует последнюю effective date `≤ D`; это не новая котировка на каждый календарный день. |
| Резерв USD/RUB | [Frankfurter v2](https://frankfurter.dev/) только с `providers=CBR` | Использовать при недоступности прямого CBR и для cross-check. Запрещён default blend. Сохранять фактического provider `CBR`; Frankfurter является transport/provenance. Более старая effective date не заменяет более новую запись CBR. |
| Текущие BTC, ETH, USDT, USDC в USD | [CoinGecko Demo](https://www.coingecko.com/en/api/pricing), `/simple/price` | Один batch по IDs `bitcoin,ethereum,tether,usd-coin` с `include_last_updated_at=true`; не использовать symbol-only identity. USDT и USDC берутся как отдельные assets, без peg `1 USD`. USDC.E не объединяется с USDC без проверенной provider mapping. |
| Исторические BTC, ETH, USDT, USDC в USD | CoinGecko `/coins/{id}/history` | Одна дневная UTC snapshot на asset/date, не старше 365 дней по бесплатному плану. Хранить requested date, returned timestamp/день, granularity и source. Это reference valuation, не цена исполнения. |
| Кросс-курсы отчёта | Вышеприведённые USD legs | Вычислять точной Decimal-арифметикой. Если хотя бы одна leg отсутствует/устарела по контракту, результат partial/unavailable; не подставлять 0, текущий курс в историю или USD/USDT/USDC=1. |
| Реальный обмен, buy/sell, P2P | Источник самой операции | Сохранять обе native amounts, direction, applicable amount, provider timestamp и известные fees. Reference source не создаёт spread/fee/executable quote. Отсутствующие поля остаются unknown. |

Этот состав бесплатен в исследованном масштабе. CBR не публикует quota в просмотренной документации; это означает `unknown`, а не unlimited. Frankfurter не имеет дневной/месячной quota, но применяет anti-abuse limiting. CoinGecko Demo на дату проверки: 10 000 calls/month, 100/min, один API key, freshness от 60 секунд, обязательная атрибуция и до одного года daily/hourly history. Условия и лимиты проверяются при реализации и показываются в состоянии источника.

## Воспроизводимое правило оценки

Для source asset `S`, target asset `T` и момента/даты `D` хранится `P_USD(X,D)` — USD за одну единицу `X`. Для USD значение равно `1`; для RUB это `1 / CBR_USD_RUB(D)`; для BTC, ETH, USDT и USDC — отдельные CoinGecko observations.

`R(S→T,D) = P_USD(S,D) / P_USD(T,D)`

`value_T = amount_S × R(S→T,D)`

Расчёт использует Decimal и явную политику округления только на presentation boundary. Все исходные observations и обе даты сохраняются. Разные effective dates у legs не скрываются: отчёт показывает source/asOf каждой leg и coverage.

Историческая операция получает календарную дату в timezone семейного бюджета. CBR leg — последняя официальная effective date `≤ D`. CoinGecko leg — дневная snapshot для этой даты UTC. Текущий остаток использует последний успешно полученный current observation; ответ без provider timestamp не считается новым наблюдением. Исправление источника создаёт valuation revision и не переписывает старую оценку молча.

Синтетический пример: если источник сообщает `BTC/USD = 50000.00`, `USDT/USD = 0.9970`, а CBR сообщает `USD/RUB = 90.0000`, то `BTC/USDT = 50000.00 / 0.9970`. Нельзя заменить знаменатель на `1`, а reference-результат нельзя выдавать за цену, по которой Bybit/Aifory/EMCD выполнит обмен.

## Реестр доказательств

`confirmed` относится только к указанному документу или HTTP-наблюдению; `inference` — к проектному выводу; `unverified` — к отсутствующему доказательству. Реальные курсы не включены в примеры.

| ID | Статус и источник | Подтверждено | Предел доказательства |
| --- | --- | --- | --- |
| FX-E01 | confirmed, [CBR XML](https://www.cbr.ru/development/sxml/) | `XML_daily.asp` возвращает последний зарегистрированный день или выбранную дату; `XML_dynamic.asp` — диапазон по коду валюты | Документ не публикует quota/SLA и не даёт crypto prices или executable bank quote |
| FX-E02 | confirmed, [CBR о сайте](https://www.cbr.ru/about/) и [пользовательское соглашение](https://www.cbr.ru/user_agreement/) | Открытые материалы допускают воспроизведение со ссылкой на источник | Проверять актуальные условия перед production; сохранять attribution |
| FX-E03 | confirmed, live GET 2026-09-07 | Current XML: HTTP 200, Windows-1251, effective date 2026-09-05, USD `Nominal/Value/VunitRate`; выбранная историческая неделя: пять observations | Это точечный доступ с текущей сети, не SLA; заголовки CBR содержали DDoS cookies/IP и не сохранены в Git |
| FX-E04 | confirmed, [Frankfurter docs](https://frankfurter.dev/) и [OpenAPI](https://api.frankfurter.dev/v2/openapi.json) | No key, historical/range API, provider filter, no daily/monthly quota, anti-abuse rate limiting; default — blend | Условия underlying provider сохраняются; сервис не предназначен для live trading |
| FX-E05 | confirmed, live GET 2026-09-07 | `USD/RUB?providers=CBR` и исторический диапазон отдали typed JSON; provider metadata обозначает CBR и daily data | Frankfurter observation была на effective date 2026-09-04, когда прямой CBR уже отдал 2026-09-05; transport может отставать |
| FX-E06 | confirmed, [CoinGecko pricing](https://www.coingecko.com/en/api/pricing) | Demo: $0, 10k calls/month, 100/min, freshness from 60 sec, one key, attribution, one year daily/hourly history | Тариф и лимиты могут измениться; для стабильного Demo нужен создаваемый владельцем key |
| FX-E07 | confirmed, [simple price](https://docs.coingecko.com/reference/simple-price) и [history](https://docs.coingecko.com/reference/coins-id-history) | Current endpoint поддерживает batch и `last_updated_at`; history возвращает market data на указанную дату | Aggregated reference market price, не execution quote отдельной платформы |
| FX-E08 | confirmed, live GET 2026-09-07 | `bitcoin`, `ethereum`, `tether` вернули отдельные USD/RUB и `last_updated_at`; `usd-coin` отдельно вернул USD + `last_updated_at` и historical USD/RUB/BTC/ETH legs | Keyless public endpoint проверен, Demo-key flow не проверен; наблюдение USDC не доказывает USDC.E identity |
| FX-E09 | confirmed, live boundary probes | 365-day BTC range вернул дневные точки; более ранняя дата/366 days вернули 401 plan-limit | Старше 365 дней бесплатный CoinGecko contract не доказан |
| FX-E10 | confirmed, [TradingView datafeed docs](https://www.tradingview.com/charting-library-docs/latest/connecting_data/) и [libraries](https://www.tradingview.com/free-charting-libraries/) | Advanced Charts/Trading Platform не предоставляют market data; Lightweight Charts — библиотека визуализации; widget data остаётся iframe display | Наличие графика/символа не является API или правом серверной оценки |
| FX-E11 | confirmed, [TradingView Terms §3](https://www.tradingview.com/policies/) | Market data лицензируется display-only; non-display processing и automated price referencing запрещены | TradingView отвергнут как источник Want Keep без отдельного письменного data agreement |
| FX-E12 | confirmed, [Coin Metrics Community](https://gitbook-docs.coinmetrics.io/packages/coin-metrics-community-data) и live catalog | No-key community access; ReferenceRateUSD есть для BTC/ETH/USDT, но community contract/catalog ограничены последними семью observations | Допустим как краткосрочный cross-check после повторной проверки лицензии; не закрывает history |
| FX-E13 | confirmed, [Kraken OHLCVT downloads](https://support.kraken.com/hc/en-us/articles/360047124832-downloadable-historical-market-data-time-and-sales-) и [OHLC API](https://docs.kraken.com/api/docs/rest-api/get-ohlc-data/) | Бесплатные archives заявляют candles/trades; REST OHLC ограничен последними 720 entries | Наличие нужных пар в archive и допустимость использования для семейного сервиса не проверены; не выбран |
| FX-E14 | confirmed, [Coinbase Market Data Terms](https://www.coinbase.com/legal/market_data) | Public market API существует, но terms ограничивают multi-party/redistribution и AI use | Не выбран для семейного AI-приложения без отдельного согласия |
| FX-E15 | confirmed, [Alpha Vantage limits](https://www.alphavantage.co/support/#api-key) | Free tier ограничен 25 requests/day | Не закрыты USDT, полная глубина и права совместного сервиса; не выбран |
| FX-E16 | confirmed, [Globalping HTTP-пробы](https://globalping.io/docs/api.globalping.io) 2026-09-07 | CBR и CoinGecko: HTTP 200 из DE/NL/BG. Frankfurter: DE/BG 200, одна NL DNS failure; повтор тремя NL networks — 3×200 | Measurement IDs `2fsM1iXZRutVZQVSb000215ZM`, `2qCuOHSFPrYPQ99xT000215ZM`, `2ZqixcPO00hNbgB50000215ZM`, `2XSpFloX7zMGNRaNO000215ZN`; snapshot не является uptime/SLA |
| FX-E17 | inference, usage calculation | Один hourly CoinGecko batch — около 744 calls за 31 день; максимум 124 asset/day history calls для четырёх assets за 31-day backfill. Вместе значительно ниже 10k до retries/manual ranges | Реализация обязана считать фактические calls, ограничивать retries и не считать расчёт runtime-доказательством |
| FX-E18 | confirmed gap, provider research | Ни один выбранный reference source не сообщает executable buy/sell quote с amount, spread и fee конкретного аккаунта | Эти поля приходят только из подтверждённого provider contract; отсутствие означает unavailable/unknown |

## TradingView: результат гипотезы

TradingView **не подходит** как серверный источник курсов для Want Keep. Charting libraries требуют собственный datafeed, Lightweight Charts решает только визуализацию, а widgets не дают контракт извлечения данных. Публичные terms отдельно запрещают автоматическое price referencing и другую non-display обработку. Поэтому адаптер, scraping, webhook или скрытый widget-bridge для валютной оценки не проектируются. Позднее можно отдельно рассмотреть open-source Lightweight Charts для UI, если это будет нужно экрану; это не часть rate contract и не новая зависимость текущей задачи.

## Failure, cache и аудит

- Raw response сохраняется как защищённое source evidence либо content hash; публичные примеры не содержат cookies, IP, key или реальные финансовые данные.
- Cache key включает provider, provider asset ID, base/quote, observation/effective date и granularity. Текущий и исторический namespace не смешиваются.
- 401/403 означает plan/auth/configuration error; 429 — rate-limited с backoff; timeout/5xx — source unavailable. Неизвестный outcome сначала сверяется, затем повторяется идемпотентный GET.
- Последнее значение можно показывать как stale с source/effective/fetched dates. Оно не становится новым current observation.
- Cross-check mismatch сохраняет обе observations и поднимает диагностику. Автоматически выбирать удобное значение или усреднять CBR/market data нельзя.
- CoinGecko key и любые будущие provider credentials хранятся вне Git; AI их не получает.

## Блокеры и передача

| ID | Статус | Нужное решение и владелец закрытия |
| --- | --- | --- |
| FX-B01 | CLOSED | Current и история до 365 дней для RUB/USD/BTC/ETH/USDT/USDC закрыты выбранным reference contract и формулой cross-rates; USDC.E требует отдельной identity/rate mapping |
| FX-B02 | OPEN | task-0.10: принять `valuation_unavailable` для crypto старше 365 дней либо отдельно проверить разрешённый archive/платный источник. Текущий курс/peg не допускаются |
| FX-B03 | OPEN | task-0.1–task-0.6/task-0.10: получить provider-specific executable buy/sell, amount, fee/spread и timestamp или утвердить UI `quote unavailable`; reference price не заменяет их |
| FX-B04 | OPEN | владелец + task-0.10: создать бесплатный CoinGecko Demo key, проверить keyed endpoints/usage endpoint и зафиксировать атрибуцию. Секрет не публиковать |
| FX-B05 | CLOSED FOR RESEARCH | DE/NL/BG reachability выбранных endpoints подтверждена датированными probes; task-0.9 всё ещё проверяет фактический VPS/runtime |
| FX-B06 | CLOSED | Гипотеза TradingView проверена и отвергнута для non-display valuation |

BLK-07 содержит FX-B02–FX-B04 до решения task-0.10. task-0.7 завершает исследование и разблокирует формализацию контракта, но task-6.1 и MVP остаются **Not Ready** до общего Ready-барьера.

## Проверка

Выполнено: официальные документы и terms, anonymous endpoint probes, current/history/limit boundaries, отдельные USDT/USDC observations, CBR/Frankfurter effective-date mismatch, DE/NL/BG reachability и RU/EN contract review. Использованы только публичные данные; rates из live-ответов не опубликованы.

Команды документации: `make docs-check`, `python3 -m unittest discover -s spec/001-want-keep-mvp/tools -p 'test_*.py'`, `git diff --check`. Они проверяют SDD, а не финансовый runtime.

Не выполнено: CoinGecko Demo key flow/usage readback, длительный soak/SLA, реальный VPS, provider buy/sell/fees, crypto history старше 365 дней, application/collector/DB и полные AC-037/AC-038/AC-039/AC-074. Причина — отсутствующие секреты, provider contracts и приложение. Эти ограничения отражены в FX-B02–FX-B04, а не выданы за успешную реализацию.
