<!-- want-keep-task: task-7.11 -->
# task-7.11 — Создать каталог компонентов на Base UI / Create the Base UI component catalog

## RU

Доступные компоненты с собственным оформлением.

**Состояние:** Не начато; задача ожидает собственные зависимости и entry gates.

**Зависимости:** `task-7.10`.

**Тип:** `implementation`.

### Изменение и контракты

Зафиксировать shadcn/ui на Base UI, проверить состав перед генерацией, кастомизировать принадлежащие проекту исходники. Каталог: buttons, inputs/money/date/select, forms, dialog/alert/sheet, tabs/sidebar, table, badges/alerts, tooltip, skeleton/empty, charts legend и chat result. Все default/hover/focus/disabled/error/loading/selected и UISTATE. Без финансовых вычислений в компонентах; каталог доступен только в dev/test.

### Границы изменений

- `web/src/design-system/`
- `web/src/design-system/catalog/`

Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-077:** Тёмные токены Want Keep: сдержанный киберпанк и архитектурный ритм Ближнего Востока.
- **REQ-078:** Pixelify Sans используется для бренда и крупных акцентов, Manrope — для повседневного интерфейса.
- **REQ-079:** Компоненты shadcn/ui на Base UI принадлежат проекту и оформляются собственными семантическими токенами.
- **REQ-082:** Экранные состояния объясняют последствия и безопасный следующий шаг без потери ввода.
- **REQ-084:** Доступность проверяется на реальных Chrome и Arc, включая клавиатуру, фокус, контраст, масштаб и reduced motion.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

#### AC-094

- **Дано:** Обзор, вход, форма и таблица используют одну дизайн-систему.
- **Когда:** Проверяются фон, поверхности, акценты и состояния.
- **Тогда:** Фон #1A1A1A, рабочие поверхности #202020–#262626, бренд #5F4EF5; нет светлой темы. Брендовые плоскости #111114/#18171E, редкие песочные акценты #C7AF8F. Нет обводок карточек/полей; группировка заливкой, пространством и типографикой. Фокус заметен инверсией заливки; статусы имеют текст/значок.
- **Уровень:** `manual+e2e`.

#### AC-095

- **Дано:** Есть RU/EN строки, RUB/USD/USDT/USDC/BTC/ETH и длинные точные суммы.
- **Когда:** Проверяются загруженные и недоступные шрифты.
- **Тогда:** Шрифты размещены локально с лицензиями; кириллица и валютные символы читаемы, fallback не теряет символы или разряды. Таблицы и формы используют Manrope, крупные суммы могут использовать Pixelify Sans.
- **Уровень:** `manual+e2e`.

#### AC-096

- **Дано:** Каталог содержит кнопки, поля, диалоги, панели, таблицы и состояния данных.
- **Когда:** Проверяются состояния и реализация примитивов.
- **Тогда:** Зафиксированы версии и Base UI; компоненты не сохраняют стандартное оформление shadcn, не дублируют бизнес-логику и поддерживают клавиатуру, подписи и фокус.
- **Уровень:** `manual+e2e`.

#### AC-099

- **Дано:** Есть загрузка, пустой список/поиск, устаревшие/частичные данные, offline, отказ и конкурирующие правки.
- **Когда:** Пользователь выполняет чтение или сохранение.
- **Тогда:** Неизвестное не становится нулём, подтверждение даётся после readback; неизвестный исход проверяется по ID команды до повторного создания. Конфликт сохраняет ввод и предлагает сравнение. Истечение сессии ведёт к входу, банковская reauth — к нужному владельцу, ожидание AI не блокирует обычный учёт.
- **Уровень:** `manual+e2e`.

#### AC-101

- **Дано:** Экраны и формы доступны в RU/EN на macOS.
- **Когда:** Проверяются 1280×720 и 1440×900 CSS px, масштаб 100%/200%, клавиатура и уменьшение движения.
- **Тогда:** Нет скрытых действий, обрезанных сумм и горизонтальной прокрутки форм; таблицы при необходимости имеют обозначенную область прокрутки. Контраст обычного текста ≥4.5:1, крупного ≥3:1, значимых границ/фокуса ≥3:1. Фокус видим и возвращается, статусы доступны без цвета. Записаны реальные версии Chrome/Arc/macOS; Chromium CI отдельно.
- **Уровень:** `manual+e2e`.

