# Roadmap

This is a planned product surface, not a delivery schedule.

Nothing here is marked done. Dates will be added only when work is scheduled.

## Personal finance core

- Accounts, transactions, categories
- Income, expenses, balances, and history
- Budgets
- Analytics over the normalized ledger

## External integrations

- Shared connector abstraction
- First connectors for planned targets such as Alfa Bank, Raiffeisen, Bybit, Aifory Pro, EMCD, and Ozon
- Incremental sync, retries, and idempotent imports

See [integrations.md](integrations.md).

## Receipts

- Image upload and preprocessing
- OCR and structured extraction
- Confidence checks and user review
- Matching receipts to transactions

## Internal transfers

- Detect movements between a user's own accounts
- Collapse matched legs into a transfer
- Keep transfers out of income and expense totals

## Multi-currency

- Original amount and currency on every transaction
- Dated exchange rates
- Conversion into the user's base currency

## Analysis

- Balances, income, expenses, and cash flow
- Spending patterns and categories
- Recurring payments
- Changes in financial behavior

## Daily summary

- Daily budget
- Spent so far
- Remaining amount for today
- Forecast to the end of the month

## AI insights

- Anomaly hints
- Explanations of spending changes
- Recurring-payment suggestions
- Recommendations
- Natural-language summaries

AI remains an enhancement layer. Ledger math stays deterministic. See [architecture.md](architecture.md).

## Self-hosting

- Documented deployment
- Example environment configuration
- Backup guidance
- Optional local AI and OCR paths

See [self-hosting.md](self-hosting.md).
