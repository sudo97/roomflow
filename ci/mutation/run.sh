#!/usr/bin/env bash
# Мутаційне тестування пакетів RoomFlow (Gremlins) для однієї або кількох версій коду.
# Використання: run.sh "<refs>"  -> звіти в /tmp/evidence/<sha>/
set -uo pipefail
REFS=${1:-HEAD}
PKGS="internal/booking internal/store internal/httpapi"
ORIG=$(git rev-parse HEAD)
mkdir -p /tmp/evidence
for ref in $REFS; do
  git checkout -q "$ref" || continue
  sha=$(git rev-parse --short HEAD); out=/tmp/evidence/$sha; mkdir -p "$out"
  git log -1 --format='%H %s' > "$out/commit.txt"
  go test -coverprofile=/tmp/c.out ./internal/... >/dev/null && go tool cover -func=/tmp/c.out | tail -1 > "$out/coverage.txt"
  for p in $PKGS; do
    name=${p##*/}
    gremlins unleash --output "$out/gremlins-$name.json" "./$p" > "$out/gremlins-$name.txt" 2>&1
    echo "== $p"; tail -n 12 "$out/gremlins-$name.txt"
  done
  # зведена таблиця по пакетах
  {
    printf "package\tkilled\tlived\tnot_covered\ttimed_out\tnot_viable\tefficacy\tmutant_coverage\n"
    for p in $PKGS; do
      f="$out/gremlins-${p##*/}.json"; [ -s "$f" ] || continue
      jq -r --arg p "$p" '[$p, .mutants_killed, .mutants_lived, .mutants_not_covered, (.mutants_timed_out // 0), .mutants_not_viable, .test_efficacy, .mutations_coverage] | @tsv' "$f"
    done
  } > "$out/summary.tsv"
  # усі вижилі мутанти з координатами
  for p in $PKGS; do
    f="$out/gremlins-${p##*/}.json"; [ -s "$f" ] || continue
    jq -r '.files[]? as $f | $f.mutations[]? | select(.status=="LIVED" or .status=="NOT COVERED") | "\(.status)\t\($f.file_name):\(.line):\(.column)\t\(.type)"' "$f"
  done > "$out/survivors.tsv"
  cat "$out/summary.tsv"; echo; cat "$out/survivors.tsv"
  git checkout -q -f "$ORIG"
done
