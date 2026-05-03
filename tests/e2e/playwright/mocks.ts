import type { Page } from '@playwright/test';

const TEST_USER = { id: 'user-1', email: 'admin@example.com', name: 'Admin User', role: 'admin' };
const TEST_ORG = { id: 'org-1', name: 'Test Org', slug: 'test-org', plan: 'pro' };
const INVALID_EMAIL = 'nobody@example.com';

function json(body: unknown) {
  return { status: 200, contentType: 'application/json', body: JSON.stringify(body) };
}

function err(status: number, message: string) {
  return { status, contentType: 'application/json', body: JSON.stringify({ code: status, message, type: 'error' }) };
}

export async function setupApiMocks(page: Page) {
  await page.route('**/auth/login', async (route) => {
    let body: { email?: string } = {};
    try { body = route.request().postDataJSON() ?? {}; } catch { /* ignore */ }
    if (body?.email === INVALID_EMAIL) {
      await route.fulfill(err(401, 'Invalid credentials'));
    } else {
      await route.fulfill(json({ token: 'test-token', user: TEST_USER, org: TEST_ORG }));
    }
  });

  await page.route('**/auth/logout', async (route) => {
    await route.fulfill(json({}));
  });

  await page.route('**/me', async (route) => {
    await route.fulfill(json({ user: TEST_USER, org: TEST_ORG }));
  });

  await page.route('**/v1/usage**', async (route) => {
    await route.fulfill(json({
      org_id: 'org-1',
      period: '30d',
      from: '2026-04-01',
      to: '2026-05-01',
      summary: {
        total_requests: 1234,
        total_tokens: 567890,
        total_energy_kwh: 0.045,
        total_co2e_grams: 18.5,
        total_cost_usd: 12.34,
        cache_hit_rate: 0.32,
        avg_latency_ms: 420,
      },
      model_distribution: { 'claude-haiku': 0.6, 'claude-sonnet': 0.4 },
      daily_breakdown: [],
    }));
  });

  await page.route('**/v1/requests**', async (route) => {
    if (route.request().method() === 'GET') {
      const sampleRequest = {
        id: 'req-1', request_id: 'req-uuid-1', prompt_original: 'Hello', task_type: 'general',
        complexity: 0.3, model_selected: 'claude-haiku', routing_score: 0.9, cache_hit: false,
        used_fallback: false, prompt_tokens: 10, completion_tokens: 5, total_tokens: 15,
        latency_ms: 350, status: 'completed', energy_kwh: 0.001, co2e_grams: 0.4,
        cost_usd: 0.002, created_at: new Date().toISOString(),
      };
      await route.fulfill(json({ requests: [sampleRequest], total: 1, page: 1, per_page: 20 }));
    } else {
      await route.continue();
    }
  });

  await page.route('**/v1/carbon**', async (route) => {
    await route.fulfill(json({
      period: '30d',
      total_co2e_grams: 18.5,
      total_energy_kwh: 0.045,
      gpt4_equivalent_co2e_grams: 92.5,
      savings_percent: 80.0,
      grid_region: 'us-east-1',
      grid_carbon_intensity: 411,
      daily_breakdown: [],
      model_energy_breakdown: [],
    }));
  });

  await page.route('**/api-keys**', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill(json([]));
    } else {
      await route.continue();
    }
  });

  await page.route('**/organizations/**', async (route) => {
    const url = route.request().url();
    const method = route.request().method();
    if (url.includes('/members') && method === 'GET') {
      await route.fulfill(json({ org_id: 'org-1', members: [] }));
    } else if (method === 'GET') {
      await route.fulfill(json({ id: 'org-1', name: 'Test Org', slug: 'test-org', plan: 'pro', created_at: '2026-01-01T00:00:00Z' }));
    } else {
      await route.continue();
    }
  });

  await page.route('**/v1/models**', async (route) => {
    await route.fulfill(json({ models: [] }));
  });

  await page.route('**/v1/chat/completions', async (route) => {
    await route.fulfill(json({
      id: 'test-completion',
      object: 'chat.completion',
      created: Date.now(),
      model: 'claude-haiku',
      choices: [{ index: 0, message: { role: 'assistant', content: 'Hello!' }, finish_reason: 'stop' }],
      usage: { prompt_tokens: 10, completion_tokens: 5, total_tokens: 15 },
      ecollm: {
        route: { task_type: 'general', complexity: 0.3, model_selected: 'claude-haiku', routing_score: 0.9, confidence: 0.85, used_fallback: false },
        energy: { total_energy_kwh: 0.001, co2e_grams: 0.4, grid_region: 'us-east-1' },
        cost: { total_cost_usd: 0.002, savings_vs_gpt4_percent: 75 },
        performance: { latency_ms: 350 },
      },
    }));
  });
}
