import { expect, test } from '@playwright/test';
import {
	ADMIN,
	ANA,
	CLEO,
	DAN,
	IVY,
	NELL,
	OTTO,
	PIA,
	REMY,
	SAMI,
	moveStubClock,
	resetStub,
	stubAs
} from './stub-api.js';

// The stub's milestone routes answer as httpapi/milestones.go does. Its catalog is a stub: tests read its shape
// (twelve ages, four domains), never its wording or counts as reference data.

test.beforeEach(async () => {
	await resetStub();
});

const invalid = (field: string) => ({ status: 422, body: { error: 'invalid', field } });
const notFound = { status: 404, body: { error: 'not_found' } };
const DAY = 24 * 3600;
const AGES = [2, 4, 6, 9, 12, 15, 18, 24, 30, 36, 48, 60];
const DOMAINS = ['social_emotional', 'language_communication', 'cognitive', 'movement_physical'];
const utcToday = () => new Date().toISOString().slice(0, 10);
const shiftDays = (date: string, days: number) =>
	new Date(Date.parse(`${date}T00:00:00Z`) + days * DAY * 1000).toISOString().slice(0, 10);
const path = (kid: string) => `/v1/kids/${kid}/milestones`;
const ITEM = '9m-se-stub-1';

type View = {
	catalog: Record<string, string>;
	age: { chronological_months: number; corrected_months: number | null; basis_months: number };
	catalog_shown: boolean;
	checklists: {
		age_months: number;
		position: string;
		items: { id: string; domain: string; text: string; reached: unknown }[];
	}[];
	custom: Record<string, unknown>[];
	retired: Record<string, unknown>[];
	reached: Record<string, unknown>[];
};
const positions = (v: View) =>
	Object.fromEntries(v.checklists.map((c) => [c.age_months, c.position]));
const view = async (kid: string): Promise<View> => {
	const r = await (await stubAs(CLEO))('GET', path(kid));
	expect(r.status).toBe(200);
	return r.body;
};

test('the view: the catalog provenance without omitted, every age and domain, positions for a 9-month kid', async () => {
	const v = await view(NELL.id);
	expect(Object.keys(v).sort()).toEqual(
		['age', 'catalog', 'catalog_shown', 'checklists', 'custom', 'reached', 'retired'].sort()
	);
	expect(Object.keys(v.catalog).sort()).toEqual(
		[
			'attribution',
			'authority',
			'edition',
			'id',
			'licence',
			'name',
			'origin_url',
			'retrieved',
			'screening_note'
		].sort()
	);
	expect(v.age).toEqual({ chronological_months: 9, corrected_months: null, basis_months: 9 });
	expect(v.catalog_shown).toBe(true);
	expect(v.checklists.map((c) => c.age_months)).toEqual(AGES);
	for (const c of v.checklists) {
		expect(c.items.map((i) => i.domain)).toEqual(DOMAINS);
		expect(c.items.every((i) => i.reached === null && i.id.startsWith(`${c.age_months}m-`))).toBe(
			true
		);
	}
	expect(positions(v)).toEqual({
		2: 'earlier',
		4: 'earlier',
		6: 'earlier',
		9: 'current',
		12: 'next',
		15: 'later',
		18: 'later',
		24: 'later',
		30: 'later',
		36: 'later',
		48: 'later',
		60: 'later'
	});
	expect([v.custom, v.retired, v.reached]).toEqual([[], [], []]);
});

test('4 years 11 months: all twelve lists, 4 years current and 5 years next; a retired row keeps its text', async () => {
	const v = await view(OTTO.id);
	expect(v.age.basis_months).toBe(59);
	expect(v.catalog_shown).toBe(true);
	expect(v.checklists).toHaveLength(12);
	expect(positions(v)).toMatchObject({ 36: 'earlier', 48: 'current', 60: 'next' });
	expect(v.retired).toEqual([
		{
			id: 'ms-retired',
			item: '4m-cg-stub-retired',
			text: 'Stub item 4m-cg-retired',
			reached_on: expect.any(String),
			note: null
		}
	]);
	expect(v.checklists[1].items.map((i) => i.id)).not.toContain('4m-cg-stub-retired');
});

