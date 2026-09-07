# OpenAPI ownership

`task-1.2` introduces the public HTTP contract. The only supported generation direction is:

1. `api/openapi.yaml` — reviewed root, with feature-owned `paths/`, `schemas/` and reusable `components.yaml`;
2. `api/oapi-codegen.yaml` — pinned generator configuration;
3. `scripts/generate-openapi.sh` — deterministic generation entry point;
4. `backend/internal/delivery/http/generated/openapi.gen.go` — generated output.
5. `web/src/api/generated/openapi.gen.ts` — generated TypeScript output.

The source, configuration, generator and both outputs are mandatory after `task-1.2`.
`make check-contracts` rejects incomplete or entirely missing state and verifies reproducibility.
Generated bindings are never edited manually.

`backend/cmd/contract-bundle` validates local references inside `api/`, preserves the registered
root component names, and assembles a temporary JSON document consumed by both generators.
Network references, escaping symlinks and unregistered external components are rejected.

`make bootstrap` installs the locked toolchains; `make generate-contracts` updates both outputs.
`make check-contracts` compares temporary generation and runs schema/converter contract tests.
OpenAPI 3.0.3 uses oapi-codegen 2.8.0, pinned by the backend Go tool directive, and
openapi-typescript 7.13.0. The latter requires TypeScript 5, so this directory has an isolated
TypeScript 5.9.3 tool package; the web application remains on TypeScript 6.0.3 and typechecks its output.

The source contract describes future handlers. It does not start a server, authenticate a session,
write a financial record, or connect to a bank. The Go boundary validates shape; application owners
must enforce authorization and state-dependent invariants transactionally. Fixtures contain synthetic data.

Decimal strings have at most 256 characters, no exponent, and no asset-specific display truncation.
Revisions are positive integers up to 9007199254740991 so JavaScript retains them exactly.
Commands are identified before submission by their UUIDv4 Idempotency-Key. Metadata lives as long
as the family; secret authentication ceremonies and upload bytes are excluded from command results.

See [RU implementation evidence](../spec/001-want-keep-mvp/evidence/task-1.2-domain-api.md)
and [EN implementation evidence](../spec/001-want-keep-mvp/evidence/task-1.2-domain-api.en.md).
