#!/usr/bin/env bash
# ЛР12, завдання 2: blue-green для frontend — Argo Rollouts + Istio VirtualService, керування через ArgoCD.
source "$(dirname "$0")/lib.sh"
E=$EVIDENCE/rollouts; mkdir -p "$E"
install_helm
kind_up rf-roll
mesh_images rf-roll
install_istio deploy/mesh/istio/istio-operator.yaml
kubectl create namespace argo-rollouts >/dev/null
kubectl apply -n argo-rollouts -f https://github.com/argoproj/argo-rollouts/releases/download/v1.7.2/install.yaml >/dev/null
curl -sSLo "$BIN/kubectl-argo-rollouts" https://github.com/argoproj/argo-rollouts/releases/download/v1.7.2/kubectl-argo-rollouts-linux-amd64 && chmod +x "$BIN/kubectl-argo-rollouts"
helm repo add argo https://argoproj.github.io/argo-helm >/dev/null && helm repo update >/dev/null
helm install argocd argo/argo-cd --version 7.6.8 -n argocd --create-namespace --set dex.enabled=false --set notifications.enabled=false --wait --timeout 6m >/dev/null
kubectl -n argo-rollouts rollout status deploy/argo-rollouts --timeout=180s
R() { kubectl argo rollouts -n rf-ui "$@"; }

log "ArgoCD Application (v1)"
kubectl apply -f deploy/mesh/rollouts-application.yaml
for _ in $(seq 1 100); do kubectl -n rf-ui get rollout frontend >/dev/null 2>&1 && break; sleep 3; done
kubectl -n rf-ui rollout status deploy/backend --timeout=300s
for _ in $(seq 1 100); do [ "$(kubectl -n rf-ui get rollout frontend -o jsonpath='{.status.phase}' 2>/dev/null)" = Healthy ] && break; sleep 3; done
R get rollout frontend --no-color | head -15
kubectl -n rf-ui run probe --image=curlimages/curl:8.10.1 --restart=Never --command -- sleep 3600 >/dev/null
kubectl -n rf-ui wait --for=condition=Ready pod/probe --timeout=180s >/dev/null
kubectl -n rf-ui exec probe -c probe -- curl -s -H 'X-Tenant-ID: demo-hotel' -X POST http://backend:8080/api/rooms -d '{"id":"r1","name":"Suite","capacity":2,"price":200000}' >/dev/null
ver() { kubectl -n rf-ui exec probe -c probe -- sh -c "for i in \$(seq 1 $2); do curl -s -m 2 $1/version; echo; done" | sed -n 's/.*"version":"\([^"]*\)".*/\1/p' | sort | uniq -c | tr '\n' ' '; echo; }
echo "v1 active: frontend -> $(ver http://frontend:8081 20)" | tee "$E/flow.txt"
endlog

release() { # tag
  kubectl -n argocd patch application rf-ui --type merge -p "{\"spec\":{\"source\":{\"kustomize\":{\"images\":[\"roomflow-fe=roomflow:$1\"]}}}}"
}

log "реліз v2 через GitOps: preview -> аналіз -> promote"
(while true; do kubectl -n rf-ui exec probe -c probe -- curl -s -o /dev/null -m 2 -w '%{http_code}\n' http://frontend:8081/version; sleep 0.1; done) > "$E/traffic-v2.log" 2>/dev/null &
T=$!
release fe-v2
for _ in $(seq 1 100); do [ "$(kubectl -n rf-ui get rollout frontend -o jsonpath='{.status.pauseConditions[0].reason}')" = "BlueGreenPause" ] && break; sleep 3; done
{
  echo "після синхронізації ArgoCD (пауза перед перемиканням):"
  echo "  active  -> $(ver http://frontend-active:8081 10)"
  echo "  preview -> $(ver http://frontend-preview:8081 10)"
  echo "  користувачі (frontend) -> $(ver http://frontend:8081 20)"
} | tee -a "$E/flow.txt"
R get rollout frontend --no-color > "$E/rollout-paused.txt" 2>&1 || true
s=$(date +%s)
R promote frontend
for _ in $(seq 1 60); do [ "$(kubectl -n rf-ui get rollout frontend -o jsonpath='{.status.phase}')" = "Healthy" ] && [ "$(ver http://frontend:8081 5 | grep -c v2)" = 1 ] && break; sleep 2; done
echo "promote -> v2 обслуговує трафік через $(( $(date +%s) - s )) с (передперемикальний аналіз: $(kubectl -n rf-ui get analysisrun -o jsonpath='{.items[-1:].status.phase}'))" | tee -a "$E/flow.txt"
echo "  користувачі (frontend) -> $(ver http://frontend:8081 20)" | tee -a "$E/flow.txt"
sleep 5; kill $T
echo "відповіді під час релізу v2: $(sort "$E/traffic-v2.log" | uniq -c | tr '\n' ' ')" | tee -a "$E/flow.txt"
R get rollout frontend --no-color > "$E/rollout-v2.txt" 2>&1 || true
endlog

log "реліз v3 (зламаний: недоступний backend) — аналіз зупиняє перемикання"
release fe-v3
for _ in $(seq 1 100); do
  ph=$(kubectl -n rf-ui get rollout frontend -o jsonpath='{.status.phase}')
  [ "$ph" = "Degraded" ] || [ "$(kubectl -n rf-ui get rollout frontend -o jsonpath='{.status.abort}')" = "true" ] && break
  [ "$(kubectl -n rf-ui get rollout frontend -o jsonpath='{.status.pauseConditions[0].reason}')" = "BlueGreenPause" ] && R promote frontend >/dev/null 2>&1
  sleep 3
done
{
  echo "v3: фаза rollout = $(kubectl -n rf-ui get rollout frontend -o jsonpath='{.status.phase}'), abort = $(kubectl -n rf-ui get rollout frontend -o jsonpath='{.status.abort}')"
  echo "  аналіз: $(kubectl -n rf-ui get analysisrun -o custom-columns=NAME:.metadata.name,PHASE:.status.phase --no-headers | tr '\n' ' ')"
  echo "  користувачі (frontend) -> $(ver http://frontend:8081 20)"
  echo "  ArgoCD: $(kubectl -n argocd get application rf-ui -o jsonpath='{.status.sync.status}/{.status.health.status}')"
} | tee -a "$E/flow.txt"
R get rollout frontend --no-color > "$E/rollout-v3.txt" 2>&1 || true
kubectl -n rf-ui get analysisrun -o yaml | grep -A3 -E 'message:|phase:' | head -40 > "$E/analysis-v3.txt"
endlog
section "Argo Rollouts blue-green" "$E/flow.txt"
