import { expect, test, type Page } from '@playwright/test';
import { expectAccessible, expectFieldError, signIn } from './helpers.js';
import { ADMIN, LEO, MIA, UI_ORIGIN, openSession, resetStub } from './stub-api.js';

const INVITE_TEXT = 'Send this link to the parent. It works once and expires in 72 hours';
const RESET_TEXT = 'Send this link to the parent. It works once and expires in 1 hour';
const SOLE_GUARDIAN =
	'This parent is the only guardian of at least one child. Another guardian must be added first.';
const LINK = /^http:\/\/127\.0\.0\.1:\d+\/(invite|reset)\/[A-Za-z0-9_-]{43}$/;

test.beforeEach(async ({ page }) => {
	await resetStub();
	await signIn(page, ADMIN);
	await expect(page).toHaveURL(/\/admin\/parents$/);
});

const row = (page: Page, username: string) =>
	page.getByRole('listitem').filter({ has: page.getByText(username, { exact: true }) });

// The create form sits in a disclosure that a refused submit leaves open.
async function addParent(page: Page, username: string, displayName: string) {
	if (!(await page.getByLabel('Username').isVisible())) {
		await page.locator('summary', { hasText: 'Create parent' }).click();
	}
	await page.getByLabel('Username').fill(username);
	await page.getByLabel('Display name').fill(displayName);
	await page.getByRole('button', { name: 'Create parent' }).click();
}

// The confirm button exists only inside the opened details.
async function disable(page: Page, username: string) {
	await row(page, username).locator('summary', { hasText: 'Disable' }).click();
	await row(page, username).getByRole('button', { name: 'Disable' }).click();
}

test('the list shows each parent with its status and no kid data', async ({ page }) => {
	await expect(page.getByRole('heading', { name: 'Parents', level: 1 })).toBeVisible();
	await expect(row(page, 'ana')).toContainText('Ana');
	await expect(row(page, 'ana')).toContainText('Active');
	await expect(row(page, 'bo')).toContainText('Invited');
	await expect(row(page, 'dee')).toContainText('Disabled');
	await expect(page.getByText('5 parents · 3 active · 1 invited · 1 disabled')).toBeVisible();
	const text = await page.locator('body').innerText();
	expect(text).not.toContain(MIA.name);
	expect(text).not.toContain(LEO.name);
	await expectAccessible(page);
});

test('create shows the invite link once, with its validity and a copy button', async ({
	page,
	context
}) => {
	await context.grantPermissions(['clipboard-read', 'clipboard-write'], { origin: UI_ORIGIN });
	await addParent(page, '  Eve ', 'Eve');
	const panel = page.getByRole('region', { name: 'Invite link for eve' });
	await expect(panel).toContainText(INVITE_TEXT);
	const link = await panel.getByLabel('One-time link').inputValue();
	expect(link).toMatch(LINK);
	expect(link).toContain('/invite/');
	await expect(row(page, 'eve')).toContainText('Invited');
	await expectAccessible(page);

	await panel.getByRole('button', { name: 'Copy link' }).click();
	await expect(panel.getByText('Copied')).toBeVisible();
	expect(await page.evaluate(() => navigator.clipboard.readText())).toBe(link);

	const token = link.split('/').pop()!;
	expect(page.url()).not.toContain(token);
	expect(JSON.stringify(await context.cookies())).not.toContain(token);
	expect(
		await page.evaluate(() => JSON.stringify({ ...localStorage, ...sessionStorage }))
	).not.toContain(token);

	await page.reload();
	await expect(page.getByRole('region', { name: /link for/ })).toHaveCount(0);
	await expect(page.getByText(token)).toHaveCount(0);
	await expect(row(page, 'eve')).toBeVisible();
});

test('a duplicate username and a malformed one get field messages', async ({ page }) => {
	await addParent(page, 'ana', 'Another Ana');
	await expectFieldError(page.getByLabel('Username'), 'That username is already taken');
	await expect(page.getByRole('region', { name: /link for/ })).toHaveCount(0);
	await expectAccessible(page);

	await addParent(page, 'no spaces', 'Someone');
	await expectFieldError(
		page.getByLabel('Username'),
		'Use 3 to 32 characters: a–z, 0–9, dot, underscore or hyphen'
	);

	await addParent(page, 'fern', '   ');
	await expectFieldError(page.getByLabel('Display name'), 'Use 1 to 64 characters');
	await expectAccessible(page);
});

