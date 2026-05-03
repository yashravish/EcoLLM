/**
 * k6 load test: EcoLLM billing and usage endpoints
 *
 * Run: k6 run --env API_URL=http://localhost:8080 --env API_KEY=ek_test_... billing-api.js
 *
 * Verifies:
 *   - /v1/billing responds quickly (< 500ms p95)
 *   - /v1/usage responds quickly (< 500ms p95)
 *   - /v1/requests paginated list responds (< 500ms p95)
 */

import http from 'k6/http';
import { check, group, sleep } from 'k6';
import { Rate } from 'k6/metrics';

const errorRate = new Rate('errors');

export const options = {
  stages: [
    { duration: '20s', target: 20 },
    { duration: '40s', target: 20 },
    { duration: '10s', target: 0 },
  ],
  thresholds: {
    'http_req_duration{endpoint:billing}': ['p(95)<500'],
    'http_req_duration{endpoint:usage}': ['p(95)<500'],
    'http_req_duration{endpoint:requests}': ['p(95)<500'],
    errors: ['rate<0.01'],
  },
};

const API_URL = __ENV.API_URL || 'http://localhost:8080';
const API_KEY = __ENV.API_KEY || 'test-key';

const headers = {
  'Content-Type': 'application/json',
  Authorization: `Bearer ${API_KEY}`,
};

export default function () {
  group('billing endpoint', () => {
    const res = http.get(`${API_URL}/v1/billing`, { headers, tags: { endpoint: 'billing' } });
    const ok = check(res, {
      'billing status 200': (r) => r.status === 200,
      'billing has events array': (r) => {
        try {
          const b = JSON.parse(r.body);
          return Array.isArray(b.events);
        } catch {
          return false;
        }
      },
    });
    errorRate.add(!ok);
  });

  sleep(0.2);

  group('usage endpoint', () => {
    const res = http.get(
      `${API_URL}/v1/usage?period=daily`,
      { headers, tags: { endpoint: 'usage' } },
    );
    const ok = check(res, {
      'usage status 200': (r) => r.status === 200,
      'usage has summary': (r) => {
        try {
          const b = JSON.parse(r.body);
          return b.summary !== undefined;
        } catch {
          return false;
        }
      },
    });
    errorRate.add(!ok);
  });

  sleep(0.2);

  group('requests list endpoint', () => {
    const res = http.get(
      `${API_URL}/v1/requests?limit=20`,
      { headers, tags: { endpoint: 'requests' } },
    );
    const ok = check(res, {
      'requests status 200': (r) => r.status === 200,
      'requests has array': (r) => {
        try {
          const b = JSON.parse(r.body);
          return Array.isArray(b.requests);
        } catch {
          return false;
        }
      },
    });
    errorRate.add(!ok);
  });

  sleep(Math.random() * 1 + 0.5);
}
