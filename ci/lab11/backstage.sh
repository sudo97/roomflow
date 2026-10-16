#!/usr/bin/env bash
# ЛР11, завдання 1–2: портал Backstage з каталогом RoomFlow (Component, API, Resource, System)
# і Software Template; PostgreSQL для catalog backend. Обмеження стенду: замість GitHub OAuth
# використано guest-провайдер (OAuth App потребує секретів організації), портал запускається процесом на runner-і.
source "$(dirname "$0")/../lib/common.sh"
E=$EVIDENCE/backstage; mkdir -p "$E"
REPO=$PWD
BS=/tmp/portal
log "create-app"
# Backstage вимагає Node.js 22/24 LTS
if [ "$(node -p 'process.versions.node.split(".")[0]')" -lt 22 ]; then
  curl -sSL https://nodejs.org/dist/v22.11.0/node-v22.11.0-linux-x64.tar.xz | tar -xJ -C /tmp
  export PATH=/tmp/node-v22.11.0-linux-x64/bin:$PATH
fi
node --version
corepack enable
printf 'roomflow-portal\n' | npx -y @backstage/create-app@latest --path $BS --skip-install > "$E/create-app.txt" 2>&1 || { cat "$E/create-app.txt"; exit 1; }
cd $BS
export YARN_ENABLE_IMMUTABLE_INSTALLS=false   # у CI (CI=true) Yarn за замовчуванням забороняє змінювати lockfile
if ! yarn install > "$E/yarn-install.txt" 2>&1; then
  # пакет @backstage/create-app не містить каталогу .yarn/patches шаблону, а package.json/yarn.lock
  # посилаються на патчі звідти (patch:got@npm%3A11.8.2#~/.yarn/patches/...). Відновлюємо їх із репозиторію Backstage.
  grep -rn "patch:" --include=package.json --include=.yarnrc.yml . 2>/dev/null | grep -v node_modules > "$E/patch-refs.txt" || true
  grep -rln "yarn/patches" . --exclude-dir=node_modules --exclude-dir=cache 2>/dev/null >> "$E/patch-refs.txt" || true
  cat "$E/patch-refs.txt"
  mkdir -p .yarn/patches
  for p in $(grep -rhoE '\.yarn/patches/[^"#]+\.patch' --include=package.json --include=yarn.lock . 2>/dev/null | sort -u); do
    curl -fsSL "https://raw.githubusercontent.com/backstage/backstage/master/packages/create-app/templates/default-app/$p" -o "$p" \
      && echo "restored $p" || echo "not found upstream: $p"
  done | tee -a "$E/patch-refs.txt"
  if ! yarn install > "$E/yarn-install-retry.txt" 2>&1; then
    # запасний варіант: прибрати patch:-посилання (патч стосується лише сумісності got) і перебудувати lockfile
    grep -rlE 'patch:[^" ]*yarn/patches' . --exclude-dir=node_modules --exclude-dir=cache 2>/dev/null \
      | xargs -r sed -i -E 's#"?patch:[^"@ ]+@npm%3A([^"# ]+)\#[^" ]*"?#"\1"#g'
    rm -f yarn.lock
    yarn install > "$E/yarn-install-retry2.txt" 2>&1 || { tail -50 "$E/yarn-install-retry2.txt"; exit 1; }
  fi
fi
grep -m1 '"version"' node_modules/@backstage/cli/package.json | tee "$E/versions.txt"
yarn add --cwd packages/backend pg >/dev/null 2>&1 || (cd packages/backend && yarn add pg >/dev/null)
endlog

cat > app-config.local.yaml <<YAML
backend:
  database:
    client: pg
    connection: { host: localhost, port: 5432, user: backstage, password: backstage }
auth:
  providers:
    guest:
      dangerouslyAllowOutsideDevelopment: true
catalog:
  rules:
    - allow: [Component, System, API, Resource, Location, Group, User, Domain, Template]
  locations:
    - { type: file, target: $REPO/catalog-info.yaml }
    - { type: file, target: $REPO/platform/templates/service/template.yaml, rules: [{ allow: [Template] }] }
YAML

log "start portal"
(yarn start > "$E/portal.log" 2>&1 &)
wait_http http://localhost:7007/.backstage/health/v1/readiness 600
wait_http http://localhost:3000 600
sleep 30
endlog

log "catalog API і dry-run шаблону"
TOKEN=$(curl -s -X POST http://localhost:7007/api/auth/guest/refresh | jq -r '.backstageIdentity.token // empty')
[ -z "$TOKEN" ] && TOKEN=$(curl -s http://localhost:7007/api/auth/guest/refresh | jq -r '.backstageIdentity.token // empty')
AUTH="Authorization: Bearer $TOKEN"
curl -s -H "$AUTH" "http://localhost:7007/api/catalog/entities?fields=kind,metadata.name,spec.type,spec.owner" | jq -r '.[] | "\(.kind) \(.metadata.name) \(.spec.type // "") \(.spec.owner // "")"' | sort | tee "$E/entities.txt"
pip install -q pyyaml 2>/dev/null || pip install -q --break-system-packages pyyaml
python3 - "$REPO/platform/templates/service" > /tmp/dryrun.json <<'PY'
import base64, json, os, sys, yaml
root = sys.argv[1]
tmpl = yaml.safe_load(open(os.path.join(root, "template.yaml")))
files = []
for d, _, fs in os.walk(root):
    for f in fs:
        p = os.path.join(d, f)
        files.append({"path": os.path.relpath(p, root), "base64Content": base64.b64encode(open(p, "rb").read()).decode()})
values = {"name": "guest-reviews", "description": "Відгуки гостей", "owner": "group:default/platform-team",
          "language": "go", "database": "postgres", "repoUrl": "github.com?owner=sudo97&repo=guest-reviews"}
print(json.dumps({"template": tmpl, "values": values, "secrets": {}, "directoryContents": files}))
PY
curl -s -H "$AUTH" -H 'Content-Type: application/json' -X POST http://localhost:7007/api/scaffolder/v2/dry-run --data @/tmp/dryrun.json > "$E/dry-run.json" || true
jq -r '(.steps // [])[] | "step \(.id): \(.name)"' "$E/dry-run.json" | tee "$E/dry-run.txt" || true
jq -r '(.directoryContents // [])[] | .path' "$E/dry-run.json" | sort | tee -a "$E/dry-run.txt" || true
jq -r '(.log // [])[] | .body.message' "$E/dry-run.json" | tail -20 >> "$E/dry-run.txt" || true
endlog

log "screenshots"
playwright_setup
NODE_PATH=/tmp/pw/node_modules node "$REPO/ci/lab11/backstage-shots.mjs" "$E" || true
endlog
section "Каталог Backstage" "$E/entities.txt"
section "Dry-run шаблону" "$E/dry-run.txt"
