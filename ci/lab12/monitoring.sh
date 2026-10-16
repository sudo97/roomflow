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
# навантаження триває 6 хв: усі запити до Prometheus і знімки екрана робляться ПІД навантаженням,
# інакше rate() за 1 хв після його завершення порожній (так було в першому прогоні: SLI = null)
# /api/rooms, а не календар r1: стан backend зберігається в пам'яті кожної репліки, тож номер, створений
# на одній репліці, на іншій дає 404, і fortio переривав навантаження на першій же помилці
kubectl -n $NS exec fortio -- fortio load -quiet -c 4 -qps 30 -t 360s -H 'X-Tenant-ID: demo-hotel' http://frontend:8081/api/rooms > "$E/load.txt" 2>&1 &
L=$!
kubectl -n monitoring port-forward svc/kps-kube-prometheus-stack-prometheus 9090:9090 >/dev/null 2>&1 &
PF1=$!
kubectl -n monitoring port-forward svc/kps-grafana 3000:80 >/dev/null 2>&1 &
PF2=$!
sleep 75
q() { curl -s --data-urlencode "query=$1" localhost:9090/api/v1/query | jq -r '.data.result[0].value[1] // "null"'; }
echo "до деградації: error ratio = $(q roomflow:http_errors:ratio_rate1m), availability(5m) = $(q slo:availability:ratio_rate5m)" | tee "$E/sli-baseline.txt"
kubectl -n $NS set env deploy/backend FAULT_RATE=0.2 >/dev/null   # деградація для демонстрації тривог SLO
kubectl -n $NS rollout status deploy/backend --timeout=120s >/dev/null
T0=$(date +%s)
for _ in $(seq 1 40); do
  st=$(curl -s localhost:9090/api/v1/alerts | jq -r '[.data.alerts[] | select(.labels.alertname=="HighErrorRate") | .state][0] // "none"')
  [ "$st" = firing ] && break; sleep 5
done
echo "HighErrorRate: стан $st через $(( $(date +%s) - T0 )) с після деградації" | tee "$E/alert-latency.txt"
sleep 30   # щоб ErrorBudgetFastBurn (burn rate за 5 хв) теж встиг перейти у firing
curl -s localhost:9090/api/v1/targets | jq -r '.data.activeTargets[] | select(.labels.namespace=="rf-mon") | "\(.labels.job) \(.labels.pod) \(.health)"' | tee "$E/targets.txt"
for q in 'roomflow:http_requests:rate1m' 'roomflow:http_errors:ratio_rate1m' 'roomflow:http_latency_seconds:p95_1m' 'slo:availability:ratio_rate5m' 'slo:error_budget_burn_rate:1m'; do
  echo "$q = $(q "$q")"
done | tee "$E/sli.txt"
curl -s localhost:9090/api/v1/alerts | jq -r '.data.alerts[] | select(.labels.namespace=="rf-mon" or .labels.alertname=="HighErrorRate" or .labels.alertname=="ErrorBudgetFastBurn") | "\(.labels.alertname) \(.state)"' | sort -u | tee "$E/alerts.txt"
SHOT_AUTH="Basic $(printf admin:admin | base64)" shoot "$E/grafana-slo.png" "http://localhost:3000/d/roomflow/roomflow?orgId=1&kiosk&from=now-10m&to=now" 8000 1600 1300 || true
shoot "$E/prometheus-alerts.png" "http://localhost:9090/alerts" 3000 || true
shoot "$E/prometheus-targets.png" "http://localhost:9090/targets?search=rf-mon" 3000 || true
kill $PF1 $PF2
wait $L || true
endlog
section "ServiceMonitor targets" "$E/targets.txt"
section "SLI/SLO" "$E/sli.txt"
