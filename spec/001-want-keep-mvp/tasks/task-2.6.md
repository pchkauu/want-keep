<!-- want-keep-task: task-2.6 -->
# task-2.6 — Разделить категории, продавцов и товары / Separate categories, merchants and items

## RU

Дать независимую категоризацию с сохранением правок владельца.

**Состояние:** Заблокировано зависимостями и проверкой SDD Ready; реализация не начата.

**Зависимости:** `task-2.3`.

**Тип:** `implementation`.

### Изменение и контракты

Создать редактируемые категории/подкатегории, отдельного продавца с подтверждёнными алиасами и позиции чеков. Категории расхода не кодировать названиями продавцов. AI работает через этот публичный контракт; неоднозначный продавец/категория обозначается явно, исправления сохраняют историю и не меняют утверждённый бюджет.

### Границы изменений

- `backend/internal/categories/`

Это планируемые пути. Общие контракты: `spec/001-want-keep-mvp/contracts.md`; архитектура и команды: `constraints.md`. Менять только владельца поведения и затронутые тесты; при незакрытом контракте обновить evidence и остановить зависимую реализацию.

### Связанные требования

- **REQ-014:** Категория, подкатегория, продавец и позиция чека являются отдельными аналитическими признаками.
- **REQ-016:** Позиции чека распределяют одну оплаченную сумму по категориям без дублирования итога.
- **REQ-054:** Интерфейс, чат и документация поддерживают RU/EN без изменения финансовой семантики.

### Критерии приёмки

Связь с критерием задаёт покрытие; исследование или частичная задача не доказывает весь критерий продукта. Точный результат этой задачи указан ниже в проверке.

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

Записать изменённые контракты, команды и результаты, ограничения, незакрытые вопросы и разблокированные зависимости. Обновить обе языковые версии и трассировку. Закрывать задачу только по доказательству её результата; GitHub Closed само по себе не означает Ready MVP.

**Commit boundary:** логическая граница этой задачи; commit/push/deploy не разрешены данной карточкой и требуют действующей авторизации пользователя.

## EN

Provide independent classification while retaining owner corrections.

**Status:** Blocked by dependencies and the SDD Ready gate; implementation has not started.

**Dependencies:** `task-2.3`.

**Kind:** `implementation`.

### Change and contracts

Create editable categories/subcategories, separate merchants with verified aliases and receipt items. Do not encode merchants as expense types. AI uses this public contract; ambiguous merchants/categories stay explicit; corrections preserve history without changing approved budgets.

### Change boundaries

- `backend/internal/categories/`

These are planned paths. Shared contracts: `spec/001-want-keep-mvp/contracts.en.md`; architecture and commands: `constraints.en.md`. Change only the behavior owner and affected tests; an unresolved contract requires updated evidence and stops dependent implementation.

### Linked requirements

- **REQ-014:** Category, subcategory, merchant and receipt item are separate analytical dimensions.
- **REQ-016:** Receipt items allocate one paid amount across categories without duplicating the total.
- **REQ-054:** UI, chat and documentation support RU/EN without changing financial semantics.

### Acceptance criteria

A criterion link establishes coverage; research or a partial task does not prove the entire product criterion. This task's exact outcome is specified in verification below.

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

Record changed contracts, commands/results, limitations, unresolved questions and unblocked dependencies. Update both languages and traceability. Close the task only with evidence of its outcome; GitHub Closed alone does not mean the MVP is Ready.

**Commit boundary:** this task's logical boundary; this card does not authorize commit/push/deploy, which require current user authorization.
