import { expect, test, type Page } from '@playwright/test';
import { instantToLocalInput } from '../src/lib/time-zone.js';
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
	receivedRequests,
	resetStub,
	setScenario,
	stubAs
} from './stub-api.js';

test.beforeEach(async () => {
	await resetStub();
});

type Row = Record<string, unknown>;
type Api = Awaited<ReturnType<typeof stubAs>>;

const MS_PER_MINUTE = 60_000;
const MS_PER_HOUR = 60 * MS_PER_MINUTE;
const TIMELINE_DAY = /\/timeline\?from=(\d{4}-\d{2}-\d{2})&to=\1$/;

const rfc = (ms: number) => new Date(ms).toISOString().replace(/\.\d{3}Z$/, 'Z');
// An instant on a whole minute, so a datetime-local input shows it exactly.
const minute = (offset: number) =>
	rfc(Math.floor(Date.now() / MS_PER_MINUTE) * MS_PER_MINUTE + offset * MS_PER_MINUTE);
const inUtc = (instant: string) => instantToLocalInput(instant, 'UTC');
const day = (instant: string, zone = 'UTC') => instantToLocalInput(instant, zone).slice(0, 10);

const logUrl = (kind: string, kid = MIA.id) => `/kids/${kid}/log/${kind}`;
const entryUrl = (kind: string, id: unknown, kid = MIA.id) => `${logUrl(kind, kid)}/${id}`;
const rows = async (api: Api, kind: string, kid = MIA.id) =>
	((await api('GET', `/v1/kids/${kid}/${kind}`)).body[kind] ?? []) as Row[];
const seed = async (api: Api, kind: string, body: Row, kid = MIA.id) => {
	const r = await api('POST', `/v1/kids/${kid}/${kind}`, body);
	expect(r.status).toBe(201);
	return r.body as Row;
};
const written = async (method: string, path: string) => {
	const writes = (await receivedRequests()).filter((r) => r.method === method && r.path === path);
	return JSON.parse(writes.at(-1)?.body ?? 'null') as Row;
};

const field = (page: Page, label: string) => page.getByLabel(label, { exact: true });
const save = (page: Page) => page.getByRole('button', { name: 'Save', exact: true }).click();
// The save lands on a timeline page that exists, not just on its URL.
async function onTimeline(page: Page) {
	await page.waitForURL(TIMELINE_DAY);
	await expect(page.getByRole('heading', { name: /^Timeline for /, level: 1 })).toBeVisible();
}

async function deleteEntry(page: Page) {
	await page.getByText('Delete this entry', { exact: true }).click();
	await page.getByRole('button', { name: 'Yes, delete this entry' }).click();
	await onTimeline(page);
}

type Case = {
	kind: string;
	heading: string;
	create: (page: Page) => Promise<void>;
	created: (row: Row) => void;
	edit: (page: Page) => Promise<void>;
	edited: (row: Row) => void;
};

// Fixed once: each is in the past however late the run gets.
const SLEEP_START = minute(-120);
const SLEEP_END = minute(-60);
const NURSING_START = minute(-60);
const NURSING_END = minute(-30);
const CASES: Case[] = [
	{
		kind: 'bottles',
		heading: 'Log a bottle',
		create: async (page) => {
			await page.getByRole('checkbox', { name: 'Breast milk' }).check();
			await page.getByRole('checkbox', { name: 'Formula' }).check();
			await field(page, 'Volume').fill('120');
		},
		created: (row) => {
			expect(row.contents).toEqual(['breast_milk', 'formula']);
			expect(row.volume).toMatchObject({ ml: 120, typed_unit: 'ml' });
		},
		edit: async (page) => {
			await expect(page.getByRole('checkbox', { name: 'Formula' })).toBeChecked();
			await expect(page.getByRole('checkbox', { name: 'Other' })).not.toBeChecked();
			await field(page, 'Volume').fill('150');
		},
		edited: (row) => {
			expect(row.contents).toEqual(['breast_milk', 'formula']);
			expect(row.volume).toMatchObject({ ml: 150, typed_unit: 'ml' });
		}
	},
	{
		kind: 'diapers',
		heading: 'Log a diaper',
		create: async (page) => {
			await page.getByRole('radio', { name: 'Wet' }).check();
		},
		created: (row) => expect(row.kind).toBe('wet'),
		edit: async (page) => {
			await expect(page.getByRole('radio', { name: 'Wet' })).toBeChecked();
			await page.getByRole('radio', { name: 'Both' }).check();
		},
		edited: (row) => expect(row.kind).toBe('both')
	},
	{
		kind: 'meals',
		heading: 'Log a meal',
		create: async (page) => {
			await field(page, 'What').fill('Rice');
			await field(page, 'Reaction').fill('liked it');
		},
		created: (row) => expect(row).toMatchObject({ what: 'Rice', reaction: 'liked it' }),
		edit: async (page) => {
			await expect(field(page, 'What')).toHaveValue('Rice');
			await field(page, 'What').fill('Pasta');
		},
		edited: (row) => expect(row).toMatchObject({ what: 'Pasta', reaction: 'liked it' })
	},
	{
		kind: 'pumping',
		heading: 'Log pumping',
		create: async (page) => {
			await field(page, 'Minutes').fill('20');
			await field(page, 'Volume').fill('80');
			await page.getByRole('radio', { name: 'Both' }).check();
		},
		created: (row) => {
			expect(row).toMatchObject({ duration_seconds: 1200, side: 'both' });
			expect(row.volume).toMatchObject({ ml: 80 });
		},
		edit: async (page) => {
			await expect(field(page, 'Minutes')).toHaveValue('20');
			await field(page, 'Minutes').fill('25');
		},
		edited: (row) => expect(row).toMatchObject({ duration_seconds: 1500, side: 'both' })
	},
	{
		kind: 'sleep',
		heading: 'Log sleep',
		create: async (page) => {
			await field(page, 'Started at').fill(inUtc(SLEEP_START));
			await field(page, 'Ended at').fill(inUtc(SLEEP_END));
		},
		created: (row) => expect(row.ended_at).toBe(SLEEP_END),
		edit: async (page) => {
			await expect(field(page, 'Ended at')).toHaveValue(inUtc(SLEEP_END));
			await field(page, 'Ended at').fill(inUtc(NURSING_END));
		},
		edited: (row) => expect(row.ended_at).toBe(NURSING_END)
	},
	{
		kind: 'nursing',
		heading: 'Log nursing',
		create: async (page) => {
			await field(page, 'Started at').fill(inUtc(NURSING_START));
			await field(page, 'Ended at').fill(inUtc(NURSING_END));
			await field(page, 'Left minutes').fill('10');
			await field(page, 'Right minutes').fill('5');
		},
		created: (row) => {
			expect(row).toMatchObject({ left_seconds: 600, right_seconds: 300, side: null });
			expect(row.ended_at).toBe(NURSING_END);
		},
		edit: async (page) => {
			await expect(field(page, 'Left minutes')).toHaveValue('10');
			await field(page, 'Left minutes').fill('12');
		},
		edited: (row) => expect(row).toMatchObject({ left_seconds: 720, right_seconds: 300 })
	}
];

