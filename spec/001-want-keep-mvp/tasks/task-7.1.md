<!-- want-keep-task: task-7.1 -->
# task-7.1 — Создать desktop-оболочку и вход RU/EN / Create the desktop shell and RU/EN sign-in

## RU

Предоставить вход, навигацию и доступные состояния на ноутбуке macOS Chrome/Arc.

**Состояние:** Не начато; задача ожидает собственные зависимости и entry gates.

**Зависимости:** `task-1.4`, `task-1.2`, `task-7.11`, `task-1.6`.

**Тип:** `implementation`.

### Изменение и контракты

Создать React/TypeScript/Vite shell для macOS Chrome/Arc, маршруты SCR-001–SCR-005, общую навигацию, passkey/recovery и typed errors. Использовать дизайн-систему task-7.11; вход следует референсу: компактный логотип, название, свободное пространство и одна основная кнопка. Сессия только в защищённой cookie. Desktop 1280×720/1440×900, zoom 200%, клавиатура/фокус; при offline нет ложного сохранения. Мобильные экраны и установка приложения исключены. Bootstrap/invite используют policy task-1.6.

### Границы изменений

- `web/src/app/`
- `web/src/features/identity/`
- `web/src/locales/`

### Экранный контракт

### SCR-001 — Вход

`/login`

**Вопрос:** Как войти?

**Главный ответ:** Один вход с личным passkey.

**Структура сверху вниз:** Центр: логотип, Want Keep, свободное пространство, основная кнопка; язык/помощь ненавязчивы.

**Следующее действие:** Войти с passkey → SCR-006 или незавершённый SCR-005; помощь → SCR-002.

**Объяснение и детализация:** Системный prompt, локальная причина ошибки и повтор; не копировать размеры экспорта.

**Права:** До входа только собственная авторизация; финансовые данные скрыты.

Forms: FORM-01.

States: UISTATE-01, UISTATE-07, UISTATE-08, UISTATE-09, UISTATE-10, UISTATE-16, UISTATE-17.

### SCR-002 — Восстановление

`/recovery`

**Вопрос:** Как вернуть свой доступ?

**Главный ответ:** Личный одноразовый код и новый passkey.

**Структура сверху вниз:** Объяснение → код → проверка → новый passkey → новые коды/результат.

**Следующее действие:** Восстановить свой вход, затем SCR-006; отмена → SCR-001.

**Объяснение и детализация:** Старые сессии этого пользователя отозваны; invalid/used/expired код без раскрытия чужой identity.

**Права:** До входа только собственная авторизация; финансовые данные скрыты.

Forms: FORM-01.

States: UISTATE-01, UISTATE-07, UISTATE-08, UISTATE-09, UISTATE-10, UISTATE-16, UISTATE-17.

### SCR-003 — Первичная настройка

`/setup`

**Вопрос:** Как начать нашу семью?

**Главный ответ:** Закрытая настройка первого участника.

**Структура сверху вниз:** Проверка bootstrap → имя/семья → passkey/recovery → приглашение.

**Следующее действие:** Создать семью → SCR-004 или SCR-005.

**Объяснение и детализация:** Повторный bootstrap закрыт, никакой публичной регистрации.

**Права:** До входа только собственная авторизация; финансовые данные скрыты.

Forms: FORM-02.

States: UISTATE-01, UISTATE-07, UISTATE-08, UISTATE-09, UISTATE-10, UISTATE-12, UISTATE-16, UISTATE-17.

### SCR-004 — Приглашение

`/invite`

**Вопрос:** Как присоединиться партнёру?

**Главный ответ:** Проверенное закрытое приглашение в конкретную семью.

**Структура сверху вниз:** Семья/пригласивший после проверки → имя → свой passkey → свои коды.

**Следующее действие:** Принять → SCR-005; истёкшее приглашение ведёт к запросу нового у участника.

**Объяснение и детализация:** Использованное приглашение и полный состав имеют разные понятные статусы.

**Права:** До входа только собственная авторизация; финансовые данные скрыты.

Forms: FORM-02.

States: UISTATE-01, UISTATE-07, UISTATE-08, UISTATE-09, UISTATE-10, UISTATE-12, UISTATE-16, UISTATE-17.

### SCR-005 — Начало учёта

`/onboarding`

**Вопрос:** Как получить первую полезную сводку?

**Главный ответ:** Можно начать с наличных и уже доступных счетов.

