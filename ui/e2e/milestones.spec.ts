import { expect, test, type Locator, type Page } from '@playwright/test';
import { MILESTONE_TEXT_MAX, NOTE_MAX_LEN } from '../src/lib/constants.js';
import { todayIn } from '../src/lib/format.js';
import { reachedLabel } from '../src/lib/milestones.js';
import {
	expectAccessible,
	expectFieldError,
	expectNotFoundPage,
	navLink,
	signIn,
	submit,
	untilHydrated
} from './helpers.js';
import {
	ANA,
	CLEO,
	DAN,
	NELL,
	OTTO,
	PIA,
	REMY,
	SAMI,
	moveStubClock,
	receivedRequests,
	resetStub,
	stubAs
} from './stub-api.js';

// The Milestones page, its one-tap Reached form, the create and edit pages. The stub catalog has one item per age
// and domain ("Stub item 9m-se-1"); Cleo guards the milestone kids alone. Every save waits with submit().

test.beforeEach(async () => {
	await resetStub();
});

type Api = Awaited<ReturnType<typeof stubAs>>;
type Row = Record<string, unknown>;

const DAY_S = 24 * 3600;
const MS_PER_DAY = 86_400_000;
const TODAY = todayIn('UTC');
const addDays = (date: string, n: number) =>
	new Date(Date.parse(`${date}T00:00:00Z`) + n * MS_PER_DAY).toISOString().slice(0, 10);
const TITLES = [
	'By 2 months',
	'By 4 months',
	'By 6 months',
	'By 9 months',
	'By 1 year',
	'By 15 months',
	'By 18 months',
	'By 2 years',
	'By 30 months',
	'By 3 years',
	'By 4 years',
	'By 5 years'
];
const CURRENT = 'Current list for this age';
const NOTE_OFF = (name: string) => `Milestones are off for ${name}.`;
const DENYLIST = [
	'late',
	'behind',
	'delayed',
	'delay',
	'normal',
	'abnormal',
	'typical',
	'atypical',
	'on track',
	'ahead',
	'missed',
	'missing',
	'concern',
	'worry',
	'should',
	'red flag',
	'warning'
];

const pageUrl = (kid: string) => `/kids/${kid}/milestones`;
const newUrl = (kid: string, item?: string) => `${pageUrl(kid)}/new${item ? `?item=${item}` : ''}`;
const editUrl = (kid: string, id: unknown) => `${pageUrl(kid)}/${id}`;
const apiPath = (kid: string, id?: unknown) => `/v1/kids/${kid}/milestones${id ? `/${id}` : ''}`;
const itemText = (id: string) => {
	const [age, domain] = id.split('-');
	return `Stub item ${age}-${domain}-1`;
};

const summaries = (page: Page) => page.locator('main details > summary');
const list = (page: Page, title: string) =>
	page.locator('main details').filter({
		has: page.locator(':scope > summary', { hasText: new RegExp(`^${title}( ·|\\s*$)`) })
	});
const item = (scope: Page | Locator, id: string) =>
	scope.getByRole('listitem').filter({ hasText: itemText(id) });
const reachButton = (scope: Page | Locator, id: string) =>
	scope.getByRole('button', { name: `Reached ${itemText(id)}`, exact: true });
const editLink = (scope: Page | Locator, text: string) =>
	scope.getByRole('link', { name: `Edit ${text}`, exact: true });
const field = (page: Page, label: string) => page.getByLabel(label, { exact: true });
const save = (page: Page) => submit(page.getByRole('button', { name: 'Save', exact: true }));
const custom = (page: Page) => page.getByRole('region', { name: 'Your milestones' });
const quickLog = (page: Page) => page.getByRole('region', { name: 'Quick log' });
const overflows = (page: Page) =>
	page.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth);

