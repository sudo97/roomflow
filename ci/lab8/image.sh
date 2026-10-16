#!/usr/bin/env bash
# ЛР8, завдання 3 та інд. 3, 9: безпечний образ, Trivy, SBOM, підпис Cosign (keyless) і верифікація.
source "$(dirname "$0")/../lib/common.sh"
E=$EVIDENCE
TRIVY="docker run --rm -v /var/run/docker.sock:/var/run/docker.sock -v $E:/out aquasec/trivy:0.56.2"
IMAGE=ghcr.io/${GITHUB_REPOSITORY:-sudo97/roomflow}

log "hadolint"
docker run --rm -i hadolint/hadolint:v2.12.0 < Dockerfile > "$E/hadolint.txt" 2>&1 || true
endlog

log "наївний образ проти мінімального"
docker build -q -f ci/lab8/Dockerfile.naive -t roomflow:naive . >/dev/null
build_image roomflow:hardened
for img in naive hardened; do
  $TRIVY image --quiet --severity LOW,MEDIUM,HIGH,CRITICAL --format json -o /out/trivy-$img.json roomflow:$img || true
  size=$(docker image inspect roomflow:$img --format '{{.Size}}')
  user=$(docker image inspect roomflow:$img --format '{{.Config.User}}')
  python3 - "$E/trivy-$img.json" "$img" "$size" "${user:-root}" <<'PY' | tee -a "$E/image-comparison.txt"
import json, sys, collections
r = json.load(open(sys.argv[1])); c = collections.Counter()
for res in r.get("Results", []):
    for v in res.get("Vulnerabilities") or []: c[v["Severity"]] += 1
base = r.get("Metadata", {}).get("OS", {})
print(f'{sys.argv[2]:9} os={base.get("Family","-")} {base.get("Name","")} size={int(sys.argv[3])/1e6:.1f}MB user={sys.argv[4]} '
      f'CRITICAL={c["CRITICAL"]} HIGH={c["HIGH"]} MEDIUM={c["MEDIUM"]} LOW={c["LOW"]}')
PY
done
endlog

log "ворота: HIGH/CRITICAL у робочому образі блокують випуск"
$TRIVY image --quiet --severity HIGH,CRITICAL --ignore-unfixed --exit-code 1 roomflow:hardened | tee "$E/trivy-gate.txt"
docker run --rm -v /var/run/docker.sock:/var/run/docker.sock -v "$PWD:/w" aquasec/trivy:0.56.2 image --quiet --format sarif -o /w/trivy-image.sarif roomflow:hardened || true
endlog

log "SBOM (Syft)"
docker run --rm -v /var/run/docker.sock:/var/run/docker.sock -v "$E:/out" anchore/syft:v1.16.0 roomflow:hardened -o spdx-json=/out/sbom.spdx.json -o cyclonedx-json=/out/sbom.cdx.json -o table=/out/sbom.txt
head -20 "$E/sbom.txt"
endlog

if [ -n "${ACTIONS_ID_TOKEN_REQUEST_URL:-}" ]; then
  log "push + cosign keyless"
  curl -sSLo "$BIN/cosign" https://github.com/sigstore/cosign/releases/download/v2.4.1/cosign-linux-amd64 && chmod +x "$BIN/cosign"
  echo "$GITHUB_TOKEN" | docker login ghcr.io -u "$GITHUB_ACTOR" --password-stdin >/dev/null
  TAG=$IMAGE:sec-$(git rev-parse --short HEAD)
  docker tag roomflow:hardened "$TAG" && docker push -q "$TAG"
  DIGEST=$(docker inspect --format '{{index .RepoDigests 0}}' "$TAG")
  echo "$DIGEST" > "$E/image-digest.txt"
  cosign sign --yes "$DIGEST" > "$E/cosign-sign.txt" 2>&1
  cosign attest --yes --type spdxjson --predicate "$E/sbom.spdx.json" "$DIGEST" >> "$E/cosign-sign.txt" 2>&1
  ID="https://github.com/${GITHUB_REPOSITORY}/.github/workflows/.*"
  ISS=https://token.actions.githubusercontent.com
  cosign verify --certificate-identity-regexp "$ID" --certificate-oidc-issuer $ISS "$DIGEST" 2>&1 | tee "$E/cosign-verify.txt" | head -c 2000
  cosign verify-attestation --type spdxjson --certificate-identity-regexp "$ID" --certificate-oidc-issuer $ISS "$DIGEST" > /dev/null 2>"$E/cosign-verify-attestation.txt" && echo "SBOM attestation verified" >> "$E/cosign-verify-attestation.txt"
  # Негативний тест: непідписаний образ не проходить перевірку.
  docker tag roomflow:naive "$IMAGE:unsigned-demo" && docker push -q "$IMAGE:unsigned-demo"
  set +e
  cosign verify --certificate-identity-regexp "$ID" --certificate-oidc-issuer $ISS "$(docker inspect --format '{{index .RepoDigests 0}}' "$IMAGE:unsigned-demo")" > "$E/cosign-verify-unsigned.txt" 2>&1
  echo "exit code: $?" >> "$E/cosign-verify-unsigned.txt"
  set -e
  endlog
fi
section "Порівняння образів" "$E/image-comparison.txt"
