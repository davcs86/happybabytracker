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

const WEIGH_LINK = 'Advanced: weigh before and after';
const HIDE_LINK = 'Hide weigh-ins';
const BEFORE = 'Weight before';
const AFTER = 'Weight after';
const SCALE_HINT =
	'In your weight unit. These weights stay with this session; they are not added to growth. A single-feed reading from a home scale can be off by several grams.';
const BASIS =
	'The volume assumes about 1.03 g per ml, an approximation in the range commonly used (1.0–1.03 g/ml); not measured. A single-feed reading from a home scale can be off by several grams.';
const JUDGEMENT = /good|low|enough|concern|normal/i;

const weight = (page: Page, name: string) => page.getByRole('group', { name, exact: true });
const kilograms = (page: Page, name: string) => weight(page, name).getByLabel('Kilograms');
const running = (page: Page) => page.getByRole('region', { name: 'Running now' });
const growthRows = async (api: Api, kid = MIA.id) =>
	((await api('GET', `/v1/kids/${kid}/growth`)).body.growth ?? []) as Row[];

// A finished session on the create page with its weigh-ins open, the minutes typed.
async function logFinished(page: Page, before: string, after: string) {
	await page.goto(logUrl('nursing') + ADVANCED);
	await field(page, 'Started at').fill(inUtc(NURSING_START));
	await field(page, 'Ended at').fill(inUtc(NURSING_END));
	await field(page, 'Left minutes').fill('10');
	await field(page, 'Right minutes').fill('5');
	if (before) await kilograms(page, BEFORE).fill(before);
	if (after) await kilograms(page, AFTER).fill(after);
	await saveAndWait(page);
	await onTimeline(page);
}

