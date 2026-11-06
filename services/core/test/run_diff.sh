#!/bin/sh
# Differential tests: the same client sessions against the reference core (wormix-core:ref, C#) and the Go
# core (wormix-core:go; REF_CORE_IMAGE / API_IMAGE override the references), each with fresh backends; the transcripts must be identical.
#   1. fake: deterministic, scripted API answers incl. edge cases (diff_fake.py + fakeapi.py)
#   2. real: realistic flows against a fresh reference API (diff_real.py)
# Run where the images exist, from the repository root: sh services/core/test/run_diff.sh [fake|real]
set -eu
TEST_DIR="$(cd "$(dirname "$0")" && pwd)"
export TEST_DIR
OUT="${OUT:-/tmp/core-diff}"
mkdir -p "$OUT"
which="${1:-all}"

run() {   # run <mode> <tag> <core image>
  mode=$1 tag=$2 image=$3 project="coretest-$1-$2"
  if [ "$mode" = fake ]; then file="$TEST_DIR/compose.fake.yml"; else file="$TEST_DIR/compose.yml"; fi
  CORE_IMAGE=$image docker compose -p "$project" -f "$file" up -d >/dev/null 2>&1
  if [ "$mode" = fake ]; then
    docker run --rm --network "${project}_default" -v "$TEST_DIR:/t:ro" python:3.12-alpine \
      python -u /t/diff_fake.py core http://fakeapi:8000 > "$OUT/$mode-$tag.txt" || true
  else
    docker run --rm --network "${project}_default" -v "$TEST_DIR:/t:ro" python:3.12-alpine \
      python -u /t/diff_real.py core http://api:8000/api/ > "$OUT/$mode-$tag.txt" || true
  fi
  docker compose -p "$project" -f "$file" logs core > "$OUT/$mode-$tag.core.log" 2>&1 || true
  docker compose -p "$project" -f "$file" down -v >/dev/null 2>&1
}

status=0
for mode in fake real; do
  [ "$which" = all ] || [ "$which" = "$mode" ] || continue
  run "$mode" ref "${REF_CORE_IMAGE:-wormix-core:ref}" &
  run "$mode" go wormix-core:go &
  wait
  echo "== $mode: $(grep -c '^==' "$OUT/$mode-ref.txt") cases (ref), $(grep -c '^==' "$OUT/$mode-go.txt") cases (go)"
  if [ ! -s "$OUT/$mode-ref.txt" ] || [ ! -s "$OUT/$mode-go.txt" ]; then
    echo "   NO OUTPUT (driver failed)"; status=1; continue
  fi
  if diff -u "$OUT/$mode-ref.txt" "$OUT/$mode-go.txt" > "$OUT/$mode.diff"; then
    echo "   identical"
  else
    echo "   DIFFERENT: see $OUT/$mode.diff"; status=1
  fi
done
exit $status
