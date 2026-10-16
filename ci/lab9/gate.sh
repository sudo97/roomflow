#!/usr/bin/env bash
# ЛР9, індивідуальне завдання 4: ворота продуктивності в CI/CD.
# Короткий тест availability (50 VU) на реалістичному обсязі даних; поріг p95 < P95_MS.
# Порушення порогу (k6 exit code 99) зупиняє конвеєр і блокує розгортання.
# Використання: gate.sh [REF]  (без REF — поточний коміт)
source "$(dirname "$0")/../lib/common.sh"
REF=${1:-}
P95_MS=${P95_MS:-100}
E=$EVIDENCE/gate-${REF:-head}; mkdir -p "$E"; chmod 777 "$E"   # k6 у контейнері працює не від root
build_image roomflow:gate "$REF"
docker rm -f rf-gate >/dev/null 2>&1 || true
docker network create rf-gate-net >/dev/null 2>&1 || true
docker run -d --name rf-gate --network rf-gate-net --cpus 1 --memory 512m -p 8080:8080 roomflow:gate >/dev/null
wait_http http://localhost:8080/healthz 30
go run ./test/load/seed -url http://localhost:8080 -tenants 20 -rooms 10 -bookings 30000 | tee "$E/seed.txt"
set +e
docker run --rm --network rf-gate-net -e BASE_URL=http://rf-gate:8080 -v "$PWD/test/load/k6:/k6:ro" -v "$E:/out" grafana/k6:0.54.0 run --quiet \
  -e ENDPOINT=availability -e MAX_VUS=50 -e RAMP=20s -e HOLD=20s -e P95_MS="$P95_MS" \
  --summary-export /out/gate.json /k6/load.js > "$E/gate.txt" 2>&1
code=$?
grep -v 'level=warning' "$E/gate.txt" | tail -40
set -e
docker inspect rf-gate --format 'status={{.State.Status}} exit={{.State.ExitCode}} oom={{.State.OOMKilled}} restarts={{.RestartCount}}' | tee "$E/container-state.txt"
docker logs rf-gate 2>&1 | grep -v '"msg":"http_request"' | tail -60 > "$E/container-log.txt"
docker stats --no-stream rf-gate >> "$E/container-state.txt" 2>/dev/null || true
docker rm -f rf-gate >/dev/null
p95=$(jq -r '.metrics.http_req_duration["p(95)"]' "$E/gate.json")
echo "### Ворота продуктивності: p95 = ${p95} мс (поріг ${P95_MS} мс), k6 exit code ${code}" | summary
exit "$code"
