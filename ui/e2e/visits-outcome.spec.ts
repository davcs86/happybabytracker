import { expect, test, type Locator, type Page } from '@playwright/test';
import { VACCINE_LOT_MAX_LEN } from '../src/lib/constants.js';
import { expectAccessible, expectFieldError, openSheet, signIn, submit } from './helpers.js';
import { ANA, MIA, receivedRequests, resetStub, stubAs } from './stub-api.js';

// After the visit: the outcome sheet and its page. Visits are made through the stub API on the real clock; every
// save waits for its response with submit().

test.beforeEach(async () => {
	await resetStub();
});

type Row = Record<string, unknown>;
type Api = Awaited<ReturnType<typeof stubAs>>;

const MS_PER_DAY = 24 * 60 * 60_000;
const days = (n: number) =>
	new Date(Math.floor(Date.now() / 60_000) * 60_000 + n * MS_PER_DAY)
		.toISOString()
		.replace(/\.\d{3}Z$/, 'Z');

const visitUrl = (id: unknown) => `/kids/${MIA.id}/visits/${id}`;
const outcomeUrl = (id: unknown) => `${visitUrl(id)}/outcome`;
const VISITS = `/v1/kids/${MIA.id}/visits`;

async function seedVisit(api: Api, body: Row = {}) {
	const r = await api('POST', VISITS, { kind: 'well_child', starts_at: days(-2), ...body });
	expect(r.status).toBe(201);
	return r.body as Row;
}
const visitOf = async (api: Api, id: unknown) =>
	(await api('GET', `${VISITS}/${id}`)).body as Row & {
		linked: { growth: Row[]; vaccinations: Row[]; medications: Row[] };
	};
const outcomePosts = async (id: unknown) =>
	(await receivedRequests()).filter(
		(r) => r.method === 'POST' && r.path === `${VISITS}/${id}/outcome`
	);

const field = (where: Page | Locator, label: string) => where.getByLabel(label, { exact: true });
const box = (where: Page | Locator, label: string) =>
	where.getByRole('checkbox', { name: label, exact: true });
const vaccine = (where: Page | Locator, n: number) =>
	where.getByRole('group', { name: `Vaccine ${n}`, exact: true });
const medication = (where: Page | Locator, n: number) =>
	where.getByRole('group', { name: `Medication ${n}`, exact: true });
const save = (where: Page | Locator) =>
	submit(where.getByRole('button', { name: 'Save', exact: true }));
const recordLink = (page: Page) => page.getByRole('link', { name: 'Record the visit' });
const toast = (page: Page, text: string) =>
	page.locator('[data-sonner-toast]').filter({ hasText: text });
const STALE = 'Someone changed this visit on another phone. It’s shown now; your entries are kept.';