**Структура сверху вниз:** Прогресс → добавить счёт/подключение → дата истории/остатки → первый план → обзор.

**Следующее действие:** Добавить FORM-03/13 или продолжить с доступным → SCR-006.

**Объяснение и детализация:** Неизвестная история видна как ограничение; незавершённое подключение не блокирует доступные функции.

**Права:** Оба участника видят; действия проверяет сервер по членству и владельцу ресурса.

Forms: FORM-03, FORM-13.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-14, UISTATE-15.

#### FORM-01 — Passkey и личное восстановление

**Поля:** Системный запрос passkey; recovery: личный код, затем создание нового passkey.

**Проверки и права:** Проверенные challenge/origin/RP; код одноразовый и скрытый. Чужой партнёр не может восстановить вход. Нет email/password fallback.

**Результат:** Сессия текущего участника; recovery отзывает только его старые сессии; отмена возвращает к входу.

#### FORM-02 — Начало семьи и приглашение

**Поля:** Имя участника/семьи, язык, таймзона/валюта; закрытое приглашение, имя второго участника и passkey.

**Проверки и права:** Однократный операторский bootstrap; invitation ограничен семьёй, сроком и одноразовым использованием; конфигурация максимум 2 активных участника.

**Результат:** Создано членство, показаны личные recovery-коды; далее onboarding. Секреты не в URL журналов/аналитики.

#### FORM-03 — Счёт и начальный остаток

**Поля:** Название, тип продукта, валюта, личный владелец/семейный, дата начала, начальные собственные/заёмные/заблокированные суммы по типу.

**Проверки и права:** Оба member создают счета и исправляют факты учёта. При смене владельца или personal/household принадлежности существующего личного счёта требуется его текущий владелец; для семейного счёта — любой member. Проверенный внешний владелец и история операций этим не меняются. Точные decimal, валюта обязательна. Импортируемые поля меняются через correction; начальный остаток не доход.

**Результат:** Счёт в учёте создан/исправлен с audit; это не открытие банковского продукта.

#### FORM-13 — Подключение и reauth

**Поля:** Платформа, владелец внешнего аккаунта, дата истории, доступные продукты; секрет только в изолированном авторизационном потоке.

**Проверки и права:** Управляют оба, ввод ключа/пароля/MFA только внешним владельцем. Read-only scopes, identity и coverage подтверждены адаптером, без обхода MFA/CAPTCHA.

**Результат:** Подключено/синхронизация/ожидается владелец/ошибка; отключение сохраняет историю и инвалидирует generation.

- **UISTATE-01 — Загрузка:** Скелетон структуры и подпись загрузки; суммы не подменяются нулями.
- **UISTATE-02 — Обновление:** Сохранить предыдущие данные и контекст, показать время последнего успеха; блокировать только конфликтующие действия.
- **UISTATE-03 — Пусто:** Объяснить полезный результат и предложить первое действие: счёт, чек, план или цель.
- **UISTATE-05 — Частичные данные:** Назвать отсутствующий источник/период и последствия для суммы; доступные блоки работают; неизвестное обозначить отдельно.
- **UISTATE-06 — Устаревшие данные:** Показать дату последнего успеха и влияние на решение; дать обновить или перейти к подключению.
- **UISTATE-07 — Ошибка:** Понятная причина и следующий шаг у проблемного блока; ввод и исправные данные сохранить, диагностику раскрывать отдельно.
- **UISTATE-08 — Offline:** Показать отсутствие связи; не обещать сохранение. Чувствительные черновики только в памяти текущей вкладки, без новой offline-очереди.
- **UISTATE-09 — Сохранение:** Немедленно показать прогресс текущего действия и не допускать дублирующую отправку команды.
- **UISTATE-10 — Исход неизвестен:** Сохранить ID команды/ввод, запросить её результат; не создавать новую финансовую команду вслепую. После перезагрузки сверять серверный список недавних команд.
- **UISTATE-11 — Конфликт версии:** Показать авторов и различия, сохранить мой ввод; загрузить актуальную версию и дать повторно применить выбранные изменения после проверки.
- **UISTATE-12 — Недостаточно прав:** Финансовые данные доступны семье; запрещённое изменение объясняет владельца. Сервер отклоняет команду независимо от видимости кнопки.
- **UISTATE-13 — Сессия истекла:** Закрыть защищённое содержимое; вход для того же участника, безопасный возврат по внутреннему маршруту. Чужой вход не получает прежний черновик.
- **UISTATE-14 — Ожидание AI:** Отличать очередь, обработку, уточнение и паузу из-за лимита/API; обычный учёт доступен, результат не выдумывать.
- **UISTATE-15 — Нужен банковский вход:** Назвать подключение и владельца, дать ему безопасно войти; партнёру показать ожидание без доступа к секрету.
- **UISTATE-16 — Подтверждено:** После подтверждённого сервером результата показать что изменилось, ссылку на объект и доступное исправление; не полагаться на исчезающий toast.
- **UISTATE-17 — Отмена:** Объяснить отсутствие нового подтверждённого результата, дать повторить явно; не выдавать отмену системного passkey за поломку.