for (const c of CASES) {
	test(`${c.kind}: create with a note, edit one field, delete`, async ({ page }) => {
		const api = await stubAs(ANA);
		await signIn(page, ANA);
		await page.goto(logUrl(c.kind));
		await expect(page.getByRole('heading', { name: c.heading, level: 1 })).toBeVisible();
		await expectAccessible(page);
		await c.create(page);
		await field(page, 'Note').fill('first note');
		await save(page);
		await onTimeline(page);

		const [created] = await rows(api, c.kind);
		expect(created.note).toBe('first note');
		c.created(created);
		const when = String(created.at ?? created.started_at);
		expect(page.url()).toContain(`?from=${day(when)}&to=${day(when)}`);

		await page.goto(entryUrl(c.kind, created.id));
		await expect(page.getByRole('heading', { name: /^Edit /, level: 1 })).toBeVisible();
		await expect(field(page, 'Note')).toHaveValue('first note');
		await expectAccessible(page);
		await c.edit(page);
		await save(page);
		await onTimeline(page);
		const [edited] = await rows(api, c.kind);
		expect(edited.id).toBe(created.id);
		expect(edited.note).toBe('first note');
		c.edited(edited);

		await page.goto(entryUrl(c.kind, created.id));
		await deleteEntry(page);
		expect(await rows(api, c.kind)).toEqual([]);
	});
}

test('a volume is typed in ounces for a parent with that preference, in millilitres otherwise', async ({
	page
}) => {
	const api = await stubAs(ANA);
	await signIn(page, ANA);
	await page.goto(logUrl('bottles'));
	await expect(field(page, 'Volume unit')).toHaveValue('ml');
	await page.getByRole('checkbox', { name: 'Formula' }).check();
	await field(page, 'Volume').fill('120');
	await save(page);
	await onTimeline(page);
	const [inMl] = await rows(api, 'bottles');
	expect(inMl.volume).toMatchObject({ ml: 120, typed_unit: 'ml' });
	await page.goto(entryUrl('bottles', inMl.id));
	await expect(field(page, 'Volume')).toHaveValue('120');
	await expect(field(page, 'Volume unit')).toHaveValue('ml');

	expect((await api('PATCH', '/v1/me', { units: { volume: 'floz' } })).status).toBe(200);
	await page.goto(logUrl('bottles'));
	await expect(field(page, 'Volume unit')).toHaveValue('floz');
	await expectAccessible(page);
	await page.getByRole('checkbox', { name: 'Formula' }).check();
	await field(page, 'Volume').fill('4.25');
	await save(page);
	await onTimeline(page);
	const inOz = (await rows(api, 'bottles')).find((r) => r.id !== inMl.id) as Row;
	expect(inOz.volume).toMatchObject({ floz_hundredths: 425, typed_unit: 'floz' });
	await page.goto(entryUrl('bottles', inOz.id));
	await expect(field(page, 'Volume')).toHaveValue('4.25');
	await expect(field(page, 'Volume unit')).toHaveValue('floz');

	// The 120 ml entry, opened by the ounce parent, shows its ounce column.
	const hundredths = (inMl.volume as { floz_hundredths: number }).floz_hundredths;
	await page.goto(entryUrl('bottles', inMl.id));
	await expect(field(page, 'Volume')).toHaveValue(String(Number((hundredths / 100).toFixed(2))));
	await expect(field(page, 'Volume unit')).toHaveValue('floz');
});

test('editing only the note sends no volume and no time; an edited amount sends the volume', async ({
	page
}) => {
	const api = await stubAs(ANA);
	const bottle = await seed(api, 'bottles', {
		at: minute(-10),
		contents: ['formula'],
		volume: { unit: 'floz', amount: 425 }
	});
	const path = `/v1/kids/${MIA.id}/bottles/${bottle.id}`;
	await signIn(page, ANA);
	await page.goto(entryUrl('bottles', bottle.id));
	await expect(field(page, 'Volume')).toHaveValue('126');
	await field(page, 'Note').fill('only the note');
	await save(page);
	await onTimeline(page);
	const body = await written('PATCH', path);
	expect(body).toEqual({ contents: ['formula'], note: 'only the note' });
	const [kept] = await rows(api, 'bottles');
	expect(kept.volume).toEqual({ ml: 126, floz_hundredths: 425, typed_unit: 'floz' });
	expect(kept.at).toBe(bottle.at);

	await page.goto(entryUrl('bottles', bottle.id));
	await field(page, 'Volume').fill('130');
	await save(page);
	await onTimeline(page);
	expect((await written('PATCH', path)).volume).toEqual({ unit: 'ml', amount: 130 });
	expect((await rows(api, 'bottles'))[0].volume).toMatchObject({ ml: 130, typed_unit: 'ml' });
});

test('a time an hour ahead shows the field error and keeps the other values', async ({ page }) => {
	const api = await stubAs(ANA);
	await signIn(page, ANA);
	await page.goto(logUrl('bottles'));
	const ahead = inUtc(minute(60));
	await field(page, 'Time').fill(ahead);
	await page.getByRole('checkbox', { name: 'Formula' }).check();
	await field(page, 'Volume').fill('100');
	await field(page, 'Note').fill('keep me');
	await save(page);
	await expectFieldError(field(page, 'Time'), 'not in the future');
	await expect(field(page, 'Time')).toHaveValue(ahead);
	await expect(page.getByRole('checkbox', { name: 'Formula' })).toBeChecked();
	await expect(field(page, 'Volume')).toHaveValue('100');
	await expect(field(page, 'Note')).toHaveValue('keep me');
	await expectAccessible(page);
	expect(await rows(api, 'bottles')).toEqual([]);

	await field(page, 'Time').fill(inUtc(minute(-5)));
	await save(page);
	await onTimeline(page);
	expect(await rows(api, 'bottles')).toHaveLength(1);
});

test('a bottle needs at least one content, and says so on the boxes', async ({ page }) => {
	await signIn(page, ANA);
	await page.goto(logUrl('bottles'));
	await field(page, 'Volume').fill('100');
	await save(page);
	for (const name of ['Breast milk', 'Formula', "Cow's milk", 'Other']) {
		await expectFieldError(page.getByRole('checkbox', { name }), 'Choose at least one');
	}
	await expect(field(page, 'Volume')).toHaveValue('100');
	await expectAccessible(page);
});

