# Task-7.10 — tokens, typography and brand

[Русский](task-7.10-design.md) · [Contract](../design.en.md) · [Contrast](../design-contrast.md)

The foundation lives in `web/src/design-system/`; task-1.1 is already included in the base. CSS custom properties own colors, states, spacing, radii, typography and durations. React/Base UI/Tailwind, Node/npm and dependencies stay unchanged. Backend, OpenAPI, migrations and production are untouched.

User refinements strengthen cyberpunk and Middle Eastern architecture: obsidian planes, violet accent, white/violet wordmark and sparse sand color. No card/field outlines. Keyboard focus inverts fill/text, including hover/active. Existing shadcn aliases remain without requiring visible contours. The arch is decorative geometry in a brand specimen; the original logo is byte-identical. Advertising references do not add any banking capability.

## Font coverage

| Resource | Google Fonts revision | Weight | Coverage and limitations |
| --- | --- | --- | --- |
| Pixelify Sans | `8b0a1d0f5983c89bc2b93f1b5fb55f9e252744b5` | 400–700 | Selected full TTF lacks U+041E, U+041F, U+20BD, U+20BF, U+202F. No `tnum`; digits differ in width. |
| Manrope | `fb629caaa15ad25c051089c98f09cf6c8e30a86b` | 200–800 | Covers those letters/currency signs; U+202F uses system fallback. Supports `tnum`. |

Full original TTF files were neither converted nor subsetted. Sources, font/license SHA-256, weight ranges and limitations are pinned in `web/public/fonts/manifest.json`; original OFL files are adjacent. cmap/GSUB/fvar were inspected with FontTools during asset preparation; it is not an application dependency. Browser checks verify loading and equal tabular-digit widths. `font-display: swap`; synthesis disabled. Missing glyphs use Manrope/system-ui. No external font CDN at runtime.

Specimens include RU/EN, minus, percentages, NBSP/narrow NBSP and RUB/USD/USDT/USDC/BTC/ETH. Fractional USDC/ETH preserve 18 digits, BTC 8; the 256-character string wraps without digit loss. No monetary arithmetic or `Number` conversion is introduced.

## Verification and reproduction

- `make test-web FILTER=design-tokens`: values/aliases, radii, MOT durations, unrounded sRGB contrast, hash/OFL, unchanged logo and dark first paint.
- `make e2e SCENARIO=design-tokens`: Chromium at 1280×720/1440×900, loaded/failed fonts, tabular figures/long amounts, RU/EN, states/focus, reduced motion and system light preference. Seven scenarios, including reflow at 640×360/720×450 to supplement actual zoom, included in CI.
- `make check`: formatting, lint/typecheck, unit/tooling, production build, RU/EN/traceability and OpenAPI regressions. Compiled CSS validation checks source parity, local resources and absence of dev specimens.
- `git diff --check`: final diff check. Exact results and current candidate SHA are recorded in the PR; earlier passes never replace checks for changed inputs.

Open `/__design/tokens` in dev/test: brand → typography/amounts → states → allowed pairs. The page, its CSS and specimens are absent from production. Update contrast Markdown with `node web/scripts/write-contrast-matrix.ts`; documentation through the catalog and `spec_tool.py render`.

Per direct user refinement, manual checks for this task use Chrome. Arc is unverified and remains in the later product matrix. Actual Chrome: the user set 200%; DPR 4 versus initial 2 and viewport 689×449 CSS px versus 1378×899 were confirmed. All six amounts and the 256-character string remain fully accessible, without horizontal overflow; input works. This is actual zoom, not CSS zoom; automated 640×360/720×450 checks supplement it with reflow coverage. Exact viewport/current candidate results are recorded in the PR.

## Boundaries

AC-094/095 cover tokens and specimens only. Task-7.11 owns the complete component catalog, task-7.1 and feature tasks own screens, task-7.15 owns effects and financial MOT triggers. This task sets 800/800/1400/1800/200 ms and zero reduced-motion durations. Full AC-102, all screens, real passkey/banking, AI, push and production readiness are not claimed. SDD remains **Ready for development**.
