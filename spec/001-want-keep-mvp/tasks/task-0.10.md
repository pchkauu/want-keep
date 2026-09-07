<!-- want-keep-task: task-0.10 -->
# task-0.10 — Закрыть контракты и проверить готовность SDD / Close contracts and review SDD readiness

## RU

Получить Ready-спецификацию перед реализацией приложения.

**Состояние:** Заблокировано зависимостями и проверкой SDD Ready; реализация не начата.

**Зависимости:** `task-0.1`, `task-0.2`, `task-0.3`, `task-0.4`, `task-0.5`, `task-0.6`, `task-0.7`, `task-0.8`, `task-0.9`.

**Тип:** `specification`.

### Изменение и контракты

Свести результаты исследований, закрыть контракты, поля, источники и фундаментальные вопросы. Обновить RU/EN и все затронутые задачи. При блокерах сохранить Not Ready и не создавать plan.md. После Ready создать решение-полный план по SDD с точными проверками и отправить независимому read-only reviewer; только свежий Ready допускает реализацию. Закрыть сроки хранения command status/idempotency для retry/recovery, не смешивая их с бессрочным финансовым audit. Для Aifory действует D-33: закрыть AIFORY-B02–B04 только для RUB, USDT, ETH и используемой карты; AIFORY-B05/остальные продукты не блокируют. Согласовать структурированный read-контракт, право автоматизации, identity/history и card lifecycle; учесть ETH в оценке. Для Bybit по D-36 использовать успешное RSA readOnly чтение evidence/bybit-api: BYBIT-B02/B05 закрыты для владельца, включая P2P. Закрыть BYBIT-B03/B04: детерминированную identity/связь журналов и неоднозначность, precision-aware сверку, границы истории, отсутствующий hourly ID/корректировки и базу прогноза Earn/расхождение lifetime totalPnl USDT. VPS проверяет task-0.9; второй владелец, отзыв и executable conformance — task-4.4. Funding USDT/USDC/ETH/BTC, Flexible Easy Earn и P2P обязательны; Fixed/прочие неиспользуемые BYBIT-B06 не блокируют. USDC остаётся отдельной валютой оценки. Для Raiffeisen по D-35 закрыть RAIF-B02/B03/B04/B06: CAMT mapping и ID при исправлениях, глубину/полноту истории, текущие/доступные/заблокированные остатки и комиссии, auth lifecycle и независимость аккаунтов. Использовать evidence/raiffeisen; RAIF-B01/B05 закрыты только в указанном объёме. Закрытие task-0.2 не разблокирует task-4.2 без Ready.

### Границы изменений

- `spec/001-want-keep-mvp/constraints.md`
- `spec/001-want-keep-mvp/contracts.md`
- `spec/001-want-keep-mvp/verification.md`
- `spec/001-want-keep-mvp/plan.md`

Это планируемые пути. Общие контракты: `spec/001-want-keep-mvp/contracts.md`; архитектура и команды: `constraints.md`. Менять только владельца поведения и затронутые тесты; при незакрытом контракте обновить evidence и остановить зависимую реализацию.

### Связанные требования

