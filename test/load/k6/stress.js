// ЛР9, індивідуальне завдання 1: стрес-тест з агресивним наростанням до 500 VU для пошуку точки відмови.
import http from 'k6/http';
import { check } from 'k6';

const BASE = __ENV.BASE_URL || 'http://localhost:8080';

export const options = {
  scenarios: {
    stress: {
      executor: 'ramping-vus',
      startVUs: 10,
      stages: [
        { duration: '30s', target: 100 },
        { duration: '30s', target: 250 },
        { duration: '30s', target: 500 },
        { duration: '30s', target: 500 },
        { duration: '15s', target: 0 },
      ],
    },
  },
  thresholds: {
    // Інформаційні пороги: стрес-тест не блокує, а фіксує, де система перестає відповідати SLO.
    'http_req_duration': [{ threshold: 'p(95)<1000', abortOnFail: false }],
    'http_req_failed': [{ threshold: 'rate<0.05', abortOnFail: false }],
  },
  summaryTrendStats: ['avg', 'med', 'p(95)', 'p(99)', 'max'],
};

function day(offset) {
  return new Date(Date.UTC(2027, 0, 1) + offset * 86400000).toISOString().slice(0, 10);
}

export default function () {
  const t = Math.floor(Math.random() * 20), r = 1 + Math.floor(Math.random() * 10);
  const res = http.get(`${BASE}/api/rooms/r${r}/availability?from=${day(0)}&to=${day(60)}`,
    { headers: { 'X-Tenant-ID': `hotel-${t}` }, timeout: '10s' });
  check(res, { ok: (x) => x.status === 200 });
}