test.describe('the nursing weigh-ins', () => {
	test('the simple form is unchanged and offers the weigh-ins', async ({ page }) => {
		await stubAs(ANA);
		await signIn(page, ANA);
		await page.goto(logUrl('nursing'));
		await expect(page.getByRole('group', { name: BEFORE })).toHaveCount(0);
		await expect(page.locator('input[name="advanced"]')).toHaveCount(0);
		await expectAccessible(page);
		await page.getByRole('link', { name: WEIGH_LINK }).click();
		await expect(page).toHaveURL(new RegExp(`/log/nursing\\${ADVANCED}$`));
		await expect(page.getByRole('link', { name: HIDE_LINK })).toBeVisible();
		await expect(page.getByRole('link', { name: 'Use one total' })).toHaveCount(0);
	});

	test('each weight shows the inputs of its unit, with the hint under both', async ({ page }) => {
		await stubAs(ANA);
		await signIn(page, ANA);
		await page.goto(logUrl('nursing') + ADVANCED);
		await expect(weight(page, BEFORE)).toBeVisible();
		await expect(weight(page, AFTER)).toBeVisible();
		await expect(field(page, 'Weight before unit')).toHaveValue('kg');
		await expect(kilograms(page, BEFORE)).toBeVisible();
		await expect(weight(page, BEFORE).getByLabel('Pounds')).toBeHidden();
		await expect(page.getByText(SCALE_HINT, { exact: true })).toBeVisible();
		await expect(page.getByText(/Weight gain|No estimate/)).toHaveCount(0);
		await expectAccessible(page);
		await field(page, 'Weight before unit').selectOption('lb');
		await expect(weight(page, BEFORE).getByLabel('Pounds')).toBeVisible();
		await expect(weight(page, BEFORE).getByLabel('Ounces')).toBeVisible();
		await expect(kilograms(page, BEFORE)).toBeHidden();
		await expect(kilograms(page, AFTER)).toBeVisible();
		await expectAccessible(page);
	});

	test('a finished session weighed 3.4 and 3.46 kg shows its estimate on the timeline and the edit page', async ({
		page
	}) => {
		const api = await stubAs(ANA);
		await signIn(page, ANA);
		await logFinished(page, '3.4', '3.46');
		await expect(page.getByRole('link', { name: /Nursing L 10 min · R 5 min/ })).toContainText(
			'weight gain 60 g, about 58 ml'
		);
		const body = await written('POST', `/v1/kids/${MIA.id}/nursing`);
		expect(body.weight_before).toEqual({ unit: 'kg', amount: 3400 });
		expect(body.weight_after).toEqual({ unit: 'kg', amount: 3460 });
		const [made] = await rows(api, 'nursing');
		expect(made).toMatchObject({ milk: { gain: { grams: 60 }, volume: { ml: 58 } } });

		await page.goto(entryUrl('nursing', made.id));
		await expect(page.getByRole('link', { name: HIDE_LINK })).toBeVisible();
		await expect(kilograms(page, BEFORE)).toHaveValue('3.4');
		await expect(kilograms(page, AFTER)).toHaveValue('3.46');
		await expect(
			page.getByText(`Weight gain 60 g, about 58 ml. ${BASIS}`, { exact: true })
		).toBeVisible();
		await expect(page.locator('main')).not.toContainText(JUDGEMENT);
		await expectAccessible(page);
	});

	test('an imperial parent types pounds and ounces, reads them back and sees ounces', async ({
		page
	}) => {
		const api = await stubAs(ANA);
		expect((await api('PATCH', '/v1/me', { units: { weight: 'lb', volume: 'floz' } })).status).toBe(
			200
		);
		await signIn(page, ANA);
		await page.goto(logUrl('nursing') + ADVANCED);
		await expect(field(page, 'Weight before unit')).toHaveValue('lb');
		await expect(weight(page, BEFORE).getByLabel('Pounds')).toBeVisible();
		await expect(kilograms(page, BEFORE)).toBeHidden();
		await field(page, 'Started at').fill(inUtc(NURSING_START));
		await field(page, 'Ended at').fill(inUtc(NURSING_END));
		await field(page, 'Left minutes').fill('10');
		await weight(page, BEFORE).getByLabel('Pounds').fill('7');
		await weight(page, BEFORE).getByLabel('Ounces').fill('8');
		await weight(page, AFTER).getByLabel('Pounds').fill('7');
		await weight(page, AFTER).getByLabel('Ounces').fill('10,5');
		await saveAndWait(page);
		await onTimeline(page);
		const body = await written('POST', `/v1/kids/${MIA.id}/nursing`);
		expect(body.weight_before).toEqual({ unit: 'lb', amount: 12000 });
		expect(body.weight_after).toEqual({ unit: 'lb', amount: 12250 });
		await expect(page.getByRole('link', { name: /Nursing L 10 min/ })).toContainText(
			'weight gain 2.5 oz, about 2.33 fl oz'
		);

		const [made] = await rows(api, 'nursing');
		await page.goto(entryUrl('nursing', made.id));
		await expect(weight(page, BEFORE).getByLabel('Pounds')).toHaveValue('7');
		await expect(weight(page, BEFORE).getByLabel('Ounces')).toHaveValue('8');
		await expect(weight(page, AFTER).getByLabel('Pounds')).toHaveValue('7');
		await expect(weight(page, AFTER).getByLabel('Ounces')).toHaveValue('10.5');
		await expect(field(page, 'Weight after unit')).toHaveValue('lb');
		await expect(
			page.getByText(`Weight gain 2.5 oz, about 2.33 fl oz. ${BASIS}`, { exact: true })
		).toBeVisible();
		await field(page, 'Note').fill('weighed');
		await saveAndWait(page);
		await onTimeline(page);
		expect(await written('PATCH', `/v1/kids/${MIA.id}/nursing/${made.id}`)).toEqual({
			note: 'weighed'
		});
	});

	test('an after weight below the before weight is stored with no estimate and no message', async ({
		page
	}) => {
		const api = await stubAs(ANA);
		await signIn(page, ANA);
		await logFinished(page, '3.4', '3.3');
		await expect(page.getByRole('link', { name: /Nursing L 10 min/ })).not.toContainText('weight');
		const [made] = await rows(api, 'nursing');
		expect(made).toMatchObject({
			weight_before: { grams: 3400 },
			weight_after: { grams: 3300 },
			milk: null
		});
		await page.goto(entryUrl('nursing', made.id));
		await expect(kilograms(page, AFTER)).toHaveValue('3.3');
		await expect(page.getByText('No estimate', { exact: true })).toBeVisible();
		await expect(page.getByText(/Weight gain|The volume assumes/)).toHaveCount(0);
		await expect(page.getByRole('alert')).toHaveCount(0);
		await expect(page.locator('main')).not.toContainText(JUDGEMENT);
		await expectAccessible(page);
	});

	test('a before weight alone shows no estimate paragraph', async ({ page }) => {
		const api = await stubAs(ANA);
		await signIn(page, ANA);
		await logFinished(page, '3.4', '');
		const [made] = await rows(api, 'nursing');
		expect(made).toMatchObject({ weight_before: { grams: 3400 }, weight_after: null, milk: null });
		await page.goto(entryUrl('nursing', made.id));
		await expect(kilograms(page, BEFORE)).toHaveValue('3.4');
		await expect(kilograms(page, AFTER)).toHaveValue('');
		await expect(page.getByText(/Weight gain|No estimate|The volume assumes/)).toHaveCount(0);
	});

	test('clearing a weight sends null for it, removes it, and drops the estimate', async ({
		page
	}) => {
		const api = await stubAs(ANA);
		await signIn(page, ANA);
		await logFinished(page, '3.4', '3.46');
		const [made] = await rows(api, 'nursing');
		const path = `/v1/kids/${MIA.id}/nursing/${made.id}`;
		await page.goto(entryUrl('nursing', made.id));
		await kilograms(page, AFTER).fill('');
		await saveAndWait(page);
		await onTimeline(page);
		expect(await written('PATCH', path)).toEqual({ weight_after: null, note: null });
		expect((await rows(api, 'nursing'))[0]).toMatchObject({
			weight_before: { grams: 3400 },
			weight_after: null,
			milk: null
		});
		await expect(page.getByRole('link', { name: /Nursing L 10 min/ })).not.toContainText('weight');
		await page.goto(entryUrl('nursing', made.id));
		await expect(kilograms(page, AFTER)).toHaveValue('');
		await expect(page.getByText(/Weight gain|No estimate/)).toHaveCount(0);
	});

	test('hiding the weigh-ins keeps the stored weights', async ({ page }) => {
		const api = await stubAs(ANA);
		const nursing = await seed(api, 'nursing', {
			started_at: NURSING_START,
			ended_at: NURSING_END,
			left_seconds: 600,
			right_seconds: 300,
			weight_before: { unit: 'kg', amount: 3400 },
			weight_after: { unit: 'kg', amount: 3460 }
		});
		const path = `/v1/kids/${MIA.id}/nursing/${nursing.id}`;
		await signIn(page, ANA);
		await page.goto(entryUrl('nursing', nursing.id));
		await page.getByRole('link', { name: HIDE_LINK }).click();
		await expect(page.getByRole('link', { name: WEIGH_LINK })).toBeVisible();
		await expect(page.getByRole('group', { name: BEFORE })).toHaveCount(0);
		await field(page, 'Note').fill('plain edit');
		await saveAndWait(page);
		await onTimeline(page);
		expect(await written('PATCH', path)).toEqual({ note: 'plain edit' });
		expect((await rows(api, 'nursing'))[0]).toMatchObject({
			weight_before: { grams: 3400 },
			weight_after: { grams: 3460 }
		});
	});

	test('a running session takes a before weight, ends from the home, then gets its after weight', async ({
		page
	}) => {
		const api = await stubAs(ANA);
		await signIn(page, ANA);
		await page.goto(logUrl('nursing') + ADVANCED);
		await page.getByRole('radio', { name: 'Left' }).check();
		await kilograms(page, BEFORE).fill('3.4');
		await saveAndWait(page);
		await onTimeline(page);
		const [started] = await rows(api, 'nursing');
		expect(started).toMatchObject({ ended_at: null, weight_before: { grams: 3400 } });

		await page.goto(`/kids/${MIA.id}`);
		await submit(running(page).getByRole('button', { name: 'End nursing' }));
		await expect(running(page)).toHaveCount(0);
		expect((await rows(api, 'nursing'))[0].ended_at).not.toBeNull();

		await page.goto(entryUrl('nursing', started.id));
		await expect(kilograms(page, BEFORE)).toHaveValue('3.4');
		await expect(kilograms(page, AFTER)).toHaveValue('');
		await expect(page.getByText(/Weight gain|No estimate/)).toHaveCount(0);
		await kilograms(page, AFTER).fill('3.46');
		await saveAndWait(page);
		await onTimeline(page);
		expect(await written('PATCH', `/v1/kids/${MIA.id}/nursing/${started.id}`)).toEqual({
			weight_after: { unit: 'kg', amount: 3460 },
			note: null
		});
		await expect(page.getByRole('link', { name: /Nursing/ })).toContainText(
			'weight gain 60 g, about 58 ml'
		);
	});

	test('a running session edit shows the before weight and no estimate', async ({ page }) => {
		const api = await stubAs(ANA);
		const nursing = await seed(api, 'nursing', {
			started_at: minute(-20),
			side: 'right',
			weight_before: { unit: 'kg', amount: 3400 }
		});
		await signIn(page, ANA);
		await page.goto(entryUrl('nursing', nursing.id));
		await expect(kilograms(page, BEFORE)).toHaveValue('3.4');
		await expect(page.getByLabel('Left minutes')).toHaveCount(0);
		await expect(page.getByText(/Weight gain|No estimate/)).toHaveCount(0);
		await expectAccessible(page);
	});

	test('a weight out of range shows the message on the weight and keeps every value', async ({
		page
	}) => {
		const api = await stubAs(ANA);
		await signIn(page, ANA);
		await page.goto(logUrl('nursing') + ADVANCED);
		await field(page, 'Started at').fill(inUtc(NURSING_START));
		await field(page, 'Ended at').fill(inUtc(NURSING_END));
		await field(page, 'Left minutes').fill('10');
		await kilograms(page, BEFORE).fill('0.1');
		await kilograms(page, AFTER).fill('3,46');
		await field(page, 'Note').fill('kept');
		await saveAndWait(page);
		await expectFieldError(kilograms(page, BEFORE), 'Enter a weight');
		await expect(kilograms(page, AFTER)).toHaveValue('3,46');
		await expect(kilograms(page, BEFORE)).toHaveValue('0.1');
		await expect(field(page, 'Left minutes')).toHaveValue('10');
		await expect(field(page, 'Note')).toHaveValue('kept');
		await expect(page.getByRole('link', { name: HIDE_LINK })).toBeVisible();
		expect(await rows(api, 'nursing')).toEqual([]);
		await expectAccessible(page);

		await kilograms(page, BEFORE).fill('3.4');
		await saveAndWait(page);
		await onTimeline(page);
		expect((await rows(api, 'nursing'))[0]).toMatchObject({
			weight_before: { grams: 3400 },
			weight_after: { grams: 3460 }
		});
	});

	test('a weighed session adds nothing to the growth page', async ({ page }) => {
		const api = await stubAs(ANA);
		await signIn(page, ANA);
		await logFinished(page, '3.4', '3.46');
		expect(await growthRows(api)).toEqual([]);
		await page.goto(`/kids/${MIA.id}/growth?measure=weight`);
		await expect(page.getByText('No measurements yet')).toBeVisible();
	});

	test('a caregiver can record weigh-ins', async ({ page }) => {
		const api = await stubAs(CLEO);
		await signIn(page, ANA);
		await page.goto(logUrl('nursing', LEO.id) + ADVANCED);
		await page.getByRole('radio', { name: 'Right' }).check();
		await kilograms(page, BEFORE).fill('4.1');
		await saveAndWait(page);
		await page.waitForURL(/\/timeline/);
		const [started] = await rows(api, 'nursing', LEO.id);
		expect(started).toMatchObject({ weight_before: { grams: 4100 } });
		await page.goto(entryUrl('nursing', started.id, LEO.id));
		await kilograms(page, AFTER).fill('4.15');
		await saveAndWait(page);
		await page.waitForURL(/\/timeline/);
		expect((await rows(api, 'nursing', LEO.id))[0]).toMatchObject({
			weight_before: { grams: 4100 },
			weight_after: { grams: 4150 }
		});
	});

	test('a nursing edit opened advanced stays advanced after a stale timer result', async ({
		page
	}) => {
		const api = await stubAs(ANA);
		const nursing = await seed(api, 'nursing', {
			started_at: minute(-20),
			side: 'right',
			weight_before: { unit: 'kg', amount: 3400 }
		});
		const path = `/v1/kids/${MIA.id}/nursing/${nursing.id}`;
		await signIn(page, ANA);
		await page.goto(entryUrl('nursing', nursing.id));
		await expect(page.getByLabel('Left minutes')).toHaveCount(0);
		expect((await api('POST', `${path}/end`, {})).status).toBe(200);
		await field(page, 'Note').fill('typed while running');
		await saveAndWait(page);
		await expect(page.getByText('This timer has ended')).toBeVisible();
		await expect(field(page, 'Left minutes')).toBeVisible();
		await expect(page.locator('input[name="advanced"]')).toHaveValue('1');
		await expect(page.getByRole('link', { name: HIDE_LINK })).toBeVisible();
		await expect(kilograms(page, BEFORE)).toHaveValue('3.4');
		await expect(page.locator('input[name="weight_before_kg_original"]')).toHaveValue('3.4');
		expect((await receivedRequests()).filter((r) => r.method === 'PATCH')).toEqual([]);

		await kilograms(page, AFTER).fill('3.46');
		await saveAndWait(page);
		await onTimeline(page);
		expect(await written('PATCH', path)).toEqual({
			weight_after: { unit: 'kg', amount: 3460 },
			note: null
		});
	});

	test('a long note does not overflow the page at phone width', async ({ page }, testInfo) => {
		test.skip(testInfo.project.name !== 'phone', 'the overflow is a phone-width check');
		const api = await stubAs(ANA);
		const nursing = await seed(api, 'nursing', {
			started_at: NURSING_START,
			ended_at: NURSING_END,
			left_seconds: 600,
			right_seconds: 300,
			weight_before: { unit: 'kg', amount: 3400 },
			weight_after: { unit: 'kg', amount: 3460 },
			note: 'n'.repeat(200)
		});
		const overflows = () =>
			page.evaluate(
				() => document.documentElement.scrollWidth > document.documentElement.clientWidth
			);
		await signIn(page, ANA);
		await page.goto(entryUrl('nursing', nursing.id));
		await expect(page.getByText(/^Weight gain 60 g/)).toBeVisible();
		expect(await overflows()).toBe(false);
		await field(page, 'Weight before unit').selectOption('lb');
		expect(await overflows()).toBe(false);
		await page.goto(logUrl('nursing') + ADVANCED);
		await field(page, 'Weight after unit').selectOption('lb');
		await field(page, 'Note').fill('n'.repeat(200));
		expect(await overflows()).toBe(false);
	});
});

