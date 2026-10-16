#!/usr/bin/env bash
# Перевірка мережевої сегментації (аналог груп безпеки і NAT gateway). Кожен рядок: PASS/FAIL опис.
P=$1
ok() { echo "PASS $*"; }
bad() { echo "FAIL $*"; FAILED=1; }
FAILED=0
img=busybox:1.36

docker run --rm --network "$P-private" $img wget -q -T 4 -O /dev/null http://example.com \
  && bad "приватна мережа має доступ до Інтернету" || ok "приватна мережа ізольована від Інтернету (internal)"

cid=$(docker create --network "$P-private" $img wget -q -T 6 -O /dev/null http://example.com)
docker network connect "$P-egress" "$cid"
docker start -a "$cid" && ok "ресурс у приватній мережі через egress (NAT) має вихідний доступ" || bad "egress не працює"
docker rm -f "$cid" >/dev/null

docker run --rm --network "$P-public" postgres:16-alpine pg_isready -h db -t 3 >/dev/null 2>&1 \
  && bad "БД доступна з публічної мережі" || ok "БД недоступна з публічної мережі"
docker run --rm --network "$P-private" postgres:16-alpine pg_isready -h db -t 5 >/dev/null 2>&1 \
  && ok "БД доступна з приватної мережі застосунку" || bad "БД недоступна для застосунку"

docker port "$P-db" 2>/dev/null | grep -q . && bad "БД публікує порти" || ok "БД не публікує портів на хості"
exit $FAILED