// Hydration can close a list opened just before it: click again until it stays open.
async function open(page: Page, title: string) {
	const details = list(page, title);
	await expect(async () => {
		if ((await details.getAttribute('open')) === null) {
			await details.locator(':scope > summary').click();
		}
		await expect(details).toHaveAttribute('open', '', { timeout: 1000 });
	}).toPass();
	return details;
}
async function onPage(page: Page, kid: { id: string; name: string }) {
	await page.waitForURL((u) => u.pathname === pageUrl(kid.id) && u.search === '');
	await expect(
		page.getByRole('heading', { name: 'Milestones', level: 1, exact: true })
	).toBeVisible();
}
// A one-tap mark returns to the item: its list named in the query, the row in the fragment.
async function onItem(page: Page, kid: { id: string; name: string }, id: string) {
	await page.waitForURL(
		(u) => u.pathname === pageUrl(kid.id) && u.search === `?open=${id}` && u.hash === `#item-${id}`
	);
	await expect(
		page.getByRole('heading', { name: 'Milestones', level: 1, exact: true })
	).toBeVisible();
}
const stored = async (api: Api, kid: string) => {
	const v = (await api('GET', apiPath(kid))).body;
	const items = (v.checklists as { items: { id: string; reached: Row | null }[] }[]).flatMap(
		(c) => c.items
	);
	return { view: v, reached: (id: string) => items.find((i) => i.id === id)?.reached ?? null };
};
async function seed(api: Api, kid: string, body: Row) {
	const r = await api('POST', apiPath(kid), { reached_on: TODAY, ...body });
	expect(r.status).toBe(201);
	return r.body as Row;
}
const written = async (method: string, path: string) => {
	const writes = (await receivedRequests()).filter((r) => r.method === method && r.path === path);
	return JSON.parse(writes.at(-1)?.body ?? 'null') as Row;
};
async function caregiverOf(kid: string) {
	const cleo = await stubAs(CLEO);
	const r = await cleo('PUT', `/v1/kids/${kid}/access/${ANA.username}`, { role: 'caregiver' });
	expect(r.status).toBeLessThan(300);
	return stubAs(ANA);
}
// The page's own words: the stub catalog items (CDC's text in production) are left out.
async function ownWords(page: Page) {
	const text = await page.locator('main').innerText();
	return text.replace(/Stub item \S+/g, '');
}

test('a 9-month kid sees all twelve lists in age order, the current one open and marked', async ({
	page
}) => {
	await stubAs(CLEO);
	await signIn(page, CLEO);
	await page.goto(pageUrl(NELL.id));
	await onPage(page, NELL);
	await expect(summaries(page)).toHaveText(
		TITLES.map((t) => (t === 'By 9 months' ? `${t} · ${CURRENT}` : t))
	);
	await expect(list(page, 'By 9 months')).toHaveAttribute('open', '');
	await expect(page.locator('main details[open]')).toHaveCount(1);
	await expect(page.getByText(CURRENT)).toHaveCount(1);
	await expect(page.getByText('Next list')).toHaveCount(0);
	// Grouped by domain, each item with its text and the two ways to mark it.
	const nine = list(page, 'By 9 months');
	await expect(nine.getByRole('heading', { level: 3 })).toHaveText([
		'Social/Emotional',
		'Language/Communication',
		'Cognitive (learning, thinking, problem-solving)',
		'Movement/Physical Development'
	]);
	await expect(reachButton(nine, '9m-se-stub-1')).toBeVisible();
	await expect(
		nine.getByRole('link', { name: `Choose a date for ${itemText('9m-lc-stub-1')}` })
	).toHaveAttribute('href', newUrl(NELL.id, '9m-lc-stub-1'));
	await expect(nine).not.toContainText('Reached on');
	// No colour, icon, count or progress: nothing but the absence of a date marks an unreached item.
	// The chevrons and the Edit and Choose a date buttons are controls, marked data-chrome.
	await expect(
		page.locator('main').locator('svg:not([data-chrome]), img, progress, meter')
	).toHaveCount(0);
	await expectAccessible(page);

	await expect(reachButton(page, '30m-se-stub-1')).toHaveCount(0);
	await open(page, 'By 30 months');
	await expect(reachButton(page, '30m-se-stub-1')).toBeVisible();

	const jump = page.getByRole('link', { name: 'Go to the current list' });
	await expect(jump).toHaveAttribute('href', '#list-9');
	await page.mouse.wheel(0, 10_000);
	await jump.click();
	await expect(page).toHaveURL(new RegExp(`${pageUrl(NELL.id)}#list-9$`));
	await expect(list(page, 'By 9 months').locator(':scope > summary')).toBeInViewport();
});

test('the CDC sentence and attribution from the API sit under the lists', async ({ page }) => {
	await stubAs(CLEO);
	await signIn(page, CLEO);
	await page.goto(pageUrl(NELL.id));
	const note = page.getByText('Stub screening note.', { exact: true });
	const attribution = page.getByText('Stub attribution, not reference data.', { exact: true });
	await expect(note).toBeVisible();
	await expect(attribution).toBeVisible();
	// Beside the lists from 1280 px: after them in the document, so also in the reading order.
	const last = (await summaries(page).last().elementHandle())!;
	for (const text of [note, attribution]) {
		expect(
			await text.evaluate(
				(el, ref) => !!(ref.compareDocumentPosition(el) & Node.DOCUMENT_POSITION_FOLLOWING),
				last
			)
		).toBe(true);
	}
	await expect(page.locator('main')).not.toContainText('omitted');
});

