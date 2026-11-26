import type { Locator, Page } from '@playwright/test';
import { openSheet, submit } from '../helpers.js';
import { VACCINE_LOT_MAX_LEN } from '../../src/lib/constants.js';
import { ANA, CLEO, LEO, MIA, setScenario } from '../stub-api.js';
import { as, at, kidPath, must, seedVisits } from './seed.js';
import { expect, openDetails, shot, signInAs, test } from './shots.js';

const list = (kid: string = MIA.id) => `/kids/${kid}/visits`;
const visit = (id: string, kid: string = MIA.id) => `${list(kid)}/${id}`;
const field = (where: Page | Locator, label: string) => where.getByLabel(label, { exact: true });
const link = (name: string | RegExp) => (p: Page) => p.getByRole('link', { name });

async function open(page: Page, path: string) {
	await page.goto(path);
	await page.waitForLoadState('networkidle');
}

test('visits list and the visit page', async ({ page }) => {
	const ana = await as(ANA);
	await must(ana, 'PATCH', kidPath(MIA.id), { vaccine_schema: 'us' });
	await signInAs(page, ANA);
	await open(page, list());
	await shot(page, 'visits', '01-no-visits', 'No visit yet: the sentence and Add a visit');
	const v = await seedVisits(ana, MIA.id);
	await open(page, list());
	await shot(
		page,
		'visits',
		'02-list',
		'Upcoming soonest first, with the cancelled visit and its badge; Past newest first'
	);
	await open(page, visit(v.checkup.id));
	await shot(
		page,
		'visits',
		'03-visit-upcoming',
		'An upcoming visit: when, doctor and place, reason, the kid’s zone, and Add to calendar'
	);
	await open(page, visit(v.plain.id));
	await shot(
		page,
		'visits',
		'04-visit-past-no-outcome',
		'A past visit with nothing recorded yet: Record the visit'
	);
	await open(page, visit(v.sick.id));
	await shot(
		page,
		'visits',
		'05-visit-past-outcome',
		'A past visit with its notes under After the visit and the three records under Recorded at this visit'
	);
	await open(page, visit(v.cancelled.id));
	await shot(
		page,
		'visits',
		'06-visit-cancelled',
		'A cancelled visit: the badge, Restore visit, and Add to calendar for the cancelling file'
	);
	await open(page, visit(v.sick.id));
	await openDetails(page, 'Delete visit');
	await shot(
		page,
		'visits',
		'07-delete-confirm-linked',
		'Delete visit opened: what the visit recorded, the course with its logged dose, and Also delete these records ticked'
	);
	await open(page, visit(v.plain.id));
	await openDetails(page, 'Delete visit');
	await shot(
		page,
		'visits',
		'08-delete-confirm-plain',
		'Delete visit opened on a visit with nothing recorded: one sentence, no checkbox'
	);
});

test('visit sheets', async ({ page }) => {
	const ana = await as(ANA);
	await must(ana, 'PATCH', kidPath(MIA.id), { vaccine_schema: 'us' });
	const v = await seedVisits(ana, MIA.id);
	await signInAs(page, ANA);

	await openSheet(page, list(), link('Add a visit'), 'Add a visit');
	await shot(page, 'visits', '09-add-sheet', 'Add a visit, opened over the list');
	await openSheet(page, visit(v.checkup.id), link('Edit visit'), 'Edit visit');
	await shot(page, 'visits', '10-edit-sheet', 'Edit visit: the stored visit, over its page');

	const sheet = await openSheet(
		page,
		visit(v.plain.id),
		link('Record the visit'),
		'After the visit'
	);
	await shot(
		page,
		'visits',
		'11-outcome-sheet',
		'After the visit: notes, and a box for each of Growth, Vaccines and Medication'
	);
	await field(sheet, 'Notes from the visit').fill('Feeding well. Weigh again in two weeks.');
	await sheet.getByRole('checkbox', { name: 'Add a growth measurement', exact: true }).check();
	await field(sheet, 'Kilograms').fill('6.1');
	await sheet.getByRole('checkbox', { name: 'Add vaccines given', exact: true }).check();
	const vaccine = (n: number) => sheet.getByRole('group', { name: `Vaccine ${n}`, exact: true });
	await field(vaccine(1), 'Measles').check();
	await sheet.getByRole('button', { name: 'Add another vaccine' }).click();
	await field(vaccine(2), 'Measles').check();
	await field(vaccine(2), 'Lot number').fill('x'.repeat(VACCINE_LOT_MAX_LEN + 1));
	await sheet.getByRole('button', { name: 'Save', exact: true }).click();
	await expect(sheet.getByText('Check the fields with a message.')).toBeVisible();
	await field(vaccine(2), 'Lot number').scrollIntoViewIfNeeded();
	await shot(
		page,
		'visits',
		'12-outcome-refused',
		'A save with a too-long lot in vaccine 2: the sheet stays open and the message names its block'
	);
});

