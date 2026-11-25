import { expect, test, type Page } from '@playwright/test';
import { MEDICATION_TEXT_MAX_LEN } from '../src/lib/constants.js';
import { formatAge, todayIn } from '../src/lib/format.js';
import {
	KID_COOKIE,
	expectAccessible,
	expectLanding,
	expectNotFoundPage,
	signIn,
	untilHydrated
} from './helpers.js';
import {
	ANA,
	CLEO,
	DAN,
	IVY,
	LEO,
	MIA,
	UI_ORIGIN,
	moveStubClock,
	receivedRequests,
	resetStub,
	setScenario,
	stubAs
} from './stub-api.js';

test.beforeEach(async () => {
	await resetStub();
});

const quickLog = (page: Page) => page.getByRole('region', { name: 'Quick log' });
const mainNav = (page: Page) => page.getByRole('navigation', { name: 'Primary' });
const running = (page: Page) => page.getByRole('region', { name: 'Running now' });
const today = (page: Page) => page.getByRole('region', { name: 'Today' });
const due = (page: Page) => page.getByRole('region', { name: 'Due' });
const MS_PER_MINUTE = 60_000;
const minutesAgo = (m: number) =>
	new Date(Date.now() - m * MS_PER_MINUTE).toISOString().replace(/\.\d{3}Z$/, 'Z');
// The wall clock in a zone, computed here so the page's own helper is not what proves it.
const clock = (iso: string, timeZone: string) =>
	new Intl.DateTimeFormat('en-GB', {
		timeZone,
		hour: '2-digit',
		minute: '2-digit',
		hourCycle: 'h23'
	}).format(new Date(iso));

// Entries minutes old fall on yesterday just after UTC midnight, and Today would not show them.
function skipNearMidnight(minutes: number) {
	const since = Date.now() - Date.parse(`${todayIn('UTC')}T00:00:00Z`);
	test.skip(since < minutes * MS_PER_MINUTE, 'entries would fall on yesterday');
}

test('a kid under 12 months: Bottle, Nursing, Diaper, Sleep, then the rest; links go to the log pages', async ({
	page
}) => {
	await signIn(page, ANA);
	await expectLanding(page, { kid: MIA });
	await expect(page.getByText(formatAge(MIA.birthDate, todayIn(MIA.timeZone)))).toBeVisible();
	await expect(quickLog(page).getByRole('listitem')).toHaveText([
		'Bottle',
		'Nursing',
		'Diaper',
		'Sleep',
		'Pumping',
		'Feeding',
		'Milestones',
		'Growth',
		'Vaccination',
		'Sickness',
		'Medication'
	]);
	const log = (name: string) => quickLog(page).getByRole('link', { name, exact: true });
	await expect(log('Bottle')).toHaveAttribute('href', `/kids/${MIA.id}/log/bottles`);
	await expect(log('Nursing')).toHaveAttribute('href', `/kids/${MIA.id}/log/nursing`);
	await expect(log('Diaper')).toHaveAttribute('href', `/kids/${MIA.id}/log/diapers`);
	await expect(log('Pumping')).toHaveAttribute('href', `/kids/${MIA.id}/log/pumping`);
	await expect(log('Feeding')).toHaveAttribute('href', `/kids/${MIA.id}/log/meals`);
	await expect(log('Growth')).toHaveAttribute('href', `/kids/${MIA.id}/growth/new`);
	await expect(quickLog(page).getByRole('button', { name: 'Sleep' })).toBeVisible();
	await expect(page.getByRole('heading', { name: 'Due' })).toHaveCount(0);
	await expectAccessible(page);
});

test('a 13-month kid starts with Feeding', async ({ page }) => {
	await signIn(page, ANA);
	await page.goto(`/kids/${IVY.id}`);
	await expect(page.getByRole('heading', { name: IVY.name, level: 1 })).toBeVisible();
	await expect(quickLog(page).getByRole('listitem')).toHaveText([
		'Feeding',
		'Bottle',
		'Sleep',
		'Diaper',
		'Nursing',
		'Pumping',
		'Milestones',
		'Growth',
		'Vaccination',
		'Sickness',
		'Medication'
	]);
	await expectAccessible(page);
});

