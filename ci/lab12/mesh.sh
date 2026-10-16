#!/usr/bin/env bash
# ЛР12, завдання 1, 3, 4 та інд. 3, 5: Istio service mesh для RoomFlow (frontend -> backend -> redis),
# розподілене трасування, circuit breaker, zero trust і хаос-експерименти.
source "$(dirname "$0")/lib.sh"
E=$EVIDENCE/mesh; mkdir -p "$E"
NS=rf-mesh
kind_up rf-mesh
mesh_images rf-mesh
install_istio deploy/mesh/istio/istio-operator.yaml
kubectl apply -f deploy/mesh/istio/jaeger.yaml
kubectl apply -f deploy/mesh/core/namespace.yaml
kubectl apply -f deploy/mesh/core/redis.yaml -f deploy/mesh/core/apps.yaml -f deploy/mesh/core/resilience.yaml
for d in redis backend frontend; do kubectl -n $NS rollout status deploy/$d --timeout=300s; done
kubectl -n istio-system rollout status deploy/jaeger --timeout=300s
fortio_pod $NS
kubectl -n $NS get pods -o wide > "$E/pods.txt"
F="http://frontend:8081"
call() { kubectl -n $NS exec fortio -c fortio -- fortio curl -quiet -H "X-Tenant-ID: demo-hotel" "$@" 2>&1 || true; }  # fortio curl повертає 1 на будь-який статус, крім 200 (напр. 201)
set +e  # далі — експерименти: доказова база збирається навіть при частковій невдачі окремого кроку

log "завдання 1: Redis у mesh, кеш backend, персистентність"
call -payload '{"id":"r1","name":"Suite","capacity":2,"price":200000}' $F/api/rooms >/dev/null
for i in $(seq 1 20); do call "$F/api/rooms/r1/availability?from=2027-01-01&to=2027-02-01" >/dev/null; done
{
  for p in $(kubectl -n $NS get pod -l app=backend -o name); do
    echo "$p: $(kubectl -n $NS exec "$p" -c backend -- /roomflow -version 2>/dev/null) $(kubectl -n $NS exec fortio -c fortio -- fortio curl -quiet http://$(kubectl -n $NS get "$p" -o jsonpath='{.status.podIP}'):8080/metrics 2>/dev/null | grep -E '^roomflow_cache_(hits|misses)_total' | tr '\n' ' ')"
  done
  echo "redis keys before restart: $(kubectl -n $NS exec deploy/redis -c redis -- redis-cli dbsize)"
  kubectl -n $NS delete pod -l app=redis --wait=true >/dev/null
  kubectl -n $NS rollout status deploy/redis --timeout=180s >/dev/null
  sleep 3
  echo "redis keys after pod restart (AOF on PVC): $(kubectl -n $NS exec deploy/redis -c redis -- redis-cli dbsize)"
  kubectl -n $NS get pvc redis-data
} | tee "$E/redis.txt"
endlog

log "завдання 3: розподілене трасування"
for i in $(seq 1 30); do call "$F/api/rooms/r1/availability?from=2027-0$((1 + i % 8))-01&to=2027-0$((1 + i % 8))-20" >/dev/null; done
sleep 10
kubectl -n istio-system port-forward svc/tracing 16686:16686 >/dev/null 2>&1 &
PF=$!; sleep 4
curl -s "http://localhost:16686/api/services" | jq -c . | tee "$E/jaeger-services.txt"
curl -s "http://localhost:16686/api/traces?service=frontend&operation=&limit=20&lookback=1h" > /tmp/traces.json
python3 - /tmp/traces.json > "$E/traces.txt" <<'PY'
import json, sys, collections
data = json.load(open(sys.argv[1]))["data"]
best = max(data, key=lambda t: len(t["spans"])) if data else None
print(f"traces: {len(data)}")
for t in data[:5]:
    procs = {k: v["serviceName"] for k, v in t["processes"].items()}
    c = collections.Counter(procs[s["processID"]] for s in t["spans"])
    print(f'{t["traceID"]}: {len(t["spans"])} spans {dict(c)}')