- **REQ-020:** AI-инсайты по доходам и расходам ссылаются на проверяемые данные и отделяют прогноз от факта.
- **REQ-031:** Кредитные карты показывают задолженность, собственные средства, лимит, минимальный платёж и дату по данным источника.
- **REQ-032:** Грейс-период опирается на условия конкретной карты и показывает сумму и срок сохранения льготы.
- **REQ-033:** Накопления показывают фактические начисления и прогноз по ставкам, срокам, капитализации и денежным потокам.
- **REQ-034:** Доходность вложений сравнивается с учётом дат денежных потоков и валюты оценки.
- **REQ-038:** Курсы обмена учитывают направление, сервис, время, сумму применимости и известные комиссии.
- **REQ-039:** Отсутствующие курсы и неподдерживаемые активы не превращаются в нулевые суммы или условный паритет USD/USDT/USDC.
- **REQ-042:** Интеграция Альфа-Банк автоматически читает дебетовые/кредитные карты, текущие/накопительные счета и вклады в пределах подтверждённого контракта.
- **REQ-043:** Raiffeisen через RBO API читает только расчётный счёт ИП: остатки, поступления, списания, комиссии и историю (D-35).
- **REQ-044:** Интеграция Ozon Банк автоматически читает дебетовую карту и связанный основной счёт: остатки, операции и доступные сведения в пределах подтверждённого контракта. Другие продукты Ozon отложены до расширения контракта.
- **REQ-045:** Bybit автоматически читает Funding USDT/USDC/ETH/BTC, используемый Easy Earn и P2P; официальный API приоритетен. Остальные продукты отложены без блокировки по D-36.
- **REQ-046:** Aifory Pro автоматически читает RUB-счета, USDT, ETH и используемую криптокарту, включая движения и комиссии этих продуктов. Остальные продукты отложены и не блокируют MVP.
- **REQ-047:** Интеграция EMCD автоматически читает используемые криптокарты, Coinhold/Grow, кошелёк USDT и исторические P2P-ордера по D-34; майнинг никогда не использовался и вместе с другими неиспользуемыми продуктами отложен без блокировки.
- **REQ-051:** AI ограничен бюджетом $50/месяц и деградирует в очередь ожидания без остановки обычного учёта.
- **REQ-055:** Веб-приложение предназначено для ноутбука macOS в Chrome и Arc; изменение окна и масштаба сохраняет доступность ежедневного учёта.
- **REQ-056:** Развёртывание укладывается в $40/месяц на сервер в DE/NL/BG; отдельные платные источники не используются.
- **REQ-062:** Архитектура использует Go/PostgreSQL, React/TypeScript/Vite и отдельный Playwright-сборщик с зависимостями к домену.
- **REQ-063:** Пользователь, семья и членство моделируются отдельно; ограничение двух участников задаётся конфигурацией.
- **REQ-064:** Оба участника видят все финансовые данные и изменяют операции; личные цели и части плана изменяет только их владелец.
- **REQ-065:** Принадлежность счёта, владелец внешнего аккаунта, автор записи и принадлежность расхода являются отдельными признаками.
- **REQ-066:** Все доходы и доступные средства входят в семейный пул; общий бюджет и личные разрезы используют один финансовый факт.
- **REQ-067:** Расходы и позиции чеков имеют личное или совместное назначение; общая доля по умолчанию 50/50 с исключениями статьи или покупки.
- **REQ-068:** Взаимный долг учитывается только по явному указанию и не увеличивает активы или расходы семьи.
- **REQ-069:** Резервы личных и совместных целей задаются явно; совместные цели отображаются отдельным общим блоком без персональных долей.
- **REQ-070:** Сумма индивидуальных дневных лимитов не превышает семейный предел одной валюты; счёт плательщика не меняет долю расходов.
- **REQ-071:** Один общий чат сохраняет автора сообщения и проверяет полномочия инициатора AI-команды при исполнении.
- **REQ-072:** Конкурирующие изменения, ответы на уточнения и отмены проверяют версию и текущие права, сохраняя обоих авторов.
- **REQ-073:** Оба управляют подключениями; банковскую авторизацию выполняет владелец внешнего аккаунта без раскрытия секретов партнёру или AI.
- **REQ-074:** Изменения плана и целей уведомляют второго участника; прочтение и push-подписки принадлежат конкретному пользователю.
- **REQ-075:** Восстановление данных сохраняет пользователей, членство, принадлежность, роли, историю и общий семейный учёт.
- **REQ-076:** Семейная область проверяется для API, файлов, AI, фоновых задач и внешних ID независимо от присланных actor/owner.

### Критерии приёмки

Связь с критерием задаёт покрытие; исследование или частичная задача не доказывает весь критерий продукта. Точный результат этой задачи указан ниже в проверке.

#### AC-032

- **Дано:** Для карты подтверждены условия, выписка, исключения и крайняя дата.
- **Когда:** Совершаются покупка, частичное погашение и операция, исключённая из льготы.
- **Тогда:** Сумма и срок согласованы с подтверждёнными условиями; при нехватке условий отображается неизвестность, а не обещание сохранения льготы.
- **Уровень:** `contract`.

#### AC-033

- **Дано:** Есть вклад или Earn с подтверждёнными условиями, пополнением и выводом.
- **Когда:** Рассчитывается доход за период и прогноз.
- **Тогда:** Факт отделён от прогноза и переоценки; смена ставки и капитализация учитываются по условиям; неизвестные условия блокируют точный прогноз.
- **Уровень:** `integration`.

#### AC-034

- **Дано:** Два вложения имеют разные даты пополнений и одинаковый конечный остаток.
- **Когда:** Строится сравнение доходности.
- **Тогда:** Показаны фактический доход и годовая денежно-взвешенная доходность с датами/методом; некорректные или неоднозначные расчёты обозначены недоступными, не нулём.
- **Уровень:** `unit`.

#### AC-038

- **Дано:** Два сервиса дают разные bid/ask и один не раскрывает комиссию.
- **Когда:** Владелец сравнивает RUB → USDT.
- **Тогда:** Показаны сопоставимые направления и свежесть; неизвестная комиссия не считается нулевой; недоступная котировка не заменяется обещанием рыночного курса.
- **Уровень:** `integration`.

#### AC-042

- **Дано:** Подключён разрешённый личный аккаунт Альфа-Банк с тестируемыми продуктами.
- **Когда:** Запрошены счета, остатки, операции и необходимые условия продуктов.
- **Тогда:** Для каждого обязательного продукта получены сопоставимые с источником данные и свидетельство чтения; отсутствие доступа фиксируется блокером, а не успешным покрытием.
- **Уровень:** `contract+manual`.

#### AC-043

- **Дано:** Подключён разрешённый расчётный счёт ИП в RBO API.
- **Когда:** Запрошены остатки и движения, повторный импорт и intraday no-statements.
- **Тогда:** Данные совпадают с источником; дублей нет, комиссии учтены отдельно. Неизвестный текущий остаток не равен нулю: видны последний подтверждённый остаток, его дата и пробел покрытия.
- **Уровень:** `contract+manual`.

