#!/usr/bin/env bash
# ЛР8, завдання 4: HashiCorp Vault — ініціалізація, розпечатування, KV v2, політика, AppRole, аудит,
# отримання секрету застосунком під час запуску та ротація без зміни коду.
source "$(dirname "$0")/../lib/common.sh"
E=$EVIDENCE
V=hashicorp/vault:1.17.6
NET=rf-vault
docker network create $NET >/dev/null
mkdir -p /tmp/vault && cat > /tmp/vault/config.hcl <<'HCL'
storage "file" { path = "/vault/file" }
listener "tcp" {
  address     = "0.0.0.0:8200"
  tls_disable = 1   # лише для навчального стенду; у production — TLS
}
ui           = true
disable_mlock = true
HCL
docker run -d --name vault --network $NET -p 8200:8200 --cap-add IPC_LOCK -v /tmp/vault:/vault/config:ro $V server >/dev/null
export VAULT_ADDR=http://localhost:8200
vault() { docker exec -e VAULT_ADDR=http://127.0.0.1:8200 -e VAULT_TOKEN="${VAULT_TOKEN:-}" vault vault "$@"; }
sleep 3

log "init / unseal"
vault operator init -key-shares=1 -key-threshold=1 -format=json > /tmp/vault/init.json
UNSEAL=$(jq -r '.unseal_keys_b64[0]' /tmp/vault/init.json); export VAULT_TOKEN=$(jq -r .root_token /tmp/vault/init.json)
vault status > "$E/vault-status-sealed.txt" 2>&1 || true
vault operator unseal "$UNSEAL" > /dev/null
vault status > "$E/vault-status.txt"
endlog

log "secrets engine, policy, approle, audit"
vault audit enable file file_path=/tmp/audit.log >/dev/null
vault secrets enable -path=secret kv-v2 >/dev/null
ADMIN1=$(openssl rand -hex 16)
vault kv put secret/roomflow admin_token="$ADMIN1" db_username=roomflow db_password="$(openssl rand -hex 16)" >/dev/null
docker exec -i -e VAULT_ADDR=http://127.0.0.1:8200 -e VAULT_TOKEN="$VAULT_TOKEN" vault vault policy write roomflow - <<'POL'
# Принцип найменших привілеїв: застосунок може лише читати власні секрети.
path "secret/data/roomflow" { capabilities = ["read"] }
POL
vault policy read roomflow > "$E/vault-policy.txt"
vault auth enable approle >/dev/null
vault write auth/approle/role/roomflow token_policies=roomflow token_ttl=15m token_max_ttl=1h secret_id_ttl=10m secret_id_num_uses=1 >/dev/null
ROLE_ID=$(vault read -field=role_id auth/approle/role/roomflow/role-id)
endlog

app_token() { # AppRole login: одноразовий secret_id -> короткоживучий токен з політикою roomflow
  local sid; sid=$(vault write -f -field=secret_id auth/approle/role/roomflow/secret-id)
  curl -s -X POST $VAULT_ADDR/v1/auth/approle/login -d "{\"role_id\":\"$ROLE_ID\",\"secret_id\":\"$sid\"}" | jq -r .auth.client_token
}
start_app() {
  docker rm -f rf-app >/dev/null 2>&1 || true
  docker run -d --name rf-app --network $NET -p 8080:8080 -e VAULT_ADDR=http://vault:8200 -e VAULT_TOKEN="$(app_token)" roomflow:vault >/dev/null
  wait_http http://localhost:8080/healthz 30
}
code() { curl -s -o /dev/null -w '%{http_code}' -X POST -H "Authorization: Bearer $1" http://localhost:8080/admin/flags/reload; }

log "застосунок отримує секрет із Vault під час запуску"
build_image roomflow:vault
start_app
{
  echo "admin token з Vault:      $(code "$ADMIN1")  (очікується 204)"
  echo "довільний токен:          $(code wrong)  (очікується 401)"
  APPTOKEN=$(app_token)
  echo "токен застосунку читає secret/roomflow: $(curl -s -o /dev/null -w '%{http_code}' -H "X-Vault-Token: $APPTOKEN" $VAULT_ADDR/v1/secret/data/roomflow)"
  echo "токен застосунку пише secret/roomflow:  $(curl -s -o /dev/null -w '%{http_code}' -X POST -H "X-Vault-Token: $APPTOKEN" $VAULT_ADDR/v1/secret/data/roomflow -d '{"data":{"x":"y"}}') (очікується 403)"
  echo "токен застосунку читає sys/policy:      $(curl -s -o /dev/null -w '%{http_code}' -H "X-Vault-Token: $APPTOKEN" $VAULT_ADDR/v1/sys/policy) (очікується 403)"
} | tee "$E/vault-app.txt"
endlog

log "ротація секрету"
ADMIN2=$(openssl rand -hex 16)
vault kv patch secret/roomflow admin_token="$ADMIN2" >/dev/null
start_app
{
  echo "після ротації: новий токен $(code "$ADMIN2"), старий токен $(code "$ADMIN1") (очікується 204 і 401)"
  vault kv metadata get -format=json secret/roomflow | jq -r '"версій секрету: \(.data.current_version)"'
} | tee -a "$E/vault-app.txt"
endlog

log "журнал аудиту"
docker exec vault sh -c 'cat /tmp/audit.log' | jq -c 'select(.type=="response") | {time, path: .request.path, op: .request.operation, policies: .auth.policies, error: .error}' | tail -15 > "$E/vault-audit.txt"
echo "входжень значення секрету у відкритому вигляді в журналі аудиту: $(docker exec vault sh -c "grep -c '$ADMIN2' /tmp/audit.log" || true) (значення зберігаються як HMAC)" > "$E/vault-audit-plaintext-check.txt"
docker logs rf-app 2>&1 | head -3 > "$E/vault-app-log.txt"
endlog
docker rm -f rf-app vault >/dev/null; docker network rm $NET >/dev/null
section "Vault" "$E/vault-app.txt"
