import type { Page } from '@playwright/test';
import { submit } from '../helpers.js';
import { ANA, MIA } from '../stub-api.js';
import { as, at, kidPath, must, seedDay } from './seed.js';
import { expect, minutesAgo, openDetails, shot, signInAs, test } from './shots.js';

const log = (tracker: string, query = '') => `/kids/${MIA.id}/log/${tracker}${query}`;
const edit = (tracker: string, id: string) => `/kids/${MIA.id}/log/${tracker}/${id}`;
const field = (page: Page, name: string) => page.getByLabel(name, { exact: true });
const save = (page: Page) => submit(page.getByRole('button', { name: 'Save' }));

async function open(page: Page, path: string) {
	await page.goto(path);
	await page.waitForLoadState('networkidle');
}

const openDelete = (page: Page) => openDetails(page, 'Delete this entry');

test('bottles', async ({ page }) => {
	const ana = await as(ANA);
	const rows = await seedDay(ana, MIA.id);
	await signInAs(page, ANA);
	await open(page, log('bottles'));
	await shot(
		page,
		'trackers',
		'01-bottle-new-empty',
		'Log a bottle: the empty form, time prefilled'
	);
	await page.getByRole('checkbox', { name: 'Formula', exact: true }).check();
	await field(page, 'Volume').fill('120');
	await field(page, 'Note').fill('Before the nap');
	await shot(
		page,
		'trackers',
		'02-bottle-new-filled',
		'Log a bottle: formula, 120 ml and a note typed'
	);
	await page.getByRole('checkbox', { name: 'Formula', exact: true }).uncheck();
	await save(page);
	await expect(page.getByRole('alert').first()).toBeVisible();
	await shot(
		page,
		'trackers',
		'03-bottle-new-no-contents',
		'Refused: no content ticked, said on the boxes'
	);
	await page.getByRole('checkbox', { name: 'Formula', exact: true }).check();
	await field(page, 'Volume').fill('5000');
	await save(page);
	await expect(field(page, 'Volume')).toHaveAttribute('aria-invalid', 'true');
	await shot(
		page,
		'trackers',
		'04-bottle-new-volume-range',
		'Refused: a volume out of range names the range'
	);
	await field(page, 'Volume').fill('120');
	await field(page, 'Time').fill(at('11:30').slice(0, 16));
	await save(page);
	await expect(field(page, 'Time')).toHaveAttribute('aria-invalid', 'true');
	await shot(
		page,
		'trackers',
		'05-bottle-new-future-time',
		'Refused: a time an hour ahead, other values kept'
	);

	await open(page, log('bottles', '?advanced=1'));
	await page.getByRole('checkbox', { name: 'Breast milk', exact: true }).check();
	await page.getByRole('checkbox', { name: 'Formula', exact: true }).check();
	await field(page, 'Breast milk volume').fill('60');
	await field(page, 'Formula volume').fill('90');
	await shot(page, 'trackers', '06-bottle-new-advanced', 'Advanced bottle: a volume per source');
	await open(page, edit('bottles', rows.bottle.id));
	await shot(page, 'trackers', '07-bottle-edit', 'Edit bottle: a stored formula bottle');
	await open(page, edit('bottles', rows.bottleMixed.id));
	await shot(
		page,
		'trackers',
		'08-bottle-edit-advanced',
		'Edit bottle with volumes per source (opens advanced)'
	);
	await openDelete(page);
	await shot(page, 'trackers', '09-bottle-delete-confirm', 'Delete opened: the confirm step');

	await must(ana, 'PATCH', '/v1/me', { units: { volume: 'floz' } });
	await open(page, log('bottles'));
	await shot(
		page,
		'trackers',
		'10-bottle-new-imperial',
		'Log a bottle for a parent who uses fluid ounces'
	);
	await open(page, edit('bottles', rows.bottle.id));
	await shot(
		page,
		'trackers',
		'11-bottle-edit-imperial',
		'A 120 ml bottle on the edit page of a fl oz parent'
	);
});

