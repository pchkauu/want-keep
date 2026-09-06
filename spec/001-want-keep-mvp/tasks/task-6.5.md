<!-- want-keep-task: task-6.5 -->
# task-6.5 — Сводить торговый результат и майнинг / Aggregate trading P&L and mining

## RU

Получать согласованный результат по криптопродуктам без двойных начислений.

**Состояние:** Заблокировано зависимостями и проверкой SDD Ready; реализация не начата.

**Зависимости:** `task-4.4`, `task-4.6`, `task-6.1`, `task-2.4`.

**Тип:** `implementation`.

### Изменение и контракты

Через доменные контракты разделить gross/net realized P&L, unrealized P&L, fees, funding и rewards. Учитывать общие IDs нескольких журналов, transfers Funding/Spot/Earn/Coinhold/mining и капитализацию; нереализованный результат не считать наличным доходом. Не добавлять торговый терминал или управление майнингом.

### Границы изменений

- `backend/internal/trading/`
- `backend/internal/mining/`

Это планируемые пути. Общие контракты: `spec/001-want-keep-mvp/contracts.md`; архитектура и команды: `constraints.md`. Менять только владельца поведения и затронутые тесты; при незакрытом контракте обновить evidence и остановить зависимую реализацию.

### Связанные требования

- **REQ-007:** Обмен и P2P-конвертация собственных денег сохраняют обе валютные суммы, фактический курс и комиссии.
- **REQ-035:** Торговая аналитика отделяет реализованный результат, нереализованный результат, комиссии и funding.
- **REQ-036:** Вознаграждения майнинга отделены от переводов между собственными кошельками.
- **REQ-045:** Интеграция Bybit автоматически читает Funding, Spot, Earn, P2P и фьючерсы в пределах подтверждённого контракта.
- **REQ-047:** Интеграция EMCD автоматически читает кошелёк, Coinhold, P2P, криптокарту и майнинг в пределах подтверждённого контракта.

### Критерии приёмки

Связь с критерием задаёт покрытие; исследование или частичная задача не доказывает весь критерий продукта. Точный результат этой задачи указан ниже в проверке.

#### AC-007

- **Дано:** Собственные RUB 9 000 обменены на USDT 100, комиссия RUB 50.
- **Когда:** Приходят банковская и криптовалютная стороны подтверждённого обмена.
- **Тогда:** Основные суммы не становятся расходом/доходом; курс RUB 90/USDT отделён от комиссии; неоднозначная связь требует уточнения.
- **Уровень:** `integration`.

#### AC-035

- **Дано:** Источник передал реализованный результат USDT 10, комиссию USDT 1 и нереализованный результат USDT 5.
- **Когда:** Обновляется отчёт торгового счёта.
- **Тогда:** Показатели разделены; нереализованные USDT 5 не становятся полученным доходом; чистый результат не дублируется отдельным повторным вычетом уже включённой комиссии.
- **Уровень:** `integration`.

#### AC-036

- **Дано:** Награда BTC 0.0001 начислена на mining-счёт и затем переведена на основной.
- **Когда:** Обе записи импортируются.
- **Тогда:** Доход учитывается при подтверждённом начислении один раз; последующий перевод дохода не создаёт.
- **Уровень:** `integration`.

#### AC-045

- **Дано:** Подключён разрешённый личный аккаунт Bybit с тестируемыми продуктами.
- **Когда:** Запрошены счета, остатки, операции и необходимые условия продуктов.
- **Тогда:** Для каждого обязательного продукта получены сопоставимые с источником данные и свидетельство чтения; отсутствие доступа фиксируется блокером, а не успешным покрытием.
- **Уровень:** `contract+manual`.

#### AC-047

- **Дано:** Подключён разрешённый личный аккаунт EMCD с тестируемыми продуктами.
- **Когда:** Запрошены счета, остатки, операции и необходимые условия продуктов.
- **Тогда:** Для каждого обязательного продукта получены сопоставимые с источником данные и свидетельство чтения; отсутствие доступа фиксируется блокером, а не успешным покрытием.
- **Уровень:** `contract+manual`.

#### AC-071

- **Дано:** Источник различает gross P&L, net P&L, fee, funding и reward/transfer.
- **Когда:** Одна экономическая операция встречается в нескольких журналах.
- **Тогда:** Происхождение показателей сохранено; комиссия и доход не удваиваются; выбор net/gross подтверждён контрактом.
- **Уровень:** `contract+integration`.

### Проверка результата

```sh
make test-integration AREA=crypto-results
```

Контрактные cases gross/net, funding, mining→wallet и капитализация дают единственный экономический эффект.

