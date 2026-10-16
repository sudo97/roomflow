#!/usr/bin/env bash
# ЛР10, завдання 1: Blue-Green на рівні Kubernetes Service selector.
source "$(dirname "$0")/lib.sh"
E=$EVIDENCE/bluegreen; mkdir -p "$E"
cluster rf-bg
kubectl apply -f deploy/k8s/bluegreen/ > "$E/apply.txt"
kubectl -n rf-bg rollout status deploy/roomflow-blue --timeout=180s
kubectl -n rf-bg rollout status deploy/roomflow-green --timeout=180s
probe rf-bg
{
  echo "preview blue:  $(sample rf-bg http://roomflow-blue-preview 20)"
  echo "preview green: $(sample rf-bg http://roomflow-green-preview 20)"
  echo "main (before): $(sample rf-bg http://roomflow 100)"
} | tee "$E/routing.txt"

traffic rf-bg http://roomflow "$E/traffic.log"
sleep 5
s=$(date +%s%N)
kubectl -n rf-bg patch service roomflow --type merge -p '{"metadata":{"annotations":{"roomflow.io/active-color":"green"}},"spec":{"selector":{"app":"roomflow","color":"green"}}}'
echo "switch command took $(( ($(date +%s%N) - s) / 1000000 )) ms" | tee "$E/switch.txt"
sleep 8
echo "main (after):  $(sample rf-bg http://roomflow 100)" | tee -a "$E/routing.txt"
# відкат — повернення selector
s=$(date +%s%N)
kubectl -n rf-bg patch service roomflow --type merge -p '{"spec":{"selector":{"app":"roomflow","color":"blue"}}}'
echo "rollback command took $(( ($(date +%s%N) - s) / 1000000 )) ms" | tee -a "$E/switch.txt"
sleep 5
stop_traffic
echo "main (rollback): $(sample rf-bg http://roomflow 50)" | tee -a "$E/routing.txt"
summary_codes "$E/traffic.log" | tee "$E/traffic-summary.txt"
kubectl -n rf-bg get deploy,svc,endpoints -o wide > "$E/resources.txt"
section "Blue-Green: маршрутизація" "$E/routing.txt"
section "Blue-Green: відповіді під час перемикання" "$E/traffic-summary.txt"