test.describe('the nursing weigh-ins without JavaScript', () => {
	test.use({ javaScriptEnabled: false });

	test('create, fail on a weight out of range, fix it, then edit and clear', async ({ page }) => {
		const api = await stubAs(ANA);
		await signIn(page, ANA);
		await page.goto(logUrl('nursing'));
		await page.getByRole('link', { name: WEIGH_LINK }).click();
		await expect(page).toHaveURL(new RegExp(`/log/nursing\\${ADVANCED}$`));
		await field(page, 'Started at').fill(inUtc(NURSING_START));
		await field(page, 'Ended at').fill(inUtc(NURSING_END));
		await field(page, 'Left minutes').fill('10');
		await kilograms(page, BEFORE).fill('0.1');
		await kilograms(page, AFTER).fill('3.46');
		await field(page, 'Note').fill('no script');
		await saveAndWait(page);
		await expectFieldError(kilograms(page, BEFORE), 'Enter a weight');
		await expect(kilograms(page, AFTER)).toHaveValue('3.46');
		await expect(field(page, 'Note')).toHaveValue('no script');
		await expect(page.getByRole('link', { name: HIDE_LINK })).toBeVisible();
		expect(await rows(api, 'nursing')).toEqual([]);

		await kilograms(page, BEFORE).fill('3.4');
		await saveAndWait(page);
		await onTimeline(page);
		await expect(page.getByRole('link', { name: /Nursing L 10 min/ })).toContainText(
			'weight gain 60 g, about 58 ml'
		);
		const [made] = await rows(api, 'nursing');

		await page.goto(entryUrl('nursing', made.id));
		await expect(kilograms(page, BEFORE)).toHaveValue('3.4');
		await expect(
			page.getByText(`Weight gain 60 g, about 58 ml. ${BASIS}`, { exact: true })
		).toBeVisible();
		await kilograms(page, AFTER).fill('');
		await saveAndWait(page);
		await onTimeline(page);
		expect((await rows(api, 'nursing'))[0]).toMatchObject({
			weight_before: { grams: 3400 },
			weight_after: null
		});
	});

	test('a failed save on an edit keeps the form advanced, and the link back works', async ({
		page
	}) => {
		const api = await stubAs(ANA);
		const nursing = await seed(api, 'nursing', {
			started_at: NURSING_START,
			ended_at: NURSING_END,
			left_seconds: 600,
			right_seconds: 300,
			weight_before: { unit: 'kg', amount: 3400 }
		});
		await signIn(page, ANA);
		await page.goto(entryUrl('nursing', nursing.id));
		await kilograms(page, AFTER).fill('abc');
		await saveAndWait(page);
		await expectFieldError(kilograms(page, AFTER), 'Enter a weight');
		await expect(kilograms(page, BEFORE)).toHaveValue('3.4');
		await page.getByRole('link', { name: HIDE_LINK }).click();
		await expect(page.getByRole('group', { name: BEFORE })).toHaveCount(0);
		await expect(page.getByRole('link', { name: WEIGH_LINK })).toBeVisible();
	});
});

// The advanced pumping form: a volume per side. Every save waits for its response.
const SIDES_LINK = 'Advanced: volume per side';
const ONE_TOTAL = 'Use one total';
const SIDES_HINT = 'Saving one total replaces the volumes per side';
const SIDES_MESSAGE = 'Enter a volume for at least one side';
type Side = { side: string; volume: Record<string, unknown> };
const sidesOf = (row: Row) => row.sides as Side[];
const PUMP_START = minute(-30);
const pumped = (sides: Record<string, number>, unit = 'ml') => ({
	started_at: PUMP_START,
	duration_seconds: 900,
	sides: { unit, amounts: sides }
});

async function logSides(page: Page, left: string, right: string) {
	await page.goto(logUrl('pumping') + ADVANCED);
	await field(page, 'Minutes').fill('15');
	await field(page, 'Left volume').fill(left);
	await field(page, 'Right volume').fill(right);
	await saveAndWait(page);
}

