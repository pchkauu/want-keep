<!-- want-keep-task: task-2.6 -->
# task-2.6 — Разделить категории, продавцов и товары / Separate categories, merchants and items

## RU

Дать независимую категоризацию с сохранением правок владельца.

**Состояние:** Не начато; задача ожидает собственные зависимости и entry gates.

**Зависимости:** `task-2.3`.

**Тип:** `implementation`.

### Изменение и контракты

Создать редактируемые категории/подкатегории, отдельного продавца с подтверждёнными алиасами и позиции чеков. Категории расхода не кодировать названиями продавцов. AI работает через этот публичный контракт; неоднозначный продавец/категория обозначается явно, исправления сохраняют историю и не меняют утверждённый бюджет.

### Границы изменений

- `backend/internal/categories/`

Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-014:** Категория, подкатегория, продавец и позиция чека являются отдельными аналитическими признаками.
- **REQ-016:** Позиции чека распределяют одну оплаченную сумму по категориям без дублирования итога.
- **REQ-054:** Интерфейс, чат и документация поддерживают RU/EN без изменения финансовой семантики.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

#### AC-014

- **Дано:** Чек содержит молоко, продавец — условный магазин; другая покупка — ресторан.
- **Когда:** Владелец фильтрует расходы и исправляет категорию.
- **Тогда:** Доступны независимые срезы по виду расхода, продавцу и товару; пользовательская правка сохраняется.
- **Уровень:** `end-to-end`.

#### AC-016

- **Дано:** Оплачено RUB 900 за позиции RUB 600 и RUB 400 со скидкой RUB 100.
- **Когда:** AI извлекает и категоризирует позиции.
- **Тогда:** Сумма распределений точно RUB 900; скидка сохраняется; расхождение суммы направляется на уточнение, а не исправляется выдуманной позицией.
- **Уровень:** `integration`.

#### AC-054

- **Дано:** Есть русская и английская версии одной операции, бюджета и ошибки.
- **Когда:** Переключается язык.
- **Тогда:** Суммы, даты, валюты и смысл совпадают; форматирование локализовано, идентификаторы и категории пользователя не переводятся с потерей данных.
- **Уровень:** `end-to-end+static`.

### Проверка результата

```sh
make test-go PKG=./internal/categories/...
```

Независимые срезы и исправления работают на RU/EN; повторная классификация не теряет owner override.

Команды `make` — будущий контракт, создаваемый task-1.1; сейчас они не существуют. Live/paid/manual проверки отдельно фиксируют доступ и фактический результат. Исследования не обходят блокер отсутствующего доступа.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Provide independent classification while retaining owner corrections.

**Status:** Not started; the task awaits its own dependencies and entry gates.

**Dependencies:** `task-2.3`.

**Kind:** `implementation`.

### Change and contracts

Create editable categories/subcategories, separate merchants with verified aliases and receipt items. Do not encode merchants as expense types. AI uses this public contract; ambiguous merchants/categories stay explicit; corrections preserve history without changing approved budgets.

### Change boundaries

- `backend/internal/categories/`

Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-014:** Category, subcategory, merchant and receipt item are separate analytical dimensions.
- **REQ-016:** Receipt items allocate one paid amount across categories without duplicating the total.
- **REQ-054:** UI, chat and documentation support RU/EN without changing financial semantics.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

#### AC-014

- **Given:** A receipt contains milk from a fictional store; another purchase is from a restaurant.
- **When:** The owner filters expenses and corrects a category.
- **Then:** Expense type, merchant and item can be analyzed independently; owner corrections persist.
- **Level:** `end-to-end`.

#### AC-016

- **Given:** RUB 900 was paid for RUB 600 and RUB 400 items with a RUB 100 discount.
- **When:** AI extracts and categorizes the items.
- **Then:** Allocations total exactly RUB 900 and retain the discount; a mismatch is clarified rather than patched with an invented item.
- **Level:** `integration`.

#### AC-054

- **Given:** Russian and English versions of the same transaction, budget and error exist.
- **When:** The language is switched.
- **Then:** Amounts, dates, currencies and meaning agree; formatting is localized while IDs and owner categories are not destructively translated.
- **Level:** `end-to-end+static`.

### Verification

```sh
make test-go PKG=./internal/categories/...
```

Independent dimensions and corrections work in RU/EN; reclassification preserves owner overrides.

The `make` commands are a future contract established by task-1.1; they do not exist yet. Live/paid/manual checks separately record access and actual outcomes. Research does not bypass missing-access blockers.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
