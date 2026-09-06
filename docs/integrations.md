# Integrations

This document describes the **intended** connector layer.

No bank, exchange, or wallet integration is implemented yet. Do not treat the names below as supported services.

## Purpose

Connectors pull accounts, balances, and transactions from external services, then hand raw payloads to the normalization layer.

The rest of Want Keep should depend on this abstraction, not on a provider SDK.

## Connector contract

Each connector should eventually support the operations that apply to its service:

```text
authenticate
refreshAuthentication
fetchAccounts
fetchBalances
fetchTransactions
sync
handleWebhook
```

Not every service will have webhooks. `sync` is the common entry point for polling.

A connector should also expose enough metadata for operators and the UI: display name, service type, auth model, supported currencies, and whether webhooks are available.

## What a connector must make explicit

| Topic | Why it matters |
| --- | --- |
| Auth model | OAuth, API key, username/password, or session. Refresh and revocation paths must be documented. |
| Account mapping | How provider accounts become Want Keep accounts. |
| Transaction mapping | How provider records become raw transactions, then normalized entities. |
| Pagination | Cursor, page, or date-window behavior. |
| Rate limits | How the connector backs off and resumes. |
| Sync strategy | Full history, incremental window, webhook-triggered, or a mix. |
| Error handling | Auth failure, provider outage, partial pages, and retry policy. |
| Idempotency | Re-running sync must not create duplicate ledger entries. |
| Supported currencies | What the provider returns and how unknown currencies are handled. |

Raw provider payloads should be stored or referenced separately from normalized transactions. See [data-model.md](data-model.md).

## Mapping rules

- Map at the boundary. Do not leak provider DTO types into the domain.
- Preserve the provider's external id when one exists.
- Keep the original amount and currency.
- Do not decide income vs expense from a transfer between the user's own accounts.
- Prefer provider timestamps over local ingest time for the transaction date.

## Sync and idempotency

`sync` should be safe to run again.

Identity should prefer a stable external id. When a provider has none, matching may use a conservative fingerprint from account, amount, currency, timestamp, and description. Ambiguous matches must not silently merge.

Webhooks, when available, should enqueue work for the same sync path rather than writing ledger entries through a second code path.

## Planned targets

These are product targets, not implemented connectors:

- Alfa Bank
- Raiffeisen
- Bybit
- Aifory Pro
- EMCD
- Ozon

Other banks, exchanges, wallets, and fintech services may be added later.

## Adding a connector

1. Open an [integration request](../.github/ISSUE_TEMPLATE/integration_request.yml).
2. Confirm official API documentation or another lawful basis.
3. Describe auth, mapping, pagination, rate limits, and sync.
4. Keep credentials out of the repository.

Do not add reverse-engineered or unofficial private APIs unless this repository already has a clear, lawful basis for that work.

Do not commit fake clients that pretend to call a live banking API.

## Security

Connector credentials, cookies, and tokens are secrets. Never add them to the repository, fixtures, screenshots, or logs.

See [CONTRIBUTING.md](../CONTRIBUTING.md) and [../SECURITY.md](../SECURITY.md).
