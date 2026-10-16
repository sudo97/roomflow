// ЛР9, індивідуальне завдання 8: перевірка обмежувача швидкості на рівні орендаря.
// "Галасливий" орендар надсилає 200 запитів/с, звичайний — 20 запитів/с; ліміт — RATE_LIMIT_RPS на орендаря.
import http from 'k6/http';
import { check } from 'k6';
import { Counter } from 'k6/metrics';

const BASE = __ENV.BASE_URL || 'http://localhost:8080';
const limited = new Counter('rate_limited');

http.setResponseCallback(http.expectedStatuses(200, 429));

export const options = {
  scenarios: {
    noisy: { executor: 'constant-arrival-rate', rate: 200, timeUnit: '1s', duration: '40s', preAllocatedVUs: 50, maxVUs: 200, env: { TENANT: 'hotel-0' }, tags: { tenant: 'noisy' } },
    regular: { executor: 'constant-arrival-rate', rate: 20, timeUnit: '1s', duration: '40s', preAllocatedVUs: 10, maxVUs: 50, env: { TENANT: 'hotel-1' }, tags: { tenant: 'regular' } },
  },
  thresholds: {
    'checks{tenant:regular}': ['rate>0.99'],
    'rate_limited{tenant:regular}': ['count<1'],
    'rate_limited{tenant:noisy}': ['count>1000'],
  },
};

export default function () {
  const res = http.get(`${BASE}/api/rooms/r1/availability?from=2027-01-01&to=2027-01-15`, { headers: { 'X-Tenant-ID': __ENV.TENANT } });
  if (res.status === 429) limited.add(1);
  check(res, { 'served or limited': (r) => r.status === 200 || r.status === 429, 'served': (r) => r.status === 200 });
}
