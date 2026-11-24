import { expect, test, type Locator, type Page } from '@playwright/test';
import { todayIn } from '../src/lib/format.js';
import {
	expectAccessible,
	expectFieldError,
	expectNotFoundPage,
	signIn,
	submit,
	untilHydrated
} from './helpers.js';
import {
	ANA,
	CLEO,
	DAN,
	IVY,
	LEO,
	MIA,
	receivedRequests,
	resetStub,
	storeVaccineSchema,
	stubAs,
	UI_ORIGIN
} from './stub-api.js';

// The stub's schedules are made up: a test checks the page against the stub, never against real data.

test.beforeEach(async () => {
	await resetStub();
});

type Api = Awaited<ReturnType<typeof stubAs>>;
type Row = Record<string, unknown>;

const MS_PER_DAY = 86_400_000;
const addDays = (date: string, n: number) =>
	new Date(Date.parse(`${date}T00:00:00Z`) + n * MS_PER_DAY).toISOString().slice(0, 10);
const TODAY = todayIn(IVY.timeZone);
const AT_TWO_MONTHS = addDays(IVY.birthDate, 60);

const pageUrl = (kid: string) => `/kids/${kid}/vaccines`;
const newUrl = (kid: string) => `${pageUrl(kid)}/new`;
const editUrl = (kid: string, id: unknown) => `${pageUrl(kid)}/${id}`;
const apiPath = (kid: string, id?: unknown) => `/v1/kids/${kid}/vaccinations${id ? `/${id}` : ''}`;
const field = (page: Page, label: string) => page.getByLabel(label, { exact: true });
const save = (page: Page) => submit(page.getByRole('button', { name: 'Save', exact: true }));
const groups = (page: Page) => page.getByRole('region', { name: /^(Birth|\d+-\w+ vaccines)/ });
const group = (page: Page, title: string) =>
	page.getByRole('region', { name: new RegExp(`^${title}`) });
const line = (scope: Page | Locator, name: string) =>
	scope.getByRole('listitem', { name: new RegExp(`^${name.replace(/[()]/g, '\\$&')}`) });
const overflows = (page: Page) =>
	page.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth);
const href = (link: Locator) => link.evaluate((a) => (a as HTMLAnchorElement).href);

// Rendered as the page alert, not only in the serialised form data.
const SHOWN_REFUSAL = /role="alert"[^>]*>Only a guardian can dismiss vaccines</;
const HIB = 'Haemophilus influenzae type b (Hib)';

async function onSchedule(api: Api, kid = IVY.id, schema = 'us') {
	expect((await api('PATCH', `/v1/kids/${kid}`, { vaccine_schema: schema })).status).toBe(200);
}
const stored = async (api: Api, kid = IVY.id) =>
	((await api('GET', apiPath(kid))).body.vaccinations ?? []) as Row[];
const seed = async (api: Api, body: Row, kid = IVY.id) => {
	const r = await api('POST', apiPath(kid), body);
	expect(r.status).toBe(201);
	return r.body as Row;
};
const written = async (method: string, path: string) => {
	const writes = (await receivedRequests()).filter((r) => r.method === method && r.path === path);
	return JSON.parse(writes.at(-1)?.body ?? 'null') as Row;
};

async function onPage(page: Page, kid = IVY.id, name = IVY.name) {
	await page.waitForURL((u) => u.pathname === pageUrl(kid) && u.search === '');
	await expect(page.getByRole('heading', { name: `Vaccines for ${name}`, level: 1 })).toBeVisible();
}

