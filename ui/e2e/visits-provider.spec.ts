import { expect, test, type Locator, type Page } from '@playwright/test';
import { instantToLocalInput } from '../src/lib/time-zone.js';
import { expectAccessible, expectFieldError, openSheet, signIn, submit } from './helpers.js';
import { ANA, MIA, receivedRequests, resetStub, setScenario, stubAs } from './stub-api.js';

// The "Find a US provider" panel of the add and edit visit sheets, against the stub's fictional registry
// (scenario npi_on; the default is off, as at the edge).

test.beforeEach(async () => {
	await resetStub();
});

type Row = Record<string, unknown>;
type Api = Awaited<ReturnType<typeof stubAs>>;

const MS_PER_MINUTE = 60_000;
const rfc = (ms: number) => new Date(ms).toISOString().replace(/\.\d{3}Z$/, 'Z');
const days = (n: number) =>
	rfc(Math.floor(Date.now() / MS_PER_MINUTE) * MS_PER_MINUTE + n * 24 * 60 * MS_PER_MINUTE);

const listUrl = `/kids/${MIA.id}/visits`;
const newUrl = `${listUrl}/new`;
const visitUrl = (id: unknown) => `${listUrl}/${id}`;
const editUrl = (id: unknown) => `${visitUrl(id)}/edit`;
const VISITS = `/v1/kids/${MIA.id}/visits`;
const AVERY = {
	label: 'AVERY QUILL, MD',
	npi: '9000000015',
	address: '12 PINE ST, SUITE 4, SEATTLE, WA 98101',
	description: 'Pediatrics · 12 PINE ST, SUITE 4, SEATTLE, WA 98101 · 206-555-0100'
};
const REGISTRY_LINE = `From the US provider registry, NPI ${AVERY.npi}`;
const MORE = 'There may be more. Add a first name, a city or a ZIP code to narrow the search.';
const EMPTY = 'No US providers match. Check the spelling, or type the doctor or clinic yourself.';
const UNAVAILABLE =
	'The provider search isn’t available right now. You can type the doctor or clinic yourself.';
const RATE_LIMITED =
	'Too many searches, try again in a minute. You can type the doctor or clinic yourself.';

const field = (where: Page | Locator, label: string) => where.getByLabel(label, { exact: true });
const button = (where: Page | Locator, name: string) =>
	where.getByRole('button', { name, exact: true });
const summary = (where: Page | Locator) => where.getByText('Find a US provider', { exact: true });
const results = (where: Page | Locator) =>
	where.getByRole('radiogroup', { name: 'Choose a provider' });
const toast = (page: Page, text: string) =>
	page.locator('[data-sonner-toast]').filter({ hasText: text });
const addLink = (page: Page) => page.getByRole('link', { name: 'Add a visit', exact: true });
const editLink = (page: Page) => page.getByRole('link', { name: 'Edit visit', exact: true });
const written = async (method: string, path: string) => {
	const writes = (await receivedRequests()).filter((r) => r.method === method && r.path === path);
	return JSON.parse(writes.at(-1)?.body ?? 'null') as Row;
};
const searches = async () =>
	(await receivedRequests()).filter((r) => r.path === '/v1/providers/search');