test('a second nursing timer is refused with a link home; the first one keeps running', async ({
	page
}) => {
	const api = await stubAs(ANA);
	await signIn(page, ANA);
	await page.goto(logUrl('nursing'));
	await page.getByRole('radio', { name: 'Left' }).check();
	await save(page);
	await onTimeline(page);
	expect(await rows(api, 'nursing')).toMatchObject([{ ended_at: null, side: 'left' }]);

	await page.goto(logUrl('nursing'));
	await page.getByRole('radio', { name: 'Right' }).check();
	await save(page);
	const alert = page.getByRole('alert').filter({ hasText: 'A nursing timer is already running' });
	await expect(alert).toBeVisible();
	await expect(alert.getByRole('link')).toHaveAttribute('href', `/kids/${MIA.id}`);
	await expect(page.getByRole('radio', { name: 'Right' })).toBeChecked();
	await expectAccessible(page);
	expect(await rows(api, 'nursing')).toHaveLength(1);
});

test('a second sleep timer is refused the same way', async ({ page }) => {
	const api = await stubAs(ANA);
	await seed(api, 'sleep', {});
	await signIn(page, ANA);
	await page.goto(logUrl('sleep'));
	await save(page);
	const alert = page.getByRole('alert').filter({ hasText: 'A sleep timer is already running' });
	await expect(alert).toBeVisible();
	await expect(alert.getByRole('link')).toHaveAttribute('href', `/kids/${MIA.id}`);
	expect(await rows(api, 'sleep')).toHaveLength(1);
});

test('a time in the skipped hour of a Madrid night saves an hour later, in the kid zone', async ({
	page
}) => {
	const api = await stubAs(ANA);
	await signIn(page, ANA);
	await page.goto(logUrl('diapers', LEO.id));
	await expect(page.getByText('Europe/Madrid')).toBeVisible();
	await field(page, 'Time').fill('2026-03-29T02:30');
	await page.getByRole('radio', { name: 'Wet' }).check();
	await save(page);
	await page.waitForURL(/\/timeline\?from=2026-03-29&to=2026-03-29$/);
	await expect(page.getByRole('heading', { name: 'Timeline for Leo', level: 1 })).toBeVisible();
	const [stored] = await rows(api, 'diapers', LEO.id);
	expect(stored.at).toBe('2026-03-29T01:30:00Z');

	await page.goto(entryUrl('diapers', stored.id, LEO.id));
	await expect(field(page, 'Time')).toHaveValue('2026-03-29T03:30');
	await field(page, 'Note').fill('after the change');
	await save(page);
	await onTimeline(page);
	expect(await written('PATCH', `/v1/kids/${LEO.id}/diapers/${stored.id}`)).toEqual({
		kind: 'wet',
		note: 'after the change'
	});
	expect((await rows(api, 'diapers', LEO.id))[0].at).toBe('2026-03-29T01:30:00Z');
});

for (const [who, role] of [
	[ANA, 'caregiver'],
	[CLEO, 'guardian']
] as const) {
	test(`a ${role} can edit and delete an entry`, async ({ page }) => {
		const api = await stubAs(CLEO);
		const diaper = await seed(api, 'diapers', { at: minute(-10), kind: 'wet' }, LEO.id);
		await signIn(page, who);
		await page.goto(entryUrl('diapers', diaper.id, LEO.id));
		await page.getByRole('radio', { name: 'Dirty' }).check();
		await save(page);
		await onTimeline(page);
		expect((await rows(api, 'diapers', LEO.id))[0].kind).toBe('dirty');
		await page.goto(entryUrl('diapers', diaper.id, LEO.id));
		await deleteEntry(page);
		expect(await rows(api, 'diapers', LEO.id)).toEqual([]);
	});
}

test('a finished nursing edit keeps the stored seconds unless a side is changed', async ({
	page
}) => {
	const api = await stubAs(ANA);
	// Off the minute on purpose: an untouched time must not be re-sent truncated.
	const started = Math.floor(Date.now() / 1000) * 1000 - 3_600_000 - 7000;
	const nursing = await seed(api, 'nursing', {
		started_at: rfc(started),
		ended_at: rfc(started + 900_000),
		left_seconds: 723,
		right_seconds: 61,
		note: 'seeded'
	});
	const path = `/v1/kids/${MIA.id}/nursing/${nursing.id}`;
	await signIn(page, ANA);
	await page.goto(entryUrl('nursing', nursing.id));
	await expect(field(page, 'Left minutes')).toHaveValue('12');
	await expect(field(page, 'Right minutes')).toHaveValue('1');
	await expect(page.locator('input[name="left_seconds_original"]')).toHaveValue('723');
	await expect(page.locator('input[name="right_seconds_original"]')).toHaveValue('61');
	await expect(page.getByRole('radiogroup', { name: 'Side' })).toHaveCount(0);
	await expectAccessible(page);
	await field(page, 'Note').fill('only the note');
	await save(page);
	await onTimeline(page);
	expect(await written('PATCH', path)).toEqual({ note: 'only the note' });
	expect((await rows(api, 'nursing'))[0]).toMatchObject({
		left_seconds: 723,
		right_seconds: 61,
		started_at: nursing.started_at,
		ended_at: nursing.ended_at
	});

	await page.goto(entryUrl('nursing', nursing.id));
	await field(page, 'Left minutes').fill('10');
	await save(page);
	await onTimeline(page);
	expect(await written('PATCH', path)).toEqual({ left_seconds: 600, note: 'only the note' });
	expect((await rows(api, 'nursing'))[0]).toMatchObject({ left_seconds: 600, right_seconds: 61 });
});

test('a running nursing edit has no minutes and no side; an end time ends it', async ({ page }) => {
	const api = await stubAs(ANA);
	const startedAt = minute(-20);
	const endedAt = minute(-5);
	const nursing = await seed(api, 'nursing', { started_at: startedAt, side: 'right' });
	await signIn(page, ANA);
	await page.goto(entryUrl('nursing', nursing.id));
	await expect(field(page, 'Started at')).toHaveValue(inUtc(startedAt));
	await expect(page.getByLabel('Left minutes')).toHaveCount(0);
	await expect(page.getByLabel('Right minutes')).toHaveCount(0);
	await expect(page.getByRole('radiogroup', { name: 'Side' })).toHaveCount(0);
	await expect(page.locator('input[name$="_seconds_original"]')).toHaveCount(0);
	await expect(page.getByText(/at most 24 hours/)).toHaveCount(0);
	await expectAccessible(page);
	await field(page, 'Ended at').fill(inUtc(endedAt));
	await save(page);
	await onTimeline(page);
	expect(await written('PATCH', `/v1/kids/${MIA.id}/nursing/${nursing.id}`)).toEqual({
		ended_at: endedAt,
		note: null
	});
	expect((await rows(api, 'nursing'))[0]).toMatchObject({
		ended_at: endedAt,
		right_seconds: 900,
		left_seconds: 0,
		side: null
	});
});

