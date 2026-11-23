import { expect, test, type Locator, type Page } from '@playwright/test';
import { MEDICATION_TEXT_MAX_LEN } from '../src/lib/constants.js';
import { whenLabel } from '../src/lib/medication.js';
import { instantToLocalInput, todayInOrUtc } from '../src/lib/time-zone.js';
import {
	expectAccessible,
	expectFieldError,
	expectNotFoundPage,
	signIn,
	submit
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

// The medication page, the one-tap dose and the dose edit page. Courses are made through the stub API: the form
// that creates them is a later part. Every save waits for its response with submit().

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
	await expect(page.getByRole('heading', { level: 1, name: /^Medication for / })).toBeVisible();
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
	await expect(page.getByRole('heading', { name: 'Medication for Mia', level: 1 })).toBeVisible();

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

test('the course name links to its edit page and the page offers a new course', async ({
	page
}) => {
	const api = await stubAs(ANA);
	const calpol = await seedCourse(api, { name: 'Calpol' });
	await signIn(page, ANA);
	await page.goto(pageUrl());
	await expect(page.getByRole('link', { name: 'Calpol', exact: true })).toHaveAttribute(
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

test('a course may be dosed with the medication module off, and the page still lists it', async ({
	page
}) => {
	const api = await stubAs(ANA);
	await seedCourse(api, { name: 'Calpol' });
	const off = await api('PUT', `/v1/kids/${MIA.id}/modules/medication`, { enabled: false });
	expect(off.status).toBeLessThan(300);
	await signIn(page, ANA);
	await page.goto(pageUrl());
	await expect(card(page, 'Calpol')).toBeVisible();
	await submit(logNow(card(page, 'Calpol')));
	await onMedication(page);
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
	await expect(page).toHaveURL(doseUrl(dose.id));
	await expect(page.getByRole('heading', { name: 'Edit dose', level: 1 })).toBeVisible();
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
	await field(page, 'Given at').fill(instantToLocalInput(minute(-20), 'UTC'));
	await saveDose(page);
	await onMedication(page);
	expect(await written('PATCH', path)).toEqual({ given_at: minute(-20), note: 'typed note' });
});

test('a refused amount on one course keeps its text and leaves the other courses alone', async ({
	page
}) => {
	const api = await stubAs(ANA);
	await seedCourse(api, { name: 'Calpol' });
	await seedCourse(api, { name: 'Amoxil' });
	await signIn(page, ANA);
	await page.goto(pageUrl());
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
		await page.getByRole('link', { name: `Medication for ${kid.name}` }).click();
		await expect(page).toHaveURL(pageUrl(kid.id));
		await expect(
			page.getByRole('heading', { name: `Medication for ${kid.name}`, level: 1 })
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