Команды `make` — будущий контракт, создаваемый task-1.1; сейчас они не существуют. Live/paid/manual проверки отдельно фиксируют доступ и фактический результат. Исследования не обходят блокер отсутствующего доступа.

### Передача следующему агенту

Записать изменённые контракты, команды и результаты, ограничения, незакрытые вопросы и разблокированные зависимости. Обновить обе языковые версии и трассировку. Закрывать задачу только по доказательству её результата; GitHub Closed само по себе не означает Ready MVP.

**Commit boundary:** логическая граница этой задачи; commit/push/deploy не разрешены данной карточкой и требуют действующей авторизации пользователя.

## EN

Produce consistent crypto-product results without double accruals.

**Status:** Blocked by dependencies and the SDD Ready gate; implementation has not started.

**Dependencies:** `task-4.4`, `task-4.6`, `task-6.1`, `task-2.4`.

**Kind:** `implementation`.

### Change and contracts

Use domain contracts to separate gross/net realized P&L, unrealized P&L, fees, funding and rewards. Account for shared IDs across logs, transfers among Funding/Spot/Earn/Coinhold/mining and compounding; unrealized P&L is not cash income. Do not add a trading terminal or mining control.

### Change boundaries

- `backend/internal/trading/`
- `backend/internal/mining/`

These are planned paths. Shared contracts: `spec/001-want-keep-mvp/contracts.en.md`; architecture and commands: `constraints.en.md`. Change only the behavior owner and affected tests; an unresolved contract requires updated evidence and stops dependent implementation.

### Linked requirements

- **REQ-007:** Exchange and P2P conversion of owned money preserve both currency amounts, the actual rate and fees.
- **REQ-035:** Trading analytics separates realized P&L, unrealized P&L, fees and funding.
- **REQ-036:** Mining rewards are separate from transfers between owned wallets.
- **REQ-045:** The Bybit integration automatically reads Funding, Spot, Earn, P2P and futures under a verified contract.
- **REQ-047:** The EMCD integration automatically reads wallet, Coinhold, P2P, crypto card and mining under a verified contract.

### Acceptance criteria

A criterion link establishes coverage; research or a partial task does not prove the entire product criterion. This task's exact outcome is specified in verification below.

#### AC-007

- **Given:** Owned RUB 9,000 is exchanged for USDT 100 with a RUB 50 fee.
- **When:** The bank and crypto legs of a confirmed exchange arrive.
- **Then:** Principal is not income/expense; the RUB 90/USDT rate is separate from the fee; an ambiguous match requires clarification.
- **Level:** `integration`.

#### AC-035

- **Given:** A source reports USDT 10 realized P&L, USDT 1 fee and USDT 5 unrealized P&L.
- **When:** The trading account report updates.
- **Then:** Metrics are separate; unrealized USDT 5 is not received income; net results do not suffer a second deduction for already-included fees.
- **Level:** `integration`.

#### AC-036

- **Given:** A BTC 0.0001 reward is credited to mining and transferred to the main account.
- **When:** Both records are imported.
- **Then:** Income is recognized once on confirmed credit; the subsequent transfer creates no additional income.
- **Level:** `integration`.

#### AC-045

- **Given:** An authorized personal Bybit account with the tested products is connected.
- **When:** Accounts, balances, transactions and required product terms are requested.
- **Then:** Every mandatory product has source-matching data and read evidence; inaccessible products are blockers, not successful coverage.
- **Level:** `contract+manual`.

#### AC-047

- **Given:** An authorized personal EMCD account with the tested products is connected.
- **When:** Accounts, balances, transactions and required product terms are requested.
- **Then:** Every mandatory product has source-matching data and read evidence; inaccessible products are blockers, not successful coverage.
- **Level:** `contract+manual`.

#### AC-071

- **Given:** A source distinguishes gross P&L, net P&L, fee, funding and reward/transfer.
- **When:** One economic event appears in several logs.
- **Then:** Metric provenance is preserved; fees and income are not doubled; net/gross semantics are contract-verified.
- **Level:** `contract+integration`.

### Verification

```sh
make test-integration AREA=crypto-results
```

Contract cases for gross/net, funding, mining→wallet and compounding produce one economic effect.

The `make` commands are a future contract established by task-1.1; they do not exist yet. Live/paid/manual checks separately record access and actual outcomes. Research does not bypass missing-access blockers.

### Handoff to the next agent

Record changed contracts, commands/results, limitations, unresolved questions and unblocked dependencies. Update both languages and traceability. Close the task only with evidence of its outcome; GitHub Closed alone does not mean the MVP is Ready.

**Commit boundary:** this task's logical boundary; this card does not authorize commit/push/deploy, which require current user authorization.
