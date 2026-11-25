import { expect, test, type Locator, type Page } from '@playwright/test';
import { VISIT_KIND_LABEL } from '../src/lib/constants.js';
import { instantToLocalInput } from '../src/lib/time-zone.js';
import type { VisitKind } from '../src/lib/types.js';
import { visitWhen, visitWhenLong } from '../src/lib/visits.js';
import {
	expectAccessible,
	expectFieldError,
	expectNotFoundPage,
	navLink,
	openSheet,
	signIn,
	submit,
	untilHydrated
} from './helpers.js';
import { ANA, CLEO, DAN, IVY, LEO, MIA, receivedRequests, resetStub, stubAs } from './stub-api.js';

// The visits list, the visit page and its cancel and restore, and the add and edit sheets. Visits are mostly made
// through the stub API on the real clock; every save waits for its response with submit().

test.beforeEach(async () => {
	await resetStub();
});

type Row = Record<string, unknown>;
type Api = Awaited<ReturnType<typeof stubAs>>;
type Kid = { id: string; timeZone: string };

const MS_PER_MINUTE = 60_000;
const DAY_MIN = 24 * 60;
const rfc = (ms: number) => new Date(ms).toISOString().replace(/\.\d{3}Z$/, 'Z');
// An instant on a whole minute, offset minutes from now, so a datetime-local input shows it exactly.
const minute = (offset: number) =>
	rfc(Math.floor(Date.now() / MS_PER_MINUTE) * MS_PER_MINUTE + offset * MS_PER_MINUTE);
const days = (n: number) => minute(n * DAY_MIN);

const listUrl = (kid: string = MIA.id) => `/kids/${kid}/visits`;
const visitUrl = (id: unknown, kid: string = MIA.id) => `${listUrl(kid)}/${id}`;
const newUrl = (kid: string = MIA.id) => `${listUrl(kid)}/new`;
const editUrl = (id: unknown, kid: string = MIA.id) => `${visitUrl(id, kid)}/edit`;
const VISITS = (kid: string = MIA.id) => `/v1/kids/${kid}/visits`;

async function seedVisit(api: Api, body: Row, kid: string = MIA.id) {
	const r = await api('POST', VISITS(kid), { kind: 'well_child', ...body });
	expect(r.status).toBe(201);
	return r.body as Row;
}
const written = async (method: string, path: string) => {
	const writes = (await receivedRequests()).filter((r) => r.method === method && r.path === path);
	return JSON.parse(writes.at(-1)?.body ?? 'null') as Row;
};
const setVisits = (api: Api, kid: string, enabled: boolean) =>
	api('PUT', `/v1/kids/${kid}/modules/visits`, { enabled });

// A row is named by its visible text: the kind, the date and time, then the provider and the badge.
const rowName = (v: Row, kid: Kid = MIA) => {
	const start = `${VISIT_KIND_LABEL[v.kind as VisitKind]} ${visitWhen(String(v.starts_at), kid.timeZone)}`;
	return new RegExp(`^${start.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}(\\s|$)`);
};
const group = (page: Page, name: 'Upcoming' | 'Past') =>
	page.getByRole('list', { name, exact: true });
const rowsOf = (page: Page, name: 'Upcoming' | 'Past') => group(page, name).getByRole('link');
const field = (where: Page | Locator, label: string) => where.getByLabel(label, { exact: true });
const save = (where: Page | Locator) =>
	submit(where.getByRole('button', { name: 'Save', exact: true }));
const addLink = (page: Page) => page.getByRole('link', { name: 'Add a visit', exact: true });
const editLink = (page: Page) => page.getByRole('link', { name: 'Edit visit', exact: true });
const toast = (page: Page, text: string) =>
	page.locator('[data-sonner-toast]').filter({ hasText: text });

async function onList(page: Page, kid: string = MIA.id) {
	await page.waitForURL(new RegExp(`${listUrl(kid)}$`));
	await expect(page.getByRole('heading', { level: 1, name: 'Medical visits' })).toBeVisible();
}
async function onVisit(page: Page, label: string) {
	await expect(page.getByRole('heading', { level: 1, name: label, exact: true })).toBeVisible();
}
async function fillVisit(where: Page | Locator, at: string, kind: string, kid: Kid = MIA) {
	await field(where, 'Date and time').fill(instantToLocalInput(at, kid.timeZone));
	await where.getByRole('radio', { name: kind, exact: true }).check();
}

