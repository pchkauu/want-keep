# Self-hosting

Self-hosting is a core goal of Want Keep.

A user should be able to run their own instance, keep financial data on infrastructure they control, and choose whether AI or OCR providers are local or remote.

## Current status

There is **no application runtime** in this repository yet.

Do not expect `docker compose up -d` to work. A Compose file will be added only when it can start the real application.

`.env.example` will be added with the first runtime configuration, using comments and safe development defaults. It will not contain real secrets.

## Intended direction

When application code exists, this document should cover:

- supported deployment shape
- required services
- environment variables
- first-user setup
- backup and restore
- upgrading
- exposing the app safely on a private network

If the stack fits containers, the preferred developer and operator path will be Compose. That is a future decision, not a current command.

## Privacy expectations

A self-hosted instance should be able to run without sending data to Want Keep authors.

Optional outbound calls, once implemented, may include:

- bank, exchange, or wallet APIs the user connects
- an exchange-rate provider
- an OCR provider, if configured
- an AI provider, if configured

See [security.md](security.md).

## What not to do yet

- Do not invent a Compose stack for services that do not exist.
- Do not commit real API keys or bank credentials.
- Do not assume a hosted Want Keep cloud exists.
