# Контракты и финансовые правила

[English](contracts.en.md)

Контракт проекта версии 8, целевой; task-0.10 признала SDD Ready for development 2026-09-07. Реальных финансовых API или полной схемы БД ещё нет. D-37–D-43 закрывают фундаментальные правила; provider-specific разрешения и conformance остаются entry/deployment gates task-4.x/task-8.x. REQ/AC имеют приоритет над предположением адаптера.

## Доменные сущности

| Сущность | Минимальный контракт |
| --- | --- |
| Money | `amount` — точная десятичная строка, `asset` — код; знак проводки явный. Нет NaN/Infinity/exponent float coercion. |
| Account | ID, householdId, scope personal/household, personalOwnerId при personal, asset, purpose, connection/product reference при импорте, карточные alias, состояние и opening point. |
| SourceRecord | household/provider/external-account/product/log namespace, connection как provenance, source ID или документированная identity strategy, revision/hash, fetchedAt, occurredAt/status, raw evidence reference. |
| Transaction | ID/revision, economic type/state, native postings, cash date, expense attribution date, fees, links, evidence, actorId и human overrides. |
| BalanceSnapshot | account, sourceAsOf, fetchedAt, owned/available/locked/debt с отдельной известностью каждого поля, coverage. |
| Valuation | amount pair, direction, rate, asOf, source, method/reference vs executed quote, fee coverage, revision. |
| Budget / Obligation / ExpectedIncome | month/timezone, currency, category allocation, dated occurrences, planned amount, matched actual amount, approval revision. |
| Goal / Reservation | amount/asset/deadline, virtual или dedicated mode, funding account и уникальная allocation; одна сумма не резервируется дважды. |
| Receipt / Item | attachment, selected account, extraction revision, total/discount/items, merchant, currency/date, validity and matching outcome. |
| AIReview / Proposal / Clarification | subject revision, allowed action, evidence, validated payload, pending/applied/rejected/superseded state, question/answer and usage. |

## Денежный журнал

Каждая проведённая операция атомарно хранит все денежные стороны. Перевод одной валюты сохраняет основную сумму между собственными счетами; обмен хранит разные native amounts и фактический курс. Суммы разных активов не складываются для проверки баланса. Комиссия — отдельный экономический эффект с источником; source net P&L не уменьшается второй раз уже включённой fee.

Начальный остаток, депозит/вывод на собственный счёт, погашение основной суммы кредитки, перемещение в Earn/Coinhold и резервирование цели не являются доходом/потребительским расходом. Проценты, funding, торговый результат и mining reward имеют подтверждённую семантику источника и отдельные показатели. Нереализованный P&L не увеличивает полученный доход.

Pending влияет на доступность через hold, а фактический расход возникает при posted. Неизвестный статус не преобразуется в posted. Source evidence сохраняется независимо от того, удалось ли классифицировать операцию.

Исправления не переписывают оригинал: новая revision и корректирующий эффект/сторно с actor/reason/evidence, ожидаемой версией и ссылкой на предыдущий результат. Повтор команды имеет один эффект. Ручная правка приоритетна над последующим переимпортом того же факта, но не удаляет доказательство противоречия источника.

## Возвраты и валюты

Покупка имеет реальную cash date и бюджетный expense month. Возврат имеет собственную cash date и ссылку на исходный расход; аналитика уменьшает исходный месяц/категорию. Частичный возврат ограничен ещё не возвращённой стоимостью. Если нельзя установить покупку или позиции, создаётся clarification; догадка не пересчитывает историю.

При возврате USD 4 из покупки USD 10 с исходной оценкой RUB 900 исторический расход уменьшается на RUB 360. Реальная сумма поступления/обмена и FX-разница показываются отдельно. Сумма распределения позиций/скидок точно совпадает с оплатой; остаток округления распределяется детерминированно по наибольшим дробным остаткам, при равенстве — по стабильному ID позиции.

Исторический rate snapshot фиксируется по дате операции. Обновление текущих котировок его не меняет; исправление ошибочной исторической цены создаёт аудируемую valuation revision. Неизвестная цена даёт unavailable/partial, а не 0, текущую цену вместо исторической или USD/USDT/USDC=1. Отчёт сохраняет доступ к native amounts.

## Контракт справочных курсов

[Evidence task-0.7](evidence/fx.md) и D-40 выбирают Банк России как основной USD/RUB, Frankfurter v2 только с `providers=CBR` как fallback/cross-check и CoinGecko Demo для отдельных BTC/USD, ETH/USD, USDT/USD и USDC/USD observations current и не старше 365 дней. Default blend и TradingView запрещены.

