import { expect, test, type Page } from '@playwright/test';
import { expectAccessible, openSheet, signIn } from './helpers.js';
import { ANA, MIA, resetStub } from './stub-api.js';

test.beforeEach(async () => {
	await resetStub();
});

const profile = `/kids/${MIA.id}/profile`;
const link = (name: string) => (p: Page) => p.getByRole('link', { name });
const toast = (page: Page, text: string) => page.locator('[data-sonner-toast]').getByText(text);

test('Edit details opens in a sheet, saves, closes and says so', async ({ page }) => {
	await signIn(page, ANA);
	const sheet = await openSheet(page, profile, link('Edit details'), 'Edit details');
	await expect(page).toHaveURL(`/kids/${MIA.id}/settings`);
	await expectAccessible(page);
	await sheet.getByLabel('Name', { exact: true }).fill('Mia Rose');
	await sheet.getByRole('button', { name: 'Save', exact: true }).click();
	await expect(sheet).toBeHidden();
	await expect(page).toHaveURL(profile);
	await expect(toast(page, 'Details saved')).toBeVisible();
	await expect(page.getByText('Mia Rose').first()).toBeVisible();
});

test('a settings row opens Edit details with its field focused', async ({ page }) => {
	await signIn(page, ANA);
	const sheet = await openSheet(page, profile, link(/^Growth chart source/), 'Edit details');
	await expect(page).toHaveURL(`/kids/${MIA.id}/settings#kid-growth_source`);
	await expect(sheet.getByLabel('Growth chart source')).toBeFocused();
});

test('Modules opens in a sheet and saves', async ({ page }) => {
	await signIn(page, ANA);
	const sheet = await openSheet(page, profile, link(/^Trackers/), `Modules for ${MIA.name}`);
	await expect(page).toHaveURL(`/kids/${MIA.id}/modules`);
	await expectAccessible(page);
	await sheet.getByRole('button', { name: 'Save', exact: true }).click();
	await expect(sheet).toBeHidden();
	await expect(toast(page, 'Modules saved')).toBeVisible();
});

test('Access changes update the open sheet in place', async ({ page }) => {
	await signIn(page, ANA);
	const sheet = await openSheet(page, profile, link('Manage access'), 'Access');
	await expect(page).toHaveURL(`/kids/${MIA.id}/access`);
	await sheet.getByLabel('Parent’s username').fill('dan');
	await sheet.getByRole('button', { name: 'Give access', exact: true }).click();
	const dan = sheet.getByRole('listitem').filter({ hasText: 'Dan' });
	await expect(dan).toContainText('Caregiver');
	await expect(sheet).toBeVisible();
	await expect(toast(page, 'Access added')).toBeVisible();
	await dan.getByRole('button', { name: 'Make Dan a guardian' }).click();
	await expect(dan).toContainText('Guardian');
	await sheet.getByRole('button', { name: 'Close' }).click();
	await expect(sheet).toBeHidden();
	await expect(page).toHaveURL(profile);
});

test('Add a child opens from the switcher and lands on the new child', async ({ page }) => {
	await signIn(page, ANA);
	const switcher = page.getByRole('dialog', { name: 'Switch child' });
	await expect(async () => {
		await page.goto('/');
		await page.getByRole('link', { name: /Switch child$/ }).click();
		await expect(switcher).toBeVisible({ timeout: 1500 });
	}).toPass();
	await switcher.getByRole('link', { name: 'Add a child' }).click();
	const sheet = page.getByRole('dialog', { name: 'Add a child' });
	await expect(sheet).toBeVisible();
	await expect(page).toHaveURL('/kids/new');
	await sheet.getByLabel('Name', { exact: true }).fill('Noa');
	await sheet.getByLabel('Birth date').fill('2026-10-01');
	await sheet.getByRole('radio', { name: 'Female' }).check();
	await sheet.getByRole('button', { name: 'Add child' }).click();
	await expect(page.getByRole('heading', { name: 'Noa', level: 1 })).toBeVisible();
	await expect(sheet).toBeHidden();
	await expect(toast(page, 'Child added')).toBeVisible();
});

test.describe('without JavaScript', () => {
	test.use({ javaScriptEnabled: false });
	test('the links are the pages', async ({ page }) => {
		await signIn(page, ANA);
		await page.goto(profile);
		await page.getByRole('link', { name: 'Edit details' }).click();
		await expect(page).toHaveURL(`/kids/${MIA.id}/settings`);
		await expect(page.getByRole('heading', { name: 'Edit details', level: 1 })).toBeVisible();
	});
});
