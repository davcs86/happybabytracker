import { expect, test, type Locator, type Page } from '@playwright/test';
import {
	KID_NAME_MAX_LEN,
	MAX_EVERY_HOURS,
	MAX_SCHEDULE_TIMES,
	MEDICATION_TEXT_MAX_LEN
} from '../src/lib/constants.js';
import { whenLabel } from '../src/lib/medication.js';
import { instantToLocalInput, todayInOrUtc } from '../src/lib/time-zone.js';
import {
	expectAccessible,
	expectFieldError,
	expectNotFoundPage,
	navLink,
	signIn,
	submit,
	untilHydrated,
	formView
} from './helpers.js';
import {
	ANA,
	CLEO,
	DAN,
	LEO,
	MIA,
	moveStubClock,
	receivedRequests,
	resetStub,
	stubAs
} from './stub-api.js';

// The medication page, the one-tap dose, the dose edit page and the course create, edit and delete pages. Courses are
// mostly made through the stub API; every save waits for its response with submit().

test.beforeEach(async () => {
	await resetStub();
});

type Row = Record<string, unknown>;
type Api = Awaited<ReturnType<typeof stubAs>>;

const MS_PER_MINUTE = 60_000;
const H = 3600;
const rfc = (ms: number) => new Date(ms).toISOString().replace(/\.\d{3}Z$/, 'Z');
// An instant on a whole minute, so a datetime-local input shows it exactly.
const minute = (offset: number) =>
	rfc(Math.floor(Date.now() / MS_PER_MINUTE) * MS_PER_MINUTE + offset * MS_PER_MINUTE);
// What the page shows for an instant: a bare time on the kid's today, else with the date.
const shown = (instant: unknown, zone = 'UTC') =>
	whenLabel(String(instant), zone, todayInOrUtc(zone));
const ymd = (unixS: number) => new Date(unixS * 1000).toISOString().slice(0, 10);
const realNow = () => Math.floor(Date.now() / 1000);
// Noon UTC tomorrow: Mia and Ivy live in UTC, so it is never the UI's own today.
const NOON = (() => {
	const d = new Date();
	return Date.UTC(d.getUTCFullYear(), d.getUTCMonth(), d.getUTCDate() + 1, 12) / 1000;
})();
// The stub clock only moves forward (resetStub puts it back); a moment a few seconds after unixS.
const clockAt = (unixS: number) => moveStubClock(unixS - realNow());

const pageUrl = (kid: string = MIA.id) => `/kids/${kid}/medication`;
const doseUrl = (id: unknown, kid: string = MIA.id) => `${pageUrl(kid)}/doses/${id}`;
const COURSES = (kid: string = MIA.id) => `/v1/kids/${kid}/medications`;
const DOSES = (kid: string = MIA.id) => `/v1/kids/${kid}/medication-doses`;

async function seedCourse(api: Api, body: Row, kid: string = MIA.id) {
	const r = await api('POST', COURSES(kid), {
		dose: '2.5 ml',
		schedule: { kind: 'none' },
		...body
	});
	expect(r.status).toBe(201);
	return r.body as Row;
}
async function seedDose(api: Api, medication: unknown, body: Row = {}, kid: string = MIA.id) {
	const r = await api('POST', DOSES(kid), { medication, ...body });
	expect(r.status).toBe(201);
	return r.body as Row;
}
const doses = async (api: Api, kid: string = MIA.id) =>
	((await api('GET', DOSES(kid))).body.doses ?? []) as Row[];
const written = async (method: string, path: string) => {
	const writes = (await receivedRequests()).filter((r) => r.method === method && r.path === path);
	return JSON.parse(writes.at(-1)?.body ?? 'null') as Row;
};
const writes = async (method: string, path: string) =>
	(await receivedRequests()).filter((r) => r.method === method && r.path === path);

const card = (page: Page, name: string) =>
	page
		.getByRole('listitem')
		.filter({ has: page.getByRole('heading', { name, level: 3, exact: true }) });
const logNow = (where: Page | Locator) => where.getByRole('button', { name: 'Log dose now' });
const amountOf = (where: Page | Locator) => where.getByLabel('Amount (optional)');
const field = (page: Page, label: string) => page.getByLabel(label, { exact: true });
const saveDose = (page: Page) => submit(page.getByRole('button', { name: 'Save', exact: true }));
const overflows = (page: Page) =>
	page.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth);
async function onMedication(page: Page, kid: string = MIA.id) {
	await page.waitForURL(new RegExp(`${pageUrl(kid)}$`));
	await expect(
		page.getByRole('heading', { level: 1, name: 'Medication', exact: true })
	).toBeVisible();
}

test('lists the active courses with their schedule, last dose and next due, following the clock', async ({
	page
}) => {
	const api = await stubAs(ANA);
	await clockAt(NOON);
	await seedCourse(api, {
		name: 'Calpol',
		dose: '2.5 ml',
		schedule: { kind: 'times', times: ['20:00', '08:00'] }
	});
	await seedCourse(api, { name: 'Amoxil', dose: '5 ml', schedule: { kind: 'every', hours: 6 } });
	await seedCourse(api, { name: 'Vitamin D', dose: '1 drop' });
	await signIn(page, ANA);
	await page.goto(pageUrl());
	await expect(
		page.getByRole('heading', { name: 'Medication', exact: true, level: 1 })
	).toBeVisible();

	const calpol = card(page, 'Calpol');
	await expect(calpol).toContainText('2.5 ml');
	await expect(calpol).toContainText('08:00, 20:00');
	await expect(calpol).toContainText('No dose logged yet');
	// The clock is tomorrow noon, not the UI's today, so the next due carries its date.
	await expect(calpol).toContainText(`Next due ${ymd(NOON)} 20:00`);
	const amoxil = card(page, 'Amoxil');
	await expect(amoxil).toContainText('Every 6 hours from 12:00');
	await expect(amoxil).toContainText(`Next due ${ymd(NOON)} 18:00`);
	const vitamin = card(page, 'Vitamin D');
	await expect(vitamin).toContainText('As needed');
	await expect(vitamin).toContainText('No dose logged yet');
	await expect(vitamin.getByText(/Next due/)).toHaveCount(0);
	await expectAccessible(page);

	await clockAt(NOON + 9 * H);
	await page.reload();
	await expect(card(page, 'Calpol')).toContainText(`Next due ${ymd(NOON + 24 * H)} 08:00`);
	await expect(card(page, 'Vitamin D').getByText(/Next due/)).toHaveCount(0);
});

test('the edit button of a course links to its edit page and the page offers a new course', async ({
	page
}) => {
	const api = await stubAs(ANA);
	const calpol = await seedCourse(api, { name: 'Calpol' });
	await signIn(page, ANA);
	await page.goto(pageUrl());
	await expect(page.getByRole('link', { name: 'Edit Calpol', exact: true })).toHaveAttribute(
		'href',
		`${pageUrl()}/${calpol.id}`
	);
	await expect(page.getByRole('link', { name: 'Add a course' })).toHaveAttribute(
		'href',
		`${pageUrl()}/new`
	);
});

test('a course with no doses and none at all says so', async ({ page }) => {
	await stubAs(ANA);
	await signIn(page, ANA);
	await page.goto(pageUrl());
	await expect(page.getByText('No courses yet')).toBeVisible();
	await expect(page.getByText('No doses logged yet')).toBeVisible();
	await expectAccessible(page);
});

test('the one-tap form logs a dose now and the page shows when it was', async ({ page }) => {
	const api = await stubAs(ANA);
	const calpol = await seedCourse(api, { name: 'Calpol' });
	await signIn(page, ANA);
	await page.goto(pageUrl());
	await submit(logNow(card(page, 'Calpol')));
	await onMedication(page);
	const [first] = await doses(api);
	expect(await written('POST', DOSES())).toEqual({ medication: calpol.id });
	expect(first).toMatchObject({ medication: calpol.id, amount: null });
	await expect(card(page, 'Calpol')).toContainText(`Last dose logged at ${shown(first.given_at)}`);
	await expect(card(page, 'Calpol')).not.toContainText('No dose logged yet');
	await expect(
		page.getByRole('link', { name: new RegExp(`^Calpol, ${shown(first.given_at)}`) })
	).toBeVisible();
	await expectAccessible(page);

	await amountOf(card(page, 'Calpol')).fill('  2.5 ml ');
	await submit(logNow(card(page, 'Calpol')));
	await onMedication(page);
	expect(await written('POST', DOSES())).toEqual({ medication: calpol.id, amount: '2.5 ml' });
	expect(await doses(api)).toHaveLength(2);
	await expect(amountOf(card(page, 'Calpol'))).toHaveValue('');
	await expect(
		page.getByRole('region', { name: 'Recent doses' }).getByText('2.5 ml')
	).toBeVisible();
});

