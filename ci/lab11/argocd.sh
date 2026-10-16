#!/usr/bin/env bash
# ЛР11, завдання 4 та інд. 1: ArgoCD (auto-sync, self-heal) і ApplicationSet для staging/production.
source "$(dirname "$0")/../lib/common.sh"
E=$EVIDENCE/argocd; mkdir -p "$E"
install_helm
kind_up rf-argo
build_image roomflow:v1
docker build -q --target production --build-arg VERSION=v2 -t roomflow:v2 . >/dev/null
kind load docker-image roomflow:v1 roomflow:v2 --name rf-argo >/dev/null
helm repo add argo https://argoproj.github.io/argo-helm >/dev/null && helm repo update >/dev/null
helm install argocd argo/argo-cd --version 7.6.8 -n argocd --create-namespace --set dex.enabled=false --set notifications.enabled=false \
  --set 'configs.params.server\.insecure=true' --wait --timeout 6m > "$E/install.txt"

app_state() { kubectl -n argocd get application "$1" -o jsonpath='{.status.sync.status}/{.status.health.status}'; }
wait_app() { # app timeout
  local s; s=$(date +%s)
  until [ "$(app_state "$1")" = "Synced/Healthy" ]; do [ $(( $(date +%s) - s )) -gt "$2" ] && { echo "timeout: $(app_state "$1")"; return 1; }; sleep 3; done
  echo "$1 Synced/Healthy after $(( $(date +%s) - s )) s"
}

log "Application roomflow"
kubectl apply -f deploy/gitops/application.yaml
wait_app roomflow 300 | tee "$E/sync.txt"
kubectl -n rf-gitops get deploy,svc,cm > "$E/resources.txt"
endlog

log "self-heal: ручні зміни в кластері"
kubectl -n rf-gitops scale deploy/roomflow-blue --replicas=5
s=$(date +%s)
for _ in $(seq 1 120); do [ "$(kubectl -n rf-gitops get deploy roomflow-blue -o jsonpath='{.spec.replicas}')" = 2 ] && break; sleep 1; done
echo "scale 2->5: повернуто до $(kubectl -n rf-gitops get deploy roomflow-blue -o jsonpath='{.spec.replicas}') за $(( $(date +%s) - s )) с" | tee "$E/self-heal.txt"
kubectl -n rf-gitops delete service roomflow
s=$(date +%s)
for _ in $(seq 1 120); do kubectl -n rf-gitops get service roomflow >/dev/null 2>&1 && break; sleep 1; done
echo "видалений Service відновлено за $(( $(date +%s) - s )) с" | tee -a "$E/self-heal.txt"
kubectl -n rf-gitops patch configmap roomflow-flags --type merge -p '{"data":{"flags.json":"{\"price_breakdown\":{\"enabled\":true}}"}}'
s=$(date +%s)
for _ in $(seq 1 120); do kubectl -n rf-gitops get cm roomflow-flags -o jsonpath='{.data.flags\.json}' | grep -q '"enabled":false' && break; sleep 1; done
echo "ручне ввімкнення прапорця в ConfigMap скасовано за $(( $(date +%s) - s )) с (джерело правди — Git)" | tee -a "$E/self-heal.txt"
kubectl -n argocd get application roomflow -o jsonpath='{range .status.history[*]}{.id} {.revision} {.deployedAt}{"\n"}{end}' > "$E/history.txt"
endlog

log "ApplicationSet staging/production"
kubectl apply -f deploy/gitops/applicationset.yaml
sleep 10
wait_app roomflow-staging 300 | tee -a "$E/sync.txt"
wait_app roomflow-production 300 | tee -a "$E/sync.txt"
{
  kubectl -n argocd get applications -o custom-columns=NAME:.metadata.name,SYNC:.status.sync.status,HEALTH:.status.health.status,REVISION:.status.sync.revision
  for env in staging production; do echo "== rf-$env"; kubectl -n rf-$env get deploy,hpa,pdb,networkpolicy 2>/dev/null; done
} | tee "$E/applicationset.txt"
endlog

log "screenshots"
kubectl -n argocd port-forward svc/argocd-server 8088:80 >/dev/null 2>&1 &
PF=$!; sleep 5
PASS=$(kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath='{.data.password}' | base64 -d)
TOKEN=""
for _ in $(seq 1 10); do
  TOKEN=$(curl -skL localhost:8088/api/v1/session -d "{\"username\":\"admin\",\"password\":\"$PASS\"}" | jq -r '.token // empty' 2>/dev/null || true)
  [ -n "$TOKEN" ] && break; sleep 3
done
SHOT_COOKIE="argocd.token=$TOKEN" shoot "$E/argocd-applications.png" "http://localhost:8088/applications" 5000 || true
SHOT_COOKIE="argocd.token=$TOKEN" shoot "$E/argocd-roomflow.png" "http://localhost:8088/applications/argocd/roomflow?view=tree" 6000 1600 900 || true
kill $PF
endlog
section "ArgoCD self-heal" "$E/self-heal.txt"
section "ApplicationSet" "$E/applicationset.txt"