test('a kid over 24 months has no timer trackers and no empty headings', async ({ page }) => {
	await signIn(page, ANA);
	await page.goto(`/kids/${LEO.id}`);
	await expect(page.getByRole('heading', { name: LEO.name, level: 1 })).toBeVisible();
	await expect(page.getByRole('button', { name: /Sleep|Nursing/ })).toHaveCount(0);
	await expect(page.getByRole('link', { name: /Sleep|Nursing/ })).toHaveCount(0);
	await expect(quickLog(page).getByRole('listitem')).toHaveText([
		'Sickness',
		'Medication',
		'Growth',
		'Vaccination'
	]);
	for (const heading of ['Due', 'Running now', 'Today']) {
		await expect(page.getByRole('heading', { name: heading })).toHaveCount(0);
	}
	await expectAccessible(page);
});

test('the Vaccination tile opens the Vaccines page in every band and hides with the module off', async ({
	page
}) => {
	await signIn(page, ANA);
	for (const kid of [MIA, IVY, LEO]) {
		await page.goto(`/kids/${kid.id}`);
		await expect(page.getByRole('heading', { name: kid.name, level: 1 })).toBeVisible();
		await expect(
			quickLog(page).getByRole('link', { name: 'Vaccination', exact: true })
		).toHaveAttribute('href', `/kids/${kid.id}/vaccines`);
	}
	await expectAccessible(page);
	await quickLog(page).getByRole('link', { name: 'Vaccination', exact: true }).click();
	await expect(
		page.getByRole('heading', { name: `Vaccines for ${LEO.name}`, level: 1 })
	).toBeVisible();

	const ana = await stubAs(ANA);
	expect(
		(await ana('PUT', `/v1/kids/${MIA.id}/modules/vaccination`, { enabled: false })).status
	).toBe(204);
	await page.goto(`/kids/${MIA.id}`);
	await expect(quickLog(page).getByRole('listitem')).toHaveText([
		'Bottle',
		'Nursing',
		'Diaper',
		'Sleep',
		'Pumping',
		'Feeding',
		'Milestones',
		'Growth',
		'Sickness',
		'Medication'
	]);
});

test('the Milestones tile opens the Milestones page and hides with the module off', async ({
	page
}) => {
	await signIn(page, ANA);
	const tile = quickLog(page).getByRole('link', { name: 'Milestones', exact: true });
	await expect(tile).toHaveAttribute('href', `/kids/${MIA.id}/milestones`);
	await expectAccessible(page);
	await tile.click();
	await expect(
		page.getByRole('heading', { name: `Milestones for ${MIA.name}`, level: 1 })
	).toBeVisible();

	const ana = await stubAs(ANA);
	expect(
		(await ana('PUT', `/v1/kids/${MIA.id}/modules/milestones`, { enabled: false })).status
	).toBe(204);
	await page.goto(`/kids/${MIA.id}`);
	await expect(
		quickLog(page).getByRole('link', { name: 'Vaccination', exact: true })
	).toBeVisible();
	await expect(tile).toHaveCount(0);
});

test('Due lists each visit with its status as text, linked to the Vaccines page', async ({
	page
}) => {
	skipNearMidnight(5);
	const ana = await stubAs(ANA);
	expect((await ana('PATCH', `/v1/kids/${IVY.id}`, { vaccine_schema: 'us' })).status).toBe(200);
	expect((await ana('POST', `/v1/kids/${IVY.id}/diapers`, { kind: 'wet' })).status).toBe(201);
	await signIn(page, ANA);
	await page.goto(`/kids/${IVY.id}`);
	await expect(page.getByRole('heading', { name: 'Due', level: 2 })).toBeVisible();
	await expect(due(page).getByRole('listitem')).toHaveText([
		'8-week vaccines · Overdue',
		'12-month vaccines · Due'
	]);
	const links = due(page).getByRole('link');
	await expect(links).toHaveCount(2);
	for (const link of await links.all()) {
		expect(await link.evaluate((a) => (a as HTMLAnchorElement).href)).toBe(
			`${UI_ORIGIN}/kids/${IVY.id}/vaccines`
		);
	}
	// After Running now (the left column) and before Today.
	await expect(today(page)).toBeVisible();
	const headings = await page.getByRole('heading', { level: 2 }).allTextContents();
	expect(headings.indexOf('Due')).toBeGreaterThan(headings.indexOf('Quick log'));
	expect(headings.indexOf('Due')).toBeLessThan(headings.indexOf('Today'));
	await expectAccessible(page);
	await links.first().click();
	await expect(
		page.getByRole('heading', { name: `Vaccines for ${IVY.name}`, level: 1 })
	).toBeVisible();

	expect(
		(await ana('PUT', `/v1/kids/${IVY.id}/modules/vaccination`, { enabled: false })).status
	).toBe(204);
	await page.goto(`/kids/${IVY.id}`);
	await expect(page.getByRole('heading', { name: IVY.name, level: 1 })).toBeVisible();
	await expect(page.getByRole('heading', { name: 'Due' })).toHaveCount(0);
	await expect(quickLog(page).getByRole('link', { name: 'Vaccination' })).toHaveCount(0);
});

