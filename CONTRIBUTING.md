# Contributing to Want Keep

Thanks for helping improve Want Keep.

This repository is **source-available**. You may fork it, modify it, self-host it, use it for your own needs, and contribute changes back. Read [LICENSE](LICENSE) before you start.

## Current scope

Application code is not published yet. Useful contributions today are documentation and repository process.

The agreed target stack and task dependencies are documented in the [MVP specification](spec/001-want-keep-mvp/README.en.md). When application code lands, this file will gain concrete runtime, test, and lint commands. Do not present planned commands as available or claim that the application can be started locally.

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

For hand-authored specification changes, update both RU/EN Markdown versions. Requirements, acceptance criteria, screens/forms/states and task cards are generated from `spec/001-want-keep-mvp/catalog.json`; edit that source and run these existing documentation commands from the repository root:

```sh
python3 spec/001-want-keep-mvp/tools/spec_tool.py render
python3 spec/001-want-keep-mvp/tools/spec_tool.py check
git diff --check
```

Review semantic parity separately: the script checks translation presence and traceability, not whether the prose means the same thing. See the package README for the SDD readiness gate and future application work.

## Branch naming

Use a short prefix and a descriptive slug:

| Prefix | Use |
| --- | --- |
| `feat/` | New behavior |
| `fix/` | Bug fix |
| `docs/` | Documentation only |
| `chore/` | Tooling, templates, or repo hygiene |

Examples: `docs/readme`, `chore/gitignore`.

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
- contain no secrets and no personal data

Maintainers may ask for smaller PRs if a change mixes unrelated work.

## Tests

When application code exists:

- add or update tests for changed behavior
- do not weaken tests to hide a failure

Until then, documentation PRs should match what is actually in the repository.

## Code quality

Prefer small, readable changes.

Once a stack is published:

- use the project formatter, linter, and test commands
- do not introduce dependencies without a clear need

Do not replace existing tooling with alternatives unless there is a concrete reason.

## Security rules

Never add real API keys, credentials, access tokens, cookies, or personal data to the repository, fixtures, screenshots, or logs.

Also:

- do not log secrets in examples
- strip credentials from reproduced errors

Report vulnerabilities privately. See [SECURITY.md](SECURITY.md). Do not file public issues for security problems.

## Code of Conduct

Participation is governed by [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
