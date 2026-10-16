#!/usr/bin/env bash
# ЛР6: повний життєвий цикл інфраструктури у CI.
# plan -> перевірка політик -> apply -> тести -> blue/green -> тривоги -> дрейф -> блокування -> destroy.
# Очікує: PostgreSQL для стану (PG_CONN_STR) і локальний реєстр образів на localhost:5000.
source "$(dirname "$0")/../lib/common.sh"
install_terraform
command -v dot >/dev/null || sudo apt-get install -y -qq graphviz >/dev/null
export TF_IN_AUTOMATION=1 TF_INPUT=0
P=infra/terraform/envs/platform
A=infra/terraform/envs/app
E=$EVIDENCE
LB=http://localhost:8088

tf() { terraform -chdir="$1" "${@:2}" -no-color; }
conftest() { docker run --rm -v "$PWD:/work" -v "$EVIDENCE:$EVIDENCE" -w /work openpolicyagent/conftest:v0.55.0 test --policy policies/terraform -o table "$@"; }

log "images v1/v2 -> local registry"
build_image roomflow:v1
docker tag roomflow:v1 localhost:5000/roomflow:v1 && docker push -q localhost:5000/roomflow:v1
docker build -q --target production --build-arg VERSION=v2 --build-arg VCS_REF="$(git rev-parse --short HEAD)" -t localhost:5000/roomflow:v2 . >/dev/null
docker push -q localhost:5000/roomflow:v2
endlog

log "platform: init / plan / policy / apply"
tf $P init -backend-config="conn_str=$PG_CONN_STR" -backend-config="schema_name=platform" >/dev/null
tf $P plan -out=plan.out -var active_color=blue > "$E/platform-plan.txt"
tf $P show -json plan.out > "$E/platform-plan.json"
conftest "$E/platform-plan.json" | tee "$E/platform-policy.txt"
start=$(date +%s)
tf $P apply plan.out > "$E/platform-apply.txt"
echo "platform apply: $(( $(date +%s) - start )) s" | tee "$E/timings.txt"
tf $P output > "$E/platform-outputs.txt"
tf $P graph | dot -Tpng -o "$E/platform-graph.png" || true
endlog

deploy_color() { # color image
  tf $A workspace select -or-create "$1" >/dev/null
  tf $A plan -out=plan-$1.out -var image="$2" > "$E/app-$1-plan.txt"
  tf $A show -json plan-$1.out > "$E/app-$1-plan.json"
  conftest "$E/app-$1-plan.json" | tee "$E/app-$1-policy.txt"
  local s; s=$(date +%s)
  tf $A apply plan-$1.out > "$E/app-$1-apply.txt"
  echo "app $1 apply: $(( $(date +%s) - s )) s" | tee -a "$E/timings.txt"
}

log "app: workspace blue (v1)"
tf $A init -backend-config="conn_str=$PG_CONN_STR" -backend-config="schema_name=app" >/dev/null
deploy_color blue localhost:5000/roomflow:v1
tf $A workspace list > "$E/workspaces.txt"
endlog

log "smoke через балансувальник"
wait_http $LB/healthz 60
bash scripts/smoke.sh $LB | tee "$E/smoke-blue.txt"
curl -s $LB/version | tee "$E/version-before.txt"
endlog

log "мережева ізоляція і NAT"
bash ci/lab6/connectivity.sh rf-ci | tee "$E/connectivity.txt"
endlog

log "green (v2): розгортання в неактивне середовище і приймальні тести"
deploy_color green localhost:5000/roomflow:v2
docker run --rm --network rf-ci-app curlimages/curl:8.10.1 -s http://app-green:8080/version | tee "$E/green-acceptance.txt"
docker run --rm --network rf-ci-app -v "$PWD/scripts:/s:ro" alpine:3.20 sh -c "apk add -q bash curl >/dev/null && bash /s/smoke.sh http://app-green:8080" | tee -a "$E/green-acceptance.txt"
endlog