Это планируемые пути. Общие контракты: `spec/001-want-keep-mvp/contracts.md`; архитектура и команды: `constraints.md`. Менять только владельца поведения и затронутые тесты; при незакрытом контракте обновить evidence и остановить зависимую реализацию.

### Связанные требования

- **REQ-001:** Семейный пилот обслуживает двух участников с раздельным входом и закрытым присоединением; публичной регистрации нет.
- **REQ-004:** Начало учёта задаётся датой; начальные остатки отделены от доходов и расходов.
- **REQ-040:** Каждый источник обновляется раз в час и по запросу с видимым временем успешного обновления.
- **REQ-041:** История сохраняет границы покрытия, курсоры, пробелы и статусы источника.
- **REQ-049:** Каждый участник входит со своими passkey и одноразовыми кодами восстановления; сброс чужого входа партнёром недоступен.
- **REQ-050:** Файлы, ключи источников, сессии и финансовые журналы защищены от постороннего доступа.
- **REQ-053:** Напоминания и сводки доступны внутри приложения и через разрешённый web-push.
- **REQ-054:** Интерфейс, чат и документация поддерживают RU/EN без изменения финансовой семантики.
- **REQ-055:** Веб-приложение предназначено для ноутбука macOS в Chrome и Arc; изменение окна и масштаба сохраняет доступность ежедневного учёта.
- **REQ-063:** Пользователь, семья и членство моделируются отдельно; ограничение двух участников задаётся конфигурацией.
- **REQ-064:** Оба участника видят все финансовые данные и изменяют операции; личные цели и части плана изменяет только их владелец.
- **REQ-076:** Семейная область проверяется для API, файлов, AI, фоновых задач и внешних ID независимо от присланных actor/owner.
- **REQ-081:** Навигация desktop сохраняет контекст и не меняет права при смене представления семьи.
- **REQ-082:** Экранные состояния объясняют последствия и безопасный следующий шаг без потери ввода.
- **REQ-083:** Экран входа сохраняет композицию присланного референса и личное восстановление доступа.
- **REQ-084:** Доступность проверяется на реальных Chrome и Arc, включая клавиатуру, фокус, контраст, масштаб и reduced motion.

### Критерии приёмки

Связь с критерием задаёт покрытие; исследование или частичная задача не доказывает весь критерий продукта. Точный результат этой задачи указан ниже в проверке.

#### AC-001

- **Дано:** Создана семья, первый участник вошёл, лимит активных участников равен 2.
- **Когда:** Второй участник принимает приглашение; посторонний пробует открытый вход, повтор приглашения и присоединение сверх лимита.
- **Тогда:** Приглашение создаёт отдельное членство один раз; посторонний не получает данных, повтор и превышение лимита отклонены.
- **Уровень:** `end-to-end`.

#### AC-049

- **Дано:** Оба участника зарегистрировали собственные passkey и личные коды восстановления.
- **Когда:** Участник восстанавливает свой вход, повторяет код, пробует чужой origin и сброс входа партнёра.
- **Тогда:** Свой вход восстановлен с отзывом своих старых сессий/подписок; сессии партнёра сохранены; повтор кода, чужой origin и сброс чужого входа отклонены.
- **Уровень:** `end-to-end`.

#### AC-050

- **Дано:** Существует приватный чек и активное подключение источника.
- **Когда:** Проверяются прямой URL файла, экспорт без сессии, логи и отзыв подключения.
- **Тогда:** Без авторизации доступ закрыт; секреты зашифрованы и не журналируются; отзыв подключения прекращает дальнейший сбор.
- **Уровень:** `integration`.

#### AC-054

