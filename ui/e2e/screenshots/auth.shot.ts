import { login } from '../helpers.js';
import { ANA, TOKENS, setScenario } from '../stub-api.js';
import { expect, shot, test } from './shots.js';

const GOOD = 'a long enough password';

async function setPassword(
	page: import('@playwright/test').Page,
	password: string,
	confirm: string
) {
	await page.getByLabel('New password').fill(password);
	await page.getByLabel('Confirm password').fill(confirm);
	await page.getByRole('button', { name: 'Set password' }).click();
}

test('login', async ({ page }) => {
	await page.goto('/login');
	await shot(page, 'auth', '01-login-empty', 'Sign-in card, nothing typed');
	await login(page, ANA.username, 'wrong-password-0000');
	await expect(page.getByRole('alert')).toBeVisible();
	await shot(
		page,
		'auth',
		'02-login-wrong-password',
		'Wrong password: one generic message, username kept'
	);
	await setScenario('rate_limited');
	await login(page, ANA.username, ANA.password);
	await expect(page.getByRole('alert')).toHaveText(/Too many attempts/);
	await shot(page, 'auth', '03-login-rate-limited', 'The API rate-limits sign-in: wait a minute');
	await setScenario('down');
	await login(page, ANA.username, ANA.password);
	await expect(page.getByRole('alert')).toHaveText(/cannot reach its server/);
	await shot(page, 'auth', '04-login-api-down', 'The API is down while signing in');
});

test('invite', async ({ page }) => {
	await page.goto(`/invite/${TOKENS.invite}`);
	await shot(page, 'auth', '05-invite-valid', 'Invite link: username shown, set-password form');
	await setPassword(page, GOOD, `${GOOD}!`);
	await expect(page.getByText('The passwords do not match')).toBeVisible();
	await shot(
		page,
		'auth',
		'06-invite-mismatch',
		'Confirmation does not match (checked in the browser)'
	);
	await setPassword(page, 'x'.repeat(11), 'x'.repeat(11));
	await expect(page.getByText('Use at least 12 characters')).toBeVisible();
	await shot(page, 'auth', '07-invite-too-short', 'An 11-character password is refused');
	await setScenario('rate_limited');
	await setPassword(page, GOOD, GOOD);
	await expect(page.getByRole('alert')).toBeVisible();
	await shot(page, 'auth', '08-invite-rate-limited', 'The API rate-limits the redeem');
	await setScenario('normal');
	await setPassword(page, GOOD, GOOD);
	await expect(page.getByRole('heading', { name: 'Welcome, Bo' })).toBeVisible();
	await shot(
		page,
		'auth',
		'09-invite-success-welcome',
		'After the invite: signed in, no child yet'
	);
	await page.context().clearCookies();
	await page.goto(`/invite/${TOKENS.used}`);
	await expect(page.getByRole('alert')).toContainText('This link is no longer valid');
	await shot(page, 'auth', '10-invite-invalid', 'A used, expired or unknown invite link');
});

test('reset', async ({ page }) => {
	await page.goto(`/reset/${TOKENS.reset}`);
	await shot(page, 'auth', '11-reset-valid', 'Password reset link: set a new password');
	await setPassword(page, 'x'.repeat(11), 'x'.repeat(11));
	await expect(page.getByText('Use at least 12 characters')).toBeVisible();
	await shot(page, 'auth', '12-reset-too-short', 'Reset form refuses a short password');
	await page.goto(`/reset/${TOKENS.expired}`);
	await expect(page.getByRole('alert')).toContainText('This link is no longer valid');
	await shot(page, 'auth', '13-reset-invalid', 'An expired or used reset link');
});