Для `P_USD(X,D)` — USD за единицу актива — кросс равен `R(S→T,D) = P_USD(S,D) / P_USD(T,D)`. `P_USD(USD,D)=1`, `P_USD(RUB,D)=1/CBR_USD_RUB(D)`. Каждая leg хранит provider asset ID, requested date, observed/effective time, fetchedAt, granularity, source/transport и revision. Расчёт выполняется Decimal; округление — только на границе отображения.

Для CBR берётся последняя effective date `≤ D`; выходной не создаёт observation. CoinGecko history — дневная UTC snapshot для даты операции в timezone бюджета. Crypto history старше 365 дней возвращает `valuation_unavailable`; native amount, source coverage и причина сохраняются. Последний cache можно показать только как stale с датами. Free Demo key, quota/usage и attribution проверяются в task-6.1 перед runtime.

Reference valuation не заменяет фактический обмен или исполнимую котировку. Platform quote существует только при известных direction, applicable amount, provider timestamp и fee/spread coverage; иначе возвращается `quote_unavailable`. Mismatch primary/cross-check сохраняет обе observations и диагностику без скрытого усреднения.

## Дневные лимиты

Расчёт выполняется независимо для каждой валюты и календарного месяца в timezone бюджета. `N` — оставшиеся календарные дни, включая сегодня. Для завершённого месяца дневной лимит недоступен; деления на ноль нет.

- `S`: доступные собственные средства подходящих расходных счетов, уже с учётом проведённых движений и holds. Кредитный лимит, debt, locked funds и выделенные счета целей исключены. Source snapshot согласуется с журналом по времени; неполнота обозначается явно.
- `G`: виртуальные резервы целей внутри `S`. Средства уже исключённого dedicated account повторно не вычитаются.
- `O`: оставшиеся обязательные платежи месяца и запланированные погашения долга. Один платёж не дублируется как minimum и repayment; уже вычтенный из `S` связанный hold повторно не резервируется.
- `R = max(0, Σ plannedFlexible − Σ actualFlexible − U)`, где `U` — ещё не категоризированные подтверждённые расходы. Перерасход одной категории уменьшает общий остаток; нельзя суммировать только положительные остатки и потерять перерасход.
- `K = max(0, min(R, S − G − O))`; доступный общий лимит `K / N`. Негативный остаток/дефицит показывается отдельно; нулевой лимит не скрывает проблему.

Семейный `K` распределяется один раз по положительным остаткам ячеек участник×категория; персональные и категорийные итоги — суммы одной матрицы. Их пределы не независимые кошельки; кросс-валютные эквиваленты — информационный срез. Расход без категории учитывается в общем факте и `U`, а не исчезает из бюджета. При неизвестном соответствии обязательству возможна консервативная дополнительная резервация с явным пояснением до уточнения.

Прогноз строится по дням: `F(d) = S − G + expectedReceipts(≤d) − outstandingPayments(≤d)`. Для равномерного прогнозного лимита `q = max(0, min(R/N, min_d F(d)/elapsedDays(d)))`. Отрицательный `F(d)` показывает кассовый разрыв даже при нулевых гибких тратах. Ожидаемые поступления не увеличивают доступный лимит; перенос даты дохода пересчитывает прогноз, а утверждённый план меняется только по решению участника с соответствующим правом. Выполнение платежа одновременно меняет `S` и незакрытую часть `O`, без второго вычета.

Расчёт точный; округление отображения вниз для безопасного лимита не меняет ledger. Неиспользованные доли остаются в месячном остатке и перераспределяются при следующем расчёте. Резерв цели не расход, возврат прошлого месяца увеличивает текущую ликвидность без выдуманного дохода текущего месяца.

## Кредитки, накопления и доходность

Грейс, minimum/due и eligibility опираются на structured provider fields либо явно подтверждённую владельцем модель условий. Рекламный текст не управляет расчётом. Отсутствие cycle, исключений, базы начисления или порядка погашения делает точный вывод unavailable, но не скрывает известные операции и долг.

Прогноз накоплений использует effective rate schedule, day-count/basis, compounding/payout schedule, term/lock, top-ups/withdrawals и известные условия досрочного выхода. Обещанная ставка — прогноз; фактическое начисление приходит отдельной операцией. Пополнение не является доходностью.

