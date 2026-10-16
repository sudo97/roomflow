#!/usr/bin/env bash
# ЛР7: розгортання стеку спостережуваності, генерація трафіку, інцидент із невдалим релізом,
# автоматичне вимірювання DORA (lead time, CFR, MTTR) і знімки дашбордів.
source "$(dirname "$0")/../lib/common.sh"
E=$EVIDENCE
OBS=deploy/observability
dc() { docker compose -f $OBS/docker-compose.yml "$@"; }
prom() { curl -s --data-urlencode "query=$1" http://localhost:9090/api/v1/query | jq -c '.data.result | map({metric: .metric, value: .value[1]})'; }
event() { curl -s -X POST http://localhost:9102/events -d "$1" >/dev/null; }
k6run() { # name rate duration
  docker run --rm --network host -v "$PWD/test/load/k6:/k6:ro" -v "$E:/out" -e BASE_URL=http://localhost:8080 -e RATE="$2" -e DURATION="$3" \
    grafana/k6:0.54.0 run --quiet --summary-export "/out/k6-$1.json" /k6/mixed.js > "$E/k6-$1.txt" 2>&1 || true
}

log "build and start stack"
build_image roomflow:obs
export ROOMFLOW_IMAGE=roomflow:obs GITHUB_REPOSITORY GITHUB_TOKEN
dc up -d --quiet-pull
for u in http://localhost:8080/healthz http://localhost:9090/-/ready http://localhost:3000/api/health http://localhost:3100/ready http://localhost:9093/-/ready http://localhost:9102/healthz; do
  wait_http "$u" 180
done
COMMIT_TS=$(git log -1 --format=%ct)
event "{\"type\":\"deployment\",\"commit_time\":$COMMIT_TS}"
endlog

log "normal traffic (2 min)"
k6run normal 30 2m
endlog

log "faulty release v2 (FAULT_RATE=0.3) -> alert -> rollback"
FAULT_RATE=0.3 COLOR=green dc up -d roomflow
event "{\"type\":\"deployment\",\"commit_time\":$(date +%s)}"
k6run faulty 30 150s &
LOAD=$!
s=$(date +%s)
for _ in $(seq 1 150); do
  curl -s http://localhost:9093/api/v2/alerts | jq -e 'map(select(.labels.alertname=="HighErrorRate" and .status.state=="active")) | length > 0' >/dev/null && break
  sleep 2
done
echo "HighErrorRate active after $(( $(date +%s) - s )) s" | tee "$E/incident.txt"
sleep 20
shoot "$E/grafana-incident.png" "http://localhost:3000/d/roomflow/roomflow?orgId=1&kiosk&refresh=10s&from=now-10m&to=now" 8000 1600 1300 || true
shoot "$E/prometheus-alerts-firing.png" "http://localhost:9090/alerts?state=firing" 3000 || true
FAULT_RATE=0 COLOR=blue dc up -d roomflow
echo "rollback at $(( $(date +%s) - s )) s" | tee -a "$E/incident.txt"
event "{\"type\":\"deployment\",\"commit_time\":$COMMIT_TS}"
wait $LOAD || true
for _ in $(seq 1 120); do
  [ "$(prom 'dora_open_incidents' | jq -r '.[0].value')" = "0" ] && break; sleep 3
done
echo "incident resolved (time to restore recorded by exporter) after $(( $(date +%s) - s )) s" | tee -a "$E/incident.txt"
endlog

log "recovery traffic (1 min) and evidence"
k6run recovery 30 1m
sleep 20
{
  for q in 'dora_deployments_total' 'dora_failed_deployments_total' 'dora:deployments:increase6h' 'dora:lead_time_seconds:avg_6h' 'dora:lead_time_seconds:p50_6h' \
           'dora:change_failure_ratio:6h' 'dora:restore_seconds:avg_6h' 'dora_deployment_frequency_per_day' 'dora_history_runs' \
           'roomflow:http_requests:rate1m' 'roomflow:http_latency_seconds:p95_1m' 'slo:availability:ratio_rate5m' 'max_over_time(roomflow:http_errors:ratio_rate1m[15m])' \
           'sum by (code) (increase(roomflow_http_requests_total[15m]))'; do
    echo "$q => $(prom "$q")"
  done
} | tee "$E/promql.txt"
curl -s http://localhost:9090/api/v1/targets | jq -r '.data.activeTargets[] | "\(.labels.job) \(.health) \(.lastScrapeDuration)"' | tee "$E/targets.txt"
curl -s http://localhost:9090/api/v1/rules | jq -r '.data.groups[] | .name as $g | .rules[] | "\($g) \(.type) \(.name) \(.health) \(.state // "")"' > "$E/rules.txt"
curl -s -G http://localhost:3100/loki/api/v1/query --data-urlencode 'query=sum by (status) (count_over_time({container="roomflow"} | json | status >= 400 [15m]))' | jq -c '.data.result' | tee "$E/logql-errors.txt"
curl -s -G http://localhost:3100/loki/api/v1/query_range --data-urlencode 'query={container="roomflow"} | json | level="ERROR"' --data-urlencode limit=5 | jq -r '.data.result[].values[][1]' | head -5 > "$E/logql-sample.txt" || true
dc logs alert-sink 2>/dev/null | grep ALERT | sed 's/.*ALERT //' | jq -c '{status, alerts: [.alerts[] | {alertname: .labels.alertname, status, startsAt, endsAt}]}' > "$E/alert-notifications.txt" || true
curl -s http://localhost:9102/metrics | grep '^dora_' > "$E/dora-metrics.txt"
endlog

log "screenshots"
shoot "$E/grafana-dashboard.png" "http://localhost:3000/d/roomflow/roomflow?orgId=1&kiosk&from=now-15m&to=now" 8000 1600 1300 || true
shoot "$E/prometheus-rules.png" "http://localhost:9090/rules" 3000 || true
shoot "$E/prometheus-targets.png" "http://localhost:9090/targets" 3000 || true
shoot "$E/alertmanager.png" "http://localhost:9093/#/alerts" 3000 || true
endlog
dc ps > "$E/compose-ps.txt"
dc down -v >/dev/null
section "Інцидент" "$E/incident.txt"
section "PromQL" "$E/promql.txt"