test('a double tap on Log dose now logs one dose', async ({ page }) => {
	const api = await stubAs(ANA);
	await seedCourse(api, { name: 'Calpol' });
	await signIn(page, ANA);
	await page.goto(pageUrl());
	await untilHydrated(amountOf(card(page, 'Calpol')));
	const response = page.waitForResponse(
		(r) => r.request().method() === 'POST' && new URL(r.url()).search.startsWith('?/')
	);
	await logNow(card(page, 'Calpol')).dblclick();
	await (await response).finished();
	await onMedication(page);
	await expect(logNow(card(page, 'Calpol'))).toBeEnabled();
	await page.waitForLoadState('networkidle');
	expect(await writes('POST', DOSES())).toHaveLength(1);
	expect(await doses(api)).toHaveLength(1);
});

test('a dose logged from the filtered list keeps the filter', async ({ page }) => {
	const api = await stubAs(ANA);
	const calpol = await seedCourse(api, { name: 'Calpol' });
	await seedCourse(api, { name: 'Amoxil' });
	await signIn(page, ANA);
	await page.goto(`${pageUrl()}?medication=${calpol.id}`);
	await expect(page.getByRole('heading', { name: 'Doses of Calpol' })).toBeVisible();
	await submit(logNow(card(page, 'Calpol')));
	// The new dose shows once the redirect's page has loaded.
	await expect(page.getByRole('link', { name: /^Calpol, / })).toBeVisible();
	expect(page.url()).toMatch(new RegExp(`${pageUrl()}\\?medication=${calpol.id}$`));
	await expect(page.getByRole('heading', { name: 'Doses of Calpol' })).toBeVisible();
	expect(await doses(api)).toHaveLength(1);
});

test('a dose logged a day ago shows its date, never a bare time', async ({ page }) => {
	const api = await stubAs(ANA);
	const calpol = await seedCourse(api, { name: 'Calpol', start_at: minute(-3 * 24 * 60) });
	const yesterday = minute(-24 * 60);
	await seedDose(api, calpol.id, { given_at: yesterday });
	await signIn(page, ANA);
	await page.goto(pageUrl());
	await expect(card(page, 'Calpol')).toContainText(
		`Last dose logged at ${instantToLocalInput(yesterday, 'UTC').replace('T', ' ')}`
	);
});

test('an inactive course shows no next due and an ended one waits behind the details', async ({
	page
}) => {
	const api = await stubAs(ANA);
	const times = { kind: 'times', times: ['08:00', '20:00'] };
	await seedCourse(api, { name: 'Current', schedule: times });
	await seedCourse(api, { name: 'Paused', schedule: times, active: false });
	await seedCourse(api, {
		name: 'Finished',
		schedule: times,
		start_at: minute(-10 * 24 * 60),
		end_on: ymd(realNow() - 5 * 24 * H)
	});
	await signIn(page, ANA);
	await page.goto(pageUrl());
	await expect(card(page, 'Current')).toContainText(/Next due/);
	const summary = page.getByText('Ended and inactive courses (2)');
	await expect(summary).toBeVisible();
	await expect(card(page, 'Paused')).toBeHidden();
	await expect(card(page, 'Finished')).toBeHidden();
	await summary.click();
	for (const name of ['Paused', 'Finished']) {
		await expect(card(page, name)).toBeVisible();
		await expect(card(page, name).getByText(/Next due/)).toHaveCount(0);
		await expect(logNow(card(page, name))).toBeVisible();
	}
	await expectAccessible(page);
});

test('with the medication module off the page says so and the course is kept and still accepts a dose', async ({
	page
}) => {
	const api = await stubAs(ANA);
	const course = await seedCourse(api, { name: 'Calpol' });
	const off = await api('PUT', `/v1/kids/${MIA.id}/modules/medication`, { enabled: false });
	expect(off.status).toBeLessThan(300);
	await signIn(page, ANA);
	await page.goto(pageUrl());
	await expect(
		page.getByRole('heading', { name: `Medication is switched off for ${MIA.name}`, level: 2 })
	).toBeVisible();
	await expect(card(page, 'Calpol')).toHaveCount(0);
	await expect(page.getByRole('link', { name: 'Add a course' })).toHaveCount(0);
	await expectAccessible(page);
	// The data is kept: the API still takes a dose for the course.
	await seedDose(api, course.id);
	expect(await doses(api)).toHaveLength(1);
});

test('the recent doses name their course, link to the dose and filter by course', async ({
	page
}) => {
	const api = await stubAs(ANA);
	const a = await seedCourse(api, { name: 'Calpol' });
	const b = await seedCourse(api, { name: 'Amoxil' });
	await seedDose(api, a.id, { given_at: minute(-30), amount: '2.5 ml', note: 'with food' });
	const second = await seedDose(api, b.id, { given_at: minute(-10), amount: '5 ml' });
	await signIn(page, ANA);
	await page.goto(pageUrl());
	const recent = page.getByRole('region', { name: 'Recent doses' });
	await expect(recent.getByRole('listitem')).toHaveCount(2);
	await expect(recent.getByRole('listitem').first()).toContainText('Amoxil');
	await expect(recent.getByRole('link', { name: /^Amoxil, / })).toHaveAttribute(
		'href',
		doseUrl(second.id)
	);
	await expect(recent).toContainText('with food');

	await card(page, 'Calpol')
		.getByRole('link', { name: /Show doses/ })
		.click();
	await expect(page).toHaveURL(`${pageUrl()}?medication=${a.id}`);
	const filtered = page.getByRole('region', { name: 'Doses of Calpol' });
	await expect(filtered.getByRole('listitem')).toHaveCount(1);
	await expect(filtered).not.toContainText('Amoxil');
	await expectAccessible(page);
	await page.getByRole('link', { name: 'Show all doses' }).click();
	await expect(page).toHaveURL(pageUrl());
	await expect(recent.getByRole('listitem')).toHaveCount(2);
});

test('an edit of the note alone sends only the note, and the amount and time go the same way', async ({
	page
}) => {
	const api = await stubAs(ANA);
	const calpol = await seedCourse(api, { name: 'Calpol' });
	// Off the minute on purpose: an untouched time must not be re-sent truncated.
	const given = rfc(Math.floor(Date.now() / 1000) * 1000 - 3_600_000 - 7000);
	const dose = await seedDose(api, calpol.id, { given_at: given, amount: '2.5 ml', note: 'first' });
	const path = `${DOSES()}/${dose.id}`;
	await signIn(page, ANA);
	await page.goto(pageUrl());
	await page.getByRole('link', { name: /^Calpol, / }).click();
	await untilHydrated(field(page, 'Note'));
	await expect(page).toHaveURL(doseUrl(dose.id));
	await expect(formView(page, 'Edit dose')).toBeVisible();
	await expect(page.getByText('Dose of Calpol')).toBeVisible();
	await expect(field(page, 'Given at')).toHaveValue(instantToLocalInput(given, 'UTC'));
	await expect(field(page, 'Amount')).toHaveValue('2.5 ml');
	await expect(field(page, 'Note')).toHaveValue('first');
	await expectAccessible(page);

	await field(page, 'Note').fill('after lunch');
	await saveDose(page);
	await onMedication(page);
	expect(await written('PATCH', path)).toEqual({ note: 'after lunch' });
	expect((await doses(api))[0]).toMatchObject({
		given_at: given,
		amount: '2.5 ml',
		note: 'after lunch'
	});
	await expect(page.getByRole('region', { name: 'Recent doses' })).toContainText('after lunch');

	const earlier = minute(-90);
	await page.goto(doseUrl(dose.id));
	await untilHydrated(field(page, 'Given at'));
	await field(page, 'Given at').fill(instantToLocalInput(earlier, 'UTC'));
	await field(page, 'Amount').fill('');
	await field(page, 'Note').fill('');
	await saveDose(page);
	await onMedication(page);
	expect(await written('PATCH', path)).toEqual({ given_at: earlier, amount: null, note: null });
	expect((await doses(api))[0]).toMatchObject({ given_at: earlier, amount: null, note: null });
});