test.describe('the advanced pumping form', () => {
	test('the simple form is unchanged and offers the advanced one', async ({ page }) => {
		await stubAs(ANA);
		await signIn(page, ANA);
		await page.goto(logUrl('pumping'));
		await expect(field(page, 'Volume')).toBeVisible();
		await expect(field(page, 'Volume unit')).toHaveValue('ml');
		await expect(page.getByRole('radio')).toHaveCount(3);
		await expect(field(page, 'Left volume')).toHaveCount(0);
		await expect(page.getByText(/^Total:/)).toHaveCount(0);
		await expect(page.getByText(SIDES_HINT)).toHaveCount(0);
		await expect(page.locator('input[name="advanced"]')).toHaveCount(0);
		await expectAccessible(page);
		await page.getByRole('link', { name: SIDES_LINK }).click();
		await expect(page).toHaveURL(new RegExp(`/log/pumping\\${ADVANCED}$`));
		await expect(page.getByRole('link', { name: ONE_TOTAL })).toBeVisible();
	});

	test('?advanced=1 shows one input per side, one unit and no side radio', async ({ page }) => {
		await stubAs(ANA);
		await signIn(page, ANA);
		await page.goto(logUrl('pumping') + ADVANCED);
		await expect(page.getByRole('heading', { name: 'Log pumping', level: 1 })).toBeVisible();
		await expect(field(page, 'Volume unit')).toBeVisible();
		await expect(field(page, 'Left volume')).toBeVisible();
		await expect(field(page, 'Right volume')).toBeVisible();
		await expect(field(page, 'Volume')).toHaveCount(0);
		await expect(page.getByRole('radio')).toHaveCount(0);
		await expect(page.getByText('Total: 0 ml')).toBeVisible();
		await expect(page.locator('input[name="advanced"]')).toHaveValue('1');
		await expectAccessible(page);
	});

	test('left 60 and right 50 is one 110 ml session with its sides', async ({ page }) => {
		const api = await stubAs(ANA);
		await signIn(page, ANA);
		await logSides(page, '60', '50');
		await onTimeline(page);
		await expect(page.getByText(/Pumping 15 min, 110 ml \(L 60 ml · R 50 ml\)/)).toBeVisible();
		const [created] = await rows(api, 'pumping');
		expect(created).toMatchObject({ side: 'both', volume: { ml: 110, typed_unit: 'ml' } });
		expect(sidesOf(created).map((s) => [s.side, s.volume.ml])).toEqual([
			['left', 60],
			['right', 50]
		]);
		const body = await written('POST', `/v1/kids/${MIA.id}/pumping`);
		expect(body.sides).toEqual({ unit: 'ml', amounts: { left: 60, right: 50 } });
		expect(body).not.toHaveProperty('volume');
		expect(body).not.toHaveProperty('side');

		await page.goto(entryUrl('pumping', created.id));
		await expect(page.getByRole('link', { name: ONE_TOTAL })).toBeVisible();
		await expect(field(page, 'Left volume')).toHaveValue('60');
		await expect(field(page, 'Right volume')).toHaveValue('50');
		await expect(page.getByRole('radio')).toHaveCount(0);
		await expect(page.getByText('Total: 110 ml')).toBeVisible();
		await expectAccessible(page);

		await field(page, 'Right volume').fill('55');
		await saveAndWait(page);
		await onTimeline(page);
		const [edited] = await rows(api, 'pumping');
		expect(sidesOf(edited).map((s) => [s.side, s.volume.ml])).toEqual([
			['left', 60],
			['right', 55]
		]);
		expect(edited.volume).toMatchObject({ ml: 115 });
		expect(sidesOf(edited)[0]).toEqual(sidesOf(created)[0]);
		await expect(page.getByText(/Pumping 15 min, 115 ml \(L 60 ml · R 55 ml\)/)).toBeVisible();
	});

	test('left only saves as left, and the other side stays blank on edit', async ({ page }) => {
		const api = await stubAs(ANA);
		await signIn(page, ANA);
		await logSides(page, '60', '');
		await onTimeline(page);
		await expect(page.getByText(/Pumping 15 min, 60 ml \(L 60 ml\)/)).toBeVisible();
		const [created] = await rows(api, 'pumping');
		expect(created).toMatchObject({ side: 'left', volume: { ml: 60 } });
		expect(sidesOf(created)).toHaveLength(1);
		const body = await written('POST', `/v1/kids/${MIA.id}/pumping`);
		expect(body.sides).toEqual({ unit: 'ml', amounts: { left: 60 } });

		await page.goto(entryUrl('pumping', created.id));
		await expect(field(page, 'Left volume')).toHaveValue('60');
		await expect(field(page, 'Right volume')).toHaveValue('');
		await expect(page.getByText('Total: 60 ml')).toBeVisible();
		await field(page, 'Right volume').fill('40');
		await saveAndWait(page);
		await onTimeline(page);
		const [both] = await rows(api, 'pumping');
		expect(both).toMatchObject({ side: 'both', volume: { ml: 100 } });
		expect(sidesOf(both)[0]).toEqual(sidesOf(created)[0]);
	});

	test('editing only the note of an advanced session sends neither sides nor volume', async ({
		page
	}) => {
		const api = await stubAs(ANA);
		const session = await seed(api, 'pumping', pumped({ left: 60, right: 50 }));
		await signIn(page, ANA);
		await page.goto(entryUrl('pumping', session.id));
		await field(page, 'Note').fill('only the note');
		await saveAndWait(page);
		await onTimeline(page);
		const body = await written('PATCH', `/v1/kids/${MIA.id}/pumping/${session.id}`);
		expect(body).toEqual({ note: 'only the note' });
		const [kept] = await rows(api, 'pumping');
		expect(kept.volume).toMatchObject({ ml: 110 });
		expect(sidesOf(kept)).toHaveLength(2);
	});

	test('a parent with fl oz types fl oz per side and reads them back exactly', async ({ page }) => {
		const api = await stubAs(ANA);
		expect((await api('PATCH', '/v1/me', { units: { volume: 'floz' } })).status).toBe(200);
		await signIn(page, ANA);
		await logSides(page, '2,5', '3.25');
		await onTimeline(page);
		await expect(page.getByText(/\(L 2\.5 fl oz · R 3\.25 fl oz\)/)).toBeVisible();
		const [created] = await rows(api, 'pumping');
		expect(created.volume).toMatchObject({ floz_hundredths: 575, typed_unit: 'floz' });
		expect(sidesOf(created).map((s) => [s.side, s.volume.floz_hundredths])).toEqual([
			['left', 250],
			['right', 325]
		]);
		await page.goto(entryUrl('pumping', created.id));
		await expect(field(page, 'Volume unit')).toHaveValue('floz');
		await expect(field(page, 'Left volume')).toHaveValue('2.5');
		await expect(field(page, 'Right volume')).toHaveValue('3.25');
		await expect(page.getByText('Total: 5.75 fl oz')).toBeVisible();
	});

	test('both sides blank shows the message and keeps what was typed', async ({ page }) => {
		const api = await stubAs(ANA);
		await signIn(page, ANA);
		await page.goto(logUrl('pumping') + ADVANCED);
		await field(page, 'Minutes').fill('15');
		await field(page, 'Note').fill('keep me');
		await saveAndWait(page);
		await expectFieldError(field(page, 'Left volume'), SIDES_MESSAGE);
		await expectFieldError(field(page, 'Right volume'), SIDES_MESSAGE);
		await expect(page.getByRole('alert').filter({ hasText: SIDES_MESSAGE })).toHaveCount(1);
		await expect(field(page, 'Minutes')).toHaveValue('15');
		await expect(field(page, 'Note')).toHaveValue('keep me');
		await expect(page.getByText('Total: 0 ml')).toBeVisible();
		await expect(page.getByRole('link', { name: ONE_TOTAL })).toBeVisible();
		await expectAccessible(page);
		expect(await rows(api, 'pumping')).toEqual([]);

		await field(page, 'Right volume').fill('30');
		await saveAndWait(page);
		await onTimeline(page);
		expect(await rows(api, 'pumping')).toHaveLength(1);
	});

	test('text that is not a volume is refused with the values kept', async ({ page }) => {
		const api = await stubAs(ANA);
		await signIn(page, ANA);
		await page.goto(logUrl('pumping') + ADVANCED);
		await field(page, 'Minutes').fill('15');
		await field(page, 'Left volume').fill('1e3');
		await field(page, 'Right volume').fill('50');
		await saveAndWait(page);
		await expectFieldError(field(page, 'Left volume'), SIDES_MESSAGE);
		await expect(field(page, 'Left volume')).toHaveValue('1e3');
		await expect(page.getByText('Total: 50 ml')).toBeVisible();
		expect(await rows(api, 'pumping')).toEqual([]);
	});

	test('a total over the range is refused on the sides, keeping the values', async ({ page }) => {
		const api = await stubAs(ANA);
		await signIn(page, ANA);
		await logSides(page, '1500', '600');
		await expectFieldError(field(page, 'Right volume'), 'The total must be from 1 to 2000 ml');
		await expect(page.getByText('Total: 2100 ml')).toBeVisible();
		await expect(field(page, 'Left volume')).toHaveValue('1500');
		await expectAccessible(page);
		expect(await rows(api, 'pumping')).toEqual([]);
	});

	test('a side typed and then blanked is not sent', async ({ page }) => {
		const api = await stubAs(ANA);
		await signIn(page, ANA);
		await page.goto(logUrl('pumping') + ADVANCED);
		await field(page, 'Minutes').fill('15');
		await field(page, 'Left volume').fill('60');
		await field(page, 'Right volume').fill('50');
		await field(page, 'Right volume').fill('');
		await saveAndWait(page);
		await onTimeline(page);
		const body = await written('POST', `/v1/kids/${MIA.id}/pumping`);
		expect(body.sides).toEqual({ unit: 'ml', amounts: { left: 60 } });
		expect((await rows(api, 'pumping'))[0]).toMatchObject({ side: 'left' });
	});

	test('"Use one total" over an advanced session saves a plain session and the sides are gone', async ({
		page
	}) => {
		const api = await stubAs(ANA);
		const session = await seed(api, 'pumping', pumped({ left: 60, right: 50 }));
		const path = `/v1/kids/${MIA.id}/pumping/${session.id}`;
		await signIn(page, ANA);
		await page.goto(entryUrl('pumping', session.id));
		await page.getByRole('link', { name: ONE_TOTAL }).click();
		await expect(page).toHaveURL(new RegExp(`/${session.id}\\?advanced=0$`));
		await expect(field(page, 'Volume')).toHaveValue('110');
		await expect(field(page, 'Left volume')).toHaveCount(0);
		await expect(page.getByRole('radio', { name: 'Both' })).toBeChecked();
		await expect(page.getByText(SIDES_HINT)).toBeVisible();
		// The hidden original is what makes an untouched total replace the sides.
		await expect(page.locator('input[name="sides_original"]')).toHaveValue('left,right');
		await expectAccessible(page);
		await saveAndWait(page);
		await onTimeline(page);
		expect(await written('PATCH', path)).toMatchObject({
			volume: { unit: 'ml', amount: 110 },
			side: 'both'
		});
		const [plain] = await rows(api, 'pumping');
		expect(sidesOf(plain)).toEqual([]);
		expect(plain).toMatchObject({ side: 'both', volume: { ml: 110 } });
		await expect(page.getByText(/Pumping 15 min, 110 ml \(both\)/)).toBeVisible();

		await page.goto(entryUrl('pumping', session.id));
		await expect(page.getByRole('link', { name: SIDES_LINK })).toBeVisible();
		await expect(field(page, 'Volume')).toHaveValue('110');
		await expect(page.getByText(SIDES_HINT)).toHaveCount(0);
	});

	test('the simple form over an advanced session sends the changed total and side', async ({
		page
	}) => {
		const api = await stubAs(ANA);
		const session = await seed(api, 'pumping', pumped({ left: 60, right: 50 }));
		await signIn(page, ANA);
		await page.goto(entryUrl('pumping', session.id) + '?advanced=0');
		await field(page, 'Volume').fill('90');
		await page.getByRole('radio', { name: 'Left' }).check();
		await saveAndWait(page);
		await onTimeline(page);
		const [plain] = await rows(api, 'pumping');
		expect(sidesOf(plain)).toEqual([]);
		expect(plain).toMatchObject({ side: 'left', volume: { ml: 90 } });
	});

	test('a failed save over a session with sides stays advanced and keeps the values', async ({
		page
	}) => {
		const api = await stubAs(ANA);
		const session = await seed(api, 'pumping', pumped({ left: 60, right: 50 }));
		await signIn(page, ANA);
		await page.goto(entryUrl('pumping', session.id));
		await field(page, 'Left volume').fill('abc');
		await saveAndWait(page);
		await expectFieldError(field(page, 'Left volume'), SIDES_MESSAGE);
		await expect(page.getByRole('link', { name: ONE_TOTAL })).toBeVisible();
		await expect(field(page, 'Left volume')).toHaveValue('abc');
		await expect(field(page, 'Right volume')).toHaveValue('50');
		await expect(field(page, 'Volume')).toHaveCount(0);
		await expect(page.getByRole('radio')).toHaveCount(0);
		expect(sidesOf((await rows(api, 'pumping'))[0])).toHaveLength(2);

		await field(page, 'Left volume').fill('65');
		await saveAndWait(page);
		await onTimeline(page);
		expect((await rows(api, 'pumping'))[0].volume).toMatchObject({ ml: 115 });
	});

	test('a session created simple opens simple, and can be opened advanced', async ({ page }) => {
		const api = await stubAs(ANA);
		const session = await seed(api, 'pumping', {
			started_at: PUMP_START,
			duration_seconds: 900,
			volume: { unit: 'ml', amount: 120 },
			side: 'right'
		});
		await signIn(page, ANA);
		await page.goto(entryUrl('pumping', session.id));
		await expect(field(page, 'Volume')).toHaveValue('120');
		await expect(page.getByRole('radio', { name: 'Right' })).toBeChecked();
		await expect(field(page, 'Left volume')).toHaveCount(0);
		await expect(page.getByText(SIDES_HINT)).toHaveCount(0);
		await page.getByRole('link', { name: SIDES_LINK }).click();
		await expect(page).toHaveURL(new RegExp(`/${session.id}\\${ADVANCED}$`));
		await expect(field(page, 'Left volume')).toHaveValue('');
		await expect(page.getByText('Total: 0 ml')).toBeVisible();
		await field(page, 'Left volume').fill('50');
		await field(page, 'Right volume').fill('70');
		await saveAndWait(page);
		await onTimeline(page);
		const [advanced] = await rows(api, 'pumping');
		expect(advanced).toMatchObject({ side: 'both', volume: { ml: 120 } });
		expect(sidesOf(advanced).map((s) => [s.side, s.volume.ml])).toEqual([
			['left', 50],
			['right', 70]
		]);
	});

	test('a caregiver can log an advanced session', async ({ page }) => {
		const api = await stubAs(CLEO);
		await signIn(page, ANA);
		await page.goto(logUrl('pumping', LEO.id) + ADVANCED);
		await field(page, 'Minutes').fill('10');
		await field(page, 'Right volume').fill('80');
		await saveAndWait(page);
		await onTimeline(page);
		const [created] = await rows(api, 'pumping', LEO.id);
		expect(created).toMatchObject({ side: 'right', volume: { ml: 80 } });
	});

	test('a long note does not overflow the page at phone width', async ({ page }, testInfo) => {
		test.skip(testInfo.project.name !== 'phone', 'the overflow is a phone-width check');
		const api = await stubAs(ANA);
		const session = await seed(api, 'pumping', {
			...pumped({ left: 60, right: 50 }),
			note: 'n'.repeat(200)
		});
		const overflows = () =>
			page.evaluate(
				() => document.documentElement.scrollWidth > document.documentElement.clientWidth
			);
		await signIn(page, ANA);
		await page.goto(entryUrl('pumping', session.id));
		await expect(page.getByText('Total: 110 ml')).toBeVisible();
		expect(await overflows()).toBe(false);
		await page.goto(logUrl('pumping') + ADVANCED);
		await field(page, 'Note').fill('n'.repeat(200));
		expect(await overflows()).toBe(false);
	});
});