- **Дано:** Есть русская и английская версии одной операции, бюджета и ошибки.
- **Когда:** Переключается язык.
- **Тогда:** Суммы, даты, валюты и смысл совпадают; форматирование локализовано, идентификаторы и категории пользователя не переводятся с потерей данных.
- **Уровень:** `end-to-end+static`.

#### AC-055

- **Дано:** Владелец проверяет день с расходами, чеком, уточнением, бюджетом и целью.
- **Когда:** Участник проходит сценарий в реальных Chrome и Arc при 1280×720 и 1440×900 CSS px, затем увеличивает масштаб до 200%.
- **Тогда:** Основные действия доступны без потери данных и горизонтального прокручивания форм; измерено фактическое время сценария относительно личного ориентира до 45 минут в день.
- **Уровень:** `manual`.

#### AC-072

- **Дано:** Активны две сессии и push-подписка.
- **Когда:** Владелец восстанавливает доступ и отзывает старое устройство.
- **Тогда:** Старые сессии/привязанные подписки отозваны; ссылка из push требует действующей авторизации; финансовых деталей в push по умолчанию нет.
- **Уровень:** `end-to-end+manual`.

#### AC-075

- **Дано:** Один сценарий ввода чека и исправления категории доступен на двух языках.
- **Когда:** Сценарий выполняется с клавиатурой в Chrome и Arc на macOS в обоих контрольных размерах и при увеличении масштаба.
- **Тогда:** Все обязательные поля и ошибки доступны; переключение языка не сбрасывает ввод; суммы локализуются только при отображении.
- **Уровень:** `end-to-end+manual`.

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

#### AC-098

- **Дано:** Пользователь отфильтровал месяц, валюту, участника и список.
- **Когда:** Он открывает детализацию, возвращается, меняет язык и открывает прямую ссылку.
- **Тогда:** Обзор открывается после входа; левое меню содержит Обзор, Деньги, План, Аналитика, Чат, внизу Подключения/Настройки, сверху уведомления. Контекст сохраняется; сервер проверяет текущего автора независимо от фильтра.
- **Уровень:** `manual+e2e`.

#### AC-099

- **Дано:** Есть загрузка, пустой список/поиск, устаревшие/частичные данные, offline, отказ и конкурирующие правки.
- **Когда:** Пользователь выполняет чтение или сохранение.
- **Тогда:** Неизвестное не становится нулём, подтверждение даётся после readback; неизвестный исход проверяется по ID команды до повторного создания. Конфликт сохраняет ввод и предлагает сравнение. Истечение сессии ведёт к входу, банковская reauth — к нужному владельцу, ожидание AI не блокирует обычный учёт.
- **Уровень:** `manual+e2e`.

#### AC-100

- **Дано:** Участник открывает /login в обоих языках.
- **Когда:** Вход ожидает passkey, отменён или завершился ошибкой.
- **Тогда:** Компактные логотип/Want Keep по центру, свободное пространство, одна основная кнопка «Войти с passkey» / «Log in with Passkeys», ненавязчивые язык и помощь. Статус не ломает композицию; помощь ведёт к личному восстановлению. SVG не искажается.
- **Уровень:** `manual+e2e`.

#### AC-101

- **Дано:** Экраны и формы доступны в RU/EN на macOS.
- **Когда:** Проверяются 1280×720 и 1440×900 CSS px, масштаб 100%/200%, клавиатура и уменьшение движения.
- **Тогда:** Нет скрытых действий, обрезанных сумм и горизонтальной прокрутки форм; таблицы при необходимости имеют обозначенную область прокрутки. Контраст обычного текста ≥4.5:1, крупного ≥3:1, значимых границ/фокуса ≥3:1. Фокус видим и возвращается, статусы доступны без цвета. Записаны реальные версии Chrome/Arc/macOS; Chromium CI отдельно.
- **Уровень:** `manual+e2e`.

#### AC-090

- **Дано:** В тестах созданы две изолированные семьи; запрос или задача подменяет householdId/actor/resourceId.
- **Когда:** Проверяются чтение файла, импорт, исправление, поиск AI и дедупликация.
- **Тогда:** Чужие объекты недоступны и не объединяются; сервер берёт principal из сессии или проверенного контекста задания. Отказ не раскрывает чужое содержимое.
- **Уровень:** `integration`.

#### AC-004