test('the 5th birthday hides the lists and lists reached rows; the day before shows them', async () => {
	const cleo = await stubAs(CLEO);
	const kid = (await cleo('GET', `/v1/kids/${PIA.id}`)).body;
	// Years-based: 29 February takes the 28th, five years earlier.
	const [y, md] = [Number(utcToday().slice(0, 4)), utcToday().slice(4)];
	expect(kid.birth_date).toBe(`${y - 5}${md === '-02-29' ? '-02-28' : md}`);
	const reached = await cleo('POST', path(PIA.id), {
		item: '60m-lc-stub-1',
		reached_on: utcToday(),
		note: 'x'
	});
	expect(reached.status).toBe(201);
	const custom = await cleo('POST', path(PIA.id), { text: 'Rides a bike', reached_on: utcToday() });
	const on = await view(PIA.id);
	expect(on.catalog_shown).toBe(false);
	expect(on.checklists).toEqual([]);
	expect(on.age).toEqual({ chronological_months: 60, corrected_months: null, basis_months: 60 });
	expect(on.reached).toEqual([
		{
			id: reached.body.id,
			item: '60m-lc-stub-1',
			age_months: 60,
			domain: 'language_communication',
			text: 'Stub item 60m-lc-1',
			reached_on: utcToday(),
			note: 'x'
		}
	]);
	expect(on.custom).toEqual([
		{ id: custom.body.id, text: 'Rides a bike', reached_on: utcToday(), note: null }
	]);
	// A catalog item is still accepted from the 5th birthday on.
	expect(
		(await cleo('POST', path(PIA.id), { item: '2m-se-stub-1', reached_on: utcToday() })).status
	).toBe(201);

	// On 29 February Pia's 5th birthday was the 28th, so the day before it is two days back.
	await moveStubClock(md === '-02-29' ? -2 * DAY : -DAY);
	const before = await view(PIA.id);
	expect(before.catalog_shown).toBe(true);
	expect(before.age.basis_months).toBe(59);
	expect(before.reached).toEqual([]);
	expect(positions(before)).toMatchObject({ 48: 'current', 60: 'next' });
});

test('a 32-week kid at 3 months: basis 1 by corrected age, no current list and 2 months next', async () => {
	const v = await view(REMY.id);
	expect(v.age).toEqual({ chronological_months: 3, corrected_months: 1, basis_months: 1 });
	expect(positions(v)).toEqual({
		2: 'next',
		...Object.fromEntries(AGES.slice(1).map((a) => [a, 'later']))
	});
	await moveStubClock(62 * DAY);
	const later = await view(REMY.id);
	expect(later.age.chronological_months).toBe(5);
	expect(positions(later)).toMatchObject({ 2: 'current', 4: 'next' });
});

test('a 30-month kid: milestones off until the first mark, which the band lookback keeps on', async () => {
	const cleo = await stubAs(CLEO);
	const milestones = async () =>
		(
			(await cleo('GET', `/v1/kids/${SAMI.id}/modules`)).body.modules as Record<string, unknown>[]
		).find((m) => m.module === 'milestones');
	expect(await milestones()).toEqual({
		module: 'milestones',
		enabled: false,
		default: false,
		overridden: false
	});
	const v = await view(SAMI.id);
	expect(v.age.basis_months).toBe(30);
	expect(positions(v)).toMatchObject({ 30: 'current', 36: 'next' });
	expect(
		(await cleo('POST', path(SAMI.id), { item: '30m-mp-stub-1', reached_on: utcToday() })).status
	).toBe(201);
	expect(await milestones()).toEqual({
		module: 'milestones',
		enabled: true,
		default: false,
		overridden: true
	});
	const home = (await cleo('GET', `/v1/kids/${SAMI.id}/home`)).body.modules;
	expect(home.find((m: { module: string }) => m.module === 'milestones').enabled).toBe(true);
	// The others past 24 months are kept on by a guardian; growth is on by default.
	for (const kid of [OTTO, PIA]) {
		const states = (await cleo('GET', `/v1/kids/${kid.id}/modules`)).body.modules;
		expect(states.find((m: { module: string }) => m.module === 'milestones')).toMatchObject({
			enabled: true,
			overridden: true
		});
	}
});