async function seedVisit(api: Api, body: Row) {
	const r = await api('POST', VISITS, { kind: 'well_child', ...body });
	expect(r.status).toBe(201);
	return r.body as Row;
}
async function fillVisit(where: Page | Locator, at: string) {
	await field(where, 'Date and time').fill(instantToLocalInput(at, MIA.timeZone));
	await where.getByRole('radio', { name: 'Dental visit', exact: true }).check();
	await field(where, 'Reason (optional)').fill('Check-up');
}
async function expectVisitKept(where: Page | Locator, at: string) {
	await expect(field(where, 'Date and time')).toHaveValue(instantToLocalInput(at, MIA.timeZone));
	await expect(where.getByRole('radio', { name: 'Dental visit', exact: true })).toBeChecked();
	await expect(field(where, 'Reason (optional)')).toHaveValue('Check-up');
}
async function search(where: Page | Locator, lastName: string, state = 'WA') {
	await field(where, 'Last name').fill(lastName);
	await field(where, 'State').selectOption(state);
	await submit(button(where, 'Search'));
}
async function pick(where: Page | Locator, label: string) {
	await where.getByRole('radio', { name: label, exact: true }).check();
	await submit(button(where, 'Use this provider'));
}
async function expectAccessibleInBothThemes(page: Page) {
	await expectAccessible(page);
	await page.emulateMedia({ colorScheme: 'dark' });
	await expectAccessible(page);
	await page.emulateMedia({ colorScheme: 'light' });
}
async function addSheet(page: Page) {
	await signIn(page, ANA);
	return openSheet(page, listUrl, addLink, 'Add a visit');
}

test('search off: neither sheet offers the panel', async ({ page }) => {
	const api = await stubAs(ANA);
	const v = await seedVisit(api, { starts_at: days(3) });
	const sheet = await addSheet(page);
	await expect(field(sheet, 'Doctor or clinic (optional)')).toBeVisible();
	await expect(summary(sheet)).toHaveCount(0);
	await page.goto(editUrl(v.id));
	await expect(field(page, 'Doctor or clinic (optional)')).toBeVisible();
	await expect(summary(page)).toHaveCount(0);
	expect(await searches()).toEqual([]);
});

test('search, see only in-state practices, pick one and save it with its registry number', async ({
	page
}) => {
	await setScenario('npi_on');
	await stubAs(ANA);
	const sheet = await addSheet(page);
	await expect(summary(sheet)).toBeVisible();
	await expect(field(sheet, 'Last name')).toBeHidden();
	await expectAccessibleInBothThemes(page);

	const at = days(4);
	await fillVisit(sheet, at);
	await summary(sheet).click();
	await expect(
		sheet.getByText(
			'Only US doctors and clinics are listed. Your search is sent to the US national provider registry (NPPES).'
		)
	).toBeVisible();
	await expect(sheet.getByRole('radio', { name: 'A doctor', exact: true })).toBeChecked();
	await expect(field(sheet, 'Clinic or practice name')).toBeHidden();
	await expect(field(sheet, 'State').locator('option').first()).toHaveText('Choose a state');
	await search(sheet, 'Quill');

	await expect(sheet).toBeVisible();
	await expect(toast(page, 'Visit added')).toHaveCount(0);
	const rows = results(sheet).getByRole('radio');
	await expect(rows).toHaveCount(2);
	await expect(rows.nth(0)).toHaveAccessibleName(AVERY.label);
	await expect(rows.nth(0)).toHaveAccessibleDescription(AVERY.description);
	await expect(rows.nth(1)).toHaveAccessibleName('BLAIR QUILLAN');
	await expect(rows.nth(1)).toHaveAccessibleDescription(
		'Family Medicine · 300 CEDAR AVE, TACOMA, WA 98402'
	);
	// CASEY QUILLBY gets mail in WA but practises in OR.
	await expect(sheet.getByText('CASEY QUILLBY')).toHaveCount(0);
	await expect(sheet.getByText(MORE)).toHaveCount(0);
	await expect(field(sheet, 'Last name')).toHaveValue('Quill');
	await expectVisitKept(sheet, at);
	await expectAccessibleInBothThemes(page);
	expect(JSON.parse((await searches())[0].body ?? '')).toEqual({
		kind: 'person',
		last_name: 'Quill',
		state: 'WA'
	});

	await pick(sheet, AVERY.label);
	await expect(field(sheet, 'Last name')).toBeHidden();
	await expect(field(sheet, 'Doctor or clinic (optional)')).toHaveValue(AVERY.label);
	await expect(field(sheet, 'Place (optional)')).toHaveValue(AVERY.address);
	await expect(field(sheet, 'Doctor or clinic (optional)')).toHaveAccessibleDescription(
		REGISTRY_LINE
	);
	await expectVisitKept(sheet, at);
	await expectAccessibleInBothThemes(page);

	await submit(button(sheet, 'Save'));
	await expect(sheet).toBeHidden();
	await expect(toast(page, 'Visit added')).toBeVisible();
	expect(await written('POST', VISITS)).toEqual({
		starts_at: at,
		duration_min: 30,
		kind: 'dental',
		provider: AVERY.label,
		location: AVERY.address,
		provider_npi: AVERY.npi,
		note: 'Check-up'
	});
});