- **Дано:** История запрошена с 1 августа; начальный остаток RUB 5 000 подтверждён.
- **Когда:** Импортируется расход RUB 500 от 2 августа.
- **Тогда:** Остаток равен RUB 4 500; доход августа не увеличивается на начальные RUB 5 000; неподтверждённое начало обозначается явно.
- **Уровень:** `integration`.

#### AC-040

- **Дано:** Два источника доступны, третий требует повторного входа.
- **Когда:** Срабатывает расписание и одновременно нажата кнопка обновления.
- **Тогда:** Нет параллельного дублирования одного задания; доступные источники обновлены, проблемный имеет отдельный статус и старый timestamp.
- **Уровень:** `integration`.

#### AC-041

- **Дано:** Источник выдаёт несколько страниц с ограничением глубины; второй запрос завершился ошибкой.
- **Когда:** Импорт возобновляется.
- **Тогда:** Подтверждённые страницы сохранены без дублей; курсор не перескакивает пропуск; неполная история и её границы видны.
- **Уровень:** `integration`.

### Проверка результата

```sh
make test-web FILTER=identity && make e2e SCENARIO=access
```

RU/EN вход/восстановление и навигация работают; unauthorized/offline состояния понятны.

Команды `make` — будущий контракт, создаваемый task-1.1; сейчас они не существуют. Live/paid/manual проверки отдельно фиксируют доступ и фактический результат. Исследования не обходят блокер отсутствующего доступа.

### Передача следующему агенту

Записать изменённые контракты, команды и результаты, ограничения, незакрытые вопросы и разблокированные зависимости. Обновить обе языковые версии и трассировку. Закрывать задачу только по доказательству её результата; GitHub Closed само по себе не означает Ready MVP.

**Commit boundary:** логическая граница этой задачи; commit/push/deploy не разрешены данной карточкой и требуют действующей авторизации пользователя.

## EN

Provide sign-in, navigation and accessible states on a macOS laptop in Chrome/Arc.

**Status:** Not started; the task awaits its own dependencies and entry gates.

**Dependencies:** `task-1.4`, `task-1.2`, `task-7.11`, `task-1.6`.

**Kind:** `implementation`.

### Change and contracts

Create a React/TypeScript/Vite shell for macOS Chrome/Arc, SCR-001–SCR-005 routes, shared navigation, passkey/recovery and typed errors. Use task-7.11 design system; reference sign-in has a compact logo, name, whitespace and one primary button. Session stays in a protected cookie. Desktop 1280×720/1440×900, 200% zoom, keyboard/focus; offline never implies saved. Mobile screens and installation are excluded. Bootstrap/invite use task-1.6 policy.

### Change boundaries

- `web/src/app/`
- `web/src/features/identity/`
- `web/src/locales/`

### Screen contract

### SCR-001 — Sign in

`/login`

**Question:** How do I sign in?

**Primary answer:** One sign-in with a personal passkey.

**Top-down structure:** Center: logo, Want Keep, whitespace, primary button; subtle language/help.

**Next action:** Log in with Passkeys → SCR-006 or unfinished SCR-005; help → SCR-002.

**Explanation and details:** System prompt, local error reason and retry; do not copy export dimensions.

**Permissions:** Before sign-in only own authentication; financial data hidden.

Forms: FORM-01.

States: UISTATE-01, UISTATE-07, UISTATE-08, UISTATE-09, UISTATE-10, UISTATE-16, UISTATE-17.

### SCR-002 — Recovery

`/recovery`

**Question:** How do I regain my access?

**Primary answer:** Personal single-use code and a new passkey.

**Top-down structure:** Explanation → code → verification → new passkey → new codes/outcome.

**Next action:** Recover own access, then SCR-006; cancel → SCR-001.

**Explanation and details:** This user’s old sessions revoked; invalid/used/expired code without exposing another identity.

**Permissions:** Before sign-in only own authentication; financial data hidden.

Forms: FORM-01.

States: UISTATE-01, UISTATE-07, UISTATE-08, UISTATE-09, UISTATE-10, UISTATE-16, UISTATE-17.

### SCR-003 — Initial setup

`/setup`

**Question:** How do we start our household?

**Primary answer:** Restricted first-member setup.

**Top-down structure:** Bootstrap verification → name/household → passkey/recovery → invitation.

**Next action:** Create household → SCR-004 or SCR-005.

**Explanation and details:** Repeat bootstrap is closed; no public signup.

