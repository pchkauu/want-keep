#!/bin/sh
set -eu

repository_root=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
mode=${1:-generate}
if [ "$mode" != generate ] && [ "$mode" != --check ]; then
  echo "Usage: generate-ingestion-contracts.sh [--check]" >&2
  exit 2
fi
temporary_directory=$(mktemp -d "${TMPDIR:-/tmp}/want-keep-ingestion-contract.XXXXXX")
cleanup() {
  trap - EXIT HUP INT TERM
  rm -f "$temporary_directory/ingestion.gen.go" "$temporary_directory/ingestion.gen.ts"
  rmdir "$temporary_directory"
}
trap cleanup EXIT HUP INT TERM

source_file="$repository_root/collector/contracts/v10/ingestion.openapi.yaml"
config_file="$repository_root/collector/contracts/v10/oapi-codegen.yaml"
go_output="$repository_root/backend/internal/integrations/contract/generated/ingestion.gen.go"
ts_output="$repository_root/collector/src/contracts/generated/ingestion.gen.ts"

cd "$repository_root/backend"
go tool oapi-codegen --config "$config_file" -o "$temporary_directory/ingestion.gen.go" "$source_file"
cd "$repository_root/api"
./node_modules/.bin/openapi-typescript "$source_file" --output "$temporary_directory/ingestion.gen.ts"
"$repository_root/web/node_modules/.bin/prettier" --write "$temporary_directory/ingestion.gen.ts" --no-config --no-editorconfig

if [ "$mode" = --check ]; then
  cmp "$temporary_directory/ingestion.gen.go" "$go_output"
  cmp "$temporary_directory/ingestion.gen.ts" "$ts_output"
  echo "Go and TypeScript ingestion outputs are reproducible."
else
  mkdir -p "$(dirname "$go_output")" "$(dirname "$ts_output")"
  cp "$temporary_directory/ingestion.gen.go" "$go_output"
  cp "$temporary_directory/ingestion.gen.ts" "$ts_output"
fi