#### AC-044

- **Дано:** Подключён разрешённый личный аккаунт Ozon Банк с дебетовой картой и связанным основным счётом.
- **Когда:** Запрошены остатки, операции и доступные сведения дебетового продукта; та же карта и счёт встречаются в нескольких представлениях.
- **Тогда:** Данные сопоставимы с источником, свидетельство чтения сохранено, карта не удваивает остаток счёта. Недоступность обязательных полей дебетового продукта отмечена явно. Отсутствие кредитки, накоплений или вкладов Ozon не блокирует MVP: эти продукты вне текущего контракта и не показаны как реализованные.
- **Уровень:** `contract+manual`.

#### AC-045

- **Дано:** Безопасно подключён read-only аккаунт Bybit с подтверждёнными контрактами Funding, используемого Easy Earn и P2P.
- **Когда:** Читаются остатки, история с выбранной даты, повторные страницы, Convert, начисление/выплата Earn и P2P; моделируется отказ прав выбранного продукта.
- **Тогда:** Точные native amounts, ID, статусы, комиссии и coverage сопоставимы с источником; Funding и детали не удваивают обмен, комиссию или доход. P2P связывает crypto/fiat с банком либо требует уточнения. Отказ доступа/неполная история явно блокируют соответствующее покрытие; отсутствие Spot/UTA trading, futures, карты, On-Chain/Advanced Earn и иных неиспользуемых продуктов не блокирует.
- **Уровень:** `contract+manual`.

#### AC-046

- **Дано:** Подключён разрешённый личный аккаунт Aifory с RUB-счетами, USDT/TRC-20, ETH/Ethereum и используемой картой USD (D-33); другие продукты не подключены.
- **Когда:** Повторно прочитаны остатки и история, обмен RUB/USDT, ETH-вывод с комиссией, пополнение карты USDT/USD и похожие pending/confirmed оплаты с отдельной fee.
- **Тогда:** Каждый включённый продукт имеет структурированное доказательство чтения; RUB-группа не дублирует дочерние счета, ETH точен. Движения и комиссии учтены один раз по подтверждённым ID/связям/статусам; знак UI и паритет USDT/USD не предполагаются. Неизвестная связь требует уточнения. Пробелы включённых продуктов блокируют адаптер; остальные продукты не требуются, но их движения по выбранным кошелькам не пропускаются.
- **Уровень:** `contract+manual`.

#### AC-047

- **Дано:** Подключён разрешённый личный аккаунт EMCD с кошельком USDT, действующими Grow, существующими картами, включая заблокированную, и историей P2P. Майнинг не использовался ни сейчас, ни ранее.
- **Когда:** Запрошены счета, остатки, операции и необходимые условия продуктов.
- **Тогда:** По каждому включённому продукту подтверждены сопоставимые с источником данные и автоматическое чтение: сводки не дублируют дочерние остатки, начисление/капитализация/выплата не утраивают доход, отказ карты не расход по основной сумме, P2P связан с денежными сторонами без дубля. Неизвестные поля/история отмечены явно; отсутствие контракта выбранных продуктов блокирует адаптер. Майнинг и другие неиспользуемые продукты не требуются.
- **Уровень:** `contract+manual`.

#### AC-056

- **Дано:** Выбран конкретный тариф, регион, валюта счёта и налоги.
- **Когда:** Проверяется эксплуатационная смета для сотен операций в месяц.
- **Тогда:** Зафиксирована датированная полная смета сервера не выше $40; OpenAI учитывается отдельно до $50; обязательный платный источник остаётся блокером.
- **Уровень:** `manual`.

#### AC-062

- **Дано:** Создана структура приложения и контракты компонентов.
- **Когда:** Проверяются зависимости и публичные интерфейсы.
- **Тогда:** Домен не импортирует HTTP, SQL, UI, OpenAI SDK или браузерные типы; адаптеры маппят внешние модели; сборщик не владеет финансовыми решениями.
- **Уровень:** `static`.

#### AC-070

- **Дано:** Банк передаёт баланс, но не условия грейса; ставка Earn имеет неизвестную базу начисления.
- **Когда:** Открываются прогнозы.
- **Тогда:** Баланс отображается; льгота и точный прогноз имеют причину недоступности; AI не извлекает гарантированную бизнес-логику из рекламной формулировки.
- **Уровень:** `contract+end-to-end`.

#### AC-076

- **Дано:** Подготовлен синтетический набор сотен операций со сложными переводами, чеками и эталонными ответами.
- **Когда:** Выполняются AI-eval, нагрузочная проверка и ручной дневной сценарий.
- **Тогда:** Отчёт показывает ошибки, уточнения, латентность, токены/стоимость и время пользователя; финансовые инварианты проходят, бюджет оценивается по измерению; непроверенное качество не объявлено доказанным.
- **Уровень:** `manual+integration`.

