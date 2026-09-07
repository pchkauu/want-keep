# OpenAPI ownership

`task-1.2` introduces the public HTTP contract. The only supported generation direction is:

1. `api/openapi.yaml` — reviewed source contract;
2. `api/oapi-codegen.yaml` — pinned generator configuration;
3. `scripts/generate-openapi.sh` — deterministic generation entry point;
4. `backend/internal/delivery/http/generated/openapi.gen.go` — generated output.

The source, configuration, generator and output must either be absent together before `task-1.2`,
or present together. `make check-contracts` rejects partial state and asks the generator to verify
reproducibility once the contract exists. Generated bindings are never edited manually.