D-42 задаёт XIRR: агрегировать потоки одной календарной даты и использовать Actual/365. Cash-flow amounts остаются точными decimal. Дробную степень считать как `exp((days/365) × ln(1+r))` в decimal context минимум 50 значащих цифр с ROUND_HALF_EVEN; реализация ln/exp и накопления NPV обязана доказать общую численную погрешность `≤ 1e-24 × max(1, Σ|CF_i|)`, иначе результат `unavailable`. После удаления нулевых агрегатов нужны хотя бы один отрицательный и один положительный поток и ровно одна смена знака в хронологическом порядке.

Решать `Σ CF_i / (1+r)^((date_i−date_0)/365) = 0` bracketed bisection между `rLow = -1 + 1e-12` и `rHigh = 1 000 000`. Bracket существует только при разных знаках NPV на границах или попадании границы в tolerance. Остановиться, когда `|NPV| ≤ 1e-12 × max(1, Σ|CF_i|)` либо ширина интервала `≤ 1e-12 × max(1, |rMid|)`; максимум 512 итераций. Вернуть `rMid`, округлённый ROUND_HALF_EVEN до 12 знаков после запятой.

Нет bracket, несколько смен знака, нулевой период, missing valuation, неполная история, недоказанная numeric error bound или отсутствие сходимости возвращают объяснённый `unavailable`, не 0%. Native и reporting-currency результаты раздельны; provider APR не выдаётся за XIRR. Эталоны: `-1000` и `+1100` через 365 дней дают `0.100000000000`; `-1000` и `+1050` через 182 дня дают `0.102795595422`; `-1000/+0.000000001` через 365 дней принимают `rLow`, а `-1/+1000002` требует root выше `rHigh` и даёт `unavailable`. `-100,+230,-132`, один знак и same-day net zero недопустимы.

## API веб-приложения

Целевой префикс `/api/v1`, JSON, Money строками, UTC RFC3339 timestamps плюс явная budget timezone/date. Actor определяется сессией, семейный доступ — проверенным членством, право изменения — сохранённой принадлежностью ресурса. Списки используют cursor pagination. Изменения имеют `Idempotency-Key`; корректировки и подтверждения — `expectedRevision`. Ошибка: `code`, безопасное сообщение, field violations, retryable и correlation ID; без raw provider payload или секретов. Unknown/partial — часть ответа, не скрытый null/zero.

| Группа | Предусмотренный интерфейс |
| --- | --- |
| Auth | POST login options/verify; enrollment options/verify только с bootstrap, действительным приглашением или свежей собственной auth; POST recovery/logout. |
| Accounts | GET/POST `/accounts`, GET `/accounts/{id}`; ручное создание собственных cash accounts и opening point через проверенную команду. |
| Transactions | GET `/transactions`, POST ручной записи, GET `/{id}`; POST `/{id}/corrections` и `/{id}/links` с версией/evidence. |
| Connections | GET/POST `/connections`, POST `/{id}/sync`, DELETE `/{id}`; отдельный защищённый enrollment flow для credentials/session, secret fields не возвращаются. |
| Files/chat | POST `/attachments`, family-authorized GET `/{id}`; GET/POST threads/messages, POST clarification answer. Receipt message требует accountId. |
| Plans/goals | GET/POST budgets/goals, versioned changes; POST `/proposals/{id}/apply` после явного решения уполномоченного участника по текущей версии. |
| Reports | GET dashboard, valuation, daily-limit, credit, savings, returns и insights с filters/date/currency/coverage. Чтение не запускает скрытую мутацию. |
| Notifications | GET in-app notifications, POST read acknowledgment, POST/DELETE push subscriptions. Delivery receipt не означает прочтение. |

task-1.2 материализует этот контракт в OpenAPI только после Ready; точные поля каждой формы следуют моделям выше и проверенным provider contracts. Клиентские/generated типы не становятся доменными.

По D-43 server-owned `ProviderDeploymentAdmission` адресуется `provider + environment`. Его aggregate и repository interface принадлежат application boundary `backend/internal/connections/admission/`, persistence adapter — storage layer task-1.3. Binding содержит `adapterBuildDigest`, `collectorImageDigest`, `contractVersion`, `allowlistRevision`, `nonSecretConfigRevision` и `operatorPermissionRevision`; monotonic `admissionRevision` повышается при каждом изменении state/evidence/binding. task-4.x создаёт provider evidence, task-8.x — host/deployment evidence для того же binding; только application admission service атомарно объединяет оба pass и переводит его в `admitted`. Клиент, AI и provider response не меняют admission.