test.describe('the advanced pumping form without JavaScript', () => {
	test.use({ javaScriptEnabled: false });

	test('create, fail on both sides blank, fix it, then edit and switch to one total', async ({
		page
	}) => {
		const api = await stubAs(ANA);
		await signIn(page, ANA);
		await page.goto(logUrl('pumping'));
		await page.getByRole('link', { name: SIDES_LINK }).click();
		await expect(page).toHaveURL(new RegExp(`${ADVANCED.replace('?', '\\?')}$`));
		await field(page, 'Minutes').fill('15');
		await field(page, 'Note').fill('no script');
		await saveAndWait(page);
		await expectFieldError(field(page, 'Left volume'), SIDES_MESSAGE);
		await expect(field(page, 'Minutes')).toHaveValue('15');
		await expect(field(page, 'Note')).toHaveValue('no script');
		await expect(page.getByText('Total: 0 ml')).toBeVisible();
		expect(await rows(api, 'pumping')).toEqual([]);

		await field(page, 'Left volume').fill('60');
		await field(page, 'Right volume').fill('50');
		await saveAndWait(page);
		await onTimeline(page);
		await expect(page.getByText(/Pumping 15 min, 110 ml \(L 60 ml · R 50 ml\)/)).toBeVisible();
		const [created] = await rows(api, 'pumping');
		expect(sidesOf(created)).toHaveLength(2);

		await page.goto(entryUrl('pumping', created.id));
		await expect(field(page, 'Right volume')).toHaveValue('50');
		await expect(page.getByText('Total: 110 ml')).toBeVisible();
		await field(page, 'Right volume').fill('');
		await saveAndWait(page);
		await onTimeline(page);
		expect((await rows(api, 'pumping'))[0]).toMatchObject({ side: 'left', volume: { ml: 60 } });

		await page.goto(entryUrl('pumping', created.id));
		await page.getByRole('link', { name: ONE_TOTAL }).click();
		await expect(field(page, 'Volume')).toHaveValue('60');
		await saveAndWait(page);
		await onTimeline(page);
		const [plain] = await rows(api, 'pumping');
		expect(sidesOf(plain)).toEqual([]);
		expect(plain).toMatchObject({ side: 'left', volume: { ml: 60 } });
	});

	test('a failed simple save over an advanced session stays simple and keeps the hint', async ({
		page
	}) => {
		const api = await stubAs(ANA);
		const session = await seed(api, 'pumping', pumped({ left: 60, right: 50 }));
		await signIn(page, ANA);
		await page.goto(entryUrl('pumping', session.id) + '?advanced=0');
		await field(page, 'Volume').fill('');
		await saveAndWait(page);
		// The post went to ?/save: the query string is gone, the form must not turn advanced.
		await expectFieldError(field(page, 'Volume'), 'Enter a volume');
		await expect(field(page, 'Left volume')).toHaveCount(0);
		await expect(page.getByText(SIDES_HINT)).toBeVisible();
		await expect(page.getByRole('link', { name: SIDES_LINK })).toBeVisible();
		expect(sidesOf((await rows(api, 'pumping'))[0])).toHaveLength(2);

		await field(page, 'Volume').fill('90');
		await saveAndWait(page);
		await onTimeline(page);
		const [plain] = await rows(api, 'pumping');
		expect(sidesOf(plain)).toEqual([]);
		expect(plain).toMatchObject({ side: 'both', volume: { ml: 90 } });
	});

	test('the link back to one total works from a failed post', async ({ page }) => {
		await stubAs(ANA);
		await signIn(page, ANA);
		await page.goto(logUrl('pumping') + ADVANCED);
		await field(page, 'Minutes').fill('15');
		await saveAndWait(page);
		await expectFieldError(field(page, 'Left volume'), SIDES_MESSAGE);
		await page.getByRole('link', { name: ONE_TOTAL }).click();
		await expect(field(page, 'Volume')).toBeVisible();
		await expect(field(page, 'Left volume')).toHaveCount(0);
	});
});

