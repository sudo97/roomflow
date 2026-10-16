#!/usr/bin/env bash
# ЛР10, завдання 3 і 4: Helm-чарт blue/green (атомарне перемикання без простою)
# та feature flags через ConfigMap без перезапуску подів.
source "$(dirname "$0")/lib.sh"
E=$EVIDENCE/helm; mkdir -p "$E"
CHART=deploy/helm/roomflow
cluster rf-helm
helm lint "$CHART" | tee "$E/lint.txt"
helm template rf "$CHART" --set activeColor=green > "$E/template-green.yaml"

log "install (blue active)"
kubectl create namespace rf-helm >/dev/null
helm install rf "$CHART" -n rf-helm --wait --timeout 3m | tee "$E/install.txt"
probe rf-helm
echo "before: $(sample rf-helm http://rf 100)" | tee "$E/routing.txt"
endlog

log "upgrade --atomic: blue -> green під навантаженням"
traffic rf-helm http://rf "$E/traffic.log"
sleep 5
s=$(date +%s%N)
helm upgrade rf "$CHART" -n rf-helm --reuse-values --set activeColor=green --atomic --wait --timeout 3m > "$E/upgrade.txt"
echo "helm upgrade: $(( ($(date +%s%N) - s) / 1000000 )) ms" | tee "$E/timings.txt"
sleep 5
stop_traffic
echo "after:  $(sample rf-helm http://rf 100)" | tee -a "$E/routing.txt"
summary_codes "$E/traffic.log" | tee "$E/traffic-summary.txt"
helm history rf -n rf-helm | tee "$E/history.txt"
endlog

log "атомарність: невдалий upgrade відкочується автоматично"
set +e
helm upgrade rf "$CHART" -n rf-helm --reuse-values --set green.tag=does-not-exist --atomic --wait --timeout 45s > "$E/failed-upgrade.txt" 2>&1
echo "exit code: $?" >> "$E/failed-upgrade.txt"
set -e
helm history rf -n rf-helm | tee -a "$E/history.txt"
echo "after failed upgrade: $(sample rf-helm http://rf 50)" | tee -a "$E/routing.txt"
endlog

log "feature flag price_breakdown: зміна ConfigMap без перезапуску"
POD=$(kubectl -n rf-helm get pod -l color=green -o jsonpath='{.items[0].metadata.name}')
pod_state() { kubectl -n rf-helm get pod "$POD" -o jsonpath='{.metadata.uid} restarts={.status.containerStatuses[0].restartCount}'; }
api() { kubectl -n rf-helm exec probe -- curl -s -H 'X-Tenant-ID: beta-hotel' -H 'Content-Type: application/json' "$@"; }
BEFORE=$(pod_state)
PODIP=$(kubectl -n rf-helm get pod "$POD" -o jsonpath='{.status.podIP}')
U=http://$PODIP:8080
api -X POST $U/api/rooms -d '{"id":"r1","name":"Suite","capacity":2,"price":200000}' >/dev/null
{
  echo "flags (до):    $(api $U/api/flags)"
  echo "бронювання (до): $(api -X POST $U/api/bookings -d '{"room_id":"r1","guest_name":"A","guest_email":"a@x.io","guests":1,"check_in":"2027-05-07","check_out":"2027-05-09"}')"
} | tee "$E/flags.txt"
s=$(date +%s)
helm upgrade rf "$CHART" -n rf-helm --reuse-values --set flags.price_breakdown.enabled=true --wait >/dev/null
for _ in $(seq 1 60); do api $U/api/flags | grep -q '"price_breakdown":true' && break; sleep 3; done
{
  echo "прапорець застосовано через $(( $(date +%s) - s )) с (синхронізація ConfigMap kubelet + перечитування файлу)"
  echo "flags (після): $(api $U/api/flags)"
  echo "бронювання (після): $(api -X POST $U/api/bookings -d '{"room_id":"r1","guest_name":"B","guest_email":"b@x.io","guests":1,"check_in":"2027-05-14","check_out":"2027-05-16"}')"
  echo "под до:    $BEFORE"
  echo "под після: $(pod_state)"
} | tee -a "$E/flags.txt"
# поступове ввімкнення: лише для обраних орендарів
helm upgrade rf "$CHART" -n rf-helm --reuse-values --set flags.price_breakdown.enabled=false --set 'flags.price_breakdown.tenants={beta-hotel}' --wait >/dev/null
for _ in $(seq 1 60); do kubectl -n rf-helm exec probe -- curl -s -H 'X-Tenant-ID: other-hotel' $U/api/flags | grep -q '"price_breakdown":false' && break; sleep 3; done
echo "таргетування: beta-hotel=$(api $U/api/flags) other-hotel=$(kubectl -n rf-helm exec probe -- curl -s -H 'X-Tenant-ID: other-hotel' $U/api/flags)" | tee -a "$E/flags.txt"
endlog
section "Helm: маршрутизація" "$E/routing.txt"
section "Helm: відповіді під час upgrade" "$E/traffic-summary.txt"
section "Feature flags" "$E/flags.txt"
