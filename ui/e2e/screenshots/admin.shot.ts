import type { Page } from '@playwright/test';
import { submit } from '../helpers.js';
import { ADMIN } from '../stub-api.js';
import { expect, shot, signInAs, test } from './shots.js';

// The stub draws random link tokens: the shots show this one instead.
const FROZEN_TOKEN = 'f'.repeat(43);

async function freezeLink(page: Page) {
	await page
		.getByTestId('parent-link')
		.evaluate(
			(el, token) => (el.textContent = el.textContent!.replace(/[A-Za-z0-9_-]{43}/, token)),
			FROZEN_TOKEN
		);
}

const row = (page: Page, username: string) =>
	page.getByRole('listitem').filter({ has: page.getByText(username, { exact: true }) });

async function addParent(page: Page, username: string, displayName: string) {
	await page.getByLabel('Username').fill(username);
	await page.getByLabel('Display name').fill(displayName);
	await submit(page.getByRole('button', { name: 'Add parent' }));
}

test('parents', async ({ page }) => {
	await signInAs(page, ADMIN);
	await expect(page.getByRole('heading', { name: 'Parents', level: 1 })).toBeVisible();
	await shot(
		page,
		'admin',
		'01-parents',
		'The admin landing: add a parent, every account with its status, no kid data'
	);
	await addParent(page, 'Not Valid!', 'Eve');
	await expect(page.getByLabel('Username')).toHaveAttribute('aria-invalid', 'true');
	await shot(page, 'admin', '02-add-malformed-username', 'Refused: a malformed username');
	await addParent(page, 'ana', 'Another Ana');
	await expect(page.getByLabel('Username')).toHaveAttribute('aria-invalid', 'true');
	await shot(page, 'admin', '03-add-duplicate-username', 'Refused: a username already taken');
	await addParent(page, 'eve', 'Eve');
	await expect(page.getByTestId('parent-link')).toBeVisible();
	await freezeLink(page);
	await shot(
		page,
		'admin',
		'04-invite-link',
		'Parent created: the 72-hour invite link shown once, with Copy link'
	);
	await submit(row(page, 'ana').getByRole('button', { name: /^Reset link/ }));
	await expect(page.getByText(/expires in 1 hour/)).toBeVisible();
	await freezeLink(page);
	await shot(
		page,
		'admin',
		'05-reset-link',
		'A reset link for a parent with a password lasts 1 hour'
	);
	await submit(row(page, 'ana').getByRole('button', { name: /^Disable/ }));
	await expect(page.getByRole('alert').first()).toBeVisible();
	await shot(
		page,
		'admin',
		'06-disable-sole-guardian-refused',
		'Disabling the only guardian of a child is refused'
	);
	await submit(row(page, 'dan').getByRole('button', { name: /^Disable/ }));
	await expect(row(page, 'dan').getByRole('button', { name: /^Enable/ })).toBeVisible();
	await shot(page, 'admin', '07-parent-disabled', 'Dan disabled: Enable offered');
});
