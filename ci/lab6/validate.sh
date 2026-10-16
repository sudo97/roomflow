#!/usr/bin/env bash
# ЛР6: статична перевірка IaC — форматування, валідація, модульні тести terraform test, тести політик OPA.
source "$(dirname "$0")/../lib/common.sh"
install_terraform
TF=infra/terraform
out=$EVIDENCE/validate.txt; : > "$out"

log "terraform fmt"
terraform fmt -check -recursive -diff "$TF" | tee -a "$out"
echo "fmt: OK" >> "$out"; endlog

for dir in $TF/modules/* $TF/envs/*; do
  log "validate $dir"
  terraform -chdir="$dir" init -backend=false -input=false -no-color >/dev/null
  terraform -chdir="$dir" validate -no-color | sed "s|^|$(basename "$dir"): |" | tee -a "$out"
  endlog
done

log "terraform test"
for dir in $TF/modules/*; do
  [ -d "$dir/tests" ] || continue
  echo "== terraform test: $(basename "$dir")" | tee -a "$EVIDENCE/tftest.txt"
  terraform -chdir="$dir" test -no-color 2>&1 | tee -a "$EVIDENCE/tftest.txt"
done
endlog

log "conftest verify"
docker run --rm -v "$PWD/policies:/policies" -w /policies openpolicyagent/conftest:v0.55.0 verify --policy terraform -o table 2>&1 | tee "$EVIDENCE/policy-tests.txt"
endlog
section "terraform test" "$EVIDENCE/tftest.txt"
section "Тести політик OPA" "$EVIDENCE/policy-tests.txt"
