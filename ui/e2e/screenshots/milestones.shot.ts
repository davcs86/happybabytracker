import type { Locator, Page } from '@playwright/test';
import { openSheet, submit } from '../helpers.js';
import { ANA, CLEO, NELL, OTTO, PIA, REMY, SAMI } from '../stub-api.js';
import { as, at, kidPath, must } from './seed.js';
import { expect, openDetails, shot, signInAs, test } from './shots.js';

const TODAY = at('00:00').slice(0, 10);
const url = (kid: string, rest = '') => `/kids/${kid}/milestones${rest}`;
const itemText = (id: string) => `Stub item ${id.split('-').slice(0, 2).join('-')}-1`;
const field = (where: Locator, label: string) => where.getByLabel(label, { exact: true });
const save = (sheet: Locator) => submit(sheet.getByRole('button', { name: 'Save', exact: true }));
const link = (name: string) => (p: Page) => p.getByRole('link', { name, exact: true });

async function open(page: Page, path: string) {
	await page.goto(path);
	await page.waitForLoadState('networkidle');
}

async function openAllLists(page: Page) {
	const summaries = page.locator('main details > summary');
	for (let i = 0; i < (await summaries.count()); i++) {
		const details = summaries.nth(i).locator('xpath=..');
		await expect(async () => {
			if ((await details.getAttribute('open')) === null) await summaries.nth(i).click();
			await expect(details).toHaveAttribute('open', '', { timeout: 1000 });
		}).toPass({ timeout: 10_000 });
	}
}

test('lists by age', async ({ page }) => {
	const cleo = await as(CLEO);
	await signInAs(page, CLEO);
	await open(page, url(NELL.id));
	await shot(
		page,
		'milestones',
		'01-nell-9-months',
		'Nine months: twelve lists in age order, the current one open and marked'
	);
	await openAllLists(page);
	await shot(page, 'milestones', '02-nell-all-lists-open', 'Every list opened (long page)');
	await open(page, url(NELL.id));
	await submit(
		page.getByRole('button', { name: `Reached ${itemText('9m-se-stub-1')}`, exact: true })
	);
	await page.waitForURL(/open=/);
	await shot(
		page,
		'milestones',
		'03-one-tap-reached',
		'One tap on Reached: today’s date shown on the item'
	);
	await open(page, url(NELL.id));
	await must(cleo, 'POST', kidPath(NELL.id, 'milestones'), {
		item: '9m-lc-stub-1',
		reached_on: TODAY
	});
	await submit(
		page.getByRole('button', { name: `Reached ${itemText('9m-lc-stub-1')}`, exact: true })
	);
	await expect(page.getByRole('alert').first()).toBeVisible();
	await shot(
		page,
		'milestones',
		'04-already-reached',
		'An item marked meanwhile on another device: already reached, in its row'
	);

	await open(page, url(OTTO.id));
	await shot(
		page,
		'milestones',
		'05-otto-4-years-11-months',
		'4 years 11 months: 4 years current, 5 years reachable; a retired row keeps its text'
	);
	await must(cleo, 'POST', kidPath(PIA.id, 'milestones'), {
		item: '60m-lc-stub-1',
		reached_on: TODAY
	});
	await must(cleo, 'POST', kidPath(PIA.id, 'milestones'), {
		text: 'Rides a bike without stabilisers',
		reached_on: TODAY,
		note: 'In the park'
	});
	await open(page, url(PIA.id));
	await shot(
		page,
		'milestones',
		'06-pia-5th-birthday',
		'The 5th birthday: no lists, the reached items and the custom rows'
	);
	await open(page, url(REMY.id));
	await shot(
		page,
		'milestones',
		'07-remy-preterm',
		'Born at 32 weeks, 3 months: both ages, the 2-month list as the next one'
	);
	await open(page, url(SAMI.id));
	await shot(
		page,
		'milestones',
		'08-sami-off-guardian',
		'30 months with the module off: a guardian gets a link to Modules'
	);
	await must(cleo, 'PUT', kidPath(SAMI.id, 'access/ana'), { role: 'caregiver' });
	await signInAs(page, ANA);
	await open(page, url(SAMI.id));
	await shot(
		page,
		'milestones',
		'09-sami-off-caregiver',
		'The same for a caregiver: a sentence, no link'
	);
});

test('forms', async ({ page }) => {
	const cleo = await as(CLEO);
	const yesterday = at('00:00', 1).slice(0, 10);
	await must(cleo, 'POST', kidPath(NELL.id, 'milestones'), {
		item: '9m-cg-stub-1',
		reached_on: yesterday,
		note: 'At the park'
	});
	await must(cleo, 'POST', kidPath(NELL.id, 'milestones'), {
		text: 'Waves bye-bye',
		reached_on: yesterday
	});
	await must(cleo, 'POST', kidPath(NELL.id, 'milestones'), {
		text: 'x'.repeat(200),
		reached_on: TODAY
	});
	await signInAs(page, CLEO);
	await open(page, url(NELL.id));
	await shot(
		page,
		'milestones',
		'10-reached-and-custom',
		'Items reached with a date, Your milestones with a 200-character unbroken text'
	);
	const mark = await openSheet(
		page,
		url(NELL.id),
		link(`Choose a date for ${itemText('9m-lc-stub-1')}`),
		'Mark as reached'
	);
	await shot(
		page,
		'milestones',
		'11-mark-with-date',
		'The Mark as reached sheet over the list: choose a date and a note'
	);
	await field(mark, 'Date reached').fill(at('00:00', -1).slice(0, 10));
	await save(mark);
	await expect(field(mark, 'Date reached')).toHaveAttribute('aria-invalid', 'true');
	await shot(page, 'milestones', '12-mark-future-date', 'Refused in the sheet: a future date');
	const add = await openSheet(page, url(NELL.id), link('Add a milestone'), 'Add a milestone');
	await shot(page, 'milestones', '13-new-custom', 'The Add a milestone sheet: text, date, note');
	await field(add, 'Milestone').fill('a'.repeat(201));
	await save(add);
	await expect(field(add, 'Milestone')).toHaveAttribute('aria-invalid', 'true');
	await shot(page, 'milestones', '14-new-custom-201-chars', 'Refused in the sheet: 201 characters');
	await openSheet(page, url(NELL.id), link(`Edit ${itemText('9m-cg-stub-1')}`), 'Edit milestone');
	await shot(
		page,
		'milestones',
		'15-edit-item-mark',
		'The Edit milestone sheet: a catalogue item’s date and note'
	);
	await openDetails(page, /undo|Undo/);
	await shot(
		page,
		'milestones',
		'16-undo-reached-confirm',
		'Undo reached opened inside the sheet: the confirm step'
	);
	await openSheet(page, url(NELL.id), link('Edit Waves bye-bye'), 'Edit milestone');
	await openDetails(page, /Delete/);
	await shot(
		page,
		'milestones',
		'17-edit-custom-delete-confirm',
		'The Edit milestone sheet for a custom milestone, delete opened'
	);
});