test('a kid on a schedule sees the visits in age order with status texts', async ({ page }) => {
	await onSchedule(await stubAs(ANA));
	await signIn(page, ANA);
	await page.goto(pageUrl(IVY.id));
	await expect(page.getByRole('heading', { name: 'Vaccines for Ivy', level: 1 })).toBeVisible();
	await expect(page.getByText('Schedule: Stub authority, Stub edition')).toBeVisible();
	await page.getByText('About this schedule', { exact: true }).click();
	await expect(page.getByText('Stub licence')).toBeVisible();
	await expect(page.getByText('Stub attribution')).toBeVisible();
	await page.getByText('What is not included', { exact: true }).click();
	await expect(page.getByText('Stub omission')).toBeVisible();

	await expect(groups(page)).toHaveCount(3);
	await expect(groups(page).getByRole('heading', { level: 2 })).toHaveText([
		/8-week vaccines.*Overdue/,
		/12-month vaccines.*Due/,
		/18-month vaccines.*Upcoming/
	]);
	const eight = group(page, '8-week vaccines');
	await expect(line(eight, 'Diphtheria, dose 1')).toHaveAccessibleName(/Overdue/);
	await expect(line(eight, 'Diphtheria, dose 1')).toContainText(
		/Due from \d{4}-\d{2}-\d{2} · overdue from/
	);
	await expect(line(eight, `${HIB}, dose 1`)).toHaveAccessibleName(/Overdue/);
	const rota = line(eight, 'Rotavirus, dose 1');
	await expect(rota).toHaveAccessibleName(/Available/);
	await expect(rota).not.toContainText('Due from');
	await expect(line(group(page, '12-month vaccines'), 'Measles, dose 1')).toHaveAccessibleName(
		/Due/
	);
	await expect(line(group(page, '18-month vaccines'), 'Diphtheria, dose 2')).toHaveAccessibleName(
		/Upcoming/
	);

	const also = page.getByRole('region', { name: 'Also available' });
	await expect(line(also, 'Respiratory syncytial virus (RSV), dose 1')).toContainText(
		/Available from \d{4}-\d{2}-\d{2}/
	);
	await expect(also.getByRole('button', { name: /Dismiss/ })).toHaveCount(0);
	await expect(page.getByRole('region', { name: 'Every season' })).toContainText('Influenza');
	await expect(page.getByRole('region', { name: 'Other recorded vaccinations' })).toBeVisible();
	await expect(page.getByText(/^Completed/)).toHaveCount(0);
	await expectAccessible(page);
});

test('a Record link opens the form with its visit’s outstanding antigens ticked', async ({
	page
}) => {
	await onSchedule(await stubAs(ANA));
	await signIn(page, ANA);
	await page.goto(pageUrl(IVY.id));
	const record = group(page, '12-month vaccines').getByRole('link', { name: /^Record/ });
	expect(await href(record)).toBe(`${UI_ORIGIN}${newUrl(IVY.id)}?antigens=hib,measles`);
	await record.click();
	await page.waitForURL((u) => u.pathname === newUrl(IVY.id));
	await expect(page.getByRole('heading', { name: 'Record a vaccination', level: 1 })).toBeVisible();
	await expect(field(page, 'Measles')).toBeChecked();
	await expect(field(page, HIB)).toBeChecked();
	await expect(field(page, 'Diphtheria')).not.toBeChecked();

	await page.goto(`${newUrl(IVY.id)}?antigens=measles,BAD,nope,../x`);
	await expect(field(page, 'Measles')).toBeChecked();
	await expect(page.getByRole('checkbox', { checked: true })).toHaveCount(1);
});

test('a visit shows its schedule groups as sub-headings in API order', async ({ page }) => {
	await onSchedule(await stubAs(ANA));
	await signIn(page, ANA);
	await page.goto(pageUrl(IVY.id));
	const eight = group(page, '8-week vaccines');
	await expect(eight.getByRole('heading', { level: 3 })).toHaveText(['6-in-1', 'Rotavirus']);
	const sixInOne = eight.getByRole('heading', { name: '6-in-1', level: 3 }).locator('..');
	await expect(sixInOne.getByRole('listitem')).toHaveCount(2);
	await expect(line(sixInOne, 'Diphtheria, dose 1')).toHaveAccessibleName(/Overdue/);
	await expect(line(sixInOne, `${HIB}, dose 1`)).toBeVisible();
	await expect(group(page, '12-month vaccines').getByRole('heading', { level: 3 })).toHaveText([
		'Hib',
		'MMR'
	]);
	await expectAccessible(page);
});

