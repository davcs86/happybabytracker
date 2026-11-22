import { expect, test, type Page } from '@playwright/test';
import { todayIn } from '../src/lib/format.js';
import { formatDayHeading } from '../src/lib/timeline.js';
import { expectAccessible, expectFieldError, expectNotFoundPage, signIn } from './helpers.js';
import { ANA, CLEO, DAN, LEO, MIA, resetStub, stubAs } from './stub-api.js';

test.beforeEach(async () => {
	await resetStub();
});

type Api = Awaited<ReturnType<typeof stubAs>>;
type Row = Record<string, unknown>;

const MS_PER_MINUTE = 60_000;
const DAY1 = '2026-10-14';
const DAY2 = '2026-10-15';
const RANGE = `from=${DAY1}&to=${DAY2}`;
const PAGE_SIZE = 50;

const seed = async (api: Api, kind: string, body: Row, kid: string = MIA.id) => {
	const r = await api('POST', `/v1/kids/${kid}/${kind}`, body);
	expect(r.status).toBe(201);
	return r.body as Row;
};
const ml = (amount: number) => ({ unit: 'ml', amount });
const timelineUrl = (query = '', kid: string = MIA.id) => `/kids/${kid}/timeline?${query}`;

// Six entries on two days, one of each tracker, with a note on the bottle.
async function seedTwoDays(api: Api) {
	const bottle = await seed(api, 'bottles', {
		at: `${DAY1}T09:05:00Z`,
		contents: ['breast_milk', 'formula'],
		volume: ml(120),
		note: 'woke hungry'
	});
	await seed(api, 'diapers', { at: `${DAY1}T11:30:00Z`, kind: 'wet' });
	await seed(api, 'sleep', { started_at: `${DAY1}T13:05:00Z`, ended_at: `${DAY1}T14:40:00Z` });
	await seed(api, 'nursing', {
		started_at: `${DAY2}T08:00:00Z`,
		ended_at: `${DAY2}T08:25:00Z`,
		left_seconds: 480,
		right_seconds: 300
	});
	await seed(api, 'pumping', {
		started_at: `${DAY2}T10:00:00Z`,
		duration_seconds: 1200,
		volume: ml(90),
		side: 'both'
	});
	await seed(api, 'meals', { at: `${DAY2}T12:15:00Z`, what: 'pear', reaction: 'liked it' });
	return { bottle };
}

const day = (page: Page, date: string) =>
	page.getByRole('region', { name: formatDayHeading(date) });
const entries = (page: Page, date: string) =>
	day(page, date).getByRole('list', { name: 'Entries' }).getByRole('listitem');
const totals = (page: Page, date: string) =>
	day(page, date).getByRole('list', { name: 'Totals' }).getByRole('listitem');
const show = (page: Page) => page.getByRole('button', { name: 'Show', exact: true }).click();
const box = (page: Page, name: string) => page.getByRole('checkbox', { name, exact: true });
const overflows = (page: Page) =>
	page.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth);

test('six entries on two days sit under two headings, newest first, with their day totals', async ({
	page
}) => {
	const api = await stubAs(ANA);
	const { bottle } = await seedTwoDays(api);
	await signIn(page, ANA);
	const response = await page.goto(timelineUrl(RANGE));
	expect(response?.headers()['cache-control']).toBe('no-store');
	await expect(page.getByRole('heading', { name: 'Timeline for Mia', level: 1 })).toBeVisible();

	const headings = page.getByRole('heading', { level: 2 });
	await expect(headings).toHaveText([formatDayHeading(DAY2), formatDayHeading(DAY1)]);
	await expect(entries(page, DAY2)).toHaveText([
		'12:15 Solids: pear · liked it',
		'10:00 Pumping 20 min, 90 ml (both)',
		'08:00 Nursing L 8 min · R 5 min'
	]);
	await expect(entries(page, DAY1)).toHaveText([
		'Sleep 13:05–14:40 (1 h 35 min)',
		'11:30 Diaper wet',
		/^09:05 Bottle 120 ml, breast milk \+ formula\s+woke hungry$/
	]);
	await expect(totals(page, DAY2)).toHaveText(['Nursing 1 · 13 min']);
	await expect(totals(page, DAY1)).toHaveText([
		'Bottles 1 · 120 ml',
		'Diapers 1 (1 wet)',
		'Sleep 1 h 35 min'
	]);
	await expect(page.getByRole('link', { name: /^09:05 Bottle/ })).toHaveAttribute(
		'href',
		`/kids/${MIA.id}/log/bottles/${bottle.id}`
	);
	await expect(page.getByRole('link', { name: 'Log something' })).toHaveAttribute(
		'href',
		`/kids/${MIA.id}`
	);
	await expect(page.getByRole('link', { name: 'Older' })).toHaveCount(0);
	expect(await overflows(page)).toBe(false);
	await expectAccessible(page);
});