#### AC-077

- **Дано:** Два пользователя состоят в одной семье.
- **Когда:** Проверяются схема, авторизация и ограничение членства.
- **Тогда:** Нет полей partner1/partner2 и ветвлений по конкретным пользователям; роли и принадлежность отделены от личности. Выход, замена и новые роли не реализованы.
- **Уровень:** `integration`.

#### AC-078

- **Дано:** У A есть личная цель и статья плана; у семьи общая статья и операции обоих.
- **Когда:** B читает все данные, исправляет операцию A и общий план, затем пытается изменить личную цель/план A через API и AI.
- **Тогда:** Чтение, операции и общее изменение разрешены; личные план/цель A защищены сервером. Одного уполномоченного подтверждения достаточно, второй уведомлён.
- **Уровень:** `end-to-end`.

#### AC-079

- **Дано:** A и B имеют разные аккаунты одного провайдера и общий счёт; B заносит покупку A со счёта B.
- **Когда:** Выполняются ввод, импорт обоих аккаунтов и повторное подключение того же внешнего аккаунта.
- **Тогда:** Разные аккаунты не сливаются; повторный источник не удваивает остатки. Плательщик, автор и получатель расхода сохраняются независимо. Неустановленное совпадение блокирует новый учёт до уточнения.
- **Уровень:** `integration`.

#### AC-080

- **Дано:** Зарплата поступила на счёт A, общая аренда оплачена B, у A нет доступного остатка.
- **Когда:** Строятся семейный бюджет, персональные расходы и обеспеченность по валютам.
- **Тогда:** Доход общий с сохранением получателя; аренда учтена в семье один раз и в личных видах по долям. Доступность семьи включает средства обоих без автоматического обмена валют и без кредитного лимита.
- **Уровень:** `integration`.

#### AC-081

- **Дано:** Чек RUB 1000 содержит общие продукты 600 и личные покупки A 100 и B 300.
- **Когда:** Чек заносит любой участник; AI применяет правила или уточняет неизвестное назначение.
- **Тогда:** Факт семьи 1000, A 400, B 600; доли суммируются точно. Исключение покупки приоритетнее статьи, затем 50/50; неоднозначная трата сохранена без вымышленной принадлежности.
- **Уровень:** `integration`.

#### AC-082

- **Дано:** A оплачивает общий расход RUB 1000 и явно отмечает возмещение RUB 300 от B.
- **Когда:** B переводит 100, затем 200; обе стороны переводов импортируются повторно.
- **Тогда:** Долг уменьшается 300→200→0 один раз; основная сумма переводов не доход/расход. Обычная покупка без указания долг не создаёт; валютное погашение требует явного соответствия сумм.
- **Уровень:** `integration`.

#### AC-083

- **Дано:** Есть личные цели A и B, общая цель RUB 600000 и виртуальный резерв RUB 10000.
- **Когда:** Оба открывают личные и семейный виды, увеличивают разрешённый резерв и связывают выделенный счёт.
- **Тогда:** Общая цель показана целиком в общем блоке; персональные половины не создаются. Резерв уменьшает семейную доступность один раз; нет автоматического распределения свободных средств на цели.
- **Уровень:** `end-to-end`.

#### AC-084

- **Дано:** Осталось 10 дней, K=RUB 1000, положительные персональные остатки A=3000 и B=1000.
- **Когда:** Рассчитаны доступные лимиты; затем меняется плательщик общей покупки или дата ожидаемого дохода.
- **Тогда:** Семейный лимит 100/день, A 75, B 25; суммы не дублируют K. Плательщик не меняет доли; перенос дохода меняет прогноз, не доступный остаток. Неизвестное назначение не скрывает факт расхода.
- **Уровень:** `integration`.

#### AC-085

- **Дано:** Оба видят общий чат; A имеет личную цель, B просит AI изменить её от имени A.
- **Когда:** Модель предлагает действие, а затем A подтверждает новую адресованную ему версию предложения.
- **Тогда:** Сообщение B не выдаёт полномочия A; до разрешённого подтверждения изменения нет. Аудит хранит автора сообщения, подтвердившего и AI-основание; секретов в чате нет.
- **Уровень:** `integration`.

#### AC-086

- **Дано:** A и B открыли одну версию операции или уточнения.
- **Когда:** Оба отправляют несовместимые изменения и повторяют один запрос.
- **Тогда:** Один результат применяется; второй получает конфликт с необходимостью перечитать состояние. Повтор не дублирует эффект; отмена создаёт новую проверенную revision и не стирает чужую последующую правку.
- **Уровень:** `integration`.

#### AC-087

- **Дано:** A владеет внешним аккаунтом, B инициирует повторную авторизацию или отключение.
- **Когда:** Запрашивается MFA; одновременно завершает работу старое задание синхронизации.
- **Тогда:** MFA адресован A; B видит статус, но не пароль/код/сессию. Отключение отзывает lease/version и запрещает применение старого результата; реальные платежи недоступны обоим.
- **Уровень:** `integration`.

#### AC-088