test('record growth, two vaccines and a course in the sheet; the module pages show them', async ({
	page
}) => {
	const api = await stubAs(ANA);
	expect((await api('PATCH', `/v1/kids/${MIA.id}`, { vaccine_schema: 'us' })).status).toBe(200);
	const v = await seedVisit(api);
	await signIn(page, ANA);
	const sheet = await openSheet(page, visitUrl(v.id), recordLink, 'After the visit');
	await expect(page).toHaveURL(outcomeUrl(v.id));
	await expect(field(sheet, 'Notes from the visit')).toHaveValue('');
	for (const label of ['Add a growth measurement', 'Add vaccines given', 'Add a medication']) {
		await expect(box(sheet, label)).not.toBeChecked();
	}
	await expectAccessible(page);

	await field(sheet, 'Notes from the visit').fill('Healthy. Next check at 6 months.');
	await box(sheet, 'Add a growth measurement').check();
	const day = String(v.starts_at).slice(0, 10);
	await expect(field(sheet, 'Date')).toHaveValue(day);
	await field(sheet, 'Kilograms').fill('6.4');

	await box(sheet, 'Add vaccines given').check();
	await expect(vaccine(sheet, 2)).toHaveCount(0);
	await field(vaccine(sheet, 1), 'Diphtheria').check();
	await sheet.getByRole('button', { name: 'Add another vaccine' }).click();
	await expect(vaccine(sheet, 2)).toBeFocused();
	await expect(field(vaccine(sheet, 2), 'Date given')).toHaveValue(day);
	await field(vaccine(sheet, 2), 'Measles').check();
	await field(vaccine(sheet, 2), 'Lot number').fill('AB12');

	await box(sheet, 'Add a medication').check();
	await field(medication(sheet, 1), 'Name').fill('Amoxicillin');
	await field(medication(sheet, 1), 'Dose').fill('5 ml');
	await expectAccessible(page);

	await save(sheet);
	await expect(sheet).toBeHidden();
	await expect(toast(page, 'Visit recorded')).toBeVisible();
	await expect(page).toHaveURL(visitUrl(v.id));
	const posts = await outcomePosts(v.id);
	expect(posts).toHaveLength(1);
	expect(JSON.parse(posts[0].body ?? 'null')).toEqual({
		revision: v.revision,
		outcome_notes: 'Healthy. Next check at 6 months.',
		growth: { measured_on: day, weight: { unit: 'kg', amount: 6400 }, note: null },
		vaccinations: [
			{ given_on: day, antigens: ['diphtheria'] },
			{ given_on: day, antigens: ['measles'], lot: 'AB12' }
		],
		medications: [
			{
				name: 'Amoxicillin',
				dose: '5 ml',
				schedule: { kind: 'none' },
				start_at: v.starts_at
			}
		]
	});

	const main = page.getByRole('main');
	await expect(main.getByText('Healthy. Next check at 6 months.')).toBeVisible();
	await expect(main.getByRole('link', { name: /^Growth/ })).toHaveCount(1);
	await expect(main.getByRole('link', { name: /^Vaccines/ })).toHaveCount(2);
	await expect(main.getByRole('link', { name: /^Medication\s*Amoxicillin/ })).toHaveCount(1);
	await expect(page.getByRole('link', { name: 'Add more from the visit' })).toBeVisible();

	const { linked } = await visitOf(api, v.id);
	await page.goto(`/kids/${MIA.id}/growth`);
	await expect(page.locator(`a[href="/kids/${MIA.id}/growth/${linked.growth[0].id}"]`)).toHaveCount(
		1
	);
	await page.goto(`/kids/${MIA.id}/vaccines`);
	for (const a of linked.vaccinations) {
		await expect(page.locator(`a[href="/kids/${MIA.id}/vaccines/${a.id}"]`).first()).toBeVisible();
	}
	await page.goto(`/kids/${MIA.id}/medication`);
	await expect(
		page.locator(`a[href="/kids/${MIA.id}/medication/${linked.medications[0].id}"]`)
	).toHaveCount(1);
});

test('a refused lot in vaccine 2 is named in its block, and nothing is sent', async ({ page }) => {
	const api = await stubAs(ANA);
	const v = await seedVisit(api);
	await signIn(page, ANA);
	const sheet = await openSheet(page, visitUrl(v.id), recordLink, 'After the visit');
	await box(sheet, 'Add vaccines given').check();
	await field(vaccine(sheet, 1), 'Measles').check();
	await sheet.getByRole('button', { name: 'Add another vaccine' }).click();
	await field(vaccine(sheet, 2), 'Measles').check();
	await field(vaccine(sheet, 2), 'Lot number').fill('x'.repeat(VACCINE_LOT_MAX_LEN + 1));
	await save(sheet);
	await expect(sheet).toBeVisible();
	await expect(sheet.getByText('Check the fields with a message.')).toBeVisible();
	await expectFieldError(
		field(vaccine(sheet, 2), 'Lot number'),
		`Vaccine 2: Enter a lot number of up to ${VACCINE_LOT_MAX_LEN} characters`
	);
	await expect(field(vaccine(sheet, 1), 'Lot number')).not.toHaveAttribute('aria-invalid');
	await expect(field(vaccine(sheet, 2), 'Measles')).toBeChecked();
	expect(await outcomePosts(v.id)).toHaveLength(0);
	await expectAccessible(page);
});

test('the add buttons stop at the caps', async ({ page }) => {
	const api = await stubAs(ANA);
	const v = await seedVisit(api);
	await signIn(page, ANA);
	const sheet = await openSheet(page, visitUrl(v.id), recordLink, 'After the visit');
	await box(sheet, 'Add a medication').check();
	const more = sheet.getByRole('button', { name: 'Add another medication' });
	for (let n = 2; n <= 5; n++) {
		await more.click();
		await expect(medication(sheet, n)).toBeFocused();
	}
	await expect(more).toHaveCount(0);
});