**Permissions:** Before sign-in only own authentication; financial data hidden.

Forms: FORM-02.

States: UISTATE-01, UISTATE-07, UISTATE-08, UISTATE-09, UISTATE-10, UISTATE-12, UISTATE-16, UISTATE-17.

### SCR-004 — Invitation

`/invite`

**Question:** How does my partner join?

**Primary answer:** Verified private invitation to a specific household.

**Top-down structure:** Household/inviter after verification → name → own passkey → own codes.

**Next action:** Accept → SCR-005; expired invitation explains obtaining a new one from the member.

**Explanation and details:** Used invitation and full membership have distinct understandable states.

**Permissions:** Before sign-in only own authentication; financial data hidden.

Forms: FORM-02.

States: UISTATE-01, UISTATE-07, UISTATE-08, UISTATE-09, UISTATE-10, UISTATE-12, UISTATE-16, UISTATE-17.

### SCR-005 — Onboarding

`/onboarding`

**Question:** How do I get a useful first overview?

**Primary answer:** Start with cash and accounts already available.

**Top-down structure:** Progress → add account/connection → history date/balances → first plan → overview.

**Next action:** Add FORM-03/13 or continue with available data → SCR-006.

**Explanation and details:** Unknown history stays a visible limitation; unfinished connection does not block available features.

**Permissions:** Both members can read; server checks membership and resource ownership for actions.

Forms: FORM-03, FORM-13.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-14, UISTATE-15.

#### FORM-01 — Passkey and personal recovery

**Fields:** System passkey prompt; recovery: personal code followed by new passkey enrollment.

**Validation and permissions:** Verified challenge/origin/RP; code is hidden and single-use. Partner cannot recover access. No email/password fallback.

**Outcome:** Current-member session; recovery revokes only their old sessions; cancellation returns to sign-in.

#### FORM-02 — Household setup and invitation

**Fields:** Member/household name, language, timezone/currency; private invitation, joining member name and passkey.

**Validation and permissions:** Single-use operator bootstrap; invitation is household-bound, expiring and single-use; configured maximum 2 active members.

**Outcome:** Membership created, personal recovery codes shown; proceed to onboarding. Secrets excluded from URL logs/analytics.

#### FORM-03 — Account and opening balance

**Fields:** Name, product type, currency, personal owner/household, start date, own/borrowed/blocked opening amounts by type.

**Validation and permissions:** Both members create accounts and correct accounting facts. Changing owner or personal/household scope of an existing personal account requires its current owner; either member may change a household account. This never changes verified external ownership or transaction history. Exact decimals and currency required. Imported fields change through correction; opening balance is not income.

**Outcome:** Accounting account created/corrected with audit; this does not open a bank product.

#### FORM-13 — Connection and reauth

**Fields:** Platform, external-account owner, history start, available products; secret only in isolated authorization flow.

**Validation and permissions:** Both manage; external owner alone supplies key/password/MFA. Adapter-confirmed read-only scopes, identity and coverage; no MFA/CAPTCHA bypass.

**Outcome:** Connected/syncing/awaiting owner/error; disconnect preserves history and invalidates generation.

- **UISTATE-01 — Loading:** Structural skeleton and loading label; amounts are never replaced by zero.
- **UISTATE-02 — Refreshing:** Keep previous data/context and last-success time; block only conflicting actions.
- **UISTATE-03 — Empty:** Explain the useful outcome and offer a first account, receipt, plan or goal action.
- **UISTATE-05 — Partial data:** Name the missing source/period and its effect on the amount; available sections work and unknowns stay explicit.
- **UISTATE-06 — Stale data:** Show last-success date and impact on the decision; offer refresh or connection details.
- **UISTATE-07 — Error:** Plain cause and next step beside the affected section; preserve input/healthy data and expand diagnostics separately.
- **UISTATE-08 — Offline:** Show missing connectivity and do not promise saved data. Sensitive drafts remain only in current-tab memory, without a new offline queue.
- **UISTATE-09 — Saving:** Immediately show current-action progress and prevent duplicate command submission.
- **UISTATE-10 — Unknown outcome:** Keep command ID/input and query its result; never blindly create another financial command. After reload reconcile the server list of recent commands.
- **UISTATE-11 — Version conflict:** Show authors/differences and keep my input; load current version and allow chosen changes to be reapplied after validation.
- **UISTATE-12 — Insufficient permission:** Household can read financial data; forbidden edits explain ownership. Server rejects the command regardless of button visibility.
- **UISTATE-13 — Session expired:** Hide protected contents; require the same member to sign in and return through a safe internal route. Another identity never receives the prior draft.
- **UISTATE-14 — AI waiting:** Distinguish queued, processing, clarification and budget/API pause; ordinary accounting remains available and results are not invented.
- **UISTATE-15 — Bank sign-in needed:** Name connection and owner, offer safe owner sign-in; partner sees waiting without secret access.
- **UISTATE-16 — Confirmed:** After server-confirmed outcome show what changed, an object link and available correction; do not rely on a disappearing toast.
- **UISTATE-17 — Cancelled:** Explain that no new outcome was confirmed and offer explicit retry; cancelled system passkey prompts are not a malfunction.