test('with a timer running, Due sits between Running now and Today', async ({ page }) => {
	skipNearMidnight(5);
	const ana = await stubAs(ANA);
	expect((await ana('PATCH', `/v1/kids/${IVY.id}`, { vaccine_schema: 'us' })).status).toBe(200);
	expect(
		(await ana('POST', `/v1/kids/${IVY.id}/sleep`, { started_at: minutesAgo(5) })).status
	).toBe(201);
	expect((await ana('POST', `/v1/kids/${IVY.id}/diapers`, { kind: 'wet' })).status).toBe(201);
	await signIn(page, ANA);
	await page.goto(`/kids/${IVY.id}`);
	await expect(running(page)).toBeVisible();
	await expect(today(page)).toBeVisible();
	const headings = await page.getByRole('heading', { level: 2 }).allTextContents();
	expect(headings.indexOf('Running now')).toBeGreaterThan(-1);
	expect(headings.indexOf('Running now')).toBeLessThan(headings.indexOf('Due'));
	expect(headings.indexOf('Due')).toBeLessThan(headings.indexOf('Today'));
});

test('Due shows at most three visits and links the rest from the Vaccines page', async ({
	page
}) => {
	await setScenario('home_due_many');
	await signIn(page, ANA);
	await page.goto(`/kids/${IVY.id}`);
	await expect(due(page).getByRole('listitem')).toHaveText([
		'1-year vaccines · Overdue',
		'2-year vaccines · Due',
		'3-year vaccines · Due',
		'2 more on the Vaccines page'
	]);
	const more = due(page).getByRole('link', { name: '2 more on the Vaccines page' });
	expect(await more.evaluate((a) => (a as HTMLAnchorElement).href)).toBe(
		`${UI_ORIGIN}/kids/${IVY.id}/vaccines`
	);
	await expectAccessible(page);
});

// A due entry is listed only before the kid-local day ends: stay an hour clear of UTC and Madrid midnight.
function skipNearDayEnd() {
	const left = (zone: string) => {
		const [h, m] = clock(new Date().toISOString(), zone).split(':').map(Number);
		return 24 * 60 - (h * 60 + m);
	};
	test.skip(Math.min(left('UTC'), left(LEO.timeZone)) < 60, 'entries would fall on tomorrow');
}
// A whole minute that many minutes from now, as the instant the API takes and the clock the page shows.
const inMinutes = (n: number) =>
	new Date(Math.floor(Date.now() / MS_PER_MINUTE) * MS_PER_MINUTE + n * MS_PER_MINUTE);
const instant = (d: Date) => d.toISOString().replace(/\.\d{3}Z$/, 'Z');
const hhmm = (d: Date) => instant(d).slice(11, 16);
const dailyCourse = (name: string, startAt: Date) => ({
	name,
	dose: '1 drop',
	schedule: { kind: 'every', hours: 24 },
	start_at: instant(startAt)
});