test('a section whose module is off is not offered', async ({ page }) => {
	const api = await stubAs(ANA);
	expect(
		(await api('PUT', `/v1/kids/${MIA.id}/modules/medication`, { enabled: false })).status
	).toBe(200);
	const v = await seedVisit(api);
	await signIn(page, ANA);
	await page.goto(outcomeUrl(v.id));
	await expect(box(page, 'Add a growth measurement')).toBeVisible();
	await expect(box(page, 'Add vaccines given')).toBeVisible();
	await expect(box(page, 'Add a medication')).toHaveCount(0);
	await expect(medication(page, 1)).toHaveCount(0);
});

test('an upcoming visit cannot be recorded yet', async ({ page }) => {
	const api = await stubAs(ANA);
	const v = await seedVisit(api, { starts_at: days(2) });
	await signIn(page, ANA);
	await page.goto(outcomeUrl(v.id));
	await expect(page.getByRole('heading', { level: 1, name: 'After the visit' })).toBeVisible();
	await expect(page.getByRole('alert')).toHaveText('This visit hasn’t happened yet.');
	await expect(page.getByRole('button', { name: 'Save' })).toHaveCount(0);
	await expect(page.getByRole('link', { name: 'Cancel', exact: true })).toBeVisible();
	await expectAccessible(page);
});

test.describe('without JavaScript', () => {
	test.use({ javaScriptEnabled: false });

	test('the page offers one vaccine and one medication block; saving again adds more', async ({
		page
	}) => {
		const api = await stubAs(ANA);
		const v = await seedVisit(api, { kind: 'sick' });
		await signIn(page, ANA);
		await page.goto(visitUrl(v.id));
		await recordLink(page).click();
		await expect(page).toHaveURL(outcomeUrl(v.id));
		await expect(page.getByRole('heading', { level: 1, name: 'After the visit' })).toBeVisible();
		await expect(vaccine(page, 1)).toHaveCount(1);
		await expect(vaccine(page, 2)).toHaveCount(0);
		await expect(medication(page, 1)).toHaveCount(1);
		await expect(page.getByRole('button', { name: /^Add another/ })).toHaveCount(0);
		await expectAccessible(page);

		await box(page, 'Add vaccines given').check();
		await field(vaccine(page, 1), 'Measles').check();
		await box(page, 'Add a medication').check();
		await field(medication(page, 1), 'Name').fill('Amoxicillin');
		await field(medication(page, 1), 'Dose').fill('5 ml');
		await save(page);
		await expect(page).toHaveURL(visitUrl(v.id));
		await expect(
			page.getByRole('heading', { level: 2, name: 'Recorded at this visit' })
		).toBeVisible();

		await page.getByRole('link', { name: 'Add more from the visit' }).click();
		await expect(page).toHaveURL(outcomeUrl(v.id));
		await box(page, 'Add vaccines given').check();
		await field(vaccine(page, 1), 'Diphtheria').check();
		await save(page);
		await expect(page).toHaveURL(visitUrl(v.id));
		const { linked } = await visitOf(api, v.id);
		expect(linked.vaccinations.map((a) => a.antigens)).toEqual([['measles'], ['diphtheria']]);
		expect(linked.medications).toHaveLength(1);
	});

	test('a second submit of the same form is stale, writes nothing and keeps what was typed', async ({
		page,
		context
	}) => {
		const api = await stubAs(ANA);
		const v = await seedVisit(api);
		await signIn(page, ANA);
		const other = await context.newPage();
		for (const p of [page, other]) {
			await p.goto(outcomeUrl(v.id));
			await field(p, 'Notes from the visit').fill('Healthy.');
			await box(p, 'Add a growth measurement').check();
			await field(p, 'Kilograms').fill('6.4');
		}
		await save(page);
		await expect(page).toHaveURL(visitUrl(v.id));

		await save(other);
		await expect(other).toHaveURL(outcomeUrl(v.id));
		await expect(other.getByRole('alert')).toHaveText(STALE);
		await expect(field(other, 'Notes from the visit')).toHaveValue('Healthy.');
		await expect(box(other, 'Add a growth measurement')).toBeChecked();
		await expect(field(other, 'Kilograms')).toHaveValue('6.4');
		const after = await visitOf(api, v.id);
		expect(after.linked.growth).toHaveLength(1);
		expect(after.revision).toBe(Number(v.revision) + 1);
		await expect(other.locator('input[name="revision"]')).toHaveValue(String(after.revision));
		await expectAccessible(other);
	});
});
