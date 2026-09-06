# Want Keep

**One place for all your money.**

> **Want Keep — your unified financial hub.**
>
> Connect banks, exchanges and wallets, scan receipts, automatically organize transactions, and turn raw financial data into a single picture of your money.

Want Keep is an AI-powered personal finance application. The goal is to bring accounts, transactions, receipts, and insights from many sources into one place you control.

This repository is **source-available** under the [PolyForm Shield License 1.0.0](LICENSE). You may view the source, fork it, modify it, self-host it, use it for your own needs, and contribute back. Using Want Keep to build a competing product or service is restricted by the license.

## Status

Want Keep is an **early public repository**.

The repository currently contains project documentation, contribution rules, and the intended architecture. Application code is not published yet. Nothing below is claimed as implemented.

See [docs/roadmap.md](docs/roadmap.md) for the planned product surface.

## Features

All items are **planned**. Status will be updated when code lands.

| Area | Intent |
| --- | --- |
| Personal finance core | Accounts, transactions, categories, income, expenses, budgets, balances, history, analytics |
| External sync | Pull transactions and balances from banks, exchanges, wallets, and fintech services |
| Receipt OCR | Extract amount, date, merchant, line items, currency, and category from photos and screenshots |
| Internal transfers | Detect transfers between a user's own accounts and exclude them from income and expenses |
| Multi-currency | Keep original currency, fetch exchange rates, convert to a user base currency with historical rates |
| Analysis | Balances, cash flow, spending patterns, categories, recurring payments, behavior changes |
| Daily summary | Daily budget, spent so far, remaining today, forecast to month end |
| AI insights | Anomalies, explanations, recurring-payment detection, recommendations, natural-language summaries |

## How it works

The intended pipeline is:

```text
Banks / Exchanges / Wallets / Receipts
                  ↓
             Connectors
                  ↓
          Raw Transactions
                  ↓
        Normalization Layer
                  ↓
       Deduplication / Matching
                  ↓
       Internal Transfer Detection
                  ↓
          Categorization
                  ↓
      Currency Normalization
                  ↓
          Financial Ledger
                  ↓
      Analytics / AI Insights
                  ↓
         API / Applications
```

Raw external payloads stay separate from normalized financial entities. Domain models must not depend on a bank-specific DTO.

Details: [docs/architecture.md](docs/architecture.md).

## Integrations

No bank, exchange, or wallet integration is implemented yet.

Planned first targets include:

- Alfa Bank
- Raiffeisen
- Bybit
- Aifory Pro
- EMCD
- Ozon

Other banks, exchanges, wallets, and fintech services may follow.

Connectors will sit behind a shared abstraction. Unofficial or private APIs will not be added without a clear, lawful basis in this repository.

Details: [docs/integrations.md](docs/integrations.md).

## AI & OCR

AI is an enhancement layer, not the source of truth.

Deterministic financial calculations must not depend on LLM output. Balances, amounts, exchange calculations, accounting logic, and daily budget math will be computed in code.

OCR is planned as:

```text
Image → Preprocessing → OCR → Structured Extraction
  → Merchant / Date / Total / Items
  → Confidence Validation → Transaction Matching
```

Low-confidence OCR must not create a confirmed transaction without a user review path.

## Self-hosting

Self-hosting is a core goal. Users should be able to run their own Want Keep.

There is no application runtime to deploy yet. When one exists, setup docs will live in [docs/self-hosting.md](docs/self-hosting.md). A Compose-based path will be added only if it matches the actual stack.

## Architecture

Intended design, not an implemented system:

- [Architecture](docs/architecture.md)
- [Integrations](docs/integrations.md)
- [Data model](docs/data-model.md)
- [Security and privacy](docs/security.md)
- [Self-hosting](docs/self-hosting.md)
- [Roadmap](docs/roadmap.md)

## Getting started

There is no application to install yet.

1. Read this README and [docs/architecture.md](docs/architecture.md).
2. Review [CONTRIBUTING.md](CONTRIBUTING.md) before opening a pull request.
3. Use the issue templates for bugs, features, or new integrations.

Runtime install steps will be added when application code is published.

## Development

The application stack is not published yet, so this repository does not ship formatter, linter, test, or CI commands.

Current expectations:

- Keep documentation accurate and concise.
- Do not claim unimplemented behavior.
- Never commit secrets or personal financial data.

When code lands, this section will list the real project commands. See [CONTRIBUTING.md](CONTRIBUTING.md).

## Contributing

Contributions are welcome.

Preferred flow:

1. Fork
2. Create a branch
3. Make changes
4. Add or update tests (once code exists)
5. Run validation (once commands exist)
6. Open a pull request

Read [CONTRIBUTING.md](CONTRIBUTING.md) and the [Code of Conduct](CODE_OF_CONDUCT.md).

Never add real API keys, bank credentials, access tokens, cookies, or personal financial data to the repository, fixtures, screenshots, or logs.

## Security

Want Keep will handle highly sensitive financial data. Please report vulnerabilities privately.

Do not file public GitHub issues for security problems.

See [SECURITY.md](SECURITY.md) and [docs/security.md](docs/security.md).

## Privacy

The intended posture is **local-first**, **privacy-first**, and **self-hosting-friendly** where that is reasonable.

This repository does not claim end-to-end encryption, zero-knowledge architecture, or compliance certifications.

See [docs/security.md](docs/security.md) for what we expect to store, what may be sent to AI or OCR providers, and how deletion and backups should work.

## Roadmap

Planned work is listed in [docs/roadmap.md](docs/roadmap.md). Dates and implementation status will be added only when they are real.

## License

Copyright 2026 Want Keep authors.

```text
Required Notice: Copyright 2026 Want Keep authors
```

Want Keep is **source-available** software licensed under the [PolyForm Shield License 1.0.0](LICENSE).

The license is intended to allow viewing, forking, modifying, self-hosting, personal use, and contributions. Building a competing product or service with Want Keep is restricted by the license terms.

This is not an OSI open-source license. Read [LICENSE](LICENSE) for the full legal text.