Connection read model показывает `deploymentGate.status = pending|admitted|blocked`, binding, `admissionRevision`, `checkedAt` и безопасные причины отдельно от `connected|reauth_required`. POST `/{id}/sync` требует `admitted`, точное совпадение binding с запущенными artifacts/config/allowlist/permission и действительное авторизованное connection; admission check и создание job происходят в одной storage transaction. Иначе сервер возвращает `provider_not_admitted` до job или provider IO. Job/result несут неизменяемые binding и revision. Любая смена binding, отзыв permission или failed check атомарно возвращает `pending|blocked`, повышает revision и инвалидирует ещё не начатые jobs; collector повторно сверяет выданные значения перед provider IO. Если смена произошла после начала read, cancel выполняется best effort, а обязательная commit-time проверка current admitted binding/revision идёт в одной транзакции с source revision/posting/outbox. Stale result сохраняется только в quarantine, без source record и финансового эффекта. Pre-admission conformance также работает в quarantine.

### Callback авторизации Raiffeisen

В RAIF-E15 зарегистрирован `https://want-keep.tech/api/v1/connections/raiffeisen/callback`: будущий `GET /connections/raiffeisen/callback` под общим API prefix. Это добавление к защищённому enrollment flow Connections, а не существующий endpoint приложения. Совместимость: путь зафиксирован внешней регистрацией; при его изменении сначала подготовить новый обработчик и обновить регистрацию в банке. RAIF-E16 подтверждает DNS/HTTPS; серверный OAuth-обработчик ещё не реализован, заглушка возвращает 503. Code Flow до проверки обработчика не запускается. Первичный выпуск Refresh-токена в RBO для исследования не заменяет этот контракт. См. [evidence](evidence/raiffeisen.md).

Callback принимает `state` и `code` либо безопасно обрабатывает отказ провайдера. Исключение из общих JSON/Idempotency-Key правил обусловлено браузерным OAuth redirect: это GET с одноразовым состоянием, а не команда финансового журнала. Защищённое начало авторизации создаёт ограниченную по сроку попытку со state, nonce, PKCE S256/verifier и привязкой к household, connection, его версии, текущему principal и externalAccountOwnerId. Срок задаётся конфигурацией; callback требует ту же авторизованную сессию владельца. Параметры URL не назначают пользователя, семью или владельца; отключение подключения, истечение попытки и смена версии запрещают применение результата.

На callback сервер сверяет state и привязки, затем атомарно захватывает действительную попытку один раз и обменивает code с точным зарегистрированным redirect_uri и verifier. Client secret/verifier/токены остаются на сервере; проверка ID token включает подпись, issuer, audience, сроки и nonce по подтверждённому OIDC-контракту. Неопределённый результат обмена не запускает автоматический повтор одноразового code; попытка получает явный статус, а следующий вход создаёт новую попытку без дубликата connection. Code, state и токены исключаются из логов запросов/ошибок, трассировки и аналитики; callback не загружает сторонние ресурсы и возвращает `Cache-Control: no-store`, `Referrer-Policy: no-referrer`. Результат ведёт через 303 на безопасный экран подключения без секретов в URL. Серверное хранение результата позволяет повторному redirect показать статус без повторного обмена.

task-4.2 проверяет успех, отказ банка, отсутствующий/чужой/истёкший state, повтор callback, чужую или истёкшую пользовательскую сессию, отключение/смену версии connection, ошибочный nonce/ID token, неизвестный исход обмена и отсутствие секретов в логах. Это уточняет REQ-048/REQ-073 и AC-048/AC-087; реализация и runtime-проверки остаются впереди.

## Source identity и provider gates

D-39 задаёт ключ исходной записи: `householdId + provider + stableExternalAccountId + productOrLogNamespace + providerRecordId`. `connectionId`, session/profile, cursor и fetch job — provenance. Сумма, время, merchant, текст и локализованная подпись не являются identity.

Если source не даёт record ID, адаптер может использовать только документированный provider-specific immutable composite внутри одного namespace. Payload hash не заменяет identity: он определяет revision/conflict. Повтор с тем же ключом и payload идемпотентен; изменённый payload сохраняется как новая source revision. Два разных факта с одним ключом или неоднозначный composite дают `source_ambiguous`: сохранить обе evidence revisions, создать clarification/reconciliation и не создавать финансовую проводку до решения.