test('visits elsewhere', async ({ page }) => {
	const ana = await as(ANA);
	const v = await seedVisits(ana, MIA.id);
	await signInAs(page, ANA);
	await open(page, `/kids/${MIA.id}`);
	await shot(
		page,
		'visits',
		'13-home-due',
		'Home’s Due: the course a visit started, then the well-child visit in three days'
	);
	const from = at('00:00', 14).slice(0, 10);
	const to = at('00:00', -21).slice(0, 10);
	await open(page, `/kids/${MIA.id}/timeline?from=${from}&to=${to}&types=visit`);
	await shot(
		page,
		'visits',
		'14-timeline-visits',
		'The timeline narrowed to Visits over five weeks: one row per visit, the cancelled one marked'
	);
	await openSheet(
		page,
		`/kids/${MIA.id}/growth`,
		(p) => p.locator(`a[href="/kids/${MIA.id}/growth/${v.growth}"]`).first(),
		'Edit measurement'
	);
	await shot(
		page,
		'visits',
		'15-recorded-at-a-visit',
		'Edit measurement for a weight taken at a visit: Recorded at a visit links back to it'
	);
});

test('visits switched off and caregiver', async ({ page }) => {
	const ana = await as(ANA);
	await seedVisits(ana, MIA.id);
	await must(ana, 'PUT', kidPath(MIA.id, 'modules/visits'), { enabled: false });
	await signInAs(page, ANA);
	await open(page, list());
	await shot(
		page,
		'visits',
		'16-module-off',
		'Medical visits switched off: the visits are kept but not listed; a guardian gets Open modules'
	);
	const cleo = await as(CLEO);
	await seedVisits(cleo, LEO.id);
	await open(page, list(LEO.id));
	await shot(
		page,
		'visits',
		'17-caregiver',
		'A caregiver sees the same list and controls (kid zone Europe/Madrid)'
	);
});

// The "Find a US provider" panel against the stub's fictional registry (search is off by default, as at the edge).
const summary = (where: Page | Locator) => where.getByText('Find a US provider', { exact: true });
const button = (where: Page | Locator, name: string) =>
	where.getByRole('button', { name, exact: true });

async function findQuill(where: Page | Locator) {
	await summary(where).click();
	await field(where, 'Last name').fill('Quill');
	await field(where, 'State').selectOption('WA');
	await submit(button(where, 'Search'));
}

test('find a US provider', async ({ page }) => {
	await setScenario('npi_on');
	await signInAs(page, ANA);
	let sheet = await openSheet(page, list(), link('Add a visit'), 'Add a visit');
	await summary(sheet).scrollIntoViewIfNeeded();
	await shot(
		page,
		'visits',
		'18-provider-closed',
		'Add a visit with the provider search on: Find a US provider, closed, above Doctor or clinic'
	);
	await findQuill(sheet);
	const results = sheet.getByRole('radiogroup', { name: 'Choose a provider' });
	await expect(results.getByRole('radio')).toHaveCount(2);
	await results.scrollIntoViewIfNeeded();
	await shot(
		page,
		'visits',
		'19-provider-results',
		'A search for Quill in Washington: the two in-state practices, each with its specialty and address'
	);
	await sheet.getByRole('radio', { name: 'AVERY QUILL, MD', exact: true }).check();
	await submit(button(sheet, 'Use this provider'));
	await expect(field(sheet, 'Last name')).toBeHidden();
	await field(sheet, 'Place (optional)').scrollIntoViewIfNeeded();
	await shot(
		page,
		'visits',
		'20-provider-picked',
		'After Use this provider: Doctor or clinic and Place filled in, with the registry line and its NPI'
	);

	sheet = await openSheet(page, list(), link('Add a visit'), 'Add a visit');
	await summary(sheet).click();
	await submit(button(sheet, 'Search'));
	await expect(sheet.getByText('Enter a last name of 2 to 60 letters')).toBeVisible();
	await field(sheet, 'Last name').evaluate((el) => el.scrollIntoView({ block: 'center' }));
	await shot(
		page,
		'visits',
		'21-provider-refused',
		'Search with no last name: the message sits on the field, and no alert above the form'
	);
});

test('find a US provider when the registry does not answer', async ({ page }) => {
	await signInAs(page, ANA);
	for (const [scenario, name, caption] of [
		[
			'npi_unavailable',
			'22-provider-unavailable',
			'The registry is unavailable: the alert in the panel; the visit can still be typed and saved'
		],
		[
			'npi_rate_limited',
			'23-provider-rate-limited',
			'Too many searches in a minute: the alert in the panel'
		]
	] as const) {
		await setScenario(scenario);
		const sheet = await openSheet(page, list(), link('Add a visit'), 'Add a visit');
		await findQuill(sheet);
		const alert = sheet.getByRole('alert');
		await expect(alert).toBeVisible();
		await alert.scrollIntoViewIfNeeded();
		await shot(page, 'visits', name, caption);
	}
});

test.describe('find a US provider without JavaScript', () => {
	test.use({ javaScriptEnabled: false });

	test('results on the page', async ({ page }) => {
		await setScenario('npi_on');
		await signInAs(page, ANA);
		await open(page, `${list()}/new`);
		await findQuill(page);
		await expect(page.getByRole('radiogroup', { name: 'Choose a provider' })).toBeVisible();
		await shot(
			page,
			'visits',
			'24-provider-no-js',
			'Without JavaScript Search posts the whole form: the Add a visit page again, the results in the open panel'
		);
	});
});
