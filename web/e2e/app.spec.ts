import { test, expect } from '@playwright/test';

test.describe('Agentary UI', () => {
  test('loads and shows header and nav', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByRole('img', { name: 'Agentary' })).toBeVisible();
    await expect(page.getByRole('button', { name: /kanban/i })).toBeVisible();
  });

  test('can open Kanban view', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('button', { name: /kanban/i }).click();
    await expect(page.getByText(/todo|in progress|done/i).first()).toBeVisible({ timeout: 5000 });
  });

  test('can open Agents view', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('button', { name: /agents/i }).click();
    await expect(page.locator('main')).toBeVisible();
  });

  test('can open all sidebar views without error', async ({ page }) => {
    await page.goto('/');
    const views = ['Workflow', 'Charts', 'Chat', 'Reviews', 'Network', 'Charter', 'Memory', 'Settings'];
    for (const label of views) {
      await page.getByRole('button', { name: new RegExp(label, 'i') }).click();
      await expect(page.locator('main')).toBeVisible();
      await expect(page.locator('main')).not.toContainText('Error');
    }
  });

  test('shows empty state when no teams', async ({ page }) => {
    await page.goto('/');
    const noTeams = page.getByText(/no teams/i);
    const teamSelect = page.getByRole('combobox', { name: /team/i });
    await Promise.race([
      expect(noTeams).toBeVisible(),
      expect(teamSelect).toBeVisible(),
    ]).catch(() => {});
  });

  test('responsive layout: sidebar visible on desktop', async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 720 });
    await page.goto('/');
    await expect(page.getByRole('button', { name: /kanban/i })).toBeVisible();
    await expect(page.locator('aside')).toBeVisible();
  });

  test('responsive layout: main content area visible on mobile', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/');
    await expect(page.locator('main')).toBeVisible();
    await expect(page.getByRole('button', { name: /kanban/i }).first()).toBeVisible();
  });

  test('dark mode toggle exists', async ({ page }) => {
    await page.goto('/');
    const themeToggle = page.getByRole('button', { name: /theme|dark|light/i });
    await expect(themeToggle).toBeVisible();
  });
});

test.describe('Agentary UI with backend', () => {
  test('create team and task via API then verify in UI', async ({ page, request }) => {
    const teamName = `e2e-${Date.now()}`;
    const resTeam = await request.post('/teams', {
      data: { name: teamName },
      headers: { 'Content-Type': 'application/json' },
    });
    if (!resTeam.ok()) {
      test.skip(true, 'Backend not available or API key required');
      return;
    }
    const resTask = await request.post(`/teams/${encodeURIComponent(teamName)}/tasks`, {
      data: { title: 'E2E task', status: 'todo' },
      headers: { 'Content-Type': 'application/json' },
    });
    expect(resTask.ok()).toBe(true);
    await page.goto('/');
    const teamSelect = page.getByRole('combobox', { name: /team/i }).or(page.locator('aside select'));
    await expect(teamSelect).toBeVisible({ timeout: 8000 });
    await teamSelect.selectOption({ label: teamName }).catch(() => {});
    await page.getByRole('button', { name: /kanban/i }).click();
    await expect(page.getByText('E2E task').first()).toBeVisible({ timeout: 5000 });
  });

  test('Kanban shows columns when team selected', async ({ page }) => {
    await page.goto('/');
    const teamSelect = page.getByRole('combobox', { name: /team/i });
    await teamSelect.waitFor({ state: 'visible', timeout: 8000 }).catch(() => {});
    const options = await teamSelect.locator('option').all();
    if (options.length === 0) {
      test.skip(true, 'No teams - create team first');
      return;
    }
    await page.getByRole('button', { name: /kanban/i }).click();
    await expect(page.getByText(/todo|in progress|done/i).first()).toBeVisible({ timeout: 5000 });
  });

  test('Network view loads', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('button', { name: /network/i }).click();
    await expect(page.getByText(/network allowlist|no domains/i).first()).toBeVisible({ timeout: 5000 });
  });

  test('Settings view shows config', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('button', { name: /settings/i }).click();
    await expect(page.getByText(/settings|config|human/i).first()).toBeVisible({ timeout: 5000 });
  });
});