// The timer is ended elsewhere between the page loading and the save; the page then shows the finished form.
async function openStaleNursingEdit(page: Page) {
	const api = await stubAs(ANA);
	const nursing = await seed(api, 'nursing', { started_at: minute(-20), side: 'right' });
	const path = `/v1/kids/${MIA.id}/nursing/${nursing.id}`;
	await signIn(page, ANA);
	await page.goto(entryUrl('nursing', nursing.id));
	await expect(page.getByLabel('Left minutes')).toHaveCount(0);
	expect((await api('POST', `${path}/end`, {})).status).toBe(200);
	const [ended] = await rows(api, 'nursing');
	const right = ended.right_seconds as number;

	await field(page, 'Note').fill('typed while running');
	await save(page);
	const message = page.getByRole('alert').filter({ hasText: 'This timer has ended' });
	await expect(message).toHaveText('This timer has ended; check the times and save again');
	await expectFreshForm(page, right);
	await expect(field(page, 'Ended at')).toHaveValue(inUtc(String(ended.ended_at)));
	await expect(field(page, 'Note')).toHaveValue('');
	expect((await receivedRequests()).filter((r) => r.method === 'PATCH')).toEqual([]);
	return { api, path, right };
}

// The finished form: floored minutes, and the stored seconds as the hidden originals.
async function expectFreshForm(page: Page, right: number) {
	await expect(field(page, 'Left minutes')).toBeVisible();
	await expect(page.locator('input[name="left_seconds_original"]')).toHaveValue('0');
	await expect(page.locator('input[name="right_seconds_original"]')).toHaveValue(String(right));
	await expect(field(page, 'Right minutes')).toHaveValue(String(Math.floor(right / 60)));
}

async function staleNursingEdit(page: Page) {
	const { api, path, right } = await openStaleNursingEdit(page);
	await field(page, 'Note').fill('after the end');
	await save(page);
	await onTimeline(page);
	expect(await written('PATCH', path)).toEqual({ note: 'after the end' });
	expect((await rows(api, 'nursing'))[0]).toMatchObject({
		right_seconds: right,
		note: 'after the end'
	});
}

// Later failures keep the finished form: the field error needs its minutes input.
async function staleNursingEditThenInvalid(page: Page) {
	const { api, path, right } = await openStaleNursingEdit(page);
	await field(page, 'Left minutes').fill('abc');
	await save(page);
	await expectFieldError(field(page, 'Left minutes'), 'whole minutes');
	await expectFreshForm(page, right);
	await expect(page.getByText('This timer has ended')).toHaveCount(0);

	await field(page, 'Left minutes').fill('0');
	await field(page, 'Note').fill('fixed');
	await save(page);
	await onTimeline(page);
	expect(await written('PATCH', path)).toEqual({ note: 'fixed' });
	expect((await rows(api, 'nursing'))[0]).toMatchObject({ right_seconds: right, note: 'fixed' });
}

async function staleNursingEditThenUnavailable(page: Page) {
	const { api, path, right } = await openStaleNursingEdit(page);
	await setScenario('patch_unavailable');
	await field(page, 'Note').fill('unsent');
	await save(page);
	await expect(
		page.getByRole('alert').filter({ hasText: 'cannot reach its server' })
	).toBeVisible();
	await expectFreshForm(page, right);
	await setScenario('normal');
	await save(page);
	await onTimeline(page);
	expect(await written('PATCH', path)).toEqual({ note: 'unsent' });
	expect((await rows(api, 'nursing'))[0]).toMatchObject({ right_seconds: right, note: 'unsent' });
}

const staleTests = [
	['shows a message and the fresh form', staleNursingEdit],
	['keeps the fresh form when the next save is invalid', staleNursingEditThenInvalid],
	['keeps the fresh form when the next save finds the API down', staleNursingEditThenUnavailable]
] as const;

for (const [name, run] of staleTests) {
	test(`a nursing edit opened while the timer ran ${name}`, async ({ page }) => {
		await run(page);
	});
}

test('a failed save on a running nursing page stays on the running form', async ({ page }) => {
	const api = await stubAs(ANA);
	const nursing = await seed(api, 'nursing', { started_at: minute(-20), side: 'right' });
	await signIn(page, ANA);
	await page.goto(entryUrl('nursing', nursing.id));
	await field(page, 'Ended at').fill(inUtc(minute(60)));
	await save(page);
	await expectFieldError(field(page, 'Ended at'), 'after the start');
	await expect(page.getByLabel('Left minutes')).toHaveCount(0);
	await expect(page.locator('input[name$="_seconds_original"]')).toHaveCount(0);
});

test('a timer over 24 hours says how to fix it; an end within 24 hours of the start ends it', async ({
	page
}) => {
	const api = await stubAs(ANA);
	const start = minute(-25 * 60);
	const sleep = await seed(api, 'sleep', { started_at: start });
	await signIn(page, ANA);
	await page.goto(entryUrl('sleep', sleep.id));
	await expect(
		page.getByText('A timer can last at most 24 hours; set the end time or delete it')
	).toBeVisible();
	await expectAccessible(page);

	const late = rfc(Date.parse(start) + 30 * MS_PER_HOUR);
	await field(page, 'Ended at').fill(inUtc(late));
	await save(page);
	await expectFieldError(field(page, 'Ended at'), 'after the start');
	await expect(field(page, 'Ended at')).toHaveValue(inUtc(late));

	const ok = rfc(Date.parse(start) + 2 * MS_PER_HOUR);
	await field(page, 'Ended at').fill(inUtc(ok));
	await save(page);
	await onTimeline(page);
	expect((await rows(api, 'sleep'))[0].ended_at).toBe(ok);
});

test('a timer under 24 hours shows no such hint', async ({ page }) => {
	const api = await stubAs(ANA);
	const sleep = await seed(api, 'sleep', { started_at: minute(-60) });
	await signIn(page, ANA);
	await page.goto(entryUrl('sleep', sleep.id));
	await expect(field(page, 'Started at')).toBeVisible();
	await expect(page.getByText(/at most 24 hours/)).toHaveCount(0);
});

test('a stale entry is the 404 page, also when it vanishes under an open form', async ({
	page
}) => {
	const api = await stubAs(ANA);
	const diaper = await seed(api, 'diapers', { at: minute(-10), kind: 'wet' });
	const others = await seed(
		await stubAs(CLEO),
		'diapers',
		{ at: minute(-10), kind: 'wet' },
		LEO.id
	);
	await signIn(page, ANA);
	const unknown = await expectNotFoundPage(page, entryUrl('diapers', 'e-000000009999'));
	// Another kid's entry, another tracker's entry and a name that is no tracker look the same.
	expect(await expectNotFoundPage(page, entryUrl('diapers', others.id))).toBe(unknown);
	expect(await expectNotFoundPage(page, entryUrl('bottles', diaper.id))).toBe(unknown);
	expect(await expectNotFoundPage(page, logUrl('growth'))).toBe(unknown);

	await page.goto(entryUrl('diapers', diaper.id));
	expect((await api('DELETE', `/v1/kids/${MIA.id}/diapers/${diaper.id}`)).status).toBe(204);
	await save(page);
	await expect(page.getByRole('heading', { name: 'Not found' })).toBeVisible();
});

