#!/usr/bin/env bash
# Спільні функції ЛР10: кластер kind з образами RoomFlow і клієнт-зонд.
source "$(dirname "${BASH_SOURCE[0]}")/../lib/common.sh"

# cluster NAME: кластер + образи v1 (стабільна), v2 (нова) і v2-faulty (30% помилок на /api) — завантажуються в kind
cluster() {
  CLUSTER=$1
  install_helm
  kind_up "$CLUSTER"
  build_image roomflow:v1
  docker build -q --target production --build-arg VERSION=v2 -t roomflow:v2 . >/dev/null
  kind load docker-image roomflow:v1 roomflow:v2 --name "$CLUSTER" >/dev/null
}

# probe NS: под-клієнт у просторі імен NS
probe() {
  kubectl -n "$1" apply -f deploy/k8s/probe/probe.yaml >/dev/null
  kubectl -n "$1" wait --for=condition=Ready pod/probe --timeout=120s >/dev/null
}

LOOP="$(dirname "${BASH_SOURCE[0]}")/loop.sh"

# sample NS URL N: N запитів до URL із пода probe; друкує розподіл кольорів (версій)
sample() {
  kubectl -n "$1" exec -i probe -- sh -s -- "$2" "$3" < "$LOOP" | awk '{print $3}' | sort | uniq -c | awk '{printf "%s=%s ", $2, $1} END {print ""}'
}

# traffic NS URL OUTFILE [PATH]: безперервні запити у фоні (рядки "час код колір"); PID у TRAFFIC_PID
traffic() {
  kubectl -n "$1" exec -i probe -- sh -s -- "$2" 0 "${4:-/version}" < "$LOOP" > "$3" 2>/dev/null &
  TRAFFIC_PID=$!
}
stop_traffic() { kill "$TRAFFIC_PID" 2>/dev/null || true; wait "$TRAFFIC_PID" 2>/dev/null || true; }
# summary_codes FILE: кількість відповідей за кодом і кольором
summary_codes() { awk '{print "code=" $2, "color=" $3}' "$1" | sort | uniq -c; }