test('Due shows a medication entry with its time, then overdue as text, then drops it', async ({
	page
}) => {
	skipNearDayEnd();
	const ana = await stubAs(ANA);
	const at = inMinutes(10);
	expect((await ana('PATCH', `/v1/kids/${IVY.id}`, { reminder_overdue_minutes: 5 })).status).toBe(
		200
	);
	expect(
		(await ana('POST', `/v1/kids/${IVY.id}/medications`, dailyCourse('Vitamin D', at))).status
	).toBe(201);
	await signIn(page, ANA);
	await page.goto(`/kids/${IVY.id}`);
	await expect(due(page).getByRole('listitem')).toHaveText([`Vitamin D · ${hhmm(at)}`]);
	const link = due(page).getByRole('link');
	expect(await link.evaluate((a) => (a as HTMLAnchorElement).href)).toBe(
		`${UI_ORIGIN}/kids/${IVY.id}/medication`
	);
	await expectAccessible(page);

	// The stub clock is now two to three minutes past the due time, inside the kid's five.
	await moveStubClock(12 * 60);
	await page.reload();
	await expect(due(page).getByRole('listitem')).toHaveText([`Vitamin D · ${hhmm(at)} · overdue`]);
	await expectAccessible(page);

	await moveStubClock(16 * 60);
	await page.reload();
	await expect(page.getByRole('heading', { name: IVY.name, level: 1 })).toBeVisible();
	await expect(page.getByRole('heading', { name: 'Due' })).toHaveCount(0);
});

test('medication entries come before vaccine groups and the module off hides only them', async ({
	page
}) => {
	skipNearDayEnd();
	const ana = await stubAs(ANA);
	expect((await ana('PATCH', `/v1/kids/${IVY.id}`, { vaccine_schema: 'us' })).status).toBe(200);
	const at = inMinutes(10);
	expect(
		(await ana('POST', `/v1/kids/${IVY.id}/medications`, dailyCourse('Vitamin D', at))).status
	).toBe(201);
	await signIn(page, ANA);
	await page.goto(`/kids/${IVY.id}`);
	await expect(due(page).getByRole('listitem')).toHaveText([
		`Vitamin D · ${hhmm(at)}`,
		'8-week vaccines · Overdue',
		'12-month vaccines · Due'
	]);
	const hrefs = await due(page)
		.getByRole('link')
		.evaluateAll((as) => as.map((a) => (a as HTMLAnchorElement).href));
	expect(hrefs).toEqual([
		`${UI_ORIGIN}/kids/${IVY.id}/medication`,
		`${UI_ORIGIN}/kids/${IVY.id}/vaccines`,
		`${UI_ORIGIN}/kids/${IVY.id}/vaccines`
	]);
	await expectAccessible(page);

	expect(
		(await ana('PUT', `/v1/kids/${IVY.id}/modules/medication`, { enabled: false })).status
	).toBe(204);
	await page.goto(`/kids/${IVY.id}`);
	await expect(due(page).getByRole('listitem')).toHaveText([
		'8-week vaccines · Overdue',
		'12-month vaccines · Due'
	]);
	await expectAccessible(page);
});

test('Due shows three entries and one more line for each hidden kind, linked to its page', async ({
	page
}) => {
	skipNearDayEnd();
	const ana = await stubAs(ANA);
	const names = ['Alfa', 'Bravo', 'Charlie', 'Delta', 'Echo'];
	const first = inMinutes(3);
	const at = (i: number) => new Date(first.getTime() + i * MS_PER_MINUTE);
	for (const [i, name] of names.entries()) {
		const r = await ana('POST', `/v1/kids/${IVY.id}/medications`, dailyCourse(name, at(i)));
		expect(r.status).toBe(201);
	}
	expect((await ana('PATCH', `/v1/kids/${IVY.id}`, { vaccine_schema: 'us' })).status).toBe(200);
	await signIn(page, ANA);
	await page.goto(`/kids/${IVY.id}`);
	await expect(due(page).getByRole('listitem')).toHaveText([
		`Alfa · ${hhmm(at(0))}`,
		`Bravo · ${hhmm(at(1))}`,
		`Charlie · ${hhmm(at(2))}`,
		'2 more on the Medication page',
		'2 more on the Vaccines page'
	]);
	const medication = due(page).getByRole('link', { name: '2 more on the Medication page' });
	expect(await medication.evaluate((a) => (a as HTMLAnchorElement).href)).toBe(
		`${UI_ORIGIN}/kids/${IVY.id}/medication`
	);
	const vaccines = due(page).getByRole('link', { name: '2 more on the Vaccines page' });
	expect(await vaccines.evaluate((a) => (a as HTMLAnchorElement).href)).toBe(
		`${UI_ORIGIN}/kids/${IVY.id}/vaccines`
	);
	await expectAccessible(page);
});

