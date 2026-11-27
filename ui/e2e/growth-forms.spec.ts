import { expect, test, type Page } from '@playwright/test';
import { formatAge, todayIn } from '../src/lib/format.js';
import {
	expectAccessible,
	expectFieldError,
	expectNotFoundPage,
	signIn,
	untilHydrated
} from './helpers.js';
import { ANA, CLEO, DAN, IVY, LEO, MIA, receivedRequests, resetStub, stubAs } from './stub-api.js';

test.beforeEach(async () => {
	await resetStub();
});

type Api = Awaited<ReturnType<typeof stubAs>>;
type Row = Record<string, unknown>;

const MS_PER_DAY = 86_400_000;
const addDays = (date: string, n: number) =>
	new Date(Date.parse(`${date}T00:00:00Z`) + n * MS_PER_DAY).toISOString().slice(0, 10);
const TODAY = todayIn(IVY.timeZone);
const CDC = 'CDC 2000 Growth Charts';

const newUrl = (kid: string) => `/kids/${kid}/growth/new`;
const entryUrl = (kid: string, id: unknown) => `/kids/${kid}/growth/${id}`;
const apiPath = (kid: string, id?: unknown) => `/v1/kids/${kid}/growth${id ? `/${id}` : ''}`;
const field = (page: Page, label: string) => page.getByLabel(label, { exact: true });
const save = (page: Page) => page.getByRole('button', { name: 'Save', exact: true }).click();
const rows = (page: Page) => page.getByRole('list', { name: 'Measurements' }).getByRole('listitem');
const quickLog = (page: Page) => page.getByRole('region', { name: 'Quick log' });

const stored = async (api: Api, kid = IVY.id) =>
	((await api('GET', apiPath(kid))).body.growth ?? []) as Row[];
const seed = async (api: Api, body: Row, kid = IVY.id) => {
	const r = await api('POST', apiPath(kid), body);
	expect(r.status).toBe(201);
	return r.body as Row;
};
const written = async (method: string, path: string) => {
	const writes = (await receivedRequests()).filter((r) => r.method === method && r.path === path);
	return JSON.parse(writes.at(-1)?.body ?? 'null') as Row;
};

// The save lands on the growth page that exists, on the chart of the first measure entered.
async function onGrowth(page: Page, kid: string, measure: string) {
	await page.waitForURL(
		(u) => u.pathname === `/kids/${kid}/growth` && u.search === `?measure=${measure}`
	);
	await expect(page.getByRole('heading', { name: 'Growth', level: 1 })).toBeVisible();
}

test('create with kilograms and centimetres lands on the growth page with the row', async ({
	page
}) => {
	const api = await stubAs(ANA);
	await signIn(page, ANA);
	await page.goto(newUrl(IVY.id));
	await expect(page.getByRole('heading', { name: 'Add a measurement', level: 1 })).toBeVisible();
	await expect(field(page, 'Date')).toHaveValue(TODAY);
	await expect(field(page, 'Weight unit')).toHaveValue('kg');
	await expect(field(page, 'Length unit')).toHaveValue('cm');
	await expect(field(page, 'Head circumference unit')).toHaveValue('cm');
	await expect(field(page, 'Pounds')).toBeHidden();
	await expectAccessible(page);
	await field(page, 'Kilograms').fill('9.5');
	await field(page, 'Length or height').fill('76');
	await field(page, 'Note').fill('at the clinic');
	await save(page);
	await onGrowth(page, IVY.id, 'weight');
	await expect(rows(page)).toHaveCount(1);
	await expect(rows(page).first()).toContainText('Weight 9.5 kg');
	await expect(rows(page).first()).toContainText('Length 76 cm');
	await expect(rows(page).first()).toContainText('at the clinic');
	expect((await stored(api))[0]).toMatchObject({
		measured_on: TODAY,
		weight: { grams: 9500, typed_unit: 'kg' },
		length: { mm: 760, typed_unit: 'cm' },
		head: null
	});

	await page.goto(newUrl(IVY.id));
	await field(page, 'Head circumference').fill('45');
	await save(page);
	await onGrowth(page, IVY.id, 'head');
});

