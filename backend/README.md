# Want Keep backend

This module will contain the Want Keep modular monolith. Product behavior is not implemented in
the foundation build.

Feature-owned business code belongs under `internal/<feature>/domain` and application coordination
under `internal/<feature>/application`. Inbound delivery, storage implementations, provider
integrations and external gateways stay in their named outer-layer packages. The architecture test
rejects imports that point from stable inner layers toward those implementations.

The OpenAPI contract is introduced by `task-1.2`. Generated delivery bindings must never be edited
by hand.
