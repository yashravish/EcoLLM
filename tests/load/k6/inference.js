/**
 * k6 load test: EcoLLM inference endpoint
 *
 * Run: k6 run --env API_URL=http://localhost:8080 --env API_KEY=ek_test_... inference.js
 *
 * Stages:
 *   0→30s  ramp to 10 VUs  (warm-up)
 *   30→90s hold at 10 VUs  (steady state)
 *   90→120s hold at 50 VUs (spike)
 *   120→150s ramp to 0     (cool-down)
 *
 * SLOs:
 *   p95 latency < 5000ms
 *   error rate  < 1%
 */

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

const errorRate = new Rate('errors');
const routingLatency = new Trend('routing_latency_ms', true);
const co2ePerRequest = new Trend('co2e_grams');

export const options = {
  stages: [
    { duration: '30s', target: 10 },
    { duration: '60s', target: 10 },
    { duration: '30s', target: 50 },
    { duration: '30s', target: 0 },
  ],
  thresholds: {
    http_req_duration: ['p(95)<5000'],
    errors: ['rate<0.01'],
  },
};

const API_URL = __ENV.API_URL || 'http://localhost:8080';
const API_KEY = __ENV.API_KEY || 'test-key';

const PROMPTS = [
  'What is the capital of France?',
  'Summarize the concept of entropy in thermodynamics.',
  'Write a Python function to reverse a linked list.',
  'Explain the difference between supervised and unsupervised learning.',
  'What are the main causes of climate change?',
];

export default function () {
  const prompt = PROMPTS[Math.floor(Math.random() * PROMPTS.length)];

  const payload = JSON.stringify({
    messages: [{ role: 'user', content: prompt }],
    max_tokens: 256,
    ecollm: { prefer: 'efficiency', include_metadata: true },
  });

  const res = http.post(`${API_URL}/v1/chat/completions`, payload, {
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${API_KEY}`,
    },
    timeout: '10s',
  });

  const ok = check(res, {
    'status 200': (r) => r.status === 200,
    'has choices': (r) => {
      try {
        const body = JSON.parse(r.body);
        return Array.isArray(body.choices) && body.choices.length > 0;
      } catch {
        return false;
      }
    },
    'has ecollm metadata': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.ecollm && body.ecollm.route && body.ecollm.energy;
      } catch {
        return false;
      }
    },
  });

  errorRate.add(!ok);

  if (res.status === 200) {
    try {
      const body = JSON.parse(res.body);
      if (body.ecollm?.performance?.latency_ms) {
        routingLatency.add(body.ecollm.performance.latency_ms);
      }
      if (body.ecollm?.energy?.co2e_grams) {
        co2ePerRequest.add(body.ecollm.energy.co2e_grams);
      }
    } catch (_) {
      // non-fatal
    }
  }

  sleep(Math.random() * 2 + 0.5); // 0.5–2.5s think time
}
