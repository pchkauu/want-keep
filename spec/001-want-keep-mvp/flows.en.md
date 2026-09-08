# Data flows

[Русский](flows.md)

## Import and AI review

```mermaid
flowchart LR
  P[Platform] --> G[API gateway / Playwright]
  G --> R[SourceRecord and coverage]
  R --> M[Mapping and domain validation]
  M --> L[Atomic ledger + outbox]
  L --> A[Revision-specific AI review]
  A --> V[Authorized command validation]
  V --> C[Category / substantiated link]
  V --> Q[Clarification]
  C --> D[Projections and dashboard]
  L --> D
  P --> S[BalanceSnapshot]
  S --> X[Reconciliation]
  L --> X
```

Retain SourceRecord even with incomplete mapping. Ledger operations can continue while AI waits; unknown financial values do not post. The dashboard separately shows source freshness, coverage and AI-review status.

## Receipt and late import

```mermaid
sequenceDiagram
  actor O as Member
  participant W as Web chat
  participant A as Application
  participant AI as OpenAI
  participant L as Ledger
  O->>W: Select account and send receipt
  W->>A: Message + attachmentId + accountId + idempotency
  A->>A: Authorize and validate file
  A->>AI: Authorized data without secrets
  AI-->>A: Structured proposal
  alt Exact data is missing
    A-->>W: Clarify specific fields
  else Document is irrelevant
    A-->>W: Explicit skip and reason
  else Data is established
    A->>L: Create or link transaction, expectedRevision
    A-->>W: Outcome and transaction link
  end
  A->>L: Late bank import
  L->>L: Exact idempotency + economic match validation
  L-->>W: One transaction, multiple evidence records
```

Equal amounts/times yield candidates, not uniqueness proof. Multiple purchases require clarification. A transaction change while AI responds makes the old proposal inapplicable.

## Plan and daily allowance

```mermaid
flowchart TD
  F[Confirmed funds by currency] --> K[Available limit]
  R[Unique goal reservations] --> K
  O[Outstanding obligations] --> K
  B[Remaining flexible budget] --> K
  E[Expected income dates] --> Q[Daily forecast and cash shortfalls]
  F --> Q
  R --> Q
  O --> Q
  B --> Q
  AI[AI proposal] --> U[Authorized member decision]
  U --> B
```

Received income changes actuals; a planned date does not create money. A fulfilled payment releases the corresponding reserved obligation.

## Mac backups

```mermaid
sequenceDiagram
  participant M as MacBook
  participant S as Application VPS
  participant P as Managed PostgreSQL
  participant F as Immutable attachments
  participant B as Local backup
  M->>S: Authorized hourly pull while reachable
  S->>P: Logical dump over private VPC
  S->>F: Inventory at consistent cutoff
  P-->>S: DB stream
  F-->>S: Attachments + manifest
  S-->>M: Restricted export stream
  M->>M: Verify parts and checksums
  M->>B: Atomically complete encrypted set
  M->>S: Acknowledge verified manifest
  Note over M,P: Unreachable Mac increases the age of the last complete copy
```

A partial set never replaces the last complete backup. Recovery keys must remain available outside the sole failed system. Retention is 48 hourly, 30 daily, 8 weekly and 12 monthly points within a 20 GiB cap; never delete the last complete set. The one-hour RPO depends on Mac availability and set verification. Restoring an old set into clean managed PostgreSQL does not automatically activate old bank sessions. Full contract and open runtime gates: [hosting evidence](evidence/hosting.en.md).

## Household permissions and views

```mermaid
flowchart LR
  U[Separate user session] --> M[Membership and household scope]
  M --> C[Command and current permissions]
  C --> L[Ledger: amount and actor]
  L --> A[Member and category allocations]
  A --> F[One household actual]
  A --> P[Individual views]
  F --> K[Household K per currency]
  K --> D[One allowance matrix]
  D --> P
```

## Task-2.4: from facts to one effect

```mermaid
flowchart LR
  A[Manual entry / trusted normalization] --> B[Household transaction and current rights]
  I[CommitPage: admission / generation / lease] --> B
  B --> C[D-39 idempotency]
  C --> D{Proven correspondence?}
  D -->|Yes| E[Validate composition and field protections]
  D -->|Probable candidate only| F[Retain waiting and evidence without a second effect]
  F --> G[Member: link or separate with revisions]
  G --> E
  E --> H[Decision + journal revisions + projections + review/outbox]
  H --> J[Command outcome / checkpoint]
  H --> K[History and compound undo]
```

Original statuses and dates remain in history. Incomplete search and waiting produce matching_unresolved; source observations stay separate. Stale import jobs enter quarantine before this flow. Bank IO and document recognition are connected by their owning tasks.

## Task-3.2: connector ingestion page

```mermaid
sequenceDiagram
  participant J as Validated sync job
  participant C as API/Browser collector
  participant E as EvidenceStore
  participant G as Admission gate
  participant A as Accounts/Ledger
  participant Q as Quarantine
  J->>G: BeforeRead(binding, revision, generation, lease)
  G->>C: Server-issued request without household/actor/internal IDs
  C-->>G: Exact echo + evidence + typed records/coverage
  G->>E: Persist raw evidence before financial transaction
  alt binding/revision/generation/lease/cursor are current
    G->>A: CommitPage: resolve accounts + source revisions + observations/postings
    A-->>G: Audit/outbox/checkpoint atomically
  else result is stale
    G->>Q: Evidence reference + safe reason
  end
```

A page is self-contained: every supported account used by a balance or posting has an account descriptor on that page. A later page repeats the descriptor and prior cursor; replay creates no account/opening/financial effect. Failure on page two never advances its cursor, while the confirmed first page stays committed with partial coverage.
