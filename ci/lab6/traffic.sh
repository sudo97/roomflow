#!/usr/bin/env bash
# Безперервний трафік на балансувальник: на кожен запит друкує "код колір".
URL=$1; RPS=${2:-20}
while true; do
  r=$(curl -s -m 2 -w ' %{http_code}' "$URL/version" || echo ' 000')
  code=${r##* }; color=$(echo "$r" | sed -n 's/.*"color":"\([a-z]*\)".*/\1/p')
  echo "$code ${color:-none} $(date +%s%3N)"
  sleep "$(awk -v r="$RPS" 'BEGIN{print 1/r}')"
done