test('editing the name after picking drops the registry link', async ({ page }) => {
	await setScenario('npi_on');
	const api = await stubAs(ANA);
	const sheet = await addSheet(page);
	await fillVisit(sheet, days(2));
	await summary(sheet).click();
	await search(sheet, 'Quill');
	await pick(sheet, AVERY.label);
	await field(sheet, 'Doctor or clinic (optional)').fill('Dr. Avery Quill');
	await submit(button(sheet, 'Save'));
	await expect(toast(page, 'Visit added')).toBeVisible();
	const body = await written('POST', VISITS);
	expect(body).toMatchObject({ provider: 'Dr. Avery Quill', location: AVERY.address });
	expect(body).not.toHaveProperty('provider_npi');
	const stored = ((await api('GET', VISITS)).body.visits as Row[])[0];
	expect(stored).toMatchObject({ provider: 'Dr. Avery Quill', provider_npi: null });
});

test('no match, a full page, a clinic search and the refused fields', async ({ page }) => {
	await setScenario('npi_on');
	await stubAs(ANA);
	const sheet = await addSheet(page);
	await summary(sheet).click();

	await submit(button(sheet, 'Search'));
	await expectFieldError(field(sheet, 'Last name'), 'Enter a last name of 2 to 60 letters');
	await expect(sheet.getByText('Check the fields with a message.')).toHaveCount(0);
	await field(sheet, 'Last name').fill('Quill');
	await submit(button(sheet, 'Search'));
	await expectFieldError(field(sheet, 'State'), 'Choose a state');
	await field(sheet, 'State').selectOption('WA');
	await field(sheet, 'ZIP code (optional)').fill('981');
	await submit(button(sheet, 'Search'));
	await expectFieldError(field(sheet, 'ZIP code (optional)'), 'Enter a 5-digit ZIP code');
	await expectAccessibleInBothThemes(page);
	expect(await searches()).toEqual([]);

	await field(sheet, 'ZIP code (optional)').fill('');
	await search(sheet, 'Zed');
	await expect(sheet.getByText(EMPTY)).toBeVisible();
	await expect(results(sheet)).toHaveCount(0);

	await search(sheet, 'Lark');
	await expect(results(sheet).getByRole('radio')).toHaveCount(20);
	await expect(sheet.getByText(MORE)).toBeVisible();
	await submit(button(sheet, 'Use this provider'));
	await expectFieldError(results(sheet), 'Choose a provider');
	await expect(results(sheet).getByRole('radio')).toHaveCount(20);
	await expect(sheet.getByText('Check the fields with a message.')).toHaveCount(0);
	await expectAccessibleInBothThemes(page);

	await sheet.getByRole('radio', { name: 'A clinic or practice', exact: true }).check();
	await expect(field(sheet, 'Last name')).toBeHidden();
	await field(sheet, 'Clinic or practice name').fill('Quill & Sons');
	await field(sheet, 'State').selectOption('OR');
	await submit(button(sheet, 'Search'));
	await expect(results(sheet).getByRole('radio')).toHaveCount(1);
	await expect(results(sheet).getByRole('radio')).toHaveAccessibleName('QUILL & SONS CLINIC 2');
	expect(JSON.parse((await searches()).at(-1)?.body ?? '')).toEqual({
		kind: 'organization',
		organization: 'Quill & Sons',
		state: 'OR'
	});
});

