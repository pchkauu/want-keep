# Want Keep MVP

[English](README.en.md)

**Статус SDD:** **Ready for development** с 2026-09-07. [Итог task-0.10](evidence/task-0.10-readiness.md) закрывает фундаментальные решения D-37–D-43; [plan.md](plan.md) задаёт порядок реализации.

Ready относится к спецификации. Приложение ещё не реализовано, обязательные продуктовые AC не пройдены, а каждый коннектор выключен до собственного provider/runtime gate.

## Порядок чтения

1. [proposal.md](proposal.md) — продукт, scope и решения D-01–D-43.
2. [requirements.md](requirements.md) и [acceptance_criteria.md](acceptance_criteria.md) — REQ/AC.
3. [contracts.md](contracts.md), [constraints.md](constraints.md), [flows.md](flows.md) — домен, API, безопасность и потоки.
4. [integrations.md](integrations.md) и [operations.md](operations.md) — provider и runtime gates.
5. [design.md](design.md), [navigation.md](navigation.md), [screens.md](screens.md) — desktop UX.
6. [plan.md](plan.md), [backlog.md](backlog.md), [traceability.md](traceability.md) — реализация и связи.
7. [verification.md](verification.md) — verdict и границы доказательств.

## Текущее исполнение

- task-0.1–task-0.10: исследования и Ready-gate завершены как документационные результаты.
- task-1.1: техническая основа реализована и влита в эту ветку документации.
- Следующая задача: task-1.2, затем task-1.3 и остальные задачи по зависимостям.
- GitHub Closed не заменяет evidence задачи и не означает прохождение связанного продуктового AC.

## Provider gates

D-38 разрешает разработку по нормализованным контрактам и safe states. По D-43 конкретный provider включается только server-owned admission для точного binding build/contract/allowlist/config/permission/environment: task-4.x подтверждает provider evidence, task-8.x — target-host/deployment evidence. Несовпадение до read возвращает `provider_not_admitted`; job/result несут `admissionRevision`, а commit-time revalidation оставляет in-flight stale result в quarantine без финансового эффекта.

При пробеле используются typed states `source_partial`, `source_ambiguous`, `valuation_unavailable`, `quote_unavailable` или `command_expired`. Неизвестное значение не становится нулём, а неоднозначный source record не создаёт проводку.

## Обновление документации

Источник REQ, AC, задач, экранов, форм и состояний — [catalog.json](catalog.json). После изменения:

```sh
python3 spec/001-want-keep-mvp/tools/spec_tool.py render
make docs-check
make check
git diff --check
```

Парные ручные документы RU/EN обновляются вместе. Публичные артефакты используют только синтетические примеры; credentials, response bodies, реквизиты и локальные пути не публикуются.