test('a parent without a grant gets the same 404 page for the log and the entry', async ({
	page
}) => {
	const api = await stubAs(ANA);
	const diaper = await seed(api, 'diapers', { at: minute(-10), kind: 'wet' });
	await signIn(page, DAN);
	const unknown = await expectNotFoundPage(page, logUrl('diapers', 'k-unknown'));
	expect(await expectNotFoundPage(page, logUrl('diapers'))).toBe(unknown);
	expect(await expectNotFoundPage(page, entryUrl('diapers', diaper.id))).toBe(unknown);
});

for (const kind of ['sleep', 'nursing'] as const) {
	test(`${kind}: clearing the end of a finished session is refused on the field`, async ({
		page
	}) => {
		const api = await stubAs(ANA);
		const body =
			kind === 'sleep'
				? { started_at: NURSING_START, ended_at: NURSING_END }
				: { started_at: NURSING_START, ended_at: NURSING_END, left_seconds: 600, right_seconds: 0 };
		const entry = await seed(api, kind, body);
		await signIn(page, ANA);
		await page.goto(entryUrl(kind, entry.id));
		await field(page, 'Ended at').fill('');
		await save(page);
		await expectFieldError(field(page, 'Ended at'), 'A finished session cannot be re-opened');
		await expectAccessible(page);
		expect((await rows(api, kind))[0].ended_at).toBe(NURSING_END);
	});
}

async function finishedWithASide(page: Page) {
	const api = await stubAs(ANA);
	await signIn(page, ANA);
	await page.goto(logUrl('nursing'));
	await page.getByRole('radio', { name: 'Left' }).check();
	await field(page, 'Started at').fill(inUtc(NURSING_START));
	await field(page, 'Ended at').fill(inUtc(NURSING_END));
	await field(page, 'Left minutes').fill('10');
	await save(page);
	await expectFieldError(
		page.getByRole('radiogroup', { name: 'Side' }),
		'A finished session has no side; clear the side or the end'
	);
	await expect(field(page, 'Ended at')).toHaveValue(inUtc(NURSING_END));
	await expect(field(page, 'Left minutes')).toHaveValue('10');
	expect(await rows(api, 'nursing')).toEqual([]);
}

test('a side ticked next to an end time is refused on the side', async ({ page }) => {
	await finishedWithASide(page);
	await expectAccessible(page);
});

test('a volume out of range says the range, on the amount and its unit', async ({ page }) => {
	await signIn(page, ANA);
	await page.goto(logUrl('bottles'));
	await page.getByRole('checkbox', { name: 'Formula' }).check();
	await field(page, 'Volume').fill('1001');
	await save(page);
	await expectFieldError(field(page, 'Volume'), 'Enter from 1 to 1000 ml');
	await expect(field(page, 'Volume unit')).toHaveAttribute('aria-invalid', 'true');
	await expectAccessible(page);
});

test.describe('without JavaScript', () => {
	test.use({ javaScriptEnabled: false });

	for (const [name, run] of staleTests) {
		test(`a nursing edit opened while the timer ran ${name}`, async ({ page }) => {
			await run(page);
		});
	}

	test('a bottle with two contents posts plainly and redirects 303 to its day', async ({
		page
	}) => {
		const api = await stubAs(ANA);
		await signIn(page, ANA);
		await page.goto(logUrl('bottles'));
		await page.getByRole('checkbox', { name: 'Breast milk' }).check();
		await page.getByRole('checkbox', { name: 'Formula' }).check();
		await field(page, 'Volume').fill('90');
		await field(page, 'Note').fill('no script');
		const landed = page.waitForResponse((r) => new URL(r.url()).pathname.endsWith('/timeline'));
		await save(page);
		const hop = await (await landed).request().redirectedFrom()?.response();
		const [stored] = await rows(api, 'bottles');
		expect(stored).toMatchObject({ contents: ['breast_milk', 'formula'], note: 'no script' });
		expect(hop?.status()).toBe(303);
		expect(hop?.headers()['location']).toBe(
			`/kids/${MIA.id}/timeline?from=${day(String(stored.at))}&to=${day(String(stored.at))}`
		);
		await expect(page.getByText(/Bottle 90 ml, breast milk \+ formula/)).toBeVisible();
		await expect(page.getByText('no script')).toBeVisible();
	});

	test('a side ticked next to an end time is refused on the side', async ({ page }) => {
		await finishedWithASide(page);
	});

	test('a field error keeps the typed values', async ({ page }) => {
		await signIn(page, ANA);
		await page.goto(logUrl('meals'));
		await field(page, 'Note').fill('kept');
		await save(page);
		await expectFieldError(field(page, 'What'), 'Enter what was eaten');
		await expect(field(page, 'Note')).toHaveValue('kept');
	});

	test('delete takes the opened details and a second confirming post', async ({ page }) => {
		const api = await stubAs(ANA);
		const diaper = await seed(api, 'diapers', { at: minute(-10), kind: 'wet' });
		await signIn(page, ANA);
		await page.goto(entryUrl('diapers', diaper.id));
		await expect(page.getByRole('button', { name: 'Yes, delete this entry' })).toBeHidden();
		await deleteEntry(page);
		expect(await rows(api, 'diapers')).toEqual([]);
	});
});

// The advanced bottle: a volume per source. Every save waits for its response.
const ADVANCED = '?advanced=1';
const saveAndWait = (page: Page) => submit(page.getByRole('button', { name: 'Save', exact: true }));
type Source = { source: string; volume: Record<string, unknown> };
const sourcesOf = (row: Row) => row.sources as Source[];
const SIMPLE_LINK = 'Advanced: volume per source';

