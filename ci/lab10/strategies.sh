#!/usr/bin/env bash
# ЛР10, індивідуальні завдання 1 і 3.
# (3) Експериментальне порівняння rolling update, blue-green і canary за показниками DORA:
#     тривалість розгортання, кількість невдалих запитів користувачів (радіус ураження невдалої зміни),
#     час виявлення і час відновлення (time to restore) для навмисно несправного релізу.
# (1) Гібридна стратегія: вибір за семантичною версією релізу (major -> blue-green, minor -> canary, patch -> rolling).
source "$(dirname "$0")/lib.sh"
E=$EVIDENCE/strategies; mkdir -p "$E"
NS=rf-strat
cluster rf-strat
kubectl create namespace $NS >/dev/null
probe $NS

deployment() { # name replicas image color fault [extra-labels-yaml]
  cat <<YAML | kubectl -n $NS apply -f - >/dev/null
apiVersion: apps/v1
kind: Deployment
metadata: { name: $1 }
spec:
  replicas: $2
  strategy: { type: RollingUpdate, rollingUpdate: { maxSurge: 1, maxUnavailable: 0 } }
  selector: { matchLabels: { app: rf, deploy: $1 } }
  template:
    metadata: { labels: { app: rf, deploy: $1, color: $4 } }
    spec:
      securityContext: { runAsNonRoot: true, runAsUser: 65532 }
      containers:
        - name: roomflow
          image: $3
          imagePullPolicy: IfNotPresent
          env: [{ name: COLOR, value: "$4" }, { name: FAULT_RATE, value: "$5" }]
          readinessProbe: { httpGet: { path: /healthz, port: 8080 }, periodSeconds: 2 }
          resources: { requests: { cpu: 20m, memory: 24Mi }, limits: { cpu: 300m, memory: 96Mi } }
YAML
}
service() { # selector-json
  kubectl -n $NS apply -f - >/dev/null <<YAML
apiVersion: v1
kind: Service
metadata: { name: rf }
spec:
  selector: $1
  ports: [{ port: 80, targetPort: 8080 }]
YAML
}
reset() { kubectl -n $NS delete deploy --all --wait=true >/dev/null 2>&1 || true; }

# watch_errors LOG START: чекає, доки частка 5xx в останніх 40 відповідях > 5%; друкує секунди від START
watch_errors() {
  for _ in $(seq 1 600); do
    if [ "$(tail -n 40 "$1" | awk '$2 ~ /^5/ {n++} END {print (n+0 > 2)}')" = 1 ]; then echo $(( $(date +%s) - $2 )); return; fi
    sleep 0.5
  done
  echo -1
}
# wait_clean LOG: чекає 30 послідовних відповідей без 5xx; друкує час
wait_clean() {
  for _ in $(seq 1 600); do
    [ "$(tail -n 30 "$1" | awk '$2 ~ /^5/ {n++} END {print n+0}')" = 0 ] && [ "$(wc -l < "$1")" -gt 30 ] && { date +%s; return; }
    sleep 0.5
  done
  date +%s
}

run() { # strategy faulty(0|1)
  local s=$1 f=$2 fault=0 log="$E/$1-$2.log"
  [ "$f" = 1 ] && fault=0.5
  reset
  case $s in
    rolling)   deployment rf-main 4 roomflow:v1 v1 0; service '{ app: rf }' ;;
    bluegreen) deployment rf-blue 4 roomflow:v1 blue 0; deployment rf-green 4 roomflow:v2 green $fault; service '{ app: rf, color: blue }' ;;
    canary)    deployment rf-stable 9 roomflow:v1 v1 0; deployment rf-canary 1 roomflow:v2 canary $fault; kubectl -n $NS scale deploy/rf-canary --replicas=0 >/dev/null; service '{ app: rf }' ;;
  esac
  kubectl -n $NS wait --for=condition=Available deploy --all --timeout=180s >/dev/null
  traffic $NS http://rf "$log" /api/rooms
  sleep 5
  local t0; t0=$(date +%s)
  case $s in
    rolling)   kubectl -n $NS set image deploy/rf-main roomflow=roomflow:v2 >/dev/null; kubectl -n $NS set env deploy/rf-main FAULT_RATE=$fault COLOR=v2 >/dev/null ;;
    bluegreen) service '{ app: rf, color: green }' ;;
    canary)    kubectl -n $NS scale deploy/rf-canary --replicas=1 >/dev/null ;;
  esac
  local deployed detect=-1 restored=$t0
  if [ "$f" = 1 ]; then
    detect=$(watch_errors "$log" "$t0")
    case $s in
      rolling)   kubectl -n $NS rollout undo deploy/rf-main >/dev/null; kubectl -n $NS rollout status deploy/rf-main --timeout=180s >/dev/null ;;
      bluegreen) service '{ app: rf, color: blue }' ;;
      canary)    kubectl -n $NS scale deploy/rf-canary --replicas=0 >/dev/null ;;
    esac
    restored=$(wait_clean "$log")
    deployed=-
  else
    case $s in
      rolling)   kubectl -n $NS rollout status deploy/rf-main --timeout=180s >/dev/null ;;
      bluegreen) sleep 2 ;;
      canary)    # етапи 10% -> 50% -> 100%
                 kubectl -n $NS rollout status deploy/rf-canary --timeout=120s >/dev/null; sleep 10
                 kubectl -n $NS scale deploy/rf-canary --replicas=5 >/dev/null; kubectl -n $NS scale deploy/rf-stable --replicas=5 >/dev/null
                 kubectl -n $NS rollout status deploy/rf-canary --timeout=120s >/dev/null; sleep 10
                 kubectl -n $NS scale deploy/rf-canary --replicas=10 >/dev/null; kubectl -n $NS rollout status deploy/rf-canary --timeout=120s >/dev/null
                 kubectl -n $NS scale deploy/rf-stable --replicas=0 >/dev/null ;;
    esac
    deployed=$(( $(date +%s) - t0 ))
  fi
  sleep 5
  stop_traffic
  local total fails
  total=$(wc -l < "$log"); fails=$(awk '$2 ~ /^5/ || $2 == "000"' "$log" | wc -l)
  printf '| %s | %s | %s | %s | %s | %s | %s |\n' "$s" "$([ "$f" = 1 ] && echo несправний || echo справний)" "$deployed" \
    "$([ "$detect" = -1 ] && echo - || echo "$detect")" "$([ "$f" = 1 ] && echo $(( restored - t0 )) || echo -)" "$fails" "$total" | tee -a "$E/results.md"
}

echo "| Стратегія | Реліз | Тривалість розгортання, с | Виявлення збою, с | Відновлення (MTTR), с | Невдалих запитів | Усього запитів |" > "$E/results.md"
echo "|---|---|---|---|---|---|---|" >> "$E/results.md"
for s in rolling bluegreen canary; do
  run $s 0
  run $s 1
done

choose() { # old new -> strategy
  local o=${1#v} n=${2#v}
  if [ "${o%%.*}" != "${n%%.*}" ]; then echo bluegreen
  elif [ "$(echo "$o" | cut -d. -f2)" != "$(echo "$n" | cut -d. -f2)" ]; then echo canary
  else echo rolling; fi
}
{
  echo "Гібридна стратегія (вибір за semver):"
  for pair in "v1.4.2 v2.0.0" "v1.4.2 v1.5.0" "v1.4.2 v1.4.3"; do set -- $pair; echo "  $1 -> $2: $(choose "$1" "$2")"; done
} | tee "$E/hybrid.txt"
cat "$E/results.md" | summary
