<!-- want-keep-task: task-7.10 -->
# task-7.10 — Создать токены, типографику и брендовые ресурсы / Create tokens, typography and brand assets

## RU

Единый визуальный фундамент desktop Want Keep.

**Состояние:** Реализована визуальная основа и dev/test-образцы; task-1.1 включена в базу. Продуктовые экраны и общий каталог компонентов остаются последующим задачам.

**Зависимости:** `task-1.1`.

**Тип:** `implementation`.

### Изменение и контракты

Единый источник CSS-токенов: графит, обсидиановые брендовые плоскости, фиолетовый и песочный акцент по референсам пользователя. Без обводок; видимый фокус через инверсию. Сохранить shadcn/Tailwind-алиасы, logo_512px.svg побайтно, полные локальные variable Pixelify Sans/Manrope с OFL, закреплёнными источниками и SHA-256. Точные суммы остаются строками; табличные цифры — Manrope. Длительности и reduced motion без финансовых триггеров. /__design/tokens не входит в production bundle.

### Границы изменений

- `web/src/design-system/`
- `web/public/fonts/`
- `web/public/brand/`

Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-077:** Тёмные токены Want Keep: сдержанный киберпанк и архитектурный ритм Ближнего Востока.
- **REQ-078:** Pixelify Sans используется для бренда и крупных акцентов, Manrope — для повседневного интерфейса.

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

### Проверка результата

```sh
make test-web FILTER=design-tokens
```

Токены воспроизводимы, шрифты RU/EN и длинные суммы читаемы, лицензии включены, contrast matrix приложена.

Команды существуют: make check; make test-web FILTER=design-tokens; make e2e SCENARIO=design-tokens; git diff --check. Матрица: design-contrast.md; покрытие шрифтов и границы: evidence/task-7.10-design.md. По уточнению пользователя ручная проверка этой задачи использует Chrome; Arc остаётся последующей продуктовой приёмке.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

A consistent desktop Want Keep visual foundation.

**Status:** Visual foundation and dev/test specimens implemented; task-1.1 is included in the base. Product screens and the full component catalog remain downstream.

**Dependencies:** `task-1.1`.

**Kind:** `implementation`.

### Change and contracts

One CSS token source: graphite, obsidian brand planes, violet and sand accent following user references. No outlines; visible focus through inversion. Preserve shadcn/Tailwind aliases, byte-identical logo_512px.svg, full local variable Pixelify Sans/Manrope with OFL, pinned sources and SHA-256. Exact amounts remain strings; tabular figures use Manrope. Durations/reduced motion without financial triggers. /__design/tokens stays outside the production bundle.

### Change boundaries

- `web/src/design-system/`
- `web/public/fonts/`
- `web/public/brand/`

Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-077:** Dark Want Keep tokens: restrained cyberpunk and Middle Eastern architectural rhythm.
- **REQ-078:** Pixelify Sans serves branding and large accents; Manrope serves everyday UI.

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

### Verification

```sh
make test-web FILTER=design-tokens
```

Tokens are reproducible, RU/EN fonts and long amounts are readable, licenses included and contrast matrix attached.

Commands exist: make check; make test-web FILTER=design-tokens; make e2e SCENARIO=design-tokens; git diff --check. Matrix: design-contrast.md; font coverage and boundaries: evidence/task-7.10-design.en.md. Per user refinement this task uses Chrome for manual verification; Arc remains later product acceptance.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
