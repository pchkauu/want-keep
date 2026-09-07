# task-1.1 evidence — application foundation

[Русский](task-1.1-foundation.md)

Verified on 2026-09-07. The branch contains only the independent technical foundation; parallel task-0.1–task-0.10 results are not included.

## Established contracts

- Go 1.26.5; Node.js 24.19.0 LTS.
- Separate npm packages and lockfiles for web and collector.
- The React/Vite build shell explicitly states that product workflows are unavailable.
- A synthetic collector test loads the Playwright API without starting a browser or using external accounts.
- OpenAPI source/config/generator/output must be absent or change as a complete set.
- Development/test/production examples contain no secrets.
- A Go architecture test prevents domain/application dependencies on delivery/storage/integrations/gateways and infrastructure dependencies on delivery.

## Verification

Executed locally:

```sh
make bootstrap
make check
make test-go PKG=./internal/architecture/...
make test-web FILTER=App
make test-collector FILTER=foundation
```

All commands completed successfully. `make check` verified formatting, lint/typecheck, Go/web/collector tests, both builds, 11 spec-tool tests, 87 REQs, 105 ACs, 67 tasks and 35 screens. `shadcn info` confirmed Vite, Tailwind v4 and Base UI (`base-nova`). npm audit during clean install reported 0 known vulnerabilities.

Local Go automatically selected the pinned 1.26.5 toolchain. The MacBook has Node.js 24.2.0, older than the pinned 24.19.0, so clean bootstrap completed with an `EBADENGINE` warning; CI verifies the exact Node version.

Future-suite targets `test-integration`, `test-contract`, `e2e`, `eval-ai`, `check-deploy`, `backup-check`, `restore-check` and `generate-contracts` were exercised with negative scenarios: each exited with code 2 and identified the unavailable area.

## Limitations

This is not application Ready evidence. There is no public API, PostgreSQL schema, financial logic, browser E2E, live integration, deployment configuration or backup/restore runtime. task-0.10 remains the mandatory gate for product implementation.
