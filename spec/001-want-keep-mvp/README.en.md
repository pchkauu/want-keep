# Want Keep MVP specification

[Русский](README.md)

This package records the agreed MVP and full backlog for later execution by an AI agent. The application is not implemented. All 15 initial capabilities and all six integrations are mandatory; partial feature delivery does not make the full MVP complete.

**Status:** requirements and decomposition prepared; implementation readiness is **Not Ready** until external contracts are verified. See [verification.en.md](verification.en.md).

## Reading order

1. [Product and interview decisions](proposal.en.md).
2. [87 requirements](requirements.en.md), [105 acceptance criteria](acceptance_criteria.en.md), [traceability](traceability.en.md).
3. [Architecture](constraints.en.md), [contracts and calculations](contracts.en.md), [glossary](glossary.en.md), [data flows](flows.en.md).
4. [Integrations and sources](integrations.en.md), [operations and costs](operations.en.md).
5. [67 tasks and GitHub links](backlog.en.md).

## For the next agent

Start with task-0.1–task-0.9 and record research evidence. Without securely supplied access, do not claim live verification. task-0.10 resolves contracts, updates RU/EN and reviews SDD readiness. Application implementation is blocked until then; `plan.md` intentionally does not exist. Then follow dependencies and the specific card's criteria rather than treating its title as a sufficient assignment.

Each card in `tasks/` contains the complete RU/EN body for one GitHub Issue. IDs remain stable. Closing an Issue is not evidence. Research may finish with a documented blocker; this does not make the dependent integration ready.

## Updating documentation

[catalog.json](catalog.json) is the editable source of requirements, criteria and task cards. Requirements, acceptance_criteria, backlog, traceability and tasks are generated from it. Other documents are maintained as manual RU/EN pairs.

From the repository root:

```sh
python3 spec/001-want-keep-mvp/tools/spec_tool.py render
python3 spec/001-want-keep-mvp/tools/spec_tool.py check
python3 -m unittest discover -s spec/001-want-keep-mvp/tools -p 'test_*.py'
git diff --check
```

Validation checks links, identifiers, coverage, cycles, both language versions, generated consistency and GitHub mappings. Translation meaning requires human/independent review; a passing script alone does not prove semantic parity.

Public examples are synthetic. API keys, real statements, receipt files, bank sessions and personal data are excluded from this package and Issues.

## Family mode

Agreed 2026-09-07: separate member sign-ins, full shared visibility, personal/household accounts and goals, one shared chat and one plan with expense allocations. Permission, allocation and pooled-funds rules are in [contracts.en.md](contracts.en.md); decisions D-18–D-29 are in [proposal.en.md](proposal.en.md).

## Desktop UI/UX

[Design and animations](design.en.md), [35 screens, forms and states](screens.en.md), [navigation and flows](navigation.en.md). macOS laptop Chrome/Arc only; actual browser acceptance is separate from Chromium CI.
