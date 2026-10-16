#!/usr/bin/env bash
# Спільні функції для сценаріїв лабораторних робіт 6–12 (виконуються на runner-і GitHub Actions).
set -euo pipefail

export EVIDENCE=${EVIDENCE:-/tmp/evidence}
mkdir -p "$EVIDENCE"
BIN=${BIN:-$HOME/.local/bin}
mkdir -p "$BIN"; export PATH="$BIN:$PATH"

log() { echo "::group::$*" >&2; }
endlog() { echo "::endgroup::" >&2; }
summary() { if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then cat >> "$GITHUB_STEP_SUMMARY"; else cat; fi; }
# section TITLE FILE: додає файл у Job Summary як блок коду
section() { { echo "### $1"; echo '```'; head -c 60000 "$2"; echo; echo '```'; } | summary; }

retry() { # retry N cmd...
  local n=$1; shift
  for i in $(seq 1 "$n"); do "$@" && return 0; echo "retry $i/$n: $*" >&2; sleep $((i * 3)); done
  return 1
}

wait_http() { # wait_http URL [seconds]
  local url=$1 t=${2:-60}
  for _ in $(seq 1 "$t"); do curl -sf -o /dev/null "$url" && return 0; sleep 1; done
  echo "timeout waiting for $url" >&2; return 1
}

install_terraform() {
  command -v terraform >/dev/null && return
  local v=${TERRAFORM_VERSION:-1.9.8}
  curl -sSLo /tmp/tf.zip "https://releases.hashicorp.com/terraform/${v}/terraform_${v}_linux_amd64.zip"
  unzip -oq /tmp/tf.zip -d "$BIN" terraform
}

install_kind() {
  command -v kind >/dev/null || { curl -sSLo "$BIN/kind" https://kind.sigs.k8s.io/dl/v0.24.0/kind-linux-amd64; chmod +x "$BIN/kind"; }
  command -v kubectl >/dev/null || { curl -sSLo "$BIN/kubectl" "https://dl.k8s.io/release/v1.31.2/bin/linux/amd64/kubectl"; chmod +x "$BIN/kubectl"; }
}

install_helm() {
  command -v helm >/dev/null && return
  curl -sSL https://get.helm.sh/helm-v3.16.2-linux-amd64.tar.gz | tar -xz -C /tmp
  mv /tmp/linux-amd64/helm "$BIN/helm"
}

# kind_up NAME [config]: створює кластер kind і чекає готовності вузлів
kind_up() {
  install_kind
  local name=$1 cfg=${2:-}
  if [ -n "$cfg" ]; then kind create cluster --name "$name" --config "$cfg" --wait 120s; else kind create cluster --name "$name" --wait 120s; fi
  kubectl cluster-info
}

# build_image TAG [REF]: збирає образ RoomFlow з поточного дерева або з коміту REF
build_image() {
  local tag=$1 ref=${2:-}
  if [ -n "$ref" ]; then
    local dir; dir=$(mktemp -d); git worktree add -q -f "$dir" "$ref"
    docker build -q --target production --build-arg VERSION="$tag" --build-arg VCS_REF="$(git rev-parse --short "$ref")" -t "$tag" "$dir" >/dev/null
    git worktree remove -f "$dir"
  else
    docker build -q --target production --build-arg VERSION="$tag" --build-arg VCS_REF="$(git rev-parse --short HEAD)" -t "$tag" . >/dev/null
  fi
  echo "built $tag"
}

# playwright_setup: встановлює Playwright із Chromium у /tmp/pw (для знімків екрана)
playwright_setup() {
  [ -d /tmp/pw/node_modules/playwright ] && return
  mkdir -p /tmp/pw && (cd /tmp/pw && npm init -y >/dev/null && npm i -s playwright@1.48.2 >/dev/null && npx playwright install --with-deps chromium >/dev/null)
}

# shoot OUT.png URL [wait_ms] [width] [height]: знімок сторінки
shoot() {
  playwright_setup
  NODE_PATH=/tmp/pw/node_modules node "$(dirname "${BASH_SOURCE[0]}")/shoot.mjs" "$@"
}
