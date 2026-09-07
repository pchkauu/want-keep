# Evidence task-1.1 — основа приложения

[English](task-1.1-foundation.en.md)

Проверено 2026-09-07. Ветка содержит только независимую техническую основу; результаты параллельных task-0.1–task-0.10 не включены.

## Зафиксированные контракты

- Go 1.26.5; Node.js 24.19.0 LTS.
- Отдельные npm-пакеты и lockfiles для web и collector.
- React/Vite build-shell явно сообщает, что продуктовые сценарии отсутствуют.
- Collector загружает Playwright API в синтетическом тесте без запуска браузера и внешних аккаунтов.
- OpenAPI source/config/generator/output должны отсутствовать или изменяться полным набором.
- Development/test/production examples не содержат секретов.
- Архитектурный Go-тест запрещает зависимость domain/application от delivery/storage/integrations/gateways и инфраструктуры от delivery.

## Проверки

Локально выполнены:

```sh
make bootstrap
make check
make test-go PKG=./internal/architecture/...
make test-web FILTER=App
make test-collector FILTER=foundation
```

Все команды завершились успешно. `make check` подтвердил форматирование, lint/typecheck, Go/web/collector tests, обе сборки, 11 тестов spec tool, 87 REQ, 105 AC, 67 задач и 35 экранов. `shadcn info` подтвердил Vite, Tailwind v4 и Base UI (`base-nova`). npm audit при clean install сообщил 0 известных уязвимостей.

Локальный Go автоматически использовал закреплённый toolchain 1.26.5. Установленный на MacBook Node.js 24.2.0 старее закреплённого 24.19.0, поэтому clean bootstrap завершился успешно с `EBADENGINE` warning; точную Node-версию проверяет CI.

Future-suite targets `test-integration`, `test-contract`, `e2e`, `eval-ai`, `check-deploy`, `backup-check`, `restore-check` и `generate-contracts` проверены отрицательными сценариями: каждый завершился кодом 2 и назвал незакрытую область.

## Ограничения

Это не Ready приложения. Нет публичного API, PostgreSQL-схемы, финансовой логики, browser E2E, реальных интеграций, deploy-конфигурации и backup/restore runtime. task-0.10 остаётся обязательным барьером для продуктовой реализации.