test('an item in a later and one in an earlier list are each marked reached', async ({ page }) => {
	const cleo = await stubAs(CLEO);
	await signIn(page, CLEO);
	await page.goto(pageUrl(NELL.id));
	for (const [title, id] of [
		['By 3 years', '36m-cg-stub-1'],
		['By 2 months', '2m-mp-stub-1']
	]) {
		await submit(reachButton(await open(page, title), id));
		await onItem(page, NELL, id);
		await expect(list(page, title)).toHaveAttribute('open', '');
		const row = item(list(page, title), id);
		await expect(row).toBeInViewport();
		await expect(row).toContainText(reachedLabel(TODAY));
		await expect(reachButton(row, id)).toHaveCount(0);
		expect((await stored(cleo, NELL.id)).reached(id)).toMatchObject({ reached_on: TODAY });
	}
	// The lists the reader opened stay open across both marks.
	for (const title of ['By 3 years', 'By 2 months', 'By 9 months']) {
		await expect(list(page, title)).toHaveAttribute('open', '');
	}
});

test('the one-tap Reached form records today and shows the date', async ({ page }) => {
	const cleo = await stubAs(CLEO);
	await signIn(page, CLEO);
	await page.goto(pageUrl(NELL.id));
	await submit(reachButton(page, '9m-se-stub-1'));
	await onItem(page, NELL, '9m-se-stub-1');
	expect(await written('POST', apiPath(NELL.id))).toEqual({
		item: '9m-se-stub-1',
		reached_on: TODAY
	});
	const row = item(page, '9m-se-stub-1');
	await expect(row).toContainText(reachedLabel(TODAY));
	const reached = (await stored(cleo, NELL.id)).reached('9m-se-stub-1');
	await expect(editLink(row, itemText('9m-se-stub-1'))).toHaveAttribute(
		'href',
		editUrl(NELL.id, reached!.id)
	);
	await expect(reachButton(page, '9m-lc-stub-1')).toBeVisible();
	await expectAccessible(page);
});

test('a refused one-tap mark shows its message in the item’s row', async ({ page }) => {
	const cleo = await stubAs(CLEO);
	const id = '9m-mp-stub-1';
	await signIn(page, CLEO);
	await page.goto(pageUrl(NELL.id));
	await seed(cleo, NELL.id, { item: id });
	await submit(reachButton(page, id));
	await expect(item(page, id).getByRole('alert')).toHaveText('Already marked as reached');
	await expect(page.getByRole('alert')).toHaveCount(1);
	await expectAccessible(page);
});

test('a kid at 4 years 11 months: 4 years current, 5 years reachable; a retired row keeps its text', async ({
	page
}) => {
	await stubAs(CLEO);
	await signIn(page, CLEO);
	await page.goto(pageUrl(OTTO.id));
	await expect(summaries(page)).toHaveCount(12);
	await expect(list(page, 'By 4 years')).toHaveAttribute('open', '');
	await expect(list(page, 'By 4 years').locator(':scope > summary')).toHaveText(
		`By 4 years · ${CURRENT}`
	);
	await expect(list(page, 'By 5 years').locator(':scope > summary')).toHaveText('By 5 years');
	await submit(reachButton(await open(page, 'By 5 years'), '60m-se-stub-1'));
	await onItem(page, OTTO, '60m-se-stub-1');
	await expect(item(await open(page, 'By 5 years'), '60m-se-stub-1')).toContainText(
		reachedLabel(TODAY)
	);
	const retired = page.getByRole('region', { name: 'Reached, no longer in the lists' });
	await expect(retired.getByRole('listitem')).toHaveCount(1);
	await expect(retired).toContainText('Stub item 4m-cg-retired');
	await expect(editLink(retired, 'Stub item 4m-cg-retired')).toHaveAttribute(
		'href',
		editUrl(OTTO.id, 'ms-retired')
	);
	await expectAccessible(page);
});