test('the filter form shows the kid’s dates and every enabled tracker, ticked', async ({
	page
}) => {
	await signIn(page, ANA);
	await page.goto(timelineUrl(RANGE));
	await expect(page.getByLabel('From', { exact: true })).toHaveValue(DAY1);
	await expect(page.getByLabel('To', { exact: true })).toHaveValue(DAY2);
	for (const name of ['Bottle', 'Nursing', 'Pumping', 'Sleep', 'Diaper', 'Solids']) {
		await expect(box(page, name)).toBeChecked();
	}
	await expect(page.getByText('Nothing logged in this range')).toBeVisible();
	await expectAccessible(page);
});

test('the default range is the kid’s today', async ({ page }) => {
	const api = await stubAs(ANA);
	await seed(api, 'diapers', { kind: 'dirty' });
	await signIn(page, ANA);
	await page.goto(`/kids/${MIA.id}/timeline`);
	const today = todayIn(MIA.timeZone);
	await expect(page.getByLabel('From', { exact: true })).toHaveValue(today);
	await expect(page.getByLabel('To', { exact: true })).toHaveValue(today);
	await expect(entries(page, today)).toHaveText([/Diaper dirty\s*$/]);
});

test('unticking a tracker hides its entries, ticking it again brings them back', async ({
	page
}) => {
	await seedTwoDays(await stubAs(ANA));
	await signIn(page, ANA);
	await page.goto(timelineUrl(RANGE));
	await expect(page.getByText('Bottle 120 ml')).toBeVisible();

	await box(page, 'Bottle').uncheck();
	await show(page);
	await page.waitForURL(/types=/);
	await expect(box(page, 'Bottle')).not.toBeChecked();
	await expect(box(page, 'Sleep')).toBeChecked();
	await expect(page.getByText('Bottle 120 ml')).toHaveCount(0);
	await expect(page.getByText('11:30 Diaper wet')).toBeVisible();
	await expect(totals(page, DAY1)).toHaveText(['Diapers 1 (1 wet)', 'Sleep 1 h 35 min']);

	await box(page, 'Bottle').check();
	await show(page);
	await expect(page.getByText('Bottle 120 ml')).toBeVisible();
	await expect(totals(page, DAY1).first()).toHaveText('Bottles 1 · 120 ml');
});

test('a switched-off module’s entries stay hidden until its box is ticked', async ({ page }) => {
	const api = await stubAs(ANA);
	await seedTwoDays(api);
	expect((await api('PUT', `/v1/kids/${MIA.id}/modules/bottles`, { enabled: false })).status).toBe(
		204
	);
	await signIn(page, ANA);
	await page.goto(timelineUrl(RANGE));
	await expect(box(page, 'Bottle')).not.toBeChecked();
	await expect(box(page, 'Diaper')).toBeChecked();
	await expect(page.getByText('Bottle 120 ml')).toHaveCount(0);
	await expect(page.getByText('11:30 Diaper wet')).toBeVisible();
	await expect(totals(page, DAY1)).toHaveText(['Diapers 1 (1 wet)', 'Sleep 1 h 35 min']);

	await box(page, 'Bottle').check();
	await show(page);
	await expect(page.getByText('09:05 Bottle 120 ml')).toBeVisible();
	await expect(page.getByText('11:30 Diaper wet')).toBeVisible();
	await expect(totals(page, DAY1).first()).toHaveText('Bottles 1 · 120 ml');
});

// One more diaper than a page holds, on one day.
const PAGE_DAY = '2026-10-10';
async function seedMoreThanAPage(api: Api) {
	for (let i = 0; i <= PAGE_SIZE; i++) {
		const at = new Date(Date.parse(`${PAGE_DAY}T06:00:00Z`) + i * MS_PER_MINUTE);
		await seed(api, 'diapers', { at: at.toISOString().replace(/\.\d{3}Z$/, 'Z'), kind: 'wet' });
	}
}
// Resolved targets: the raw href is relative before hydration and absolute after it.
const rowLinks = async (page: Page) =>
	page
		.locator('a[href*="/log/diapers/"]')
		.evaluateAll((a) => a.map((e) => (e as HTMLAnchorElement).href));