Каждая страница сохраняется до checkpoint. Coverage содержит requested/observed range, next cursor/end reason и gaps. Gap даёт `source_partial`; отсутствие поля balance/fee/status остаётся typed `unknown`, а не нулём. Reconnect находит stableExternalAccountId; один внешний аккаунт не дублируется между сессиями, а разные аккаунты двух участников не смешиваются.

| Provider | Нормализованный namespace и особое правило |
| --- | --- |
| Alfa D-37 | debit/current/savings/deposit/cashback journals раздельны; стабильные account/record IDs должны прийти из structured fixture до deployment. UI selector или название продукта — не identity. |
| Raiffeisen D-35 | Account UUID и number/accountKeys раздельны. CAMT entry допускает 1:N details. Каждый проводимый detail получает versioned `camtCrossReportFingerprint` из полей, доказанно неизменных между перекрывающимися camt.052/camt.053; он всегда canonical providerRecordId. NtryRef/AcctSvcrRef/EndToEndId и statement/report ID — атомарно зарегистрированные aliases/provenance, а не альтернативный ключ. Amount/time исключены. Недостаточный fingerprint, alias→несколько fingerprints или fingerprint→несколько фактов даёт `source_ambiguous`, сохраняет evidence и не создаёт новую проводку. Corrections/reversals — revisions. |
| Ozon D-32 | accountToken/connection и groupID не identity; route-specific record ID живёт в собственном namespace. parent relation связывает fee, но не объединяет эффекты. |
| Bybit D-36 | Route-specific IDs не переносятся между Funding/Earn/P2P. Amount/time matches — кандидаты. Для hourly без ID разрешён `(coin, productId, hourlyDate)` только в hourly namespace; differing payload создаёт collision. |
| Aifory D-33 | office/address/UI path не identity. RUB, crypto и card logs разделены; stable IDs и lifecycle подтверждает structured fixture. |
| EMCD D-34 | aggregate/wallet/Grow/card/P2P namespaces разделены. UI labels, currency list order и approximate valuation не identity. |

Provider deployment выключен по умолчанию. До admission task-4.x доказывает разрешение оператора, read allowlist, structured fixture, identity/revisions/statuses/fees, пагинацию/coverage, reauth, два независимых аккаунта, stale-job rejection и отсутствие write routes. task-8.x доказывает target-host reachability/hardening; Alfa DNS/TLS route дополнительно проверяет task-4.1/task-8.1. D-43 объединяет оба pass только для точного binding. Непройденный или устаревший gate блокирует только этот коннектор.

## Collector и AI

Collector read-job: job ID, connection reference, разрешённый action/product, range/cursor, deadline и короткоживущая привязка авторизации. Вызовы только во внутренней сети с аутентификацией; credentials доступны из изолированного secret store/profile, не из AI payload. Результат: source records, account refs, balance snapshots, next cursor, coverage и typed status/error. Полный secret/session нельзя вернуть в ответе или логе. Неизвестный продукт/поле сохраняется как unsupported/unknown.

Разрешённые AI-команды: classify transaction/items; propose/link verified match; propose/create known transaction; request clarification; skip irrelevant document with reason; propose budget/goal change; explain report. Application, а не модель, решает допустимость исполнения. AI не задаёт actor/owner и не обходит revision/approval/idempotency. Нет payment/trade/SQL/browser/shell tools.

Receipt pipeline: uploaded → validating → processing → clarification / skipped / linked / recorded; failure и waiting-AI отдельны. Исходный документ сохраняется. Начальные технические ограничения: JPEG/PNG/WebP/PDF, 10 MiB на файл, 10 страниц PDF; превышение/неподдерживаемый формат даёт явную ошибку без потери сообщения. Эти пределы проверяются в task-0.8 по стоимости/нагрузке и меняются только через контракт, не скрыто.

Коды существенных отказов: unauthorized, version_conflict, duplicate_command, invalid_money, unsupported_asset, source_reauth_required, source_partial, source_ambiguous, valuation_unavailable, quote_unavailable, command_expired, provider_not_admitted, clarification_required, ai_waiting, ai_budget_exhausted, invalid_attachment, backup_stale. У каждого статуса есть понятное UI-состояние и сценарий AC.

## Семейные сущности, API и действия

