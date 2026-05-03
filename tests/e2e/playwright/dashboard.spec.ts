import { test, expect, type Page } from '@playwright/test';
import { setupApiMocks } from './mocks';

const BASE = process.env.BASE_URL || 'http://localhost:3000';
const EMAIL = process.env.TEST_EMAIL || 'admin@example.com';
const PASSWORD = process.env.TEST_PASSWORD || 'testpassword';

async function login(page: Page) {
  await page.goto(`${BASE}/login`);
  await page.getByLabel(/email/i).fill(EMAIL);
  await page.getByLabel(/password/i).fill(PASSWORD);
  await page.getByRole('button', { name: /sign in/i }).click();
  await page.waitForURL(/\/overview/, { timeout: 10000 });
}

test.describe('Dashboard', () => {
  test.beforeEach(async ({ page }) => {
    await setupApiMocks(page);
    await login(page);
  });

  test('overview page shows metric cards', async ({ page }) => {
    await expect(page.getByText(/total requests/i)).toBeVisible();
    await expect(page.getByText(/co.?2/i).first()).toBeVisible();
  });

  test('sidebar navigation works', async ({ page }) => {
    await page.getByRole('link', { name: /carbon/i }).click();
    await expect(page).toHaveURL(/\/carbon/);

    await page.getByRole('link', { name: /api keys/i }).click();
    await expect(page).toHaveURL(/\/api-keys/);
  });

  test('requests page loads and shows table headers', async ({ page }) => {
    await page.getByRole('link', { name: /requests/i }).click();
    await expect(page).toHaveURL(/\/requests/);
    await expect(page.getByRole('columnheader', { name: /model/i })).toBeVisible({ timeout: 8000 });
  });

  test('playground page renders', async ({ page }) => {
    await page.getByRole('link', { name: /playground/i }).click();
    await expect(page).toHaveURL(/\/playground/);
    await expect(page.getByPlaceholder(/type a message/i)).toBeVisible();
  });

  test('settings page renders org form', async ({ page }) => {
    await page.getByRole('link', { name: /settings/i }).click();
    await expect(page).toHaveURL(/\/settings/);
    await expect(page.getByRole('heading', { name: /organization/i })).toBeVisible();
  });
});

test.describe('Request Detail', () => {
  test.beforeEach(async ({ page }) => {
    await setupApiMocks(page);
    await login(page);
  });

  test('clicking a request row navigates to detail page', async ({ page }) => {
    await page.goto(`${BASE}/requests`);
    const firstRow = page.getByRole('link').filter({ hasText: /ago/ }).first();
    const count = await firstRow.count();
    if (count === 0) {
      test.skip(); // no requests in test environment
      return;
    }
    await firstRow.click();
    await expect(page).toHaveURL(/\/requests\/.+/);
    await expect(page.getByText(/route trace/i)).toBeVisible({ timeout: 5000 });
  });
});
