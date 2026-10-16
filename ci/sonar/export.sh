#!/usr/bin/env bash
# Експорт метрик, проблем і статусу воріт якості у JSON для звіту.
set -euo pipefail
OUT="$1"; mkdir -p "$OUT"
api() { curl -sS -u "admin:${SONAR_ADMIN_PASSWORD}" "$@"; }
M=bugs,vulnerabilities,code_smells,security_hotspots,coverage,duplicated_lines_density,ncloc,complexity,cognitive_complexity,sqale_index,sqale_debt_ratio,reliability_rating,security_rating,sqale_rating,security_hotspots_reviewed
api "$SONAR_HOST_URL/api/measures/component?component=roomflow&metricKeys=$M" > "$OUT/measures.json"
api "$SONAR_HOST_URL/api/issues/search?componentKeys=roomflow&resolved=false&ps=500" > "$OUT/issues.json"
api "$SONAR_HOST_URL/api/hotspots/search?project=roomflow&ps=100" > "$OUT/hotspots.json"
api "$SONAR_HOST_URL/api/qualitygates/project_status?projectKey=roomflow" > "$OUT/quality_gate.json"
jq -r '.component.measures[]|"\(.metric)=\(.value)"' "$OUT/measures.json" | sort | tee "$OUT/measures.txt"
jq -r '.issues[]|"\(.severity)\t\(.type)\t\(.rule)\t\(.component|sub("roomflow:";""))\t\(.line // "-")\t\(.message)"' "$OUT/issues.json" | tee "$OUT/issues.tsv"
jq -r '.projectStatus.status' "$OUT/quality_gate.json"