test.describe('the advanced bottle form', () => {
	test('the simple form is unchanged and offers the advanced one', async ({ page }) => {
		await stubAs(ANA);
		await signIn(page, ANA);
		await page.goto(logUrl('bottles'));
		await expect(field(page, 'Volume')).toBeVisible();
		await expect(field(page, 'Volume unit')).toHaveValue('ml');
		await expect(page.getByRole('checkbox')).toHaveCount(4);
		await expect(page.getByText(/^Total:/)).toHaveCount(0);
		await expect(page.getByText('Saving one total replaces the volumes per source')).toHaveCount(0);
		await expect(page.locator('input[name="advanced"]')).toHaveCount(0);
		await expectAccessible(page);
		await page.getByRole('link', { name: SIMPLE_LINK }).click();
		await expect(page).toHaveURL(new RegExp(`/log/bottles\\${ADVANCED}$`));
		await expect(page.getByRole('link', { name: 'Use one total' })).toBeVisible();
	});

	test('?advanced=1 shows a volume input beside each ticked box', async ({ page }) => {
		await stubAs(ANA);
		await signIn(page, ANA);
		await page.goto(logUrl('bottles') + ADVANCED);
		await expect(page.getByRole('heading', { name: 'Log a bottle', level: 1 })).toBeVisible();
		await expect(field(page, 'Volume unit')).toBeVisible();
		await expect(field(page, 'Volume')).toHaveCount(0);
		await expect(page.getByText('Total: 0 ml')).toBeVisible();
		await expectAccessible(page);
		const names = ['Breast milk', 'Formula', "Cow's milk", 'Other'];
		for (const name of names) {
			await expect(field(page, `${name} volume`)).toBeHidden();
			await page.getByRole('checkbox', { name, exact: true }).check();
			await expect(field(page, `${name} volume`)).toBeVisible();
		}
		await expectAccessible(page);
		await page.getByRole('checkbox', { name: 'Formula', exact: true }).uncheck();
		await expect(field(page, 'Formula volume')).toBeHidden();
	});

	test('a breast milk 60 and formula 90 bottle is one 150 ml bottle with its sources', async ({
		page
	}) => {
		const api = await stubAs(ANA);
		await signIn(page, ANA);
		await page.goto(logUrl('bottles') + ADVANCED);
		await page.getByRole('checkbox', { name: 'Breast milk', exact: true }).check();
		await page.getByRole('checkbox', { name: 'Formula', exact: true }).check();
		await field(page, 'Breast milk volume').fill('60');
		await field(page, 'Formula volume').fill('90');
		await saveAndWait(page);
		await onTimeline(page);
		await expect(page.getByText(/Bottle 150 ml, breast milk \+ formula/)).toBeVisible();
		const [created] = await rows(api, 'bottles');
		expect(created.volume).toMatchObject({ ml: 150, typed_unit: 'ml' });
		expect(created.contents).toEqual(['breast_milk', 'formula']);
		expect(sourcesOf(created).map((s) => [s.source, s.volume.ml])).toEqual([
			['breast_milk', 60],
			['formula', 90]
		]);
		const body = await written('POST', `/v1/kids/${MIA.id}/bottles`);
		expect(body).toMatchObject({
			sources: { unit: 'ml', amounts: { breast_milk: 60, formula: 90 } }
		});
		expect(body).not.toHaveProperty('volume');
		expect(body).not.toHaveProperty('contents');

		await page.goto(entryUrl('bottles', created.id));
		await expect(page.getByRole('link', { name: 'Use one total' })).toBeVisible();
		await expect(field(page, 'Breast milk volume')).toHaveValue('60');
		await expect(field(page, 'Formula volume')).toHaveValue('90');
		await expect(field(page, "Cow's milk volume")).toBeHidden();
		await expect(page.getByRole('checkbox', { name: 'Other', exact: true })).not.toBeChecked();
		await expect(page.getByText('Total: 150 ml')).toBeVisible();
		await expectAccessible(page);

		await field(page, 'Formula volume').fill('95');
		await saveAndWait(page);
		await onTimeline(page);
		const [edited] = await rows(api, 'bottles');
		expect(sourcesOf(edited).map((s) => [s.source, s.volume.ml])).toEqual([
			['breast_milk', 60],
			['formula', 95]
		]);
		expect(edited.volume).toMatchObject({ ml: 155 });
		await expect(page.getByText(/Bottle 155 ml, breast milk \+ formula/)).toBeVisible();
	});

	test('editing only the note of an advanced bottle sends neither sources nor volume', async ({
		page
	}) => {
		const api = await stubAs(ANA);
		const bottle = await seed(api, 'bottles', {
			at: minute(-10),
			sources: { unit: 'ml', amounts: { breast_milk: 60, formula: 90 } }
		});
		await signIn(page, ANA);
		await page.goto(entryUrl('bottles', bottle.id));
		await field(page, 'Note').fill('only the note');
		await saveAndWait(page);
		await onTimeline(page);
		const body = await written('PATCH', `/v1/kids/${MIA.id}/bottles/${bottle.id}`);
		expect(body).toEqual({ note: 'only the note' });
		const [kept] = await rows(api, 'bottles');
		expect(kept.volume).toMatchObject({ ml: 150 });
		expect(sourcesOf(kept)).toHaveLength(2);
	});

	test('a parent with fl oz types fl oz per source and reads them back exactly', async ({
		page
	}) => {
		const api = await stubAs(ANA);
		expect((await api('PATCH', '/v1/me', { units: { volume: 'floz' } })).status).toBe(200);
		await signIn(page, ANA);
		await page.goto(logUrl('bottles') + ADVANCED);
		await expect(field(page, 'Volume unit')).toHaveValue('floz');
		await page.getByRole('checkbox', { name: 'Formula', exact: true }).check();
		await page.getByRole('checkbox', { name: 'Other', exact: true }).check();
		await field(page, 'Formula volume').fill('2,5');
		await field(page, 'Other volume').fill('3.25');
		await saveAndWait(page);
		await onTimeline(page);
		const [created] = await rows(api, 'bottles');
		expect(created.volume).toMatchObject({ floz_hundredths: 575, typed_unit: 'floz' });
		expect(sourcesOf(created).map((s) => [s.source, s.volume.floz_hundredths])).toEqual([
			['formula', 250],
			['other', 325]
		]);
		await page.goto(entryUrl('bottles', created.id));
		await expect(field(page, 'Volume unit')).toHaveValue('floz');
		await expect(field(page, 'Formula volume')).toHaveValue('2.5');
		await expect(field(page, 'Other volume')).toHaveValue('3.25');
		await expect(page.getByText('Total: 5.75 fl oz')).toBeVisible();
	});

	test('a ticked source with a blank volume shows the message and keeps what was typed', async ({
		page
	}) => {
		const api = await stubAs(ANA);
		await signIn(page, ANA);
		await page.goto(logUrl('bottles') + ADVANCED);
		await page.getByRole('checkbox', { name: 'Breast milk', exact: true }).check();
		await page.getByRole('checkbox', { name: 'Formula', exact: true }).check();
		await field(page, 'Breast milk volume').fill('60');
		await field(page, 'Note').fill('keep me');
		await saveAndWait(page);
		const message = 'Enter a volume for each source you ticked';
		await expectFieldError(field(page, 'Formula volume'), message);
		await expectFieldError(field(page, 'Breast milk volume'), message);
		await expect(page.getByRole('alert').filter({ hasText: message })).toHaveCount(1);
		await expect(field(page, 'Breast milk volume')).toHaveValue('60');
		await expect(page.getByRole('checkbox', { name: 'Formula', exact: true })).toBeChecked();
		await expect(field(page, 'Note')).toHaveValue('keep me');
		await expect(page.getByText('Total: 60 ml')).toBeVisible();
		await expect(page.getByRole('link', { name: 'Use one total' })).toBeVisible();
		await expectAccessible(page);
		expect(await rows(api, 'bottles')).toEqual([]);

		await field(page, 'Formula volume').fill('30');
		await saveAndWait(page);
		await onTimeline(page);
		expect(await rows(api, 'bottles')).toHaveLength(1);
	});

	test('a total over the range is refused on the sources, keeping the values', async ({ page }) => {
		const api = await stubAs(ANA);
		await signIn(page, ANA);
		await page.goto(logUrl('bottles') + ADVANCED);
		await page.getByRole('checkbox', { name: 'Breast milk', exact: true }).check();
		await page.getByRole('checkbox', { name: 'Formula', exact: true }).check();
		await field(page, 'Breast milk volume').fill('600');
		await field(page, 'Formula volume').fill('600');
		await saveAndWait(page);
		await expectFieldError(field(page, 'Formula volume'), 'The total must be from 1 to 1000 ml');
		await expect(page.getByText('Total: 1200 ml')).toBeVisible();
		await expect(field(page, 'Breast milk volume')).toHaveValue('600');
		await expectAccessible(page);
		expect(await rows(api, 'bottles')).toEqual([]);
	});

	test('no box ticked is refused on the boxes, as in the simple form', async ({ page }) => {
		await signIn(page, ANA);
		await page.goto(logUrl('bottles') + ADVANCED);
		await saveAndWait(page);
		for (const name of ['Breast milk', 'Formula', "Cow's milk", 'Other']) {
			await expectFieldError(
				page.getByRole('checkbox', { name, exact: true }),
				'Choose at least one'
			);
		}
		await expectAccessible(page);
	});

	test('"Use one total" over an advanced bottle saves a plain bottle and the sources are gone', async ({
		page
	}) => {
		const api = await stubAs(ANA);
		const bottle = await seed(api, 'bottles', {
			at: minute(-10),
			sources: { unit: 'ml', amounts: { breast_milk: 60, formula: 90 } }
		});
		const path = `/v1/kids/${MIA.id}/bottles/${bottle.id}`;
		await signIn(page, ANA);
		await page.goto(entryUrl('bottles', bottle.id));
		await page.getByRole('link', { name: 'Use one total' }).click();
		await expect(page).toHaveURL(new RegExp(`/${bottle.id}\\?advanced=0$`));
		await expect(field(page, 'Volume')).toHaveValue('150');
		await expect(field(page, 'Breast milk volume')).toHaveCount(0);
		await expect(page.getByText('Saving one total replaces the volumes per source')).toBeVisible();
		// The hidden original is what makes an untouched total replace the sources.
		await expect(page.locator('input[name="sources_original"]')).toHaveValue('breast_milk,formula');
		await expectAccessible(page);
		await saveAndWait(page);
		await onTimeline(page);
		expect(await written('PATCH', path)).toMatchObject({ volume: { unit: 'ml', amount: 150 } });
		const [plain] = await rows(api, 'bottles');
		expect(sourcesOf(plain)).toEqual([]);
		expect(plain.volume).toMatchObject({ ml: 150 });

		await page.goto(entryUrl('bottles', bottle.id));
		await expect(page.getByRole('link', { name: SIMPLE_LINK })).toBeVisible();
		await expect(field(page, 'Volume')).toHaveValue('150');
		await expect(page.getByText('Saving one total replaces the volumes per source')).toHaveCount(0);
	});

	test('the simple form over an advanced bottle sends the changed total', async ({ page }) => {
		const api = await stubAs(ANA);
		const bottle = await seed(api, 'bottles', {
			at: minute(-10),
			sources: { unit: 'ml', amounts: { breast_milk: 60, formula: 90 } }
		});
		await signIn(page, ANA);
		await page.goto(entryUrl('bottles', bottle.id) + '?advanced=0');
		await field(page, 'Volume').fill('140');
		await saveAndWait(page);
		await onTimeline(page);
		const [plain] = await rows(api, 'bottles');
		expect(sourcesOf(plain)).toEqual([]);
		expect(plain.volume).toMatchObject({ ml: 140 });
		expect(plain.contents).toEqual(['breast_milk', 'formula']);
	});

	test('a bottle created simple opens simple, and can be opened advanced', async ({ page }) => {
		const api = await stubAs(ANA);
		const bottle = await seed(api, 'bottles', {
			at: minute(-10),
			contents: ['breast_milk', 'formula'],
			volume: { unit: 'ml', amount: 120 }
		});
		await signIn(page, ANA);
		await page.goto(entryUrl('bottles', bottle.id));
		await expect(field(page, 'Volume')).toHaveValue('120');
		await expect(field(page, 'Breast milk volume')).toHaveCount(0);
		await expect(page.getByText('Saving one total replaces the volumes per source')).toHaveCount(0);
		await page.getByRole('link', { name: SIMPLE_LINK }).click();
		await expect(page).toHaveURL(new RegExp(`/${bottle.id}\\${ADVANCED}$`));
		await expect(page.getByRole('checkbox', { name: 'Formula', exact: true })).toBeChecked();
		await expect(field(page, 'Formula volume')).toHaveValue('');
		await expect(page.getByText('Total: 0 ml')).toBeVisible();
		await field(page, 'Breast milk volume').fill('50');
		await field(page, 'Formula volume').fill('70');
		await saveAndWait(page);
		await onTimeline(page);
		const [advanced] = await rows(api, 'bottles');
		expect(advanced.volume).toMatchObject({ ml: 120 });
		expect(sourcesOf(advanced).map((s) => [s.source, s.volume.ml])).toEqual([
			['breast_milk', 50],
			['formula', 70]
		]);
	});

	test('a caregiver can log an advanced bottle', async ({ page }) => {
		const api = await stubAs(CLEO);
		await signIn(page, ANA);
		await page.goto(logUrl('bottles', LEO.id) + ADVANCED);
		await page.getByRole('checkbox', { name: 'Formula', exact: true }).check();
		await field(page, 'Formula volume').fill('80');
		await saveAndWait(page);
		await onTimeline(page);
		expect(sourcesOf((await rows(api, 'bottles', LEO.id))[0])).toHaveLength(1);
	});

	test('a failed save over a bottle with sources stays advanced and keeps the values', async ({
		page
	}) => {
		const api = await stubAs(ANA);
		const bottle = await seed(api, 'bottles', {
			at: minute(-10),
			sources: { unit: 'ml', amounts: { breast_milk: 60, formula: 90 } }
		});
		await signIn(page, ANA);
		await page.goto(entryUrl('bottles', bottle.id));
		await page.getByRole('checkbox', { name: 'Other', exact: true }).check();
		await saveAndWait(page);
		await expectFieldError(
			field(page, 'Other volume'),
			'Enter a volume for each source you ticked'
		);
		await expect(page.getByRole('link', { name: 'Use one total' })).toBeVisible();
		await expect(field(page, 'Breast milk volume')).toHaveValue('60');
		await expect(field(page, 'Formula volume')).toHaveValue('90');
		await expect(field(page, 'Volume')).toHaveCount(0);
		const [kept] = await rows(api, 'bottles');
		expect(sourcesOf(kept)).toHaveLength(2);

		await field(page, 'Other volume').fill('30');
		await saveAndWait(page);
		await onTimeline(page);
		expect(sourcesOf((await rows(api, 'bottles'))[0])).toHaveLength(3);
	});

	test('a source typed and then unticked is not sent', async ({ page }) => {
		const api = await stubAs(ANA);
		await signIn(page, ANA);
		await page.goto(logUrl('bottles') + ADVANCED);
		await page.getByRole('checkbox', { name: 'Formula', exact: true }).check();
		await page.getByRole('checkbox', { name: 'Other', exact: true }).check();
		await field(page, 'Formula volume').fill('90');
		await field(page, 'Other volume').fill('30');
		await page.getByRole('checkbox', { name: 'Other', exact: true }).uncheck();
		await saveAndWait(page);
		await onTimeline(page);
		const body = await written('POST', `/v1/kids/${MIA.id}/bottles`);
		expect(body.sources).toEqual({ unit: 'ml', amounts: { formula: 90 } });
		const [created] = await rows(api, 'bottles');
		expect(created.contents).toEqual(['formula']);
		expect(created.volume).toMatchObject({ ml: 90 });
	});

	test('a long note does not overflow the page at phone width', async ({ page }, testInfo) => {
		test.skip(testInfo.project.name !== 'phone', 'the overflow is a phone-width check');
		const api = await stubAs(ANA);
		const bottle = await seed(api, 'bottles', {
			at: minute(-10),
			sources: { unit: 'ml', amounts: { breast_milk: 60, formula: 90, cow_milk: 20, other: 10 } },
			note: 'n'.repeat(200)
		});
		const overflows = () =>
			page.evaluate(
				() => document.documentElement.scrollWidth > document.documentElement.clientWidth
			);
		await signIn(page, ANA);
		await page.goto(entryUrl('bottles', bottle.id));
		await expect(page.getByText('Total: 180 ml')).toBeVisible();
		expect(await overflows()).toBe(false);
		await page.goto(logUrl('bottles') + ADVANCED);
		for (const name of ['Breast milk', 'Formula', "Cow's milk", 'Other']) {
			await page.getByRole('checkbox', { name, exact: true }).check();
		}
		await field(page, 'Note').fill('n'.repeat(200));
		expect(await overflows()).toBe(false);
	});
});