test('nursing', async ({ page }) => {
	const ana = await as(ANA);
	const rows = await seedDay(ana, MIA.id);
	await signInAs(page, ANA);
	await open(page, log('nursing'));
	await shot(
		page,
		'trackers',
		'12-nursing-new-empty',
		'Log nursing: start a timer on a side, or record a finished one'
	);
	await save(page);
	await expect(page.getByRole('alert').first()).toBeVisible();
	await shot(page, 'trackers', '13-nursing-new-error', 'Refused: neither a side nor an end time');
	await open(page, log('nursing', '?advanced=1'));
	await shot(
		page,
		'trackers',
		'14-nursing-new-weigh-ins',
		'Advanced nursing: weigh before and after'
	);
	await open(page, edit('nursing', rows.nursing.id));
	await shot(
		page,
		'trackers',
		'15-nursing-edit-estimate',
		'Edit nursing: minutes per side, weigh-ins and the milk estimate'
	);
	await must(ana, 'POST', kidPath(MIA.id, 'nursing'), { started_at: minutesAgo(12), side: 'left' });
	const running = (await ana('GET', kidPath(MIA.id, 'home'))).body.running.nursing;
	await open(page, edit('nursing', running.id));
	await shot(
		page,
		'trackers',
		'16-nursing-edit-running',
		'Edit a running nursing timer: no minutes, an end time ends it'
	);
	await open(page, log('nursing'));
	await page.getByRole('radio', { name: 'Right' }).check();
	await save(page);
	await expect(page.getByRole('link', { name: /timer/ })).toBeVisible();
	await shot(
		page,
		'trackers',
		'17-nursing-second-timer',
		'A second nursing timer is refused with a link to the running one'
	);
});

test('pumping', async ({ page }) => {
	const ana = await as(ANA);
	const rows = await seedDay(ana, MIA.id);
	await signInAs(page, ANA);
	await open(page, log('pumping'));
	await shot(page, 'trackers', '18-pumping-new-empty', 'Log pumping: the empty form');
	await save(page);
	await expect(page.getByRole('alert').first()).toBeVisible();
	await shot(page, 'trackers', '19-pumping-new-error', 'Refused: the required fields are marked');
	await open(page, log('pumping', '?advanced=1'));
	await shot(page, 'trackers', '20-pumping-new-advanced', 'Advanced pumping: a volume per side');
	await open(page, edit('pumping', rows.pumping.id));
	await shot(
		page,
		'trackers',
		'21-pumping-edit',
		'Edit pumping: 20 minutes, 70 ml left and 55 ml right'
	);
});

test('diapers', async ({ page }) => {
	const ana = await as(ANA);
	const rows = await seedDay(ana, MIA.id);
	await signInAs(page, ANA);
	await open(page, log('diapers'));
	await shot(page, 'trackers', '22-diaper-new-empty', 'Log a diaper: the kinds');
	await save(page);
	await expect(page.getByRole('alert').first()).toBeVisible();
	await shot(page, 'trackers', '23-diaper-new-error', 'Refused: no kind chosen');
	await open(page, edit('diapers', rows.diaperDirty.id));
	await shot(page, 'trackers', '24-diaper-edit', 'Edit diaper: dirty, with a note');
});

