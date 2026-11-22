import { expect, test, type Page } from '@playwright/test';
import { expectAccessible, expectLanding, sessionCookie } from './helpers.js';
import { INVITEE, TOKENS, receivedRequests, resetStub } from './stub-api.js';

const ADDRESS = '198.51.100.40';
const GOOD = 'a long enough password';

test.beforeEach(async ({ context }) => {
	await resetStub();
	await context.setExtraHTTPHeaders({ 'x-forwarded-proto': 'http', 'x-forwarded-for': ADDRESS });
});

async function fill(page: Page, password: string, confirm: string) {
	await page.getByLabel('New password').fill(password);
	await page.getByLabel('Confirm password').fill(confirm);
	await page.getByRole('button', { name: 'Set password' }).click();
}

const redeems = async () => (await receivedRequests()).filter((r) => r.path.endsWith('/redeem'));

test('a valid link shows the username and a set-password form', async ({ page }) => {
	await page.goto(`/invite/${TOKENS.invite}`);
	await expect(page.getByRole('heading', { name: 'Set your password' })).toBeVisible();
	await expect(page.getByLabel('Username')).toHaveValue(INVITEE);
	await expect(page.getByLabel('New password')).toBeVisible();
	await expect(page.getByLabel('Confirm password')).toBeVisible();
	await expectAccessible(page);
});

test('a mismatched confirmation is refused in the form', async ({ page }) => {
	await page.goto(`/invite/${TOKENS.invite}`);
	await fill(page, GOOD, GOOD + '!');
	await expect(page.getByText('The passwords do not match')).toBeVisible();
	await expectAccessible(page);
	expect(await redeems()).toEqual([]);
});

test('an 11-character password is refused in the form', async ({ page }) => {
	await page.goto(`/invite/${TOKENS.invite}`);
	await fill(page, 'x'.repeat(11), 'x'.repeat(11));
	await expect(page.getByText('Use at least 12 characters')).toBeVisible();
	await expectAccessible(page);
	expect(await redeems()).toEqual([]);
});

test('success logs in and the link is then spent', async ({ page, context }) => {
	await page.goto(`/invite/${TOKENS.invite}`);
	await fill(page, GOOD, GOOD);
	await expect(page).toHaveURL(/127\.0\.0\.1:\d+\/$/);
	await expectLanding(page, { welcome: 'Bo' });
	expect(await sessionCookie(context)).toMatchObject({ httpOnly: true, secure: true });
	expect(await redeems()).toMatchObject([
		{ method: 'POST', authorized: true, clientAddress: ADDRESS }
	]);

	await context.clearCookies();
	await page.goto(`/invite/${TOKENS.invite}`);
	await expect(page.getByText('This link is no longer valid')).toBeVisible();
});

const invalid: [string, string][] = [
	['used', `/invite/${TOKENS.used}`],
	['expired', `/invite/${TOKENS.expired}`],
	['for a disabled parent', `/invite/${TOKENS.disabledParent}`],
	['unknown', `/invite/${TOKENS.unknown}`],
	['malformed', '/invite/not.a.token'],
	['a reset link', `/invite/${TOKENS.reset}`]
];
for (const [cause, path] of invalid) {
	test(`a link that is ${cause} shows "This link is no longer valid"`, async ({ page }) => {
		await page.goto(path);
		await expect(page.getByRole('alert')).toContainText('This link is no longer valid');
		await expect(page.getByLabel('New password')).toHaveCount(0);
		await expectAccessible(page);
	});
}

test.describe('without JavaScript', () => {
	test.use({ javaScriptEnabled: false });

	test('the server refuses a mismatch and a short password, then accepts', async ({ page }) => {
		await page.goto(`/invite/${TOKENS.invite}`);
		await fill(page, GOOD, GOOD + '!');
		await expect(page.getByText('The passwords do not match')).toBeVisible();
		await fill(page, 'x'.repeat(11), 'x'.repeat(11));
		await expect(page.getByText('Use at least 12 characters')).toBeVisible();
		expect(await redeems()).toEqual([]);
		await fill(page, GOOD, GOOD);
		await expectLanding(page, { welcome: 'Bo' });
	});
});