async function pageThroughOlder(page: Page) {
	await seedMoreThanAPage(await stubAs(ANA));
	await signIn(page, ANA);
	await page.goto(timelineUrl(`from=${PAGE_DAY}&to=${PAGE_DAY}`));
	const first = await rowLinks(page);
	expect(first).toHaveLength(PAGE_SIZE);
	await expect(page.getByRole('link', { name: 'Back to newest' })).toHaveCount(0);
	await expect(totals(page, PAGE_DAY)).toHaveText(['Diapers 51 (51 wet)']);

	await page.getByRole('link', { name: 'Older' }).click();
	await expect(page).toHaveURL(/cursor=/);
	await expect(page.getByRole('link', { name: 'Older' })).toHaveCount(0);
	const second = await rowLinks(page);
	expect(second).toHaveLength(1);
	expect(first).not.toContain(second[0]);
	await expect(page.getByLabel('From', { exact: true })).toHaveValue(PAGE_DAY);
	await expect(entries(page, PAGE_DAY).first()).toHaveText('06:00 Diaper wet');

	await page.getByRole('link', { name: 'Back to newest' }).click();
	await expect(page).not.toHaveURL(/cursor=/);
	expect(await rowLinks(page)).toEqual(first);
}

test('Older reaches the second page without repeating an entry, Back to newest returns', async ({
	page
}) => {
	await pageThroughOlder(page);
});

test('Older carries a narrowed filter to the next page', async ({ page }) => {
	const api = await stubAs(ANA);
	await seedMoreThanAPage(api);
	await seed(api, 'bottles', {
		at: `${PAGE_DAY}T05:00:00Z`,
		contents: ['formula'],
		volume: ml(60)
	});
	await signIn(page, ANA);
	await page.goto(timelineUrl(`from=${PAGE_DAY}&to=${PAGE_DAY}`));
	await box(page, 'Bottle').uncheck();
	await show(page);
	await page.waitForURL(/types=/);
	await page.getByRole('link', { name: 'Older' }).click();
	await expect(page).toHaveURL(/types=diaper/);
	await expect(page).not.toHaveURL(/types=bottle/);
	await expect(box(page, 'Bottle')).not.toBeChecked();
	expect(await rowLinks(page)).toHaveLength(1);
	await expect(page.getByText('Bottle 60 ml')).toHaveCount(0);
});

test('an invalid date, a reversed range and a long range show a message on the field', async ({
	page
}) => {
	await seedTwoDays(await stubAs(ANA));
	await signIn(page, ANA);
	await page.goto(timelineUrl('from=garbage&to=2026-10-15'));
	await expect(page.getByRole('heading', { name: 'Timeline for Mia', level: 1 })).toBeVisible();
	await expectFieldError(page.getByLabel('From', { exact: true }), 'Enter a valid date');
	await expect(page.getByLabel('To', { exact: true })).not.toHaveAttribute('aria-invalid', 'true');
	await expect(page.getByRole('list', { name: 'Entries' })).toHaveCount(0);
	await expectAccessible(page);

	await page.goto(timelineUrl('from=2026-10-15&to=2026-10-14'));
	await expectFieldError(
		page.getByLabel('To', { exact: true }),
		'The end date is before the start date'
	);
	await expectAccessible(page);

	await page.goto(timelineUrl('from=2026-01-01&to=2026-06-01'));
	await expectFieldError(page.getByLabel('To', { exact: true }), 'Pick at most 93 days');

	await page.goto(timelineUrl('from=2026-10-13&to=2026-10-13&types=nope'));
	await expect(page.getByText('Nothing logged in this range')).toBeVisible();
});

test('an invalid cursor shows a message and a way back, not an error page', async ({ page }) => {
	await seedTwoDays(await stubAs(ANA));
	await signIn(page, ANA);
	await page.goto(timelineUrl(`${RANGE}&cursor=bogus`));
	await expect(page.getByRole('alert').filter({ hasText: 'no longer works' })).toBeVisible();
	await expectAccessible(page);
	await page.getByRole('link', { name: 'Back to newest' }).click();
	await expect(page).not.toHaveURL(/cursor=/);
	await expect(entries(page, DAY2)).toHaveCount(3);
});

test('a row opens its edit page, and saving returns to the timeline with the change shown', async ({
	page
}) => {
	const api = await stubAs(ANA);
	const diaper = await seed(api, 'diapers', { at: `${DAY1}T11:30:00Z`, kind: 'wet' });
	await signIn(page, ANA);
	await page.goto(timelineUrl(`from=${DAY1}&to=${DAY1}`));
	await page.getByRole('link', { name: '11:30 Diaper wet' }).click();
	await expect(page).toHaveURL(`/kids/${MIA.id}/log/diapers/${diaper.id}`);
	await page.getByRole('radio', { name: 'Both' }).check();
	await page.getByRole('button', { name: 'Save', exact: true }).click();
	await expect(page).toHaveURL(timelineUrl(`from=${DAY1}&to=${DAY1}`));
	await expect(entries(page, DAY1)).toHaveText(['11:30 Diaper both']);
	await expect(totals(page, DAY1)).toHaveText(['Diapers 1 (1 both)']);
});