test('a visit whose remaining lines are all available folds under Completed', async ({ page }) => {
	const api = await stubAs(ANA);
	await onSchedule(api);
	await seed(api, { given_on: AT_TWO_MONTHS, antigens: ['diphtheria', 'hib'] });
	await signIn(page, ANA);
	await page.goto(pageUrl(IVY.id));
	await expect(groups(page)).toHaveCount(2);
	const completed = page.locator('details').filter({ hasText: 'Completed (1)' });
	await completed.getByText('Completed (1)').click();
	const eight = group(completed, '8-week vaccines');
	await expect(eight.getByRole('heading', { level: 2 })).toHaveText(/Available/);
	await expect(line(eight, 'Diphtheria, dose 1')).toContainText(`Given ${AT_TWO_MONTHS}`);
	await expect(line(eight, 'Diphtheria, dose 1')).not.toContainText('Due from');
	await expect(line(eight, 'Diphtheria, dose 1').getByRole('button')).toHaveCount(0);
	const rota = line(eight, 'Rotavirus, dose 1');
	await expect(rota).toHaveAccessibleName(/Available/);
	await expect(rota.getByRole('button', { name: 'Dismiss Rotavirus, dose 1' })).toBeVisible();
	await expectAccessible(page);
});

test('dismissing a visit’s last open line folds it under Completed; undo unfolds it', async ({
	page
}) => {
	const api = await stubAs(ANA);
	await onSchedule(api);
	await seed(api, { given_on: TODAY, antigens: ['measles'] });
	await signIn(page, ANA);
	await page.goto(pageUrl(IVY.id));
	await untilHydrated(page.getByLabel('Due before'));
	const twelve = group(page, '12-month vaccines');
	await expect(twelve.getByRole('heading', { level: 2 })).toHaveText(/Due/);
	const hib = line(twelve, `${HIB}, dose 2`);
	await hib.getByLabel(`Reason for ${HIB}, dose 2`).selectOption('not_needed');
	await hib.getByRole('button', { name: `Dismiss ${HIB}, dose 2` }).click();
	const completed = page.locator('details').filter({ hasText: 'Completed (1)' });
	await expect(completed).toHaveCount(1);
	await expect(groups(page)).toHaveCount(2);
	await completed.getByText('Completed (1)').click();
	await expect(groups(page)).toHaveCount(3);
	const folded = group(completed, '12-month vaccines');
	await expect(folded.getByRole('heading', { level: 2 })).toHaveText(/Dismissed/);
	await expect(line(folded, `${HIB}, dose 2`)).toContainText('Dismissed: not needed');
	await expectAccessible(page);

	await line(folded, `${HIB}, dose 2`)
		.getByRole('button', { name: `Undo dismissal of ${HIB}, dose 2` })
		.click();
	await expect(page.getByText(/^Completed/)).toHaveCount(0);
	await expect(group(page, '12-month vaccines').getByRole('heading', { level: 2 })).toHaveText(
		/Due/
	);
});

test('dismiss-before dismisses the older lines and says how many', async ({ page }) => {
	const api = await stubAs(ANA);
	await onSchedule(api);
	await signIn(page, ANA);
	await page.goto(pageUrl(IVY.id));
	await untilHydrated(page.getByLabel('Due before'));
	const form = page.getByRole('region', { name: 'Dismiss older lines' });
	await form.getByRole('button', { name: 'Dismiss lines' }).click();
	await expectFieldError(form.getByLabel('Due before'), 'Enter a date');

	await form.getByLabel('Due before').fill(TODAY);
	await form.getByLabel('Reason', { exact: true }).selectOption('given_unknown');
	await form.getByRole('button', { name: 'Dismiss lines' }).click();
	const status = page.getByRole('status');
	await expect(status).toHaveText(/^Dismissed \d+ lines?\.$/);
	const v = (await api('GET', `/v1/kids/${IVY.id}/vaccines`)).body;
	const dismissed = (v.groups as { lines: { dismissal: unknown }[] }[])
		.flatMap((g) => g.lines)
		.filter((l) => l.dismissal !== null).length;
	expect(dismissed).toBeGreaterThan(1);
	await expect(status).toHaveText(`Dismissed ${dismissed} lines.`);
	await expect(page.getByText('Dismissed: given, date unknown').first()).toBeAttached();
	await expectAccessible(page);
});

