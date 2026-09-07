#!/bin/sh
set -eu

source_file="api/openapi.yaml"
config_file="api/oapi-codegen.yaml"
generator="scripts/generate-openapi.sh"
generated_file="backend/internal/delivery/http/generated/openapi.gen.go"
typescript_file="web/src/api/generated/openapi.gen.ts"

present=0
for required_path in "$source_file" "$config_file" "$generator" "$generated_file" "$typescript_file"; do
  if [ -f "$required_path" ]; then
    present=$((present + 1))
  fi
done

if [ "$present" -ne 5 ]; then
  echo "OpenAPI state is incomplete; source, config, generator and generated output must change together." >&2
  exit 2
fi

sh "$generator" --check
cd backend
go test ./internal/delivery/http/...