if best:
    procs = {k: v["serviceName"] for k, v in best["processes"].items()}
    print("\nнайдовша трасa:", best["traceID"])
    for s in sorted(best["spans"], key=lambda s: s["startTime"]):
        print(f'  {procs[s["processID"]]:24} {s["operationName"][:60]:60} {s["duration"]/1000:8.2f} ms')
    json.dump(best, open("/tmp/best-trace.json", "w"))
PY
cat "$E/traces.txt"
TRACE=$(jq -r '.traceID' /tmp/best-trace.json 2>/dev/null || true)
[ -n "$TRACE" ] && shoot "$E/jaeger-trace.png" "http://localhost:16686/trace/$TRACE" 4000 1600 1000 || true
shoot "$E/jaeger-search.png" "http://localhost:16686/search?service=frontend&limit=20" 4000 1600 900 || true
kill $PF
endlog

log "завдання 4: circuit breaker, повтори, outlier detection"
B=http://backend:8080/healthz
{
  echo "== 1 з'єднання (у межах ліміту)"; fortio $NS -c 1 -qps 0 -n 30 $B | grep -E 'Code|All done|Sockets'
  echo "== 4 паралельні з'єднання (ліміт maxConnections=1, pending=1)"; fortio $NS -c 4 -qps 0 -n 80 $B | grep -E 'Code|All done'
  echo "== статистика Envoy (overflow — запити, відхилені circuit breaker)"
  kubectl -n $NS exec fortio -c istio-proxy -- pilot-agent request GET stats | grep -E 'outbound\|8080\|\|backend.*(upstream_rq_pending_overflow|upstream_cx_overflow):' | head -4
} 2>&1 | tee "$E/circuit-breaker.txt"
# outlier detection: несправна репліка backend (FAULT_RATE=1) серед справних
kubectl -n $NS apply -f - >/dev/null <<YAML
apiVersion: apps/v1
kind: Deployment
metadata: { name: backend-faulty, namespace: $NS }
spec:
  replicas: 1
  selector: { matchLabels: { app: backend, version: faulty } }
  template:
    metadata: { labels: { app: backend, version: faulty } }
    spec:
      serviceAccountName: backend
      containers:
        - name: backend
          image: roomflow:mesh
          imagePullPolicy: IfNotPresent
          env: [{ name: FAULT_RATE, value: "1" }, { name: CACHE_REDIS_ADDR, value: "redis:6379" }]
          readinessProbe: { httpGet: { path: /healthz, port: 8080 } }