test('a long unbroken medication name does not overflow the page at phone width', async ({
	page
}, testInfo) => {
	test.skip(testInfo.project.name !== 'phone', 'the overflow is a phone-width check');
	skipNearDayEnd();
	const ana = await stubAs(ANA);
	const name = 'w'.repeat(MEDICATION_TEXT_MAX_LEN);
	const at = inMinutes(10);
	expect((await ana('POST', `/v1/kids/${IVY.id}/medications`, dailyCourse(name, at))).status).toBe(
		201
	);
	await signIn(page, ANA);
	await page.goto(`/kids/${IVY.id}`);
	await expect(due(page).getByRole('listitem')).toHaveText([`${name} · ${hhmm(at)}`]);
	expect(
		await page.evaluate(
			() => document.documentElement.scrollWidth > document.documentElement.clientWidth
		)
	).toBe(false);
	await expectAccessible(page);
});

test("a caregiver sees the kid's medication entry at the kid's own clock time", async ({
	page
}) => {
	skipNearDayEnd();
	const cleo = await stubAs(CLEO);
	const at = inMinutes(10);
	expect(
		(await cleo('POST', `/v1/kids/${LEO.id}/medications`, dailyCourse('Vitamin D', at))).status
	).toBe(201);
	await signIn(page, ANA);
	await page.goto(`/kids/${LEO.id}`);
	const madrid = clock(instant(at), LEO.timeZone);
	await expect(due(page).getByRole('listitem')).toHaveText([`Vitamin D · ${madrid}`]);
	await expectAccessible(page);
});

test('one tap on Sleep starts a timer; End returns to the normal buttons', async ({ page }) => {
	await signIn(page, ANA);
	await quickLog(page).getByRole('button', { name: 'Sleep' }).click();
	await expect(running(page).getByRole('listitem')).toHaveText([
		/^\s*Asleep since \d{2}:\d{2}\s+End\s*$/
	]);
	await expect(quickLog(page).getByRole('listitem')).toHaveText([
		'Bottle',
		'Nursing',
		'Diaper',
		'Pumping',
		'Feeding',
		'Milestones',
		'Growth',
		'Vaccination',
		'Sickness',
		'Medication'
	]);
	await expectAccessible(page);
	const posts = (await receivedRequests()).filter(
		(r) => r.method === 'POST' && r.path === `/v1/kids/${MIA.id}/sleep`
	);
	expect(posts).toHaveLength(1);

	await running(page).getByRole('button', { name: 'End sleep' }).click();
	await expect(running(page)).toHaveCount(0);
	await expect(quickLog(page).getByRole('listitem')).toHaveText([
		'Bottle',
		'Nursing',
		'Diaper',
		'Sleep',
		'Pumping',
		'Feeding',
		'Milestones',
		'Growth',
		'Vaccination',
		'Sickness',
		'Medication'
	]);
	await expect(today(page).getByRole('term')).toHaveText([
		'Bottles',
		'Nursing',
		'Diapers',
		'Sleep'
	]);
	await expectAccessible(page);
});

test.describe('without JavaScript', () => {
	test.use({ javaScriptEnabled: false });

	test('Sleep and End are plain form posts', async ({ page }) => {
		await signIn(page, ANA);
		await quickLog(page).getByRole('button', { name: 'Sleep' }).click();
		await expect(running(page).getByText(/^Asleep since \d{2}:\d{2}$/)).toBeVisible();
		await running(page).getByRole('button', { name: 'End sleep' }).click();
		await expect(running(page)).toHaveCount(0);
		await expect(quickLog(page).getByRole('button', { name: 'Sleep' })).toBeVisible();
	});

	test('modules saves, and shows an off-by-default module as off', async ({ page }) => {
		await signIn(page, CLEO);
		await page.goto(`/kids/${LEO.id}/modules`);
		const bottle = page.getByRole('checkbox', { name: 'Bottle', exact: true });
		await expect(bottle).not.toBeChecked();
		await expect(bottle).toHaveAccessibleDescription('Off by default for this age');
		await expect(page.getByRole('checkbox', { name: 'Growth', exact: true })).toBeChecked();
		await bottle.check();
		await page.getByRole('button', { name: 'Save' }).click();
		await expect(page.getByRole('status')).toHaveText('Saved');
		await page.goto(`/kids/${LEO.id}`);
		await expect(quickLog(page).getByRole('link', { name: 'Bottle' })).toHaveAttribute(
			'href',
			`/kids/${LEO.id}/log/bottles`
		);
	});
});

