import { expect, test, type Page } from '@playwright/test';
import { formatAge, todayIn } from '../src/lib/format.js';
import { expectAccessible, expectNotFoundPage, signIn } from './helpers.js';
import { ANA, CLEO, LEO, MIA, resetStub } from './stub-api.js';

test.beforeEach(async () => {
	await resetStub();
});

const region = (page: Page, name: string) => page.getByRole('region', { name, exact: true });

test('a guardian sees the details, the settings, the people and the delete entry', async ({
	page
}) => {
	await signIn(page, CLEO);
	await page.goto(`/kids/${LEO.id}/profile`);
	await expect(page.getByRole('heading', { name: 'Child profile', level: 1 })).toBeVisible();
	await expect(page.getByText(formatAge(LEO.birthDate, todayIn(LEO.timeZone)))).toBeVisible();

	const details = region(page, 'Details');
	for (const text of ['Leo', 'August 20, 2024', '35 weeks 2 days', 'Male', LEO.timeZone]) {
		await expect(details.getByText(text, { exact: true })).toBeVisible();
	}
	await expect(details.getByRole('link', { name: 'Edit details' })).toHaveAttribute(
		'href',
		`/kids/${LEO.id}/settings`
	);

	const settings = region(page, `Settings for ${LEO.name}`);
	await expect(settings.getByRole('link')).toHaveText([
		/^Trackers/,
		/^Growth chart source/,
		/^Vaccination schedule/,
		/^Show a missed dose as overdue for \(minutes\)/
	]);
	await expect(settings.getByRole('link', { name: /^Trackers/ })).toHaveAttribute(
		'href',
		`/kids/${LEO.id}/modules`
	);
	await expect(settings.getByRole('link', { name: /^Growth chart source/ })).toHaveAttribute(
		'href',
		`/kids/${LEO.id}/settings#kid-growth_source`
	);
	await expect(settings.getByText('60 minutes')).toBeVisible();

	const people = region(page, 'People with access');
	const row = (name: string) => people.getByRole('listitem').filter({ hasText: name });
	await expect(people.getByRole('listitem')).toHaveCount(2);
	await expect(row('Cleo')).toContainText('Guardian');
	await expect(row('Cleo')).toContainText('You');
	await expect(row('Ana')).toContainText('Caregiver');
	await expect(people.getByRole('button')).toHaveCount(0);
	await expect(people.getByRole('link', { name: 'Manage access' })).toHaveAttribute(
		'href',
		`/kids/${LEO.id}/access`
	);

	const remove = region(page, `Delete all data for ${LEO.name}`);
	await expect(
		remove.getByRole('link', { name: `Delete all data for ${LEO.name}…` })
	).toHaveAttribute('href', `/kids/${LEO.id}/settings#delete`);
	await expectAccessible(page);
});

test('a child not born early has no weeks at birth row', async ({ page }) => {
	await signIn(page, ANA);
	await page.goto(`/kids/${MIA.id}/profile`);
	await expect(region(page, 'Details').getByText('Female', { exact: true })).toBeVisible();
	await expect(region(page, 'Details').getByText('Weeks at birth', { exact: true })).toHaveCount(0);
	await expectAccessible(page);
});

test('Edit details, Manage access and Delete lead to the existing pages', async ({ page }) => {
	await signIn(page, CLEO);
	await page.goto(`/kids/${LEO.id}/profile`);
	await page.getByRole('link', { name: 'Edit details' }).click();
	await expect(page).toHaveURL(`/kids/${LEO.id}/settings`);
	await expect(page.getByRole('heading', { name: 'Edit details', level: 1 })).toBeVisible();
	await page.getByRole('link', { name: 'Cancel' }).click();
	await expect(page).toHaveURL(`/kids/${LEO.id}/profile`);

	await page.getByRole('link', { name: 'Manage access' }).click();
	await expect(page).toHaveURL(`/kids/${LEO.id}/access`);
	await expect(page.getByRole('heading', { name: 'Access', level: 1 })).toBeVisible();

	await page.goto(`/kids/${LEO.id}/profile`);
	await page.getByRole('link', { name: `Delete all data for ${LEO.name}…` }).click();
	await expect(page).toHaveURL(`/kids/${LEO.id}/settings#delete`);
	await expect(page.getByText(`Delete all data for ${LEO.name}…`, { exact: true })).toBeVisible();
});

test('a caregiver gets the same 404 page as for an unknown child', async ({ page }) => {
	await signIn(page, ANA);
	const unknown = await expectNotFoundPage(page, '/kids/k-unknown/profile');
	expect(await expectNotFoundPage(page, `/kids/${LEO.id}/profile`)).toBe(unknown);
});

test.describe('without JavaScript', () => {
	test.use({ javaScriptEnabled: false });

	test('the page and its links work', async ({ page }) => {
		await signIn(page, CLEO);
		await page.goto(`/kids/${LEO.id}/profile`);
		await expect(page.getByRole('heading', { name: 'Child profile', level: 1 })).toBeVisible();
		await page.getByRole('link', { name: 'Manage access' }).click();
		await expect(page.getByRole('heading', { name: 'Access', level: 1 })).toBeVisible();
	});
});
