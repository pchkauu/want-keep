# Want Keep components

[Русский](design-components.md) · [Design](design.en.md) · [task-7.11](tasks/task-7.11.md)

## Purpose and ownership

`web/src/design-system/components/` owns visual component sources. They are adapted from inspected shadcn 4.21.0 `base-nova` templates using Base UI 1.8.0. This is project-owned code, not files to overwrite through generation. Before updating, use CLI `add --dry-run`/`--view` and inspect contract and dependency changes.

The interactive `/__design/components` catalog uses synthetic data only. Its code, compositions and styles are absent from the production bundle. Reusable components are available to product features; demonstration handlers are not. The catalog does not call financial APIs, persist browser storage, implement accounting or recover access.

User refinement for #58: shapes and composition from the new reference, with the Want Keep palette. A prominent accent surface, side action tiles, inset rows and a profile surface are adapted for desktop. Radii: fields/regular buttons 12 px, panels 24 px, prominent surfaces 32 px, pill badges. This replaces the previous 2/4/8 px while retaining original colors and fonts. Outlines, mobile bottom navigation and payment actions from the reference are excluded.

Adapted shadcn portions retain the MIT notice in `web/src/design-system/components/LICENSE.shadcn`. Registry previews were inspected before adaptation; committed project sources are authoritative.


## Public interfaces

| Family | Contract |
| --- | --- |
| Button / ActionTile | Events go to the caller. Ordinary Button defaults to `type=button`; submit is explicit. `disabled` and `aria-busy` are separate. Links use Base UI `render`/`nativeButton=false`. |
| Input / Textarea / MoneyField | Controlled string; MoneyField uses no Number/float/rounding or length truncation. `assetLabel` is a label. The consuming form owns validity and errors. An expandable specimen exposes the entire long amount. |
| Field / Form | Base UI associates label, description, error and control. Textarea uses explicit `htmlFor/id`. Submission state and application validation belong to the consuming form. |
| DateField | `value` is `YYYY-MM-DD` or null; optional `draft` retains original text. `onChange` returns `{draft, value}`: undefined means invalid text, null means empty, a string means an existing date. The consumer retains draft and never submits undefined. RU `DD.MM.YYYY`, EN `MM/DD/YYYY`; Gregorian, Monday first. |
| Calendar | DayPicker wrapper. CalendarDate owns JS Date to calendar-string conversion without `toISOString` or timezone shifting. Popover date selection restores focus to its trigger. |
| Select / Combobox | String ID, explicit options, labels and callback. Combobox searches supplied options; network search belongs to a feature. |
| Checkbox / Radio / Switch / Toggle | Base UI state and keyboard behavior; consuming labels are required. Focus does not replace selected/checked semantics. |
| Dialog / AlertDialog / Sheet | Required title and description. Modal focus trap, Escape and focus restoration; sheet shares the dialog boundary. The consumer implements confirmation behavior. |
| Popover / Tooltip / Toast | Brief contextual information. Tooltip never replaces a label. Base UI toast, maximum three, 8 seconds; important results remain persistent. Dismiss labels are supplied in the selected locale. |
| Sidebar / Tabs | Desktop navigation; Sidebar receives explicit links/currentId. It does not determine users, permissions or data access. Tabs uses Base UI roving focus. |
| Table / Pagination | Semantic table in a labelled scroll region. Pagination receives direction availability, summary and callbacks; it does not infer server row counts or sort financial values. |
| Surface / Avatar / ListItem | Filled surfaces, initials fallback, inset rows. Images and data are supplied by consumers. No built-in profile or external-source access. |
| Badge / Alert / Empty / Skeleton | Text explains status independently of color. Alert announces only with explicit `announce`; skeletons never display zero amounts. |
| Progress / ChartLegend | Progress receives a computed percentage or null. Legends receive formatted value labels and visibility callbacks; markers differ in shape. Calculations and charts belong to analytics. |
| ChatResult | Author, status, explanation, details/actions are explicit presentation props. It neither interprets AI output nor assigns a financial outcome or executes commands. |

## States and accessibility

The catalog demonstrates UISTATE-01–17. Applicable states are covered at their owners: action disabled/loading, field invalid, selection selected, demo-form error/unknown/conflict. Primitives do not introduce a universal financial state machine.

Unknown retains the draft and prevents duplicate submission; an explicit simulated original-command check moves the specimen to confirmed. Conflict compares saved and entered versions. Session expiry hides the synthetic draft. These examples verify presentation behavior; server guarantees remain in task-1.3 and feature tasks.

Component boundaries contain no generated DTOs, API clients or feature code. Keyboard focus inverts fill and text, including error/selected/hover combinations. Fonts are local. Reduced motion zeroes ordinary transitions; MOT-01–05 remain task-7.15. The catalog does not prove full WCAG compliance or AC-102.

## Verification and handoff

`make test-web FILTER=design-components`, `make e2e SCENARIO=design-components`, `design-tokens` regression, `make check`, `git diff --check`. Actual Chrome is checked separately from Chromium; per user decision Arc remains later acceptance. Reference viewports are 1280×720/1440×900 at 100%/200% zoom.

Task-7.1 and other screen tasks consume these primitives and own their view models, application states and command recovery. When switching locale, the consuming form reformats a valid calendar value and preserves an invalid draft for correction. Task-7.12 covers desktop UX and visual acceptance; task-7.13 owns connection screens; task-7.14 owns settings and accounting health screens; task-7.15 owns event animations. [Implementation evidence](evidence/task-7.11-components.en.md).