test('a refused edit keeps what was typed and marks the field', async ({ page }) => {
	const api = await stubAs(ANA);
	const calpol = await seedCourse(api, { name: 'Calpol' });
	const dose = await seedDose(api, calpol.id, { given_at: minute(-30), amount: '2.5 ml' });
	const path = `${DOSES()}/${dose.id}`;
	await signIn(page, ANA);
	await page.goto(doseUrl(dose.id));
	await untilHydrated(field(page, 'Amount'));

	await field(page, 'Amount').fill('x'.repeat(MEDICATION_TEXT_MAX_LEN + 1));
	await field(page, 'Note').fill('typed note');
	await saveDose(page);
	await expectFieldError(
		field(page, 'Amount'),
		`Enter an amount of up to ${MEDICATION_TEXT_MAX_LEN} characters`
	);
	await expect(field(page, 'Amount')).toHaveValue('x'.repeat(MEDICATION_TEXT_MAX_LEN + 1));
	await expect(field(page, 'Note')).toHaveValue('typed note');
	await expect(page.getByRole('alert')).toHaveCount(1);
	await expectAccessible(page);
	expect(await writes('PATCH', path)).toEqual([]);

	// The reader accepts it; the API refuses a time ahead of the clock.
	await field(page, 'Amount').fill('2.5 ml');
	await field(page, 'Given at').fill(instantToLocalInput(minute(24 * 60), 'UTC'));
	await saveDose(page);
	await expectFieldError(field(page, 'Given at'), 'Enter a time that is not in the future');
	await expect(field(page, 'Note')).toHaveValue('typed note');
	await expectAccessible(page);
	expect((await doses(api))[0].note).toBeNull();
	// The originals are the loaded ones: fixing only the time still sends only what differs from the load.
	const fixed = minute(-20);
	await field(page, 'Given at').fill(instantToLocalInput(fixed, 'UTC'));
	await saveDose(page);
	await onMedication(page);
	expect(await written('PATCH', path)).toEqual({ given_at: fixed, note: 'typed note' });
});

test('a refused amount on one course keeps its text and leaves the other courses alone', async ({
	page
}) => {
	const api = await stubAs(ANA);
	await seedCourse(api, { name: 'Calpol' });
	await seedCourse(api, { name: 'Amoxil' });
	await signIn(page, ANA);
	await page.goto(pageUrl());
	await untilHydrated(amountOf(card(page, 'Amoxil')));
	const long = 'x'.repeat(MEDICATION_TEXT_MAX_LEN + 1);
	await amountOf(card(page, 'Amoxil')).fill(long);
	await submit(logNow(card(page, 'Amoxil')));
	await expectFieldError(
		amountOf(card(page, 'Amoxil')),
		`Enter an amount of up to ${MEDICATION_TEXT_MAX_LEN} characters`
	);
	await expect(amountOf(card(page, 'Amoxil'))).toHaveValue(long);
	await expect(amountOf(card(page, 'Calpol'))).not.toHaveAttribute('aria-invalid', 'true');
	await expect(page.getByRole('alert')).toHaveCount(1);
	await expectAccessible(page);
	expect(await doses(api)).toEqual([]);

	await amountOf(card(page, 'Amoxil')).fill('5 ml');
	await submit(logNow(card(page, 'Amoxil')));
	await expect(card(page, 'Amoxil')).toContainText('Last dose logged at');
	await expect(page.getByRole('alert')).toHaveCount(0);
	expect(await doses(api)).toHaveLength(1);
});

test('a course deleted since the page loaded is said so on the page', async ({ page }) => {
	const api = await stubAs(ANA);
	const calpol = await seedCourse(api, { name: 'Calpol' });
	await seedCourse(api, { name: 'Amoxil' });
	await signIn(page, ANA);
	await page.goto(pageUrl());
	expect((await api('DELETE', `${COURSES()}/${calpol.id}`)).status).toBe(204);
	await submit(logNow(card(page, 'Calpol')));
	await expect(page.getByRole('alert')).toHaveText('This course is no longer in the list');
	await expectAccessible(page);
	expect(await doses(api)).toEqual([]);
});

for (const [who, role] of [
	[ANA, 'caregiver'],
	[CLEO, 'guardian']
] as const) {
	test(`a ${role} logs a dose and edits it on the kid clock`, async ({ page }) => {
		const api = await stubAs(who);
		await seedCourse(api, { name: 'Calpol' }, LEO.id);
		await signIn(page, who);
		await page.goto(pageUrl(LEO.id));
		await expect(page.getByText('Times are in Europe/Madrid')).toBeVisible();
		await submit(logNow(card(page, 'Calpol')));
		await onMedication(page, LEO.id);
		const [made] = await doses(api, LEO.id);
		await expect(card(page, 'Calpol')).toContainText(
			`Last dose logged at ${shown(made.given_at, 'Europe/Madrid')}`
		);
		await page.getByRole('link', { name: /^Calpol, / }).click();
		await untilHydrated(field(page, 'Note'));
		await expect(field(page, 'Given at')).toHaveValue(
			instantToLocalInput(String(made.given_at), 'Europe/Madrid')
		);
		await field(page, 'Note').fill('by the book');
		await saveDose(page);
		await onMedication(page, LEO.id);
		expect(await written('PATCH', `${DOSES(LEO.id)}/${made.id}`)).toEqual({ note: 'by the book' });
	});
}

test('a parent without a grant, an unknown kid, a dose or course that is not theirs get the same 404 page', async ({
	page
}) => {
	const api = await stubAs(ANA);
	const calpol = await seedCourse(api, { name: 'Calpol' });
	const mine = await seedDose(api, calpol.id);
	const other = await seedCourse(await stubAs(CLEO), { name: 'Elsewhere' }, LEO.id);
	const others = await seedDose(await stubAs(CLEO), other.id, {}, LEO.id);
	await signIn(page, DAN);
	const unknown = await expectNotFoundPage(page, pageUrl('k-unknown'));
	expect(await expectNotFoundPage(page, pageUrl())).toBe(unknown);
	expect(await expectNotFoundPage(page, doseUrl(mine.id))).toBe(unknown);
	expect(await expectNotFoundPage(page, doseUrl(mine.id, 'k-unknown'))).toBe(unknown);

	await page.context().clearCookies();
	await signIn(page, ANA);
	expect(await expectNotFoundPage(page, doseUrl('d-000000009999'))).toBe(unknown);
	expect(await expectNotFoundPage(page, doseUrl(others.id))).toBe(unknown);
	expect(await expectNotFoundPage(page, doseUrl(calpol.id))).toBe(unknown);
	expect(await expectNotFoundPage(page, doseUrl('a b'))).toBe(unknown);
	expect(await expectNotFoundPage(page, `${pageUrl()}?medication=m-000000009999`)).toBe(unknown);
	expect(await expectNotFoundPage(page, `${pageUrl()}?medication=${other.id}`)).toBe(unknown);
	expect(await expectNotFoundPage(page, `${pageUrl()}?medication=a%20b`)).toBe(unknown);
	expect(await expectNotFoundPage(page, `${pageUrl()}?medication=`)).toBe(unknown);
});

test('a dose gone under an open edit form is the 404 page', async ({ page }) => {
	const api = await stubAs(ANA);
	const calpol = await seedCourse(api, { name: 'Calpol' });
	const dose = await seedDose(api, calpol.id);
	await signIn(page, ANA);
	await page.goto(doseUrl(dose.id));
	await untilHydrated(field(page, 'Note'));
	expect((await api('DELETE', `${DOSES()}/${dose.id}`)).status).toBe(204);
	await field(page, 'Note').fill('too late');
	await page.getByRole('button', { name: 'Save', exact: true }).click();
	await expect(page.getByRole('heading', { name: 'Not found' })).toBeVisible();
});

