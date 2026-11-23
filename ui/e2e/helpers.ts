import AxeBuilder from '@axe-core/playwright';
import { expect, type BrowserContext, type Locator, type Page } from '@playwright/test';

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

// Waits for the action's response: an assertion the previous submit already satisfied cannot.
export async function submit(button: Locator) {
	const page = button.page();
	const response = page.waitForResponse(
		(r) => r.request().method() === 'POST' && new URL(r.url()).search.startsWith('?/')
	);
	await button.click();
	await (await response).finished();
}

// Hydration resets prefilled controls, then drops the SSR value or checked attribute: edit them after.
export const untilHydrated = (input: Locator, attribute: 'value' | 'checked' = 'value') =>
	expect(input).not.toHaveAttribute(attribute);

export const KID_COOKIE = 'hbt_kid';

export async function signIn(page: Page, who: { username: string; password: string }) {
	await page.goto('/login');
	await login(page, who.username, who.password);
	await page.waitForURL((u) => u.pathname !== '/login');
}

// A field error is the field's accessible description and marks the field invalid.
export async function expectFieldError(field: Locator, message: string) {
	await expect(field).toHaveAttribute('aria-invalid', 'true');
	const escaped = message.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
	await expect(field).toHaveAccessibleDescription(new RegExp(escaped));
}

export async function expectNotFoundPage(page: Page, path: string) {
	const response = await page.goto(path);
	expect(response?.status()).toBe(404);
	await expect(page.getByRole('heading', { name: 'Not found' })).toBeVisible();
	await expectAccessible(page);
	return page.locator('body').innerText();
}

export type Landing = { kid: { id: string; name: string } } | { welcome: string };

// Where a parent lands after signing in: the kid home, or the welcome when there is no kid.
export async function expectLanding(page: Page, landing: Landing) {
	if ('welcome' in landing) {
		await expect(page.getByRole('heading', { name: `Welcome, ${landing.welcome}` })).toBeVisible();
		return;
	}
	await expect(page).toHaveURL(new RegExp(`/kids/${landing.kid.id}$`));
	await expect(page.getByRole('heading', { name: landing.kid.name, level: 1 })).toBeVisible();
}