log "перемикання трафіку blue -> green під навантаженням"
bash ci/lab6/traffic.sh $LB 25 > "$E/switch-traffic.log" &
TRAFFIC=$!
sleep 4
s=$(date +%s%N)
tf $P apply -auto-approve -var active_color=green > "$E/switch-apply.txt"
echo "switch blue->green: $(( ($(date +%s%N) - s) / 1000000 )) ms" | tee -a "$E/timings.txt"
sleep 4
kill $TRAFFIC; wait $TRAFFIC 2>/dev/null || true
awk '{print $1, $2}' "$E/switch-traffic.log" | sort | uniq -c | tee "$E/switch-summary.txt"
curl -s $LB/version | tee "$E/version-after.txt"
endlog

log "тривоги: збій репліки активного середовища (docker pause — процес не відповідає)"
sleep 10
docker pause rf-ci-green-0 >/dev/null
s=$(date +%s)
for _ in $(seq 1 90); do docker logs rf-ci-alert-sink 2>&1 | grep -q InstanceDown && break; sleep 1; done
echo "alert InstanceDown delivered after $(( $(date +%s) - s )) s" | tee -a "$E/timings.txt"
docker logs rf-ci-alert-sink 2>&1 | grep ALERT | head -5 > "$E/alerts-webhook.txt" || true
curl -s 'http://localhost:9090/api/v1/alerts' | jq . > "$E/prometheus-alerts.json" || true
shoot "$E/prometheus-alerts.png" "http://localhost:9090/alerts?state=firing" 3000 1440 900 || true
shoot "$E/prometheus-targets.png" "http://localhost:9090/targets" 3000 1440 900 || true
shoot "$E/alertmanager.png" "http://localhost:9093/#/alerts" 3000 1440 900 || true
docker unpause rf-ci-green-0 >/dev/null
endlog

log "виявлення дрейфу"
docker rm -f rf-ci-green-1 >/dev/null
set +e
tf $A plan -detailed-exitcode -var image=localhost:5000/roomflow:v2 > "$E/drift-plan.txt"
code=$?
set -e
echo "terraform plan -detailed-exitcode = $code (2 означає дрейф)" | tee "$E/drift.txt"
tf $A apply -auto-approve -var image=localhost:5000/roomflow:v2 > "$E/drift-fix.txt"
set +e; tf $A plan -detailed-exitcode -var image=localhost:5000/roomflow:v2 > /dev/null; echo "after re-apply: exit $?" | tee -a "$E/drift.txt"; set -e
endlog

log "резервні копії БД"
sleep 5
docker run --rm -v rf-ci-db-backups:/b alpine ls -la /b | tee "$E/db-backups.txt"
endlog

log "блокування стану"
L=infra/terraform/envs/locktest
tf $L init -backend-config="conn_str=$PG_CONN_STR" -backend-config="schema_name=locktest" >/dev/null
tf $L apply -auto-approve -var run_id="$GITHUB_RUN_ID-a" > "$E/lock-holder.txt" &
HOLDER=$!
sleep 8
set +e
tf $L plan -lock-timeout=0s -var run_id=x > "$E/lock-contender.txt" 2>&1
echo "second plan exit code: $?" >> "$E/lock-contender.txt"
set -e
wait $HOLDER
tf $L destroy -auto-approve -var run_id="$GITHUB_RUN_ID-a" >/dev/null
endlog

log "стан у PostgreSQL"
docker run --rm --network host postgres:16-alpine psql "$PG_CONN_STR" -c "select schemaname, relname from pg_stat_user_tables" -c "select name, length(data) as bytes from platform.states union all select name, length(data) from app.states" > "$E/state-backend.txt" 2>&1 || true
endlog

log "destroy"
for c in green blue; do tf $A workspace select $c >/dev/null; tf $A destroy -auto-approve -var image=x:1 > "$E/destroy-$c.txt"; done
tf $P destroy -auto-approve > "$E/destroy-platform.txt"
docker ps -a --filter label=project=roomflow --format '{{.Names}}' > "$E/leftovers.txt"
echo "containers left: $(wc -l < "$E/leftovers.txt")" | tee -a "$E/timings.txt"
endlog

section "Тривалість операцій" "$E/timings.txt"
section "Розподіл відповідей під час перемикання" "$E/switch-summary.txt"
section "Мережеві тести" "$E/connectivity.txt"
section "Дрейф" "$E/drift.txt"
