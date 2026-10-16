#!/usr/bin/env bash
# Smoke-тест розгорнутого екземпляра: здоров'я, створення номера, бронювання, конфлікт, скасування.
set -euo pipefail
BASE=${1:-http://localhost:8080}
T="X-Tenant-ID: smoke-$(date +%s)"
check() { # name expected_status curl-args...
  local name=$1 want=$2; shift 2
  local got; got=$(curl -s -o /tmp/smoke.out -w '%{http_code}' "$@")
  if [ "$got" = "$want" ]; then echo "PASS $name ($got)"; else echo "FAIL $name: got $got, want $want: $(cat /tmp/smoke.out)"; exit 1; fi
}
for i in $(seq 1 30); do curl -sf "$BASE/healthz" >/dev/null && break; sleep 1; done
check healthz 200 "$BASE/healthz"
check add-room 201 -X POST -H "$T" "$BASE/api/rooms" -d '{"id":"r1","name":"Double","capacity":2,"price":120000}'
check book 201 -X POST -H "$T" "$BASE/api/bookings" -d '{"room_id":"r1","guest_name":"Smoke","guest_email":"smoke@example.com","guests":2,"check_in":"2030-05-04","check_out":"2030-05-06"}'
check double-booking 409 -X POST -H "$T" "$BASE/api/bookings" -d '{"room_id":"r1","guest_name":"Other","guest_email":"o@example.com","guests":1,"check_in":"2030-05-05","check_out":"2030-05-07"}'
check cancel 204 -X POST -H "$T" "$BASE/api/bookings/bk-1/cancel"
echo "smoke OK"