test('lists upcoming visits soonest first and past visits newest first, cancelled ones marked', async ({
	page
}) => {
	const api = await stubAs(ANA);
	const later = await seedVisit(api, { starts_at: days(3), kind: 'dental', provider: 'Dr. Lee' });
	const soon = await seedVisit(api, { starts_at: days(1) });
	const recent = await seedVisit(api, { starts_at: days(-2), kind: 'sick' });
	const old = await seedVisit(api, { starts_at: days(-5), kind: 'emergency' });
	const dropped = await seedVisit(api, { starts_at: days(2), kind: 'specialist' });
	expect((await api('PATCH', `${VISITS()}/${dropped.id}`, { cancelled: true })).status).toBe(200);
	await signIn(page, ANA);
	const response = await page.goto(listUrl());
	expect(response?.headers()['cache-control']).toBe('no-store');
	await onList(page);
	await expect(page.getByText('Times are in UTC')).toBeVisible();
	await expect(page.getByRole('heading', { level: 2, name: 'Upcoming' })).toBeVisible();
	await expect(page.getByRole('heading', { level: 2, name: 'Past' })).toBeVisible();
	const upcoming = rowsOf(page, 'Upcoming');
	await expect(upcoming).toHaveCount(3);
	for (const [i, v] of [soon, dropped, later].entries()) {
		await expect(upcoming.nth(i)).toHaveAccessibleName(rowName(v));
	}
	const past = rowsOf(page, 'Past');
	await expect(past).toHaveCount(2);
	for (const [i, v] of [recent, old].entries()) {
		await expect(past.nth(i)).toHaveAccessibleName(rowName(v));
	}
	await expect(upcoming.nth(1)).toContainText('Cancelled');
	await expect(upcoming.nth(0)).not.toContainText('Cancelled');
	await expect(upcoming.nth(2)).toContainText('Dr. Lee');
	await expect(upcoming.nth(0).getByRole('heading', { level: 3 })).toHaveText('Well-child visit');
	await expectAccessible(page);

	await upcoming.nth(2).click();
	await expect(page).toHaveURL(visitUrl(later.id));
	await onVisit(page, 'Dental visit');
});

test('the Visits entry of the navigation opens the list', async ({ page }) => {
	await stubAs(ANA);
	await signIn(page, ANA);
	await page.goto(`/kids/${MIA.id}`);
	await (await navLink(page, 'Visits')).click();
	await onList(page);
});

test('an empty list invites to add a visit', async ({ page }) => {
	await stubAs(ANA);
	await signIn(page, ANA);
	await page.goto(listUrl());
	await onList(page);
	await expect(page.getByText('No visits yet.')).toBeVisible();
	await expect(addLink(page)).toHaveCount(1);
	await expect(addLink(page)).toHaveAttribute('data-drawer', 'Add a visit');
	await expect(page.getByRole('main').getByRole('heading', { level: 2 })).toHaveCount(0);
	await expectAccessible(page);
});

test('past visits page with Older, and Back to newest returns to the first page', async ({
	page
}) => {
	const api = await stubAs(ANA);
	const seeded: Row[] = [];
	for (let i = 1; i <= 51; i++) seeded.push(await seedVisit(api, { starts_at: days(-i) }, IVY.id));
	await signIn(page, ANA);
	await page.goto(listUrl(IVY.id));
	await expect(rowsOf(page, 'Past')).toHaveCount(50);
	await page.getByRole('link', { name: 'Older', exact: true }).click();
	await expect(rowsOf(page, 'Past')).toHaveCount(1);
	await expect(rowsOf(page, 'Past').first()).toHaveAccessibleName(rowName(seeded[50], IVY));
	await expect(page.getByRole('link', { name: 'Older', exact: true })).toHaveCount(0);
	await expectAccessible(page);
	await page.getByRole('link', { name: 'Back to newest' }).click();
	await expect(rowsOf(page, 'Past')).toHaveCount(50);
});

test('an unusable page link says so on the list', async ({ page }) => {
	await stubAs(ANA);
	await signIn(page, ANA);
	await page.goto(`${listUrl()}?cursor=not-a-cursor`);
	await expect(page.getByRole('alert')).toHaveText('That page link no longer works');
	await expect(page.getByRole('link', { name: 'Back to newest' })).toBeVisible();
});