test.describe('the advanced bottle form without JavaScript', () => {
	test.use({ javaScriptEnabled: false });

	test('create, fail on a blank source, fix it, then edit', async ({ page }) => {
		const api = await stubAs(ANA);
		await signIn(page, ANA);
		await page.goto(logUrl('bottles'));
		await page.getByRole('link', { name: SIMPLE_LINK }).click();
		await expect(page).toHaveURL(new RegExp(`${ADVANCED.replace('?', '\\?')}$`));
		await page.getByRole('checkbox', { name: 'Breast milk', exact: true }).check();
		await page.getByRole('checkbox', { name: 'Formula', exact: true }).check();
		await field(page, 'Breast milk volume').fill('60');
		await field(page, 'Note').fill('no script');
		await saveAndWait(page);
		await expectFieldError(
			field(page, 'Formula volume'),
			'Enter a volume for each source you ticked'
		);
		await expect(field(page, 'Breast milk volume')).toHaveValue('60');
		await expect(field(page, 'Note')).toHaveValue('no script');
		await expect(page.getByText('Total: 60 ml')).toBeVisible();
		expect(await rows(api, 'bottles')).toEqual([]);

		await field(page, 'Formula volume').fill('90');
		await saveAndWait(page);
		await onTimeline(page);
		await expect(page.getByText(/Bottle 150 ml, breast milk \+ formula/)).toBeVisible();
		const [created] = await rows(api, 'bottles');
		expect(sourcesOf(created)).toHaveLength(2);

		await page.goto(entryUrl('bottles', created.id));
		await expect(field(page, 'Formula volume')).toHaveValue('90');
		await expect(page.getByText('Total: 150 ml')).toBeVisible();
		await field(page, 'Formula volume').fill('100');
		await saveAndWait(page);
		await onTimeline(page);
		expect(sourcesOf((await rows(api, 'bottles'))[0]).map((s) => s.volume.ml)).toEqual([60, 100]);

		await page.goto(entryUrl('bottles', created.id));
		await page.getByRole('link', { name: 'Use one total' }).click();
		await expect(field(page, 'Volume')).toHaveValue('160');
		await saveAndWait(page);
		await onTimeline(page);
		const [plain] = await rows(api, 'bottles');
		expect(sourcesOf(plain)).toEqual([]);
		expect(plain.volume).toMatchObject({ ml: 160 });
	});

	test('a failed simple save over an advanced bottle stays simple and keeps the hint', async ({
		page
	}) => {
		const api = await stubAs(ANA);
		const bottle = await seed(api, 'bottles', {
			at: minute(-10),
			sources: { unit: 'ml', amounts: { breast_milk: 60, formula: 90 } }
		});
		await signIn(page, ANA);
		await page.goto(entryUrl('bottles', bottle.id) + '?advanced=0');
		await field(page, 'Volume').fill('');
		await saveAndWait(page);
		// The post went to ?/save: the query string is gone, the form must not turn advanced.
		await expectFieldError(field(page, 'Volume'), 'Enter a volume');
		await expect(field(page, 'Breast milk volume')).toHaveCount(0);
		await expect(page.getByText('Saving one total replaces the volumes per source')).toBeVisible();
		await expect(page.getByRole('link', { name: SIMPLE_LINK })).toBeVisible();
		expect(sourcesOf((await rows(api, 'bottles'))[0])).toHaveLength(2);

		await field(page, 'Volume').fill('140');
		await saveAndWait(page);
		await onTimeline(page);
		const [plain] = await rows(api, 'bottles');
		expect(sourcesOf(plain)).toEqual([]);
		expect(plain.volume).toMatchObject({ ml: 140 });
	});

	test('the link back to one total works from a failed post', async ({ page }) => {
		await stubAs(ANA);
		await signIn(page, ANA);
		await page.goto(logUrl('bottles') + ADVANCED);
		await page.getByRole('checkbox', { name: 'Formula', exact: true }).check();
		await saveAndWait(page);
		await expectFieldError(
			field(page, 'Formula volume'),
			'Enter a volume for each source you ticked'
		);
		await page.getByRole('link', { name: 'Use one total' }).click();
		await expect(field(page, 'Volume')).toBeVisible();
		await expect(field(page, 'Formula volume')).toHaveCount(0);
	});
});