test('a running timer shows its start in the kid’s zone', async ({ page }) => {
	const started = minutesAgo(120);
	const ana = await stubAs(ANA);
	expect((await ana('POST', `/v1/kids/${LEO.id}/sleep`, { started_at: started })).status).toBe(201);
	await signIn(page, ANA);
	await page.goto(`/kids/${LEO.id}`);
	await expect(
		running(page).getByText(`Asleep since ${clock(started, LEO.timeZone)}`)
	).toBeVisible();
	await expectAccessible(page);
});

test('a nursing timer: side and start, switch side, end', async ({ page }) => {
	skipNearMidnight(15);
	const started = minutesAgo(10);
	const ana = await stubAs(ANA);
	const made = await ana('POST', `/v1/kids/${MIA.id}/nursing`, {
		started_at: started,
		side: 'left'
	});
	expect(made.status).toBe(201);
	await signIn(page, ANA);
	const timer = running(page).getByRole('listitem');
	await expect(timer).toContainText(`Nursing, left since ${clock(started, MIA.timeZone)}`);
	await expect(quickLog(page).getByRole('link', { name: 'Nursing' })).toHaveCount(0);
	await expectAccessible(page);

	await timer.getByRole('button', { name: 'Switch side' }).click();
	await expect(timer).toContainText(/Nursing, right since \d{2}:\d{2}/);
	await expect(timer).not.toContainText('Nursing, left');

	await timer.getByRole('button', { name: 'End nursing' }).click();
	await expect(running(page)).toHaveCount(0);
	await expect(quickLog(page).getByRole('link', { name: 'Nursing' })).toBeVisible();
	await expect(today(page).getByRole('definition').nth(1)).toContainText(/^1 · \d+ min$/);
	await expectAccessible(page);
});

test('a timer that ran over 24 hours asks for an end time', async ({ page }) => {
	const ana = await stubAs(ANA);
	const made = await ana('POST', `/v1/kids/${MIA.id}/sleep`, { started_at: minutesAgo(25 * 60) });
	expect(made.status).toBe(201);
	await signIn(page, ANA);
	await running(page).getByRole('button', { name: 'End sleep' }).click();
	const alert = running(page).getByRole('alert');
	await expect(alert).toHaveText('This timer ran over 24 hours: set the end time');
	await expect(alert.getByRole('link', { name: 'set the end time' })).toHaveAttribute(
		'href',
		`/kids/${MIA.id}/log/sleep/${made.body.id}`
	);
	await expect(running(page).getByText(/^Asleep since/)).toBeVisible();
	await expectAccessible(page);
});

test('Today shows the day in the parent’s units', async ({ page }) => {
	skipNearMidnight(40);
	const ana = await stubAs(ANA);
	const kid = `/v1/kids/${MIA.id}`;
	await ana('POST', `${kid}/bottles`, {
		contents: ['formula'],
		volume: { unit: 'ml', amount: 120 }
	});
	for (const kind of ['wet', 'wet', 'dirty']) await ana('POST', `${kid}/diapers`, { kind });
	await ana('POST', `${kid}/nursing`, {
		started_at: minutesAgo(30),
		ended_at: minutesAgo(10),
		left_seconds: 600,
		right_seconds: 600
	});
	await ana('POST', `${kid}/sleep`, { started_at: minutesAgo(25), ended_at: minutesAgo(5) });

	await signIn(page, ANA);
	await expect(today(page).getByRole('term')).toHaveText([
		'Bottles',
		'Nursing',
		'Diapers',
		'Sleep'
	]);
	await expect(today(page).getByRole('definition')).toHaveText([
		'1 · 120 ml',
		'1 · 20 min',
		'3 (2 wet, 1 dirty)',
		'20 min'
	]);
	await expectAccessible(page);

	await page.goto('/account');
	await untilHydrated(page.getByLabel('Display name'));
	await page.getByLabel('Volume').selectOption({ label: 'fl oz' });
	await page.getByRole('button', { name: 'Save' }).click();
	await expect(page.getByRole('status')).toHaveText('Saved');
	await page.goto(`/kids/${MIA.id}`);
	await expect(today(page).getByRole('definition').first()).toHaveText('1 · 4.06 fl oz');
});