// The sickness tracker: a symptom, and for a fever its temperature and method. Every save waits for its response.
const SYMPTOM_PATH = `/v1/kids/${MIA.id}/symptoms`;
const TEMP_MESSAGE = 'Enter a temperature from 30 to 45 °C (86 to 113 °F), with up to two decimals';
const pick = (page: Page, symptom: string) => field(page, 'Symptom').selectOption(symptom);
// Hydration overwrites a control changed before it; Svelte drops the SSR value attribute after it.
const hydrated = (page: Page) => expect(field(page, 'Temperature')).not.toHaveAttribute('value');
const fever = (over: Row = {}): Row => ({
	at: minute(-10),
	symptom: 'fever',
	temperature: { unit: 'c', amount: 3840 },
	method: 'ear',
	...over
});
const setUnits = async (api: Api, temp: 'c' | 'f') =>
	expect((await api('PATCH', '/v1/me', { units: { temp } })).status).toBe(200);
const symptomOverflows = (page: Page) =>
	page.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth);

test.describe('the sickness tracker', () => {
	test('a fever is logged with its temperature and method, and read back in °F by an imperial parent', async ({
		page
	}) => {
		const api = await stubAs(ANA);
		await setUnits(api, 'f');
		await signIn(page, ANA);
		await page.goto(logUrl('symptoms'));
		await hydrated(page);
		await expect(page.getByRole('heading', { name: 'Log a symptom', level: 1 })).toBeVisible();
		await expect(field(page, 'Left minutes')).toHaveCount(0);
		await expect(field(page, 'Temperature')).toBeHidden();
		await expect(field(page, 'Method')).toBeHidden();
		await expect(field(page, 'Other symptom')).toBeHidden();
		await expect(page.getByRole('radio', { name: 'Not set' })).toBeChecked();
		await expectAccessible(page);

		await pick(page, 'fever');
		await expect(field(page, 'Temperature')).toBeVisible();
		await expect(field(page, 'Temperature unit')).toHaveValue('f');
		await expect(field(page, 'Other symptom')).toBeHidden();
		await expectAccessible(page);
		await field(page, 'Temperature').fill('101,3');
		await field(page, 'Method').selectOption('ear');
		await field(page, 'Note').fill('fussy');
		await saveAndWait(page);
		await onTimeline(page);

		expect(await written('POST', SYMPTOM_PATH)).toMatchObject({
			symptom: 'fever',
			temperature: { unit: 'f', amount: 10130 },
			method: 'ear',
			note: 'fussy'
		});
		const [made] = await rows(api, 'symptoms');
		expect(made).toMatchObject({
			symptom: 'fever',
			method: 'ear',
			temperature: { f_hundredths: 10130, typed_unit: 'f' }
		});
		await expect(
			page.getByRole('link', { name: /^\d\d:\d\d Fever 101\.3 °F \(ear\)/ })
		).toBeVisible();

		await page.goto(entryUrl('symptoms', made.id));
		await expect(page.getByRole('heading', { name: 'Edit symptom', level: 1 })).toBeVisible();
		await expect(field(page, 'Symptom')).toHaveValue('fever');
		await expect(field(page, 'Temperature')).toHaveValue('101.3');
		await expect(field(page, 'Temperature unit')).toHaveValue('f');
		await expect(field(page, 'Method')).toHaveValue('ear');
		await expectAccessible(page);

		await setUnits(api, 'c');
		await page.goto(entryUrl('symptoms', made.id));
		await expect(field(page, 'Temperature')).toHaveValue('38.5');
		await expect(field(page, 'Temperature unit')).toHaveValue('c');
	});

	test('a Celsius fever reads back in °F, and editing only the note keeps what was typed', async ({
		page
	}) => {
		const api = await stubAs(ANA);
		const made = await seed(api, 'symptoms', fever());
		await setUnits(api, 'f');
		await signIn(page, ANA);
		await page.goto(entryUrl('symptoms', made.id));
		await hydrated(page);
		await expect(field(page, 'Temperature')).toHaveValue('101.12');
		await expect(field(page, 'Temperature unit')).toHaveValue('f');
		await field(page, 'Note').fill('warm');
		await saveAndWait(page);
		await onTimeline(page);
		expect(await written('PATCH', `${SYMPTOM_PATH}/${made.id}`)).toEqual({ note: 'warm' });
		const [kept] = await rows(api, 'symptoms');
		expect(kept.temperature).toEqual({ c_hundredths: 3840, f_hundredths: 10112, typed_unit: 'c' });
		expect(kept.at).toBe(made.at);
		await expect(page.getByRole('link', { name: /Fever 101\.12 °F \(ear\)/ })).toBeVisible();
	});

	test('a changed temperature is sent alone, with its unit', async ({ page }) => {
		const api = await stubAs(ANA);
		const made = await seed(api, 'symptoms', fever());
		await signIn(page, ANA);
		await page.goto(entryUrl('symptoms', made.id));
		await hydrated(page);
		await field(page, 'Temperature').fill('39');
		await saveAndWait(page);
		await onTimeline(page);
		expect(await written('PATCH', `${SYMPTOM_PATH}/${made.id}`)).toEqual({
			temperature: { unit: 'c', amount: 3900 },
			note: null
		});
		expect((await rows(api, 'symptoms'))[0]).toMatchObject({
			method: 'ear',
			temperature: { c_hundredths: 3900 }
		});

		await page.goto(entryUrl('symptoms', made.id));
		await hydrated(page);
		await field(page, 'Method').selectOption('oral');
		await saveAndWait(page);
		await onTimeline(page);
		expect(await written('PATCH', `${SYMPTOM_PATH}/${made.id}`)).toEqual({
			method: 'oral',
			note: null
		});
	});

	test('a cough with a severity is logged with no temperature, and the severity can be cleared', async ({
		page
	}) => {
		const api = await stubAs(ANA);
		await signIn(page, ANA);
		await page.goto(logUrl('symptoms'));
		await hydrated(page);
		await pick(page, 'cough');
		await expect(field(page, 'Temperature')).toBeHidden();
		await page.getByRole('radio', { name: 'Moderate' }).check();
		await saveAndWait(page);
		await onTimeline(page);
		const body = await written('POST', SYMPTOM_PATH);
		expect(body).toMatchObject({ symptom: 'cough', severity: 'moderate' });
		for (const key of ['temperature', 'method', 'other_text']) expect(body).not.toHaveProperty(key);
		const [made] = await rows(api, 'symptoms');
		expect(made).toMatchObject({ symptom: 'cough', severity: 'moderate', temperature: null });
		await expect(page.getByRole('link', { name: /^\d\d:\d\d Cough, moderate/ })).toBeVisible();

		await page.goto(entryUrl('symptoms', made.id));
		await hydrated(page);
		await expect(page.getByRole('radio', { name: 'Moderate' })).toBeChecked();
		await expectAccessible(page);
		await page.getByRole('radio', { name: 'Not set' }).check();
		await saveAndWait(page);
		await onTimeline(page);
		expect(await written('PATCH', `${SYMPTOM_PATH}/${made.id}`)).toEqual({
			severity: null,
			note: null
		});
		expect((await rows(api, 'symptoms'))[0].severity).toBeNull();
	});

	test('other requires its text, and keeps what was typed until it has it', async ({ page }) => {
		const api = await stubAs(ANA);
		await signIn(page, ANA);
		await page.goto(logUrl('symptoms'));
		await hydrated(page);
		await pick(page, 'other');
		await expect(field(page, 'Other symptom')).toBeVisible();
		await expect(field(page, 'Temperature')).toBeHidden();
		await field(page, 'Note').fill('kept');
		await saveAndWait(page);
		await expectFieldError(field(page, 'Other symptom'), 'Enter what the symptom is');
		await expect(field(page, 'Symptom')).toHaveValue('other');
		await expect(field(page, 'Note')).toHaveValue('kept');
		await expectAccessible(page);
		expect(await rows(api, 'symptoms')).toEqual([]);

		await field(page, 'Other symptom').fill('hiccups');
		await saveAndWait(page);
		await onTimeline(page);
		expect((await rows(api, 'symptoms'))[0]).toMatchObject({
			symptom: 'other',
			other_text: 'hiccups',
			note: 'kept'
		});
		await expect(page.getByRole('link', { name: /^\d\d:\d\d Other: hiccups/ })).toBeVisible();
	});

	test('a temperature out of range keeps every typed value, on the amount and its unit', async ({
		page
	}) => {
		const api = await stubAs(ANA);
		await signIn(page, ANA);
		await page.goto(logUrl('symptoms'));
		await hydrated(page);
		await pick(page, 'fever');
		await field(page, 'Temperature').fill('50');
		await field(page, 'Method').selectOption('rectal');
		await field(page, 'Note').fill('keep me');
		await saveAndWait(page);
		await expectFieldError(field(page, 'Temperature'), TEMP_MESSAGE);
		await expect(field(page, 'Temperature unit')).toHaveAttribute('aria-invalid', 'true');
		await expect(page.getByRole('alert')).toHaveText(TEMP_MESSAGE);
		await expect(field(page, 'Symptom')).toHaveValue('fever');
		await expect(field(page, 'Temperature')).toHaveValue('50');
		await expect(field(page, 'Method')).toHaveValue('rectal');
		await expect(field(page, 'Note')).toHaveValue('keep me');
		await expectAccessible(page);
		expect(await rows(api, 'symptoms')).toEqual([]);

		await field(page, 'Temperature').fill('38,5');
		await saveAndWait(page);
		await onTimeline(page);
		expect((await rows(api, 'symptoms'))[0].temperature).toMatchObject({
			c_hundredths: 3850,
			typed_unit: 'c'
		});
	});

	test('text that is not a temperature is refused by the form, never sent', async ({ page }) => {
		const api = await stubAs(ANA);
		await signIn(page, ANA);
		await page.goto(logUrl('symptoms'));
		await hydrated(page);
		await pick(page, 'fever');
		await field(page, 'Method').selectOption('oral');
		for (const typed of ['1e3', '0x10', '--1', '38.456', '']) {
			await field(page, 'Temperature').fill(typed);
			await saveAndWait(page);
			await expectFieldError(field(page, 'Temperature'), TEMP_MESSAGE);
			await expect(field(page, 'Temperature')).toHaveValue(typed);
			if (typed !== '') await expect(page.getByRole('alert')).not.toContainText(typed);
		}
		expect(await rows(api, 'symptoms')).toEqual([]);
		const posts = (await receivedRequests()).filter(
			(r) => r.method === 'POST' && r.path === SYMPTOM_PATH
		);
		expect(posts).toEqual([]);
	});

	test('a fever without a method is refused on the method', async ({ page }) => {
		await stubAs(ANA);
		await signIn(page, ANA);
		await page.goto(logUrl('symptoms'));
		await hydrated(page);
		await pick(page, 'fever');
		await field(page, 'Temperature').fill('38');
		await saveAndWait(page);
		await expectFieldError(field(page, 'Method'), 'Choose how the temperature was taken');
		await expect(field(page, 'Temperature')).toHaveValue('38');
		await expectAccessible(page);
	});

	test('no symptom chosen is refused on the symptom', async ({ page }) => {
		await stubAs(ANA);
		await signIn(page, ANA);
		await page.goto(logUrl('symptoms'));
		await saveAndWait(page);
		await expectFieldError(field(page, 'Symptom'), 'Choose a symptom');
		await expectAccessible(page);
	});

	test('a fever edited into a cough, then a fever, then other, then a cough sends the nulls each time', async ({
		page
	}) => {
		const api = await stubAs(ANA);
		const made = await seed(api, 'symptoms', fever());
		const path = `${SYMPTOM_PATH}/${made.id}`;
		const edit = async () => {
			await page.goto(entryUrl('symptoms', made.id));
			await hydrated(page);
			await expect(page.getByRole('heading', { name: 'Edit symptom', level: 1 })).toBeVisible();
		};
		await signIn(page, ANA);

		await edit();
		await pick(page, 'cough');
		await page.getByRole('radio', { name: 'Mild' }).check();
		await saveAndWait(page);
		await onTimeline(page);
		expect(await written('PATCH', path)).toEqual({
			symptom: 'cough',
			severity: 'mild',
			temperature: null,
			method: null,
			note: null
		});
		expect((await rows(api, 'symptoms'))[0]).toMatchObject({
			symptom: 'cough',
			severity: 'mild',
			temperature: null,
			method: null
		});

		await edit();
		await pick(page, 'fever');
		await field(page, 'Temperature').fill('39');
		await field(page, 'Method').selectOption('oral');
		await saveAndWait(page);
		await onTimeline(page);
		expect(await written('PATCH', path)).toEqual({
			symptom: 'fever',
			temperature: { unit: 'c', amount: 3900 },
			method: 'oral',
			note: null
		});

		await edit();
		await pick(page, 'other');
		await field(page, 'Other symptom').fill('hiccups');
		await saveAndWait(page);
		await onTimeline(page);
		expect(await written('PATCH', path)).toEqual({
			symptom: 'other',
			other_text: 'hiccups',
			temperature: null,
			method: null,
			note: null
		});

		await edit();
		await expect(field(page, 'Other symptom')).toHaveValue('hiccups');
		await pick(page, 'cough');
		await saveAndWait(page);
		await onTimeline(page);
		expect(await written('PATCH', path)).toEqual({
			symptom: 'cough',
			other_text: null,
			note: null
		});
		expect((await rows(api, 'symptoms'))[0]).toMatchObject({
			symptom: 'cough',
			other_text: null,
			temperature: null
		});
	});

	test('a symptom is deleted behind the confirm', async ({ page }) => {
		const api = await stubAs(ANA);
		const made = await seed(api, 'symptoms', fever());
		await signIn(page, ANA);
		await page.goto(entryUrl('symptoms', made.id));
		await hydrated(page);
		await expect(page.getByRole('button', { name: 'Yes, delete this entry' })).toBeHidden();
		await page.getByText('Delete this entry', { exact: true }).click();
		await submit(page.getByRole('button', { name: 'Yes, delete this entry' }));
		await onTimeline(page);
		expect(await rows(api, 'symptoms')).toEqual([]);
	});

	for (const [who, role] of [
		[ANA, 'caregiver'],
		[CLEO, 'guardian']
	] as const) {
		test(`a ${role} can log, edit and delete a fever`, async ({ page }) => {
			const api = await stubAs(CLEO);
			await signIn(page, who);
			await page.goto(logUrl('symptoms', LEO.id));
			await hydrated(page);
			await pick(page, 'fever');
			await field(page, 'Temperature').fill('38.2');
			await field(page, 'Method').selectOption('forehead');
			await saveAndWait(page);
			await onTimeline(page);
			const [made] = await rows(api, 'symptoms', LEO.id);
			expect(made).toMatchObject({ symptom: 'fever', method: 'forehead' });
			expect(made.temperature).toMatchObject({ c_hundredths: 3820 });

			await page.goto(entryUrl('symptoms', made.id, LEO.id));
			await hydrated(page);
			await field(page, 'Temperature').fill('38.6');
			await saveAndWait(page);
			await onTimeline(page);
			expect(((await rows(api, 'symptoms', LEO.id))[0].temperature as Row).c_hundredths).toBe(3860);

			await page.goto(entryUrl('symptoms', made.id, LEO.id));
			await hydrated(page);
			await page.getByText('Delete this entry', { exact: true }).click();
			await submit(page.getByRole('button', { name: 'Yes, delete this entry' }));
			await onTimeline(page);
			expect(await rows(api, 'symptoms', LEO.id)).toEqual([]);
		});
	}

	test('a parent without a grant, an unknown kid, an unknown entry and another kid’s entry get the same 404 page', async ({
		page
	}) => {
		const api = await stubAs(ANA);
		const mine = await seed(api, 'symptoms', fever());
		const others = await seed(await stubAs(CLEO), 'symptoms', fever(), LEO.id);
		await signIn(page, DAN);
		const unknown = await expectNotFoundPage(page, logUrl('symptoms', 'k-unknown'));
		expect(await expectNotFoundPage(page, logUrl('symptoms'))).toBe(unknown);
		expect(await expectNotFoundPage(page, entryUrl('symptoms', mine.id))).toBe(unknown);

		await page.context().clearCookies();
		await signIn(page, ANA);
		expect(await expectNotFoundPage(page, entryUrl('symptoms', 'e-000000009999'))).toBe(unknown);
		expect(await expectNotFoundPage(page, entryUrl('symptoms', others.id))).toBe(unknown);
		expect(await expectNotFoundPage(page, entryUrl('diapers', mine.id))).toBe(unknown);
	});

	test('the Sickness tile of a kid over 24 months opens the symptom form, not another', async ({
		page
	}) => {
		await stubAs(ANA);
		await signIn(page, ANA);
		await page.goto(`/kids/${LEO.id}`);
		await page.getByRole('link', { name: 'Sickness', exact: true }).click();
		await expect(page).toHaveURL(`/kids/${LEO.id}/log/symptoms`);
		await expect(page.getByRole('heading', { name: 'Log a symptom', level: 1 })).toBeVisible();
		await expect(field(page, 'Symptom')).toBeVisible();
		await expect(field(page, 'Left minutes')).toHaveCount(0);
		await expectAccessible(page);
	});

	test('a long other text and a long note do not overflow the page at phone width', async ({
		page
	}, testInfo) => {
		test.skip(testInfo.project.name !== 'phone', 'the overflow is a phone-width check');
		const api = await stubAs(ANA);
		const longText = 'w'.repeat(80);
		const made = await seed(api, 'symptoms', {
			at: minute(-10),
			symptom: 'other',
			other_text: longText,
			note: 'n'.repeat(200)
		});
		await signIn(page, ANA);
		await page.goto(entryUrl('symptoms', made.id));
		await expect(field(page, 'Other symptom')).toHaveValue(longText);
		expect(await symptomOverflows(page)).toBe(false);

		await page.goto(logUrl('symptoms'));
		await hydrated(page);
		await pick(page, 'other');
		await field(page, 'Other symptom').fill(longText);
		await field(page, 'Note').fill('n'.repeat(200));
		expect(await symptomOverflows(page)).toBe(false);
		await pick(page, 'fever');
		await field(page, 'Temperature').fill('101.12');
		await field(page, 'Temperature unit').selectOption('f');
		expect(await symptomOverflows(page)).toBe(false);

		const date = day(String(made.at));
		await page.goto(`/kids/${MIA.id}/timeline?from=${date}&to=${date}`);
		await expect(page.getByRole('link', { name: new RegExp(`Other: ${longText}`) })).toBeVisible();
		expect(await symptomOverflows(page)).toBe(false);
	});
});