YAML
kubectl -n $NS rollout status deploy/backend-faulty --timeout=180s >/dev/null
kubectl -n $NS patch destinationrule backend --type merge -p '{"spec":{"trafficPolicy":{"connectionPool":{"tcp":{"maxConnections":100},"http":{"http1MaxPendingRequests":100,"maxRequestsPerConnection":0}}}}}' >/dev/null
sleep 5
{
  # порядок важливий: спершу без захисту (видно відмови несправної репліки), потім повтори, потім outlier detection;
  # інакше вилучена outlier detection репліка лишається вилученою (30 с x кількість вилучень) і «маскує» наступні фази
  kubectl -n $NS patch virtualservice backend --type json -p '[{"op":"remove","path":"/spec/http/0/retries"}]' >/dev/null
  kubectl -n $NS patch destinationrule backend --type json -p '[{"op":"remove","path":"/spec/trafficPolicy/outlierDetection"}]' >/dev/null
  sleep 5
  echo "== 1) без повторів і без outlier detection (1 з 3 реплік backend повертає 500)"
  fortio $NS -c 2 -qps 20 -t 20s -H "X-Tenant-ID: demo-hotel" $F/api/rooms | grep -E 'Code|All done'
  kubectl apply -f deploy/mesh/core/resilience.yaml >/dev/null
  kubectl -n $NS patch destinationrule backend --type json -p '[{"op":"remove","path":"/spec/trafficPolicy/outlierDetection"}]' >/dev/null
  kubectl -n $NS patch destinationrule backend --type merge -p '{"spec":{"trafficPolicy":{"connectionPool":{"tcp":{"maxConnections":100},"http":{"http1MaxPendingRequests":100,"maxRequestsPerConnection":0}}}}}' >/dev/null
  sleep 5
  echo "== 2) з повторами (attempts 3, retryOn 5xx), без outlier detection"
  fortio $NS -c 2 -qps 20 -t 20s -H "X-Tenant-ID: demo-hotel" $F/api/rooms | grep -E 'Code|All done'
  kubectl -n $NS exec deploy/frontend -c istio-proxy -- pilot-agent request GET stats | grep -E 'outbound\|8080\|\|backend.*upstream_rq_retry(_success)?:' | head -4
  kubectl apply -f deploy/mesh/core/resilience.yaml >/dev/null
  kubectl -n $NS patch destinationrule backend --type merge -p '{"spec":{"trafficPolicy":{"connectionPool":{"tcp":{"maxConnections":100},"http":{"http1MaxPendingRequests":100,"maxRequestsPerConnection":0}}}}}' >/dev/null
  sleep 5
  echo "== 3) повтори + outlier detection (3 поспіль 5xx -> репліку вилучено на 30 с)"
  fortio $NS -c 2 -qps 20 -t 20s -H "X-Tenant-ID: demo-hotel" $F/api/rooms | grep -E 'Code|All done'
  kubectl -n $NS exec deploy/frontend -c istio-proxy -- pilot-agent request GET stats | grep -E 'outbound\|8080\|\|backend.*(outlier_detection\.ejections_(enforced_total|active)|upstream_rq_retry):' | head -6
} 2>&1 | tee "$E/retries-outlier.txt"
kubectl -n $NS delete deploy backend-faulty >/dev/null
kubectl -n $NS apply -f deploy/mesh/core/resilience.yaml >/dev/null
endlog

log "інд. 5: хаос — затримки і відмови (Istio fault injection), вбивство подів"
kubectl -n $NS patch destinationrule backend --type merge -p '{"spec":{"trafficPolicy":{"connectionPool":{"tcp":{"maxConnections":100},"http":{"http1MaxPendingRequests":100,"maxRequestsPerConnection":0}}}}}' >/dev/null
{
  echo "== базова лінія"; fortio $NS -c 4 -qps 40 -t 20s -H "X-Tenant-ID: demo-hotel" $F/api/rooms | grep -E 'Code|All done|# target 50%|# target 99%'
  kubectl apply -f deploy/mesh/chaos/faults.yaml >/dev/null; sleep 10
  echo "   (fault-конфігурація в маршрутах sidecar frontend: $(istioctl proxy-config route deploy/frontend -n $NS -o json 2>/dev/null | grep -c 'envoy.filters.http.fault') входжень)"
  echo "== затримка 1 с (30%) + 503 (10%), через frontend"; fortio $NS -c 4 -qps 40 -t 20s -H "X-Tenant-ID: demo-hotel" $F/api/rooms | grep -E 'Code|All done|# target 50%|# target 99%'
  echo "== те саме, напряму до backend (sidecar fortio)"; fortio $NS -c 4 -qps 40 -t 20s -H "X-Tenant-ID: demo-hotel" http://backend:8080/api/rooms | grep -E 'Code|All done|# target 50%|# target 99%'
  kubectl apply -f deploy/mesh/core/resilience.yaml >/dev/null
  kubectl -n $NS patch destinationrule backend --type merge -p '{"spec":{"trafficPolicy":{"connectionPool":{"tcp":{"maxConnections":100},"http":{"http1MaxPendingRequests":100,"maxRequestsPerConnection":0}}}}}' >/dev/null
  echo "== видалення поду backend під навантаженням"
  (sleep 5; kubectl -n $NS delete pod "$(kubectl -n $NS get pod -l app=backend,version=v1 -o jsonpath='{.items[0].metadata.name}')" --wait=false >/dev/null) &
  fortio $NS -c 4 -qps 40 -t 20s -H "X-Tenant-ID: demo-hotel" $F/api/rooms | grep -E 'Code|All done|# target 99%'
} 2>&1 | tee "$E/chaos.txt"
kubectl apply -f deploy/mesh/core/resilience.yaml >/dev/null
endlog

