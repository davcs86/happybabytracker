import type { Locator, Page } from '@playwright/test';
import { openSheet, submit } from '../helpers.js';
import { ANA, CLEO, IVY, LEO, MIA, storeVaccineSchema } from '../stub-api.js';
import { as, at, kidPath, must } from './seed.js';
import { expect, openDetails, shot, signInAs, test } from './shots.js';

const TODAY = at('00:00').slice(0, 10);
const AT_TWO_MONTHS = new Date(Date.parse(`${IVY.birthDate}T00:00:00Z`) + 60 * 86_400_000)
	.toISOString()
	.slice(0, 10);
const url = (kid: string, rest = '') => `/kids/${kid}/vaccines${rest}`;
const field = (where: Page | Locator, label: string) => where.getByLabel(label, { exact: true });
const save = (where: Page | Locator) =>
	submit(where.getByRole('button', { name: 'Save', exact: true }));

// A dose's Given link can sit in a folded visit card: unfold that card, then open the sheet.
async function openEdit(page: Page, id: string) {
	const href = url(IVY.id, `/${id}`);
	const sheet = page
		.getByRole('dialog', { name: 'Edit vaccination' })
		.and(page.locator('[data-state="open"]'));
	await expect(async () => {
		await open(page, url(IVY.id));
		await page
			.locator(`details:has(a[href="${href}"])`)
			.evaluateAll((ds) => ds.forEach((d) => ((d as HTMLDetailsElement).open = true)));
		await page.locator(`a[href="${href}"]:visible`).first().click({ timeout: 1500 });
		await expect(sheet).toBeVisible({ timeout: 1500 });
	}).toPass();
	// The sheet moves focus in after it opens: wait, or the capture's blur can come first.
	await expect.poll(() => sheet.evaluate((el) => el.contains(document.activeElement))).toBe(true);
}

async function open(page: Page, path: string) {
	await page.goto(path);
	await page.waitForLoadState('networkidle');
}

test('schedules and dismissals', async ({ page }) => {
	const ana = await as(ANA);
	await signInAs(page, ANA);
	await open(page, url(MIA.id));
	await shot(
		page,
		'vaccines',
		'01-no-schedule-guardian',
		'No schedule chosen: a guardian gets a link to the settings'
	);
	await must(ana, 'PATCH', kidPath(IVY.id), { vaccine_schema: 'us' });
	await must(ana, 'POST', kidPath(IVY.id, 'vaccinations'), {
		given_on: AT_TWO_MONTHS,
		antigens: ['diphtheria', 'hib'],
		product: '6-in-1 (stub)'
	});
	await must(ana, 'POST', kidPath(IVY.id, 'vaccinations'), {
		given_on: TODAY,
		antigens: ['typhoid'],
		product: 'Travel typhoid',
		note: 'Before the trip'
	});
	await open(page, url(IVY.id));
	await shot(
		page,
		'vaccines',
		'02-us-schedule',
		'United States (stub) at 13 months: due and upcoming visits, Completed collapsed, also available, every season, Other'
	);
	await openDetails(page, 'About this schedule');
	await openDetails(page, 'What is not included');
	await shot(
		page,
		'vaccines',
		'03-us-schedule-about-open',
		'About this schedule and What is not included opened'
	);
	await page
		.getByRole('button', { name: /^Dismiss / })
		.first()
		.click();
	await expect(page.getByText(/^Dismissed: /).first()).toBeVisible();
	await page.waitForLoadState('networkidle');
	await shot(
		page,
		'vaccines',
		'04-line-dismissed',
		'A guardian dismissed a line: the reason and Undo'
	);
	await field(page, 'Due before').fill(TODAY);
	await submit(page.getByRole('button', { name: 'Dismiss lines' }));
	await shot(
		page,
		'vaccines',
		'05-dismiss-before-result',
		'Dismiss older lines: says how many it dismissed'
	);
	// Completed is a region of collapsed visit cards: open them all.
	await page
		.getByRole('region', { name: /^Completed/ })
		.locator('details')
		.evaluateAll((ds) => ds.forEach((d) => ((d as HTMLDetailsElement).open = true)));
	await shot(
		page,
		'vaccines',
		'17-completed-open',
		'Completed visits opened: given and dismissed lines with Undo'
	);
	await must(ana, 'PATCH', kidPath(IVY.id), { vaccine_schema: 'uk' });
	await open(page, url(IVY.id));
	await shot(
		page,
		'vaccines',
		'06-uk-schedule',
		'United Kingdom (stub) for the same kid and doses'
	);
	await storeVaccineSchema(IVY.id, 'retired-schedule');
	await open(page, url(IVY.id));
	await shot(
		page,
		'vaccines',
		'07-retired-schedule',
		'A stored schedule the app no longer ships reads as no schedule'
	);
});

