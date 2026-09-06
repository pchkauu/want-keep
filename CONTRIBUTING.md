# Contributing to Want Keep

Thanks for helping improve Want Keep.

This repository is **source-available**. You may fork it, modify it, self-host it, use it for your own needs, and contribute changes back. Read [LICENSE](LICENSE) before you start.

## Current scope

Application code is not published yet. Most useful contributions today are documentation, design discussion, and process improvements.

When application code lands, this file will gain concrete run, test, and lint commands. Until then, do not invent a stack or claim that the project can be started locally.

## How to contribute

1. Fork the repository.
2. Create a branch.
3. Make focused changes.
4. Add or update tests when code exists.
5. Run the project validation commands when they exist.
6. Open a pull request.

Discuss large changes in an issue first.

## How to run the project

There is no application runtime in this repository yet.

For documentation changes, edit the relevant Markdown files and open a pull request.

## Branch naming

Use a short prefix and a descriptive slug:

| Prefix | Use |
| --- | --- |
| `feat/` | New behavior |
| `fix/` | Bug fix |
| `docs/` | Documentation only |
| `chore/` | Tooling, templates, or repo hygiene |

Examples: `docs/self-hosting`, `feat/alfabank-connector`, `fix/transfer-matching`.

## Commit expectations

Use [Conventional Commits](https://www.conventionalcommits.org/):

- `feat`
- `fix`
- `chore`
- `docs`
- `refactor`
- `test`

Keep commits focused. Explain why the change exists, not only what changed.

## Pull request process

Open a PR against `main` using the repository template.

A PR should:

- stay scoped to one purpose
- update documentation when behavior or contracts change
- include tests for new or changed behavior once code exists
- call out breaking changes explicitly
- contain no secrets and no real financial data

Maintainers may ask for smaller PRs if a change mixes unrelated work.

## Tests

When application code exists:

- add or update tests for business logic, mapping, validation, and error paths
- cover security-sensitive behavior with regression tests
- do not weaken tests to hide a failure

Until then, documentation PRs should be fact-checked against the repository. Do not describe unimplemented features as shipped.

## Code quality

Prefer small, readable changes that fit the existing architecture.

Once a stack is published:

- use the project formatter, linter, and test commands
- keep public APIs, schemas, and storage contracts explicit
- do not introduce dependencies without a clear need

Do not replace existing tooling with alternatives unless there is a concrete reason.

## Large changes

Open an issue before:

- adding a new integration
- changing the domain model
- introducing a new storage or sync strategy
- changing auth, encryption, or privacy behavior
- adding an AI or OCR provider

Describe the problem, the proposed approach, and the alternatives you considered.

## Adding integrations

Use the [integration request](.github/ISSUE_TEMPLATE/integration_request.yml) template first.

A connector should eventually document:

- authentication and refresh
- account and transaction mapping
- pagination
- rate limits
- sync strategy
- error handling
- idempotency
- supported currencies

Integrations belong behind the shared connector abstraction described in [docs/integrations.md](docs/integrations.md). Do not bind the domain model to a bank-specific DTO.

Do not add reverse-engineered or unofficial private APIs unless this repository already has a clear, lawful basis for that integration.

Never commit fake banking clients that pretend to call a live API.

## Security rules

Want Keep will handle bank credentials, tokens, receipts, and personal financial history.

Never add real API keys, bank credentials, access tokens, cookies, or personal financial data to the repository, fixtures, screenshots, or logs.

Also:

- do not log secrets or account identifiers in examples
- strip credentials from reproduced errors
- treat OCR images and AI prompts as sensitive input
- prefer least privilege for any future connector credential

Report vulnerabilities privately. See [SECURITY.md](SECURITY.md). Do not file public issues for security problems.

## Code of Conduct

Participation is governed by [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