### Проверка результата

```sh
make test-web FILTER=design-components
```

Состояния доступны клавиатурой, фокус возвращается, подписи и ошибки читаемы; стандартное оформление заменено по design.

Команды `make` — будущий контракт, создаваемый task-1.1; сейчас они не существуют. Live/paid/manual проверки отдельно фиксируют доступ и фактический результат. Исследования не обходят блокер отсутствующего доступа.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Accessible components with custom styling.

**Status:** Not started; the task awaits its own dependencies and entry gates.

**Dependencies:** `task-7.10`.

**Kind:** `implementation`.

### Change and contracts

Pin shadcn/ui on Base UI, inspect selected generation before applying, customize project-owned sources. Catalog: buttons, inputs/money/date/select, forms, dialog/alert/sheet, tabs/sidebar, table, badges/alerts, tooltip, skeleton/empty, chart legends and chat results. Cover default/hover/focus/disabled/error/loading/selected and UISTATE. No financial calculations in components; catalog is dev/test only.

### Change boundaries

- `web/src/design-system/`
- `web/src/design-system/catalog/`

Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-077:** Dark Want Keep tokens: restrained cyberpunk and Middle Eastern architectural rhythm.
- **REQ-078:** Pixelify Sans serves branding and large accents; Manrope serves everyday UI.
- **REQ-079:** Project-owned shadcn/ui components on Base UI use custom semantic tokens.
- **REQ-082:** Screen states explain consequences and a safe next step without losing input.
- **REQ-084:** Accessibility is checked in actual Chrome and Arc, including keyboard, focus, contrast, zoom and reduced motion.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

#### AC-094

- **Given:** Overview, sign-in, form and table use one design system.
- **When:** Backgrounds, surfaces, accents and states are inspected.
- **Then:** Base #1A1A1A, working surfaces #202020–#262626, brand #5F4EF5; no light theme. Brand planes #111114/#18171E, sparse sand accents #C7AF8F. No card/field outlines; group through fills, space and typography. Focus uses visible inverted fill; statuses have text/icons.
- **Level:** `manual+e2e`.

#### AC-095

- **Given:** RU/EN strings, RUB/USD/USDT/USDC/BTC/ETH and long precise amounts exist.
- **When:** Loaded and unavailable fonts are tested.
- **Then:** Fonts are hosted locally with licenses; Cyrillic and currency symbols remain legible and fallback loses no glyphs or digits. Tables/forms use Manrope; large amounts may use Pixelify Sans.
- **Level:** `manual+e2e`.

#### AC-096

- **Given:** A catalog contains buttons, fields, dialogs, panels, tables and data states.
- **When:** States and primitive implementations are inspected.
- **Then:** Versions and Base UI are pinned; components replace stock shadcn styling, contain no duplicated business rules and support keyboard, labels and focus.
- **Level:** `manual+e2e`.

#### AC-099

- **Given:** Loading, empty list/search, stale/partial data, offline, failure and concurrent edits occur.
- **When:** The user reads or saves.
- **Then:** Unknown never becomes zero and success follows readback; unknown outcomes are reconciled by command ID before another creation. Conflicts retain input and offer comparison. Session expiry leads to sign-in, bank reauth to the proper owner, and AI waiting does not block ordinary accounting.
- **Level:** `manual+e2e`.

#### AC-101

- **Given:** Screens and forms are available in RU/EN on macOS.
- **When:** 1280×720 and 1440×900 CSS px, 100%/200% zoom, keyboard and reduced motion are tested.
- **Then:** No hidden actions, clipped amounts or horizontally scrolling forms; tables have a labelled scroll region when needed. Normal text contrast ≥4.5:1, large text ≥3:1, meaningful boundaries/focus ≥3:1. Focus is visible and restored; states work without color. Actual Chrome/Arc/macOS versions are recorded separately from Chromium CI.
- **Level:** `manual+e2e`.

### Verification

```sh
make test-web FILTER=design-components
```

States work by keyboard, focus returns and labels/errors are legible; stock visuals are replaced according to design.

The `make` commands are a future contract established by task-1.1; they do not exist yet. Live/paid/manual checks separately record access and actual outcomes. Research does not bypass missing-access blockers.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
