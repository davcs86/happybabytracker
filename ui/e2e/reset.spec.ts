import { expect, test } from '@playwright/test';
import { expectAccessible, expectLanding, login } from './helpers.js';
import { ANA, MIA, TOKENS, resetStub } from './stub-api.js';

const NEW_PASSWORD = 'a brand new password';

test.beforeEach(async () => {
	await resetStub();
});

test('a reset link sets a new password that then logs in', async ({ page, context }) => {
	await page.goto(`/reset/${TOKENS.reset}`);
	await expect(page.getByRole('heading', { name: 'Set a new password' })).toBeVisible();
	await expect(page.getByLabel('Username')).toHaveValue(ANA.username);
	await expectAccessible(page);

	await page.getByLabel('New password').fill(NEW_PASSWORD);
	await page.getByLabel('Confirm password').fill(NEW_PASSWORD);
	await page.getByRole('button', { name: 'Set password' }).click();
	await expectLanding(page, { kid: MIA });

	await context.clearCookies();
	await page.goto('/login');
	await login(page, ANA.username, NEW_PASSWORD);
	await expectLanding(page, { kid: MIA });
});

test('the reset form refuses a short password and a mismatch', async ({ page }) => {
	await page.goto(`/reset/${TOKENS.reset}`);
	await page.getByLabel('New password').fill('x'.repeat(11));
	await page.getByLabel('Confirm password').fill('x'.repeat(11));
	await page.getByRole('button', { name: 'Set password' }).click();
	await expect(page.getByText('Use at least 12 characters')).toBeVisible();
	await page.getByLabel('New password').fill(NEW_PASSWORD);
	await page.getByRole('button', { name: 'Set password' }).click();
	await expect(page.getByText('The passwords do not match')).toBeVisible();
	await expectAccessible(page);
});

for (const [cause, token] of [
	['expired', TOKENS.expired],
	['an invite', TOKENS.invite]
]) {
	test(`a reset link that is ${cause} is no longer valid`, async ({ page }) => {
		await page.goto(`/reset/${token}`);
		await expect(page.getByRole('heading', { name: 'Set a new password' })).toBeVisible();
		await expect(page.getByRole('alert')).toContainText('This link is no longer valid');
		await expectAccessible(page);
	});
}
