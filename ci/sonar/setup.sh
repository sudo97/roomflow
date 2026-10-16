#!/usr/bin/env bash
# Конфігурація SonarQube як коду: пароль адміністратора, проєкт, якісний профіль,
# ворота якості, токен для сканера. Ідемпотентність не потрібна: сервер одноразовий (CI).
set -euo pipefail
: "${SONAR_HOST_URL:?}" "${SONAR_ADMIN_PASSWORD:?}"
api() { curl -sS -u "admin:${SONAR_ADMIN_PASSWORD}" "$@"; }

echo "== Зміна пароля адміністратора"
curl -sS -u admin:admin -X POST "$SONAR_HOST_URL/api/users/change_password" \
  --data-urlencode login=admin --data-urlencode previousPassword=admin \
  --data-urlencode "password=${SONAR_ADMIN_PASSWORD}"

echo "== Створення проєкту"
api -X POST "$SONAR_HOST_URL/api/projects/create" -d project=roomflow -d name=RoomFlow -d mainBranch=main >/dev/null

echo "== Якісний профіль RoomFlow Go (копія Sonar way)"
SW=$(api "$SONAR_HOST_URL/api/qualityprofiles/search?language=go" | jq -r '.profiles[]|select(.name=="Sonar way")|.key')
QP=$(api -X POST "$SONAR_HOST_URL/api/qualityprofiles/copy" -d fromKey="$SW" --data-urlencode "toName=RoomFlow Go" | jq -r .key)
activate() { # rule severity params
  api -X POST "$SONAR_HOST_URL/api/qualityprofiles/activate_rule" -d key="$QP" -d rule="$1" -d severity="$2" ${3:+--data-urlencode "params=$3"} \
    && echo "  $1 -> $2 ${3:-}" || echo "  WARN: $1 not activated"
}
# Пороги, суворіші за стандартні (вимоги завдання 2)
activate go:S3776 CRITICAL "threshold=8"    # когнітивна складність функції <= 8
activate go:S107  MAJOR    "max=5"          # не більше 5 параметрів функції
activate go:S138  MAJOR    "max=50"         # довжина функції <= 50 рядків
activate go:S1192 MINOR    "threshold=3"    # дубльовані рядкові літерали
activate go:S2068 BLOCKER  ""               # жорстко закодовані паролі
api "$SONAR_HOST_URL/api/rules/search?qprofile=$QP&activation=true&ps=500" | jq '{active_rules: .total}'
api -X POST "$SONAR_HOST_URL/api/qualityprofiles/add_project" -d language=go -d project=roomflow --data-urlencode "qualityProfile=RoomFlow Go"

echo "== Ворота якості RoomFlow Gate"
api -X POST "$SONAR_HOST_URL/api/qualitygates/create" --data-urlencode "name=RoomFlow Gate" >/dev/null
cond() { api -X POST "$SONAR_HOST_URL/api/qualitygates/create_condition" --data-urlencode "gateName=RoomFlow Gate" -d metric="$1" -d op="$2" -d error="$3" >/dev/null && echo "  $1 $2 $3"; }
cond coverage                 LT 70   # покриття тестами >= 70%
cond reliability_rating       GT 2    # надійність не гірше B
cond security_rating          GT 1    # безпека A
cond sqale_debt_ratio         GT 5    # технічний борг <= 5%
cond duplicated_lines_density GT 3    # дублювання <= 3%
cond security_hotspots_reviewed LT 100 # усі hotspots переглянуті
api -X POST "$SONAR_HOST_URL/api/qualitygates/select" --data-urlencode "gateName=RoomFlow Gate" -d projectKey=roomflow

echo "== Токен сканера"
api -X POST "$SONAR_HOST_URL/api/user_tokens/generate" -d name=ci -d type=PROJECT_ANALYSIS_TOKEN -d projectKey=roomflow | jq -r .token > /tmp/sonar_token
echo "done"
