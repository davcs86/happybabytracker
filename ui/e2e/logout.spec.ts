import { expect, test } from '@playwright/test';
import { expectAccessible, login, sessionCookie } from './helpers.js';
import { ADMIN, ANA, receivedRequests, resetStub } from './stub-api.js';

test.beforeEach(async () => {
	await resetStub();
});

test('the admin is kept inside /admin/ and can sign out from /admin/parents', async ({
	page,
	context
}) => {
	await page.goto('/login');
	await login(page, ADMIN.username, ADMIN.password);
	await expect(page).toHaveURL(/\/admin\/parents$/);
	for (const path of ['/', '/account', '/admin', '/invite/x']) {
		await page.goto(path);
		await expect(page).toHaveURL(/\/admin\/parents$/);
	}
	const session = (await sessionCookie(context))?.value;

	await page.getByRole('button', { name: 'Sign out' }).click();
	await expect(page).toHaveURL(/\/login$/);
	await expectAccessible(page);
	expect(await sessionCookie(context)).toBeUndefined();
	expect(await receivedRequests()).toContainEqual(
		expect.objectContaining({ method: 'DELETE', path: '/v1/session', session })
	);
	await page.goto('/admin/parents');
	await expect(page).toHaveURL(/\/login\?next=%2Fadmin%2Fparents$/);
});

test('a parent signs out and / then goes to /login', async ({ page, context }) => {
	await page.goto('/login');
	await login(page, ANA.username, ANA.password);
	await page.getByRole('button', { name: 'Sign out' }).click();
	await expect(page).toHaveURL(/\/login$/);
	expect(await sessionCookie(context)).toBeUndefined();
	await page.goto('/');
	await expect(page).toHaveURL(/\/login\?next=%2F$/);
});

test('an expired session clears the cookie and goes to /login', async ({ page, context }) => {
	await page.goto('/login');
	await login(page, ANA.username, ANA.password);
	await resetStub();
	await page.goto('/');
	await expect(page).toHaveURL(/\/login\?next=%2F$/);
	expect(await sessionCookie(context)).toBeUndefined();
});