test('add a visit in a sheet: it closes, says so and the list shows the visit', async ({
	page
}) => {
	const api = await stubAs(ANA);
	await seedVisit(api, { starts_at: days(-1) });
	await signIn(page, ANA);
	const sheet = await openSheet(page, listUrl(), addLink, 'Add a visit');
	await expect(page).toHaveURL(newUrl());
	await expect(page.getByRole('heading', { level: 1, name: 'Medical visits' })).toBeVisible();
	await expect(sheet.getByText('Times are in UTC')).toBeVisible();
	await expect(field(sheet, 'Length (minutes)')).toHaveValue('30');
	await expectAccessible(page);

	const at = days(4);
	await fillVisit(sheet, at, 'Dental visit');
	await field(sheet, 'Doctor or clinic (optional)').fill(' Dr. Lee ');
	await field(sheet, 'Place (optional)').fill('12 Main St');
	await field(sheet, 'Reason (optional)').fill('Check-up');
	await save(sheet);
	await expect(sheet).toBeHidden();
	await expect(toast(page, 'Visit added')).toBeVisible();
	await expect(page).toHaveURL(listUrl());
	expect(await written('POST', VISITS())).toEqual({
		starts_at: at,
		duration_min: 30,
		kind: 'dental',
		provider: 'Dr. Lee',
		location: '12 Main St',
		note: 'Check-up'
	});
	await expect(rowsOf(page, 'Upcoming')).toHaveCount(1);
	await expect(rowsOf(page, 'Upcoming').first()).toHaveAccessibleName(
		rowName({ kind: 'dental', starts_at: at })
	);
});

test('a refused add keeps the sheet open with its messages and values', async ({ page }) => {
	await stubAs(ANA);
	await signIn(page, ANA);
	const sheet = await openSheet(page, listUrl(), addLink, 'Add a visit');
	await field(sheet, 'Reason (optional)').fill('kept');
	await save(sheet);
	await expect(sheet.getByText('Check the fields with a message.')).toBeVisible();
	await expectFieldError(sheet.getByRole('radiogroup', { name: 'Kind' }), 'Choose a kind');
	await expect(field(sheet, 'Reason (optional)')).toHaveValue('kept');

	await sheet.getByRole('radio', { name: 'Sick visit', exact: true }).check();
	await field(sheet, 'Length (minutes)').fill('4');
	await save(sheet);
	await expectFieldError(field(sheet, 'Length (minutes)'), 'Enter a length from 5 to 720 minutes');

	await field(sheet, 'Length (minutes)').fill('30');
	await field(sheet, 'Date and time').fill('2001-01-01T09:00');
	await save(sheet);
	// The API refuses a start before the birth date; the reader has no birth date.
	await expectFieldError(
		field(sheet, 'Date and time'),
		'Enter a date and time from the birth date to 731 days from now'
	);
	await expect(page).toHaveURL(newUrl());
	await expectAccessible(page);
});

test('the add page opened directly is a page with its own heading and Cancel link', async ({
	page
}) => {
	await stubAs(ANA);
	await signIn(page, ANA);
	const response = await page.goto(newUrl());
	expect(response?.headers()['cache-control']).toBe('no-store');
	await expect(page.getByRole('heading', { level: 1, name: 'Add a visit' })).toBeVisible();
	await expect(page.getByRole('link', { name: 'Cancel' })).toHaveAttribute('href', /visits$/);
	await expectAccessible(page);
});

