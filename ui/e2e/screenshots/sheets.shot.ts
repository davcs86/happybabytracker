import type { Locator, Page } from '@playwright/test';
import { openSheet, submit } from '../helpers.js';
import { ANA, CLEO, IVY, LEO, MIA, NELL } from '../stub-api.js';
import { as, kidPath, must } from './seed.js';
import { expect, shot, signInAs, test } from './shots.js';

const link = (name: string | RegExp) => (p: Page) => p.getByRole('link', { name });
const save = (sheet: Locator) => submit(sheet.getByRole('button', { name: 'Save', exact: true }));

test('record forms', async ({ page }) => {
	const ana = await as(ANA);
	await signInAs(page, ANA);

	const growth = await openSheet(
		page,
		`/kids/${IVY.id}/growth`,
		link('Add measurement'),
		'Add a measurement'
	);
	await shot(page, 'sheets', '01-growth-add', 'Add a measurement, opened over the Growth page');
	await growth.getByLabel('Note').fill('kept');
	await growth.getByRole('button', { name: 'Save', exact: true }).click();
	await expect(growth.getByText('Check the fields with a message.')).toBeVisible();
	await shot(
		page,
		'sheets',
		'02-growth-refused',
		'A save with no measure: the summary and the field message, with the Note kept'
	);
	await growth.getByLabel('Kilograms').fill('9.5');
	await save(growth);
	await expect(page.getByText('Measurement added')).toBeVisible();
	await shot(
		page,
		'sheets',
		'03-growth-saved',
		'After the save the sheet has closed, the list shows the measurement and a toast says so'
	);

	await must(ana, 'PATCH', kidPath(IVY.id), { vaccine_schema: 'us' });
	await openSheet(
		page,
		`/kids/${IVY.id}/vaccines`,
		link('Record a vaccination'),
		'Record a vaccination'
	);
	await shot(page, 'sheets', '04-vaccine-record', 'Record a vaccination, over the Vaccines page');

	await openSheet(page, `/kids/${MIA.id}/medication`, link('Add a course'), 'Add a course');
	await shot(page, 'sheets', '05-course-add', 'Add a course, over the Medication page');
	await must(ana, 'POST', kidPath(MIA.id, 'medications'), {
		name: 'Calpol',
		dose: '2.5 ml',
		schedule: { kind: 'times', times: ['08:00', '20:00'] }
	});
	await openSheet(page, `/kids/${MIA.id}/medication`, link('Edit Calpol'), 'Edit course');
	await shot(
		page,
		'sheets',
		'06-course-edit',
		'Edit course: the stored course and, below the form, the two-step delete'
	);

	await openSheet(page, `/kids/${MIA.id}`, link(/^Bottle$/), 'Log a bottle');
	await shot(page, 'sheets', '07-bottle-log', 'Log a bottle, opened from the Home quick-log tile');
	await openSheet(page, `/kids/${LEO.id}`, link('Sickness'), 'Log a symptom');
	await shot(page, 'sheets', '08-symptom-log', 'Log a symptom, opened from the Sickness tile');
});

test('milestone forms', async ({ page }) => {
	await signInAs(page, CLEO);
	await openSheet(
		page,
		`/kids/${NELL.id}/milestones`,
		(p) => p.getByRole('link', { name: /^Choose a date for/ }).first(),
		'Mark as reached'
	);
	await shot(
		page,
		'sheets',
		'09-milestone-mark',
		'Mark as reached: the milestone text and the date it was reached'
	);
	await openSheet(page, `/kids/${NELL.id}/milestones`, link('Add a milestone'), 'Add a milestone');
	await shot(page, 'sheets', '10-milestone-add', 'Add a milestone of your own');
});

test('child details, modules, access and Add a child', async ({ page }) => {
	await signInAs(page, ANA);
	const profile = `/kids/${MIA.id}/profile`;

	await openSheet(page, profile, link('Edit details'), 'Edit details');
	await shot(page, 'sheets', '11-edit-details', 'Edit details, over the Child profile');
	await openSheet(page, profile, link(/^Trackers/), `Modules for ${MIA.name}`);
	await shot(page, 'sheets', '12-modules', 'Modules for a child, over the Child profile');
	const access = await openSheet(page, profile, link('Manage access'), 'Access');
	await shot(
		page,
		'sheets',
		'13-access',
		'Access: the people who can see the child and who to add'
	);
	await access.getByLabel('Parent’s username').fill('dan');
	await access.getByRole('button', { name: 'Give access', exact: true }).click();
	await expect(access.getByRole('listitem').filter({ hasText: 'Dan' })).toBeVisible();
	await shot(
		page,
		'sheets',
		'14-access-added',
		'After Give access the sheet stays open with the new person listed, and a toast says so'
	);

	const switcher = page.getByRole('dialog', { name: 'Switch child' });
	await expect(async () => {
		await page.goto('/');
		await page.getByRole('link', { name: /Switch child$/ }).click();
		await expect(switcher).toBeVisible({ timeout: 1500 });
	}).toPass();
	await switcher.getByRole('link', { name: 'Add a child' }).click();
	await expect(page.getByRole('dialog', { name: 'Add a child' })).toBeVisible();
	await shot(page, 'sheets', '15-add-child', 'Add a child, opened from the Switch child sheet');
});
