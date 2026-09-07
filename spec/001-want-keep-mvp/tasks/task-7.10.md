<!-- want-keep-task: task-7.10 -->
# task-7.10 — Создать токены, типографику и брендовые ресурсы / Create tokens, typography and brand assets

## RU

Единый визуальный фундамент desktop Want Keep.

**Состояние:** Не начато; задача ожидает собственные зависимости и entry gates.

**Зависимости:** `task-1.1`.

**Тип:** `implementation`.

### Изменение и контракты

По design.md создать semantic colors/spacing/radius/type/motion tokens; сохранить logo_512px.svg без изменения пропорций. Локально разместить Pixelify Sans и Manrope с лицензиями и fallback. Рассчитать контраст каждой пары; брендовый фиолетовый не использовать как мелкий текст на тёмном фоне. Без темы light.

### Границы изменений

- `web/src/design-system/`
- `web/public/fonts/`
- `web/public/brand/`

Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-077:** Только тёмная тема с токенами Want Keep и сдержанной pixel-айдентикой.
- **REQ-078:** Pixelify Sans используется для бренда и крупных акцентов, Manrope — для повседневного интерфейса.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

#### AC-094

- **Дано:** Обзор, вход, форма и таблица используют одну дизайн-систему.
- **Когда:** Проверяются фон, поверхности, акценты и состояния.
- **Тогда:** Фон #1A1A1A, поверхности #202020–#262626, бренд #5F4EF5; нет переключателя светлой темы. Семантические статусы имеют текст/значок, разделители тонкие, тени минимальны.
- **Уровень:** `manual+e2e`.

#### AC-095

- **Дано:** Есть RU/EN строки, RUB/USD/USDT/BTC и длинные точные суммы.
- **Когда:** Проверяются загруженные и недоступные шрифты.
- **Тогда:** Шрифты размещены локально с лицензиями; кириллица и валютные символы читаемы, fallback не теряет символы или разряды. Таблицы и формы используют Manrope, крупные суммы могут использовать Pixelify Sans.
- **Уровень:** `manual+e2e`.

### Проверка результата

```sh
make test-web FILTER=design-tokens
```

Токены воспроизводимы, шрифты RU/EN и длинные суммы читаемы, лицензии включены, contrast matrix приложена.

Команды `make` — будущий контракт, создаваемый task-1.1; сейчас они не существуют. Live/paid/manual проверки отдельно фиксируют доступ и фактический результат. Исследования не обходят блокер отсутствующего доступа.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

A consistent desktop Want Keep visual foundation.

**Status:** Not started; the task awaits its own dependencies and entry gates.

**Dependencies:** `task-1.1`.

**Kind:** `implementation`.

### Change and contracts

Implement semantic color/spacing/radius/type/motion tokens from design.en.md; preserve logo_512px.svg proportions. Host Pixelify Sans and Manrope locally with licenses and fallbacks. Calculate each contrast pair; do not use brand purple as small text on dark surfaces. No light theme.

### Change boundaries

- `web/src/design-system/`
- `web/public/fonts/`
- `web/public/brand/`

Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-077:** Dark-only Want Keep tokens and restrained pixel identity.
- **REQ-078:** Pixelify Sans serves branding and large accents; Manrope serves everyday UI.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

#### AC-094

- **Given:** Overview, sign-in, form and table use one design system.
- **When:** Backgrounds, surfaces, accents and states are inspected.
- **Then:** Base #1A1A1A, surfaces #202020–#262626, brand #5F4EF5; no light-theme selector. Semantic states have text/icons, thin separators and minimal shadows.
- **Level:** `manual+e2e`.

#### AC-095

- **Given:** RU/EN strings, RUB/USD/USDT/BTC and long precise amounts exist.
- **When:** Loaded and unavailable fonts are tested.
- **Then:** Fonts are hosted locally with licenses; Cyrillic and currency symbols remain legible and fallback loses no glyphs or digits. Tables/forms use Manrope; large amounts may use Pixelify Sans.
- **Level:** `manual+e2e`.

### Verification

```sh
make test-web FILTER=design-tokens
```

Tokens are reproducible, RU/EN fonts and long amounts are readable, licenses included and contrast matrix attached.

The `make` commands are a future contract established by task-1.1; they do not exist yet. Live/paid/manual checks separately record access and actual outcomes. Research does not bypass missing-access blockers.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