test('a decimal comma is read as the decimal point', async ({ page }) => {
	const api = await stubAs(ANA);
	await signIn(page, ANA);
	await page.goto(newUrl(IVY.id));
	await field(page, 'Kilograms').fill('9,25');
	await field(page, 'Length or height').fill('75,5');
	await save(page);
	await onGrowth(page, IVY.id, 'weight');
	await expect(rows(page).first()).toContainText('Weight 9.25 kg');
	expect((await stored(api))[0]).toMatchObject({ weight: { grams: 9250 }, length: { mm: 755 } });
});

test('an imperial parent enters pounds and ounces and sees them back on the edit page', async ({
	page
}) => {
	const api = await stubAs(ANA);
	expect((await api('PATCH', '/v1/me', { units: { weight: 'lb', length: 'in' } })).status).toBe(
		200
	);
	await signIn(page, ANA);
	await page.goto(newUrl(IVY.id));
	await expect(field(page, 'Weight unit')).toHaveValue('lb');
	await expect(field(page, 'Length unit')).toHaveValue('in');
	await expect(field(page, 'Head circumference unit')).toHaveValue('in');
	await expect(field(page, 'Kilograms')).toBeHidden();
	await field(page, 'Pounds').fill('8');
	await field(page, 'Ounces').fill('3.5');
	await save(page);
	await onGrowth(page, IVY.id, 'weight');
	await expect(rows(page).first()).toContainText('Weight 8 lb 3.5 oz');
	const [row] = await stored(api);
	expect(row.weight).toMatchObject({ oz_hundredths: 13150, typed_unit: 'lb' });

	await page.goto(entryUrl(IVY.id, row.id));
	await expect(page.getByRole('heading', { name: 'Edit measurement', level: 1 })).toBeVisible();
	await expect(field(page, 'Weight unit')).toHaveValue('lb');
	await expect(field(page, 'Pounds')).toHaveValue('8');
	await expect(field(page, 'Ounces')).toHaveValue('3.5');
	await expect(field(page, 'Kilograms')).toBeHidden();
	await expectAccessible(page);
});

test('editing only the note sends no measure; clearing the head sends null', async ({ page }) => {
	const api = await stubAs(ANA);
	const row = await seed(api, {
		measured_on: addDays(TODAY, -3),
		weight: { unit: 'kg', amount: 9200 },
		length: { unit: 'cm', amount: 745 },
		head: { unit: 'cm', amount: 452 },
		note: 'seeded'
	});
	const path = apiPath(IVY.id, row.id);
	await signIn(page, ANA);
	await page.goto(entryUrl(IVY.id, row.id));
	await untilHydrated(field(page, 'Note'));
	await expect(field(page, 'Date')).toHaveValue(addDays(TODAY, -3));
	await expect(field(page, 'Kilograms')).toHaveValue('9.2');
	await expect(field(page, 'Length or height')).toHaveValue('74.5');
	await expect(field(page, 'Head circumference')).toHaveValue('45.2');
	await expect(field(page, 'Note')).toHaveValue('seeded');
	await expect(page.locator('input[type="hidden"][name$="_original"]')).toHaveCount(9);
	await field(page, 'Note').fill('only the note');
	await save(page);
	await onGrowth(page, IVY.id, 'weight');
	expect(await written('PATCH', path)).toEqual({ note: 'only the note' });

	await page.goto(entryUrl(IVY.id, row.id));
	await untilHydrated(field(page, 'Head circumference'));
	await field(page, 'Head circumference').fill('');
	await save(page);
	await onGrowth(page, IVY.id, 'weight');
	expect(await written('PATCH', path)).toEqual({ head: null, note: 'only the note' });
	expect((await stored(api))[0]).toMatchObject({ head: null, weight: { grams: 9200 } });
});