test('a reset link lasts 1 hour; a parent without a password gets a 72-hour invite', async ({
	page
}) => {
	await row(page, 'ana').getByRole('button', { name: 'New reset link' }).click();
	const reset = page.getByRole('region', { name: 'Reset link for ana' });
	await expect(reset).toContainText(RESET_TEXT);
	expect(await reset.getByLabel('One-time link').inputValue()).toMatch(/\/reset\//);
	await expectAccessible(page);

	await row(page, 'bo').getByRole('button', { name: 'New invite link' }).click();
	const invite = page.getByRole('region', { name: 'Invite link for bo' });
	await expect(invite).toContainText(INVITE_TEXT);
	expect(await invite.getByLabel('One-time link').inputValue()).toMatch(/\/invite\//);
	await expect(page.getByRole('region', { name: 'Reset link for ana' })).toHaveCount(0);
});

test('disabling the only guardian of a child is refused with the sole-guardian text', async ({
	page
}) => {
	await disable(page, 'ana');
	await expect(page.getByRole('alert')).toHaveText(SOLE_GUARDIAN);
	await expect(row(page, 'ana')).toContainText('Active');
	await expectAccessible(page);
});

test('disable and enable a parent', async ({ page }) => {
	await row(page, 'dan').locator('summary', { hasText: 'Disable' }).click();
	await expect(row(page, 'dan')).toContainText('is signed out everywhere');
	await expectAccessible(page);
	await row(page, 'dan').getByRole('button', { name: 'Disable' }).click();
	await expect(row(page, 'dan')).toContainText('Disabled');
	await expectAccessible(page);
	await row(page, 'dan').getByRole('button', { name: 'Enable' }).click();
	await expect(row(page, 'dan')).toContainText('Active');
	await expect(row(page, 'dan').locator('summary', { hasText: 'Disable' })).toBeVisible();
});

test.describe('without JavaScript', () => {
	// axe runs in the page, so these pages are checked by the JavaScript tests above.
	test.use({ javaScriptEnabled: false });

	test('creating a parent shows a selectable link and no copy button', async ({ page }) => {
		await addParent(page, 'gus', 'Gus');
		const panel = page.getByRole('region', { name: 'Invite link for gus' });
		await expect(panel).toContainText(INVITE_TEXT);
		expect(await panel.getByLabel('One-time link').inputValue()).toMatch(LINK);
		await expect(panel.getByRole('button', { name: 'Copy link' })).toHaveCount(0);
		await expect(row(page, 'gus')).toContainText('Invited');
	});

	test('a duplicate username is refused without JavaScript', async ({ page }) => {
		await addParent(page, 'ana', 'Ana');
		await expectFieldError(page.getByLabel('Username'), 'That username is already taken');
	});
});

test('each parent says when they were last active and which link is waiting', async ({ page }) => {
	await openSession('cleo', 'fake-cleo-password-0000');
	await page.goto('/admin/parents');
	await expect(row(page, 'cleo')).toContainText(/Last active (Just now|\d+ min ago)/);
	await expect(row(page, 'ana')).toContainText('No recent sign-in');
	// The invited parent never signed in, and has an invite link waiting.
	await expect(row(page, 'bo')).toContainText('Hasn’t signed in yet');
	await expect(row(page, 'bo')).toContainText(
		/Invite link issued (Just now|\d+ min ago)\. Valid until \d{4}-\d{2}-\d{2} \d{2}:\d{2} UTC\./
	);
	await expect(row(page, 'ana')).toContainText(/Reset link issued /);
	await expect(row(page, 'cleo')).not.toContainText('link issued');
	await expectAccessible(page);

	await row(page, 'cleo')
		.getByRole('button', { name: /^New reset link/ })
		.click();
	await expect(row(page, 'cleo')).toContainText(/Reset link issued (Just now|\d+ min ago)/);
});