test('create: the decoder first, then item or text, reached_on and note, in that order', async () => {
	const cleo = await stubAs(CLEO);
	const post = (body: unknown) => cleo('POST', path(NELL.id), body);
	const today = utcToday();
	expect(await post({ nope: 1 })).toMatchObject({
		status: 422,
		body: { error: 'unknown_field', field: 'nope' }
	});
	expect(await post({ item: 5 })).toMatchObject({ status: 400, body: { error: 'bad_request' } });
	expect(await post([])).toMatchObject({ status: 400 });
	expect(await post({})).toMatchObject(invalid('item'));
	expect(await post({ item: ITEM, text: 'x', reached_on: today })).toMatchObject(invalid('item'));
	expect(await post({ item: null, reached_on: today })).toMatchObject(invalid('item'));
	expect(await post({ item: null, text: null })).toMatchObject(invalid('item'));
	expect(await post({ item: '9m-se-unknown', reached_on: today })).toMatchObject(invalid('item'));
	expect(await post({ item: '4m-cg-stub-retired', reached_on: today })).toMatchObject(
		invalid('item')
	);
	expect(await post({ text: null, reached_on: today })).toMatchObject(invalid('text'));
	expect(await post({ text: '   ', reached_on: today })).toMatchObject(invalid('text'));
	expect(await post({ text: 'é'.repeat(201), reached_on: today })).toMatchObject(invalid('text'));
	expect(await post({ item: 'bad', reached_on: 'bad', note: ' ' })).toMatchObject(invalid('item'));
	expect(await post({ item: ITEM })).toMatchObject(invalid('reached_on'));
	expect(await post({ item: ITEM, reached_on: null })).toMatchObject(invalid('reached_on'));
	expect(await post({ item: ITEM, reached_on: '2026-02-30' })).toMatchObject(invalid('reached_on'));
	expect(await post({ item: ITEM, reached_on: shiftDays(today, 1) })).toMatchObject(
		invalid('reached_on')
	);
	const birth = (await cleo('GET', `/v1/kids/${NELL.id}`)).body.birth_date;
	expect(await post({ item: ITEM, reached_on: shiftDays(birth, -1) })).toMatchObject(
		invalid('reached_on')
	);
	expect(await post({ item: ITEM, reached_on: today, note: '  ' })).toMatchObject(invalid('note'));
	expect(await post({ item: ITEM, reached_on: today, note: 'n'.repeat(501) })).toMatchObject(
		invalid('note')
	);
	expect(await post({ item: ITEM, reached_on: birth, note: null })).toMatchObject({
		status: 201,
		body: { item: ITEM, text: 'Stub item 9m-se-1', reached_on: birth, note: null }
	});
	const custom = await post({ text: `  ${'é'.repeat(200)}  `, reached_on: today, note: 'n' });
	expect(custom).toMatchObject({
		status: 201,
		body: { item: null, text: 'é'.repeat(200), reached_on: today, note: 'n' }
	});
	expect(Object.keys(custom.body).sort()).toEqual(['id', 'item', 'note', 'reached_on', 'text']);
});

test('a second mark of one item is already_reached; the cap is limit and comes first', async () => {
	const cleo = await stubAs(CLEO);
	const post = (body: unknown) => cleo('POST', path(NELL.id), body);
	const today = utcToday();
	expect((await post({ item: ITEM, reached_on: today })).status).toBe(201);
	expect(await post({ item: ITEM, reached_on: today })).toEqual({
		status: 409,
		body: { error: 'already_reached' }
	});
	for (let i = 1; i < 1000; i += 50) {
		await Promise.all(
			Array.from({ length: Math.min(50, 1000 - i) }, (_, j) =>
				post({ text: `custom ${i + j}`, reached_on: today })
			)
		);
	}
	expect((await view(NELL.id)).custom).toHaveLength(999);
	expect(await post({ text: 'one more', reached_on: today })).toEqual({
		status: 409,
		body: { error: 'limit' }
	});
	expect(await post({ item: ITEM, reached_on: today })).toEqual({
		status: 409,
		body: { error: 'limit' }
	});
});

test('rows are listed newest first: reached_on, then id, descending', async () => {
	const cleo = await stubAs(CLEO);
	const today = utcToday();
	const ids: string[] = [];
	for (const reached_on of [shiftDays(today, -5), today, shiftDays(today, -5)]) {
		ids.push((await cleo('POST', path(NELL.id), { text: 'x', reached_on })).body.id);
	}
	expect((await view(NELL.id)).custom.map((c) => c.id)).toEqual([ids[1], ids[2], ids[0]]);
});

