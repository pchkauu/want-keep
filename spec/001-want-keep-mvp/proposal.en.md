# Want Keep MVP product proposal and decisions

Execution mode: autonomous

Initially agreed on 2026-09-06; family amendment agreed on 2026-09-07. Basis: the detailed interview and the user's explicit request to implement the agreed documentation and GitHub backlog plan.

## Problem and outcome

Manually maintaining fragmented fiat/crypto finances is time-consuming. Want Keep combines owned accounts, transactions and documents into verifiable accounting, supports monthly planning and explains outcomes through AI.

The first release is a family pilot for a couple. Expected load is hundreds of monthly transactions; daily chat, receipt, budget and clarification work targets up to 45 minutes according to the owner. There is no current application to migrate; importing old Excel records is not mandatory scope.

This stage delivers bilingual specifications, criteria, architecture, integration matrix, full backlog and self-contained GitHub Issues. It does not implement the application, purchase infrastructure, commit/push or deploy.

## Interview decisions

| ID | Decision |
| --- | --- |
| D-01 | One household, two members with separate sign-in, desktop web on macOS laptops in Chrome and Arc. All 15 capabilities and six integrations are required for the complete MVP. Hundreds of transactions/month; up to 45 minutes/day usage target. |
| D-02 | RUB, USD, USDT, USDC (D-36), BTC, ETH (D-33); cash and bank money are different accounts. Reporting currency switches. Owned money, availability, debt and credit limits are distinct. |
| D-03 | Hourly and on-demand automatic sync. History starts on a selected date; opening balances and coverage boundaries are explicit. |
| D-04 | Owned transfers/exchanges are linked without repeated income/expense; fees remain separate. Duplicates are handled in accounting. Originals and correction history remain; discrepancies are investigated. |
| D-05 | Full expense in the payment month, including annual subscriptions. Refunds recalculate the original purchase month; cash movement retains the refund date. |
| D-06 | Category/subcategory, merchant and receipt item are separate dimensions. Milk is an item; a venue is a merchant, not a mandatory subcategory. |
| D-07 | Text, photos and PDFs in web chat. Receipts require an account dropdown, including cash. Clarify uncertainty; explain skipping obviously irrelevant documents; record/link established transactions. |
| D-08 | AI maximally automates internal accounting and reviews every new/changed transaction. Missing financial values are not invented. Full transactions/receipts may go to OpenAI, excluding secrets. Ordinary accounting continues during AI failure. |
| D-09 | Calendar month; dated expenses, flexible categories and forecast income. Copy plans without remaining/overspend rollover. AI applies approved-plan changes only on an authorized member decision. |
| D-10 | Goals have amount, currency, deadline and virtual-reservation or dedicated-account mode. No double reservation. Daily allowances are overall/per-category and available/forecast with separate funding by currency. |
| D-11 | Credit cards with debt, payments and grace; savings with actual/forecast and comparable dated-cash-flow returns; realized/unrealized trading P&L, fees, funding and mining. No trading terminal. |
| D-12 | Transaction-date expense valuation, current wealth valuation and separate FX effects. Reference rates and available provider buy/sell quotes with fees. USD, USDT and USDC are not automatically equal. |
| D-13 | Alfa-Bank: cards, current/savings accounts, deposits; Raif Russia: individual entrepreneur current account through RBO API (D-35); Ozon Bank: debit card and main account (D-32 refinement); Bybit: Funding USDT/USDC/ETH/BTC, Easy Earn, P2P (D-36); Aifory: RUB accounts, USDT, ETH and existing card with their movements (D-33); EMCD: used cards, Coinhold/Grow, USDT wallet and P2P history (D-34). Read-only; browser automation allowed. |
| D-14 | Separate passkeys and personal one-time recovery codes for each member; partner-assisted reset is unavailable. Protected secrets, attachments and sessions; AI has no payment authority. |
| D-15 | For the whole household: server up to $40/month in DE/NL/BG, OpenAI up to $50/month; separately sourced data must be free. Hourly MacBook backups while reachable; visible backup age, conditional RPO and recovery target within four hours. |
| D-16 | UI, AI interaction and documentation in RU/EN. Dashboard covers plan/actuals, income/expenses, goals and daily allowances. In-app and web-push notifications. |
| D-17 | Go, PostgreSQL, React/TypeScript/Vite, separate Playwright TypeScript collector, Docker Compose. Financial domain independent of transport/storage/UI/AI SDK; exact arithmetic and explicit boundaries. |

## Preserved boundaries

The application reads platforms and changes only its own accounting. Payments, actual transfers, trade orders and bypassing MFA/CAPTCHA are outside the product. Public registration, member exit/replacement, additional roles, mobile screens/adaptation, home-screen installation, native mobile, tax reporting, mining control and old Excel migration are outside this MVP.

Manual cash and receipt entry is mandatory. Manual bank-statement upload may aid research but does not fulfill automatic connection of a mandatory service.

## Engineering defaults

These are defaults, not user answers: initial `Europe/Moscow` timezone explicitly selected by the owner; online operation with clear offline status; two category levels; private push without monetary details; annualized money-weighted XIRR for investment comparison; retained originals for audit. Formulas, file limits and errors are in [contracts.en.md](contracts.en.md).

Fundamental contract changes require both languages, ACs and dependent tasks to change. Library versions, verified APIs, full product terms and a server tariff are not selected yet: they are task-0.1–task-0.10 outcomes, not hidden decisions left to implementers.

## Readiness

Documentation and backlog can be delivered before live access. Full-MVP Ready requires evidence for every mandatory capability/product. [verification.en.md](verification.en.md) retains Not Ready until external contracts close; [backlog.en.md](backlog.en.md) includes the full scope and blocked tasks.

