import { test, expect } from '@playwright/test';
import { setupApiMocks } from './mocks';

const BASE = process.env.BASE_URL || 'http://localhost:3000';

test.describe('Authentication', () => {
  test('login page renders', async ({ page }) => {
    await page.goto(`${BASE}/login`);
    await expect(page.getByRole('heading', { name: /sign in/i })).toBeVisible();
    await expect(page.getByLabel(/email/i)).toBeVisible();
    await expect(page.getByLabel(/password/i)).toBeVisible();
  });

  test('invalid credentials shows error', async ({ page }) => {
    await setupApiMocks(page);
    await page.goto(`${BASE}/login`);
    await page.getByLabel(/email/i).fill('nobody@example.com');
    await page.getByLabel(/password/i).fill('wrongpassword');
    await page.getByRole('button', { name: /sign in/i }).click();
    await expect(page.getByText(/invalid/i)).toBeVisible({ timeout: 5000 });
  });

  test('valid login redirects to overview', async ({ page }) => {
    await setupApiMocks(page);
    await page.goto(`${BASE}/login`);
    await page.getByLabel(/email/i).fill(process.env.TEST_EMAIL || 'admin@example.com');
    await page.getByLabel(/password/i).fill(process.env.TEST_PASSWORD || 'testpassword');
    await page.getByRole('button', { name: /sign in/i }).click();
    await expect(page).toHaveURL(/\/overview/, { timeout: 10000 });
  });
});
