# Want Keep MVP specification

[Русский](README.md)

This package records the agreed MVP and full backlog for later execution by an AI agent. Only the reproducible task-1.1 foundation is implemented; product workflows and the public API do not exist. All 15 initial capabilities and all six integrations are mandatory; partial feature delivery does not make the full MVP complete.

**Status:** requirements and decomposition prepared; implementation readiness is **Not Ready** until external contracts are verified. See [verification.en.md](verification.en.md).

## Reading order

1. [Product and interview decisions](proposal.en.md).
2. [87 requirements](requirements.en.md), [105 acceptance criteria](acceptance_criteria.en.md), [traceability](traceability.en.md).
3. [Architecture](constraints.en.md), [contracts and calculations](contracts.en.md), [glossary](glossary.en.md), [data flows](flows.en.md).
4. [Integrations and sources](integrations.en.md), [operations and costs](operations.en.md).
5. [67 tasks and GitHub links](backlog.en.md).

## For the next agent

Finish task-0.1–task-0.9 and record research evidence. Without securely supplied access, do not claim live verification. The owner authorized task-1.1 early only as an independent technical foundation. task-0.10 still resolves contracts, updates RU/EN and reviews SDD readiness; product implementation remains blocked until then and `plan.md` intentionally does not exist.

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

D-32 refinement: the current Ozon contract covers a debit card and linked main account; other products at this provider are deferred and do not block the MVP. [Research findings](evidence/ozon.en.md).

D-33 refinement: Aifory — RUB accounts, USDT, ETH and the existing USD card; other products deferred without blocking. ETH added to accounting and valuation. [Research](evidence/aifory.en.md); unresolved contract questions remain in BLK-05.

D-34 refinement: EMCD covers the USDT wallet, used Grow/crypto cards and P2P history. Mining has never been used; its data and other unused products are deferred without blocking readiness. [Research](evidence/emcd.en.md), automation questions in BLK-06.

D-36: Bybit — Funding USDT/USDC/ETH/BTC, used Easy Earn and P2P; official API reads are authenticated successfully, including P2P. Other products remain non-blocking deferred. USDC is included in accounting/valuation. [Research](evidence/bybit.en.md), [private API evidence](evidence/bybit-api.en.md); BYBIT-B03/B04 remain in BLK-04.

task-0.7 rate research: CBR selected as primary USD/RUB, CoinGecko Demo for separate BTC/ETH/USDT/USDC in USD current and ≤365-day history, and Frankfurter `providers=CBR` as fallback/cross-check. TradingView rejected as a server-side source. [Evidence and open FX-B02–FX-B04](evidence/fx.en.md); task-0.10 closes BLK-07 and Ready.

D-35 refinement: Raiffeisen covers only the individual entrepreneur current account through RBO API; personal cards, credit, savings and deposits are excluded without blocking. [Research](evidence/raiffeisen.en.md): historical data read; remaining questions are in BLK-02.