test('flipping a unit resends the value in the new unit; a blank text after a flip clears', async ({
	page
}) => {
	const api = await stubAs(ANA);
	const row = await seed(api, {
		measured_on: TODAY,
		weight: { unit: 'kg', amount: 9200 },
		length: { unit: 'cm', amount: 495 },
		head: { unit: 'cm', amount: 452 }
	});
	const ozColumn = (row.weight as Row).oz_hundredths;
	const path = apiPath(IVY.id, row.id);
	await signIn(page, ANA);
	await page.goto(entryUrl(IVY.id, row.id));
	await untilHydrated(field(page, 'Note'));
	await field(page, 'Weight unit').selectOption('lb');
	await expect(field(page, 'Pounds')).toBeVisible();
	await expect(field(page, 'Pounds')).not.toHaveValue('');
	await field(page, 'Length unit').selectOption('in');
	await save(page);
	await onGrowth(page, IVY.id, 'weight');
	expect(await written('PATCH', path)).toEqual({
		weight: { unit: 'lb', amount: ozColumn },
		length: { unit: 'in', amount: 4950 },
		note: null
	});

	await page.goto(entryUrl(IVY.id, row.id));
	await untilHydrated(field(page, 'Head circumference'));
	await field(page, 'Head circumference unit').selectOption('in');
	await field(page, 'Head circumference').fill('');
	await save(page);
	await onGrowth(page, IVY.id, 'weight');
	expect(await written('PATCH', path)).toEqual({ head: null, note: null });
});

test('a future date, a date before birth and an empty form show field messages, values kept', async ({
	page
}) => {
	await signIn(page, ANA);
	await page.goto(newUrl(IVY.id));
	await field(page, 'Date').fill(addDays(TODAY, 1));
	await field(page, 'Kilograms').fill('9');
	await field(page, 'Note').fill('kept');
	await save(page);
	const dateMessage = 'Enter a date that is not in the future and not before the birth date';
	await expectFieldError(field(page, 'Date'), dateMessage);
	await expect(field(page, 'Kilograms')).toHaveValue('9');
	await expect(field(page, 'Note')).toHaveValue('kept');
	await expectAccessible(page);

	await field(page, 'Date').fill(addDays(IVY.birthDate, -1));
	await save(page);
	await expectFieldError(field(page, 'Date'), dateMessage);
	await expect(field(page, 'Date')).toHaveValue(addDays(IVY.birthDate, -1));
	await expect(field(page, 'Kilograms')).toHaveValue('9');

	await page.goto(newUrl(IVY.id));
	await save(page);
	const atLeastOne = 'Enter at least one of weight, length or head circumference';
	await expectFieldError(field(page, 'Kilograms'), atLeastOne);
	await expectFieldError(field(page, 'Length or height'), atLeastOne);
	await expectFieldError(field(page, 'Head circumference'), atLeastOne);
	await expect(field(page, 'Date')).toHaveValue(TODAY);
	await expectAccessible(page);

	await field(page, 'Kilograms').fill('9.1234');
	await save(page);
	await expectFieldError(field(page, 'Kilograms'), 'Enter a weight');
	await expect(field(page, 'Kilograms')).toHaveValue('9.1234');
});

test('delete asks first, then the row is gone', async ({ page }) => {
	const api = await stubAs(ANA);
	const row = await seed(api, { measured_on: TODAY, length: { unit: 'cm', amount: 760 } });
	await signIn(page, ANA);
	await page.goto(entryUrl(IVY.id, row.id));
	await expect(page.getByRole('button', { name: 'Yes, delete this measurement' })).toBeHidden();
	await page.getByText('Delete this measurement', { exact: true }).click();
	await expectAccessible(page);
	await page.getByRole('button', { name: 'Yes, delete this measurement' }).click();
	await onGrowth(page, IVY.id, 'length');
	await expect(page.getByText('No measurements yet')).toBeVisible();
	expect(await stored(api)).toEqual([]);
});

test('a caregiver can add and edit', async ({ page }) => {
	const api = await stubAs(ANA);
	const leoToday = todayIn(LEO.timeZone);
	await signIn(page, ANA);
	await page.goto(newUrl(LEO.id));
	await expect(field(page, 'Date')).toHaveValue(leoToday);
	await field(page, 'Kilograms').fill('13.4');
	await save(page);
	await onGrowth(page, LEO.id, 'weight');
	const added = (await stored(api, LEO.id)).find((r) => r.measured_on === leoToday);
	expect(added).toMatchObject({ weight: { grams: 13400 } });

	await page.goto(entryUrl(LEO.id, added?.id));
	await untilHydrated(field(page, 'Note'));
	await field(page, 'Note').fill('by the caregiver');
	await save(page);
	await onGrowth(page, LEO.id, 'weight');
	expect(await written('PATCH', apiPath(LEO.id, added?.id))).toEqual({ note: 'by the caregiver' });

	await page.goto(entryUrl(LEO.id, added?.id));
	await page.getByText('Delete this measurement', { exact: true }).click();
	await page.getByRole('button', { name: 'Yes, delete this measurement' }).click();
	await onGrowth(page, LEO.id, 'weight');
	expect((await stored(api, LEO.id)).some((r) => r.id === added?.id)).toBe(false);
});