test('a Madrid day of 23 hours holds an entry at 00:30 and one at 23:30 under one heading', async ({
	page
}) => {
	const api = await stubAs(ANA);
	const madrid = (at: string, kind: string) => seed(api, 'diapers', { at, kind }, LEO.id);
	await madrid('2026-03-28T22:30:00Z', 'dry');
	await madrid('2026-03-28T23:30:00Z', 'wet');
	await madrid('2026-03-29T21:30:00Z', 'dirty');
	await madrid('2026-03-29T22:00:00Z', 'both');
	await signIn(page, ANA);

	await page.goto(timelineUrl('from=2026-03-29&to=2026-03-29', LEO.id));
	await expect(page.getByLabel('Diaper', { exact: true })).not.toBeChecked();
	await expect(page.getByText('Nothing logged in this range')).toBeVisible();

	await page.goto(timelineUrl('from=2026-03-29&to=2026-03-29&types=diaper', LEO.id));
	await expect(page.getByRole('heading', { level: 2 })).toHaveText([
		formatDayHeading('2026-03-29')
	]);
	await expect(entries(page, '2026-03-29')).toHaveText(['23:30 Diaper dirty', '00:30 Diaper wet']);
	await expect(totals(page, '2026-03-29')).toHaveText(['Diapers 2 (1 wet, 1 dirty)']);

	await page.goto(timelineUrl('from=2026-03-28&to=2026-03-30&types=diaper', LEO.id));
	await expect(page.getByRole('heading', { level: 2 })).toHaveText([
		formatDayHeading('2026-03-30'),
		formatDayHeading('2026-03-29'),
		formatDayHeading('2026-03-28')
	]);
	await expect(entries(page, '2026-03-28')).toHaveText(['23:30 Diaper dry']);
	await expect(entries(page, '2026-03-30')).toHaveText(['00:00 Diaper both']);
});

test('a parent without a grant and an unknown kid get the same 404 page', async ({ page }) => {
	await signIn(page, DAN);
	const unknown = await expectNotFoundPage(page, `/kids/k-unknown/timeline?${RANGE}`);
	expect(await expectNotFoundPage(page, timelineUrl(RANGE))).toBe(unknown);
});

test('a caregiver sees the kid’s timeline', async ({ page }) => {
	await seed(await stubAs(CLEO), 'diapers', { at: `${DAY1}T11:30:00Z`, kind: 'wet' }, LEO.id);
	await signIn(page, ANA);
	await page.goto(timelineUrl(`from=${DAY1}&to=${DAY1}&types=diaper`, LEO.id));
	await expect(entries(page, DAY1)).toHaveText(['13:30 Diaper wet']);
});

test.describe('without JavaScript', () => {
	test.use({ javaScriptEnabled: false });

	test('Older and Back to newest are plain links', async ({ page }) => {
		await pageThroughOlder(page);
	});

	test('the filter form is a plain GET', async ({ page }) => {
		await seedTwoDays(await stubAs(ANA));
		await signIn(page, ANA);
		await page.goto(timelineUrl(RANGE));
		await box(page, 'Sleep').uncheck();
		await box(page, 'Diaper').uncheck();
		await page.getByLabel('From', { exact: true }).fill(DAY2);
		await show(page);
		await expect(page).toHaveURL(/from=2026-10-15&to=2026-10-15&types=bottle/);
		await expect(entries(page, DAY2)).toHaveCount(3);
		await expect(box(page, 'Sleep')).not.toBeChecked();
	});
});

test('a sleep that ends the next day marks its end', async ({ page }) => {
	const api = await stubAs(ANA);
	await seed(api, 'sleep', { started_at: `${DAY1}T22:00:00Z`, ended_at: `${DAY2}T06:00:00Z` });
	await signIn(page, ANA);
	await page.goto(timelineUrl(RANGE));
	await expect(entries(page, DAY1)).toHaveText(['Sleep 22:00–06:00 +1 day (8 h)']);
});

test('a long unbroken meal and note wrap on a phone', async ({ page }, testInfo) => {
	test.skip(testInfo.project.name !== 'phone', 'the overflow is a phone-width check');
	const api = await stubAs(ANA);
	const token = 'w'.repeat(200);
	await seed(api, 'meals', { at: `${DAY1}T12:00:00Z`, what: token, note: token });
	await signIn(page, ANA);
	await page.goto(timelineUrl(RANGE));
	await expect(entries(page, DAY1)).toHaveCount(1);
	expect(await overflows(page)).toBe(false);
});
