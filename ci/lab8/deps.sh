#!/usr/bin/env bash
# ЛР8, завдання 1: сканування залежностей (govulncheck — аналог Snyk/Safety для Go) і файлової системи (Trivy).
source "$(dirname "$0")/../lib/common.sh"
E=$EVIDENCE
# поточний інструментарій — остання стабільна версія Go (з виправленнями стандартної бібліотеки);
# govulncheck теж збирається нею, щоб коректно аналізувати новішу stdlib
GOTOOLCHAIN=$(curl -fsS 'https://go.dev/VERSION?m=text' | head -1); export GOTOOLCHAIN
go install golang.org/x/vuln/cmd/govulncheck@latest
GV=$(go env GOPATH)/bin/govulncheck

log "govulncheck: застаріла версія інструментарію (go1.24.0) — демонстрація виявлення вразливостей"
set +e
GOTOOLCHAIN=go1.24.0 $GV ./... > "$E/govulncheck-go1.24.0.txt" 2>&1; echo "exit code: $?" >> "$E/govulncheck-go1.24.0.txt"
GOTOOLCHAIN=go1.24.0 $GV -format json ./... > "$E/govulncheck-go1.24.0.json" 2>/dev/null
echo "toolchain: $GOTOOLCHAIN" > "$E/govulncheck-current.txt"
$GV ./... >> "$E/govulncheck-current.txt" 2>&1; CUR=$?; echo "exit code: $CUR" >> "$E/govulncheck-current.txt"
$GV -format sarif ./... > govulncheck.sarif 2>/dev/null
set -e
python3 - "$E/govulncheck-go1.24.0.json" > "$E/govulncheck-summary.txt" <<'PY'
import json, sys
osv, called = {}, set()
dec = json.JSONDecoder(); s = open(sys.argv[1]).read(); i = 0
while i < len(s):
    while i < len(s) and s[i].isspace(): i += 1
    if i >= len(s): break
    obj, i = dec.raw_decode(s, i)
    if "osv" in obj: osv[obj["osv"]["id"]] = obj["osv"]
    if "finding" in obj and obj["finding"].get("trace") and obj["finding"]["trace"][0].get("function"):
        called.add(obj["finding"]["osv"])
print(f"go1.24.0: {len(osv)} відомих вразливостей у модулях/stdlib, {len(called)} досяжні з коду RoomFlow")
for v in sorted(called):
    o = osv[v]; aff = o["affected"][0]
    fixed = [e.get("fixed") for r in aff.get("ranges", []) for e in r.get("events", []) if e.get("fixed")]
    print(f"  {v}  {aff['package']['name']}  fixed in {fixed[-1] if fixed else '?'}  {o.get('summary','')[:90]}")
PY
cat "$E/govulncheck-summary.txt"
endlog

log "trivy fs (залежності, секрети, конфігурації)"
docker run --rm -v "$PWD:/src" -w /src aquasec/trivy:0.56.2 fs --quiet --scanners vuln,secret,misconfig --format table . > "$E/trivy-fs.txt" 2>&1 || true
docker run --rm -v "$PWD:/src" -w /src aquasec/trivy:0.56.2 fs --quiet --scanners vuln,secret,misconfig --format sarif -o trivy-fs.sarif . || true
endlog
section "govulncheck" "$E/govulncheck-summary.txt"
exit $CUR
