import { expect, test, type Page } from '@playwright/test';
import { expectAccessible, expectFieldError, signIn } from './helpers.js';
import { ANA, STUB_TOKEN, STUB_URL, openSession, resetStub } from './stub-api.js';

const NEW_PASSWORD = 'a brand new password';

test.beforeEach(async ({ page }) => {
	await resetStub();
	await signIn(page, ANA);
	await page.goto('/account');
	await expect(page.getByRole('heading', { name: 'Account', level: 1 })).toBeVisible();
});

const sessions = (page: Page) => page.getByRole('list', { name: 'Sessions' }).getByRole('listitem');

async function stubSessionStatus(token: string) {
	const res = await fetch(`${STUB_URL}/v1/session`, {
		headers: { authorization: `Bearer ${STUB_TOKEN}`, 'x-session-token': token }
	});
	return res.status;
}

async function changePassword(page: Page, current: string, next: string, confirm = next) {
	await page.getByLabel('Current password').fill(current);
	await page.getByLabel('New password', { exact: true }).fill(next);
	await page.getByLabel('Confirm new password').fill(confirm);
	await page.getByRole('button', { name: 'Change password' }).click();
}

test('units and display name persist', async ({ page }) => {
	await expect(page.getByLabel('Display name')).toHaveValue(ANA.displayName);
	await expect(page.getByLabel('Volume')).toHaveValue('ml');
	await expectAccessible(page);
	await page.getByLabel('Display name').fill('Ana B');
	await page.getByLabel('Weight').selectOption({ label: 'lb' });
	await page.getByLabel('Length').selectOption({ label: 'in' });
	await page.getByLabel('Volume').selectOption({ label: 'fl oz' });
	await page.getByLabel('Temperature').selectOption({ label: '°F' });
	await page.getByRole('button', { name: 'Save' }).click();
	await expect(page.getByRole('status')).toHaveText('Saved');
	await expectAccessible(page);

	await page.reload();
	await expect(page.getByLabel('Display name')).toHaveValue('Ana B');
	await expect(page.getByLabel('Weight')).toHaveValue('lb');
	await expect(page.getByLabel('Length')).toHaveValue('in');
	await expect(page.getByLabel('Volume')).toHaveValue('floz');
	await expect(page.getByLabel('Temperature')).toHaveValue('f');
	await page.goto('/');
	await expect(page.getByRole('heading', { name: 'Welcome, Ana B' })).toBeVisible();
});

test('a blank display name is refused on its field', async ({ page }) => {
	await page.getByLabel('Display name').fill('   ');
	await page.getByRole('button', { name: 'Save' }).click();
	await expectFieldError(page.getByLabel('Display name'), 'Use 1 to 64 characters');
	await expectAccessible(page);
});

test('a wrong current password gets its message', async ({ page }) => {
	await changePassword(page, 'not the password', NEW_PASSWORD);
	await expectFieldError(page.getByLabel('Current password'), 'The current password is incorrect');
	await expect(page.getByLabel('Current password')).toHaveValue('');
	await expectAccessible(page);
});

test('the new password is checked in the form', async ({ page }) => {
	await changePassword(page, ANA.password, NEW_PASSWORD, NEW_PASSWORD + '!');
	await expectFieldError(page.getByLabel('Confirm new password'), 'The passwords do not match');
	await changePassword(page, ANA.password, 'short');
	await expectFieldError(
		page.getByLabel('New password', { exact: true }),
		'Use at least 12 characters'
	);
});

test('changing the password keeps this session and ends the others', async ({ page }) => {
	const other = await openSession(ANA.username, ANA.password);
	await page.reload();
	await expect(sessions(page)).toHaveCount(2);
	await changePassword(page, ANA.password, NEW_PASSWORD);
	await expect(page.getByRole('status')).toHaveText(
		'Password changed. Your other sessions were signed out.'
	);
	await expect(sessions(page)).toHaveCount(1);
	await expect(sessions(page)).toContainText('This device');
	expect(await stubSessionStatus(other)).toBe(401);
	await expectAccessible(page);

	await page.goto('/');
	await expect(page.getByRole('heading', { name: 'Welcome, Ana' })).toBeVisible();
});

test('sign out another session', async ({ page }) => {
	const other = await openSession(ANA.username, ANA.password);
	await page.reload();
	await expect(sessions(page)).toHaveCount(2);
	const current = sessions(page).filter({ hasText: 'This device' });
	await expect(current.getByRole('button')).toHaveCount(0);
	await expectAccessible(page);

	await sessions(page)
		.filter({ hasText: 'Another device' })
		.getByRole('button', { name: /^Sign out the session started/ })
		.click();
	await expect(sessions(page)).toHaveCount(1);
	await expect(sessions(page)).toContainText('This device');
	expect(await stubSessionStatus(other)).toBe(401);
	await expectAccessible(page);
});