| Сущность | Контракт |
| --- | --- |
| User / Household / Membership | Независимые ID; membership связывает пользователя, семью, роль member и статус. `max_active_members=2` — настройка MVP. |
| Resource scope | householdId обязателен; personal/household + personalOwnerId для личных счетов, строк плана и целей. Actor и externalAccountOwnerId — отдельные поля. |
| ExpenseAllocation | transaction/item revision, личное/совместное назначение, карта memberId→amount/share; сумма назначенных долей равна сумме позиции. Неопределённая доля имеет отдельное состояние. |
| BudgetLine | Один Budget на семью/месяц; личная либо общая строка, allocation snapshot и approval revision. Семейный доход хранит фактического получателя без личного ограничения денег. |
| Reimbursement | Явные creditor/debtor member IDs, asset/amount, optional expense link, settlements и revision. Внутренние требования не входят в семейный капитал. |
| SharedThread / Message | Один thread на семью, message actorId, attachments, proposal/clarification revision. Общая видимость не означает полномочия на любую команду. |

Минимальные дополнения `/api/v1`: GET `/me` и `/household`; POST `/household/invitations`, POST `/invitations/accept`; принадлежность в accounts/transactions/budgets/goals; versioned allocation/reimbursement commands; `view=household|member` и memberId для отчётов. Эти фильтры не меняют principal. Unauthorized/forbidden/scope mismatch, invitation_expired/used, member_limit_reached и version_conflict — отдельные безопасные ошибки. Формы материализуются в OpenAPI после Ready.

Инженерные defaults: первый пользователь создаётся закрытым одноразовым bootstrap; второй принимает созданное вошедшим member одноразовое случайное приглашение со сроком 24 часа, хранимое хешированным. Приглашение связывается с новым отдельным входом; повтор/лимит проверяются атомарно. Оно не даёт сбросить чужие passkey. Система не отправляет приглашение через внешние сообщения сама. Изменение пользовательской цели или личной строки, включая удаление, смену владельца/личного статуса и применение AI-предложения, требует её текущего владельца. Нельзя обойти это переводом чужой цели в общую. Создать личную цель/строку можно для себя; общую — любому. У обоих есть чтение/создание/исправление всех учётных операций. Принадлежность личного счёта меняет его владелец, семейного — любой; это не изменяет подтверждённого внешнего владельца и историю операций.

Оба управляют connection: create/sync/disconnect/reauth-request. Secret/MFA submission принимает только проверенную сессию externalAccountOwnerId, отдельно от чата. Инициатор управления и владелец внешнего аккаунта сохраняются. Фоновая AI-проверка использует ограниченного system principal своей семьи; она не наследует право утверждать планы/цели. Proposal исполняется от подтверждающего пользователя с повторной проверкой прав и версии. Уточнения факта операции может закрыть любой, личного плана/цели — его владелец.

Смешанные чеки распределяются по позициям. Приоритет распределения: явное значение позиции, затем покупки, затем строки плана, затем равные доли текущих двух участников для установленного совместного назначения. Неустановленное назначение не подразумевает «общее». Суммы и доли валидируются; округление по наибольшему остатку с tie-break по memberId, с сохранением суммы. Новые правила не переписывают прошлые allocation snapshots. Возврат использует текущее аудируемое распределение исходной покупки/позиций и её историческую оценку; последующее явное исправление покупки согласованно пересчитывает связанные возвраты в той же версии расчёта, новые месячные defaults на прошлое не влияют.

Явный долг может быть частично погашен связанным реальным переводом/наличными. Погашение в другой валюте хранит явно подтверждённые исходную и погашенную суммы; текущая котировка не закрывает долг сама. Превышение непогашенной суммы отвергается как settlement, лишние реально переведённые деньги остаются отдельным движением. Отмена исходного расхода не удаляет долг молча — показывает необходимость подтверждённого исправления.

## Персональные лимиты и резервирование

`S`, `G`, `O`, `R`, `K` и `F(d)` выше относятся ко всей семье в одной валюте. `S` включает доступные деньги личных счетов обоих и семейных счетов; принадлежность не требует реального перевода для семейного обеспечения. Общие цели находятся в отдельном блоке без персональных половин и уменьшают семейную доступность один раз. Взносы в goal не доход/расход.

Для каждой ячейки `(member, category)` определить `r_mc = plannedFlexible_mc − attributedActualFlexible_mc`, `w_mc = max(0,r_mc)`. `U` содержит только ещё не включённые в ячейки подтверждённые расходы; он уменьшает общий `R` один раз. Неопределённое распределение или категория остаётся явным unallocated bucket семейного факта; сумма персональных фактов плюс этот bucket равна семейному факту. Общие расходы распределяются до суммирования, а не повторяются целиком у каждого.

