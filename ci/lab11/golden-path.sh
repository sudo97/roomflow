#!/usr/bin/env bash
# ЛР11: golden path — CLI платформи rfctl (інд. 7), перевірка відповідності політикам (інд. 5),
# метрики платформи з дашбордом і правилами тривог (завдання 5).
source "$(dirname "$0")/../lib/common.sh"
E=$EVIDENCE/golden-path; mkdir -p "$E"
CONFTEST="docker run --rm -v $PWD:/w -w /w openpolicyagent/conftest:v0.55.0"
go build -o "$BIN/rfctl" ./tools/rfctl
W=/tmp/gp; rm -rf $W; mkdir -p $W

log "rfctl: створення сервісів за шаблоном"
s=$(date +%s%N)
rfctl new --name guest-reviews --lang go --db postgres --owner group:platform-team --description "Відгуки гостей" --out $W/guest-reviews -o json | tee "$E/new-go.json"
T_GO=$(( ($(date +%s%N) - s) / 1000000 ))
rfctl new --name price-forecast --lang python --db redis --out $W/price-forecast | tee "$E/new-python.txt"
(cd $W/guest-reviews && go vet ./... && go test -cover ./...) | tee "$E/generated-go-test.txt"
(cd $W/price-forecast && python3 -m unittest -v) 2>&1 | tee "$E/generated-python-test.txt"
s=$(date +%s)
docker build -q -t guest-reviews:0.1.0 $W/guest-reviews > /dev/null && docker build -q -t price-forecast:0.1.0 $W/price-forecast > /dev/null
T_BUILD=$(( $(date +%s) - s ))
docker run -d --name gr -p 18080:8080 guest-reviews:0.1.0 >/dev/null; wait_http http://localhost:18080/healthz 20
curl -s localhost:18080/metrics | tee "$E/generated-metrics.txt"; docker rm -f gr >/dev/null
(cd $W && find guest-reviews price-forecast -type f | sort) > "$E/generated-tree.txt"
echo "rfctl new: ${T_GO} ms; docker build обох сервісів: ${T_BUILD} s" | tee "$E/timings.txt"
endlog

log "rfctl: інші команди"
rfctl catalog validate catalog-info.yaml | tee "$E/catalog-validate.txt"
rfctl catalog validate $W/guest-reviews/catalog-info.yaml | tee -a "$E/catalog-validate.txt"
rfctl provision --kind postgres --name reviews-db --size small --namespace team-reviews | tee "$E/provision.yaml"
rfctl status -o json > "$E/status.json" || true
rfctl status | tee "$E/status.txt" || true
endlog

log "golden path compliance (OPA/conftest)"
$CONFTEST verify -p policies/golden-path | tee "$E/policy-tests.txt"
cp -r $W ./.gp-tmp
set +e
{
  for svc in guest-reviews price-forecast; do
    echo "== $svc"
    $CONFTEST test -p policies/golden-path --all-namespaces --parser dockerfile .gp-tmp/$svc/Dockerfile
    $CONFTEST test -p policies/golden-path --all-namespaces .gp-tmp/$svc/k8s/deployment.yaml .gp-tmp/$svc/catalog-info.yaml
  done
  echo "== roomflow (репозиторій)"
  $CONFTEST test -p policies/golden-path --all-namespaces --parser dockerfile Dockerfile
  $CONFTEST test -p policies/golden-path --all-namespaces catalog-info.yaml deploy/k8s/bluegreen/deployments.yaml
  echo "== ci/lab8/Dockerfile.naive (очікувано не відповідає)"
  $CONFTEST test -p policies/golden-path --all-namespaces --parser dockerfile ci/lab8/Dockerfile.naive
} 2>&1 | tee "$E/compliance.txt"
set -e
rm -rf ./.gp-tmp
endlog

log "метрики платформи"
docker compose -f deploy/platform-metrics/docker-compose.yml up -d --quiet-pull
wait_http http://localhost:9103/metrics 180; wait_http http://localhost:3000/api/health 120
# реальні події цього запуску (поверх демонстраційної історії за 30 днів)
curl -s -X POST localhost:9103/events -d "[{\"type\":\"service_created\",\"template\":\"roomflow-service\",\"language\":\"go\"},{\"type\":\"service_created\",\"template\":\"roomflow-service\",\"language\":\"python\"},{\"type\":\"request\",\"channel\":\"self-service\"},{\"type\":\"request\",\"channel\":\"self-service\"}]"
sleep 40
for q in platform:services_created:total platform:self_service_adoption:ratio platform:provisioning_duration_seconds:p90 platform_developer_satisfaction_score; do
  echo "$q = $(curl -s --data-urlencode "query=$q" localhost:9090/api/v1/query | jq -r '.data.result[0].value[1]')"
done | tee "$E/platform-metrics.txt"
curl -s localhost:9090/api/v1/alerts | jq -r '.data.alerts[] | "\(.labels.alertname) \(.state) \(.annotations.summary)"' | tee "$E/platform-alerts.txt"
shoot "$E/platform-dashboard.png" "http://localhost:3000/d/platform/platform?orgId=1&kiosk" 6000 1600 900 || true
shoot "$E/platform-alerts.png" "http://localhost:9090/alerts" 3000 || true
docker compose -f deploy/platform-metrics/docker-compose.yml down -v >/dev/null
endlog
section "Golden path compliance" "$E/compliance.txt"
section "Метрики платформи" "$E/platform-metrics.txt"
