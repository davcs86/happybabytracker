import { expect, test, type Page } from '@playwright/test';
import {
	expectAccessible,
	expectFieldError,
	expectLanding,
	signInWithForm,
	untilHydrated
} from './helpers.js';
import {
	ANA,
	MIA,
	STUB_TOKEN,
	STUB_URL,
	openSession,
	receivedRequests,
	resetStub
} from './stub-api.js';

const NEW_PASSWORD = 'a brand new password';

test.beforeEach(async ({ page }) => {
	await resetStub();
	await signInWithForm(page, ANA);
	await page.goto('/account');
	await expect(page.getByRole('heading', { name: 'Your profile', level: 1 })).toBeVisible();
	await untilHydrated(page.getByLabel('Display name'));
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
	await expectLanding(page, { kid: MIA });
});

test('a blank display name is refused on its field', async ({ page }) => {
	await page.getByLabel('Display name').fill('   ');
	await page.getByRole('button', { name: 'Save' }).click();
	await expectFieldError(page.getByLabel('Display name'), 'Use 1 to 64 characters');
	await expectAccessible(page);
});

test('a password change carries the client address the limiter counts', async ({
	page,
	context
}) => {
	await context.setExtraHTTPHeaders({
		'x-forwarded-proto': 'http',
		'x-forwarded-for': '198.51.100.61'
	});
	await changePassword(page, ANA.password, NEW_PASSWORD);
	await expect(page.getByRole('status')).toBeVisible();
	const posted = (await receivedRequests()).find((r) => r.path === '/v1/me/password');
	expect(posted).toMatchObject({
		method: 'POST',
		authorized: true,
		clientAddress: '198.51.100.61'
	});
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
	await expectLanding(page, { kid: MIA });
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
		.getByRole('button', { name: /^Sign out/ })
		.click();
	await expect(sessions(page)).toHaveCount(1);
	await expect(sessions(page)).toContainText('This device');
	expect(await stubSessionStatus(other)).toBe(401);
	await expectAccessible(page);
});

test('the password card says when it changed and what changing it does', async ({ page }) => {
	const card = page.getByRole('region', { name: 'Password' });
	await expect(card.getByText('Last changed today.')).toBeVisible();
	await expect(
		card.getByText('Changing your password signs you out of every other device.')
	).toBeVisible();
	await expectAccessible(page);
});

test('a device shows the name it signed in with and when it was last active', async ({ page }) => {
	await openSession(ANA.username, ANA.password);
	await page.reload();
	const current = sessions(page).filter({ hasText: 'Active now' });
	await expect(current).toHaveCount(1);
	// The test browser signed in through the form, so it carries a label built from its user agent.
	await expect(current).toContainText(/^(Chrome|Safari|Firefox|Edge|Browser)\b.*This device/);
	const other = sessions(page).filter({ hasText: 'Last active' });
	await expect(other).toHaveCount(1);
	await expect(other).toContainText('Another device');
	await expect(other).toContainText(/Last active (Just now|\d+ min ago)/);
});

test('Sign out all other devices asks first, then ends every other session', async ({ page }) => {
	const one = await openSession(ANA.username, ANA.password);
	const two = await openSession(ANA.username, ANA.password);
	await page.reload();
	await expect(sessions(page)).toHaveCount(3);
	await page.getByText('Sign out all other devices', { exact: true }).click();
	await expect(page.getByText('This signs out 2 other devices.')).toBeVisible();
	await page.getByRole('button', { name: 'Yes, sign out all other devices' }).click();
	await expect(sessions(page)).toHaveCount(1);
	await expect(page.getByText('No other devices are signed in.')).toBeVisible();
	expect(await stubSessionStatus(one)).toBe(401);
	expect(await stubSessionStatus(two)).toBe(401);
	await expectAccessible(page);
});