const deleteDose = (page: Page) =>
	submit(page.getByRole('button', { name: 'Yes, delete this dose' }));

test('a dose is deleted from its edit page behind the confirm, and the last dose follows', async ({
	page
}) => {
	const api = await stubAs(ANA);
	const calpol = await seedCourse(api, { name: 'Calpol' });
	const older = await seedDose(api, calpol.id, { given_at: minute(-60) });
	const newer = await seedDose(api, calpol.id, { given_at: minute(-10) });
	await signIn(page, ANA);
	await page.goto(doseUrl(newer.id));
	await expect(page.getByRole('button', { name: 'Yes, delete this dose' })).toBeHidden();
	await page.getByText('Delete this dose', { exact: true }).click();
	await expect(
		page.getByText('This removes the dose for everyone with access to Mia.')
	).toBeVisible();
	await expectAccessible(page);
	await deleteDose(page);
	await onMedication(page);
	expect((await doses(api)).map((d) => d.id)).toEqual([older.id]);
	await expect(card(page, 'Calpol')).toContainText(`Last dose logged at ${shown(older.given_at)}`);
	await expect(
		page.getByRole('region', { name: 'Recent doses' }).getByRole('listitem')
	).toHaveCount(1);
	await deleteDoseThenNone(page, api, older.id);
});

async function deleteDoseThenNone(page: Page, api: Api, id: unknown) {
	await page.goto(doseUrl(id));
	await page.getByText('Delete this dose', { exact: true }).click();
	await deleteDose(page);
	await onMedication(page);
	expect(await doses(api)).toEqual([]);
	await expect(card(page, 'Calpol')).toContainText('No dose logged yet');
}

for (const [who, role] of [
	[ANA, 'caregiver'],
	[CLEO, 'guardian']
] as const) {
	test(`a ${role} deletes a dose`, async ({ page }) => {
		const api = await stubAs(who);
		const course = await seedCourse(api, { name: 'Calpol' }, LEO.id);
		const dose = await seedDose(api, course.id, {}, LEO.id);
		await signIn(page, who);
		await page.goto(doseUrl(dose.id, LEO.id));
		await page.getByText('Delete this dose', { exact: true }).click();
		await deleteDose(page);
		await onMedication(page, LEO.id);
		expect(await doses(api, LEO.id)).toEqual([]);
		expect(await writes('DELETE', `${DOSES(LEO.id)}/${dose.id}`)).toHaveLength(1);
	});
}

test('a dose deleted under an open delete confirm is the 404 page', async ({ page }) => {
	const api = await stubAs(ANA);
	const calpol = await seedCourse(api, { name: 'Calpol' });
	const dose = await seedDose(api, calpol.id);
	await signIn(page, ANA);
	await page.goto(doseUrl(dose.id));
	await page.getByText('Delete this dose', { exact: true }).click();
	expect((await api('DELETE', `${DOSES()}/${dose.id}`)).status).toBe(204);
	await page.getByRole('button', { name: 'Yes, delete this dose' }).click();
	await expect(page.getByRole('heading', { name: 'Not found' })).toBeVisible();
});

test('the kid page links to the medication page for every role', async ({ page }) => {
	await stubAs(ANA);
	await signIn(page, ANA);
	for (const kid of [MIA, LEO]) {
		await page.goto(`/kids/${kid.id}`);
		await (await navLink(page, 'Medication')).click();
		await expect(page).toHaveURL(pageUrl(kid.id));
		await expect(
			page.getByRole('heading', { name: 'Medication', exact: true, level: 1 })
		).toBeVisible();
	}
});

test('long unbroken texts do not overflow the page at phone width', async ({ page }, testInfo) => {
	test.skip(testInfo.project.name !== 'phone', 'the overflow is a phone-width check');
	const api = await stubAs(ANA);
	const name = 'w'.repeat(MEDICATION_TEXT_MAX_LEN);
	const text = 'd'.repeat(MEDICATION_TEXT_MAX_LEN);
	const note = 'n'.repeat(200);
	const course = await seedCourse(api, {
		name,
		dose: text,
		schedule: { kind: 'times', times: ['08:00', '10:00', '12:00', '14:00', '16:00', '18:00'] }
	});
	const dose = await seedDose(api, course.id, { amount: text, note });
	await signIn(page, ANA);
	await page.goto(pageUrl());
	await untilHydrated(amountOf(page));
	await expect(page.getByRole('heading', { name, level: 3 })).toBeVisible();
	expect(await overflows(page)).toBe(false);
	await amountOf(page).fill(text);
	expect(await overflows(page)).toBe(false);
	await submit(logNow(page));
	await onMedication(page);
	expect(await overflows(page)).toBe(false);
	await expectAccessible(page);

	await page.goto(doseUrl(dose.id));
	await expect(field(page, 'Note')).toHaveValue(note);
	expect(await overflows(page)).toBe(false);
	await expectAccessible(page);
});

test.describe('without JavaScript', () => {
	test.use({ javaScriptEnabled: false });

	test('log a dose with and without an amount, fail on a long amount and fix it', async ({
		page
	}) => {
		const api = await stubAs(ANA);
		const calpol = await seedCourse(api, { name: 'Calpol' });
		await signIn(page, ANA);
		await page.goto(pageUrl());
		await expect(page.getByRole('button', { name: 'Log dose now' })).toBeVisible();
		await submit(logNow(page));
		await onMedication(page);
		expect(await written('POST', DOSES())).toEqual({ medication: calpol.id });
		await expect(card(page, 'Calpol')).toContainText('Last dose logged at');

		const long = 'x'.repeat(MEDICATION_TEXT_MAX_LEN + 1);
		await amountOf(page).fill(long);
		await submit(logNow(page));
		await expectFieldError(
			amountOf(page),
			`Enter an amount of up to ${MEDICATION_TEXT_MAX_LEN} characters`
		);
		await expect(amountOf(page)).toHaveValue(long);
		expect(await doses(api)).toHaveLength(1);

		await amountOf(page).fill('2.5 ml');
		await submit(logNow(page));
		await onMedication(page);
		expect(await written('POST', DOSES())).toEqual({ medication: calpol.id, amount: '2.5 ml' });
		await expect(amountOf(page)).toHaveValue('');
		expect(await doses(api)).toHaveLength(2);
	});

	test('edit a dose: the note alone, then a refused amount that keeps its text', async ({
		page
	}) => {
		const api = await stubAs(ANA);
		const calpol = await seedCourse(api, { name: 'Calpol' });
		const dose = await seedDose(api, calpol.id, { given_at: minute(-30), amount: '2.5 ml' });
		const path = `${DOSES()}/${dose.id}`;
		await signIn(page, ANA);
		await page.goto(pageUrl());
		await page.getByRole('link', { name: /^Calpol, / }).click();
		await expect(page).toHaveURL(doseUrl(dose.id));
		await field(page, 'Note').fill('no script');
		await saveDose(page);
		await onMedication(page);
		expect(await written('PATCH', path)).toEqual({ note: 'no script' });

		await page.goto(doseUrl(dose.id));
		await field(page, 'Amount').fill('x'.repeat(MEDICATION_TEXT_MAX_LEN + 1));
		await saveDose(page);
		await expectFieldError(
			field(page, 'Amount'),
			`Enter an amount of up to ${MEDICATION_TEXT_MAX_LEN} characters`
		);
		await expect(field(page, 'Note')).toHaveValue('no script');
		await field(page, 'Amount').fill('5 ml');
		await saveDose(page);
		await onMedication(page);
		expect(await written('PATCH', path)).toEqual({ amount: '5 ml' });
	});

	test('delete a dose in two steps: the button exists only inside the opened details', async ({
		page
	}) => {
		const api = await stubAs(ANA);
		const calpol = await seedCourse(api, { name: 'Calpol' });
		const dose = await seedDose(api, calpol.id);
		await signIn(page, ANA);
		await page.goto(doseUrl(dose.id));
		await expect(page.getByRole('button', { name: 'Yes, delete this dose' })).toBeHidden();
		await page.getByText('Delete this dose', { exact: true }).click();
		await deleteDose(page);
		await onMedication(page);
		expect(await doses(api)).toEqual([]);
		await expect(card(page, 'Calpol')).toContainText('No dose logged yet');
	});

	test('the ended and inactive courses open and close without a script', async ({ page }) => {
		const api = await stubAs(ANA);
		await seedCourse(api, { name: 'Paused', active: false });
		await signIn(page, ANA);
		await page.goto(pageUrl());
		await expect(card(page, 'Paused')).toBeHidden();
		await page.getByText('Ended and inactive courses (1)').click();
		await expect(card(page, 'Paused')).toBeVisible();
	});
});