test('sleep', async ({ page }) => {
	const ana = await as(ANA);
	const rows = await seedDay(ana, MIA.id);
	await signInAs(page, ANA);
	await open(page, log('sleep'));
	await shot(page, 'trackers', '25-sleep-new-empty', 'Log sleep: started and ended');
	await open(page, edit('sleep', rows.sleepOvernight.id));
	await shot(
		page,
		'trackers',
		'26-sleep-edit-finished',
		'Edit sleep: an overnight sleep that ended this morning'
	);
	const running = await must(ana, 'POST', kidPath(MIA.id, 'sleep'), { started_at: minutesAgo(40) });
	await open(page, edit('sleep', running.id));
	await shot(page, 'trackers', '27-sleep-edit-running', 'Edit a running sleep timer');
	await open(page, log('sleep'));
	await save(page);
	await expect(page.getByRole('link', { name: /timer/ })).toBeVisible();
	await shot(
		page,
		'trackers',
		'28-sleep-second-timer',
		'A second sleep timer is refused with a link to the running one'
	);
	await must(ana, 'DELETE', kidPath(MIA.id, `sleep/${running.id}`));
	const over = await must(ana, 'POST', kidPath(MIA.id, 'sleep'), {
		started_at: minutesAgo(26 * 60)
	});
	await open(page, edit('sleep', over.id));
	await shot(
		page,
		'trackers',
		'29-sleep-edit-overrun',
		'A timer over 24 hours: set the end time or delete it'
	);
});

test('meals', async ({ page }) => {
	const ana = await as(ANA);
	const rows = await seedDay(ana, MIA.id);
	await signInAs(page, ANA);
	await open(page, log('meals'));
	await shot(page, 'trackers', '30-meal-new-empty', 'Log a meal: what and reaction');
	await save(page);
	await expect(field(page, 'What')).toHaveAttribute('aria-invalid', 'true');
	await shot(page, 'trackers', '31-meal-new-error', 'Refused: what was eaten is required');
	await open(page, edit('meals', rows.meal.id));
	await shot(page, 'trackers', '32-meal-edit', 'Edit meal: pureed carrot, liked it');
	const long = await must(ana, 'POST', kidPath(MIA.id, 'meals'), {
		at: at('10:20'),
		what: 'Sweetpotato'.repeat(18).slice(0, 200),
		note: 'Mashed'.repeat(50).slice(0, 200)
	});
	await open(page, edit('meals', long.id));
	await shot(
		page,
		'trackers',
		'33-meal-edit-long-text',
		'A 200-character unbroken meal and note wrap without overflow'
	);
});

test('symptoms', async ({ page }) => {
	const ana = await as(ANA);
	const rows = await seedDay(ana, MIA.id);
	await signInAs(page, ANA);
	await open(page, log('symptoms'));
	await shot(page, 'trackers', '34-symptom-new-empty', 'Log a symptom: symptom and severity');
	await save(page);
	await expect(field(page, 'Symptom')).toHaveAttribute('aria-invalid', 'true');
	await shot(page, 'trackers', '35-symptom-new-error', 'Refused: no symptom chosen');
	await field(page, 'Symptom').selectOption('fever');
	await page.getByRole('radio', { name: 'Moderate' }).check();
	await save(page);
	await expect(field(page, 'Temperature')).toHaveAttribute('aria-invalid', 'true');
	await shot(
		page,
		'trackers',
		'36-symptom-fever-no-temperature',
		'Fever chosen: temperature and method appear and are required'
	);
	await field(page, 'Temperature').fill('38.5');
	await field(page, 'Method').selectOption('oral');
	await shot(page, 'trackers', '37-symptom-fever-filled', 'Fever, 38.5 °C, oral, moderate');
	await field(page, 'Symptom').selectOption('other');
	await field(page, 'Other symptom').fill('Hiccups');
	await shot(page, 'trackers', '38-symptom-other', 'Other: a free-text symptom');
	await open(page, edit('symptoms', rows.fever.id));
	await shot(
		page,
		'trackers',
		'39-symptom-edit-fever',
		'Edit symptom: a fever with its temperature and method'
	);
	await must(ana, 'PATCH', '/v1/me', { units: { temp: 'f' } });
	await open(page, edit('symptoms', rows.fever.id));
	await shot(
		page,
		'trackers',
		'40-symptom-edit-fahrenheit',
		'The same fever for a parent who uses °F'
	);
});
