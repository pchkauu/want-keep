#!/bin/sh
set -eu

repository_root=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
temporary_directory=$(mktemp -d "${TMPDIR:-/tmp}/want-keep-make-contracts.XXXXXX")
fake_go="$temporary_directory/go"
fake_goroot="$temporary_directory/goroot"
fake_npm="$temporary_directory/npm"

cleanup() {
  trap - EXIT HUP INT TERM
  rm -f "$fake_go" "$fake_npm" "$fake_goroot/bin/gofmt"
  rmdir "$fake_goroot/bin" "$fake_goroot" "$temporary_directory"
}
trap cleanup EXIT HUP INT TERM

mkdir -p "$fake_goroot/bin"

cat >"$fake_go" <<EOF
#!/bin/sh
if [ "\$1" = "env" ] && [ "\$2" = "GOROOT" ]; then
  printf '%s\\n' '$fake_goroot'
  exit 0
fi
exit 97
EOF
cat >"$fake_goroot/bin/gofmt" <<'EOF'
#!/bin/sh
exit 96
EOF
chmod 700 "$fake_go" "$fake_goroot/bin/gofmt"

if make -s -C "$repository_root" format-check-go GO="$fake_go" >/dev/null 2>&1; then
  echo "format-check-go ignored a pinned formatter failure." >&2
  exit 1
fi

cat >"$fake_npm" <<'EOF'
#!/bin/sh
if [ "$(pwd)" != "$EXPECTED_CWD" ]; then
  printf 'Unexpected E2E cwd: %s\\n' "$(pwd)" >&2
  exit 1
fi
if [ "$*" != "$EXPECTED_ARGUMENTS" ]; then
  printf 'Unexpected E2E arguments: %s\\n' "$*" >&2
  exit 1
fi
EOF
chmod 700 "$fake_npm"

fixture_web="$repository_root/scripts/fixtures/tooling/web"
EXPECTED_CWD="$fixture_web" \
EXPECTED_ARGUMENTS="exec playwright test -- e2e/tooling.spec.ts" \
  make -s -C "$repository_root" e2e SCENARIO=tooling E2E_WEB_DIR="$fixture_web" NPM="$fake_npm"
EXPECTED_CWD="$fixture_web" \
EXPECTED_ARGUMENTS="exec playwright test --" \
  make -s -C "$repository_root" e2e SCENARIO=all E2E_WEB_DIR="$fixture_web" NPM="$fake_npm"

echo "Make command contracts are valid."
