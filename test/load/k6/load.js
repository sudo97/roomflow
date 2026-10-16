// ЛР9: навантажувальний тест однієї кінцевої точки з наростанням 10 -> 100 віртуальних користувачів.
// ENDPOINT: health | availability | booking. Пороги визначають успішність тесту (код виходу 99 при порушенні).
import http from 'k6/http';
import { check } from 'k6';

const BASE = __ENV.BASE_URL || 'http://localhost:8080';
const ENDPOINT = __ENV.ENDPOINT || 'availability';
const TENANTS = Number(__ENV.TENANTS || 20);
const ROOMS = Number(__ENV.ROOMS || 10);

http.setResponseCallback(http.expectedStatuses(200, 201, 409));

export const options = {
  scenarios: {
    ramp: {
      executor: 'ramping-vus',
      startVUs: 10,
      stages: [
        { duration: __ENV.RAMP || '60s', target: Number(__ENV.MAX_VUS || 100) },
        { duration: __ENV.HOLD || '30s', target: Number(__ENV.MAX_VUS || 100) },
      ],
      gracefulRampDown: '5s',
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: [`p(95)<${__ENV.P95_MS || 500}`],
    checks: ['rate>0.99'],
  },
  summaryTrendStats: ['avg', 'min', 'med', 'p(90)', 'p(95)', 'p(99)', 'max'],
};

function day(offset) {
  return new Date(Date.UTC(2027, 0, 1) + offset * 86400000).toISOString().slice(0, 10);
}

const requests = {
  health: () => http.get(`${BASE}/healthz`, { tags: { name: 'health' } }),
  availability: (t, r) => http.get(`${BASE}/api/rooms/r${r}/availability?from=${day(0)}&to=${day(60)}`,
    { headers: { 'X-Tenant-ID': `hotel-${t}` }, tags: { name: 'availability' } }),
  booking: (t, r) => {
    const start = 3000 + Math.floor(Math.random() * 20000);
    const body = JSON.stringify({ room_id: `r${r}`, guest_name: 'k6', guest_email: 'k6@example.com', guests: 2, check_in: day(start), check_out: day(start + 1) });
    return http.post(`${BASE}/api/bookings`, body, { headers: { 'X-Tenant-ID': `hotel-${t}`, 'Content-Type': 'application/json' }, tags: { name: 'booking' } });
  },
};

export default function () {
  const t = Math.floor(Math.random() * TENANTS);
  const r = 1 + Math.floor(Math.random() * ROOMS);
  const res = requests[ENDPOINT](t, r);
  check(res, { 'status is expected': (x) => [200, 201, 409].includes(x.status) });
}
