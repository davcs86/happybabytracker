import type { Locator, Page } from '@playwright/test';
import { openSheet, submit } from '../helpers.js';
import { TRACKER_TITLE } from '../../src/lib/constants.js';
import type { TrackerKind } from '../../src/lib/types.js';
import { ANA, MIA } from '../stub-api.js';
import { as, at, kidPath, must, seedDay } from './seed.js';
import { expect, minutesAgo, openDetails, shot, signInAs, test } from './shots.js';

const log = (tracker: string, query = '') => `/kids/${MIA.id}/log/${tracker}${query}`;
const edit = (tracker: string, id: string) => `/kids/${MIA.id}/log/${tracker}/${id}`;
const field = (where: Page | Locator, name: string) => where.getByLabel(name, { exact: true });
const save = (where: Page | Locator) => submit(where.getByRole('button', { name: 'Save' }));
const HOME = `/kids/${MIA.id}`;
// Yesterday and today: the overnight sleep and the timer over 24 hours start yesterday.
const TIMELINE = `/kids/${MIA.id}/timeline?from=${at('00:00', 1).slice(0, 10)}&to=${at('00:00').slice(0, 10)}`;

async function open(page: Page, path: string) {
	await page.goto(path);
	await page.waitForLoadState('networkidle');
}

// A log form opens from its Quick log tile on Home; an edit form from its Timeline row.
const logSheet = (page: Page, tile: string, kind: TrackerKind) =>
	openSheet(
		page,
		HOME,
		(p) => p.getByRole('link', { name: tile, exact: true }),
		TRACKER_TITLE[kind].log
	);
const editSheet = (page: Page, kind: TrackerKind, id: string) =>
	openSheet(
		page,
		TIMELINE,
		(p) => p.locator(`a[href="${edit(kind, id)}"]:visible`).first(),
		TRACKER_TITLE[kind].edit
	);

// The advanced link swaps the form inside the same sheet.
async function switchMode(sheet: Locator, label: string, back: string) {
	await sheet.getByRole('link', { name: label, exact: true }).click();
	await expect(sheet.getByRole('link', { name: back, exact: true })).toBeVisible();
}

const openDelete = (page: Page) => openDetails(page, 'Delete this entry');

test('bottles', async ({ page }) => {
	const ana = await as(ANA);
	const rows = await seedDay(ana, MIA.id);
	await signInAs(page, ANA);
	let sheet = await logSheet(page, 'Bottle', 'bottles');
	await shot(
		page,
		'trackers',
		'01-bottle-new-empty',
		'The Log a bottle sheet over Home: the empty form, time prefilled'
	);
	await sheet.getByRole('checkbox', { name: 'Formula', exact: true }).check();
	await field(sheet, 'Volume').fill('120');
	await field(sheet, 'Note').fill('Before the nap');
	await shot(
		page,
		'trackers',
		'02-bottle-new-filled',
		'The Log a bottle sheet: formula, 120 ml and a note typed'
	);
	await sheet.getByRole('checkbox', { name: 'Formula', exact: true }).uncheck();
	await save(sheet);
	await expect(sheet.getByRole('alert').first()).toBeVisible();
	await shot(
		page,
		'trackers',
		'03-bottle-new-no-contents',
		'Refused in the sheet: no content ticked, said on the boxes'
	);
	await sheet.getByRole('checkbox', { name: 'Formula', exact: true }).check();
	await field(sheet, 'Volume').fill('5000');
	await save(sheet);
	await expect(field(sheet, 'Volume')).toHaveAttribute('aria-invalid', 'true');
	await shot(
		page,
		'trackers',
		'04-bottle-new-volume-range',
		'Refused in the sheet: a volume out of range names the range'
	);
	await field(sheet, 'Volume').fill('120');
	await field(sheet, 'Time').fill(at('11:30').slice(0, 16));
	await save(sheet);
	await expect(field(sheet, 'Time')).toHaveAttribute('aria-invalid', 'true');
	await shot(
		page,
		'trackers',
		'05-bottle-new-future-time',
		'Refused in the sheet: a time an hour ahead, other values kept'
	);

	sheet = await logSheet(page, 'Bottle', 'bottles');
	await switchMode(sheet, 'Advanced: volume per source', 'Use one total');
	await sheet.getByRole('checkbox', { name: 'Breast milk', exact: true }).check();
	await sheet.getByRole('checkbox', { name: 'Formula', exact: true }).check();
	await field(sheet, 'Breast milk volume').fill('60');
	await field(sheet, 'Formula volume').fill('90');
	await shot(
		page,
		'trackers',
		'06-bottle-new-advanced',
		'Advanced bottle in the sheet: a volume per source; the total is worked out on save'
	);
	await editSheet(page, 'bottles', rows.bottle.id);
	await shot(
		page,
		'trackers',
		'07-bottle-edit',
		'The Edit bottle sheet over the Timeline: a stored formula bottle'
	);
	await editSheet(page, 'bottles', rows.bottleMixed.id);
	await shot(
		page,
		'trackers',
		'08-bottle-edit-advanced',
		'The Edit bottle sheet for volumes per source (opens advanced)'
	);
	await openDelete(page);
	await shot(
		page,
		'trackers',
		'09-bottle-delete-confirm',
		'Delete opened inside the Edit bottle sheet: the confirm step'
	);

	await must(ana, 'PATCH', '/v1/me', { units: { volume: 'floz' } });
	await logSheet(page, 'Bottle', 'bottles');
	await shot(
		page,
		'trackers',
		'10-bottle-new-imperial',
		'The Log a bottle sheet for a parent who uses fluid ounces'
	);
	await editSheet(page, 'bottles', rows.bottle.id);
	await shot(
		page,
		'trackers',
		'11-bottle-edit-imperial',
		'A 120 ml bottle in the Edit bottle sheet of a fl oz parent'
	);
});

