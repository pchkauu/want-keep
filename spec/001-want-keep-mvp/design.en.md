# Want Keep design contract

[Русский](design.md) · [Screens](screens.en.md) · [Navigation](navigation.en.md)

D-30/D-31, REQ-077–REQ-087, AC-094–AC-104. The task-7.10 visual foundation is implemented; product screens and their full acceptance remain follow-up work. [Evidence and boundaries](evidence/task-7.10-design.en.md).

## Platform and character

Desktop web only on a macOS laptop in actual Chrome and Arc. Reference windows: 1280×720 and 1440×900 CSS px, at 100% and 200% zoom. Resizing preserves access to actions, figures and forms. Mobile adaptation, bottom navigation, home-screen installation and phone reference sizes are excluded. Browser push remains.

Dark modern pixel fintech: technological, calm, composed. Whitespace and clear hierarchy take priority over decorative filling. User refinement: restrained cyberpunk and Middle Eastern architectural rhythm. Obsidian planes, localized violet light, a white/violet wordmark, a sparse sand accent and arch forms. Conversation references establish material and mood; their phones and payment cards do not expand desktop MVP functionality. No card/field outlines, decorative contours, neon grids or pervasive glow. Group through space, typography and fills; geometry belongs to brand specimens, not every transaction. Pixel character does not turn forms/tables into a retro game. No light theme, including the first loading frame and browser elements where styling is supported.

## Tokens

| Role | Target value and use |
| --- | --- |
| background | `#1A1A1A`, persistent background |
| surface / surface-raised | `#202020` / `#262626`, nested region and panel |
| brand / primary | `#5F4EF5`, primary CTA, brand, selected accent |
| text-primary / secondary | `#F5F5F5` / `#B3B3B3`, primary and secondary text |
| brand backdrop / architecture | `#111114` / `#18171E`, deep brand planes without replacing the base background |
| sand / warm surface | `#C7AF8F` / `#242220`, small warm accents |
| accent-readable | `#A79BFF`, small accent text |
| compatibility aliases | `border`, `input`, `ring` remain for shadcn/Tailwind; an alias does not prescribe an outline |
| focus | Inverted fill `#F5F5F5`, text `#1A1A1A`; no outline, visible during hover/active too |
| success / warning / error | `#83C9A0` / `#D8B36A` / `#E58B91`, text/icon on dark surface; color accompanies a label |
| spacing | 4 px step; working gaps 8/12/16/24/32/48, larger whitespace follows composition |
| radius | 2 px badge, 4 px fields/regular buttons, 8 px panels and large sign-in CTA |
| motion | Regular transitions 150–200 ms; separate event contract below |