- **Дано:** Оба имеют уведомления и устройства, A изменяет разрешённую цель.
- **Когда:** B читает уведомление; A восстанавливает вход и отзывает своё устройство.
- **Тогда:** Событие доставлено B один раз; read-state и отзыв устройства одного не меняют подписки другого. Уточнения личного плана направлены его владельцу, остальные доступны обоим.
- **Уровень:** `end-to-end`.

#### AC-089

- **Дано:** Копия содержит двух участников, личные/общие цели и операции с разными авторами.
- **Когда:** Полный набор восстанавливается на изолированном сервере.
- **Тогда:** Суммы, связи и права обоих сохранены; входы не объединены, банковские сессии автоматически не оживают. Восстановление укладывается в измеренную цель RTO.
- **Уровень:** `integration+manual`.

#### AC-090

- **Дано:** В тестах созданы две изолированные семьи; запрос или задача подменяет householdId/actor/resourceId.
- **Когда:** Проверяются чтение файла, импорт, исправление, поиск AI и дедупликация.
- **Тогда:** Чужие объекты недоступны и не объединяются; сервер берёт principal из сессии или проверенного контекста задания. Отказ не раскрывает чужое содержимое.
- **Уровень:** `integration`.

### Проверка результата

```sh
python3 spec/001-want-keep-mvp/tools/spec_tool.py check
```

Полное покрытие REQ/AC/task; нет скрытого выбора API/прав/формул; независимая проверка плана Ready или явно сохранён Not Ready.

Команды make созданы основой task-1.1; финансовые provider/integration/E2E suites ещё не реализованы. Для документации используется make docs-check. Live/paid/manual проверки отдельно фиксируют доступ и фактический результат; наличие команды или UI-доступа не доказывает runtime.

### Передача следующему агенту

Записать изменённые контракты, команды и результаты, ограничения, незакрытые вопросы и разблокированные зависимости. Обновить обе языковые версии и трассировку. Закрывать задачу только по доказательству её результата; GitHub Closed само по себе не означает Ready MVP.

**Commit boundary:** логическая граница этой задачи; commit/push/deploy не разрешены данной карточкой и требуют действующей авторизации пользователя.

## EN

Obtain a Ready specification before application implementation.

**Status:** Blocked by dependencies and the SDD Ready gate; implementation has not started.

**Dependencies:** `task-0.1`, `task-0.2`, `task-0.3`, `task-0.4`, `task-0.5`, `task-0.6`, `task-0.7`, `task-0.8`, `task-0.9`.

**Kind:** `specification`.

### Change and contracts

Reconcile research outcomes and resolve contracts, fields, sources and fundamental questions. Update RU/EN and affected tasks. With blockers retain Not Ready and do not create plan.md. After Ready, create a decision-complete SDD plan with exact checks and dispatch an independent read-only reviewer; only fresh Ready permits implementation. Resolve command-status/idempotency retention for retry/recovery, distinct from durable financial audit. Apply Aifory D-33: close AIFORY-B02–B04 only for RUB, USDT, ETH and the existing card; AIFORY-B05/other products do not block. Establish the structured read contract, automation permission, identity/history and card lifecycle; include ETH valuation. For Bybit D-36 use successful RSA readOnly evidence/bybit-api: BYBIT-B02/B05 are closed for this owner, including P2P. Close BYBIT-B03/B04: deterministic identity/cross-log linkage and ambiguity, precision-aware reconciliation, history bounds, missing hourly ID/revisions and Earn forecast basis/USDT lifetime totalPnl difference. task-0.9 checks VPS; task-4.4 checks a second owner, revocation and executable conformance. Funding USDT/USDC/ETH/BTC, Flexible Easy Earn and P2P remain required; Fixed/other unused BYBIT-B06 products do not block. USDC remains a separate valuation asset. For Raiffeisen under D-35 close RAIF-B02/B03/B04/B06: CAMT mapping and revision identity, history depth/completeness, current/available/locked balances and fees, auth lifecycle and independent accounts. Use evidence/raiffeisen; RAIF-B01/B05 are closed only within the stated scope. Closing task-0.2 does not unblock task-4.2 without Ready.

### Change boundaries

- `spec/001-want-keep-mvp/constraints.md`
- `spec/001-want-keep-mvp/contracts.md`
- `spec/001-want-keep-mvp/verification.md`
- `spec/001-want-keep-mvp/plan.md`

These are planned paths. Shared contracts: `spec/001-want-keep-mvp/contracts.en.md`; architecture and commands: `constraints.en.md`. Change only the behavior owner and affected tests; an unresolved contract requires updated evidence and stops dependent implementation.

### Linked requirements

