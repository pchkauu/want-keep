#!/bin/sh
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
expected="$root/backend/internal/gateways/openai/runtime_contract.json"
temporary=$(mktemp)
trap 'rm -f "$temporary"' EXIT HUP INT TERM

python3 "$root/scripts/generate-openai-runtime.py" --output "$temporary"
if ! cmp -s "$expected" "$temporary"; then
  echo "OpenAI runtime contract is stale. Run: make generate-ai-runtime-contract" >&2
  diff -u "$expected" "$temporary" || true
  exit 1
fi
