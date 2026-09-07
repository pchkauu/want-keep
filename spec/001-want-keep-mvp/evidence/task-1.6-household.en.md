# Task-1.6 — household, invitations and permissions

Implemented closed second-member invitations, atomic passkey/session/recovery-code enrollment and household policies. Base: `fc0f27d2a8dad4ac0db9672aff8108673e8a8c58`; branch: `feat/task-1.6-household-permissions`. The [contract](../contracts.en.md#task-16--invitations-and-household-permissions) records agreed lifetimes, revocation, revision and unknown outcomes.

## Implementation

Household domain/application owns invitation lifecycle and membership; identity application coordinates real WebAuthn verification and access issuance. HTTP reuses the delivery/identity boundary with a focused household file instead of duplicating session/Origin/CSRF guards. Domain/application does not import SQL, generated DTOs or the WebAuthn SDK.

Migration 005 adds hashed tokens, invitation outcomes, household revision, immutable audit and attempt binding. Migrations 001–004 are unchanged. Execution rechecks sessions; the identity lock precedes household/invitation locks. Enrollment, codes, session and outbox are atomic. Replay, revocation, expiry, replacement, wrong browser/purpose and full capacity cannot grant new access. An invitation token alone never permits financial reads or writes.

Ownership checks the current owner before changing scope; accounting retains separate actor/payer. ExternalOwnership permits both members to manage connections but only the external owner to perform bank authentication. Product financial handlers for new actions and secret storage are not added here.

## Verification

Required commands: `make check`, `make test-integration AREA=household`, `make test-integration AREA=identity`, `make test-integration AREA=storage`, `make test-household-race`, `make test-identity-race`, `make test-storage-race`, `git diff --check`. CI runs all three integration/race suites. Isolated PostgreSQL 17.11 is digest-pinned; missing DB fails rather than skips. Go/Node and API generator versions remain pinned.

New HTTP scenarios live in the identity suite and reuse its ES256/RS256 fixture: full second sign-in/recovery, lifetimes, one-use tokens, token errors, revision conflicts, separate cookies/codes, browser/purpose substitution, reissue/revoke, concurrent acceptance, acceptance versus revoke/reissue, restart, rollback before session commit and lost successful response. The household suite verifies real PostgreSQL personal-resource permissions, partner corrections, replay/concurrent revision, family isolation, current membership, capacity and lock ordering. Races use barriers; lifetimes use controlled clocks.

All required local commands passed before initial publication, including the missing-DB rejection. Review and CI results are recorded separately and bound to the exact published SHA in the PR and delivery report. Local fixtures do not prove browser or production readiness.

## AC boundaries and handoff

| AC | Task-1.6 evidence | Downstream verification |
| --- | --- | --- |
| AC-001 | Backend bootstrap/invitation, two sign-ins, replay/expiry/capacity | SCR-003/004/005 and real Chrome/Arc |
| AC-077 | Separate entities, configurable cap, no partner1/partner2 | New roles/exit/replacement are outside MVP |
| AC-078 | Household reads, personal/shared policies, existing reserve service | Full budgets/goals, AI approval, notifications |
| AC-086 | Concurrent accounting corrections, actor, revision, replay | AI clarifications and full undo lifecycle |
| AC-087 | Connection management versus external-owner auth; admission regression | Secret/MFA IO and real adapters |
| AC-088 | Sign-in/recovery isolation; one membership outbox event | Delivery, read state and product notifications |
| AC-090 | Family isolation and trusted principal in implemented paths | Files, AI retrieval and complete background flows |
| AC-105 | Current-owner policy and scope-bypass rejection | Account ownership handler and FORM-03 |

Task-1.5 consumes ExternalOwnership and trusted session/membership for files and bank secrets; task-2.1 invokes Ownership.RequireChange before persisting scope changes. Budget/goal/AI tasks recheck current ownership, membership and revision; payload changes never assign actor. Task-7.1 implements invitation UI, safe token transfer and unknown-response recovery. The outbox consumer addresses member_joined to the other active member without duplicate delivery.

Production, live banking/MFA, browser UI E2E, files, AI retrieval and push delivery are not claimed as verified. SDD remains **Ready for development**; operational readiness is not established.