- **REQ-020:** AI income/expense insights reference verifiable data and separate forecasts from facts.
- **REQ-031:** Credit cards show debt, own funds, credit limit, minimum payment and due date from source data.
- **REQ-032:** Grace-period tracking uses the specific card's terms and shows the amount and deadline needed to preserve the benefit.
- **REQ-033:** Savings show actual accruals and forecasts using rates, terms, compounding and cash flows.
- **REQ-034:** Investment returns are compared using dated cash flows and valuation currency.
- **REQ-038:** Exchange quotes include direction, provider, timestamp, applicable amount and known fees.
- **REQ-039:** Missing rates and unsupported assets never become zero amounts or assumed USD/USDT/USDC parity.
- **REQ-042:** The Alfa-Bank integration automatically reads debit/credit cards, current/savings accounts and deposits under a verified contract.
- **REQ-043:** Raiffeisen RBO API reads only the entrepreneur current account: balances, receipts, debits, fees and history (D-35).
- **REQ-044:** The Ozon Bank integration automatically reads the debit card and linked main account: balances, transactions and available details under a verified contract. Other Ozon products are deferred until a contract extension.
- **REQ-045:** Bybit automatically reads Funding USDT/USDC/ETH/BTC, used Easy Earn and P2P; the official API is preferred. Other products are deferred without blocking under D-36.
- **REQ-046:** Aifory Pro automatically reads RUB accounts, USDT, ETH and the existing crypto card, including these products’ movements and fees. Other products are deferred and do not block the MVP.
- **REQ-047:** The EMCD integration automatically reads used crypto cards, Coinhold/Grow, the USDT wallet and historical P2P orders under D-34; mining has never been used and is deferred with other unused products without blocking readiness.
- **REQ-051:** AI is limited to $50/month and degrades to a waiting queue without stopping ordinary accounting.
- **REQ-055:** The web app targets macOS laptops in Chrome and Arc; window resizing and zoom preserve daily accounting access.
- **REQ-056:** Deployment fits $40/month for a server in DE/NL/BG; no separately paid data sources are used.
- **REQ-062:** Architecture uses Go/PostgreSQL, React/TypeScript/Vite and a separate Playwright collector with dependencies pointing toward the domain.
- **REQ-063:** User, household and membership are separate models; the two-member limit is configured.
- **REQ-064:** Both members see all financial data and edit transactions; only the owner edits personal goals and plan portions.
- **REQ-065:** Account ownership, external-account owner, record author and expense attribution are distinct dimensions.
- **REQ-066:** All income and available funds enter the household pool; household and individual budget views share one financial fact.
- **REQ-067:** Expenses and receipt items have personal or joint attribution; joint shares default to 50/50 with line or purchase overrides.
- **REQ-068:** An inter-member debt is recorded only explicitly and does not increase household assets or expenses.
- **REQ-069:** Personal and joint goal reservations are explicit; joint goals appear in a separate shared block without personal shares.
- **REQ-070:** Individual daily allowances sum to no more than the household ceiling in one currency; the payer’s account does not change expense shares.
- **REQ-071:** One shared chat retains message authors and checks the AI command initiator’s authority at execution.
- **REQ-072:** Competing edits, clarification answers and reversals check revision and current permissions while retaining both authors.
- **REQ-073:** Both manage connections; the external-account owner performs bank authentication without exposing secrets to the partner or AI.
- **REQ-074:** Plan and goal changes notify the other member; read state and push subscriptions belong to the individual user.
- **REQ-075:** Data recovery preserves users, memberships, ownership, roles, history and shared household accounting.
- **REQ-076:** Household scope is checked for APIs, files, AI, jobs and external IDs independently of supplied actor/owner fields.

### Acceptance criteria

A criterion link establishes coverage; research or a partial task does not prove the entire product criterion. This task's exact outcome is specified in verification below.

#### AC-032

- **Given:** Card terms, statement, exclusions and deadline are confirmed.
- **When:** A purchase, partial repayment and grace-excluded transaction occur.
- **Then:** Amount and deadline follow confirmed terms; missing terms produce an unknown state rather than a promise of grace eligibility.
- **Level:** `contract`.

#### AC-033

- **Given:** A deposit or Earn product has confirmed terms, a top-up and a withdrawal.
- **When:** Period income and forecast are calculated.
- **Then:** Actual income is separate from forecast and revaluation; rate changes and compounding follow the terms; unknown terms prevent an exact forecast.
- **Level:** `integration`.

#### AC-034

- **Given:** Two investments have different top-up dates and the same ending balance.
- **When:** A return comparison is built.
- **Then:** Actual income and annualized money-weighted return show dates/method; invalid or ambiguous calculations are unavailable rather than zero.
- **Level:** `unit`.

#### AC-038

- **Given:** Two providers have different bid/ask quotes and one omits its fee.
- **When:** The owner compares RUB → USDT.
- **Then:** Directions and freshness are comparable; an unknown fee is not treated as zero; an unavailable quote is not replaced by a promise of market execution.
- **Level:** `integration`.

#### AC-042

- **Given:** An authorized personal Alfa-Bank account with the tested products is connected.
- **When:** Accounts, balances, transactions and required product terms are requested.
- **Then:** Every mandatory product has source-matching data and read evidence; inaccessible products are blockers, not successful coverage.
- **Level:** `contract+manual`.

#### AC-043