test('the add page opened directly holds the panel in its column, long results wrapping', async ({
	page
}) => {
	await setScenario('npi_on');
	await stubAs(ANA);
	await signIn(page, ANA);
	await page.goto(newUrl);
	await expect(page.getByRole('heading', { level: 1, name: 'Add a visit' })).toBeVisible();
	await summary(page).click();
	await search(page, 'Lark');
	await expect(results(page).getByRole('radio')).toHaveCount(20);
	await pick(page, 'ROBINA LARK');
	await summary(page).click();
	await search(page, 'Quill');
	expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(
		true
	);
	await expect(button(page, 'Save')).toHaveCSS('height', '48px');
	await expectAccessibleInBothThemes(page);
});

test('an unavailable search says so in the panel, and the visit still saves', async ({ page }) => {
	await setScenario('npi_unavailable');
	await stubAs(ANA);
	const sheet = await addSheet(page);
	const at = days(3);
	await fillVisit(sheet, at);
	await summary(sheet).click();
	await search(sheet, 'Quill');
	await expect(sheet.getByRole('alert').filter({ hasText: UNAVAILABLE })).toBeVisible();
	await expectVisitKept(sheet, at);
	await expectAccessibleInBothThemes(page);
	await field(sheet, 'Doctor or clinic (optional)').fill('Dr. Lee');
	await submit(button(sheet, 'Save'));
	await expect(toast(page, 'Visit added')).toBeVisible();
	expect(await written('POST', VISITS)).toMatchObject({ provider: 'Dr. Lee', kind: 'dental' });
});

test('a rate-limited search says so in the panel', async ({ page }) => {
	await setScenario('npi_rate_limited');
	await stubAs(ANA);
	const sheet = await addSheet(page);
	await summary(sheet).click();
	await search(sheet, 'Quill');
	await expect(sheet.getByRole('alert').filter({ hasText: RATE_LIMITED })).toBeVisible();
	await expectAccessible(page);
});

test('Enter in a visit field saves; Enter in a panel field searches', async ({ page }) => {
	await setScenario('npi_on');
	await stubAs(ANA);
	const sheet = await addSheet(page);
	await fillVisit(sheet, days(5));
	await summary(sheet).click();
	await field(sheet, 'State').selectOption('WA');
	await field(sheet, 'Last name').fill('Quill');
	const found = page.waitForResponse((r) => new URL(r.url()).search === '?/find');
	await field(sheet, 'Last name').press('Enter');
	await (await found).finished();
	await expect(results(sheet).getByRole('radio')).toHaveCount(2);

	const saved = page.waitForResponse((r) => new URL(r.url()).search === '?/save');
	await field(sheet, 'Reason (optional)').press('Enter');
	await (await saved).finished();
	await expect(sheet).toBeHidden();
	await expect(toast(page, 'Visit added')).toBeVisible();
	expect(await written('POST', VISITS)).not.toHaveProperty('provider_npi');
});

test('the edit sheet shows a stored registry number, and a new pick replaces it', async ({
	page
}) => {
	await setScenario('npi_on');
	const api = await stubAs(ANA);
	const v = await seedVisit(api, {
		starts_at: days(3),
		provider: AVERY.label,
		location: AVERY.address,
		provider_npi: AVERY.npi
	});
	await signIn(page, ANA);
	const sheet = await openSheet(page, visitUrl(v.id), editLink, 'Edit visit');
	await expect(field(sheet, 'Doctor or clinic (optional)')).toHaveAccessibleDescription(
		REGISTRY_LINE
	);
	await expectAccessible(page);
	await summary(sheet).click();
	await search(sheet, 'Quillan');
	await pick(sheet, 'BLAIR QUILLAN');
	await expect(sheet.getByText(REGISTRY_LINE)).toHaveCount(0);
	await expect(field(sheet, 'Doctor or clinic (optional)')).toHaveAccessibleDescription(
		'From the US provider registry, NPI 9000000023'
	);
	await submit(button(sheet, 'Save'));
	await expect(toast(page, 'Visit saved')).toBeVisible();
	expect(await written('PATCH', `${VISITS}/${v.id}`)).toEqual({
		provider: 'BLAIR QUILLAN',
		location: '300 CEDAR AVE, TACOMA, WA 98402',
		provider_npi: '9000000023'
	});
});