test.describe('without JavaScript', () => {
	test.use({ javaScriptEnabled: false });

	test('add a visit, fail on a missing kind and fix it', async ({ page }) => {
		const api = await stubAs(ANA);
		await signIn(page, ANA);
		await page.goto(listUrl());
		await addLink(page).click();
		await expect(page).toHaveURL(newUrl());
		const at = days(2);
		await field(page, 'Date and time').fill(instantToLocalInput(at, MIA.timeZone));
		await save(page);
		await expect(page.getByText('Check the fields with a message.')).toBeVisible();
		await expectFieldError(page.getByRole('radiogroup', { name: 'Kind' }), 'Choose a kind');
		await page.getByRole('radio', { name: 'Other visit', exact: true }).check();
		await save(page);
		await onList(page);
		await expect(rowsOf(page, 'Upcoming')).toHaveCount(1);
		expect(((await api('GET', VISITS())).body.visits as Row[])[0]).toMatchObject({
			kind: 'other',
			starts_at: at
		});
	});

	test('edit, cancel and restore a visit', async ({ page }) => {
		const api = await stubAs(ANA);
		const v = await seedVisit(api, { starts_at: days(3) });
		await signIn(page, ANA);
		await page.goto(visitUrl(v.id));
		await editLink(page).click();
		await expect(page).toHaveURL(editUrl(v.id));
		await field(page, 'Length (minutes)').fill('45');
		await save(page);
		await expect(page).toHaveURL(visitUrl(v.id));
		await expect(page.getByText(/· 45 min$/)).toBeVisible();

		await submit(page.getByRole('button', { name: 'Cancel visit' }));
		await expect(page.getByText('Cancelled', { exact: true })).toBeVisible();
		await submit(page.getByRole('button', { name: 'Restore visit' }));
		await expect(page.getByText('Upcoming', { exact: true })).toBeVisible();
		await expect(page.getByRole('button', { name: 'Cancel visit' })).toBeVisible();
	});
});

test('the visit page shows the visit on the kid’s clock with its actions', async ({ page }) => {
	const api = await stubAs(CLEO);
	const v = await seedVisit(
		api,
		{
			starts_at: days(2),
			duration_min: 45,
			kind: 'specialist',
			provider: 'Dr. Lee',
			location: '12 Main St',
			note: '4-month check'
		},
		LEO.id
	);
	await signIn(page, CLEO);
	const response = await page.goto(visitUrl(v.id, LEO.id));
	expect(response?.headers()['cache-control']).toBe('no-store');
	await onVisit(page, 'Specialist visit');
	await expect(page.getByText('Upcoming', { exact: true })).toBeVisible();
	await expect(
		page.getByText(
			visitWhenLong({ startsAt: String(v.starts_at), durationMin: 45 }, LEO.timeZone),
			{ exact: true }
		)
	).toBeVisible();
	await expect(page.getByText('Dr. Lee · 12 Main St', { exact: true })).toBeVisible();
	await expect(page.getByText('Reason: 4-month check', { exact: true })).toBeVisible();
	await expect(page.getByText(`Times are in ${LEO.timeZone}`)).toBeVisible();
	await expect(editLink(page)).toHaveAttribute('data-drawer', 'Edit visit');
	await expect(page.getByRole('button', { name: 'Cancel visit' })).toBeVisible();
	// An upcoming visit has no outcome to record.
	await expect(page.getByRole('link', { name: 'Record the visit' })).toHaveCount(0);
	await expectAccessible(page);
});

test('edit a visit in a sheet over the visit page: only what changed is sent', async ({ page }) => {
	const api = await stubAs(ANA);
	const v = await seedVisit(api, { starts_at: days(3), provider: 'Dr. Lee', note: 'Check' });
	await signIn(page, ANA);
	const sheet = await openSheet(page, visitUrl(v.id), editLink, 'Edit visit');
	await expect(page).toHaveURL(editUrl(v.id));
	await expect(field(sheet, 'Doctor or clinic (optional)')).toHaveValue('Dr. Lee');
	await expect(sheet.getByRole('radio', { name: 'Well-child visit' })).toBeChecked();
	await expectAccessible(page);
	await field(sheet, 'Length (minutes)').fill('60');
	await field(sheet, 'Doctor or clinic (optional)').fill('');
	await save(sheet);
	await expect(sheet).toBeHidden();
	await expect(toast(page, 'Visit saved')).toBeVisible();
	await expect(page).toHaveURL(visitUrl(v.id));
	expect(await written('PATCH', `${VISITS()}/${v.id}`)).toEqual({
		duration_min: 60,
		provider: null
	});
	await expect(page.getByText(/· 60 min$/)).toBeVisible();
	await expect(page.getByText('Dr. Lee')).toHaveCount(0);
});

