import AxeBuilder from '@axe-core/playwright';
import { expect, type BrowserContext, type Page } from '@playwright/test';

export const SESSION_COOKIE = 'hbt_session';

export async function expectAccessible(page: Page): Promise<void> {
	const results = await new AxeBuilder({ page }).analyze();
	expect(results.violations).toEqual([]);
}

export async function sessionCookie(context: BrowserContext) {
	return (await context.cookies()).find((c) => c.name === SESSION_COOKIE);
}

export async function login(page: Page, username: string, password: string) {
	await page.getByLabel('Username').fill(username);
	await page.getByLabel('Password').fill(password);
	const response = page.waitForResponse((r) => r.request().method() === 'POST');
	await page.getByRole('button', { name: 'Sign in' }).click();
	return response;
}