// Course create, edit and delete.
const newUrl = (kid: string = MIA.id) => `${pageUrl(kid)}/new`;
const courseUrl = (id: unknown, kid: string = MIA.id) => `${pageUrl(kid)}/${id}`;
const courses = async (api: Api, kid: string = MIA.id) =>
	((await api('GET', COURSES(kid))).body.medications ?? []) as Row[];
const saveCourse = (page: Page) => submit(page.getByRole('button', { name: 'Save', exact: true }));
const kindRadio = (page: Page, label: string) => page.getByRole('radio', { name: label });
const deleteCourse = (page: Page) =>
	submit(page.getByRole('button', { name: 'Yes, delete this course' }));
const openDelete = (page: Page) => page.getByText('Delete this course', { exact: true }).click();
const hoursInput = (page: Page) => field(page, 'Hours between doses');
const time = (page: Page, n: number) => field(page, `Time ${n}`);
const AS_NEEDED = 'As needed';
const FIXED = 'At fixed times';
const EVERY = 'Every few hours';

test('a fixed-times course is created through the form and listed with its times in order', async ({
	page
}) => {
	const api = await stubAs(ANA);
	await signIn(page, ANA);
	await page.goto(pageUrl());
	await page.getByRole('link', { name: 'Add a course' }).click();
	await expect(page).toHaveURL(newUrl());
	await expect(formView(page, 'Add a course')).toBeVisible();
	await expect(formView(page, 'Add a course').getByText('Times are in UTC')).toBeVisible();
	await expect(kindRadio(page, AS_NEEDED)).toBeChecked();
	await expect(field(page, 'Active')).toBeChecked();
	await expectAccessible(page);

	await field(page, 'Name').fill('  Calpol ');
	await field(page, 'Dose').fill('2.5 ml');
	await kindRadio(page, FIXED).check();
	await expect(time(page, 1)).toBeVisible();
	await expect(time(page, MAX_SCHEDULE_TIMES)).toBeVisible();
	await expect(hoursInput(page)).toBeHidden();
	await time(page, 1).fill('20:00');
	await time(page, 2).fill('08:00');
	await time(page, 3).fill('12:30');
	await expectAccessible(page);
	await saveCourse(page);
	await onMedication(page);
	expect(await written('POST', COURSES())).toEqual({
		name: 'Calpol',
		dose: '2.5 ml',
		schedule: { kind: 'times', times: ['08:00', '12:30', '20:00'] }
	});
	expect(await courses(api)).toHaveLength(1);
	await expect(card(page, 'Calpol')).toContainText('08:00, 12:30, 20:00');
	await expect(card(page, 'Calpol')).toContainText('No dose logged yet');
	await expect(card(page, 'Calpol')).toContainText(/Next due/);
});

test('an every-N-hours course is created with a start on the kid clock, a last day, a note and unticked', async ({
	page
}) => {
	const api = await stubAs(ANA);
	await signIn(page, ANA);
	await page.goto(newUrl());
	const start = minute(-90);
	const last = ymd(realNow() + 3 * 24 * H);
	await field(page, 'Name').fill('Amoxil');
	await field(page, 'Dose').fill('5 ml');
	await kindRadio(page, EVERY).check();
	await expect(hoursInput(page)).toBeVisible();
	await expect(time(page, 1)).toBeHidden();
	await hoursInput(page).fill('6');
	await field(page, 'Start').fill(instantToLocalInput(start, 'UTC'));
	await field(page, 'Last day (optional)').fill(last);
	await field(page, 'Note').fill('finish the bottle');
	await field(page, 'Active').uncheck();
	await expectAccessible(page);
	await saveCourse(page);
	await onMedication(page);
	expect(await written('POST', COURSES())).toEqual({
		name: 'Amoxil',
		dose: '5 ml',
		schedule: { kind: 'every', hours: 6 },
		start_at: start,
		end_on: last,
		active: false,
		note: 'finish the bottle'
	});
	// Inactive: behind the details, with no next due.
	await expect(card(page, 'Amoxil')).toBeHidden();
	await page.getByText('Ended and inactive courses (1)').click();
	await expect(card(page, 'Amoxil')).toContainText(
		`Every 6 hours from ${instantToLocalInput(start, 'UTC').slice(-5)}`
	);
	await expect(card(page, 'Amoxil').getByText(/Next due/)).toHaveCount(0);
	expect((await courses(api))[0]).toMatchObject({ active: false, end_on: last });

	await page.goto(newUrl());
	await field(page, 'Name').fill('Vitamin D');
	await field(page, 'Dose').fill('1 drop');
	await saveCourse(page);
	await onMedication(page);
	expect(await written('POST', COURSES())).toEqual({
		name: 'Vitamin D',
		dose: '1 drop',
		schedule: { kind: 'none' }
	});
	await expect(card(page, 'Vitamin D')).toContainText('As needed');
});

test('the schedule fields follow the chosen kind and keep what was typed in the other', async ({
	page
}) => {
	await stubAs(ANA);
	await signIn(page, ANA);
	await page.goto(newUrl());
	await expect(time(page, 1)).toBeHidden();
	await expect(hoursInput(page)).toBeHidden();
	await kindRadio(page, FIXED).check();
	await time(page, 1).fill('09:00');
	await kindRadio(page, EVERY).check();
	await expect(time(page, 1)).toBeHidden();
	await hoursInput(page).fill('4');
	await kindRadio(page, FIXED).check();
	await expect(time(page, 1)).toHaveValue('09:00');
	await kindRadio(page, AS_NEEDED).check();
	await expect(time(page, 1)).toBeHidden();
	await expect(hoursInput(page)).toBeHidden();
});

test('a refused new course keeps every typed value and marks the field', async ({ page }) => {
	const api = await stubAs(ANA);
	await signIn(page, ANA);
	await page.goto(newUrl());
	await field(page, 'Name').fill('Calpol');
	await field(page, 'Dose').fill('');
	await kindRadio(page, FIXED).check();
	await time(page, 1).fill('08:00');
	await field(page, 'Note').fill('typed note');
	await saveCourse(page);
	await expectFieldError(
		field(page, 'Dose'),
		`Enter a dose of up to ${MEDICATION_TEXT_MAX_LEN} characters`
	);
	await expect(field(page, 'Name')).toHaveValue('Calpol');
	await expect(kindRadio(page, FIXED)).toBeChecked();
	await expect(time(page, 1)).toHaveValue('08:00');
	await expect(field(page, 'Note')).toHaveValue('typed note');
	await expect(page.getByRole('alert')).toHaveCount(1);
	await expectAccessible(page);
	expect(await courses(api)).toEqual([]);

	await field(page, 'Dose').fill('2.5 ml');
	await time(page, 2).fill('08:00');
	await saveCourse(page);
	await expect(page.getByRole('alert')).toContainText('Choose as needed');
	await expect(page.getByRole('radiogroup')).toHaveAttribute('aria-invalid', 'true');
	await expect(time(page, 2)).toHaveValue('08:00');
	await expectAccessible(page);
	expect(await courses(api)).toEqual([]);

	await kindRadio(page, EVERY).check();
	await hoursInput(page).fill(String(MAX_EVERY_HOURS + 1));
	await saveCourse(page);
	await expectFieldError(hoursInput(page), 'Choose as needed');
	await expect(hoursInput(page)).toHaveValue(String(MAX_EVERY_HOURS + 1));
	await hoursInput(page).fill('8');
	await saveCourse(page);
	await onMedication(page);
	expect(await written('POST', COURSES())).toEqual({
		name: 'Calpol',
		dose: '2.5 ml',
		schedule: { kind: 'every', hours: 8 },
		note: 'typed note'
	});
});