These are planned paths. Shared contracts: `spec/001-want-keep-mvp/contracts.en.md`; architecture and commands: `constraints.en.md`. Change only the behavior owner and affected tests; an unresolved contract requires updated evidence and stops dependent implementation.

### Linked requirements

- **REQ-001:** The family pilot serves two members with separate sign-in and restricted joining; public registration is unavailable.
- **REQ-004:** Accounting starts on a selected date; opening balances are separate from income and expenses.
- **REQ-040:** Each source refreshes hourly and on demand with a visible last-success timestamp.
- **REQ-041:** History retains coverage boundaries, cursors, gaps and source status.
- **REQ-049:** Each member signs in with their own passkeys and one-time recovery codes; partner-assisted reset is unavailable.
- **REQ-050:** Files, source keys, sessions and financial records are protected against unauthorized access.
- **REQ-053:** Reminders and summaries are available in-app and through authorized web push.
- **REQ-054:** UI, chat and documentation support RU/EN without changing financial semantics.
- **REQ-055:** The web app targets macOS laptops in Chrome and Arc; window resizing and zoom preserve daily accounting access.
- **REQ-063:** User, household and membership are separate models; the two-member limit is configured.
- **REQ-064:** Both members see all financial data and edit transactions; only the owner edits personal goals and plan portions.
- **REQ-076:** Household scope is checked for APIs, files, AI, jobs and external IDs independently of supplied actor/owner fields.
- **REQ-081:** Desktop navigation preserves context and changing household views never changes authority.
- **REQ-082:** Screen states explain consequences and a safe next step without losing input.
- **REQ-083:** Sign-in preserves the supplied reference composition and personal access recovery.
- **REQ-084:** Accessibility is checked in actual Chrome and Arc, including keyboard, focus, contrast, zoom and reduced motion.

### Acceptance criteria

A criterion link establishes coverage; research or a partial task does not prove the entire product criterion. This task's exact outcome is specified in verification below.

#### AC-001

- **Given:** A household exists, the first member is signed in and the active-member limit is 2.
- **When:** The second member accepts an invitation; an outsider attempts open registration, invitation replay and joining beyond the limit.
- **Then:** The invitation creates one separate membership; outsiders receive no data and replay or exceeding the limit is rejected.
- **Level:** `end-to-end`.

#### AC-049

- **Given:** Both members enrolled their own passkeys and personal recovery codes.
- **When:** A member recovers their sign-in, reuses a code, tries an alien origin and attempts to reset their partner’s sign-in.
- **Then:** Own access is restored with own old sessions/subscriptions revoked; the partner’s sessions remain; code reuse, alien origins and resetting the partner’s sign-in fail.
- **Level:** `end-to-end`.

#### AC-050

- **Given:** A private receipt and an active source connection exist.
- **When:** A direct file URL, unauthenticated export, logs and disconnection are checked.
- **Then:** Unauthenticated access fails; secrets are encrypted and not logged; disconnecting stops further collection.
- **Level:** `integration`.

#### AC-054

- **Given:** Russian and English versions of the same transaction, budget and error exist.
- **When:** The language is switched.
- **Then:** Amounts, dates, currencies and meaning agree; formatting is localized while IDs and owner categories are not destructively translated.
- **Level:** `end-to-end+static`.

#### AC-055

- **Given:** The owner reviews a day containing expenses, a receipt, clarification, budget and goal.
- **When:** A member completes the flow in actual Chrome and Arc at 1280×720 and 1440×900 CSS px, then zooms to 200%.
- **Then:** Core actions work without data loss or horizontally scrolling forms; observed flow time is recorded against the owner's up-to-45-minutes/day target.
- **Level:** `manual`.