test('no schedule: a guardian gets a settings link, a caregiver a sentence; doses still list', async ({
	page
}) => {
	const ana = await stubAs(ANA);
	await seed(ana, { given_on: '2025-01-10', antigens: ['typhoid'] }, LEO.id);
	await signIn(page, ANA);
	await page.goto(pageUrl(IVY.id));
	await expect(page.getByText('No vaccination schedule chosen yet.')).toBeVisible();
	const settings = page.getByRole('link', { name: /Choose a schedule/ });
	expect(await href(settings)).toBe(`${UI_ORIGIN}/kids/${IVY.id}/settings`);
	await expect(groups(page)).toHaveCount(0);
	await expect(page.getByRole('region', { name: 'Dismiss older lines' })).toHaveCount(0);
	await expectAccessible(page);

	await page.goto(pageUrl(LEO.id));
	await expect(page.getByText('No vaccination schedule chosen yet.')).toBeVisible();
	await expect(page.getByText('Ask a guardian to choose one.')).toBeVisible();
	await expect(page.getByRole('link', { name: /Choose a schedule/ })).toHaveCount(0);
	const other = page.getByRole('region', { name: 'Other recorded vaccinations' });
	await expect(other.getByRole('listitem')).toHaveCount(1);
	await expect(other.getByRole('listitem')).toContainText('Typhoid');
	await expect(other.getByRole('listitem')).toContainText('2025-01-10');
	await expectAccessible(page);
});

test('a stored schedule the app no longer ships reads as no schedule', async ({ page }) => {
	const api = await stubAs(ANA);
	await seed(api, { given_on: AT_TWO_MONTHS, antigens: ['diphtheria'] });
	await storeVaccineSchema(IVY.id, 'ch');
	await signIn(page, ANA);
	await page.goto(pageUrl(IVY.id));
	await expect(page.getByText('No vaccination schedule chosen yet.')).toBeVisible();
	await expect(page.getByRole('link', { name: /Choose a schedule/ })).toBeVisible();
	await expect(page.getByText(/^Schedule:/)).toHaveCount(0);
	const other = page.getByRole('region', { name: 'Other recorded vaccinations' });
	await expect(other.getByRole('listitem')).toContainText('Diphtheria');
});

test('a 6-in-1 recorded through the product shortcut fills its lines', async ({ page }) => {
	const api = await stubAs(ANA);
	await onSchedule(api);
	await signIn(page, ANA);
	await page.goto(newUrl(IVY.id));
	await expect(field(page, 'Date given')).toHaveValue(TODAY);
	await expectAccessible(page);
	await field(page, 'Product shortcut').selectOption({ label: '6-in-1 (stub)' });
	await expect(field(page, 'Diphtheria')).toBeChecked();
	await expect(field(page, HIB)).toBeChecked();
	await expect(field(page, 'Measles')).not.toBeChecked();
	await field(page, 'Date given').fill(AT_TWO_MONTHS);
	await save(page);
	await onPage(page);
	await page.getByText('Completed (1)').click();
	const eight = group(page, '8-week vaccines');
	await expect(line(eight, 'Diphtheria, dose 1')).toHaveAccessibleName(/Given/);
	await expect(line(eight, `${HIB}, dose 1`)).toContainText(`Given ${AT_TWO_MONTHS}`);
	expect(await written('POST', apiPath(IVY.id))).toEqual({
		given_on: AT_TWO_MONTHS,
		antigens: ['diphtheria', 'hib']
	});
});

test('lot and product are stored and read back on the edit page', async ({ page }) => {
	const api = await stubAs(ANA);
	await onSchedule(api);
	await signIn(page, ANA);
	await page.goto(newUrl(IVY.id));
	await field(page, 'Date given').fill(AT_TWO_MONTHS);
	await field(page, 'Measles').check();
	await field(page, 'Lot number').fill('AB123');
	await field(page, 'Product name').fill('Generic MMR');
	await save(page);
	await onPage(page);
	const [row] = await stored(api);
	expect(row).toMatchObject({ lot: 'AB123', product: 'Generic MMR', antigens: ['measles'] });

	await page.goto(editUrl(IVY.id, row.id));
	await expect(page.getByRole('heading', { name: 'Edit vaccination', level: 1 })).toBeVisible();
	await expect(field(page, 'Date given')).toHaveValue(AT_TWO_MONTHS);
	await expect(field(page, 'Measles')).toBeChecked();
	await expect(field(page, 'Lot number')).toHaveValue('AB123');
	await expect(field(page, 'Product name')).toHaveValue('Generic MMR');
	await expect(page.locator('input[type="hidden"][name$="_original"]')).toHaveCount(5);
	await expectAccessible(page);
});

