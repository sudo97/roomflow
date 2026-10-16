# Виконується всередині пода probe (sh): loop.sh URL [N] [PATH]
# Друкує на кожен запит: <unix_s> <http_code> <color>. N=0 — нескінченно.
URL=$1; N=${2:-0}; P=${3:-/version}; i=0
while [ "$N" -eq 0 ] || [ "$i" -lt "$N" ]; do
  i=$((i + 1))
  r=$(curl -s -m 2 -w ' %{http_code}' "$URL$P")
  code=${r##* }
  color=$(echo "$r" | sed -n 's/.*"color":"\([^"]*\)".*/\1/p')
  echo "$(date +%s) $code ${color:--}"
  [ "$N" -eq 0 ] && sleep 0.05
done