test('the API refusals of the start and the last day land on their inputs, values kept', async ({
	page
}) => {
	const api = await stubAs(ANA);
	await signIn(page, ANA);
	await page.goto(newUrl());
	await field(page, 'Name').fill('Calpol');
	await field(page, 'Dose').fill('2.5 ml');
	await field(page, 'Start').fill('2026-07-01T10:00');
	await saveCourse(page);
	await expectFieldError(field(page, 'Start'), 'Enter a start that is not before the birth date');
	await expect(field(page, 'Start')).toHaveValue('2026-07-01T10:00');
	await expect(field(page, 'Name')).toHaveValue('Calpol');

	await field(page, 'Start').fill(instantToLocalInput(minute(-60), 'UTC'));
	await field(page, 'Last day (optional)').fill(ymd(realNow() - 3 * 24 * H));
	await saveCourse(page);
	await expectFieldError(
		field(page, 'Last day (optional)'),
		'Enter a last day that is a real date'
	);
	await expect(field(page, 'Last day (optional)')).toHaveValue(ymd(realNow() - 3 * 24 * H));
	await expectAccessible(page);
	expect(await courses(api)).toEqual([]);

	await field(page, 'Last day (optional)').fill('');
	await saveCourse(page);
	await onMedication(page);
	expect(await courses(api)).toHaveLength(1);
});

test('the course cap is said on the form, values kept', async ({ page }) => {
	const api = await stubAs(ANA);
	for (let i = 0; i < 200; i++) {
		const r = await api('POST', COURSES(), {
			name: `C${i}`,
			dose: '1',
			schedule: { kind: 'none' }
		});
		if (r.status === 409) break;
		expect(r.status).toBe(201);
	}
	await signIn(page, ANA);
	await page.goto(newUrl());
	await field(page, 'Name').fill('One too many');
	await field(page, 'Dose').fill('1 ml');
	await saveCourse(page);
	await expect(page.getByRole('alert')).toHaveText(
		'This child has the maximum number of medication courses'
	);
	await expect(field(page, 'Name')).toHaveValue('One too many');
	await expectAccessible(page);
});

test('the edit page shows the stored course and an edit of the note alone sends only the note', async ({
	page
}) => {
	const api = await stubAs(ANA);
	// Off the minute on purpose: an untouched start must not be re-sent truncated.
	const start = rfc(Math.floor(Date.now() / 1000) * 1000 - 3_600_000 - 7000);
	const last = ymd(realNow() + 5 * 24 * H);
	const calpol = await seedCourse(api, {
		name: 'Calpol',
		dose: '2.5 ml',
		schedule: { kind: 'times', times: ['20:00', '08:00'] },
		start_at: start,
		end_on: last,
		note: 'first'
	});
	const path = `${COURSES()}/${calpol.id}`;
	await signIn(page, ANA);
	await page.goto(pageUrl());
	await page.getByRole('link', { name: 'Edit Calpol', exact: true }).click();
	await untilHydrated(field(page, 'Note'));
	await expect(page).toHaveURL(courseUrl(calpol.id));
	await expect(formView(page, 'Edit course')).toBeVisible();
	await expect(field(page, 'Name')).toHaveValue('Calpol');
	await expect(field(page, 'Dose')).toHaveValue('2.5 ml');
	await expect(kindRadio(page, FIXED)).toBeChecked();
	await expect(time(page, 1)).toHaveValue('08:00');
	await expect(time(page, 2)).toHaveValue('20:00');
	await expect(time(page, 3)).toHaveValue('');
	await expect(field(page, 'Start')).toHaveValue(instantToLocalInput(start, 'UTC'));
	await expect(field(page, 'Last day (optional)')).toHaveValue(last);
	await expect(field(page, 'Active')).toBeChecked();
	await expect(field(page, 'Note')).toHaveValue('first');
	await expectAccessible(page);

	await field(page, 'Note').fill('after lunch');
	await saveCourse(page);
	await onMedication(page);
	expect(await written('PATCH', path)).toEqual({ note: 'after lunch' });
	expect((await courses(api))[0]).toMatchObject({
		start_at: start,
		end_on: last,
		note: 'after lunch',
		schedule: { kind: 'times', times: ['08:00', '20:00'] }
	});

	// Saving the form untouched sends nothing but still succeeds.
	await page.goto(courseUrl(calpol.id));
	await saveCourse(page);
	await onMedication(page);
	expect(await written('PATCH', path)).toEqual({});
});

test('an edit sends the whole schedule when it changed, null for a cleared last day and the flag when toggled', async ({
	page
}) => {
	const api = await stubAs(ANA);
	const calpol = await seedCourse(api, {
		name: 'Calpol',
		schedule: { kind: 'times', times: ['08:00', '20:00'] },
		end_on: ymd(realNow() + 5 * 24 * H)
	});
	const path = `${COURSES()}/${calpol.id}`;
	await signIn(page, ANA);
	await page.goto(courseUrl(calpol.id));
	await untilHydrated(field(page, 'Name'));

	await time(page, 3).fill('13:00');
	await field(page, 'Name').fill('Calpol Plus');
	await saveCourse(page);
	await onMedication(page);
	expect(await written('PATCH', path)).toEqual({
		name: 'Calpol Plus',
		schedule: { kind: 'times', times: ['08:00', '13:00', '20:00'] }
	});

	await page.goto(courseUrl(calpol.id));
	await untilHydrated(field(page, 'Name'));
	await kindRadio(page, EVERY).check();
	await hoursInput(page).fill('8');
	await field(page, 'Last day (optional)').fill('');
	await saveCourse(page);
	await onMedication(page);
	expect(await written('PATCH', path)).toEqual({
		schedule: { kind: 'every', hours: 8 },
		end_on: null
	});
	expect((await courses(api))[0]).toMatchObject({
		schedule: { kind: 'every', hours: 8 },
		end_on: null
	});
	await expect(card(page, 'Calpol Plus')).toContainText('Every 8 hours from');

	await page.goto(courseUrl(calpol.id));
	await untilHydrated(field(page, 'Name'));
	await expect(kindRadio(page, EVERY)).toBeChecked();
	await expect(hoursInput(page)).toHaveValue('8');
	await field(page, 'Active').uncheck();
	await saveCourse(page);
	await onMedication(page);
	expect(await written('PATCH', path)).toEqual({ active: false });
	await expect(card(page, 'Calpol Plus')).toBeHidden();

	await page.goto(courseUrl(calpol.id));
	await untilHydrated(field(page, 'Name'));
	await expect(field(page, 'Active')).not.toBeChecked();
	await kindRadio(page, AS_NEEDED).check();
	await field(page, 'Active').check();
	await saveCourse(page);
	await onMedication(page);
	expect(await written('PATCH', path)).toEqual({ schedule: { kind: 'none' }, active: true });
	await expect(card(page, 'Calpol Plus')).toContainText('As needed');
});

test('a refused edit keeps what was typed and the originals stay the loaded ones', async ({
	page
}) => {
	const api = await stubAs(ANA);
	const calpol = await seedCourse(api, { name: 'Calpol', note: 'first' });
	const path = `${COURSES()}/${calpol.id}`;
	await signIn(page, ANA);
	await page.goto(courseUrl(calpol.id));
	await untilHydrated(field(page, 'Name'));

	await field(page, 'Name').fill('');
	await field(page, 'Note').fill('typed note');
	await saveCourse(page);
	await expectFieldError(
		field(page, 'Name'),
		`Enter a name of up to ${MEDICATION_TEXT_MAX_LEN} characters`
	);
	await expect(field(page, 'Note')).toHaveValue('typed note');
	await expect(page.getByRole('alert')).toHaveCount(1);
	await expectAccessible(page);
	expect(await writes('PATCH', path)).toEqual([]);

	// The reader accepts it; the API refuses a last day before the start.
	await field(page, 'Name').fill('Calpol');
	await field(page, 'Last day (optional)').fill('2026-07-10');
	await saveCourse(page);
	await expectFieldError(
		field(page, 'Last day (optional)'),
		'Enter a last day that is a real date'
	);
	await expect(field(page, 'Note')).toHaveValue('typed note');
	expect((await courses(api))[0].note).toBe('first');
	// Fixing only the last day still sends only what differs from the load.
	await field(page, 'Last day (optional)').fill('');
	await saveCourse(page);
	await onMedication(page);
	expect(await written('PATCH', path)).toEqual({ note: 'typed note' });
});

