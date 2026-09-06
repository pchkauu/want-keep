# Security and privacy

Want Keep will handle highly sensitive financial data. This page records the **intended** privacy posture. It does not claim that encryption, deletion, or provider isolation is already implemented.

The design goal is:

> local-first / privacy-first / self-hosting-friendly

This repository does **not** claim end-to-end encryption, zero-knowledge architecture, or compliance certifications.

Report vulnerabilities privately. See [../SECURITY.md](../SECURITY.md).

## Data we expect to store

When the application exists, a self-hosted instance will likely persist:

- user profile and preferences
- institution connections and sync metadata
- accounts, balances, and transactions
- categories, merchants, budgets, and snapshots
- receipt images and OCR results
- exchange rates used for historical conversion
- generated insights and user corrections

Authentication secrets for banks and other providers must not live in this git repository. They should live in the operator's secret store or environment, and they should never appear in logs.

## AI providers

AI is optional enhancement, not the ledger.

If a remote model is configured, a request may include transaction descriptions, merchant names, categories, and aggregated totals needed for a summary. It should not need raw credentials, full account numbers, or complete export dumps.

Deterministic money math must not depend on the model response.

Users should be able to run without a remote AI provider. Local models are a planned option, not a current feature.

## OCR providers

If a remote OCR provider is configured, uploaded receipt images and extracted text may leave the instance.

OCR output is untrusted. Low-confidence results need a review path before they become confirmed transactions.

Users should be able to use a local OCR path when one exists.

## Local models

The intended self-hosted setup should allow:

- no AI provider
- a local model endpoint
- a remote provider chosen by the operator

Remote providers are never the system of record for balances or amounts.

## Deletion

Users should be able to delete:

- a connection and its stored credentials
- imported transactions from that connection
- receipt images and OCR artifacts
- generated insights
- the entire account on a self-hosted instance

Deletion behavior will be specified with the first storage implementation. Until then, do not claim guaranteed wipe semantics.

## Backups

Backup is an operator concern for self-hosted deployments.

Backups will contain financial history and possibly connector secrets. Treat them as sensitive as the live database. This repository does not yet define a backup format.

## Encryption considerations

Likely needs, once code exists:

- TLS in transit
- encryption at rest for the operator's database and object store
- careful handling of connector tokens
- no secrets in git, fixtures, or CI logs

At-rest encryption provided by the host is not the same as end-to-end encryption. Do not describe Want Keep as E2E-encrypted unless that design is implemented and reviewed.

## Security-sensitive surfaces

- banking integrations
- authentication
- OAuth
- API tokens
- encryption
- financial data
- transaction history
- personal data
- OCR input
- AI providers
- webhooks
- import/export
- backups
