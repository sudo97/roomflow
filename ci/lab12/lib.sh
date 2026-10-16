#!/usr/bin/env bash
source "$(dirname "${BASH_SOURCE[0]}")/../lib/common.sh"
ISTIO_VERSION=${ISTIO_VERSION:-1.23.2}

install_istio() { # profile file
  if ! command -v istioctl >/dev/null; then
    (cd /tmp && curl -sL https://istio.io/downloadIstio | ISTIO_VERSION=$ISTIO_VERSION sh - >/dev/null)
    cp /tmp/istio-$ISTIO_VERSION/bin/istioctl "$BIN/"
  fi
  istioctl install -y -f "$1" > "$EVIDENCE/istio-install.txt" 2>&1
  kubectl -n istio-system rollout status deploy/istiod --timeout=300s
}

mesh_images() { # cluster
  docker build -q --target production --build-arg VERSION=mesh -t roomflow:mesh . >/dev/null
  for v in 1 2 3; do
    local color=steelblue backend=http://backend:8080
    [ $v = 2 ] && color=seagreen
    [ $v = 3 ] && backend=http://backend-missing:8080
    printf 'FROM roomflow:mesh\nENV FRONTEND_VERSION=v%s COLOR=%s BACKEND_URL=%s\n' $v $color $backend | docker build -q -t roomflow:fe-v$v - >/dev/null
  done
  kind load docker-image roomflow:mesh roomflow:fe-v1 roomflow:fe-v2 roomflow:fe-v3 --name "$1" >/dev/null
}

# fortio NS ARGS...: навантажувальний клієнт Istio (fortio) усередині mesh
fortio_pod() {
  # образ fortio не містить shell/sleep: под працює у режимі fortio server, навантаження — через exec fortio load
  # proxyStatsMatcher: Istio за замовчуванням не експортує статистику circuit breaker / outlier detection
  local ann='proxy.istio.io/config={"proxyStatsMatcher":{"inclusionRegexps":[".*outlier_detection.*",".*upstream_rq_retry.*",".*upstream_rq_pending_overflow.*",".*upstream_cx_overflow.*"]}}'
  kubectl -n "$1" run fortio --image=fortio/fortio:latest_release --restart=Never --annotations="$ann" >/dev/null 2>&1 || true
  kubectl -n "$1" wait --for=condition=Ready pod/fortio --timeout=180s >/dev/null
}
fortio() { local ns=$1; shift; kubectl -n "$ns" exec fortio -c fortio -- fortio load -quiet "$@" 2>&1; }
