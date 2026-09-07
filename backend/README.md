# Want Keep backend

This module will contain the Want Keep modular monolith. Product behavior is not implemented in
the foundation build.

Feature-owned business code belongs under `internal/<feature>/domain` and application coordination
under `internal/<feature>/application`. Inbound delivery, storage implementations, provider
integrations and external gateways stay in their named outer-layer packages. The architecture test
rejects imports that point from stable inner layers toward those implementations.

`task-1.2` implements money, calendar, household, command and reporting value contracts. Money owns
the only permitted domain dependency on apd; transport conversion and schema validation live in
`internal/delivery/http/contract`. Generated Go interfaces describe the API but have no product
implementation or server wiring yet. Generated delivery bindings must never be edited by hand.