test('nursing', async ({ page }) => {
	const ana = await as(ANA);
	const rows = await seedDay(ana, MIA.id);
	await signInAs(page, ANA);
	let sheet = await logSheet(page, 'Nursing', 'nursing');
	await shot(
		page,
		'trackers',
		'12-nursing-new-empty',
		'The Log nursing sheet: start a timer on a side, or record a finished one'
	);
	await save(sheet);
	await expect(sheet.getByRole('alert').first()).toBeVisible();
	await shot(
		page,
		'trackers',
		'13-nursing-new-error',
		'Refused in the sheet: neither a side nor an end time'
	);
	sheet = await logSheet(page, 'Nursing', 'nursing');
	await switchMode(sheet, 'Advanced: weigh before and after', 'Hide weigh-ins');
	await shot(
		page,
		'trackers',
		'14-nursing-new-weigh-ins',
		'Advanced nursing in the sheet: weigh before and after'
	);
	await editSheet(page, 'nursing', rows.nursing.id);
	await shot(
		page,
		'trackers',
		'15-nursing-edit-estimate',
		'The Edit nursing sheet: minutes per side, weigh-ins and the milk estimate'
	);
	// The Nursing tile is gone while a timer runs: the second timer starts elsewhere meanwhile.
	sheet = await logSheet(page, 'Nursing', 'nursing');
	await must(ana, 'POST', kidPath(MIA.id, 'nursing'), { started_at: minutesAgo(12), side: 'left' });
	await sheet.getByRole('radio', { name: 'Right' }).check();
	await save(sheet);
	await expect(sheet.getByRole('link', { name: /timer/ })).toBeVisible();
	await shot(
		page,
		'trackers',
		'17-nursing-second-timer',
		'A timer started on another device meanwhile: the sheet refuses a second one, with a link to the running one'
	);
	const running = (await ana('GET', kidPath(MIA.id, 'home'))).body.running.nursing;
	await editSheet(page, 'nursing', running.id);
	await shot(
		page,
		'trackers',
		'16-nursing-edit-running',
		'The Edit nursing sheet for a running timer: no minutes, an end time ends it'
	);
});

test('pumping', async ({ page }) => {
	const ana = await as(ANA);
	const rows = await seedDay(ana, MIA.id);
	await signInAs(page, ANA);
	let sheet = await logSheet(page, 'Pumping', 'pumping');
	await shot(page, 'trackers', '18-pumping-new-empty', 'The Log pumping sheet: the empty form');
	await save(sheet);
	await expect(sheet.getByRole('alert').first()).toBeVisible();
	await shot(
		page,
		'trackers',
		'19-pumping-new-error',
		'Refused in the sheet: the first missing required field (Minutes) is marked; the form reports one problem at a time'
	);
	sheet = await logSheet(page, 'Pumping', 'pumping');
	await switchMode(sheet, 'Advanced: volume per side', 'Use one total');
	await shot(
		page,
		'trackers',
		'20-pumping-new-advanced',
		'Advanced pumping in the sheet: a volume per side; the total is worked out on save'
	);
	await editSheet(page, 'pumping', rows.pumping.id);
	await shot(
		page,
		'trackers',
		'21-pumping-edit',
		'The Edit pumping sheet: 20 minutes, 70 ml left and 55 ml right'
	);
});

test('diapers', async ({ page }) => {
	const ana = await as(ANA);
	const rows = await seedDay(ana, MIA.id);
	await signInAs(page, ANA);
	const sheet = await logSheet(page, 'Diaper', 'diapers');
	await shot(page, 'trackers', '22-diaper-new-empty', 'The Log a diaper sheet: the kinds');
	await save(sheet);
	await expect(sheet.getByRole('alert').first()).toBeVisible();
	await shot(page, 'trackers', '23-diaper-new-error', 'Refused in the sheet: no kind chosen');
	await editSheet(page, 'diapers', rows.diaperDirty.id);
	await shot(page, 'trackers', '24-diaper-edit', 'The Edit diaper sheet: dirty, with a note');
});

