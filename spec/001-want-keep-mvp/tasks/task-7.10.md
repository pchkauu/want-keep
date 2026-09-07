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

Это планируемые пути. Общие контракты: `spec/001-want-keep-mvp/contracts.md`; архитектура и команды: `constraints.md`. Менять только владельца поведения и затронутые тесты; при незакрытом контракте обновить evidence и остановить зависимую реализацию.

### Связанные требования

- **REQ-077:** Только тёмная тема с токенами Want Keep и сдержанной pixel-айдентикой.
- **REQ-078:** Pixelify Sans используется для бренда и крупных акцентов, Manrope — для повседневного интерфейса.

### Критерии приёмки

Связь с критерием задаёт покрытие; исследование или частичная задача не доказывает весь критерий продукта. Точный результат этой задачи указан ниже в проверке.

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

Записать изменённые контракты, команды и результаты, ограничения, незакрытые вопросы и разблокированные зависимости. Обновить обе языковые версии и трассировку. Закрывать задачу только по доказательству её результата; GitHub Closed само по себе не означает Ready MVP.

**Commit boundary:** логическая граница этой задачи; commit/push/deploy не разрешены данной карточкой и требуют действующей авторизации пользователя.

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

These are planned paths. Shared contracts: `spec/001-want-keep-mvp/contracts.en.md`; architecture and commands: `constraints.en.md`. Change only the behavior owner and affected tests; an unresolved contract requires updated evidence and stops dependent implementation.

### Linked requirements

- **REQ-077:** Dark-only Want Keep tokens and restrained pixel identity.
- **REQ-078:** Pixelify Sans serves branding and large accents; Manrope serves everyday UI.

### Acceptance criteria

A criterion link establishes coverage; research or a partial task does not prove the entire product criterion. This task's exact outcome is specified in verification below.

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

Record changed contracts, commands/results, limitations, unresolved questions and unblocked dependencies. Update both languages and traceability. Close the task only with evidence of its outcome; GitHub Closed alone does not mean the MVP is Ready.

**Commit boundary:** this task's logical boundary; this card does not authorize commit/push/deploy, which require current user authorization.
