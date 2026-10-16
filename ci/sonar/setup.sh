#!/usr/bin/env bash
# Конфігурація SonarQube як коду: пароль адміністратора, проєкт, токен сканера,
# якісний профіль і ворота якості. Сервер одноразовий (CI), тому ідемпотентність не потрібна.
set -uo pipefail
: "${SONAR_HOST_URL:?}" "${SONAR_ADMIN_PASSWORD:?}"

# api METHOD PATH [curl args...] -> тіло відповіді в stdout; при помилці HTTP друкує тіло в stderr
api() {
  local method=$1 path=$2; shift 2
  local body code
  body=$(curl -sS -u "admin:${SONAR_ADMIN_PASSWORD}" -X "$method" -w '\n%{http_code}' "$SONAR_HOST_URL$path" "$@")
  code=${body##*$'\n'}; body=${body%$'\n'*}
  if [ "$code" -ge 400 ]; then echo "  ! $method $path -> HTTP $code: $body" >&2; return 1; fi
  printf '%s' "$body"
}

echo "== Зміна пароля адміністратора"
curl -sS -u admin:admin -X POST "$SONAR_HOST_URL/api/users/change_password" \
  --data-urlencode login=admin --data-urlencode previousPassword=admin \
  --data-urlencode "password=${SONAR_ADMIN_PASSWORD}"

echo "== Створення проєкту і токена сканера"
api POST /api/projects/create -d project=roomflow -d name=RoomFlow -d mainBranch=main >/dev/null
api POST /api/user_tokens/generate -d name=ci -d type=PROJECT_ANALYSIS_TOKEN -d projectKey=roomflow | jq -r .token > /tmp/sonar_token
test -s /tmp/sonar_token && echo "  token: ok"

echo "== Якісний профіль «RoomFlow Go» (успадковує Sonar way)"
api GET "/api/qualityprofiles/search?language=go" | jq -r '.profiles[]|"  \(.name)\t\(.key)\tdefault=\(.isDefault)"'
api POST /api/qualityprofiles/create -d language=go --data-urlencode "name=RoomFlow Go" | jq -r '.profile.key' > /tmp/qp
QP=$(cat /tmp/qp); echo "  key: $QP"
api POST /api/qualityprofiles/change_parent -d language=go --data-urlencode "qualityProfile=RoomFlow Go" --data-urlencode "parentQualityProfile=Sonar way" >/dev/null && echo "  parent: Sonar way"
activate() { # rule severity [params]
  if api POST /api/qualityprofiles/activate_rule -d key="$QP" -d rule="$1" -d severity="$2" ${3:+--data-urlencode "params=$3"} >/dev/null; then
    echo "  $1 -> $2 ${3:-}"
  fi
}
for r in go:S3776 go:S107 go:S138 go:S1192; do
  api GET "/api/rules/show?key=$r" | jq -r '"  rule \(.rule.key): \(.rule.name) params=\([.rule.params[]?.key]|join(","))"' || true
done
# Пороги, суворіші за стандартні (вимоги завдання 2)
activate go:S3776 CRITICAL "threshold=8"   # когнітивна складність функції <= 8
activate go:S107  MAJOR    "max=5"         # не більше 5 параметрів функції
activate go:S138  MAJOR    "max=50"        # довжина функції <= 50 рядків
activate go:S1192 MINOR    "threshold=3"   # дубльовані рядкові літерали
activate go:S2068 BLOCKER                  # жорстко закодовані паролі
api GET "/api/rules/search?qprofile=$QP&activation=true&ps=1" | jq -r '"  active rules: \(.total)"'
api POST /api/qualityprofiles/add_project -d language=go -d project=roomflow --data-urlencode "qualityProfile=RoomFlow Go" && echo "  assigned to roomflow"

echo "== Ворота якості «RoomFlow Gate»"
api POST /api/qualitygates/create --data-urlencode "name=RoomFlow Gate" >/dev/null
cond() { api POST /api/qualitygates/create_condition --data-urlencode "gateName=RoomFlow Gate" -d metric="$1" -d op="$2" -d error="$3" >/dev/null && echo "  $1 $2 $3"; }
cond coverage                   LT 70   # покриття тестами >= 70%
cond reliability_rating         GT 2    # надійність не гірше B
cond security_rating            GT 1    # безпека A
cond sqale_debt_ratio           GT 5    # технічний борг <= 5%
cond duplicated_lines_density   GT 3    # дублювання <= 3%
cond security_hotspots_reviewed LT 100  # усі hotspots переглянуті
api POST /api/qualitygates/select --data-urlencode "gateName=RoomFlow Gate" -d projectKey=roomflow && echo "  assigned to roomflow"
echo "done"
