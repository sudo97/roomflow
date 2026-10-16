// Змішаний трафік для демонстрації спостережуваності (ЛР7): читання, бронювання, конфлікти, помилки клієнта.
import http from 'k6/http';
import { check } from 'k6';

const BASE = __ENV.BASE_URL || 'http://localhost:8080';
const TENANTS = ['hotel-alpha', 'hotel-beta', 'hotel-gamma'];

export const options = {
  scenarios: {
    traffic: {
      executor: 'constant-arrival-rate',
      rate: Number(__ENV.RATE || 30), timeUnit: '1s',
      duration: __ENV.DURATION || '3m', preAllocatedVUs: 20, maxVUs: 100,
    },
  },
};

export function setup() {
  for (const t of TENANTS) {
    for (let r = 1; r <= 5; r++) {
      http.post(`${BASE}/api/rooms`, JSON.stringify({ id: `r${r}`, name: `Room ${r}`, capacity: 3, price: 120000 }), { headers: { 'X-Tenant-ID': t } });
    }
  }
}

function day(offset) {
  const d = new Date(Date.UTC(2027, 0, 1) + offset * 86400000);
  return d.toISOString().slice(0, 10);
}

export default function () {
  const tenant = TENANTS[Math.floor(Math.random() * TENANTS.length)];
  const headers = { 'X-Tenant-ID': tenant, 'Content-Type': 'application/json' };
  const room = `r${1 + Math.floor(Math.random() * 5)}`;
  const p = Math.random();
  if (p < 0.4) {
    const res = http.get(`${BASE}/api/rooms/${room}/availability?from=${day(0)}&to=${day(30)}`, { headers, tags: { name: 'availability' } });
    check(res, { 'availability 200': (r) => r.status === 200 });
  } else if (p < 0.6) {
    http.get(`${BASE}/api/rooms`, { headers, tags: { name: 'rooms' } });
  } else if (p < 0.95) {
    const start = Math.floor(Math.random() * 400);
    const body = { room_id: room, guest_name: 'Load Test', guest_email: 'load@example.com', guests: 2, check_in: day(start), check_out: day(start + 1 + Math.floor(Math.random() * 3)) };
    const res = http.post(`${BASE}/api/bookings`, JSON.stringify(body), { headers, tags: { name: 'booking' } });
    check(res, { 'booking 201/409': (r) => r.status === 201 || r.status === 409 });
  } else {
    http.post(`${BASE}/api/bookings`, '{"room_id":"r1"}', { headers, tags: { name: 'invalid' } });
  }
}