test('get, update and delete an entry; a reached item shows in its checklist until deleted', async () => {
	const cleo = await stubAs(CLEO);
	const today = utcToday();
	const item = (await cleo('POST', path(NELL.id), { item: ITEM, reached_on: today })).body;
	const custom = (await cleo('POST', path(NELL.id), { text: 'Waves', reached_on: today })).body;
	const entry = `${path(NELL.id)}/${item.id}`;
	const nine = async () => (await view(NELL.id)).checklists[3].items[0];
	expect(await nine()).toMatchObject({
		id: ITEM,
		reached: { id: item.id, reached_on: today, note: null }
	});
	expect(await cleo('GET', entry)).toEqual({ status: 200, body: item });

	const patch = (id: string, body: unknown) => cleo('PATCH', `${path(NELL.id)}/${id}`, body);
	expect(await patch(item.id, { item: ITEM })).toMatchObject(invalid('item'));
	expect(await patch(item.id, { item: null })).toMatchObject(invalid('item'));
	expect(await patch(item.id, { text: 'x' })).toMatchObject(invalid('item'));
	expect(await patch(custom.id, { text: null })).toMatchObject(invalid('text'));
	expect(await patch(custom.id, { text: ' ' })).toMatchObject(invalid('text'));
	expect(await patch(item.id, { reached_on: null })).toMatchObject(invalid('reached_on'));
	expect(await patch(item.id, { reached_on: shiftDays(today, 1) })).toMatchObject(
		invalid('reached_on')
	);
	expect(await patch(item.id, { note: '' })).toMatchObject(invalid('note'));
	expect(await patch(item.id, { bogus: 1 })).toMatchObject({
		status: 422,
		body: { error: 'unknown_field', field: 'bogus' }
	});
	// A refused patch changes nothing.
	expect((await cleo('GET', entry)).body).toEqual(item);
	expect(await patch(item.id, { reached_on: shiftDays(today, -2), note: 'park' })).toEqual({
		status: 200,
		body: { ...item, reached_on: shiftDays(today, -2), note: 'park' }
	});
	expect(await patch(item.id, { note: null })).toMatchObject({ status: 200, body: { note: null } });
	expect(await patch(custom.id, { text: ' Waves bye ' })).toMatchObject({
		status: 200,
		body: { text: 'Waves bye', item: null }
	});

	expect(await cleo('DELETE', entry)).toEqual({ status: 204, body: null });
	expect(await cleo('GET', entry)).toEqual(notFound);
	expect(await cleo('DELETE', entry)).toEqual(notFound);
	expect((await nine()).reached).toBeNull();
	expect((await cleo('POST', path(NELL.id), { item: ITEM, reached_on: today })).status).toBe(201);
});

test('404 for no grant, an unknown kid, another kid’s entry and a malformed id; the body first on PATCH', async () => {
	const cleo = await stubAs(CLEO);
	const dan = await stubAs(DAN);
	const ana = await stubAs(ANA);
	const today = utcToday();
	const row = (await cleo('POST', path(NELL.id), { text: 'x', reached_on: today })).body;
	const ivyRow = (await ana('POST', path(IVY.id), { text: 'x', reached_on: today })).body;
	for (const [method, suffix, body] of [
		['GET', '', undefined],
		['POST', '', { text: 'x', reached_on: today }],
		['GET', `/${row.id}`, undefined],
		['PATCH', `/${row.id}`, { note: 'x' }],
		['DELETE', `/${row.id}`, undefined]
	] as const) {
		expect(await dan(method, `${path(NELL.id)}${suffix}`, body)).toEqual(notFound);
		expect(await ana(method, `${path(NELL.id)}${suffix}`, body)).toEqual(notFound);
		expect(await cleo(method, `${path('k-unknown')}${suffix}`, body)).toEqual(notFound);
	}
	for (const id of [ivyRow.id, 'ms-unknown', 'bad.id']) {
		for (const [method, body] of [
			['GET', undefined],
			['PATCH', { note: 'x' }],
			['DELETE', undefined]
		] as const) {
			expect(await cleo(method, `${path(NELL.id)}/${id}`, body)).toEqual(notFound);
		}
	}
	expect(await cleo('PATCH', `${path(NELL.id)}/ms-unknown`, { bogus: 1 })).toMatchObject({
		status: 422,
		body: { error: 'unknown_field' }
	});
	const admin = await stubAs(ADMIN);
	expect((await admin('GET', path(NELL.id))).status).toBe(403);
});

test('a caregiver reaches every milestone route', async () => {
	const cleo = await stubAs(CLEO);
	expect(
		(await cleo('PUT', `/v1/kids/${SAMI.id}/access/${ANA.username}`, { role: 'caregiver' })).status
	).toBe(204);
	const ana = await stubAs(ANA);
	const created = await ana('POST', path(SAMI.id), { text: 'x', reached_on: utcToday() });
	expect(created.status).toBe(201);
	const entry = `${path(SAMI.id)}/${created.body.id}`;
	expect((await ana('GET', path(SAMI.id))).status).toBe(200);
	expect((await ana('GET', entry)).status).toBe(200);
	expect((await ana('PATCH', entry, { note: 'y' })).status).toBe(200);
	expect((await ana('DELETE', entry)).status).toBe(204);
});

test('/__reset drops the marks, restores the clock and rebuilds the kids from it', async () => {
	const cleo = await stubAs(CLEO);
	expect((await cleo('POST', path(NELL.id), { item: ITEM, reached_on: utcToday() })).status).toBe(
		201
	);
	await moveStubClock(-DAY);
	await resetStub();
	const v = await view(NELL.id);
	expect(v.checklists[3].items[0].reached).toBeNull();
	expect(v.age.basis_months).toBe(9);
	expect((await view(PIA.id)).catalog_shown).toBe(false);
	expect((await view(OTTO.id)).retired).toHaveLength(1);
});
