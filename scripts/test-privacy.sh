#!/bin/sh
set -eu
project="want-keep-privacy-$(date +%s)-$$"
compose_file=deploy/document-processor/integration.yaml
compose() { docker compose -p "$project" -f "$compose_file" "$@"; }
cleanup() { compose down --volumes --remove-orphans >/dev/null; }
trap cleanup EXIT HUP INT TERM
compose build document-processor privacy-test
compose up -d --wait database document-processor
processor_id=$(compose ps -q document-processor)
python3 scripts/check-document-isolation.py "$processor_id"
compose run --rm --no-deps privacy-test
python3 scripts/check-document-recovery.py "$processor_id"
python3 scripts/check-document-isolation.py "$processor_id"
compose run --rm --no-deps privacy-test -test.v -test.count=1 -test.timeout=2m -test.run 'TestPendingProcessorAndLeaseRecovery|TestProcessorCancellationDoesNotAcceptAndRemainsAvailable'
