#!/usr/bin/env bash
# ЛР11, завдання 3: Crossplane — абстракція PostgreSQLInstance (XRD + Composition) і самообслуговування через claim.
source "$(dirname "$0")/../lib/common.sh"
E=$EVIDENCE/crossplane; mkdir -p "$E"
install_helm
kind_up rf-xp
helm repo add crossplane-stable https://charts.crossplane.io/stable >/dev/null && helm repo update >/dev/null
helm install crossplane crossplane-stable/crossplane --version 1.17.1 -n crossplane-system --create-namespace --wait --timeout 5m > "$E/install.txt"
kubectl apply -f platform/crossplane/00-packages.yaml
kubectl wait provider.pkg.crossplane.io/provider-kubernetes --for=condition=Healthy --timeout=300s
kubectl wait function.pkg.crossplane.io/function-patch-and-transform --for=condition=Healthy --timeout=300s
kubectl apply -f platform/crossplane/01-providerconfig.yaml -f platform/crossplane/02-xrd.yaml
kubectl wait xrd xpostgresqlinstances.platform.roomflow.io --for=condition=Established --timeout=120s
kubectl apply -f platform/crossplane/03-composition.yaml
kubectl get providers,functions,xrd,compositions > "$E/platform-api.txt"

provision() { # name namespace size
  kubectl create namespace "$2" --dry-run=client -o yaml | kubectl apply -f - >/dev/null
  local s; s=$(date +%s)
  go run ./tools/rfctl provision --name "$1" --namespace "$2" --size "$3" | kubectl apply -f -
  kubectl -n "$2" wait postgresqlinstance/"$1" --for=condition=Ready --timeout=300s
  kubectl -n "$2" wait deploy/"$1"-postgres --for=condition=Available --timeout=300s
  kubectl -n "$2" exec deploy/"$1"-postgres -- pg_isready -U app
  echo "$1 ($3) in $2: ready in $(( $(date +%s) - s )) s" | tee -a "$E/provisioning.txt"
}
provision reviews-db team-reviews small
provision analytics-db team-analytics medium
{
  kubectl get postgresqlinstances -A
  kubectl get xpostgresqlinstances
  kubectl get objects.kubernetes.crossplane.io
  kubectl -n team-reviews get secret,pvc,deploy,svc
  kubectl -n team-analytics get deploy analytics-db-postgres -o jsonpath='{"memory limit: "}{.spec.template.spec.containers[0].resources.limits.memory}{"\n"}'
  kubectl -n team-reviews get secret reviews-db-conn -o jsonpath='{.data}' | jq 'with_entries(.value |= (@base64d | if length > 12 then .[0:6] + "…" else . end))'
} > "$E/resources.txt" 2>&1
kubectl -n team-reviews describe postgresqlinstance reviews-db > "$E/claim-describe.txt"
# Видалення claim прибирає всі створені ресурси.
s=$(date +%s)
kubectl -n team-reviews delete postgresqlinstance reviews-db --wait=true --timeout=180s
left() { kubectl -n team-reviews get deploy,svc,secret,pvc --no-headers 2>/dev/null | grep -c reviews-db || true; }
for _ in $(seq 1 90); do [ "$(left)" = 0 ] && break; sleep 2; done
echo "deprovisioning: $(( $(date +%s) - s )) s; leftovers: $(left)" | tee -a "$E/provisioning.txt"
section "Crossplane" "$E/provisioning.txt"
