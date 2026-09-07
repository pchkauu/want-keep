#!/bin/sh
set -eu

repository_root=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
mode=${1:-generate}
if [ "$mode" != generate ] && [ "$mode" != --check ]; then
  echo "Usage: generate-openapi.sh [--check]" >&2
  exit 2
fi
temporary_directory=$(mktemp -d "${TMPDIR:-/tmp}/want-keep-openapi.XXXXXX")
cleanup() {
  trap - EXIT HUP INT TERM
  rm -f "$temporary_directory/openapi.gen.go" "$temporary_directory/openapi.gen.ts" "$temporary_directory/openapi.json"
  rmdir "$temporary_directory"
}
trap cleanup EXIT HUP INT TERM

cd "$repository_root/backend"
go run ./cmd/contract-bundle ../api/openapi.yaml > "$temporary_directory/openapi.json"
go tool oapi-codegen --config ../api/oapi-codegen.yaml -o "$temporary_directory/openapi.gen.go" "$temporary_directory/openapi.json"
cd "$repository_root/api"
./node_modules/.bin/openapi-typescript "$temporary_directory/openapi.json" --output "$temporary_directory/openapi.gen.ts"
"$repository_root/web/node_modules/.bin/prettier" --write "$temporary_directory/openapi.gen.ts" --no-config --no-editorconfig
cd "$repository_root"

go_output=backend/internal/delivery/http/generated/openapi.gen.go
ts_output=web/src/api/generated/openapi.gen.ts
if [ "$mode" = --check ]; then
  cmp "$temporary_directory/openapi.gen.go" "$go_output"
  cmp "$temporary_directory/openapi.gen.ts" "$ts_output"
  echo "Go and TypeScript OpenAPI outputs are reproducible."
else
  mkdir -p "$(dirname "$go_output")" "$(dirname "$ts_output")"
  cp "$temporary_directory/openapi.gen.go" "$go_output"
  cp "$temporary_directory/openapi.gen.ts" "$ts_output"
fi
