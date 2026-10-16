#!/usr/bin/env bash
# ЛР8, інд. завдання 7: сканування інфраструктурного коду (Checkov, Trivy config) — Terraform, Kubernetes, Helm, Dockerfile.
source "$(dirname "$0")/../lib/common.sh"
E=$EVIDENCE
pip install -q "checkov>=3.2,<4" 2>/dev/null || pip install -q --break-system-packages "checkov>=3.2,<4"
checkov -d infra/terraform --framework terraform --compact --quiet -o cli > "$E/checkov-terraform.txt" 2>&1 || true
checkov -d deploy/k8s --framework kubernetes --compact --quiet -o cli > "$E/checkov-k8s.txt" 2>&1 || true
checkov -d deploy/helm --framework helm --compact --quiet -o cli > "$E/checkov-helm.txt" 2>&1 || true
checkov -f Dockerfile --framework dockerfile --compact --quiet -o cli > "$E/checkov-dockerfile.txt" 2>&1 || true
docker run --rm -v "$PWD:/src" -w /src aquasec/trivy:0.56.2 config --quiet --severity MEDIUM,HIGH,CRITICAL . > "$E/trivy-config.txt" 2>&1 || true
for f in "$E"/checkov-*.txt; do echo "$(basename "$f"): $(grep -E '^Passed checks|Passed checks' "$f" | head -1)"; done | tee "$E/iac-summary.txt"
section "IaC scanning" "$E/iac-summary.txt"
