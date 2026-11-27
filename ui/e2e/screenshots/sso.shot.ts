import { setAuthMode } from '../stub-api.js';
import { expect, shot, test } from './shots.js';

const SSO = 'Continue with single sign-on';

test('sign-in by mode', async ({ page }) => {
	await setAuthMode('oidc');
	await page.goto('/login');
	await expect(page.getByRole('button', { name: SSO })).toBeVisible();
	await shot(
		page,
		'sso',
		'01-login-oidc',
		'AUTH_MODE=oidc: the card holds the single sign-on button only'
	);
	await setAuthMode('both');
	await page.goto('/login');
	await expect(page.getByRole('button', { name: SSO })).toBeVisible();
	await shot(
		page,
		'sso',
		'02-login-both',
		'AUTH_MODE=both: the password card, then the outline single sign-on button'
	);
});

test('outcomes', async ({ page }) => {
	await page.goto('/auth/callback');
	await expect(
		page.getByRole('heading', { name: 'Single sign-on did not complete' })
	).toBeVisible();
	await shot(
		page,
		'sso',
		'03-did-not-complete',
		'The callback could not finish (a bad state, a refusal, an unlinked account): a link back'
	);
	await page.goto('/auth/sso');
	await expect(page.getByRole('heading', { name: 'Single sign-on is unavailable' })).toBeVisible();
	await shot(
		page,
		'sso',
		'04-sso-unavailable',
		'The issuer cannot be reached, or single sign-on is off: a link back to sign in'
	);
});