test('caregiver view', async ({ page }) => {
	await signInAs(page, ANA);
	await open(page, url(LEO.id));
	await shot(
		page,
		'vaccines',
		'08-no-schedule-caregiver',
		'No schedule: a caregiver gets a sentence, no link'
	);
	await must(await as(CLEO), 'PATCH', kidPath(LEO.id), { vaccine_schema: 'uk' });
	await open(page, url(LEO.id));
	await shot(
		page,
		'vaccines',
		'09-caregiver-on-schedule',
		'A caregiver on a schedule: no dismiss controls'
	);
});

test('dose forms', async ({ page }) => {
	const ana = await as(ANA);
	await must(ana, 'PATCH', kidPath(IVY.id), { vaccine_schema: 'us' });
	const dose = await must(ana, 'POST', kidPath(IVY.id, 'vaccinations'), {
		given_on: AT_TWO_MONTHS,
		antigens: ['diphtheria', 'hib'],
		lot: 'AB123',
		product: '6-in-1 (stub)',
		note: 'Left thigh, a little fussy after'
	});
	await signInAs(page, ANA);
	const sheet = await openSheet(
		page,
		url(IVY.id),
		(p) => p.getByRole('link', { name: 'Record a vaccination', exact: true }),
		'Record a vaccination'
	);
	await shot(
		page,
		'vaccines',
		'10-new-dose-empty',
		'The Record a vaccination sheet over the Vaccines page: date, product shortcut, vaccines, lot, product, note'
	);
	await field(sheet, 'Date given').fill('');
	await save(sheet);
	await expect(sheet.getByText('Check the fields with a message.')).toBeVisible();
	await shot(
		page,
		'vaccines',
		'11-new-dose-errors',
		'Refused in the sheet: no date and no vaccine ticked; the date is the first problem shown'
	);
	await openSheet(
		page,
		url(IVY.id),
		(p) => p.locator(`a[href^="${url(IVY.id, '/new?antigens=')}"]:visible`).first(),
		'Record a vaccination'
	);
	await shot(
		page,
		'vaccines',
		'12-new-dose-from-record-link',
		'The sheet opened from a visit’s Record button: its outstanding vaccines ticked'
	);
	await openEdit(page, dose.id);
	await shot(
		page,
		'vaccines',
		'13-edit-administration',
		'The Edit vaccination sheet: lot, product and note read back'
	);
	await openDetails(page, /^Delete/);
	await shot(
		page,
		'vaccines',
		'14-edit-delete-confirm',
		'Delete opened inside the Edit vaccination sheet: the confirm step'
	);
	const long = await must(ana, 'POST', kidPath(IVY.id, 'vaccinations'), {
		given_on: TODAY,
		antigens: ['other'],
		product: 'P'.repeat(80),
		note: 'n'.repeat(200)
	});
	await open(page, url(IVY.id));
	await shot(
		page,
		'vaccines',
		'15-long-product-on-page',
		'An 80-character product under Other wraps on the page'
	);
	await openEdit(page, long.id);
	await shot(
		page,
		'vaccines',
		'16-long-note-edit',
		'A 200-character note and an 80-character product stay inside their single-line fields in the Edit vaccination sheet'
	);
});
