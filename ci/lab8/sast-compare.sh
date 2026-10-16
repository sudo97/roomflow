#!/usr/bin/env bash
# ЛР8, індивідуальне завдання 2: порівняння SAST-інструментів (gosec, Semgrep; CodeQL — окремим job)
# на трьох версіях коду з відомими (навмисно внесеними) вразливостями.
# Використання: sast-compare.sh NAME REF  -> $EVIDENCE/sast/NAME-{gosec,semgrep}.json
source "$(dirname "$0")/../lib/common.sh"
NAME=$1; REF=$2
OUT=$EVIDENCE/sast; mkdir -p "$OUT"
DIR=/tmp/src-$NAME
git worktree add -q -f "$DIR" "$REF"
go install github.com/securego/gosec/v2/cmd/gosec@v2.21.4
GOSEC=$(go env GOPATH)/bin/gosec
s=$(date +%s%N)
(cd "$DIR" && $GOSEC -quiet -no-fail -fmt json -out "$OUT/$NAME-gosec.json" ./... || true)
echo "$NAME gosec $(( ($(date +%s%N) - s) / 1000000 ))" >> "$OUT/durations.txt"
s=$(date +%s%N)
docker run --rm -v "$DIR:/src" -w /src semgrep/semgrep:1.90.0 semgrep scan --quiet --metrics=off \
  --config p/golang --config p/secrets --config p/sql-injection --json -o "/src/semgrep.json" . || true
cp "$DIR/semgrep.json" "$OUT/$NAME-semgrep.json" || echo '{"results":[]}' > "$OUT/$NAME-semgrep.json"
echo "$NAME semgrep $(( ($(date +%s%N) - s) / 1000000 ))" >> "$OUT/durations.txt"
git worktree remove -f "$DIR"
