#!/usr/bin/env bash
# ЛР8, завдання 2: статичний аналіз безпеки gosec (аналог Bandit) з baseline і SARIF для вкладки Security.
source "$(dirname "$0")/../lib/common.sh"
E=$EVIDENCE
go install github.com/securego/gosec/v2/cmd/gosec@v2.21.4
export PATH="$(go env GOPATH)/bin:$PATH"
gosec -quiet -no-fail -fmt sarif -out gosec.sarif ./... || true
gosec -quiet -no-fail -fmt text ./... > "$E/gosec-all.txt" 2>&1 || true
bash ci/lab8/gosec-gate.sh "$E/gosec.json" | tee "$E/gosec-gate.txt"