test('editing only the note sends {note}; clearing the lot sends null', async ({ page }) => {
	const api = await stubAs(ANA);
	const row = await seed(api, {
		given_on: AT_TWO_MONTHS,
		antigens: ['hib'],
		lot: 'L-9',
		note: 'seeded'
	});
	const path = apiPath(IVY.id, row.id);
	await signIn(page, ANA);
	await page.goto(editUrl(IVY.id, row.id));
	await untilHydrated(field(page, 'Note'));
	await field(page, 'Note').fill('left arm');
	await save(page);
	await onPage(page);
	expect(await written('PATCH', path)).toEqual({ note: 'left arm' });

	await page.goto(editUrl(IVY.id, row.id));
	await untilHydrated(field(page, 'Lot number'));
	await expect(field(page, 'Note')).toHaveValue('left arm');
	await field(page, 'Lot number').fill('');
	await save(page);
	await onPage(page);
	expect(await written('PATCH', path)).toEqual({ lot: null });
	expect((await stored(api))[0]).toMatchObject({ lot: null, note: 'left arm' });
});

test('a vaccination is deleted after the confirm step', async ({ page }) => {
	const api = await stubAs(ANA);
	const row = await seed(api, { given_on: AT_TWO_MONTHS, antigens: ['hib'] });
	await signIn(page, ANA);
	await page.goto(editUrl(IVY.id, row.id));
	const confirm = page.getByRole('button', { name: 'Yes, delete this vaccination' });
	await expect(confirm).toBeHidden();
	await page.getByText('Delete this vaccination', { exact: true }).click();
	await confirm.click();
	await onPage(page);
	expect(await stored(api)).toEqual([]);
});

test('refused fields show their messages and keep the values', async ({ page }) => {
	await signIn(page, ANA);
	await page.goto(newUrl(IVY.id));
	const dateMessage = 'Enter a date that is not in the future and not before the birth date';

	await field(page, 'Date given').fill(addDays(TODAY, 1));
	await field(page, 'Measles').check();
	await field(page, 'Lot number').fill('KEEP-1');
	await save(page);
	await expectFieldError(field(page, 'Date given'), dateMessage);
	await expect(field(page, 'Date given')).toHaveValue(addDays(TODAY, 1));
	await expect(field(page, 'Measles')).toBeChecked();
	await expect(field(page, 'Lot number')).toHaveValue('KEEP-1');
	await expectAccessible(page);

	await field(page, 'Date given').fill(addDays(IVY.birthDate, -1));
	await save(page);
	await expectFieldError(field(page, 'Date given'), dateMessage);
	await expect(field(page, 'Date given')).toHaveValue(addDays(IVY.birthDate, -1));

	await field(page, 'Date given').fill(TODAY);
	await field(page, 'Measles').uncheck();
	await save(page);
	await expectFieldError(field(page, 'Measles'), 'Choose at least one vaccine from the list');
	await expect(field(page, 'Lot number')).toHaveValue('KEEP-1');

	await field(page, 'Other (product name required)').check();
	await save(page);
	await expectFieldError(field(page, 'Product name'), 'Other needs one');
	await expect(field(page, 'Other (product name required)')).toBeChecked();
	await expectAccessible(page);
});