test('no grant, an unknown entry and another kid’s entry are the same 404 page', async ({
	page
}) => {
	const api = await stubAs(ANA);
	const row = await seed(api, { measured_on: TODAY, weight: { unit: 'kg', amount: 9000 } });
	await signIn(page, ANA);
	const unknown = await expectNotFoundPage(page, entryUrl(IVY.id, 'g-unknown'));
	const otherKid = await expectNotFoundPage(page, entryUrl(MIA.id, row.id));
	expect(otherKid).toBe(unknown);
	expect(await expectNotFoundPage(page, newUrl('k-nobody'))).toBe(unknown);

	await page.context().clearCookies();
	await signIn(page, DAN);
	const noGrantNew = await expectNotFoundPage(page, newUrl(IVY.id));
	expect(await expectNotFoundPage(page, entryUrl(IVY.id, row.id))).toBe(noGrantNew);
	expect(await expectNotFoundPage(page, entryUrl(IVY.id, 'g-unknown'))).toBe(noGrantNew);
});

test('kid settings offer who and cdc; saving cdc changes the growth page’s source and curves', async ({
	page
}) => {
	const api = await stubAs(ANA);
	// The chart appears with the first measurement.
	await seed(api, { measured_on: TODAY, weight: { unit: 'kg', amount: 9200 } });
	await signIn(page, ANA);
	await page.goto(`/kids/${IVY.id}/settings`);
	await untilHydrated(field(page, 'Name'));
	const select = field(page, 'Growth chart source');
	await expect(select).toHaveValue('who');
	await expect(select.locator('option')).toHaveCount(2);
	expect(
		await select.locator('option').evaluateAll((o) => o.map((x) => (x as HTMLOptionElement).value))
	).toEqual(['who', 'cdc']);
	await expect(select).toHaveAccessibleDescription(/Months covered: weight 0–240/);
	await expectAccessible(page);
	await select.selectOption('cdc');
	await save(page);
	await expect(page.getByRole('status')).toHaveText('Saved');
	expect(await written('PATCH', `/v1/kids/${IVY.id}`)).toMatchObject({
		name: IVY.name,
		birth_date: IVY.birthDate,
		time_zone: IVY.timeZone,
		growth_source: 'cdc'
	});

	await page.goto(`/kids/${IVY.id}/growth`);
	await expect(page.getByTestId('growth-source')).toContainText(CDC);
	await expect(page.locator('svg[role="img"] [data-line]')).toHaveCount(9);
});

test('a new child defaults to who', async ({ page }) => {
	const api = await stubAs(ANA);
	await signIn(page, ANA);
	await page.goto('/kids/new');
	await expect(field(page, 'Growth chart source')).toHaveValue('who');
	await page.getByLabel('Name', { exact: true }).fill('Noa');
	await page.getByLabel('Birth date').fill(addDays(TODAY, -30));
	await page.getByRole('radio', { name: 'Female', exact: true }).check();
	await page.getByRole('button', { name: 'Add child' }).click();
	await expect(page.getByRole('heading', { name: 'Noa', level: 1 })).toBeVisible();
	const kids = (await api('GET', '/v1/kids')).body.kids as Row[];
	expect(kids.find((k) => k.name === 'Noa')).toMatchObject({ growth_source: 'who' });
});

