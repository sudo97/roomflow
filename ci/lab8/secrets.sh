#!/usr/bin/env bash
# ЛР8, завдання 5: detect-secrets (baseline + pre-commit hook) і gitleaks (історія репозиторію).
source "$(dirname "$0")/../lib/common.sh"
E=$EVIDENCE
pip install -q detect-secrets==1.5.0 2>/dev/null || pip install -q --break-system-packages detect-secrets==1.5.0

log "detect-secrets: поточний стан відповідає baseline"
git ls-files -z | xargs -0 detect-secrets-hook --baseline .secrets.baseline > "$E/detect-secrets-clean.txt" 2>&1 && echo "OK: нових секретів немає" >> "$E/detect-secrets-clean.txt"
python3 -c "import json;d=json.load(open('.secrets.baseline'));print('baseline:', sum(len(v) for v in d['results'].values()), 'audited entries, all is_secret=false:', all(not x.get('is_secret', True) for v in d['results'].values() for x in v))" >> "$E/detect-secrets-clean.txt"
endlog

log "detect-secrets: спроба закомітити тестовий ключ блокується"
mkdir -p /tmp/leak && cat > /tmp/leak/config.py <<'PY'
AWS_ACCESS_KEY_ID = "AKIAIOSFODNN7EXAMPLE"
AWS_SECRET_ACCESS_KEY = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
db_password = "Sup3r-S3cret-Passw0rd!"
PY
cp /tmp/leak/config.py ./leaked_config.py
set +e
detect-secrets-hook --baseline .secrets.baseline leaked_config.py > "$E/detect-secrets-blocked.txt" 2>&1
echo "hook exit code: $? (1 = коміт заблоковано)" >> "$E/detect-secrets-blocked.txt"
set -e
rm -f leaked_config.py
endlog

log "detect-secrets на legacy-версії (32a122c)"
git worktree add -q -f /tmp/legacy 32a122c
(cd /tmp/legacy && detect-secrets scan . | python3 -c "import json,sys;d=json.load(sys.stdin);[print(f,x['type'],'line',x['line_number']) for f,v in d['results'].items() for x in v]") > "$E/detect-secrets-legacy.txt"
git worktree remove -f /tmp/legacy
endlog

log "gitleaks: уся історія"
set +e
docker run --rm -v "$PWD:/repo" -v "$E:/out" zricethezav/gitleaks:v8.21.2 git --redact --report-format json --report-path /out/gitleaks.json /repo > "$E/gitleaks.txt" 2>&1
echo "gitleaks exit code: $?" >> "$E/gitleaks.txt"
set -e
endlog
section "detect-secrets (заблокований коміт)" "$E/detect-secrets-blocked.txt"
section "gitleaks" "$E/gitleaks.txt"
