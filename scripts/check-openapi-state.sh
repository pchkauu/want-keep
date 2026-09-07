#!/bin/sh
set -eu

source_file="api/openapi.yaml"
config_file="api/oapi-codegen.yaml"
generator="scripts/generate-openapi.sh"
generated_file="backend/internal/delivery/http/generated/openapi.gen.go"
generated_directory="backend/internal/delivery/http/generated"

present=0
for required_path in "$source_file" "$config_file" "$generator" "$generated_file"; do
  if [ -f "$required_path" ]; then
    present=$((present + 1))
  fi
done

if [ "$present" -eq 0 ] && [ ! -d "$generated_directory" ]; then
  echo "OpenAPI is intentionally not materialized before task-1.2."
  exit 0
fi

if [ "$present" -ne 4 ]; then
  echo "OpenAPI state is incomplete; source, config, generator and generated output must change together." >&2
  exit 2
fi

sh "$generator" --check
