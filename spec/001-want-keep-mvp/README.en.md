# Want Keep MVP

[Русский](README.md)

**SDD status:** **Ready for development** since 2026-09-07. The [task-0.10 outcome](evidence/task-0.10-readiness.en.md) resolves fundamental D-37–D-43 decisions; [plan.en.md](plan.en.md) defines implementation order.

Ready applies to the specification. The application is not implemented, mandatory product ACs have not passed, and every connector stays disabled until its own provider/runtime gate passes.

## Reading order

1. [proposal.en.md](proposal.en.md) — product, scope and D-01–D-43 decisions.
2. [requirements.en.md](requirements.en.md) and [acceptance_criteria.en.md](acceptance_criteria.en.md) — REQ/AC.
3. [contracts.en.md](contracts.en.md), [constraints.en.md](constraints.en.md), [flows.en.md](flows.en.md) — domain, API, security and flows.
4. [integrations.en.md](integrations.en.md) and [operations.en.md](operations.en.md) — provider and runtime gates.
5. [design.en.md](design.en.md), [navigation.en.md](navigation.en.md), [screens.en.md](screens.en.md) — desktop UX.
6. [plan.en.md](plan.en.md), [backlog.en.md](backlog.en.md), [traceability.en.md](traceability.en.md) — implementation and links.
7. [verification.en.md](verification.en.md) — verdict and evidence boundaries.

## Current execution

- task-0.1–task-0.10: research and the Ready gate are complete as documentation outcomes.
- task-1.1: the technical foundation is implemented and merged into this documentation branch.
- Next task: task-1.2, followed by task-1.3 and the remaining dependency-ordered work.
- GitHub Closed never substitutes for task evidence or passing a linked product AC.

## Provider gates

D-38 permits development against normalized contracts and safe states. Under D-43, a provider is enabled only by server-owned admission for the exact build/contract/allowlist/configuration/permission/environment binding: task-4.x proves provider evidence and task-8.x proves target-host/deployment evidence. A mismatch returns `provider_not_admitted` before collector IO.

A gap uses typed `source_partial`, `source_ambiguous`, `valuation_unavailable`, `quote_unavailable` or `command_expired` states. An unknown value never becomes zero, and an ambiguous source record creates no posting.

## Updating documentation

[catalog.json](catalog.json) is the source for REQ, AC, tasks, screens, forms and states. After a change:

```sh
python3 spec/001-want-keep-mvp/tools/spec_tool.py render
make docs-check
make check
git diff --check
```

Update paired hand-authored RU/EN documents together. Public artifacts use synthetic examples only; never publish credentials, response bodies, account details or local paths.