test('the edit page opened directly keeps the stored start and its Cancel returns to the visit', async ({
	page
}) => {
	const api = await stubAs(ANA);
	const v = await seedVisit(api, { starts_at: days(3) });
	await signIn(page, ANA);
	await page.goto(editUrl(v.id));
	await expect(page.getByRole('heading', { level: 1, name: 'Edit visit' })).toBeVisible();
	await untilHydrated(field(page, 'Date and time'));
	await expect(field(page, 'Date and time')).toHaveValue(
		instantToLocalInput(String(v.starts_at), MIA.timeZone)
	);
	await save(page);
	await expect(page).toHaveURL(visitUrl(v.id));
	expect(await written('PATCH', `${VISITS()}/${v.id}`)).toEqual({});
	await page.goto(editUrl(v.id));
	await page.getByRole('link', { name: 'Cancel' }).click();
	await expect(page).toHaveURL(visitUrl(v.id));
});

test('cancel and restore a visit, each with its toast', async ({ page }) => {
	const api = await stubAs(ANA);
	const v = await seedVisit(api, { starts_at: days(-1), kind: 'sick' });
	await signIn(page, ANA);
	await page.goto(visitUrl(v.id));
	await onVisit(page, 'Sick visit');
	await expect(page.getByText('Past', { exact: true })).toBeVisible();
	await expect(page.getByRole('link', { name: 'Record the visit' })).toHaveAttribute(
		'data-drawer',
		'After the visit'
	);
	await submit(page.getByRole('button', { name: 'Cancel visit' }));
	await expect(toast(page, 'Visit cancelled')).toBeVisible();
	await expect(page.getByText('Cancelled', { exact: true })).toBeVisible();
	await expect(page.getByRole('link', { name: 'Record the visit' })).toHaveCount(0);
	expect(await written('PATCH', `${VISITS()}/${v.id}`)).toEqual({ cancelled: true });
	await expectAccessible(page);

	await submit(page.getByRole('button', { name: 'Restore visit' }));
	await expect(toast(page, 'Visit restored')).toBeVisible();
	await expect(page.getByText('Past', { exact: true })).toBeVisible();
	expect(await written('PATCH', `${VISITS()}/${v.id}`)).toEqual({ cancelled: false });
});

test('a visit with an outcome cannot be cancelled, and shows what was recorded', async ({
	page
}) => {
	const api = await stubAs(ANA);
	const v = await seedVisit(api, { starts_at: days(-2), kind: 'sick' });
	const done = await api('POST', `${VISITS()}/${v.id}/outcome`, {
		revision: v.revision,
		outcome_notes: 'Healthy.',
		growth: { weight: { unit: 'kg', amount: 6400 } },
		medications: [{ name: 'Amoxicillin', dose: '5 ml', schedule: { kind: 'none' } }]
	});
	expect(done.status).toBe(200);
	await signIn(page, ANA);
	await page.goto(visitUrl(v.id));
	await expect(page.getByRole('heading', { level: 2, name: 'After the visit' })).toBeVisible();
	await expect(page.getByText('Healthy.', { exact: true })).toBeVisible();
	await expect(
		page.getByRole('heading', { level: 2, name: 'Recorded at this visit' })
	).toBeVisible();
	const main = page.getByRole('main');
	const growth = main.getByRole('link', { name: /^Growth\s*\d/ });
	await expect(growth).toHaveAttribute('data-drawer', 'Edit measurement');
	await expect(growth).toHaveAttribute(
		'href',
		`/kids/${MIA.id}/growth/${(done.body.linked as { growth: Row[] }).growth[0].id}`
	);
	await expect(main.getByRole('link', { name: /^Medication\s*Amoxicillin/ })).toHaveAttribute(
		'data-drawer',
		'Edit course'
	);
	await expect(page.getByRole('link', { name: 'Record the visit' })).toHaveCount(0);
	await expect(page.getByRole('link', { name: 'Add more from the visit' })).toHaveAttribute(
		'href',
		`${visitUrl(v.id)}/outcome`
	);
	await expectAccessible(page);

	await submit(page.getByRole('button', { name: 'Cancel visit' }));
	await expect(page.getByRole('alert')).toHaveText(
		'This visit has notes or records from the visit. Remove them first, or delete the visit.'
	);
	await expect(page.getByText('Past', { exact: true })).toBeVisible();
	await expect(page.locator('[data-sonner-toast]')).toHaveCount(0);
	await expectAccessible(page);
});

