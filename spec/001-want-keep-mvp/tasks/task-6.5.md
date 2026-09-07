<!-- want-keep-task: task-6.5 -->
# task-6.5 — Сводить торговый результат и майнинг / Aggregate trading P&L and mining

## RU

Получать согласованный результат по криптопродуктам без двойных начислений.

**Состояние:** Не начато; задача ожидает собственные зависимости и entry gates.

**Зависимости:** `task-6.1`, `task-2.4`.

**Тип:** `implementation`.

### Изменение и контракты

Через доменные контракты разделить gross/net realized P&L, unrealized P&L, fees, funding и rewards. Учитывать общие IDs нескольких журналов, transfers Funding/Spot/Earn/mining и капитализацию; нереализованный результат не считать наличным доходом. Не добавлять торговый терминал или управление майнингом. EMCD по D-34 не является источником торговли/майнинга и не блокирует эту задачу; его Grow обрабатывается в накоплениях. Bybit по D-36 также не является источником торговли/майнинга для текущего MVP и не блокирует эту задачу. Общие функции остаются; проверяются синтетическими доменными сценариями, а новое платформенное покрытие требует отдельного решения и контрактов. Funding и Easy Earn обслуживают task-4.4 и накопления.

### Границы изменений

- `backend/internal/trading/`
- `backend/internal/mining/`

Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-007:** Обмен и P2P-конвертация собственных денег сохраняют обе валютные суммы, фактический курс и комиссии.
- **REQ-035:** Торговая аналитика отделяет реализованный результат, нереализованный результат, комиссии и funding.
- **REQ-036:** Вознаграждения майнинга отделены от переводов между собственными кошельками.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

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

Команды make созданы основой task-1.1; финансовые provider/integration/E2E suites ещё не реализованы. Для документации используется make docs-check. Live/paid/manual проверки отдельно фиксируют доступ и фактический результат; наличие команды или UI-доступа не доказывает runtime.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Produce consistent crypto-product results without double accruals.

**Status:** Not started; the task awaits its own dependencies and entry gates.

**Dependencies:** `task-6.1`, `task-2.4`.

**Kind:** `implementation`.

### Change and contracts

Use domain contracts to separate gross/net realized P&L, unrealized P&L, fees, funding and rewards. Account for shared IDs across logs, transfers among Funding/Spot/Earn/mining and compounding; unrealized P&L is not cash income. Do not add a trading terminal or mining control. EMCD under D-34 is not a trading/mining source and does not block this task; its Grow is handled by savings. Bybit under D-36 is also not a trading/mining source for the current MVP and does not block this task. Shared functions remain and use synthetic domain scenarios; new platform coverage requires a separate decision and contracts. Funding and Easy Earn belong to task-4.4 and savings.

### Change boundaries

- `backend/internal/trading/`
- `backend/internal/mining/`

Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-007:** Exchange and P2P conversion of owned money preserve both currency amounts, the actual rate and fees.
- **REQ-035:** Trading analytics separates realized P&L, unrealized P&L, fees and funding.
- **REQ-036:** Mining rewards are separate from transfers between owned wallets.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

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

The task-1.1 foundation provides make commands; financial provider/integration/E2E suites are not implemented yet. Use make docs-check for documentation. Live/paid/manual checks separately record access and outcomes; an existing command or UI access is not runtime proof.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