log "інд. 3: zero trust"
kubectl -n $NS run probe --image=curlimages/curl:8.10.1 --restart=Never --command -- sleep 3600 >/dev/null
kubectl create namespace outside >/dev/null; kubectl -n outside run probe --image=curlimages/curl:8.10.1 --restart=Never --command -- sleep 3600 >/dev/null
kubectl -n $NS wait --for=condition=Ready pod/probe --timeout=120s >/dev/null; kubectl -n outside wait --for=condition=Ready pod/probe --timeout=120s >/dev/null
code() { kubectl -n "$1" exec probe -c probe -- curl -s -o /dev/null -m 5 -w '%{http_code}' -H 'X-Tenant-ID: demo-hotel' "$2" 2>/dev/null; true; }
before_backend=$(code $NS http://backend:8080/api/rooms); before_outside=$(code outside http://backend.$NS:8080/api/rooms)
kubectl apply -f deploy/mesh/core/zero-trust.yaml >/dev/null
sleep 15
{
  echo "до політик:  probe(mesh)->backend $before_backend; outside(без sidecar)->backend $before_outside"
  echo "після:"
  echo "  probe(mesh, SA default) -> frontend /api/rooms:  $(code $NS http://frontend:8081/api/rooms)   (frontend доступний)"
  echo "  probe(mesh, SA default) -> backend /api/rooms:   $(code $NS http://backend:8080/api/rooms)   (403: лише SA frontend)"
  echo "  probe(mesh) -> backend /healthz:                 $(code $NS http://backend:8080/healthz)"
  echo "  outside(без mTLS) -> backend:                    $(code outside http://backend.$NS:8080/api/rooms)   (000: STRICT mTLS)"
  echo "  outside(без mTLS) -> frontend:                   $(code outside http://frontend.$NS:8081/)"
  echo "  probe(mesh) -> redis:6379 (TCP):                 $(kubectl -n $NS exec probe -c probe -- sh -c 'printf "PING\r\n" | timeout 3 nc redis 6379' 2>/dev/null | head -1 || echo blocked)"
  echo "  frontend -> backend через UI:                    $(kubectl -n $NS exec fortio -c fortio -- fortio curl -quiet -H 'X-Tenant-ID: demo-hotel' $F/api/rooms 2>/dev/null | head -c 60)"
  istioctl x describe pod "$(kubectl -n $NS get pod -l app=backend,version=v1 --sort-by=.metadata.creationTimestamp -o name | tail -1 | cut -d/ -f2)" -n $NS 2>&1 | head -20
} | tee "$E/zero-trust.txt"
kubectl apply -f https://raw.githubusercontent.com/aquasecurity/kube-bench/v0.9.0/job.yaml >/dev/null
kubectl wait --for=condition=complete job/kube-bench --timeout=300s >/dev/null || true
kubectl logs job/kube-bench > "$E/kube-bench.txt" 2>&1 || true
grep -E '^== Summary|checks (PASS|FAIL|WARN|INFO)' "$E/kube-bench.txt" | tee "$E/kube-bench-summary.txt" || true
endlog
section "Трасування" "$E/traces.txt"
section "Circuit breaker" "$E/circuit-breaker.txt"
section "Zero trust" "$E/zero-trust.txt"