test('on the 5th birthday there are no lists: the reached items and the custom rows are listed', async ({
	page
}) => {
	const cleo = await stubAs(CLEO);
	await seed(cleo, PIA.id, { item: '60m-lc-stub-1' });
	await seed(cleo, PIA.id, { text: 'Rides a bike' });
	await signIn(page, CLEO);
	await page.goto(pageUrl(PIA.id));
	await onPage(page, PIA);
	await expect(summaries(page)).toHaveCount(0);
	await expect(page.getByRole('link', { name: 'Go to the current list' })).toHaveCount(0);
	const reached = page.getByRole('region', { name: 'Reached', exact: true });
	await expect(reached.getByRole('listitem')).toHaveCount(1);
	for (const text of [
		'Stub item 60m-lc-1',
		'By 5 years',
		'Language/Communication',
		reachedLabel(TODAY)
	]) {
		await expect(reached).toContainText(text);
	}
	await expect(custom(page).getByRole('listitem')).toHaveCount(1);
	await expect(custom(page)).toContainText('Rides a bike');
	await expect(custom(page)).toContainText(reachedLabel(TODAY));
	await expect(page.getByText('Stub attribution, not reference data.')).toBeVisible();
	await expectAccessible(page);
});

test('a 32-week kid at 3 months shows both ages and the 2-month list as the next one', async ({
	page
}) => {
	await stubAs(CLEO);
	await signIn(page, CLEO);
	await page.goto(pageUrl(REMY.id));
	await onPage(page, REMY);
	await expect(page.getByText(/^3 months.*\(corrected 1 month/)).toBeVisible();
	await expect(list(page, 'By 2 months')).toHaveAttribute('open', '');
	await expect(list(page, 'By 2 months').locator(':scope > summary')).toHaveText(
		'By 2 months · Next list'
	);
	await expect(page.locator('main details[open]')).toHaveCount(1);
	await expect(page.getByText(CURRENT)).toHaveCount(0);
	await expect(page.getByRole('link', { name: 'Go to the next list' })).toHaveAttribute(
		'href',
		'#list-2'
	);
	await expectAccessible(page);
});

test('the 32-week kid at 5 months has the 2-month list current by corrected age', async ({
	page
}) => {
	await moveStubClock(62 * DAY_S);
	await stubAs(CLEO);
	await signIn(page, CLEO);
	await page.goto(pageUrl(REMY.id));
	await expect(list(page, 'By 2 months').locator(':scope > summary')).toHaveText(
		`By 2 months · ${CURRENT}`
	);
	await expect(list(page, 'By 4 months').locator(':scope > summary')).toHaveText('By 4 months');
	await expect(page.getByRole('link', { name: 'Go to the current list' })).toHaveAttribute(
		'href',
		'#list-2'
	);
});

test('a 30-month kid with the module off: the page loads from the kid links, the tile follows the first mark', async ({
	page
}) => {
	await stubAs(CLEO);
	await signIn(page, CLEO);
	await page.goto(`/kids/${SAMI.id}`);
	await expect(page.getByRole('heading', { name: SAMI.name, level: 1 })).toBeVisible();
	await expect(quickLog(page).getByRole('link', { name: 'Milestones', exact: true })).toHaveCount(
		0
	);
	// A switched-off module has no navigation entry; its page is still reachable.
	await expect(await navLink(page, 'Milestones')).toHaveCount(0);
	await page.goto(pageUrl(SAMI.id));
	await onPage(page, SAMI);
	await expect(page.getByText(NOTE_OFF(SAMI.name))).toBeVisible();
	await expect(summaries(page)).toHaveCount(12);
	await expect(list(page, 'By 30 months')).toHaveAttribute('open', '');
	await page.goto(`/kids/${SAMI.id}/modules`);
	await expect(page.getByLabel('Milestones', { exact: true })).not.toBeChecked();

	await page.goto(pageUrl(SAMI.id));
	await submit(reachButton(page, '30m-mp-stub-1'));
	await onItem(page, SAMI, '30m-mp-stub-1');
	await expect(item(page, '30m-mp-stub-1')).toContainText(reachedLabel(TODAY));
	await expect(page.getByText(NOTE_OFF(SAMI.name))).toHaveCount(0);
	await page.goto(`/kids/${SAMI.id}`);
	await expect(
		quickLog(page).getByRole('link', { name: 'Milestones', exact: true })
	).toHaveAttribute('href', pageUrl(SAMI.id));
});

for (const [role, guardian] of [
	['guardian', true],
	['caregiver', false]
] as const) {
	test(`with the module off a ${role} sees the note ${guardian ? 'with' : 'without'} the Modules link`, async ({
		page
	}) => {
		const who = guardian ? CLEO : ANA;
		if (guardian) await stubAs(CLEO);
		else await caregiverOf(SAMI.id);
		await signIn(page, who);
		await page.goto(pageUrl(SAMI.id));
		await onPage(page, SAMI);
		await expect(page.getByText(NOTE_OFF(SAMI.name))).toBeVisible();
		const link = page.getByRole('link', { name: 'Turn it on in Modules' });
		await expect(link).toHaveCount(guardian ? 1 : 0);
		await expect(page.locator('main')).not.toContainText(/remind/i);
		await expectAccessible(page);
		if (guardian) {
			await link.click();
			await expect(page).toHaveURL(`/kids/${SAMI.id}/modules`);
		}
	});
}

test('the page has no off note while the module is on', async ({ page }) => {
	await stubAs(CLEO);
	await signIn(page, CLEO);
	await page.goto(pageUrl(NELL.id));
	await onPage(page, NELL);
	await expect(page.getByText(/Milestones are off/)).toHaveCount(0);
	await expect(page.getByRole('link', { name: 'Turn it on in Modules' })).toHaveCount(0);
});

test('a chosen past date is stored, read back and edited; the edit sends only the change', async ({
	page
}) => {
	const cleo = await stubAs(CLEO);
	const id = '9m-lc-stub-1';
	const [past, earlier] = [addDays(TODAY, -3), addDays(TODAY, -5)];
	await signIn(page, CLEO);
	await page.goto(pageUrl(NELL.id));
	await item(page, id)
		.getByRole('link', { name: `Choose a date for ${itemText(id)}` })
		.click();
	await expect(page.getByRole('heading', { name: 'Mark as reached', level: 1 })).toBeVisible();
	await expect(page.getByText(itemText(id), { exact: true })).toBeVisible();
	await expect(field(page, 'Date reached')).toHaveValue(TODAY);
	await expectAccessible(page);
	await untilHydrated(field(page, 'Date reached'));
	await field(page, 'Date reached').fill(past);
	await field(page, 'Note').fill('  at the park ');
	await save(page);
	await onPage(page, NELL);
	expect(await written('POST', apiPath(NELL.id))).toEqual({
		item: id,
		reached_on: past,
		note: 'at the park'
	});
	const row = item(page, id);
	await expect(row).toContainText(reachedLabel(past));

	await editLink(row, itemText(id)).click();
	await expect(page.getByRole('heading', { name: 'Edit milestone', level: 1 })).toBeVisible();
	await expect(page.getByText(itemText(id), { exact: true })).toBeVisible();
	await expect(field(page, 'Milestone')).toHaveCount(0);
	await expect(field(page, 'Date reached')).toHaveValue(past);
	await expect(field(page, 'Note')).toHaveValue('at the park');
	await expectAccessible(page);
	await untilHydrated(field(page, 'Date reached'));
	await untilHydrated(field(page, 'Note'));
	await field(page, 'Date reached').fill(earlier);
	await save(page);
	await onPage(page, NELL);
	const entry = (await stored(cleo, NELL.id)).reached(id);
	expect(await written('PATCH', apiPath(NELL.id, entry!.id))).toEqual({ reached_on: earlier });
	await expect(item(page, id)).toContainText(reachedLabel(earlier));
});

test('a future date shows the field message and keeps the values', async ({ page }) => {
	await stubAs(CLEO);
	const id = '9m-cg-stub-1';
	await signIn(page, CLEO);
	await page.goto(newUrl(NELL.id, id));
	await untilHydrated(field(page, 'Date reached'));
	await field(page, 'Date reached').fill(addDays(TODAY, 1));
	await field(page, 'Note').fill('kept');
	await save(page);
	await expectFieldError(
		field(page, 'Date reached'),
		'Enter a date that is not in the future and not before the birth date'
	);
	await expect(field(page, 'Date reached')).toHaveValue(addDays(TODAY, 1));
	await expect(field(page, 'Note')).toHaveValue('kept');
	await expect(page.getByText(itemText(id), { exact: true })).toBeVisible();
	await expectAccessible(page);
	// Correcting the date saves the same item.
	await field(page, 'Date reached').fill(TODAY);
	await save(page);
	await onPage(page, NELL);
	await expect(item(page, id)).toContainText(reachedLabel(TODAY));
});

test('an item marked meanwhile says it is already reached', async ({ page }) => {
	const cleo = await stubAs(CLEO);
	const id = '9m-mp-stub-1';
	await signIn(page, CLEO);
	await page.goto(newUrl(NELL.id, id));
	await seed(cleo, NELL.id, { item: id });
	await save(page);
	await expect(page.getByRole('alert')).toHaveText('Already marked as reached');
	await expect(field(page, 'Date reached')).toHaveValue(TODAY);
});

test('undo deletes the mark and returns the item to its list', async ({ page }) => {
	const cleo = await stubAs(CLEO);
	const id = '9m-se-stub-1';
	const row = await seed(cleo, NELL.id, { item: id, reached_on: addDays(TODAY, -1) });
	await signIn(page, CLEO);
	await page.goto(editUrl(NELL.id, row.id));
	await page.getByText('Undo reached', { exact: true }).click();
	await submit(page.getByRole('button', { name: 'Yes, undo reached' }));
	await onPage(page, NELL);
	await expect(reachButton(page, id)).toBeVisible();
	await expect(item(page, id)).not.toContainText('Reached on');
	expect((await stored(cleo, NELL.id)).reached(id)).toBeNull();
});

test('a custom milestone is added, edited and deleted; 201 characters are refused', async ({
	page
}) => {
	const cleo = await stubAs(CLEO);
	await signIn(page, CLEO);
	await page.goto(pageUrl(NELL.id));
	await expect(custom(page)).toContainText('No milestones added');
	await custom(page).getByRole('link', { name: 'Add a milestone' }).click();
	await expect(page.getByRole('heading', { name: 'Add a milestone', level: 1 })).toBeVisible();
	await expect(field(page, 'Date reached')).toHaveValue(TODAY);
	await expectAccessible(page);
	await untilHydrated(field(page, 'Date reached'));
	const long = 'a'.repeat(MILESTONE_TEXT_MAX + 1);
	await field(page, 'Milestone').fill(long);
	await save(page);
	await expectFieldError(field(page, 'Milestone'), 'Enter a milestone of up to 200 characters');
	await expect(field(page, 'Milestone')).toHaveValue(long);
	expect((await stored(cleo, NELL.id)).view.custom).toEqual([]);

	await field(page, 'Milestone').fill('Waves bye');
	await save(page);
	await onPage(page, NELL);
	expect(await written('POST', apiPath(NELL.id))).toEqual({ text: 'Waves bye', reached_on: TODAY });
	await expect(custom(page).getByRole('listitem')).toHaveCount(1);
	await expect(custom(page)).toContainText(reachedLabel(TODAY));

	await editLink(custom(page), 'Waves bye').click();
	await expect(field(page, 'Milestone')).toHaveValue('Waves bye');
	await untilHydrated(field(page, 'Milestone'));
	await field(page, 'Milestone').fill('Waves bye-bye');
	await save(page);
	await onPage(page, NELL);
	const [row] = (await stored(cleo, NELL.id)).view.custom as Row[];
	expect(await written('PATCH', apiPath(NELL.id, row.id))).toEqual({ text: 'Waves bye-bye' });
	await expect(custom(page)).toContainText('Waves bye-bye');

	await editLink(custom(page), 'Waves bye-bye').click();
	await page.getByText('Delete this milestone', { exact: true }).click();
	await submit(page.getByRole('button', { name: 'Yes, delete this milestone' }));
	await onPage(page, NELL);
	await expect(custom(page).getByRole('listitem')).toHaveCount(0);
	expect((await stored(cleo, NELL.id)).view.custom).toEqual([]);
});

test('a caregiver marks, edits, undoes and adds milestones', async ({ page }) => {
	const ana = await caregiverOf(NELL.id);
	await signIn(page, ANA);
	await page.goto(pageUrl(NELL.id));
	await onPage(page, NELL);
	await submit(reachButton(page, '9m-se-stub-1'));
	await onItem(page, NELL, '9m-se-stub-1');
	await editLink(item(page, '9m-se-stub-1'), itemText('9m-se-stub-1')).click();
	await untilHydrated(field(page, 'Note'));
	await field(page, 'Note').fill('with Ana');
	await save(page);
	await onPage(page, NELL);
	expect((await stored(ana, NELL.id)).reached('9m-se-stub-1')).toMatchObject({ note: 'with Ana' });
	await editLink(item(page, '9m-se-stub-1'), itemText('9m-se-stub-1')).click();
	await page.getByText('Undo reached', { exact: true }).click();
	await submit(page.getByRole('button', { name: 'Yes, undo reached' }));
	await onPage(page, NELL);
	await expect(reachButton(page, '9m-se-stub-1')).toBeVisible();

	await page.goto(newUrl(NELL.id));
	await untilHydrated(field(page, 'Date reached'));
	await field(page, 'Milestone').fill('Claps');
	await save(page);
	await onPage(page, NELL);
	await expect(custom(page)).toContainText('Claps');

	await editLink(custom(page), 'Claps').click();
	await page.getByText('Delete this milestone', { exact: true }).click();
	await submit(page.getByRole('button', { name: 'Yes, delete this milestone' }));
	await onPage(page, NELL);
	await expect(custom(page).getByRole('listitem')).toHaveCount(0);
	expect((await stored(ana, NELL.id)).view.custom).toEqual([]);
});

test('no grant, an unknown kid and another kid’s entry get the same 404 page', async ({ page }) => {
	const cleo = await stubAs(CLEO);
	const nells = await seed(cleo, NELL.id, { text: 'Nell row' });
	await signIn(page, CLEO);
	const unknown = await expectNotFoundPage(page, editUrl(NELL.id, 'ms-000000000999'));
	// Otto's retired row, asked for under Nell.
	expect(await expectNotFoundPage(page, editUrl(NELL.id, 'ms-retired'))).toBe(unknown);
	expect(await expectNotFoundPage(page, pageUrl('k-unknown'))).toBe(unknown);
	expect(await expectNotFoundPage(page, newUrl('k-unknown'))).toBe(unknown);

	await page.context().clearCookies();
	await signIn(page, DAN);
	for (const path of [pageUrl(NELL.id), newUrl(NELL.id), editUrl(NELL.id, nells.id)]) {
		expect(await expectNotFoundPage(page, path)).toBe(unknown);
	}
});

test('the pages are sent with Cache-Control: no-store', async ({ page }) => {
	const cleo = await stubAs(CLEO);
	const row = await seed(cleo, NELL.id, { text: 'x' });
	await signIn(page, CLEO);
	for (const path of [pageUrl(NELL.id), newUrl(NELL.id), editUrl(NELL.id, row.id)]) {
		const response = await page.request.get(path);
		expect(response.status()).toBe(200);
		expect(response.headers()['cache-control']).toBe('no-store');
	}
});

test('the page’s own words avoid every judgement word', async ({ page }) => {
	const cleo = await stubAs(CLEO);
	const nells = await seed(cleo, NELL.id, { item: '9m-se-stub-1', note: 'n' });
	await seed(cleo, NELL.id, { text: 'Custom one' });
	await seed(cleo, PIA.id, { item: '60m-lc-stub-1' });
	await signIn(page, CLEO);
	for (const path of [
		pageUrl(NELL.id),
		pageUrl(OTTO.id),
		pageUrl(PIA.id),
		pageUrl(SAMI.id),
		pageUrl(REMY.id),
		newUrl(NELL.id),
		newUrl(NELL.id, '9m-lc-stub-1'),
		editUrl(NELL.id, nells.id)
	]) {
		await page.goto(path);
		await expect(page.locator('main h1')).toBeVisible();
		const words = await ownWords(page);
		for (const w of DENYLIST)
			expect(words, `${path}: ${w}`).not.toMatch(new RegExp(`\\b${w}\\b`, 'i'));
	}
});

test('a 200-character unbroken custom text does not overflow at phone width', async ({
	page
}, testInfo) => {
	test.skip(testInfo.project.name !== 'phone', 'the overflow is a phone-width check');
	const cleo = await stubAs(CLEO);
	const text = 'w'.repeat(MILESTONE_TEXT_MAX);
	const row = await seed(cleo, NELL.id, { text, note: 'n'.repeat(NOTE_MAX_LEN) });
	await signIn(page, CLEO);
	await page.goto(pageUrl(NELL.id));
	await expect(custom(page).getByRole('listitem')).toContainText(text);
	expect(await overflows(page)).toBe(false);
	await page.goto(editUrl(NELL.id, row.id));
	await expect(field(page, 'Milestone')).toHaveValue(text);
	expect(await overflows(page)).toBe(false);
	await page.goto(pageUrl(PIA.id));
	await seed(cleo, PIA.id, { text });
	await page.reload();
	await expect(custom(page).getByRole('listitem')).toContainText(text);
	expect(await overflows(page)).toBe(false);
});

test.describe('without JavaScript', () => {
	test.use({ javaScriptEnabled: false });

	test('lists open natively, and Reached, create, edit and undo work as plain forms', async ({
		page
	}) => {
		const cleo = await stubAs(CLEO);
		await signIn(page, CLEO);
		await page.goto(pageUrl(NELL.id));
		await expect(list(page, 'By 9 months')).toHaveAttribute('open', '');
		await expect(list(page, 'By 3 years')).not.toHaveAttribute('open', '');
		await list(page, 'By 3 years').locator(':scope > summary').click();
		await expect(list(page, 'By 3 years')).toHaveAttribute('open', '');
		await page.getByRole('link', { name: 'Go to the current list' }).click();
		await expect(page).toHaveURL(/#list-9$/);

		await submit(reachButton(page, '9m-se-stub-1'));
		await onItem(page, NELL, '9m-se-stub-1');
		await expect(item(page, '9m-se-stub-1')).toContainText(reachedLabel(TODAY));

		// A mark in a list the server rendered closed comes back with that list open at the row.
		await expect(list(page, 'By 3 years')).not.toHaveAttribute('open', '');
		await list(page, 'By 3 years').locator(':scope > summary').click();
		await submit(reachButton(list(page, 'By 3 years'), '36m-cg-stub-1'));
		await onItem(page, NELL, '36m-cg-stub-1');
		await expect(list(page, 'By 3 years')).toHaveAttribute('open', '');
		await expect(item(page, '36m-cg-stub-1')).toBeInViewport();
		await expect(item(page, '36m-cg-stub-1')).toContainText(reachedLabel(TODAY));

		await page.goto(newUrl(NELL.id, '9m-lc-stub-1'));
		await field(page, 'Date reached').fill(addDays(TODAY, 1));
		await save(page);
		await expectFieldError(
			field(page, 'Date reached'),
			'Enter a date that is not in the future and not before the birth date'
		);
		await expect(page.getByText(itemText('9m-lc-stub-1'), { exact: true })).toBeVisible();
		await field(page, 'Date reached').fill(addDays(TODAY, -2));
		await save(page);
		await onPage(page, NELL);
		await expect(item(page, '9m-lc-stub-1')).toContainText(reachedLabel(addDays(TODAY, -2)));

		await editLink(item(page, '9m-lc-stub-1'), itemText('9m-lc-stub-1')).click();
		await expect(field(page, 'Date reached')).toHaveValue(addDays(TODAY, -2));
		await field(page, 'Date reached').fill(addDays(TODAY, -3));
		await save(page);
		await onPage(page, NELL);
		await expect(item(page, '9m-lc-stub-1')).toContainText(reachedLabel(addDays(TODAY, -3)));

		await custom(page).getByRole('link', { name: 'Add a milestone' }).click();
		await field(page, 'Milestone').fill('Stacks blocks');
		await save(page);
		await onPage(page, NELL);
		await expect(custom(page)).toContainText('Stacks blocks');

		await editLink(custom(page), 'Stacks blocks').click();
		await field(page, 'Note').fill('two blocks');
		await save(page);
		await onPage(page, NELL);
		const [row] = (await stored(cleo, NELL.id)).view.custom as Row[];
		expect(row).toMatchObject({ text: 'Stacks blocks', note: 'two blocks' });

		await editLink(item(page, '9m-se-stub-1'), itemText('9m-se-stub-1')).click();
		await page.getByText('Undo reached', { exact: true }).click();
		await submit(page.getByRole('button', { name: 'Yes, undo reached' }));
		await onPage(page, NELL);
		await expect(reachButton(page, '9m-se-stub-1')).toBeVisible();
	});

	test('a refused Reached opens its list and shows the message in the item’s row', async ({
		page
	}) => {
		const cleo = await stubAs(CLEO);
		const id = '36m-cg-stub-1';
		await signIn(page, CLEO);
		await page.goto(pageUrl(NELL.id));
		await list(page, 'By 3 years').locator(':scope > summary').click();
		await seed(cleo, NELL.id, { item: id });
		await submit(reachButton(list(page, 'By 3 years'), id));
		await expect(list(page, 'By 3 years')).toHaveAttribute('open', '');
		await expect(item(page, id).getByRole('alert')).toHaveText('Already marked as reached');
		await expect(item(page, id).getByRole('alert')).toBeInViewport();
	});

	test('the off note and its link are rendered by the server', async ({ page }) => {
		await stubAs(CLEO);
		await signIn(page, CLEO);
		await page.goto(pageUrl(SAMI.id));
		await expect(page.getByText(NOTE_OFF(SAMI.name))).toBeVisible();
		await page.getByRole('link', { name: 'Turn it on in Modules' }).click();
		await expect(page).toHaveURL(`/kids/${SAMI.id}/modules`);
	});
});