- **Given:** An authorized entrepreneur current account is connected through RBO API.
- **When:** Request balances, movements, repeated import and intraday no-statements.
- **Then:** Data matches the source; no duplicates, fees recorded separately. Unknown current balance is not zero: show the last verified balance, its date and the coverage gap.
- **Level:** `contract+manual`.

#### AC-044

- **Given:** An authorized personal Ozon Bank account with a debit card and linked main account is connected.
- **When:** Debit-product balances, transactions and available details are requested; the same card and account appear in several views.
- **Then:** Data matches the source, read evidence is retained and the card does not duplicate its account balance. Unavailable mandatory debit-product fields are explicit. Missing Ozon credit cards, savings or deposits do not block the MVP: these products are outside the current contract and are not presented as implemented.
- **Level:** `contract+manual`.

#### AC-045

- **Given:** A read-only Bybit account is securely connected with verified Funding, used Easy Earn and P2P contracts.
- **When:** Balances, history from the selected date, replayed pages, Convert, Earn accrual/payout and P2P are read; a selected-product permission failure is simulated.
- **Then:** Exact native amounts, IDs, statuses, fees and coverage match the source; Funding and details do not duplicate an exchange, fee or income. P2P links crypto/fiat with the bank or requires clarification. Access failure/incomplete history explicitly blocks the corresponding coverage; absent Spot/UTA trading, futures, card, On-Chain/Advanced Earn and other unused products do not block.
- **Level:** `contract+manual`.

#### AC-046

- **Given:** An authorized personal Aifory account provides RUB accounts, USDT/TRC-20, ETH/Ethereum and the existing USD card (D-33); other products are not connected.
- **When:** Balances/history are read again, including RUB/USDT exchange, ETH withdrawal with a fee, USDT/USD card funding and similar pending/confirmed payments with a separate fee.
- **Then:** Each included product has structured read evidence; RUB groups do not duplicate child accounts and ETH remains exact. Movements/fees count once using verified IDs/links/statuses; neither UI signs nor USDT/USD parity are assumed. Unknown linkage requires clarification. Gaps in included products block the adapter; other products are not required, but their movements through selected wallets are retained.
- **Level:** `contract+manual`.

#### AC-047

- **Given:** An authorized personal EMCD account has a USDT wallet, existing Grow deposits, existing cards including a blocked card, and P2P history. Mining has never been used.
- **When:** Accounts, balances, transactions and required product terms are requested.
- **Then:** Each included product has source-matching data and verified automatic reading: aggregates do not duplicate child balances; accrual/capitalization/payout do not triple income; declined card principal is not an expense; P2P links to monetary legs without duplicates. Unknown fields/history are explicit; missing selected-product contracts block the adapter. Mining and other unused products are not required.
- **Level:** `contract+manual`.

#### AC-056

- **Given:** A concrete plan, region, billing currency and taxes are selected.
- **When:** Operating costs are checked for hundreds of monthly transactions.
- **Then:** A dated all-in server estimate is at most $40; OpenAI has a separate $50 cap; a mandatory paid data source remains a blocker.
- **Level:** `manual`.

#### AC-062

- **Given:** Application structure and component contracts exist.
- **When:** Dependencies and public interfaces are checked.
- **Then:** Domain imports no HTTP, SQL, UI, OpenAI SDK or browser types; adapters map external models; the collector owns no financial decisions.
- **Level:** `static`.

#### AC-070

- **Given:** A bank exposes balance but no grace terms; an Earn rate has an unknown accrual basis.
- **When:** Forecasts are opened.
- **Then:** Balance is shown; grace eligibility and exact forecasts explain unavailability; AI does not turn marketing wording into guaranteed business rules.
- **Level:** `contract+end-to-end`.

#### AC-076

- **Given:** A synthetic hundreds-of-transactions set includes difficult transfers, receipts and reference answers.
- **When:** AI evaluation, load checks and a manual daily flow run.
- **Then:** The report shows errors, clarifications, latency, tokens/cost and user time; financial invariants pass and cost uses measurements; untested quality is not claimed as proven.
- **Level:** `manual+integration`.

#### AC-077

- **Given:** Two users belong to one household.
- **When:** Schema, authorization and the membership limit are inspected.
- **Then:** There are no partner1/partner2 fields or specific-user branches; roles and ownership are separate from identity. Exit, replacement and new roles are not implemented.
- **Level:** `integration`.

#### AC-078

- **Given:** A has a personal goal and plan line; the household has a joint line and both members’ transactions.
- **When:** B reads all data, edits A’s transaction and the joint plan, then attempts to change A’s personal goal/plan through API and AI.
- **Then:** Reads, transaction edits and joint changes succeed; A’s personal plan/goal are protected server-side. One authorized confirmation suffices and the other member is notified.
- **Level:** `end-to-end`.

#### AC-079

- **Given:** A and B have separate accounts at one provider and a joint account; B enters A’s purchase paid from B’s account.
- **When:** Entry, import of both accounts and reconnection of the same external account run.
- **Then:** Distinct accounts are not merged; a repeated source does not double balances. Payer, author and expense beneficiary remain independent. Unresolved source identity blocks new posting pending clarification.
- **Level:** `integration`.