test.describe('without JavaScript', () => {
	test.use({ javaScriptEnabled: false });

	test('find and pick keep what was typed, and Save stores the pick', async ({ page }) => {
		await setScenario('npi_on');
		const api = await stubAs(ANA);
		await signIn(page, ANA);
		await page.goto(newUrl);
		const at = days(2);
		await fillVisit(page, at);
		await summary(page).click();
		await search(page, 'Quill');
		await expect(page).toHaveURL(new RegExp(`${newUrl}\\?/find$`));
		await expect(results(page).getByRole('radio')).toHaveCount(2);
		await expectVisitKept(page, at);
		await expect(field(page, 'Last name')).toHaveValue('Quill');
		await expect(field(page, 'State')).toHaveValue('WA');

		await submit(button(page, 'Use this provider'));
		await expectFieldError(results(page), 'Choose a provider');
		await pick(page, AVERY.label);
		await expect(field(page, 'Last name')).toBeHidden();
		await expect(field(page, 'Doctor or clinic (optional)')).toHaveValue(AVERY.label);
		await expect(field(page, 'Place (optional)')).toHaveValue(AVERY.address);
		await expect(page.getByText(REGISTRY_LINE)).toBeVisible();
		await expectVisitKept(page, at);

		await submit(button(page, 'Save'));
		await expect(page).toHaveURL(new RegExp(`${listUrl}$`));
		expect(((await api('GET', VISITS)).body.visits as Row[])[0]).toMatchObject({
			kind: 'dental',
			starts_at: at,
			provider: AVERY.label,
			location: AVERY.address,
			provider_npi: AVERY.npi,
			note: 'Check-up'
		});
	});

	test('a clinic search shows its own field, and the outcomes show in the panel', async ({
		page
	}) => {
		await setScenario('npi_on');
		const api = await stubAs(ANA);
		const v = await seedVisit(api, { starts_at: days(3), note: 'Check-up' });
		await signIn(page, ANA);
		await page.goto(editUrl(v.id));
		await summary(page).click();
		await page.getByRole('radio', { name: 'A clinic or practice', exact: true }).check();
		await expect(field(page, 'Last name')).toBeHidden();
		await field(page, 'Clinic or practice name').fill('Quillfeather');
		await field(page, 'State').selectOption('WA');
		await submit(button(page, 'Search'));
		await expect(results(page).getByRole('radio')).toHaveAccessibleName('QUILLFEATHER PEDIATRICS');
		await expect(field(page, 'Clinic or practice name')).toHaveValue('Quillfeather');
		await pick(page, 'QUILLFEATHER PEDIATRICS');
		await submit(button(page, 'Save'));
		await expect(page).toHaveURL(new RegExp(`${visitUrl(v.id)}$`));
		expect(await written('PATCH', `${VISITS}/${v.id}`)).toEqual({
			provider: 'QUILLFEATHER PEDIATRICS',
			location: '40 ELM AVE, SEATTLE, WA 98101',
			provider_npi: '9000000049'
		});

		await setScenario('npi_unavailable');
		await page.goto(editUrl(v.id));
		await summary(page).click();
		await search(page, 'Quill');
		await expect(page.getByText(UNAVAILABLE)).toBeVisible();
		await expect(field(page, 'Reason (optional)')).toHaveValue('Check-up');
	});
});