CSS custom properties in `web/src/design-system/theme.css` are the single source; shadcn/Tailwind reference them. The [allowed-pair matrix](design-contrast.md) is computed from CSS without rounding before threshold comparison. Normal CTA is `#5F4EF5`, hover `#6B5AF6`, active `#5142D5`, with white labels. Selected uses readable violet on `#262626` plus a selection mark; disabled preserves readable text without an available action. Focus inverts fill and label. Decorative/inactive alias exemptions never apply to text or meaningful indicators. Normal text ≥4.5:1, large text ≥3:1; meaningful boundaries/indicators ≥3:1. Brand `#5F4EF5` on `#1A1A1A` is about 3.23:1 and unsuitable for small text; white on purple is about 5.39:1. Purple logo is a brand mark; labels/help use legible text. [W3C: contrast](https://www.w3.org/WAI/WCAG22/Understanding/contrast-minimum.html).

Status colors are engineering defaults within the agreed muted palette. Error is not red alone, forecast not opacity alone, selected member not background alone. Minimal shadows may separate popovers; permanent cards use fills and space without outlines.

## Typography and logo

- Pixelify Sans: logo/name, large headings, key amounts, selected badges/accent buttons. Avoid dense transaction rows, multiline explanations and small forms.
- Manrope: transactions, forms, tables, descriptions, settings and secondary information. Base UI 14–16 px with sufficient line height; key amounts 28–40 px depending on space. Critical amounts are never ellipsized.
- Local full variable TTF files: Pixelify Sans 400–700 and Manrope 200–800, without conversion/subsetting. OFL files, source URLs, pinned revisions and SHA-256 are in `web/public/fonts/manifest.json`. Use `font-display: swap` and disable synthetic styles.
- Brand chain: Pixelify Sans → Manrope → system-ui → sans-serif; interface: Manrope → system-ui → sans-serif. Selected Pixelify Sans lacks Cyrillic “О”, “П”, ₽, ₿ and narrow no-break space; fallback supplies those glyphs. Manrope covers those letters/currency signs; narrow space uses system fallback. Pixelify Sans has no `tnum`; tables/comparable amounts use Manrope tabular figures.
- RUB/USD/USDT/USDC/BTC/ETH remain exact strings, including 18 fractional USDC/ETH digits and the 256-character limit. No `Number` conversion or ellipsis; wrapping preserves every digit. [Font coverage report](evidence/task-7.10-design.en.md#font-coverage).


The user supplied the [original logo](assets/logo_512px.svg). Despite its 512px filename it has `viewBox="0 0 455 512"`: preserve proportions and pad a square container. Color `#5F4EF5`, pixel geometry without decorative distortion. Do not automatically redraw or replace it with emoji.

## Reference sign-in

Source: the user’s image in the conversation; Figma node/exact dimensions were not inspected. This is a composition reference, not a measured design.

Compact mark and Want Keep centered in dark space. Below with a substantial gap, one primary button `Log in with Passkeys` / «Войти с passkey». The block has bounded width accommodating both locales; exported pixels are not copied directly. Subtle language and “Trouble signing in?” / «Не получается войти?» do not compete with the primary action. No bank logos, dashboard cards or password fields on sign-in.

System passkey waiting, cancellation and error use reserved status space; mark/name/CTA do not jump. Error provides retry/help; cancellation is not a malfunction. Recovery remains SCR-002. Financial-event accent animations never play on sign-in.

## Components and accessibility

shadcn/ui on Base UI, with Want Keep owning component sources. Explicitly pin Base UI selection during setup and versions/lockfile in task-1.1/task-7.11; never rely on changing CLI defaults. Official shadcn supports Base UI. [shadcn documentation](https://ui.shadcn.com/docs/changelog/2026-07-base-ui-default).

`web/src/design-system/` owns tokens, primitives and motion. Features own screens/forms/presentation; server domain calculates money, authority and budgets. No universal global screen or hidden API access inside visual components. No mandatory new animation library: start with CSS/SVG and the existing stack.

The component catalog covers default/hover/active/focus/selected/disabled/loading/error: buttons, money/date/text fields, select/combobox, forms, dialog/alert/sheet, navigation/tabs, tables/filters/pagination, badges/alerts/toasts, tooltip, empty/skeleton, chart legends and AI result cards. Replace stock shadcn visuals with project tokens, typography and composition. The catalog is a dev/test tool, not a household-product screen.

Interactive elements support Tab/Shift+Tab/Enter/Space according to semantics. Escape closes temporary layers and restores focus; sticky regions never obscure focus. Dialog has name/description, traps focus and never nests another dialog; long tasks use pages/panels. Tooltip supplements a label rather than providing the sole explanation. Errors associate with fields; progress/outcome reaches assistive technology without spam. Charts have textual conclusions and accessible tables. Zoom preserves CTA/amount access; horizontal scrolling is allowed within labelled complex tables, not forms/whole pages.

## Contextual animations

Animation responds to an event; static text explains the outcome and remains accessible. Neither effect nor color is the sole confirmation. No sound, flashing, strobe or continuously moving decorative backgrounds.

| ID | Event and effect | Constraint |
| --- | --- | --- |
| MOT-01 | Account added to accounting: short pixel rocket launch beside confirmation, 600–1000 ms | Confirmed creation only; not viewing an account or opening a bank product |
| MOT-02 | Confirmed savings top-up: a few sparkles around progress, 600–1000 ms | Principal is not return; forecast and repeat import never trigger it |
| MOT-03 | Goal actually achieved: local pixel confetti, 1000–1600 ms | Once per confirmed achievement event; refund/correction never hidden by celebration |
| MOT-04 | Major joint achievement: soft disco with slowly shifting geometric accents, up to 1800 ms | MOT-03 variant for completed joint goal, no flashing or abrupt brightness change; does not stack with confetti |
| MOT-05 | Limit first exceeded: calm warning-icon appearance, 150–200 ms | Static excess amount and “View plan”; no celebratory reward for overspending |

Decorative effects default on when `prefers-reduced-motion` is absent; each member may disable them in SCR-031. Reduced motion replaces all MOT with static icon/text regardless of preference. Escape ends decoration without cancelling the transaction; effects never capture pointer/focus, obscure amounts/forms or shift layout. Financial numbers immediately show confirmed values without count-up through false amounts.

Trigger is a confirmed typed application event; page read, AI response, optimistic update and quote are not achievement events. Event ID and personal presentation acknowledgement prevent replay in another tab, retry/reconnect or refresh. Historical backfill creates financial facts without a queue of old celebrations. Show one eligible effect at a time; others remain static events. Presentation dedup does not affect personal notification reading. Unknown outcome waits for reconciliation; later correction leaves a clear new result. task-7.15 materializes eligibility/ack contracts; task-7.12 tests actual browsers.

## UX evaluation

For each SCR: question → primary answer → action → explanation → details. Technical fields stay below the primary level; amounts are labelled available/reserved/expected. Errors name the problem/action; partial history explains limits of conclusions. Clear status, familiar language, control, error prevention and progressive disclosure apply to concrete flows. [Nielsen Norman Group](https://www.nngroup.com/articles/ten-usability-heuristics/).

AC-102: without developer hints, both members identify allowance/cash risk, explain an overview amount, enter a receipt and distinguish creation/linking, correct shares, explain a goal reserve, handle reauth/AI waiting, and sign in/recover own access. Protocol records expected answers on synthetic data, actual answers, mistakes, time and browser. Chromium automation replaces neither observation nor actual Chrome/Arc. Full WCAG conformance is not claimed without a separate audit.

## Reproducible specimens

`/__design/tokens` is dev/test only: brand composition, RU/EN typography, exact amounts, states and contrast pairs. Specimens and their CSS are excluded from the production bundle. `make test-web FILTER=design-tokens` checks tokens/resources/matrix; `make e2e SCENARIO=design-tokens` checks Chromium. MOT-01–05 durations are 800/800/1400/1800/200 ms; reduced motion zeroes them and regular transitions. Financial triggers and effects belong to task-7.15.

Per user refinement, manual verification for current task-7.10 uses Chrome. Arc remains in the later product acceptance matrix and is not claimed verified here.
