# Task-7.1: desktop access and onboarding

[Русский](task-7.1-access.md) · [Issue #48](https://github.com/pchkauu/want-keep/issues/48) · [Navigation](../navigation.en.md)

## Implementation

React Router Data Mode owns the shell and SCR-001–005 routes. Identity and accounts own their state and HTTP mapping; the design system receives presentation props. The only new runtime dependency is react-router 8.3.1. Backend, OpenAPI, migrations and production are unchanged.

Closed bootstrap, native WebAuthn, separate member sign-in, personal recovery, invitation preview/acceptance and invitation issue/reissue/revocation in onboarding are implemented. Minimal `/settings/security` regenerates recovery codes after fresh own authentication. Codes appear once and are cleared when access is lost; copying/downloading is explicit. An unknown enrollment response first checks `/me`, then offers sign-in with the new key. Completion is never automatically repeated.

CSRF stays in memory and the session in a protected cookie. Server-enforced 12-hour/30-minute limits also close the client view; actual activity signals occur only in a visible tab at most once per minute. Background reads do not renew idle expiry. Old responses are discarded after session changes. A lost logout response hides access without claiming server revocation. Tabs exchange only a revalidation signal.

Onboarding creates a real task-2.1 cash account with asset, ownership, calendar date and exact opening funds. A UUID precedes POST. Timeout and `not_found` do not authorize a new key; explicit retry retains the original key and payload. Reload checks recent commands. Confirmation requires a command result and account read; the list refreshes after creation. Money never passes through Number. Changing members clears financial drafts; expiry retains them only in memory for the same member.

Overview shows available accounts and next actions. Other financial sections display unavailable states. Locale priority is browser preference → confirmed profile → supported browser locale → RU; only locale enters localStorage. Forms and ceremonies survive RU/EN switching. Invitation fragments are removed before rendering; secrets never enter URLs or persistent storage.

After local expiry, sign-in first re-reads /me: a live shared cookie restores the session without another prompt or idle renewal. A pending command permits explicit original-request replay. After reload, the user may reconstruct the fields with the same key; the server checks the hash and rejects any difference without an effect.

## Verification

`make e2e SCENARIO=access` starts isolated PostgreSQL 17.11 and the real Go API, applies migrations and creates closed test bootstrap. Chromium uses separate virtual authenticators. Missing infrastructure fails the check. Successful auth APIs are not mocked; response loss follows a real commit. Secret-bearing traces are disabled. A new CI job runs this scenario.

`make check` passed: formatting, lint, TypeScript, 99 web unit tests, Go/collector tests, build, SDD and reproducible OpenAPI. Identity has 15 focused tests; accounts has 8. Access E2E passed its real end-to-end household flow, lost bootstrap/invitation/account responses, Origin/RP rejection and client idle-expiry hiding. The aggregate `make e2e SCENARIO=all` passed all 27 scenarios through their owning launchers. A real PostgreSQL write fault after durable command registration proves same-key retry, reload/reconstruction, changed-payload rejection and one final account. A tab-clock expiry with a live server cookie restores the same-member draft. Design regressions passed 7 token and 19 component scenarios. PostgreSQL identity/household/accounts suites passed. `git diff --check` passed. The build reports a non-fatal 650.85 kB main-chunk advisory; no configured size gate failed. Actual Chrome on macOS was verified at 1414×943 CSS px / 100% and 707×472 / 200%, then restored to 100%. User-confirmed native passkey creation and subsequent sign-in succeeded. Manual checks include user-assisted system passkey creation, RU/EN, exact cash funds, navigation and 100%/200% zoom. Automated reference areas are 1280×720, 1440×900 and 640×360/720×450 for enlarged content. They do not prove physical Chrome window dimensions.

## Boundaries

AC-001/049/050/054/055/072/075/077/078/098/099/100/101/090/004/040/041 are covered only by verified access, shell and cash-account scenarios. Full financial screens, connections, notifications, keys/devices and server preferences belong to later tasks. Arc, bank OAuth, live AI, production and full product acceptance were not checked. SDD remains Ready for development; operational readiness is not claimed. Self review, published-candidate CI and merge are confirmed separately in the PR/issue.

Manual verification: `WANT_KEEP_ACCESS_MANUAL=1 make e2e SCENARIO=access`; the private bootstrap path and local URL are printed without the token. Stopping removes only the created test container. Real keyring files, document processing and proxy settings are not inherited.
