# Task-7.11: Want Keep components

[Русский](task-7.11-components.md) · [Contracts](../design-components.en.md) · [Issue #58](https://github.com/pchkauu/want-keep/issues/58)

## Result

Project-owned Base UI components and the interactive RU/EN `/__design/components` catalog are implemented. The new reference supplies rounded shapes, a prominent surface, side actions, inset history and a profile surface; the user retained the Want Keep palette and fonts. CSS and RU/EN radius contracts were updated together. Fonts and the logo are unchanged.

Components accept presentation props and callbacks. Money strings never pass through Number; manual dates retain invalid drafts, and the Gregorian calendar does not convert dates to UTC. All 17 synthetic UISTATE scenarios, a nested calendar/select in a dialog, focus restoration, unknown outcomes, conflicts and persistent confirmation are provided. The form prevents duplicate demo submissions.

The catalog and its styles are excluded from the production bundle. shadcn aliases point to design-system ownership, pinned CLI template previews were inspected, and the MIT notice is retained. Only agreed dependencies were added. Backend, API, migrations and production are unchanged.

## Verification

| Check | Result |
| --- | --- |
| `make test-web FILTER=design-components` | 49 unit/component checks: 6 assets, exact/intermediate strings, 256/257 characters, calendar, semantics, UISTATE and import boundaries |
| `make e2e SCENARIO=design-components` + `design-tokens` | 26 passing Chromium scenarios: 19 components + 7 tokens; the same files run through separate CI targets |
| Computed styles | Normal/hover/active/focus contrast, selected/error; no outlines; reduced motion; font fallback |
| Chromium reference viewports | 1280×720, 1440×900 and corresponding 640×360, 720×450 CSS areas for enlarged content |
| Actual Chrome | Chrome 152.0.7977.76, macOS 26.5.1 (25F80); 100%: 1378×899 CSS/DPR 2, 200%: 689×449 CSS/DPR 4. Composition, long amounts, dialog, reachable submission and absence of page overflow verified; restored to 100% |
| `make check` | Formatting, lint, typecheck, unit, build, SDD and reproducible OpenAPI checks passed |
| `git diff --check` | Checked before each publication |

Initial checks found and fixed clearing a selected date on repeated selection and an assistive-technology-hidden toast dismiss button. Tabs follow Base UI: arrows move focus and Enter selects. A locale switch reformats a valid date and preserves an invalid draft. Sandbox restrictions on the local port and Go cache were resolved by running authorized checks with required access; they were not product failures.

Round 1 independently confirmed four issues: calendar year boundaries, avatar accessible names, indeterminate progress and portal language. Corrections include unit and browser regression cases. The boundary test locator was narrowed to textbox semantics because a closing popover shared its label. Affected checks were repeated; unchanged token scenarios retain their passing evidence. Actual Chrome also confirmed Russian portal language and localized unknown progress after correction. The published correction requires a fresh Avida round.

## Evidence limits

AC-096 is confirmed within the catalog. AC-094/095/099/101 are covered only in the described components and demonstrations. Actual Chrome and automated Chromium evidence are separate; reference viewport checks are not claimed as physical Chrome resizing. Arc is deferred by user decision. Product screens, server command recovery, live providers, AI calls, financial-event animations and complete WCAG/MVP acceptance were not checked here. SDD remains Ready for development; operational readiness is not claimed.

Self review, published-candidate CI and merge are confirmed separately through PR and issue links; local checks do not replace them.