#### AC-080

- **Given:** Salary arrived in A’s account, B paid joint rent and A has no available balance.
- **When:** The household budget, individual expenses and currency funding are calculated.
- **Then:** Income is pooled with recipient retained; rent appears once for the household and by shares in individual views. Household availability includes both members’ funds without automatic currency exchange or credit limits.
- **Level:** `integration`.

#### AC-081

- **Given:** A RUB 1,000 receipt contains joint groceries of 600 and personal purchases of A 100 and B 300.
- **When:** Either member enters the receipt; AI applies rules or clarifies unknown attribution.
- **Then:** Household actual is 1,000, A 400, B 600; shares sum exactly. Purchase override takes precedence over plan line, then 50/50; ambiguous spending persists without invented attribution.
- **Level:** `integration`.

#### AC-082

- **Given:** A pays a RUB 1,000 joint expense and explicitly records RUB 300 reimbursement due from B.
- **When:** B transfers 100 and then 200; both legs of the transfers are imported again.
- **Then:** Debt falls 300→200→0 once; transfer principal is not income/expense. An ordinary purchase creates no debt without instruction; cross-currency settlement requires explicit amount mapping.
- **Level:** `integration`.

#### AC-083

- **Given:** There are personal goals of A and B, a joint RUB 600,000 goal and a RUB 10,000 virtual reserve.
- **When:** Both open individual and household views, increase an authorized reserve and link a dedicated account.
- **Then:** The joint goal appears whole in the shared block; personal halves are not created. The reserve reduces household availability once; free funds are not automatically allocated to goals.
- **Level:** `end-to-end`.

#### AC-084

- **Given:** 10 days remain, K=RUB 1,000 and positive individual remainders are A=3,000 and B=1,000.
- **When:** Available allowances are calculated; then a joint purchase payer or expected-income date changes.
- **Then:** Household allowance is 100/day, A 75, B 25; totals do not duplicate K. Payer does not change shares; rescheduling income changes forecast, not available funds. Unknown attribution does not hide actual expense.
- **Level:** `integration`.

#### AC-085

- **Given:** Both see the shared chat; A has a personal goal and B asks AI to change it as A.
- **When:** The model proposes an action and A later confirms a new proposal version addressed to A.
- **Then:** B’s message grants no authority of A; nothing changes before authorized confirmation. Audit records message author, approver and AI rationale; chat contains no secrets.
- **Level:** `integration`.

#### AC-086

- **Given:** A and B opened the same transaction or clarification revision.
- **When:** Both submit conflicting edits and replay one request.
- **Then:** One result applies; the other receives a conflict requiring refresh. Replay does not duplicate effects; reversal creates a checked new revision without erasing the other member’s later edit.
- **Level:** `integration`.

#### AC-087

- **Given:** A owns the external account and B initiates reauthorization or disconnect.
- **When:** MFA is requested while an old sync job completes.
- **Then:** MFA is addressed to A; B sees status but no password/code/session. Disconnect revokes lease/version and prevents stale-result application; actual payments are unavailable to both.
- **Level:** `integration`.

#### AC-088

- **Given:** Both have notifications and devices and A changes an authorized goal.
- **When:** B reads the notification; A recovers sign-in and revokes their device.
- **Then:** B receives the event once; one member’s read state and device revocation do not alter the other’s subscriptions. Personal-plan clarifications target its owner; other clarifications are open to both.
- **Level:** `end-to-end`.

#### AC-089

- **Given:** A backup contains two members, personal/joint goals and transactions by different authors.
- **When:** The complete set is restored onto an isolated server.
- **Then:** Amounts, relationships and both users’ permissions are preserved; sign-ins are not merged and bank sessions do not revive automatically. Recovery meets the measured RTO target.
- **Level:** `integration+manual`.

#### AC-090

- **Given:** Tests contain two isolated households; a request or job forges householdId/actor/resourceId.
- **When:** File reads, import, correction, AI retrieval and deduplication are exercised.
- **Then:** Foreign objects are inaccessible and never merged; the server takes principal from the session or validated job context. Denial reveals no foreign content.
- **Level:** `integration`.

### Verification

```sh
python3 spec/001-want-keep-mvp/tools/spec_tool.py check
```

Complete REQ/AC/task coverage; no hidden API/permission/formula decisions; independent plan review is Ready or Not Ready is explicitly retained.

The task-1.1 foundation provides make commands; financial provider/integration/E2E suites are not implemented yet. Use make docs-check for documentation. Live/paid/manual checks separately record access and outcomes; an existing command or UI access is not runtime proof.

### Handoff to the next agent

Record changed contracts, commands/results, limitations, unresolved questions and unblocked dependencies. Update both languages and traceability. Close the task only with evidence of its outcome; GitHub Closed alone does not mean the MVP is Ready.

**Commit boundary:** this task's logical boundary; this card does not authorize commit/push/deploy, which require current user authorization.
