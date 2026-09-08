# Потоки данных

[English](flows.en.md)

## Импорт и AI-проверка

```mermaid
flowchart LR
  P[Платформа] --> G[API gateway / Playwright]
  G --> R[SourceRecord и coverage]
  R --> M[Mapping и domain validation]
  M --> L[Атомарный ledger + outbox]
  L --> A[AI review версии]
  A --> V[Проверка разрешённой команды]
  V --> C[Категория / подтверждённая связь]
  V --> Q[Уточнение]
  C --> D[Проекции и дашборд]
  L --> D
  P --> S[BalanceSnapshot]
  S --> X[Сверка]
  L --> X
```

SourceRecord сохраняется и при неполном mapping. Ledger может работать, пока AI ждёт; неустановленные финансовые значения не проводятся. У дашборда отдельные признаки свежести источника, покрытия и AI-review.

## Чек и поздний импорт

```mermaid
sequenceDiagram
  actor O as Участник
  participant W as Веб-чат
  participant A as Application
  participant AI as OpenAI
  participant L as Ledger
  O->>W: Выбрать счёт и отправить чек
  W->>A: Message + attachmentId + accountId + idempotency
  A->>A: Авторизация и проверка файла
  A->>AI: Разрешённые данные без секретов
  AI-->>A: Структурированное предложение
  alt Не хватает точных данных
    A-->>W: Уточнение конкретных полей
  else Документ не относится к учёту
    A-->>W: Явный пропуск и причина
  else Данные установлены
    A->>L: Создать или связать операцию, expectedRevision
    A-->>W: Результат и ссылка на операцию
  end
  A->>L: Поздний банковский импорт
  L->>L: Точная идемпотентность + проверка экономической связи
  L-->>W: Одна операция, несколько доказательств
```

Равные сумма и время формируют кандидатов, не доказательство единственности. Если найдено несколько покупок, требуется уточнение. Изменение операции во время ответа AI делает старое предложение неприменимым.

## План и дневной лимит

```mermaid
flowchart TD
  F[Подтверждённые деньги по валюте] --> K[Available limit]
  R[Уникальные резервы целей] --> K
  O[Непогашенные обязательства] --> K
  B[Остаток гибкого бюджета] --> K
  E[Доходы по ожидаемым датам] --> Q[Дневной forecast и кассовые разрывы]
  F --> Q
  R --> Q
  O --> Q
  B --> Q
  AI[Предложение AI] --> U[Решение уполномоченного участника]
  U --> B
```

Получение дохода меняет факт; наступление плановой даты само по себе не создаёт деньги. Погашенный платёж снимает соответствующую резервируемую часть обязательства.

## Резервирование на Mac

```mermaid
sequenceDiagram
  participant M as MacBook
  participant S as VPS приложения
  participant P as Managed PostgreSQL
  participant F as Неизменяемые вложения
  participant B as Локальная копия
  M->>S: Авторизованный hourly pull при доступности
  S->>P: Logical dump по private VPC
  S->>F: Inventory по consistent cutoff
  P-->>S: Поток DB
  F-->>S: Вложения + manifest
  S-->>M: Ограниченный поток export
  M->>M: Проверить части и checksum
  M->>B: Атомарно завершить зашифрованный набор
  M->>S: Подтверждение проверенного manifest
  Note over M,P: Недоступный Mac увеличивает возраст последней полной копии
```

Последняя полная копия не заменяется незавершённой. Recovery-ключ должен быть доступен вне единственной аварийной системы. Retention: 48 почасовых, 30 дневных, 8 недельных и 12 месячных точек при cap 20 GiB; последний полный набор не удаляется. RPO до часа условен доступностью Mac и проверкой набора. Восстановление старого набора в чистый managed PostgreSQL не активирует старые банковские сессии автоматически. Подробный контракт и открытые runtime gates: [hosting evidence](evidence/hosting.md).

## Семейные права и разрезы

```mermaid
flowchart LR
  U[Отдельная сессия пользователя] --> M[Членство и scope семьи]
  M --> C[Команда и текущие права]
  C --> L[Журнал: сумма и автор]
  L --> A[Доли участников и категории]
  A --> F[Один семейный факт]
  A --> P[Индивидуальные разрезы]
  F --> K[Общий K по валюте]
  K --> D[Одна матрица лимитов]
  D --> P
```

## Task-2.4: от факта к одному эффекту

```mermaid
flowchart LR
  A[Ручной ввод / проверенная нормализация] --> B[Семейная транзакция и текущие права]
  I[CommitPage: admission / generation / lease] --> B
  B --> C[Идемпотентность D-39]
  C --> D{Доказанная связь?}
  D -->|Да| E[Проверить состав и защиту полей]
  D -->|Только вероятный кандидат| F[Сохранить waiting и evidence без второго эффекта]
  F --> G[Участник: link или separate с revisions]
  G --> E
  E --> H[Решение + journal revisions + проекции + review/outbox]
  H --> J[Результат команды / checkpoint]
  H --> K[История и составной undo]
```

Отдельные исходные статусы и даты остаются в истории. Неполный поиск и ожидание дают matching_unresolved; source observations сохраняются отдельно. Устаревший import job попадает в quarantine до этой цепочки. Банковский IO и распознавание документов подключаются в профильных задачах.

## Task-3.2: входная страница коннектора

```mermaid
sequenceDiagram
  participant J as Проверенное sync job
  participant C as API/Browser collector
  participant E as EvidenceStore
  participant G as Admission gate
  participant A as Accounts/Ledger
  participant Q as Quarantine
  J->>G: Проверить exact gateway binding
  J->>G: BeforeRead(binding, revision, generation, lease)
  G->>C: Server-issued request без household/actor/internal IDs
  C-->>G: Exact echo + evidence + typed records/coverage
  G->>E: Сохранить raw evidence до финансовой транзакции
  alt provider failure
    G->>Q: Связать evidence с household/job
    G->>G: Атомарно сохранить waiting/retry/failed
  else binding/revision/generation/lease/cursor актуальны
    G->>A: CommitPage: resolve accounts + source revisions + observations/postings
    A-->>G: Audit/outbox/checkpoint атомарно
    opt account/source ambiguity
      A->>Q: Evidence + source_ambiguous/transaction_unresolved
      A-->>G: Partial coverage без неподтверждённого эффекта
    end
  else результат устарел
    G->>Q: Evidence reference + safe reason
  end
```

Страница самостоятельна: каждый поддерживаемый счёт, используемый balance или posting, имеет account descriptor в той же странице. Следующая страница повторяет descriptor и исходный cursor; повтор не создаёт account/opening/financial effect. Ошибка второй страницы не продвигает её cursor, а уже подтверждённая первая страница остаётся зафиксированной с partial coverage. Неоднозначный счёт или source не превращает страницу в полный успех и не отменяет независимые поддержанные записи.