При `W=Σw_mc > 0`: `k_mc = K×w_mc/W`, персональный дневной лимит `Σ_c k_mc/N`, категорийный `Σ_m k_mc/N`. При `W=0` персональные/категорийные лимиты равны нулю. Прогнозный `q` распределяется по той же матрице весов. Округлять отображаемые лимиты вниз; показать нераспределённую дробь отдельно, не увеличивая K. Пример K=1000, N=10, веса A=3000 и B=1000: 75 и 25 в день; семейный лимит 100.

В `F(d)` outstandingPayments — та же непогашенная и не покрытая уже вычтенным hold часть обязательств, что входит в O, разложенная по датам. Все expectedReceipts общие, но даты и источники сохранены. Резерв цели и проверка семейной свободной суммы атомарны; финансовый факт расхода не отвергается из-за превышения резерва, вместо этого фиксируется дефицит без скрытого изменения цели. Публикация общего плана не утверждает чужие личные черновики: личные строки включает владелец, общие — любой участник, с уведомлением второго.

## Контракты представления, команд и событий

UI routes SCR-001–SCR-035 не являются API endpoints. [Каталог экранов](screens.md) задаёт поля FORM-01–FORM-15 и сценарии ошибок; task-1.2 уточняет OpenAPI после Ready. Reports возвращают native amounts, reporting amounts с отдельной известностью, asOf/coverage, actual/forecast/reserved тип, входы расчёта и ссылки на объясняющие операции. Клиент форматирует и раскрывает эти данные, не повторяет финансовые формулы.

Для mutating command сервер связывает Idempotency-Key с householdId, actorId, типом и payload hash; тот же ключ с другим payload отклоняется. Результат и финансовый эффект атомарны. GET `/api/v1/commands/{id}` и `/api/v1/commands/recent` возвращают только разрешённые команды текущего principal с `pending|succeeded|failed`, ссылкой на результат и безопасной ошибкой. `unknown` — знание клиента, не разрешение создать новую команду.

По D-41 terminal command detail/status/result хранится 90 дней после terminal outcome. Unresolved command хранится до reconciliation, затем ещё 90 дней. Минимальный tombstone `(commandId, household, actor, type, key, payloadHash, outcomeRef)` живёт всё unresolved-состояние и 400 дней после terminal/reconciled outcome. `/commands/recent` возвращает terminal за последние 30 дней и все unresolved. После удаления detail известный command возвращает HTTP 410 `command_expired`; живой tombstone по тому же key/hash возвращает outcome reference и запрещает повторный эффект, а другой hash отклоняется. После истечения tombstone replay recovery не гарантируется: клиент создаёт уникальный key и никогда намеренно не переиспользует старый. Финансовые source records, postings, revisions и audit хранятся независимо от command retention.

UIState выводится из typed errors/coverage/result. `version_conflict` содержит разрешённую актуальную версию для сравнения; сервер снова проверяет права при повторном применении. Session expiry закрывает защищённый экран, другой principal не получает черновик. Личные preferences включают locale, reporting currency, notification options и decorativeEffectsEnabled; изменение предпочтения не меняет семейный факт или права.

Motion — подписчик подтверждённых domain events, не владелец проводки. Event хранит ID, householdId, subject/revision, kind, occurredAt, origin (`interactive|live_sync|historical_backfill`), eligibility и последствия коррекции. Accounts/goals/budget определяют бизнес-событие и eligibility через application/outbox, notifications владеет доставкой/персональным presentation ack. Для первого импорта старой истории eligibility false; top-up и goal achievement не выводятся из изменения баланса при чтении. Серверная атомарная claim/ack по event+user исключает два показа в конкурирующих вкладках; потеря подтверждения показа предпочитает статичный результат, а не повтор праздника. Ack эффекта отделён от notification read. Reduced motion/off дают статичный результат; отсутствие animation support не влияет на учёт.

Пример синтетического ответа обзора: «Доступно сегодня RUB 400; ожидается USD 100 через 5 дней; RUB 3 000 зарезервировано на цель». Эти поля не суммируются без явной оценки; клик по RUB 400 раскрывает расчёт K/N и его входы. Числа примера не являются тарифами или личными данными.

## Aifory и ETH: D-33