test('the delete confirm names how many doses go with the course, and they go', async ({
	page
}) => {
	const api = await stubAs(ANA);
	const doomed = await seedCourse(api, { name: 'Doomed' });
	const kept = await seedCourse(api, { name: 'Kept' });
	for (let i = 0; i < 12; i++) await seedDose(api, doomed.id, { given_at: minute(-60 - i) });
	await seedDose(api, kept.id);
	await signIn(page, ANA);
	await page.goto(courseUrl(doomed.id));
	await expect(page.getByRole('button', { name: 'Yes, delete this course' })).toBeHidden();
	await openDelete(page);
	await expect(
		page.getByText('This also deletes the 12 doses logged for this course')
	).toBeVisible();
	await expectAccessible(page);
	await deleteCourse(page);
	await onMedication(page);
	expect(await writes('DELETE', `${COURSES()}/${doomed.id}`)).toHaveLength(1);
	expect((await courses(api)).map((c) => c.id)).toEqual([kept.id]);
	expect((await doses(api)).map((d) => d.medication)).toEqual([kept.id]);
	await expect(card(page, 'Doomed')).toHaveCount(0);
	await expect(card(page, 'Kept')).toBeVisible();
	await expect(
		page.getByRole('region', { name: 'Recent doses' }).getByRole('listitem')
	).toHaveCount(1);
});

test('the delete confirm says one dose and no doses correctly', async ({ page }) => {
	const api = await stubAs(ANA);
	const one = await seedCourse(api, { name: 'One' });
	const none = await seedCourse(api, { name: 'None' });
	await seedDose(api, one.id);
	await signIn(page, ANA);
	await page.goto(courseUrl(one.id));
	await openDelete(page);
	await expect(page.getByText('This also deletes the 1 dose logged for this course')).toBeVisible();
	await page.goto(courseUrl(none.id));
	await openDelete(page);
	await expect(
		page.getByText('This also deletes the 0 doses logged for this course')
	).toBeVisible();
	await deleteCourse(page);
	await onMedication(page);
	expect((await courses(api)).map((c) => c.id)).toEqual([one.id]);
});

for (const [who, role] of [
	[ANA, 'caregiver'],
	[CLEO, 'guardian']
] as const) {
	test(`a ${role} creates, edits and deletes a course on the kid clock`, async ({ page }) => {
		const api = await stubAs(who);
		await signIn(page, who);
		await page.goto(newUrl(LEO.id));
		await expect(page.getByText('Times are in Europe/Madrid')).toBeVisible();
		const start = minute(-120);
		await field(page, 'Name').fill('Calpol');
		await field(page, 'Dose').fill('2.5 ml');
		await kindRadio(page, EVERY).check();
		await hoursInput(page).fill('6');
		await field(page, 'Start').fill(instantToLocalInput(start, 'Europe/Madrid'));
		await saveCourse(page);
		await onMedication(page, LEO.id);
		expect(await written('POST', COURSES(LEO.id))).toMatchObject({ start_at: start });
		const [made] = await courses(api, LEO.id);
		await expect(card(page, 'Calpol')).toContainText(
			`Every 6 hours from ${instantToLocalInput(start, 'Europe/Madrid').slice(-5)}`
		);

		await page.getByRole('link', { name: 'Edit Calpol', exact: true }).click();
		await untilHydrated(field(page, 'Note'));
		await expect(field(page, 'Start')).toHaveValue(instantToLocalInput(start, 'Europe/Madrid'));
		await field(page, 'Note').fill('by the book');
		await saveCourse(page);
		await onMedication(page, LEO.id);
		expect(await written('PATCH', `${COURSES(LEO.id)}/${made.id}`)).toEqual({
			note: 'by the book'
		});

		await page.goto(courseUrl(made.id, LEO.id));
		await openDelete(page);
		await deleteCourse(page);
		await onMedication(page, LEO.id);
		expect(await courses(api, LEO.id)).toEqual([]);
	});
}

test('the course pages answer the same 404 page for no grant, an unknown kid and a course that is not theirs', async ({
	page
}) => {
	const api = await stubAs(ANA);
	const mine = await seedCourse(api, { name: 'Calpol' });
	const other = await seedCourse(await stubAs(CLEO), { name: 'Elsewhere' }, LEO.id);
	await signIn(page, DAN);
	const unknown = await expectNotFoundPage(page, pageUrl('k-unknown'));
	expect(await expectNotFoundPage(page, newUrl())).toBe(unknown);
	expect(await expectNotFoundPage(page, newUrl('k-unknown'))).toBe(unknown);
	expect(await expectNotFoundPage(page, courseUrl(mine.id))).toBe(unknown);
	expect(await expectNotFoundPage(page, courseUrl(mine.id, 'k-unknown'))).toBe(unknown);

	await page.context().clearCookies();
	await signIn(page, ANA);
	expect(await expectNotFoundPage(page, courseUrl('m-000000009999'))).toBe(unknown);
	expect(await expectNotFoundPage(page, courseUrl(other.id))).toBe(unknown);
	expect(await expectNotFoundPage(page, courseUrl('a b'))).toBe(unknown);
	expect(await expectNotFoundPage(page, newUrl('k-unknown'))).toBe(unknown);
});

test('a course gone under an open edit form or an open delete confirm is the 404 page', async ({
	page
}) => {
	const api = await stubAs(ANA);
	const first = await seedCourse(api, { name: 'First' });
	const second = await seedCourse(api, { name: 'Second' });
	await signIn(page, ANA);
	await page.goto(courseUrl(first.id));
	await untilHydrated(field(page, 'Note'));
	expect((await api('DELETE', `${COURSES()}/${first.id}`)).status).toBe(204);
	await field(page, 'Note').fill('too late');
	await saveCourse(page);
	await expect(page.getByRole('heading', { name: 'Not found' })).toBeVisible();

	await page.goto(courseUrl(second.id));
	await openDelete(page);
	expect((await api('DELETE', `${COURSES()}/${second.id}`)).status).toBe(204);
	await deleteCourse(page);
	await expect(page.getByRole('heading', { name: 'Not found' })).toBeVisible();
});

test('the course pages work with the medication module off', async ({ page }) => {
	const api = await stubAs(ANA);
	const calpol = await seedCourse(api, { name: 'Calpol' });
	const off = await api('PUT', `/v1/kids/${MIA.id}/modules/medication`, { enabled: false });
	expect(off.status).toBeLessThan(300);
	await signIn(page, ANA);
	await page.goto(newUrl());
	await field(page, 'Name').fill('Amoxil');
	await field(page, 'Dose').fill('5 ml');
	await saveCourse(page);
	await onMedication(page);
	expect(await courses(api)).toHaveLength(2);
	await page.goto(courseUrl(calpol.id));
	await expect(field(page, 'Name')).toHaveValue('Calpol');
});

const pausedNote = (name: string) => `Medication is switched off for ${name}`;
const turnOnLink = (page: Page) => page.getByRole('link', { name: 'Open modules' });
const setMedication = (api: Api, kid: string, enabled: boolean) =>
	api('PUT', `/v1/kids/${kid}/modules/medication`, { enabled });

test('the medication page has no switched-off card while the module is on', async ({ page }) => {
	await stubAs(ANA);
	await signIn(page, ANA);
	await page.goto(pageUrl());
	await onMedication(page);
	await expect(page.getByText('is switched off for')).toHaveCount(0);
	await expect(turnOnLink(page)).toHaveCount(0);
	await expectAccessible(page);
});