test('a guardian turns a module off and on; the data stays', async ({ page }) => {
	const ana = await stubAs(ANA);
	const bottle = await ana('POST', `/v1/kids/${MIA.id}/bottles`, {
		contents: ['formula'],
		volume: { unit: 'ml', amount: 90 }
	});
	await signIn(page, ANA);
	await expect(page.getByRole('link', { name: `Modules for ${MIA.name}` })).toHaveAttribute(
		'href',
		`/kids/${MIA.id}/modules`
	);
	await page.getByRole('link', { name: `Modules for ${MIA.name}` }).click();
	await untilHydrated(page.getByRole('checkbox', { name: 'Diaper', exact: true }), 'checked');
	await expect(
		page.getByRole('heading', { name: `Modules for ${MIA.name}`, level: 1 })
	).toBeVisible();
	const box = (name: string) => page.getByRole('checkbox', { name, exact: true });
	await expect(page.getByRole('checkbox')).toHaveCount(11);
	await expect(box('Diaper')).toBeChecked();
	await expect(box('Diaper')).toHaveAccessibleDescription('On by default for this age');
	await expectAccessible(page);

	await box('Diaper').uncheck();
	await page.getByRole('button', { name: 'Save' }).click();
	await expect(page.getByRole('status')).toHaveText('Saved');
	await expect(box('Diaper')).not.toBeChecked();
	await expect(box('Diaper')).toHaveAccessibleDescription('On by default for this age');
	await expectAccessible(page);

	await page.goto(`/kids/${MIA.id}`);
	await expect(quickLog(page).getByRole('listitem')).toHaveText([
		'Bottle',
		'Nursing',
		'Sleep',
		'Pumping',
		'Feeding',
		'Milestones',
		'Growth',
		'Vaccination',
		'Sickness',
		'Medication'
	]);
	// The entry stays in the data: only the default timeline view follows the modules.
	const kept = await ana('GET', `/v1/kids/${MIA.id}/timeline?types=bottle`);
	expect(kept.body.items).toHaveLength(1);
	expect(bottle.status).toBe(201);

	await page.goto(`/kids/${MIA.id}/modules`);
	await untilHydrated(page.getByRole('checkbox', { name: 'Bottle', exact: true }), 'checked');
	await box('Diaper').check();
	await page.getByRole('button', { name: 'Save' }).click();
	await expect(page.getByRole('status')).toHaveText('Saved');
	await page.goto(`/kids/${MIA.id}`);
	await expect(quickLog(page).getByRole('link', { name: 'Diaper' })).toBeVisible();
});

test('a caregiver sees no Modules link and the modules page is the 404 page', async ({ page }) => {
	await signIn(page, ANA);
	await page.goto(`/kids/${LEO.id}`);
	await expect(page.getByRole('heading', { name: LEO.name, level: 1 })).toBeVisible();
	await expect(page.getByRole('link', { name: /Modules/ })).toHaveCount(0);
	const forbidden = await expectNotFoundPage(page, `/kids/${LEO.id}/modules`);
	const unknown = await expectNotFoundPage(page, '/kids/k-unknown/modules');
	expect(forbidden).toBe(unknown);
	await expectNotFoundPage(page, '/kids/k-mia/modules/x');
});

test('a caregiver cannot change a module by posting to the page', async ({ page }) => {
	await signIn(page, ANA);
	const response = await page.request.post(`/kids/${LEO.id}/modules?/save`, {
		form: { module: 'growth' },
		headers: { origin: UI_ORIGIN }
	});
	expect(response.status()).toBe(404);
	const ana = await stubAs(ANA);
	const modules = (await ana('GET', `/v1/kids/${LEO.id}/modules`)).body.modules as {
		module: string;
		enabled: boolean;
	}[];
	// Sickness is on for a kid this age and was not posted: it must not have been switched off.
	expect(modules.find((m) => m.module === 'sickness')?.enabled).toBe(true);
});

