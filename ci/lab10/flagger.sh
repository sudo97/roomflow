#!/usr/bin/env bash
# ЛР10, завдання 5: автоматизований Canary (Flagger + ingress-nginx + Prometheus):
# успішний реліз проходить етапи 10%..50% і промотується; реліз із помилками автоматично відкочується.
source "$(dirname "$0")/lib.sh"
E=$EVIDENCE/flagger; mkdir -p "$E"
cluster rf-flagger
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx >/dev/null
helm repo add flagger https://flagger.app >/dev/null
helm repo update >/dev/null

log "ingress-nginx + flagger + prometheus + loadtester"
helm upgrade -i ingress-nginx ingress-nginx/ingress-nginx --version 4.11.3 -n ingress-nginx --create-namespace \
  --set controller.metrics.enabled=true \
  --set-string controller.podAnnotations."prometheus\.io/scrape"=true \
  --set-string controller.podAnnotations."prometheus\.io/port"=10254 \
  --set controller.service.type=ClusterIP --set controller.admissionWebhooks.enabled=false --wait --timeout 5m >/dev/null
helm upgrade -i flagger flagger/flagger -n ingress-nginx --set prometheus.install=true --set meshProvider=nginx --wait --timeout 5m >/dev/null
kubectl create namespace rf-flagger >/dev/null
helm upgrade -i flagger-loadtester flagger/loadtester -n rf-flagger --wait --timeout 5m >/dev/null
helm list -A > "$E/helm-releases.txt"
endlog

wait_phase() { # phase timeout_s
  local s; s=$(date +%s)
  until [ "$(kubectl -n rf-flagger get canary roomflow -o jsonpath='{.status.phase}')" = "$1" ]; do
    [ $(( $(date +%s) - s )) -gt "$2" ] && { echo "timeout waiting for $1"; return 1; }
    sleep 5
  done
  echo "$1 after $(( $(date +%s) - s )) s"
}
watch_weights() { # file: стан canary кожні 5 с
  (while true; do echo "$(date +%T) $(kubectl -n rf-flagger get canary roomflow -o jsonpath='{.status.phase} weight={.status.canaryWeight} failed={.status.failedChecks}')"; sleep 5; done) > "$1" &
  WATCH=$!
}

log "ініціалізація"
kubectl apply -f deploy/k8s/flagger/app.yaml > "$E/apply.txt"
wait_phase Initialized 300 | tee "$E/phases.txt"
kubectl -n rf-flagger get deploy,svc,ingress > "$E/resources-initialized.txt"
endlog

log "успішний реліз v2"
watch_weights "$E/progress-good.txt"
kubectl -n rf-flagger set image deploy/roomflow roomflow=roomflow:v2 >/dev/null
kubectl -n rf-flagger set env deploy/roomflow COLOR=v2 >/dev/null
wait_phase Progressing 120 | tee -a "$E/phases.txt"
wait_phase Succeeded 600 | tee -a "$E/phases.txt"
kill $WATCH
endlog

log "реліз із помилками (FAULT_RATE=0.4) -> автоматичний відкат"
watch_weights "$E/progress-faulty.txt"
kubectl -n rf-flagger set env deploy/roomflow FAULT_RATE=0.4 COLOR=v3-faulty >/dev/null
wait_phase Progressing 120 | tee -a "$E/phases.txt"
wait_phase Failed 600 | tee -a "$E/phases.txt"
kill $WATCH
endlog
kubectl -n rf-flagger describe canary roomflow > "$E/canary-describe.txt"
kubectl -n rf-flagger get events --sort-by=.lastTimestamp | grep -i -E 'canary|flagger|roomflow' | tail -40 > "$E/events.txt" || true
kubectl -n ingress-nginx logs deploy/flagger --tail=200 | jq -r 'select(.canary=="roomflow.rf-flagger") | "\(.ts) \(.msg)"' > "$E/flagger-log.txt" 2>/dev/null || kubectl -n ingress-nginx logs deploy/flagger --tail=100 > "$E/flagger-log.txt"
kubectl -n rf-flagger get deploy roomflow-primary -o jsonpath='{.spec.template.spec.containers[0].image} {.spec.template.spec.containers[0].env}' > "$E/primary-after.txt"
section "Flagger: фази" "$E/phases.txt"
section "Flagger: журнал аналізу" "$E/flagger-log.txt"
