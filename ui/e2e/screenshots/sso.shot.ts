import { ADMIN, ANA, TOKENS, moveStubClock, setAuthMode } from '../stub-api.js';
import { configureIssuer, resetIssuer } from '../stub-issuer.js';
import { expect, shot, signInAs, test } from './shots.js';

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

test('invite and reset by mode', async ({ page }) => {
	await setAuthMode('oidc');
	await page.goto(`/invite/${TOKENS.invite}`);
	await expect(page.getByRole('button', { name: SSO })).toBeVisible();
	await shot(
		page,
		'sso',
		'05-invite-oidc',
		'AUTH_MODE=oidc: the invite keeps the username and finishes by single sign-on'
	);
	await setAuthMode('both');
	await page.goto(`/invite/${TOKENS.invite}`);
	await expect(page.getByRole('button', { name: SSO })).toBeVisible();
	await shot(
		page,
		'sso',
		'06-invite-both',
		'AUTH_MODE=both: the set-password card, then the outline single sign-on button'
	);
	await setAuthMode('oidc');
	await page.goto(`/reset/${TOKENS.reset}`);
	await expect(page.getByRole('alert')).toBeVisible();
	await shot(
		page,
		'sso',
		'08-reset-oidc',
		'AUTH_MODE=oidc: a reset link works only while password sign-in is on'
	);
});

test('invite confirmation', async ({ page }) => {
	await resetIssuer();
	await configureIssuer({ user: { sub: 'stub-subject-bo-0001', preferredUsername: 'bo.sso' } });
	await setAuthMode('oidc');
	await page.goto(`/invite/${TOKENS.invite}`);
	await page.getByRole('button', { name: SSO }).click();
	await expect(page.getByRole('button', { name: 'Confirm' })).toBeVisible();
	await shot(
		page,
		'sso',
		'07-confirm-invite',
		'Back from the issuer: confirm which account the single sign-on joins, or cancel'
	);
	await moveStubClock(72 * 3600 + 1);
	await page.getByRole('button', { name: 'Confirm' }).click();
	await expect(page.getByRole('alert')).toContainText('This link is no longer valid');
	await shot(
		page,
		'sso',
		'09-confirm-link-no-longer-valid',
		'The invite expired, was replaced or used before Confirm: nothing is linked'
	);
});

test('link single sign-on', async ({ page }) => {
	await resetIssuer();
	await setAuthMode('both');
	await signInAs(page, ANA);
	await page.goto('/account');
	const section = page.getByRole('region', { name: 'Single sign-on' });
	await expect(section.getByRole('button', { name: 'Link single sign-on' })).toBeVisible();
	await shot(
		page,
		'sso',
		'10-account-linkable',
		'AUTH_MODE=both: a Single sign-on card follows Password; linking asks for the password first'
	);
	await section.getByLabel('Password', { exact: true }).fill('not the password');
	await section.getByRole('button', { name: 'Link single sign-on' }).click();
	await expect(page.getByText('The password is incorrect')).toBeVisible();
	await shot(
		page,
		'sso',
		'12-link-sso-wrong-password',
		'A wrong password: the link page says so under the field and nothing reaches the issuer'
	);
	await page.goto('/account');
	await section.getByLabel('Password', { exact: true }).fill(ANA.password);
	await section.getByRole('button', { name: 'Link single sign-on' }).click();
	await expect(page.getByRole('button', { name: 'Confirm' })).toBeVisible();
	await shot(
		page,
		'sso',
		'13-confirm-link',
		'Back from the issuer: confirm which single sign-on account joins this one, or cancel'
	);
	await page.getByRole('button', { name: 'Confirm' }).click();
	await expect(section).toContainText('Single sign-on is linked.');
	await shot(
		page,
		'sso',
		'11-account-linked',
		'Linked: the section says so; only the admin can unlink'
	);
});

test('admin account', async ({ page }) => {
	await setAuthMode('both');
	await signInAs(page, ADMIN);
	await page.goto('/admin/account');
	await expect(page.getByRole('heading', { name: 'Account', level: 1 })).toBeVisible();
	await shot(
		page,
		'sso',
		'14-admin-account',
		'The admin’s Account page: the permanent username, then Link single sign-on'
	);
});