RUB, USD, USDT, USDC, BTC и ETH доступны в Money/valuation как разные активы; network — отдельный source attribute. Aifory scope: RUB-счета, USDT, ETH и используемая карта USD. RUB aggregate не создаёт второй остаток; USD card и USDT funding — разные native facts. Funding legs, gross/net, rate/fee basis и authorization/clearing/refund lifecycle связываются только по structured evidence.

Другие продукты отложены без блокировки. До deployment task-4.5 получает разрешённый structured fixture, allowlist, stable identity, history/coverage, revisions/statuses/fees, reauth и второй аккаунт. Flutter/UI-текст и похожие pending/confirmed строки не являются контрактом; неоднозначность даёт `source_ambiguous` без двойного списания. [Evidence](evidence/aifory.md).

## EMCD: D-34

Scope D-34: кошелёк USDT, используемые Grow/Coinhold, криптокарты и история P2P. Майнинг и другие неиспользуемые продукты отложены без блокировки. Aggregate, wallet, Grow и card owned/available/reserve не суммируются дважды. Reward/capitalization/payout и authorization/clearing/refund/reversal связываются; неизвестная fee или owner side остаётся unknown. USDT funding, USD card, EUR purchase и approximate valuation — разные факты.

До deployment task-4.6 получает разрешённые structured fixtures каждого журнала, allowlist, stable identity, полную пагинацию/coverage, revisions/statuses/fees, reauth и второй аккаунт. UI-текст и порядок валют не создают проводку; collision даёт `source_ambiguous`. [Evidence](evidence/emcd.md), [синтетические сценарии](evidence/emcd.samples.json).

## Bybit: D-36

Funding USDT/USDC/ETH/BTC, используемый Flexible Easy Earn и P2P обязательны; остальные продукты не блокируют. Официальный read-only API приоритетен. Разрешённые read POST list/detail включаются в allowlist; create/pay/release/ads/transfer/stake/redeem запрещены. Секреты и отражённые provider key fields фильтруются до логов, AI и evidence.

Route-specific provider IDs живут в отдельных Funding/Earn/P2P namespaces D-39. Совпадение суммы/времени между журналами — кандидат связи. Hourly запись без ID использует `(coin, productId, hourlyDate)` только в hourly namespace; differing payload сохраняет обе revisions как `source_ambiguous` без проводки. Principal, distribution и Funding credit учитываются один раз; zero yield не создаёт credit. Сохраняются exact decimals, seconds/ms semantics, cursor даже на короткой странице, coverage/revisions и независимые P2P fiat/quantity/quote; empty fee — unknown.

Ограничения опубликованной истории не расширяются наблюдаемой выборкой. Gaps и lifetime mismatch дают `source_partial`, не корректирующий расход. До deployment task-4.4 проверяет RSA readOnly, precision reconciliation, history/lifecycle, два аккаунта, rotation/revocation и stale jobs. Collector допустим только при новом доказанном API-пробеле. [Матрица](evidence/bybit.md), [API evidence](evidence/bybit-api.md), [проекции](evidence/bybit.samples.json).

## OpenAI: контракт выбора и граница исполнения

[task-0.8 evidence](evidence/openai.md): выбрана gpt-5.6-terra xhigh; зафиксированы тариф, strict proposal schema, лимиты, retention и квалификация. Luna/Sol/MiniMax/DeepSeek не используются автоматически. reasoning.effort=xhigh — решение владельца. Исследовательский JSON не является публичной командой API приложения: gateway преобразует его в AIReview/Proposal, а application повторно проверяет источник, actor/household, expected revision, Money, распределение и полномочия. Неизвестная комиссия отличается от подтверждённого нуля; тип, сумма и месяц возврата не назначаются по догадке. Ответ/отказ/incomplete/ошибка схемы/unknown имеют разные состояния.

Модель, reasoning, разрешённые инструменты и цена задаются серверной конфигурацией, не чатом. У каждой попытки — model/prompt/schema/pricing revision, input count, output cap, reservation, фактический usage и проверенный исход. `cache_write_tokens` не смешивается с cached input; reasoning не оплачивается второй раз поверх output. Период бюджета UTC, незакрытые резервы переживают смену месяца и восстановление. При изменении модели или контракта повторить eval до допуска; автоматический переход на дорогую модель при сетевой ошибке запрещён.

Лимиты загрузки 10 MiB/10 страниц сохранены. Разбиение по страницам не удаляет источник и не создаёт отдельные расходы без сопоставления. Исследование не заменяет серверные regression/authorization/retry проверки. Уточнение относится к целевому AI-контракту; действующего API/хранилища AI ещё нет, миграция данных не требуется.