test('a guardian sees the switched-off card with a link to the modules page, and no courses or add link', async ({
	page
}) => {
	const api = await stubAs(ANA);
	await seedCourse(api, { name: 'Calpol' });
	expect((await setMedication(api, MIA.id, false)).status).toBeLessThan(300);
	await signIn(page, ANA);
	await page.goto(pageUrl());
	await onMedication(page);
	await expect(page.getByText(pausedNote('Mia'))).toBeVisible();
	await expect(page.getByRole('alert')).toHaveCount(0);
	await expect(card(page, 'Calpol')).toHaveCount(0);
	await expect(page.getByRole('link', { name: 'Add a course' })).toHaveCount(0);
	await expectAccessible(page);
	await turnOnLink(page).click();
	await expect(page).toHaveURL(`/kids/${MIA.id}/modules`);
});

for (const [who, role, expected] of [
	[ANA, 'caregiver', false],
	[CLEO, 'guardian', true]
] as const) {
	test(`a ${role} of Leo sees the switched-off card ${expected ? 'with' : 'without'} the modules link`, async ({
		page
	}) => {
		expect((await setMedication(await stubAs(CLEO), LEO.id, false)).status).toBeLessThan(300);
		await stubAs(who);
		await signIn(page, who);
		await page.goto(pageUrl(LEO.id));
		await onMedication(page, LEO.id);
		await expect(page.getByText(pausedNote('Leo'))).toBeVisible();
		await expect(turnOnLink(page)).toHaveCount(expected ? 1 : 0);
		await expect(page.getByText('A guardian can switch it on.')).toHaveCount(expected ? 0 : 1);
		await expectAccessible(page);
	});
}

test('the switched-off card wraps a long kid name at phone width', async ({ page }, testInfo) => {
	test.skip(testInfo.project.name !== 'phone', 'the overflow is a phone-width check');
	const api = await stubAs(ANA);
	const name = 'w'.repeat(KID_NAME_MAX_LEN);
	expect((await api('PATCH', `/v1/kids/${MIA.id}`, { name })).status).toBeLessThan(300);
	expect((await setMedication(api, MIA.id, false)).status).toBeLessThan(300);
	await signIn(page, ANA);
	await page.goto(pageUrl());
	await expect(page.getByText(pausedNote(name))).toBeVisible();
	// Only the card's sentence is measured.
	const box = await page.getByText(pausedNote(name)).boundingBox();
	expect(box!.x + box!.width).toBeLessThanOrEqual(page.viewportSize()!.width);
});

test.describe('the switched-off card without JavaScript', () => {
	test.use({ javaScriptEnabled: false });

	test('the server renders the card and the link', async ({ page }) => {
		const api = await stubAs(ANA);
		expect((await setMedication(api, MIA.id, false)).status).toBeLessThan(300);
		await signIn(page, ANA);
		await page.goto(pageUrl());
		await expect(page.getByText(pausedNote('Mia'))).toBeVisible();
		await turnOnLink(page).click();
		await expect(page).toHaveURL(`/kids/${MIA.id}/modules`);
	});
});

test('long unbroken texts and twelve time fields do not overflow the course pages at phone width', async ({
	page
}, testInfo) => {
	test.skip(testInfo.project.name !== 'phone', 'the overflow is a phone-width check');
	const api = await stubAs(ANA);
	const name = 'w'.repeat(MEDICATION_TEXT_MAX_LEN);
	const text = 'd'.repeat(MEDICATION_TEXT_MAX_LEN);
	const note = 'n'.repeat(200);
	await signIn(page, ANA);
	await page.goto(newUrl());
	await field(page, 'Name').fill(name);
	await field(page, 'Dose').fill(text);
	await field(page, 'Note').fill(note);
	expect(await overflows(page)).toBe(false);
	await kindRadio(page, FIXED).check();
	await expect(time(page, MAX_SCHEDULE_TIMES)).toBeVisible();
	expect(await overflows(page)).toBe(false);
	await kindRadio(page, EVERY).check();
	await hoursInput(page).fill('999');
	await saveCourse(page);
	await expect(field(page, 'Name')).toHaveValue(name);
	expect(await overflows(page)).toBe(false);
	await expectAccessible(page);

	const course = await seedCourse(api, {
		name,
		dose: text,
		note,
		schedule: { kind: 'times', times: ['08:00', '10:00', '12:00', '14:00', '16:00', '18:00'] }
	});
	await page.goto(courseUrl(course.id));
	await expect(field(page, 'Name')).toHaveValue(name);
	expect(await overflows(page)).toBe(false);
	await openDelete(page);
	expect(await overflows(page)).toBe(false);
	await expectAccessible(page);
});

test.describe('the course pages without JavaScript', () => {
	test.use({ javaScriptEnabled: false });

	test('create a fixed-times course: the radio shows its fields, a refusal keeps them', async ({
		page
	}) => {
		const api = await stubAs(ANA);
		await signIn(page, ANA);
		await page.goto(newUrl());
		await expect(time(page, 1)).toBeHidden();
		await kindRadio(page, FIXED).check();
		await expect(time(page, 1)).toBeVisible();
		await field(page, 'Name').fill('Calpol');
		await time(page, 1).fill('20:00');
		await time(page, 2).fill('08:00');
		await saveCourse(page);
		await expectFieldError(
			field(page, 'Dose'),
			`Enter a dose of up to ${MEDICATION_TEXT_MAX_LEN} characters`
		);
		await expect(field(page, 'Name')).toHaveValue('Calpol');
		await expect(kindRadio(page, FIXED)).toBeChecked();
		await expect(time(page, 1)).toHaveValue('20:00');
		await field(page, 'Dose').fill('2.5 ml');
		await saveCourse(page);
		await onMedication(page);
		expect(await written('POST', COURSES())).toEqual({
			name: 'Calpol',
			dose: '2.5 ml',
			schedule: { kind: 'times', times: ['08:00', '20:00'] }
		});
		expect(await courses(api)).toHaveLength(1);
	});

	test('create an every-N-hours course and leave the box unticked', async ({ page }) => {
		const api = await stubAs(ANA);
		await signIn(page, ANA);
		await page.goto(newUrl());
		await field(page, 'Name').fill('Amoxil');
		await field(page, 'Dose').fill('5 ml');
		await kindRadio(page, EVERY).check();
		await hoursInput(page).fill('12');
		await field(page, 'Active').uncheck();
		await saveCourse(page);
		await onMedication(page);
		expect(await written('POST', COURSES())).toEqual({
			name: 'Amoxil',
			dose: '5 ml',
			schedule: { kind: 'every', hours: 12 },
			active: false
		});
		expect((await courses(api))[0]).toMatchObject({ active: false });
	});

	test('edit the note alone, then change the schedule and clear the last day', async ({ page }) => {
		const api = await stubAs(ANA);
		const calpol = await seedCourse(api, {
			name: 'Calpol',
			schedule: { kind: 'times', times: ['08:00'] },
			end_on: ymd(realNow() + 4 * 24 * H)
		});
		const path = `${COURSES()}/${calpol.id}`;
		await signIn(page, ANA);
		await page.goto(pageUrl());
		await page.getByRole('link', { name: 'Edit Calpol', exact: true }).click();
		await expect(page).toHaveURL(courseUrl(calpol.id));
		await field(page, 'Note').fill('no script');
		await saveCourse(page);
		await onMedication(page);
		expect(await written('PATCH', path)).toEqual({ note: 'no script' });

		await page.goto(courseUrl(calpol.id));
		await time(page, 2).fill('20:00');
		await field(page, 'Last day (optional)').fill('');
		await saveCourse(page);
		await onMedication(page);
		expect(await written('PATCH', path)).toEqual({
			schedule: { kind: 'times', times: ['08:00', '20:00'] },
			end_on: null
		});
	});

	test('delete a course in two steps: the button exists only inside the opened details', async ({
		page
	}) => {
		const api = await stubAs(ANA);
		const calpol = await seedCourse(api, { name: 'Calpol' });
		await seedDose(api, calpol.id);
		await seedDose(api, calpol.id);
		await signIn(page, ANA);
		await page.goto(courseUrl(calpol.id));
		await expect(page.getByRole('button', { name: 'Yes, delete this course' })).toBeHidden();
		await openDelete(page);
		await expect(
			page.getByText('This also deletes the 2 doses logged for this course')
		).toBeVisible();
		await deleteCourse(page);
		await onMedication(page);
		expect(await courses(api)).toEqual([]);
		expect(await doses(api)).toEqual([]);
	});
});