test('a caregiver records a dose but gets no dismiss controls, and a posted dismiss is refused', async ({
	page
}) => {
	await onSchedule(await stubAs(CLEO), LEO.id);
	await signIn(page, ANA);
	await page.goto(pageUrl(LEO.id));
	await expect(groups(page).first()).toBeVisible();
	await expect(page.getByRole('button', { name: /Dismiss|Undo/ })).toHaveCount(0);
	await expect(page.getByRole('region', { name: 'Dismiss older lines' })).toHaveCount(0);
	await expectAccessible(page);

	await page.getByRole('link', { name: 'Record a vaccination' }).click();
	await page.waitForURL((u) => u.pathname === newUrl(LEO.id));
	await field(page, 'Diphtheria').check();
	await save(page);
	await onPage(page, LEO.id, LEO.name);
	await expect(line(page, 'Diphtheria, dose 1')).toHaveAccessibleName(/Given/);

	const response = await page.request.post(`${pageUrl(LEO.id)}?/dismiss`, {
		form: { line: 'hib-1', reason: 'not_needed' },
		headers: { origin: UI_ORIGIN, accept: 'text/html' }
	});
	expect(response.status()).toBe(403);
	expect(await response.text()).toMatch(SHOWN_REFUSAL);
	const cleo = await stubAs(CLEO);
	const v = (await cleo('GET', `/v1/kids/${LEO.id}/vaccines`)).body;
	const lines = (v.groups as { lines: { dismissal: unknown }[] }[]).flatMap((g) => g.lines);
	expect(lines.every((l) => l.dismissal === null)).toBe(true);

	const before = await page.request.post(`${pageUrl(LEO.id)}?/dismissBefore`, {
		form: { before: TODAY, reason: 'not_needed' },
		headers: { origin: UI_ORIGIN, accept: 'text/html' }
	});
	expect(before.status()).toBe(403);
	expect(await before.text()).toMatch(SHOWN_REFUSAL);
});

test('no grant, an unknown and another kid’s administration are the same 404 page', async ({
	page
}) => {
	const ana = await stubAs(ANA);
	const mias = await seed(ana, { given_on: MIA.birthDate, antigens: ['hib'] }, MIA.id);
	const ivys = await seed(ana, { given_on: AT_TWO_MONTHS, antigens: ['hib'] });
	await signIn(page, ANA);
	const unknown = await expectNotFoundPage(page, editUrl(IVY.id, 'v-000000000999'));
	expect(await expectNotFoundPage(page, editUrl(IVY.id, mias.id))).toBe(unknown);
	expect(await expectNotFoundPage(page, editUrl(IVY.id, 'bad%20id'))).toBe(unknown);

	await page.context().clearCookies();
	await signIn(page, DAN);
	for (const path of [pageUrl(IVY.id), newUrl(IVY.id), editUrl(IVY.id, ivys.id)]) {
		expect(await expectNotFoundPage(page, path)).toBe(unknown);
	}
});

test('an extra dose is listed under Other with its product and opens its edit page', async ({
	page
}) => {
	const api = await stubAs(ANA);
	await onSchedule(api);
	const typhoid = await seed(api, {
		given_on: AT_TWO_MONTHS,
		antigens: ['typhoid'],
		product: 'Travel jab'
	});
	await signIn(page, ANA);
	await page.goto(pageUrl(IVY.id));
	const other = page.getByRole('region', { name: 'Other recorded vaccinations' });
	await expect(other.getByRole('listitem')).toHaveCount(1);
	await expect(other.getByRole('listitem')).toContainText('Typhoid');
	await expect(other.getByRole('listitem')).toContainText('Travel jab');
	const link = other.getByRole('link');
	expect(await href(link)).toBe(`${UI_ORIGIN}${editUrl(IVY.id, typhoid.id)}`);
});

test('no horizontal overflow with an 80-character product on the page and a 200-character note on the edit page', async ({
	page
}, testInfo) => {
	test.skip(testInfo.project.name !== 'phone', 'the overflow is a phone-width check');
	const api = await stubAs(ANA);
	await onSchedule(api);
	const row = await seed(api, {
		given_on: AT_TWO_MONTHS,
		antigens: ['typhoid'],
		product: 'p'.repeat(80),
		note: 'n'.repeat(200)
	});
	await signIn(page, ANA);
	await page.goto(pageUrl(IVY.id));
	const other = page.getByRole('region', { name: 'Other recorded vaccinations' });
	await expect(other.getByText('p'.repeat(80))).toBeVisible();
	expect(await overflows(page)).toBe(false);
	await page.goto(editUrl(IVY.id, row.id));
	await expect(field(page, 'Note')).toHaveValue('n'.repeat(200));
	expect(await overflows(page)).toBe(false);
});

