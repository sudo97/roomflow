#!/usr/bin/env bash
# ЛР9: навантажувальне тестування до і після оптимізацій.
#   before  — версія без індексу (ListBookings сканує всі бронювання);
#   after   — версія з індексом бронювань за (tenant, room);
#   cache   — after + Redis cache-aside для календаря доступності;
#   limit   — after + обмеження 50 запитів/с на орендаря.
# Використання: run.sh REF_BEFORE REF_AFTER
source "$(dirname "$0")/../lib/common.sh"
BEFORE=${1:?ref before}; AFTER=${2:?ref after}
E=$EVIDENCE; mkdir -p "$E/k6" "$E/jmeter" "$E/pprof" "$E/stats"
K6="docker run --rm --network host -v $PWD/test/load/k6:/k6:ro -v $E/k6:/out grafana/k6:0.54.0"
NET=rf-load
docker network create $NET >/dev/null 2>&1 || true

start_app() { # image [env...]
  docker rm -f rf-load >/dev/null 2>&1 || true
  local img=$1; shift
  local envs=(); for e in "$@"; do envs+=(-e "$e"); done
  docker run -d --name rf-load --network $NET --cpus 1 --memory 512m -p 8080:8080 -p 6060:6060 -e PPROF_ADDR=:6060 "${envs[@]}" "$img" >/dev/null
  wait_http http://localhost:8080/healthz 30
}
seed() { # label
  go run ./test/load/seed -url http://localhost:8080 -tenants 20 -rooms 10 -bookings 30000 | tee "$E/seed-$1.txt"
}
stats_on() { (while true; do docker stats --no-stream --format '{{.CPUPerc}},{{.MemUsage}}' rf-load | sed "s/^/$(date +%s),/"; sleep 2; done) > "$E/stats/$1.csv" 2>/dev/null & STATS=$!; }
stats_off() { kill $STATS 2>/dev/null || true; }
k6() { # label endpoint [extra -e ...]
  local label=$1 ep=$2; shift 2
  stats_on "$label-$ep"
  set +e
  $K6 run --quiet -e ENDPOINT="$ep" "$@" --summary-export "/out/$label-$ep.json" /k6/load.js > "$E/k6/$label-$ep.txt" 2>&1
  echo "k6 exit code: $?" >> "$E/k6/$label-$ep.txt"
  set -e
  stats_off
}
stress() { # label
  stats_on "$1-stress"
  $K6 run --quiet --summary-export "/out/$1-stress.json" /k6/stress.js > "$E/k6/$1-stress.txt" 2>&1 || true
  stats_off
}
jmeter() { # label endpoint
  rm -rf "/tmp/jm-$2"; mkdir -p "/tmp/jm-$2"
  docker run --rm --network host -v "$PWD/test/load/jmeter:/plan:ro" -v "/tmp/jm-$2:/out" justb4/jmeter:5.5 \
    -n -t /plan/roomflow.jmx -Jhost=localhost -Jport=8080 -Jendpoint="$2" -Jramp=60 -Jduration=90 \
    -l /out/results.jtl -e -o /out/report > "$E/jmeter/$1-$2.log" 2>&1 || true
  cp "/tmp/jm-$2/results.jtl" "$E/jmeter/$1-$2.jtl" 2>/dev/null || true
  (cd "/tmp/jm-$2" && tar czf "$E/jmeter/$1-$2-report.tgz" report 2>/dev/null) || true
}
profile() { # label (CPU-профіль під навантаженням на availability)
  ( sleep 25; curl -s -o "$E/pprof/$1.cpu.pb.gz" "http://localhost:6060/debug/pprof/profile?seconds=20" ) &
  local P=$!
  k6 "$1-profiled" availability -e HOLD=20s
  wait $P
  go tool pprof -top -nodecount=15 "$E/pprof/$1.cpu.pb.gz" > "$E/pprof/$1-top.txt" 2>/dev/null || true
  curl -s http://localhost:6060/debug/pprof/heap > "$E/pprof/$1.heap.pb.gz"
}

log "images"
build_image roomflow:before "$BEFORE"
build_image roomflow:after "$AFTER"
docker pull -q grafana/k6:0.54.0 justb4/jmeter:5.5 redis:7.4-alpine >/dev/null
endlog

log "BEFORE: k6 + JMeter, стрес, профіль"
start_app roomflow:before; seed before
k6 before health; jmeter before health
k6 before availability; jmeter before availability
profile before
stress before
k6 before booking; jmeter before booking
endlog

log "AFTER (індекс)"
start_app roomflow:after; seed after
k6 after health
k6 after availability
profile after
stress after
k6 after booking
endlog

log "AFTER + Redis cache-aside"
docker rm -f rf-redis >/dev/null 2>&1 || true
docker run -d --name rf-redis --network $NET redis:7.4-alpine >/dev/null
start_app roomflow:after CACHE_REDIS_ADDR=rf-redis:6379; seed cache
k6 cache availability
stress cache
curl -s http://localhost:8080/metrics | grep -E '^roomflow_cache_(hits|misses|errors)_total' > "$E/cache-metrics.txt" || true
endlog

log "rate limiting 50 rps / tenant"
start_app roomflow:after RATE_LIMIT_RPS=50 RATE_LIMIT_BURST=50
go run ./test/load/seed -url http://localhost:8080 -tenants 2 -rooms 1 -bookings 0 > "$E/seed-limit.txt"
set +e
$K6 run --quiet --summary-export /out/limit.json /k6/ratelimit.js > "$E/k6/limit.txt" 2>&1; echo "k6 exit code: $?" >> "$E/k6/limit.txt"
set -e
curl -s http://localhost:8080/metrics | grep -E '^roomflow_(rate_limited_total|http_requests_total.*availability)' > "$E/limit-metrics.txt" || true
endlog
docker rm -f rf-load rf-redis >/dev/null

python3 ci/lab9/report.py "$E" | tee "$E/results.md"
cat "$E/results.md" | summary