## Family amendment

| ID | Decision |
| --- | --- |
| D-18 | User, household, membership and roles are separate. MVP has two members with a configured limit; no fixed member fields, public registration, exit or replacement. |
| D-19 | Both see everything and edit all transactions. Personal goals/plan portions are owner-editable; joint ones are editable by either member. One authorized approval suffices and the other is notified. |
| D-20 | Personal and household accounts. Account ownership, external owner, payer, actor and expense beneficiary differ; reconnecting one source does not duplicate it. |
| D-21 | One monthly household plan with individual views; all income and available funds are pooled. A personal allowance may be funded by the other member’s money. |
| D-22 | Personal/joint attribution by transaction and item; joint shares are 50/50 with plan-line/purchase overrides. AI applies rules and clarifies unknown attribution. |
| D-23 | Household transfers are not income/expense; inter-member debts and settlements are explicit and do not increase household wealth. |
| D-24 | Personal and joint goals receive explicit reservations; joint goals appear in a separate shared block without personal shares. No automatic allocation to goals. |
| D-25 | The household cap is calculated once per currency; individual and category limits partition it instead of repeating it. Goals, holds and obligations are deducted once. |
| D-26 | One shared AI chat; message and decision authors are retained. Text cannot replace principal; personal plan/goal changes require the authorized owner. |
| D-27 | Revisions and permissions are checked for edits, clarifications and reversal. Concurrent conflict requires refresh; the other member’s later edit is preserved. |
| D-28 | Both manage connections. The external-account owner supplies password/MFA; partner and AI receive no secrets. Platforms remain read-only. |
| D-29 | Goal/plan changes notify the other member; read state and push subscriptions are individual. Recovering one sign-in does not reset the other. |

## Desktop design and interaction

| ID | Decision |
| --- | --- |
| D-30 | macOS laptop Chrome/Arc only, 1280×720/1440×900 and zoom; dark #1A1A1A / #5F4EF5, Pixelify Sans + Manrope, shadcn on Base UI. Reference sign-in, 35 question/answer/action screens, clear states and accessibility. Mobile scope excluded, push retained. |
| D-31 | Contextual pixel animations: rocket for adding an accounting account, top-up sparkles, confetti/soft disco for achievement, calm overspend signal. Confirmed events only, presentation dedup, off and reduced motion; no payment execution or bank-product opening. |

## Ozon coverage refinement, 2026-09-07

| ID | Decision |
| --- | --- |
| D-32 | The current MVP supports the available Ozon debit card and linked main account. Credit cards, savings and deposits belong to a future contract extension; their absence does not block Ozon/the MVP. This explicitly changes the earlier matrix, rather than claiming those products do not exist at the bank. Debit-data reading quality/automation, household identity and shared accounting rules remain. Other providers are unchanged. |

## Aifory coverage refinement, 2026-09-07

| ID | Decision |
| --- | --- |
| D-33 | Current Aifory scope: RUB accounts, USDT, ETH and the existing crypto card with its actual USD balance. Other products/currency wallets, other cards, standalone P2P/referral products and service catalogs are deferred and do not block the MVP. Retain all included-wallet movements even when the related service is deferred. ETH joins accounting and selectable valuation currencies. Selected-product reliability/automation remain mandatory; expansion needs new verified contracts. Other providers and shared functions are not reduced. |

REQ-002/REQ-003/REQ-046, their ACs and downstream tasks were updated with IDs preserved. This is target-contract version 4; no application exists yet, so no runtime/data migration is required. [Research outcome](evidence/aifory.en.md).

## EMCD refinement, 2026-09-07

| ID | Decision |
| --- | --- |
| D-34 | Current EMCD scope: used crypto cards, Coinhold/Grow, the USDT wallet and historical P2P orders. Mining has never been used; neither its current data nor history is required. Mining and other unused products/wallets are deferred without blocking the MVP; expansion requires a new decision and verified contracts. Retain all included-wallet movements. Automation and quality for selected products, shared features and other providers are not reduced. |

## Raiffeisen refinement, 2026-09-07

| ID | Decision |
| --- | --- |
| D-35 | The owner explicitly confirmed the current Raif scope as the individual entrepreneur current account only: balances, incoming/outgoing movements, fees and history through RBO API. Raif personal cards, credit, savings accounts and deposits are excluded from the current MVP and do not block readiness. Other providers and shared features are unchanged. |

REQ-043/AC-043 and task-0.2/task-4.2 were refined with IDs preserved; the Raif link to AC-070 on credit/savings terms was removed. This changes the target contract; no runtime schemas or data are migrated. [Research outcome](evidence/raiffeisen.en.md).

## Bybit refinement, 2026-09-07

| ID | Decision |
| --- | --- |
| D-36 | Funding USDT/USDC/ETH/BTC, used Easy Earn and P2P. Official APIs take priority over Playwright. Spot/UTA trading, futures, options, card, On-Chain/Advanced Earn and other unused products are deferred without blocking; retain included-wallet movements. USDC is a distinct accounting and selectable valuation asset without assumed USD/USDT/USDC parity. Expansion requires a new decision and verified contracts. |

REQ-002/REQ-003/REQ-039/REQ-045 and task IDs remain; AC-071 and task-6.5 no longer require unused Bybit products. Shared financial functions remain. Target-contract version 6; task-1.1 foundation exists, financial API/database do not and no migration is needed. [Research](evidence/bybit.en.md). Authenticated RSA reads later closed BYBIT-B02/B05 access; BYBIT-B03/B04 and BLK-04 remain open. See [private API evidence](evidence/bybit-api.en.md).
