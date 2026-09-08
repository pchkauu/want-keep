# Task-7.9: household context in the desktop interface

[Русский](task-7.9-household-context.md) · [Issue #56](https://github.com/pchkauu/want-keep/issues/56) · [Navigation](../navigation.en.md)

## Implementation

The public `web/src/features/household/` module owns `/household` and invitation transport mapping, loading state and the selected view. Its controller binds a response to household, user and session; every session ID change starts a new load and a late response from the previous session is discarded. Reauthentication by the same member keeps the current screen mounted while revalidating household state. Identity retains WebAuthn and invitation acceptance; accounts receives only the typed household view and members.

The view lives in the URL as `?view=household` or `?view=member&member=<userId>`. Unknown, missing or pending members fall back to household view. Shell navigation carries only view/member. The module exposes the exact existing OpenAPI shape `{view}` or `{view, memberId}` to future report consumers. The signed-in member has a separate label; changing view never changes the session principal.

`/settings/household` shows membership composition and status, member cap, timezone, current invitation, personal/household resource boundaries and a link to own security settings. Issue and reissue require personal reauthentication; revocation uses the active session. A lost response or revision conflict requires reading current metadata; the secret link is never replayed. Offline blocks changes while retaining available read data. Role management, member exit/replacement and partner-assisted recovery are absent.

Account presentation retains server-returned ownership and external account owner. Household view groups every account; member view shows household accounts plus the selected member's personal accounts. Labels distinguish household, personal with member name, and platform owner. Hidden rows explicitly remain in the household money pool. Personal cash creation still receives owner from `CashController.userId`, created for the signed-in actor; the URL filter never enters the mutation.

Backend, OpenAPI, migrations, dependencies and production are unchanged.

## Verification

`make test-web FILTER=household` checks URL normalization, report-context shape, inactive/unknown members, session changes and stale responses, membership/revision transport, household account filtering and actor independence from selected view.

`make e2e SCENARIO=family-access` starts isolated PostgreSQL 17.11 and the Go API, applies migrations and uses two virtual WebAuthn authenticators. The scenario covers bootstrap, personal and household cash accounts, a lost invitation response, stale revision, revocation and denial of the revoked token, partner enrollment, both `/settings/household` views, deep links/back/forward, return from an unfinished section, a safe household view when `/household` fails, account grouping and partner-owned personal account creation while Andrey is selected. RU/EN, keyboard, reduced motion, 1280×720, 1440×900 and 200% equivalents 640×360/720×450 run without horizontal overflow.

`make check` passed: formatting, lint, TypeScript, 117 web unit tests, Go/collector, build, SDD and reproducible OpenAPI. `make test-web FILTER=household` contains 10 tests. `make e2e SCENARIO=all` passed 28 scenarios: access, 19 component, 7 token and family-access. PostgreSQL household/accounts integration and both race suites passed against an isolated database. `git diff --check` passed.

Real Chrome on macOS was checked at 1414×943 CSS px / 100% and 707×471 / 200%, then returned to 100%. The user confirmed native passkey creation. The check covered the household selector, separate actor label, `/settings/household`, view retention in the own-security link, scroll access to actions and no horizontal overflow. A clean reload followed the dev-only locale HMR; the current page had no console errors. Automated 1280×720, 1440×900 and 640×360/720×450 areas cover the complete two-member scenario. This does not prove physical Chrome window dimensions.

Self review, published-SHA CI and merge evidence are recorded separately in the PR and issue.

## Boundaries

REQ-001/063/064/065/071/074/076 and AC-001/077/078/079/085/088/090 are covered only for the implemented shell, household, invitation and account scope. Complete goals, plans, chat, personal notification read state and other financial screens remain with their owning tasks. Arc, live banks, AI, production and operational readiness are not verified here. SDD remains Ready for development.
