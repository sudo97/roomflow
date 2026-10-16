#!/usr/bin/env bash
# Повний цикл перевірки якості: збірка -> тести з покриттям -> статичний аналіз -> ворота якості.
set -euo pipefail
: "${SONAR_HOST_URL:=http://localhost:9000}" "${SONAR_TOKEN:?SONAR_TOKEN is required}"
echo "== build"; go build ./... || { echo "Build failed, skipping static analysis"; exit 1; }
echo "== vet";   go vet ./...
echo "== test";  go test -coverprofile=coverage.out ./... || { echo "Tests failed, skipping static analysis"; exit 1; }
echo "== sonar"
docker run --rm --network host -e SONAR_HOST_URL -e SONAR_TOKEN -v "$PWD:/usr/src" \
  sonarsource/sonar-scanner-cli -Dsonar.qualitygate.wait=true
echo "Static analysis completed successfully"
