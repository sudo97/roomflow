#!/usr/bin/env bash
# ЛР12, завдання 5: Prometheus Operator (kube-prometheus-stack), ServiceMonitor для сервісів,
# PrometheusRule (SLI/SLO, тривоги) і дашборд Grafana як ConfigMap.
source "$(dirname "$0")/lib.sh"
E=$EVIDENCE/monitoring; mkdir -p "$E"
NS=rf-mon
install_helm
kind_up rf-mon
mesh_images rf-mon
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts >/dev/null && helm repo update >/dev/null
helm install kps prometheus-community/kube-prometheus-stack --version 65.1.1 -n monitoring --create-namespace \
  --set grafana.adminPassword=admin --set alertmanager.enabled=true \
  --set prometheus.prometheusSpec.evaluationInterval=15s --set prometheus.prometheusSpec.scrapeInterval=15s --wait --timeout 8m >/dev/null
kubectl create namespace $NS >/dev/null
sed -e 's/namespace: rf-mesh/namespace: rf-mon/' -e '/CACHE_REDIS_ADDR/d' -e '/OTEL_/d' deploy/mesh/core/apps.yaml | kubectl apply -f - >/dev/null
sed 's/namespace: rf-mesh/namespace: rf-mon/' deploy/mesh/core/namespace.yaml | grep -v istio-injection | kubectl apply -f - >/dev/null
kubectl -n $NS rollout status deploy/backend --timeout=180s; kubectl -n $NS rollout status deploy/frontend --timeout=180s
kubectl apply -f deploy/mesh/monitoring/servicemonitors.yaml
python3 ci/lab12/prometheusrule.py deploy/observability/prometheus/recording.rules.yml deploy/observability/prometheus/alerts.rules.yml | kubectl apply -f -
kubectl -n monitoring create configmap roomflow-dashboard --from-file=roomflow.json=deploy/observability/grafana/dashboards/roomflow.json --dry-run=client -o yaml \
  | kubectl label --local -f - grafana_dashboard=1 -o yaml | kubectl apply -f -

log "навантаження (у т.ч. помилки) і перевірки"
fortio_pod $NS
kubectl -n $NS exec fortio -- fortio curl -quiet -H 'X-Tenant-ID: demo-hotel' -payload '{"id":"r1","name":"Suite","capacity":2,"price":200000}' http://backend:8080/api/rooms >/dev/null 2>&1 || true
kubectl -n $NS exec fortio -- fortio load -quiet -c 4 -qps 30 -t 150s -H 'X-Tenant-ID: demo-hotel' 'http://frontend:8081/api/rooms/r1/availability?from=2027-01-01&to=2027-02-01' > "$E/load.txt" 2>&1 &
L=$!
sleep 60
kubectl -n $NS set env deploy/backend FAULT_RATE=0.2 >/dev/null   # деградація для демонстрації тривог SLO
wait $L || true
kubectl -n monitoring port-forward svc/kps-kube-prometheus-stack-prometheus 9090:9090 >/dev/null 2>&1 &
PF1=$!
kubectl -n monitoring port-forward svc/kps-grafana 3000:80 >/dev/null 2>&1 &
PF2=$!
sleep 45
curl -s localhost:9090/api/v1/targets | jq -r '.data.activeTargets[] | select(.labels.namespace=="rf-mon") | "\(.labels.job) \(.labels.pod) \(.health)"' | tee "$E/targets.txt"
for q in 'roomflow:http_requests:rate1m' 'roomflow:http_errors:ratio_rate1m' 'roomflow:http_latency_seconds:p95_1m' 'slo:availability:ratio_rate5m' 'slo:error_budget_burn_rate:1m'; do
  echo "$q = $(curl -s --data-urlencode "query=$q" localhost:9090/api/v1/query | jq -r '.data.result[0].value[1]')"
done | tee "$E/sli.txt"
curl -s localhost:9090/api/v1/alerts | jq -r '.data.alerts[] | select(.labels.namespace=="rf-mon" or .labels.alertname=="HighErrorRate" or .labels.alertname=="ErrorBudgetFastBurn") | "\(.labels.alertname) \(.state)"' | sort -u | tee "$E/alerts.txt"
SHOT_AUTH="Basic $(printf admin:admin | base64)" shoot "$E/grafana-slo.png" "http://localhost:3000/d/roomflow/roomflow?orgId=1&kiosk&from=now-10m&to=now" 8000 1600 1300 || true
shoot "$E/prometheus-alerts.png" "http://localhost:9090/alerts" 3000 || true
shoot "$E/prometheus-targets.png" "http://localhost:9090/targets?search=rf-mon" 3000 || true
kill $PF1 $PF2
endlog
section "ServiceMonitor targets" "$E/targets.txt"
section "SLI/SLO" "$E/sli.txt"
