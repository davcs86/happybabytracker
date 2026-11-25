import type { Page } from '@playwright/test';
import { submit, untilHydrated } from '../helpers.js';
import { ANA, openSession } from '../stub-api.js';
import { expect, shot, signInAs, test } from './shots.js';

async function open(page: Page, path: string) {
	await page.goto(path);
	await page.waitForLoadState('networkidle');
}

async function changePassword(page: Page, current: string, next: string, confirm: string) {
	await page.getByLabel('Current password').fill(current);
	await page.getByLabel('New password', { exact: true }).fill(next);
	await page.getByLabel('Confirm new password').fill(confirm);
	await page.getByRole('button', { name: 'Change password' }).click();
}

test('account', async ({ page }) => {
	await openSession(ANA.username, ANA.password);
	await signInAs(page, ANA);
	await open(page, '/account');
	await shot(
		page,
		'account',
		'01-account',
		'Your profile: details and units, password, signed-in devices and the children you can see'
	);
	await untilHydrated(page.getByLabel('Display name'));
	await page.getByLabel('Weight').selectOption({ label: 'lb' });
	await page.getByLabel('Length').selectOption({ label: 'in' });
	await page.getByLabel('Volume').selectOption({ label: 'fl oz' });
	await page.getByLabel('Temperature').selectOption({ label: '°F' });
	await submit(page.getByRole('button', { name: 'Save' }));
	await expect(page.getByRole('status')).toHaveText('Saved');
	await shot(page, 'account', '02-units-imperial-saved', 'Imperial units saved');
	await page.getByLabel('Display name').fill('');
	await submit(page.getByRole('button', { name: 'Save' }));
	await expect(page.getByLabel('Display name')).toHaveAttribute('aria-invalid', 'true');
	await shot(page, 'account', '03-display-name-blank', 'Refused: a blank display name');
	await open(page, '/account');
	await changePassword(page, ANA.password, 'a long enough password', 'a long enough passwort');
	await expect(page.getByText('The passwords do not match')).toBeVisible();
	await shot(
		page,
		'account',
		'04-password-mismatch',
		'New password and confirmation differ (checked in the browser)'
	);
	await changePassword(
		page,
		'not-the-password-0',
		'a long enough password',
		'a long enough password'
	);
	await expect(page.getByRole('alert').first()).toBeVisible();
	await page.waitForLoadState('networkidle');
	await shot(page, 'account', '05-wrong-current-password', 'Refused: a wrong current password');
	await changePassword(page, ANA.password, 'a long enough password', 'a long enough password');
	await expect(page.getByText(/Password changed/)).toBeVisible();
	await page.waitForLoadState('networkidle');
	await shot(
		page,
		'account',
		'06-password-changed',
		'Password changed: the other session is signed out'
	);
});