test('an Also available line carries its status in its name and links a given dose', async ({
	page
}) => {
	const api = await stubAs(ANA);
	await onSchedule(api);
	await signIn(page, ANA);
	await page.goto(pageUrl(IVY.id));
	const also = page.getByRole('region', { name: 'Also available' });
	const rsv = 'Respiratory syncytial virus (RSV), dose 1';
	await expect(line(also, rsv)).toHaveAccessibleName(/Available from \d{4}-\d{2}-\d{2}/);

	const dose = await seed(api, { given_on: AT_TWO_MONTHS, antigens: ['rsv'] });
	await page.reload();
	await expect(line(also, rsv)).toHaveAccessibleName(new RegExp(`Given ${AT_TWO_MONTHS}`));
	const link = line(also, rsv).getByRole('link', { name: `Given ${AT_TWO_MONTHS}` });
	expect(await href(link)).toBe(`${UI_ORIGIN}${editUrl(IVY.id, dose.id)}`);
	await expectAccessible(page);
});

test('the page is linked from the kid home and stays reachable with the module off', async ({
	page
}) => {
	const api = await stubAs(ANA);
	expect(
		(await api('PUT', `/v1/kids/${IVY.id}/modules/vaccination`, { enabled: false })).status
	).toBe(204);
	await signIn(page, ANA);
	await page.goto(`/kids/${IVY.id}`);
	await page.getByRole('link', { name: `Vaccines for ${IVY.name}` }).click();
	await onPage(page);
	await page.goto(`/kids/${LEO.id}`);
	const leo = page.getByRole('link', { name: `Vaccines for ${LEO.name}` });
	expect(await href(leo)).toBe(`${UI_ORIGIN}${pageUrl(LEO.id)}`);
	const row = await seed(api, { given_on: AT_TWO_MONTHS, antigens: ['hib'] });
	for (const path of [pageUrl(IVY.id), newUrl(IVY.id), editUrl(IVY.id, row.id)]) {
		const response = await page.request.get(path);
		expect(response.status()).toBe(200);
		expect(response.headers()['cache-control']).toBe('no-store');
	}
});

test.describe('without JavaScript', () => {
	test.use({ javaScriptEnabled: false });

	test('record, dismiss, undo and dismiss-before work as plain forms', async ({ page }) => {
		const api = await stubAs(ANA);
		await onSchedule(api);
		await signIn(page, ANA);
		await page.goto(newUrl(IVY.id));
		await field(page, 'Product shortcut').selectOption({ label: '6-in-1 (stub)' });
		await expect(field(page, 'Diphtheria')).not.toBeChecked();
		await field(page, 'Diphtheria').check();
		await field(page, HIB).check();
		await field(page, 'Date given').fill(AT_TWO_MONTHS);
		await save(page);
		await onPage(page);
		await page.getByText('Completed (1)').click();
		await expect(line(page, 'Diphtheria, dose 1')).toHaveAccessibleName(/Given/);
		expect((await stored(api))[0]).toMatchObject({ antigens: ['diphtheria', 'hib'] });

		const rota = line(page, 'Rotavirus, dose 1');
		await rota.getByRole('button', { name: 'Dismiss Rotavirus, dose 1' }).click();
		await onPage(page);
		await page.getByText('Completed (1)').click();
		await expect(line(page, 'Rotavirus, dose 1')).toContainText('Dismissed: not needed');
		await page.getByRole('button', { name: 'Undo dismissal of Rotavirus, dose 1' }).click();
		await onPage(page);
		await page.getByText('Completed (1)').click();
		await expect(line(page, 'Rotavirus, dose 1')).toHaveAccessibleName(/Available/);

		const form = page.getByRole('region', { name: 'Dismiss older lines' });
		await form.getByLabel('Due before').fill(TODAY);
		await form.getByRole('button', { name: 'Dismiss lines' }).click();
		await expect(page.getByRole('status')).toHaveText(/^Dismissed \d+ lines?\.$/);
	});
});
