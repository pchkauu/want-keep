# Want Keep

**One place for all your money.**

Want Keep is an early **source-available** family finance project for multi-currency accounting, budgets, savings and AI-assisted analysis.

The agreed MVP targets Go, PostgreSQL, React/TypeScript/Vite and a separate Playwright collector. The repository contains the specification, backlog, a buildable foundation, exact money primitives and generated API contracts. Product handlers and persistence are not implemented yet; external integration contracts remain under investigation.

## Project foundation

Go 1.26.5 and Node.js 24.19.0 are pinned. Install the locked dependencies and run every currently available check from the repository root:

```sh
make bootstrap
make check
```

The web build contains an explicit non-product shell. Commands for integrations, E2E, AI evaluations, deployment and backup/restore reject missing suites until their owning tasks implement them.

## MVP specification

- [Русская документация](spec/001-want-keep-mvp/README.md) / [English documentation](spec/001-want-keep-mvp/README.en.md)
- [Desktop design and animations](spec/001-want-keep-mvp/design.en.md) / [35 screens and navigation](spec/001-want-keep-mvp/screens.en.md)
- [Requirements and acceptance](spec/001-want-keep-mvp/traceability.en.md)
- [Implementation backlog and GitHub Issues](spec/001-want-keep-mvp/backlog.en.md)
- [Readiness and blockers](spec/001-want-keep-mvp/verification.en.md)

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) and the [Code of Conduct](CODE_OF_CONDUCT.md).

## Security

Report vulnerabilities privately. Do not file public GitHub issues for security problems.

See [SECURITY.md](SECURITY.md).

## License

Copyright 2026 Want Keep authors.

```text
Required Notice: Copyright 2026 Want Keep authors
```

Want Keep is **source-available** software licensed under the [PolyForm Shield License 1.0.0](LICENSE).

The license is intended to allow viewing, forking, modifying, self-hosting, personal use, and contributions. Building a competing product or service with Want Keep is restricted by the license terms.

This is not an OSI open-source license. Read [LICENSE](LICENSE) for the full legal text.
