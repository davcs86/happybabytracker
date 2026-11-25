import type { Page } from '@playwright/test';
import { submit, untilHydrated } from '../helpers.js';
import { ANA, CLEO, LEO, MIA } from '../stub-api.js';
import { as, at, must } from './seed.js';
import { expect, shot, signInAs, test } from './shots.js';

const field = (page: Page, label: string) => page.getByLabel(label, { exact: true });

async function open(page: Page, path: string) {
	await page.goto(path);
	await page.waitForLoadState('networkidle');
}

test('modules', async ({ page }) => {
	await signInAs(page, ANA);
	await open(page, `/kids/${MIA.id}/modules`);
	await shot(
		page,
		'kid-admin',
		'01-modules-under-12-months',
		'Modules for a 4-month-old: each with its default for the age'
	);
	await untilHydrated(page.getByLabel('Pumping', { exact: true }), 'checked');
	await page.getByLabel('Pumping', { exact: true }).uncheck();
	await page.getByLabel('Feeding', { exact: true }).uncheck();
	await submit(page.getByRole('button', { name: 'Save' }));
	await expect(page.getByRole('status')).toHaveText('Saved');
	await shot(
		page,
		'kid-admin',
		'02-modules-saved-two-off',
		'Pumping and Feeding turned off: Saved'
	);
	await open(page, `/kids/${MIA.id}`);
	await shot(
		page,
		'kid-admin',
		'03-home-with-modules-off',
		'The kid home without the Pumping and Feeding tiles'
	);
	await page.goto(`/kids/${LEO.id}/modules`);
	await shot(
		page,
		'kid-admin',
		'04-modules-caregiver-404',
		'A caregiver gets the 404 page on Modules'
	);
	await signInAs(page, CLEO);
	await open(page, `/kids/${LEO.id}/modules`);
	await shot(
		page,
		'kid-admin',
		'05-modules-24-plus',
		'Modules for a 2-year-old: the baby trackers off by default'
	);
});

test('settings', async ({ page }) => {
	await signInAs(page, ANA);
	await open(page, `/kids/${MIA.id}/settings`);
	await shot(
		page,
		'kid-admin',
		'06-settings',
		'Settings: name, birth date, sex, born early, time zone, growth source, schedule, overdue minutes'
	);
	await untilHydrated(field(page, 'Show a missed dose as overdue for (minutes)'));
	await field(page, 'Show a missed dose as overdue for (minutes)').fill('2');
	await submit(page.getByRole('button', { name: /^Save/ }));
	await expect(page.getByRole('alert').first()).toBeVisible();
	await shot(
		page,
		'kid-admin',
		'07-settings-overdue-out-of-range',
		'Refused: 2 overdue minutes, outside 5 to 1440'
	);
	await open(page, `/kids/${MIA.id}/settings`);
	await field(page, `Type ${MIA.name} to confirm`).fill('Mila');
	await submit(page.getByRole('button', { name: `Delete ${MIA.name}` }));
	await expect(page.getByText('The name does not match')).toBeVisible();
	await shot(page, 'kid-admin', '08-settings-delete-name-mismatch', 'Delete needs the exact name');
	await page.goto(`/kids/${LEO.id}/settings`);
	await shot(
		page,
		'kid-admin',
		'09-settings-caregiver-404',
		'A caregiver gets the 404 page on Settings'
	);
	await signInAs(page, CLEO);
	await open(page, `/kids/${LEO.id}/settings`);
	await shot(
		page,
		'kid-admin',
		'10-settings-born-early',
		'Settings of a kid born at 35 weeks 2 days, Europe/Madrid'
	);
});

test('access', async ({ page }) => {
	await signInAs(page, ANA);
	await open(page, `/kids/${MIA.id}/access`);
	await shot(
		page,
		'kid-admin',
		'11-access-sole-guardian',
		'Access: the only guardian, and the add form'
	);
	await submit(page.getByRole('button', { name: 'Leave' }));
	await expect(page.getByRole('alert').first()).toBeVisible();
	await shot(
		page,
		'kid-admin',
		'12-access-last-guardian',
		'The only guardian cannot leave: a child needs at least one guardian'
	);
	await field(page, 'Username').fill('nobody');
	await submit(page.getByRole('button', { name: 'Add' }));
	await expect(field(page, 'Username')).toHaveAttribute('aria-invalid', 'true');
	await shot(
		page,
		'kid-admin',
		'13-access-unknown-username',
		'Add someone: no active account has that username'
	);
	await field(page, 'Username').fill('dan');
	await page.getByLabel('Role', { exact: true }).selectOption('caregiver');
	await submit(page.getByRole('button', { name: 'Add' }));
	await expect(page.getByRole('button', { name: 'Remove Dan' })).toBeVisible();
	await shot(
		page,
		'kid-admin',
		'14-access-caregiver-added',
		'Dan added as a caregiver: role select, Change role and Remove'
	);
	await page.goto(`/kids/${LEO.id}/access`);
	await shot(
		page,
		'kid-admin',
		'15-access-caregiver-404',
		'A caregiver gets the 404 page on Access'
	);
	await signInAs(page, CLEO);
	await open(page, `/kids/${LEO.id}/access`);
	await shot(page, 'kid-admin', '16-access-guardian-and-caregiver', 'A guardian and a caregiver');
});

test('new kid', async ({ page }) => {
	const ana = await as(ANA);
	await signInAs(page, ANA);
	await open(page, '/kids/new');
	await shot(
		page,
		'kid-admin',
		'17-new-kid-empty',
		'Add a child: the time zone defaults to the device’s'
	);
	await page.getByRole('button', { name: 'Add child' }).click();
	await expect(page.getByRole('alert').first()).toBeVisible();
	await page.waitForLoadState('networkidle');
	await shot(page, 'kid-admin', '18-new-kid-errors', 'Refused: name, birth date and sex required');
	await field(page, 'Name').fill('Noa');
	await field(page, 'Birth date').fill(at('00:00', 20).slice(0, 10));
	await page.getByLabel('Born early?').check();
	await field(page, 'Weeks').fill('33');
	await field(page, 'Days').fill('4');
	await shot(
		page,
		'kid-admin',
		'19-new-kid-born-early',
		'Born early ticked: weeks and days appear'
	);
	const long = await must(ana, 'POST', '/v1/kids', {
		name: 'N'.repeat(64),
		birth_date: at('00:00', 40).slice(0, 10),
		sex: 'male',
		time_zone: 'UTC'
	});
	await open(page, `/kids/${long.id}`);
	await shot(
		page,
		'kid-admin',
		'20-long-kid-name-home',
		'A 64-character unbroken kid name: it overflows the kid home and the switcher at phone width (a UI defect this shot documents)'
	);
});