test('sleep', async ({ page }) => {
	const ana = await as(ANA);
	const rows = await seedDay(ana, MIA.id);
	await signInAs(page, ANA);
	// No tile or link opens Log sleep (the Sleep tile starts a timer): it is only ever a page.
	await open(page, log('sleep'));
	await shot(
		page,
		'trackers',
		'25-sleep-new-empty',
		'The Log sleep page, opened by its address (no link opens it in a sheet): started and ended'
	);
	await editSheet(page, 'sleep', rows.sleepOvernight.id);
	await shot(
		page,
		'trackers',
		'26-sleep-edit-finished',
		'The Edit sleep sheet over the Timeline: an overnight sleep that ended this morning'
	);
	const running = await must(ana, 'POST', kidPath(MIA.id, 'sleep'), { started_at: minutesAgo(40) });
	await editSheet(page, 'sleep', running.id);
	await shot(page, 'trackers', '27-sleep-edit-running', 'The Edit sleep sheet for a running timer');
	await open(page, log('sleep'));
	await save(page);
	await expect(page.getByRole('link', { name: /timer/ })).toBeVisible();
	await shot(
		page,
		'trackers',
		'28-sleep-second-timer',
		'The Log sleep page (opened by its address) refuses a second sleep timer, with a link to the running one'
	);
	await must(ana, 'DELETE', kidPath(MIA.id, `sleep/${running.id}`));
	const over = await must(ana, 'POST', kidPath(MIA.id, 'sleep'), {
		started_at: minutesAgo(26 * 60)
	});
	await editSheet(page, 'sleep', over.id);
	await shot(
		page,
		'trackers',
		'29-sleep-edit-overrun',
		'The Edit sleep sheet for a timer over 24 hours: set the end time or delete it'
	);
});

test('meals', async ({ page }) => {
	const ana = await as(ANA);
	const rows = await seedDay(ana, MIA.id);
	await signInAs(page, ANA);
	const sheet = await logSheet(page, 'Feeding', 'meals');
	await shot(page, 'trackers', '30-meal-new-empty', 'The Log a meal sheet: what and reaction');
	await save(sheet);
	await expect(field(sheet, 'What')).toHaveAttribute('aria-invalid', 'true');
	await shot(
		page,
		'trackers',
		'31-meal-new-error',
		'Refused in the sheet: what was eaten is required'
	);
	await editSheet(page, 'meals', rows.meal.id);
	await shot(page, 'trackers', '32-meal-edit', 'The Edit meal sheet: pureed carrot, liked it');
	const long = await must(ana, 'POST', kidPath(MIA.id, 'meals'), {
		at: at('10:20'),
		what: 'Sweetpotato'.repeat(18).slice(0, 200),
		note: 'Mashed'.repeat(50).slice(0, 200)
	});
	await editSheet(page, 'meals', long.id);
	await shot(
		page,
		'trackers',
		'33-meal-edit-long-text',
		'A 200-character unbroken meal and note stay inside their single-line fields: the sheet does not overflow'
	);
});

test('symptoms', async ({ page }) => {
	const ana = await as(ANA);
	const rows = await seedDay(ana, MIA.id);
	await signInAs(page, ANA);
	const sheet = await logSheet(page, 'Sickness', 'symptoms');
	await shot(
		page,
		'trackers',
		'34-symptom-new-empty',
		'The Log a symptom sheet: symptom and severity'
	);
	await save(sheet);
	await expect(field(sheet, 'Symptom')).toHaveAttribute('aria-invalid', 'true');
	await shot(page, 'trackers', '35-symptom-new-error', 'Refused in the sheet: no symptom chosen');
	await field(sheet, 'Symptom').selectOption('fever');
	await sheet.getByRole('radio', { name: 'Moderate' }).check();
	await save(sheet);
	await expect(field(sheet, 'Temperature')).toHaveAttribute('aria-invalid', 'true');
	await shot(
		page,
		'trackers',
		'36-symptom-fever-no-temperature',
		'Fever chosen in the sheet: temperature and method appear and are required'
	);
	await field(sheet, 'Temperature').fill('38.5');
	await field(sheet, 'Method').selectOption('oral');
	await shot(
		page,
		'trackers',
		'37-symptom-fever-filled',
		'Fever, 38.5 °C, oral, moderate typed in the sheet; the earlier refusal stays until the next save'
	);
	await field(sheet, 'Symptom').selectOption('other');
	await field(sheet, 'Other symptom').fill('Hiccups');
	await shot(page, 'trackers', '38-symptom-other', 'Other in the sheet: a free-text symptom');
	await editSheet(page, 'symptoms', rows.fever.id);
	await shot(
		page,
		'trackers',
		'39-symptom-edit-fever',
		'The Edit symptom sheet: a fever with its temperature and method'
	);
	await must(ana, 'PATCH', '/v1/me', { units: { temp: 'f' } });
	await editSheet(page, 'symptoms', rows.fever.id);
	await shot(
		page,
		'trackers',
		'40-symptom-edit-fahrenheit',
		'The same fever in the Edit symptom sheet for a parent who uses °F'
	);
});
