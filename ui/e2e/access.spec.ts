import { expect, test, type Page } from '@playwright/test';
import { expectAccessible, expectFieldError, expectNotFoundPage, signIn } from './helpers.js';
import { ANA, CLEO, LEO, MIA, resetStub } from './stub-api.js';

const PARENT_NOT_FOUND = 'No active account has that username.';
const LAST_GUARDIAN = 'A child needs at least one guardian.';

test.beforeEach(async () => {
	await resetStub();
});

const grant = (page: Page, displayName: string) =>
	page
		.getByRole('list', { name: 'People with access' })
		.getByRole('listitem')
		.filter({ hasText: displayName });

async function add(page: Page, username: string, role: 'Caregiver' | 'Guardian') {
	await page.getByLabel('Username').fill(username);
	await page.getByLabel('Role', { exact: true }).selectOption({ label: role });
	await page.getByRole('button', { name: 'Add', exact: true }).click();
}

test('a guardian sees the grants with their roles', async ({ page }) => {
	await signIn(page, CLEO);
	await page.goto(`/kids/${LEO.id}/access`);
	await expect(page.getByRole('heading', { name: `Access to ${LEO.name}` })).toBeVisible();
	await expect(grant(page, 'Cleo')).toContainText('Guardian');
	await expect(grant(page, 'Cleo')).toContainText('(you)');
	await expect(grant(page, 'Ana')).toContainText('Caregiver');
	await expect(grant(page, 'Cleo').getByRole('button', { name: 'Leave' })).toBeVisible();
	await expect(grant(page, 'Ana').getByRole('button', { name: 'Leave' })).toHaveCount(0);
	await expectAccessible(page);
});

test('grant, change role and remove', async ({ page }) => {
	await signIn(page, ANA);
	await page.goto(`/kids/${MIA.id}/access`);
	await add(page, '  DAN ', 'Caregiver');
	await expect(grant(page, 'Dan')).toContainText('Caregiver');
	await expectAccessible(page);

	await grant(page, 'Dan').getByLabel('Role for Dan').selectOption({ label: 'Guardian' });
	await grant(page, 'Dan').getByRole('button', { name: 'Change role for Dan' }).click();
	await expect(grant(page, 'Dan')).toContainText('Guardian');
	await expect(grant(page, 'Dan').getByLabel('Role for Dan')).toHaveValue('guardian');

	await grant(page, 'Dan').getByRole('button', { name: 'Remove Dan' }).click();
	await expect(grant(page, 'Dan')).toHaveCount(0);
	await expect(grant(page, 'Ana')).toBeVisible();
	await expectAccessible(page);
});

test(`an unknown or disabled username reads "${PARENT_NOT_FOUND}"`, async ({ page }) => {
	await signIn(page, ANA);
	await page.goto(`/kids/${MIA.id}/access`);
	for (const username of ['nobody', 'dee', '   ']) {
		await add(page, username, 'Caregiver');
		await expectFieldError(page.getByLabel('Username'), PARENT_NOT_FOUND);
	}
	await expectAccessible(page);
});

test(`the only guardian cannot leave: "${LAST_GUARDIAN}"`, async ({ page }) => {
	await signIn(page, ANA);
	await page.goto(`/kids/${MIA.id}/access`);
	await grant(page, 'Ana').getByRole('button', { name: 'Leave' }).click();
	await expect(page.getByRole('alert')).toHaveText(LAST_GUARDIAN);
	await expect(grant(page, 'Ana')).toContainText('Guardian');
	await expectAccessible(page);
});

test(`once the other guardian left, the remaining one cannot leave: "${LAST_GUARDIAN}"`, async ({
	page
}) => {
	await signIn(page, ANA);
	await page.goto(`/kids/${MIA.id}/access`);
	await add(page, 'cleo', 'Guardian');
	await grant(page, 'Ana').getByRole('button', { name: 'Leave' }).click();
	await expect(page).toHaveURL(/127\.0\.0\.1:\d+\/$/);

	await page.getByRole('button', { name: 'Sign out', exact: true }).click();
	await signIn(page, CLEO);
	await page.goto(`/kids/${MIA.id}/access`);
	await expect(grant(page, 'Ana')).toHaveCount(0);
	await grant(page, 'Cleo').getByRole('button', { name: 'Leave' }).click();
	await expect(page.getByRole('alert')).toHaveText(LAST_GUARDIAN);
});

test('leaving as one of two guardians goes home without the kid', async ({ page }) => {
	await signIn(page, ANA);
	await page.goto(`/kids/${MIA.id}/access`);
	await add(page, 'cleo', 'Guardian');
	await expect(grant(page, 'Cleo')).toContainText('Guardian');
	await grant(page, 'Ana').getByRole('button', { name: 'Leave' }).click();
	await expect(page).toHaveURL(/127\.0\.0\.1:\d+\/$/);
	await expect(page.getByRole('heading', { name: MIA.name, exact: true })).toHaveCount(0);
	await expectAccessible(page);
	await expectNotFoundPage(page, `/kids/${MIA.id}`);
});

test('the caregiver view has no admin controls', async ({ page }) => {
	await signIn(page, ANA);
	await page.goto(`/kids/${LEO.id}`);
	await expect(page.getByRole('heading', { name: LEO.name, level: 1 })).toBeVisible();
	await expect(page.getByRole('link', { name: /Access|Settings|Parents/ })).toHaveCount(0);
	await expect(
		page.getByRole('button', { name: /Remove|Change role|Add|Delete|Leave/ })
	).toHaveCount(0);
	await expectAccessible(page);
	await page.goto('/');
	await expect(page.getByRole('link', { name: `Access for ${LEO.name}` })).toHaveCount(0);
	await expect(page.getByRole('link', { name: `Settings for ${LEO.name}` })).toHaveCount(0);
	await expect(page.getByRole('link', { name: 'Parents' })).toHaveCount(0);
	await expectNotFoundPage(page, `/kids/${LEO.id}/access`);
});