test('a guardian sees the switched-off card with a link to the modules page, and no list, add link or nav entry', async ({
	page
}) => {
	const api = await stubAs(ANA);
	await seedVisit(api, { starts_at: days(1) });
	expect((await setVisits(api, MIA.id, false)).status).toBeLessThan(300);
	await signIn(page, ANA);
	await page.goto(listUrl());
	await onList(page);
	await expect(page.getByText('Medical visits is switched off for Mia')).toBeVisible();
	await expect(page.getByText('Visits are kept. Nothing is deleted.')).toBeVisible();
	await expect(group(page, 'Upcoming')).toHaveCount(0);
	await expect(addLink(page)).toHaveCount(0);
	await expect(page.getByRole('link', { name: /^\s*Visits/ })).toHaveCount(0);
	await expectAccessible(page);
	await page.getByRole('link', { name: 'Open modules' }).click();
	await expect(page).toHaveURL(`/kids/${MIA.id}/modules`);
});

test('a caregiver of Leo reads that a guardian can switch visits on', async ({ page }) => {
	expect((await setVisits(await stubAs(CLEO), LEO.id, false)).status).toBeLessThan(300);
	await stubAs(ANA);
	await signIn(page, ANA);
	await page.goto(listUrl(LEO.id));
	await onList(page, LEO.id);
	await expect(page.getByText('Medical visits is switched off for Leo')).toBeVisible();
	await expect(page.getByRole('link', { name: 'Open modules' })).toHaveCount(0);
	await expect(page.getByText('A guardian can switch it on.')).toBeVisible();
	await expectAccessible(page);
});

test('a caregiver adds, edits and cancels a visit as a guardian does', async ({ page }) => {
	const cleo = await stubAs(CLEO);
	await stubAs(ANA);
	await signIn(page, ANA);
	const sheet = await openSheet(page, listUrl(LEO.id), addLink, 'Add a visit');
	await expect(sheet.getByText(`Times are in ${LEO.timeZone}`)).toBeVisible();
	const at = days(5);
	await fillVisit(sheet, at, 'Well-child visit', LEO);
	await save(sheet);
	await expect(toast(page, 'Visit added')).toBeVisible();
	expect(await written('POST', VISITS(LEO.id))).toMatchObject({
		starts_at: at,
		kind: 'well_child'
	});
	const [made] = (await cleo('GET', VISITS(LEO.id))).body.visits as Row[];

	await rowsOf(page, 'Upcoming').first().click();
	await expect(page).toHaveURL(visitUrl(made.id, LEO.id));
	const edit = await openSheet(page, visitUrl(made.id, LEO.id), editLink, 'Edit visit');
	await edit.getByRole('radio', { name: 'Dental visit', exact: true }).check();
	await save(edit);
	await expect(toast(page, 'Visit saved')).toBeVisible();
	await onVisit(page, 'Dental visit');
	await submit(page.getByRole('button', { name: 'Cancel visit' }));
	await expect(toast(page, 'Visit cancelled')).toBeVisible();
	expect(((await cleo('GET', `${VISITS(LEO.id)}/${made.id}`)).body as Row).status).toBe(
		'cancelled'
	);
});

test('a parent without a grant, an unknown kid, a visit that is not this kid’s get the same 404 page', async ({
	page
}) => {
	const api = await stubAs(ANA);
	const mine = await seedVisit(api, { starts_at: days(1) });
	const others = await seedVisit(await stubAs(CLEO), { starts_at: days(1) }, LEO.id);
	await signIn(page, DAN);
	const unknown = await expectNotFoundPage(page, listUrl('k-unknown'));
	for (const path of [
		listUrl(),
		newUrl(),
		newUrl('k-unknown'),
		visitUrl(mine.id),
		editUrl(mine.id),
		visitUrl(mine.id, 'k-unknown')
	]) {
		expect(await expectNotFoundPage(page, path)).toBe(unknown);
	}

	await page.context().clearCookies();
	await signIn(page, ANA);
	for (const path of [
		visitUrl('vis-000000009999'),
		editUrl('vis-000000009999'),
		visitUrl(others.id),
		editUrl(others.id),
		visitUrl('a b'),
		editUrl('a b')
	]) {
		expect(await expectNotFoundPage(page, path)).toBe(unknown);
	}
});