test('a corrected kid shows the corrected age beside the age on the home and the growth page', async ({
	page
}) => {
	const api = await stubAs(ANA);
	const born = addDays(todayIn('UTC'), -120);
	const kid = await api('POST', '/v1/kids', {
		name: 'Noa',
		birth_date: born,
		sex: 'female',
		gestation_weeks: 30,
		gestation_days: 0,
		time_zone: 'UTC'
	});
	expect(kid.status).toBe(201);
	const ages = `${formatAge(born, todayIn('UTC'))} (corrected ${formatAge(addDays(born, 70), todayIn('UTC'))})`;
	await signIn(page, ANA);
	await page.goto(`/kids/${kid.body.id}`);
	// Home puts the corrected age on its own line, with the weeks at birth.
	await expect(page.getByText(formatAge(born, todayIn('UTC')), { exact: true })).toBeVisible();
	await expect(
		page.getByText(`Corrected age ${formatAge(addDays(born, 70), todayIn('UTC'))} · born 30w 0d`, {
			exact: true
		})
	).toBeVisible();
	await expectAccessible(page);
	await page.goto(`/kids/${kid.body.id}/growth`);
	await expect(page.getByText(ages, { exact: true })).toBeVisible();

	// Leo was born early but is past his second birthday: only his age shows.
	await page.goto(`/kids/${LEO.id}`);
	await expect(
		page.getByText(formatAge(LEO.birthDate, todayIn(LEO.timeZone)), { exact: true })
	).toBeVisible();
	await expect(page.getByText(/\(corrected /)).toHaveCount(0);
});

test('a preterm kid before the due date says before expected term, never a zero corrected age', async ({
	page
}) => {
	const api = await stubAs(ANA);
	const today = todayIn('UTC');
	const born = addDays(today, -20);
	const kid = await api('POST', '/v1/kids', {
		name: 'Noa',
		birth_date: born,
		sex: 'female',
		gestation_weeks: 30,
		gestation_days: 0,
		time_zone: 'UTC'
	});
	expect(kid.status).toBe(201);
	const ages = `${formatAge(born, today)} (before expected term)`;
	await signIn(page, ANA);
	await page.goto(`/kids/${kid.body.id}`);
	await expect(page.getByText(formatAge(born, today), { exact: true })).toBeVisible();
	await expect(page.getByText('Before expected term · born 30w 0d', { exact: true })).toBeVisible();
	await expect(page.getByText(/corrected 0 days/i)).toHaveCount(0);
	await page.goto(`/kids/${kid.body.id}/growth`);
	await expect(page.getByText(ages, { exact: true })).toBeVisible();
	await expect(page.getByText(/corrected 0 days/)).toHaveCount(0);
	await expectAccessible(page);
});

test('the Growth tile follows Sickness and Medication in the 24+ band and is gone with the module off', async ({
	page
}) => {
	await signIn(page, ANA);
	await page.goto(`/kids/${LEO.id}`);
	await expect(quickLog(page).getByRole('listitem')).toHaveText([
		/^\s*Sickness/,
		/^\s*Medication/,
		/^\s*Growth/,
		/^\s*Vaccination/
	]);
	await expect(quickLog(page).getByRole('link', { name: 'Growth', exact: true })).toHaveAttribute(
		'href',
		newUrl(LEO.id)
	);
	await expectAccessible(page);

	const cleo = await stubAs(CLEO);
	expect((await cleo('PUT', `/v1/kids/${LEO.id}/modules/growth`, { enabled: false })).status).toBe(
		204
	);
	await page.reload();
	await expect(page.getByRole('heading', { name: LEO.name, level: 1 })).toBeVisible();
	await expect(quickLog(page).getByRole('listitem')).toHaveText([
		/^\s*Sickness/,
		/^\s*Medication/,
		/^\s*Vaccination/
	]);
});

test.describe('without JavaScript', () => {
	test.use({ javaScriptEnabled: false });

	test('create works as a plain form post, the unit fields following the select', async ({
		page
	}) => {
		const api = await stubAs(ANA);
		await signIn(page, ANA);
		await page.goto(newUrl(IVY.id));
		await expect(field(page, 'Pounds')).toBeHidden();
		await field(page, 'Kilograms').fill('9,5');
		await field(page, 'Length or height').fill('76');
		await save(page);
		await onGrowth(page, IVY.id, 'weight');
		await expect(rows(page).first()).toContainText('Weight 9.5 kg');
		expect((await stored(api))[0]).toMatchObject({ weight: { grams: 9500 }, length: { mm: 760 } });

		await page.goto(newUrl(IVY.id));
		await field(page, 'Weight unit').selectOption('lb');
		await expect(field(page, 'Pounds')).toBeVisible();
		await expect(field(page, 'Kilograms')).toBeHidden();
		await save(page);
		await expectFieldError(
			field(page, 'Pounds'),
			'Enter at least one of weight, length or head circumference'
		);
		await expect(field(page, 'Weight unit')).toHaveValue('lb');
	});
});