test('/ is a 303 to the remembered kid, else the first; no kids keeps the welcome', async ({
	page,
	context
}) => {
	await signIn(page, ANA);
	const first = await page.request.get('/', { maxRedirects: 0 });
	expect(first.status()).toBe(303);
	expect(first.headers()['location']).toBe(`/kids/${MIA.id}`);
	expect(first.headers()['cache-control']).toBe('no-store');

	await page.goto(`/kids/${LEO.id}`);
	expect((await context.cookies()).find((c) => c.name === KID_COOKIE)?.value).toBe(LEO.id);
	const second = await page.request.get('/', { maxRedirects: 0 });
	expect(second.status()).toBe(303);
	expect(second.headers()['location']).toBe(`/kids/${LEO.id}`);

	await context.clearCookies();
	await signIn(page, DAN);
	await expect(page.getByRole('link', { name: 'Add your first child' })).toBeVisible();
	await expectLanding(page, { welcome: DAN.displayName });
	expect((await page.request.get('/', { maxRedirects: 0 })).status()).toBe(200);
	await expectAccessible(page);
});

test('/ with a remembered kid the parent cannot see is a 303 to the first kid', async ({
	page,
	context
}) => {
	await signIn(page, CLEO);
	await context.addCookies([{ name: KID_COOKIE, value: MIA.id, url: UI_ORIGIN }]);
	const response = await page.request.get('/', { maxRedirects: 0 });
	expect(response.status()).toBe(303);
	expect(response.headers()['location']).toBe(`/kids/${LEO.id}`);
});

test('/timeline is a 303 to the current kid’s timeline, or home without kids', async ({
	page,
	context
}) => {
	await signIn(page, ANA);
	await expect(page.getByRole('link', { name: 'Timeline' })).toHaveAttribute('href', '/timeline');
	const first = await page.request.get('/timeline', { maxRedirects: 0 });
	expect(first.status()).toBe(303);
	expect(first.headers()['location']).toBe(`/kids/${MIA.id}/timeline`);

	await page.goto(`/kids/${LEO.id}`);
	const second = await page.request.get('/timeline', { maxRedirects: 0 });
	expect(second.headers()['location']).toBe(`/kids/${LEO.id}/timeline`);
	await page.getByRole('link', { name: 'Timeline' }).click();
	await expect(page).toHaveURL(`/kids/${LEO.id}/timeline`);
	await expect(page.getByRole('heading', { name: 'Timeline for Leo', level: 1 })).toBeVisible();

	await context.clearCookies();
	await signIn(page, DAN);
	const none = await page.request.get('/timeline', { maxRedirects: 0 });
	expect(none.status()).toBe(303);
	expect(none.headers()['location']).toBe('/');
});

test('/growth is a 303 to the current kid’s growth page, or home without kids', async ({
	page,
	context
}) => {
	await signIn(page, ANA);
	await expect(mainNav(page).getByRole('link', { name: 'Growth', exact: true })).toHaveAttribute(
		'href',
		'/growth'
	);
	const first = await page.request.get('/growth', { maxRedirects: 0 });
	expect(first.status()).toBe(303);
	expect(first.headers()['location']).toBe(`/kids/${MIA.id}/growth`);

	await page.goto(`/kids/${LEO.id}`);
	const second = await page.request.get('/growth', { maxRedirects: 0 });
	expect(second.headers()['location']).toBe(`/kids/${LEO.id}/growth`);
	await mainNav(page).getByRole('link', { name: 'Growth', exact: true }).click();
	await expect(page).toHaveURL(`/kids/${LEO.id}/growth`);
	await expect(page.getByRole('heading', { name: 'Growth for Leo', level: 1 })).toBeVisible();
	await expect(mainNav(page).getByRole('link', { name: 'Growth', exact: true })).toHaveAttribute(
		'aria-current',
		'page'
	);

	await context.clearCookies();
	await signIn(page, DAN);
	const none = await page.request.get('/growth', { maxRedirects: 0 });
	expect(none.status()).toBe(303);
	expect(none.headers()['location']).toBe('/');
});

test('the kid home is not reachable signed out and sets no-store', async ({ page, request }) => {
	const signedOut = await request.get(`/kids/${MIA.id}`, { maxRedirects: 0 });
	expect(signedOut.status()).toBe(303);
	expect(signedOut.headers()['location']).toContain('/login');
	await signIn(page, ANA);
	const response = await page.request.get(`/kids/${MIA.id}`);
	expect(response.headers()['cache-control']).toBe('no-store');
});
