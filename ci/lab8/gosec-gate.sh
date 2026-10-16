#!/usr/bin/env bash
# SAST-ворота gosec: блокує нові знахідки рівня MEDIUM+ (severity і confidence), яких немає в baseline.
# Baseline (ci/security/gosec-baseline.json) містить прийняті ризики з обґрунтуванням.
set -euo pipefail
OUT=${1:-/tmp/gosec.json}
command -v gosec >/dev/null || go install github.com/securego/gosec/v2/cmd/gosec@v2.21.4
GOSEC=$(command -v gosec || echo "$(go env GOPATH)/bin/gosec")
"$GOSEC" -quiet -no-fail -fmt json -out "$OUT" -severity medium -confidence medium -exclude-dir=test ./... || true
python3 - "$OUT" ci/security/gosec-baseline.json <<'PY'
import json, os, sys
found = json.load(open(sys.argv[1])).get("Issues") or []
base = json.load(open(sys.argv[2]))["accepted"]
accepted = {(b["rule_id"], b["file"]) for b in base}
new, known = [], []
for i in found:
    rel = os.path.relpath(i["file"], os.getcwd())
    item = (i["rule_id"], rel)
    (known if item in accepted else new).append(f'{i["rule_id"]} {i["severity"]}/{i["confidence"]} {rel}:{i["line"]} {i["details"]}')
print(f"gosec: {len(found)} findings, {len(known)} accepted in baseline, {len(new)} new")
for k in known: print("  baseline:", k)
for n in new: print("  NEW:", n)
sys.exit(1 if new else 0)
PY