#### AC-072

- **Given:** Two sessions and a push subscription are active.
- **When:** The owner recovers access and revokes an old device.
- **Then:** Old sessions/associated subscriptions are revoked; push links require current authorization; push contains no financial details by default.
- **Level:** `end-to-end+manual`.

#### AC-075

- **Given:** The same receipt-entry/category-correction flow exists in both languages.
- **When:** The flow runs with a keyboard in Chrome and Arc on macOS at both reference sizes and with zoom.
- **Then:** Required fields and errors remain accessible; language switching preserves input; amounts are localized only for display.
- **Level:** `end-to-end+manual`.

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

#### AC-098

- **Given:** The user filtered month, currency, member and list.
- **When:** They open details, return, change language and open a direct link.
- **Then:** Overview opens after sign-in; left navigation contains Overview, Money, Plan, Analytics, Chat, lower Connections/Settings and top notifications. Context survives; the server checks the current actor independently of filters.
- **Level:** `manual+e2e`.

#### AC-099

- **Given:** Loading, empty list/search, stale/partial data, offline, failure and concurrent edits occur.
- **When:** The user reads or saves.
- **Then:** Unknown never becomes zero and success follows readback; unknown outcomes are reconciled by command ID before another creation. Conflicts retain input and offer comparison. Session expiry leads to sign-in, bank reauth to the proper owner, and AI waiting does not block ordinary accounting.
- **Level:** `manual+e2e`.

#### AC-100

- **Given:** A member opens /login in either language.
- **When:** Passkey sign-in is waiting, cancelled or failed.
- **Then:** Compact centered logo/Want Keep, whitespace, one primary “Войти с passkey” / “Log in with Passkeys” button, subtle language/help. Status preserves composition; help leads to personal recovery. SVG proportions remain intact.
- **Level:** `manual+e2e`.

#### AC-101

- **Given:** Screens and forms are available in RU/EN on macOS.
- **When:** 1280×720 and 1440×900 CSS px, 100%/200% zoom, keyboard and reduced motion are tested.
- **Then:** No hidden actions, clipped amounts or horizontally scrolling forms; tables have a labelled scroll region when needed. Normal text contrast ≥4.5:1, large text ≥3:1, meaningful boundaries/focus ≥3:1. Focus is visible and restored; states work without color. Actual Chrome/Arc/macOS versions are recorded separately from Chromium CI.
- **Level:** `manual+e2e`.

#### AC-090

- **Given:** Tests contain two isolated households; a request or job forges householdId/actor/resourceId.
- **When:** File reads, import, correction, AI retrieval and deduplication are exercised.
- **Then:** Foreign objects are inaccessible and never merged; the server takes principal from the session or validated job context. Denial reveals no foreign content.
- **Level:** `integration`.

#### AC-004

- **Given:** History is requested from August 1; an opening RUB 5,000 balance is confirmed.
- **When:** A RUB 500 expense dated August 2 is imported.
- **Then:** Balance is RUB 4,500; August income excludes the opening RUB 5,000; an unverified opening is explicit.
- **Level:** `integration`.

#### AC-040

- **Given:** Two sources are available and a third requires sign-in again.
- **When:** The schedule fires while the refresh button is pressed.
- **Then:** The same job is not duplicated concurrently; available sources refresh and the failing source has its own status and old timestamp.
- **Level:** `integration`.

#### AC-041

- **Given:** A source provides paginated history with a retention limit; the second request fails.
- **When:** Import resumes.
- **Then:** Confirmed pages remain without duplicates; the cursor does not skip the gap; incomplete history and its boundaries are visible.
- **Level:** `integration`.

### Verification

```sh
make test-web FILTER=identity && make e2e SCENARIO=access
```

RU/EN sign-in/recovery/navigation work; unauthorized/offline states are understandable.

The `make` commands are a future contract established by task-1.1; they do not exist yet. Live/paid/manual checks separately record access and actual outcomes. Research does not bypass missing-access blockers.

### Handoff to the next agent

Record changed contracts, commands/results, limitations, unresolved questions and unblocked dependencies. Update both languages and traceability. Close the task only with evidence of its outcome; GitHub Closed alone does not mean the MVP is Ready.

**Commit boundary:** this task's logical boundary; this card does not authorize commit/push/deploy, which require current user authorization.
