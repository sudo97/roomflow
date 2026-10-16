#!/usr/bin/env bash
# ЛР10, завдання 2: Canary через співвідношення реплік: 10% -> 30% -> 50% -> 100%.
source "$(dirname "$0")/lib.sh"
E=$EVIDENCE/canary; mkdir -p "$E"
cluster rf-canary
kubectl apply -f deploy/k8s/canary/ > "$E/apply.txt"
kubectl -n rf-canary rollout status deploy/roomflow-stable --timeout=180s
kubectl -n rf-canary rollout status deploy/roomflow-canary --timeout=180s
probe rf-canary
: > "$E/stages.txt"
for stage in "9 1" "7 3" "5 5" "0 10"; do
  set -- $stage
  kubectl -n rf-canary scale deploy/roomflow-stable --replicas="$1" >/dev/null
  kubectl -n rf-canary scale deploy/roomflow-canary --replicas="$2" >/dev/null
  kubectl -n rf-canary rollout status deploy/roomflow-canary --timeout=120s >/dev/null
  sleep 5
  dist=$(sample rf-canary http://roomflow 400)
  errs=$(kubectl -n rf-canary exec probe -- sh -c 'n=0; for i in $(seq 1 100); do c=$(curl -s -o /dev/null -m 2 -w "%{http_code}" http://roomflow/healthz); [ "$c" = 200 ] || n=$((n+1)); done; echo $n')
  echo "stable=$1 canary=$2 (план $(( 100 * $2 / ($1 + $2) ))%): $dist errors=$errs/100" | tee -a "$E/stages.txt"
done
section "Canary: етапи" "$E/stages.txt"