test.describe('the sickness tracker without JavaScript', () => {
	test.use({ javaScriptEnabled: false });

	test('create a fever, fail on the method, fix it, edit it into a cough and delete it', async ({
		page
	}) => {
		const api = await stubAs(ANA);
		await signIn(page, ANA);
		await page.goto(logUrl('symptoms'));
		await expect(field(page, 'Temperature')).toBeHidden();
		await pick(page, 'fever');
		await field(page, 'Temperature').fill('38,4');
		await field(page, 'Note').fill('no script');
		await saveAndWait(page);
		await expectFieldError(field(page, 'Method'), 'Choose how the temperature was taken');
		await expect(field(page, 'Symptom')).toHaveValue('fever');
		await expect(field(page, 'Temperature')).toBeVisible();
		await expect(field(page, 'Temperature')).toHaveValue('38,4');
		await expect(field(page, 'Note')).toHaveValue('no script');
		expect(await rows(api, 'symptoms')).toEqual([]);

		await field(page, 'Method').selectOption('ear');
		await saveAndWait(page);
		await onTimeline(page);
		const [made] = await rows(api, 'symptoms');
		expect(made).toMatchObject({ symptom: 'fever', method: 'ear', note: 'no script' });
		expect(made.temperature).toMatchObject({ c_hundredths: 3840, typed_unit: 'c' });
		await expect(page.getByText(/Fever 38\.4 °C \(ear\)/)).toBeVisible();

		await page.goto(entryUrl('symptoms', made.id));
		await expect(field(page, 'Temperature')).toBeVisible();
		await expect(field(page, 'Temperature')).toHaveValue('38.4');
		await field(page, 'Note').fill('edited');
		await saveAndWait(page);
		await onTimeline(page);
		expect(await written('PATCH', `${SYMPTOM_PATH}/${made.id}`)).toEqual({ note: 'edited' });

		await page.goto(entryUrl('symptoms', made.id));
		await pick(page, 'cough');
		await saveAndWait(page);
		await onTimeline(page);
		expect((await rows(api, 'symptoms'))[0]).toMatchObject({
			symptom: 'cough',
			temperature: null,
			method: null
		});

		await page.goto(entryUrl('symptoms', made.id));
		await expect(page.getByRole('button', { name: 'Yes, delete this entry' })).toBeHidden();
		await page.getByText('Delete this entry', { exact: true }).click();
		await submit(page.getByRole('button', { name: 'Yes, delete this entry' }));
		await onTimeline(page);
		expect(await rows(api, 'symptoms')).toEqual([]);
	});

	test('other shows its text field and requires it', async ({ page }) => {
		await stubAs(ANA);
		await signIn(page, ANA);
		await page.goto(logUrl('symptoms'));
		await pick(page, 'other');
		await saveAndWait(page);
		await expectFieldError(field(page, 'Other symptom'), 'Enter what the symptom is');
		await expect(field(page, 'Other symptom')).toBeVisible();
	});
});
