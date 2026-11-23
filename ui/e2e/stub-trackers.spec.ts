import { expect, test } from '@playwright/test';
import { ageMonths, todayIn } from '../src/lib/format.js';
import {
	ANA,
	CLEO,
	IVY,
	LEO,
	MIA,
	STUB_TOKEN,
	STUB_URL,
	openSession,
	resetStub
} from './stub-api.js';

// The stub's tracker routes answer as the Go API does; the UI specs that build on them rely on it.

test.beforeEach(async () => {
	await resetStub();
});

type Reply = { status: number; body: ReturnType<typeof JSON.parse> };

async function as(who: { username: string; password: string }) {
	const token = await openSession(who.username, who.password);
	return async (method: string, path: string, body?: unknown): Promise<Reply> => {
		const res = await fetch(`${STUB_URL}${path}`, {
			method,
			headers: {
				authorization: `Bearer ${STUB_TOKEN}`,
				'x-session-token': token,
				'content-type': 'application/json'
			},
			body: body === undefined ? undefined : JSON.stringify(body)
		});
		const text = await res.text();
		return { status: res.status, body: text ? JSON.parse(text) : null };
	};
}

const invalid = (field: string) => ({ status: 422, body: { error: 'invalid', field } });
// Fixed for the whole file, so two instants minutes apart stay exactly that far apart.
const T0 = Math.floor(Date.now() / 1000) * 1000;
const minutesAgo = (m: number) => new Date(T0 - m * 60_000).toISOString().replace(/\.\d{3}Z$/, 'Z');

test('Ivy is thirteen months old on the stub clock, Mia is young and Leo is in the 24+ band', async () => {
	const ana = await as(ANA);
	const kids = (await ana('GET', '/v1/kids')).body.kids;
	expect(kids.map((k: { id: string }) => k.id)).toEqual([MIA.id, LEO.id, IVY.id]);
	expect(ageMonths(IVY.birthDate, todayIn(IVY.timeZone))).toBe(13);
	expect(ageMonths(MIA.birthDate, todayIn(MIA.timeZone))).toBeLessThan(12);
	expect(ageMonths(LEO.birthDate, todayIn(LEO.timeZone))).toBeGreaterThanOrEqual(24);
	expect((await ana('GET', `/v1/kids/${IVY.id}/home`)).body).toMatchObject({
		age_months: 13,
		band: '12-23'
	});
});

test('bottles: validation names the field, volumes convert, an unchanged volume is kept', async () => {
	const ana = await as(ANA);
	const path = `/v1/kids/${IVY.id}/bottles`;
	const volume = { unit: 'ml', amount: 120 };
	expect(await ana('POST', path, { contents: [], volume })).toMatchObject(invalid('contents'));
	expect(await ana('POST', path, { contents: ['formula', 'formula'], volume })).toMatchObject(
		invalid('contents')
	);
	expect(await ana('POST', path, { contents: ['juice'], volume })).toMatchObject(
		invalid('contents')
	);
	expect(await ana('POST', path, { contents: ['formula'] })).toMatchObject(invalid('volume'));
	expect(
		await ana('POST', path, { contents: ['formula'], volume: { unit: 'ml', amount: 1001 } })
	).toMatchObject(invalid('volume'));
	expect(
		await ana('POST', path, { contents: ['formula'], volume, at: '2001-01-01T00:00:00Z' })
	).toMatchObject(invalid('at'));
	expect(
		await ana('POST', path, { contents: ['formula'], volume, note: 'n'.repeat(501) })
	).toMatchObject(invalid('note'));
	expect(await ana('POST', path, { contents: ['formula'], volume, extra: 1 })).toMatchObject({
		status: 422,
		body: { error: 'unknown_field' }
	});
	expect(await ana('POST', path, { contents: 'formula', volume })).toMatchObject({ status: 400 });

	const made = await ana('POST', path, {
		contents: ['formula', 'breast_milk'],
		volume: { unit: 'floz', amount: 400 }
	});
	expect(made.status).toBe(201);
	expect(made.body).toMatchObject({
		contents: ['breast_milk', 'formula'],
		volume: { ml: 118, floz_hundredths: 400, typed_unit: 'floz' },
		note: null
	});
	const patched = await ana('PATCH', `${path}/${made.body.id}`, {
		volume: { unit: 'floz', amount: 400 },
		note: 'warm'
	});
	expect(patched.body).toMatchObject({ volume: made.body.volume, note: 'warm' });
	expect(await ana('PATCH', `${path}/${made.body.id}`, { contents: [] })).toMatchObject(
		invalid('contents')
	);
	expect((await ana('GET', `${path}/${made.body.id}`)).body.note).toBe('warm');
});

test('advanced bottles: the sources sum into the total, switch modes and validate as the API does', async () => {
	const ana = await as(ANA);
	const path = `/v1/kids/${IVY.id}/bottles`;
	const sources = { unit: 'ml', amounts: { formula: 90, breast_milk: 60 } };

	const simple = await ana('POST', path, {
		contents: ['formula'],
		volume: { unit: 'ml', amount: 90 }
	});
	expect(simple.body.sources).toEqual([]);

	const made = await ana('POST', path, { sources });
	expect(made.status).toBe(201);
	expect(made.body).toMatchObject({
		contents: ['breast_milk', 'formula'],
		volume: { ml: 150, floz_hundredths: 507, typed_unit: 'ml' },
		sources: [
			{ source: 'breast_milk', volume: { ml: 60, floz_hundredths: 203, typed_unit: 'ml' } },
			{ source: 'formula', volume: { ml: 90, floz_hundredths: 304, typed_unit: 'ml' } }
		]
	});
	const one = `${path}/${made.body.id}`;
	expect((await ana('GET', one)).body).toEqual(made.body);

	// A note alone keeps every column; one changed source keeps the other and re-sums the total.
	expect((await ana('PATCH', one, { note: 'x' })).body).toMatchObject({
		volume: made.body.volume,
		sources: made.body.sources,
		note: 'x'
	});
	const edited = await ana('PATCH', one, {
		sources: { unit: 'ml', amounts: { breast_milk: 60, formula: 100 } }
	});
	expect(edited.body.volume).toMatchObject({ ml: 160, typed_unit: 'ml' });
	expect(edited.body.sources[0]).toEqual(made.body.sources[0]);
	expect(edited.body.sources[1].volume).toMatchObject({ ml: 100 });
	// A new set replaces contents; contents beside it must be its keys.
	expect(
		(await ana('PATCH', one, { sources: { unit: 'ml', amounts: { cow_milk: 30 } } })).body
	).toMatchObject({ contents: ['cow_milk'], volume: { ml: 30 } });
	expect(
		await ana('PATCH', one, {
			contents: ['formula'],
			sources: { unit: 'ml', amounts: { other: 5 } }
		})
	).toMatchObject(invalid('contents'));
	// Contents that no longer match the stored sources are refused; a volume clears them.
	expect(await ana('PATCH', one, { contents: ['formula', 'other'] })).toMatchObject(
		invalid('contents')
	);
	const cleared = await ana('PATCH', one, { volume: { unit: 'ml', amount: 100 } });
	expect(cleared.body).toMatchObject({
		contents: ['cow_milk'],
		volume: { ml: 100 },
		sources: []
	});
	expect((await ana('PATCH', one, { contents: ['formula', 'other'] })).body.contents).toEqual([
		'formula',
		'other'
	]);

	// Typed in ounces: the total is the exact sum, each source has its own columns.
	const oz = await ana('POST', path, {
		sources: { unit: 'floz', amounts: { breast_milk: 225, formula: 300 } }
	});
	expect(oz.body.volume).toMatchObject({ floz_hundredths: 525, typed_unit: 'floz' });
	expect(oz.body.sources).toMatchObject([
		{ volume: { ml: 67, floz_hundredths: 225, typed_unit: 'floz' } },
		{ volume: { ml: 89, floz_hundredths: 300, typed_unit: 'floz' } }
	]);

	for (const bad of [
		{ unit: 'ml', amounts: {} },
		{ unit: 'ml', amounts: { juice: 5 } },
		{ unit: 'ml', amounts: { formula: 0 } },
		{ unit: 'ml', amounts: { formula: -3 } },
		{ unit: 'ml', amounts: { formula: null } },
		{ unit: 'cup', amounts: { formula: 5 } },
		{ unit: null, amounts: { formula: 5 } },
		{ amounts: { formula: 5 } },
		{ unit: 'ml' }
	]) {
		expect(await ana('POST', path, { sources: bad })).toMatchObject(invalid('sources'));
	}
	expect(
		await ana('POST', path, { sources: { unit: 'ml', amounts: { formula: 600, other: 401 } } })
	).toMatchObject(invalid('volume'));
	expect(await ana('POST', path, { sources, volume: { unit: 'ml', amount: 150 } })).toMatchObject(
		invalid('sources')
	);
	expect(await ana('POST', path, { contents: ['formula'], sources })).toMatchObject(
		invalid('contents')
	);
	expect(await ana('POST', path, { contents: ['formula', 'breast_milk'], sources })).toMatchObject({
		status: 201
	});
	// Order: at, contents, volume or sources, note.
	expect(
		await ana('POST', path, { at: '2001-01-01T00:00:00Z', sources: { amounts: {} } })
	).toMatchObject(invalid('at'));
	expect(await ana('POST', path, { sources, note: 'n'.repeat(501) })).toMatchObject(
		invalid('note')
	);
	expect(await ana('POST', path, { sources: { unit: 'ml', amounts: {}, extra: 1 } })).toMatchObject(
		{
			status: 422,
			body: { error: 'unknown_field', field: 'extra' }
		}
	);
	expect(await ana('POST', path, { sources: 5 })).toMatchObject({ status: 400 });
	expect(
		await ana('POST', path, { sources: { unit: 'ml', amounts: { formula: '5' } } })
	).toMatchObject({ status: 400 });

	const timeline = await ana('GET', `/v1/kids/${IVY.id}/timeline?types=bottle`);
	expect(timeline.body.items.every((i: { sources: unknown[] }) => Array.isArray(i.sources))).toBe(
		true
	);
});

test('advanced pumping: the sides sum into the total, derive the side, switch modes and validate as the API does', async () => {
	const ana = await as(ANA);
	const path = `/v1/kids/${IVY.id}/pumping`;
	const base = { duration_seconds: 900 };
	const sides = { unit: 'ml', amounts: { right: 50, left: 60 } };

	const simple = await ana('POST', path, {
		...base,
		volume: { unit: 'ml', amount: 90 },
		side: 'left'
	});
	expect(simple.body.sides).toEqual([]);

	const made = await ana('POST', path, { ...base, sides });
	expect(made.status).toBe(201);
	expect(made.body).toMatchObject({
		side: 'both',
		volume: { ml: 110, floz_hundredths: 372, typed_unit: 'ml' },
		sides: [
			{ side: 'left', volume: { ml: 60, floz_hundredths: 203, typed_unit: 'ml' } },
			{ side: 'right', volume: { ml: 50, floz_hundredths: 169, typed_unit: 'ml' } }
		]
	});
	const one = `${path}/${made.body.id}`;
	expect((await ana('GET', one)).body).toEqual(made.body);
	for (const [amounts, side] of [
		[{ left: 60 }, 'left'],
		[{ right: 50 }, 'right']
	] as const) {
		expect(
			(await ana('POST', path, { ...base, sides: { unit: 'ml', amounts } })).body
		).toMatchObject({ side, sides: [{ side }] });
	}

	// A note alone keeps every column; one changed side keeps the other and re-sums the total.
	expect((await ana('PATCH', one, { note: 'x' })).body).toMatchObject({
		volume: made.body.volume,
		sides: made.body.sides,
		side: 'both',
		note: 'x'
	});
	const edited = await ana('PATCH', one, {
		sides: { unit: 'ml', amounts: { left: 60, right: 55 } }
	});
	expect(edited.body.volume).toMatchObject({ ml: 115, typed_unit: 'ml' });
	expect(edited.body.sides[0]).toEqual(made.body.sides[0]);
	expect(edited.body.sides[1].volume).toMatchObject({ ml: 55 });
	// Dropping a side re-derives the side; a side beside the set must be the derived one.
	expect(
		(await ana('PATCH', one, { sides: { unit: 'ml', amounts: { right: 55 } } })).body
	).toMatchObject({ side: 'right', volume: { ml: 55 } });
	expect(
		await ana('PATCH', one, { sides: { unit: 'ml', amounts: { left: 5 } }, side: 'right' })
	).toMatchObject(invalid('side'));
	expect(
		(await ana('PATCH', one, { sides: { unit: 'ml', amounts: { left: 5 } }, side: 'left' })).status
	).toBe(200);
	// A side alone must still fit the sides kept; a volume clears them and keeps the stored side.
	expect(await ana('PATCH', one, { side: 'right' })).toMatchObject(invalid('side'));
	expect((await ana('PATCH', one, { side: 'left' })).status).toBe(200);
	const cleared = await ana('PATCH', one, { volume: { unit: 'ml', amount: 100 } });
	expect(cleared.body).toMatchObject({ side: 'left', volume: { ml: 100 }, sides: [] });
	expect(
		(await ana('PATCH', one, { volume: { unit: 'ml', amount: 100 }, side: 'both' })).body.side
	).toBe('both');

	// Typed in ounces: the total is the exact sum, each side has its own columns.
	const oz = await ana('POST', path, {
		...base,
		sides: { unit: 'floz', amounts: { left: 225, right: 300 } }
	});
	expect(oz.body.volume).toMatchObject({ floz_hundredths: 525, typed_unit: 'floz' });
	expect(oz.body.sides).toMatchObject([
		{ volume: { ml: 67, floz_hundredths: 225, typed_unit: 'floz' } },
		{ volume: { ml: 89, floz_hundredths: 300, typed_unit: 'floz' } }
	]);

	for (const bad of [
		{ unit: 'ml', amounts: {} },
		{ unit: 'ml', amounts: { both: 5 } },
		{ unit: 'ml', amounts: { middle: 5 } },
		{ unit: 'ml', amounts: { left: 0 } },
		{ unit: 'ml', amounts: { left: -3 } },
		{ unit: 'ml', amounts: { left: null } },
		{ unit: 'cup', amounts: { left: 5 } },
		{ unit: null, amounts: { left: 5 } },
		{ amounts: { left: 5 } },
		{ unit: 'ml' }
	]) {
		expect(await ana('POST', path, { ...base, sides: bad })).toMatchObject(invalid('sides'));
	}
	expect(
		await ana('POST', path, { ...base, sides: { unit: 'ml', amounts: { left: 1500, right: 501 } } })
	).toMatchObject(invalid('volume'));
	expect(
		await ana('POST', path, { ...base, sides, volume: { unit: 'ml', amount: 110 } })
	).toMatchObject(invalid('sides'));
	expect(await ana('POST', path, { ...base, sides, side: 'left' })).toMatchObject(invalid('side'));
	expect(await ana('POST', path, { ...base, sides, side: 'both' })).toMatchObject({ status: 201 });
	// Order: started_at, duration_seconds, volume or sides, side, note.
	expect(
		await ana('POST', path, { started_at: '2001-01-01T00:00:00Z', sides: { amounts: {} } })
	).toMatchObject(invalid('started_at'));
	expect(await ana('POST', path, { duration_seconds: 0, sides: { amounts: {} } })).toMatchObject(
		invalid('duration_seconds')
	);
	expect(await ana('POST', path, { ...base, sides, note: 'n'.repeat(501) })).toMatchObject(
		invalid('note')
	);
	expect(
		await ana('POST', path, { ...base, sides: { unit: 'ml', amounts: {}, extra: 1 } })
	).toMatchObject({ status: 422, body: { error: 'unknown_field', field: 'extra' } });
	expect(await ana('POST', path, { ...base, sides: 5 })).toMatchObject({ status: 400 });
	expect(
		await ana('POST', path, { ...base, sides: { unit: 'ml', amounts: { left: '5' } } })
	).toMatchObject({ status: 400 });

	const timeline = await ana('GET', `/v1/kids/${IVY.id}/timeline?types=pumping`);
	expect(timeline.body.items.every((i: { sides: unknown[] }) => Array.isArray(i.sides))).toBe(true);
});

test('nursing weigh-ins: stored as typed, the estimate from grams, and refused as the API refuses', async () => {
	const ana = await as(ANA);
	const base = `/v1/kids/${IVY.id}/nursing`;
	const finished = {
		started_at: minutesAgo(40),
		ended_at: minutesAgo(10),
		left_seconds: 600,
		right_seconds: 300
	};
	const kg = (amount: number) => ({ unit: 'kg', amount });

	const plain = await ana('POST', base, { side: 'left', started_at: minutesAgo(5) });
	expect(plain.body).toMatchObject({ weight_before: null, weight_after: null, milk: null });
	await ana('DELETE', `${base}/${plain.body.id}`);

	const made = await ana('POST', base, {
		...finished,
		weight_before: kg(3400),
		weight_after: kg(3462)
	});
	expect(made.status).toBe(201);
	expect(made.body).toMatchObject({
		weight_before: { grams: 3400, oz_hundredths: 11993, typed_unit: 'kg' },
		weight_after: { grams: 3462, oz_hundredths: 12212, typed_unit: 'kg' },
		milk: { gain: { grams: 62, oz_hundredths: 219 }, volume: { ml: 60, floz_hundredths: 204 } }
	});
	const one = `${base}/${made.body.id}`;

	// A note alone keeps the weights; the same typed value keeps the columns; null clears one.
	expect((await ana('PATCH', one, { note: 'x' })).body).toMatchObject({
		weight_before: made.body.weight_before,
		milk: made.body.milk,
		note: 'x'
	});
	expect((await ana('PATCH', one, { weight_before: kg(3400) })).body.weight_before).toEqual(
		made.body.weight_before
	);
	const cleared = await ana('PATCH', one, { weight_after: null });
	expect(cleared.body).toMatchObject({ weight_after: null, milk: null });
	expect(cleared.body.weight_before).toEqual(made.body.weight_before);

	// A lighter "after" is stored and answers 200 with no estimate; equal weights are a zero estimate.
	expect((await ana('PATCH', one, { weight_after: kg(3300) })).body).toMatchObject({
		weight_after: { grams: 3300 },
		milk: null
	});
	expect((await ana('PATCH', one, { weight_after: kg(3400) })).body.milk).toEqual({
		gain: { grams: 0, oz_hundredths: 0 },
		volume: { ml: 0, floz_hundredths: 0 }
	});
	// Pounds and ounces are typed exactly in hundredths of an ounce; grams derive half-up.
	const lb = await ana('PATCH', one, { weight_before: { unit: 'lb', amount: 12050 } });
	expect(lb.body.weight_before).toEqual({ grams: 3416, oz_hundredths: 12050, typed_unit: 'lb' });
	// 3416 g before against 3400 g after: lighter, so no estimate.
	expect(lb.body.milk).toBeNull();

	// A running session takes weights too, and shows them on the home.
	const running = await ana('POST', base, {
		side: 'right',
		started_at: minutesAgo(5),
		weight_before: kg(3400)
	});
	expect(running.body).toMatchObject({ side: 'right', weight_before: { grams: 3400 }, milk: null });
	const after = await ana('PATCH', `${base}/${running.body.id}`, { weight_after: kg(3450) });
	expect(after.body.milk).toMatchObject({ gain: { grams: 50 }, volume: { ml: 49 } });
	const home = await ana('GET', `/v1/kids/${IVY.id}/home`);
	expect(home.body.running.nursing).toMatchObject({ weight_before: { grams: 3400 } });
	const item = (await ana('GET', `/v1/kids/${IVY.id}/timeline?types=nursing`)).body.items.find(
		(i: { id: string }) => i.id === made.body.id
	);
	expect(item).toMatchObject({ type: 'nursing', weight_before: { grams: 3416 } });

	// The range is 200 g to 250 kg, in grams; every refusal names the weight.
	for (const bad of [
		kg(199),
		kg(250001),
		kg(0),
		kg(-5),
		{ unit: 'st', amount: 5 },
		{ unit: null, amount: 5 },
		{ unit: 'kg', amount: null },
		{ unit: 'lb', amount: 1 },
		{},
		5,
		'5',
		[3400],
		{ unit: 'kg', amount: '3400' },
		{ unit: 'kg', amount: 3400.5 },
		{ unit: 5, amount: 3400 }
	]) {
		expect(await ana('PATCH', one, { weight_after: bad })).toMatchObject(invalid('weight_after'));
		expect(await ana('POST', base, { ...finished, weight_before: bad })).toMatchObject(
			invalid('weight_before')
		);
	}
	expect(await ana('PATCH', one, { weight_after: kg(200) })).toMatchObject({ status: 200 });
	expect(await ana('PATCH', one, { weight_after: kg(250000) })).toMatchObject({ status: 200 });
	// A growth weight with the same mistake is a 400: each route keeps its own convention.
	expect(await ana('POST', `/v1/kids/${IVY.id}/growth`, { weight: 5 })).toMatchObject({
		status: 400
	});
	expect(await ana('PATCH', one, { weight_after: { unit: 'kg', extra: 1 } })).toMatchObject({
		status: 422,
		body: { error: 'unknown_field', field: 'extra' }
	});

	// Order: the seconds and side before the weights, weight_before before weight_after, then the note.
	expect(
		await ana('POST', base, {
			started_at: finished.started_at,
			ended_at: finished.ended_at,
			weight_before: 5
		})
	).toMatchObject(invalid('left_seconds'));
	expect(
		await ana('POST', base, {
			...finished,
			weight_before: 5,
			weight_after: 5,
			note: 'n'.repeat(501)
		})
	).toMatchObject(invalid('weight_before'));
	expect(
		await ana('POST', base, { ...finished, weight_after: kg(3400), note: 'n'.repeat(501) })
	).toMatchObject(invalid('note'));

	// A weigh-in is not a growth row.
	expect((await ana('GET', `/v1/kids/${IVY.id}/growth`)).body.growth).toEqual([]);
});

test('a kid never sees another kid’s entry, and an unknown or malformed id is the same 404', async () => {
	const ana = await as(ANA);
	const cleo = await as(CLEO);
	const made = await ana('POST', `/v1/kids/${IVY.id}/diapers`, { kind: 'wet' });
	const id = made.body.id;
	const notFound = { status: 404, body: { error: 'not_found' } };
	// Ana is Leo's caregiver, so the kid is hers; the entry is not.
	expect(await ana('GET', `/v1/kids/${LEO.id}/diapers/${id}`)).toEqual(notFound);
	expect(await ana('PATCH', `/v1/kids/${LEO.id}/diapers/${id}`, { kind: 'dry' })).toEqual(notFound);
	expect(await ana('DELETE', `/v1/kids/${LEO.id}/diapers/${id}`)).toEqual(notFound);
	expect(await ana('GET', `/v1/kids/${IVY.id}/diapers/nope`)).toEqual(notFound);
	expect(await ana('GET', `/v1/kids/${IVY.id}/diapers/a%20b`)).toEqual(notFound);
	// No grant on Ivy at all: the same body as an unknown kid.
	expect(await cleo('GET', `/v1/kids/${IVY.id}/diapers/${id}`)).toEqual(notFound);
	expect(await cleo('GET', `/v1/kids/${IVY.id}/diapers`)).toEqual(notFound);
	expect(await ana('GET', `/v1/kids/${IVY.id}/diapers/${id}`)).toMatchObject({ status: 200 });
});

test('lists newest first, in pages of the asked size, with an opaque cursor and a half-open range', async () => {
	const ana = await as(ANA);
	const path = `/v1/kids/${IVY.id}/meals`;
	for (const [what, ago] of [
		['a', 50],
		['b', 40],
		['c', 30],
		['d', 20],
		['e', 10]
	] as const) {
		await ana('POST', path, { what, at: minutesAgo(ago) });
	}
	const first = (await ana('GET', `${path}?limit=2`)).body;
	expect(first.meals.map((m: { what: string }) => m.what)).toEqual(['e', 'd']);
	const second = (await ana('GET', `${path}?limit=2&cursor=${first.next_cursor}`)).body;
	expect(second.meals.map((m: { what: string }) => m.what)).toEqual(['c', 'b']);
	const third = (await ana('GET', `${path}?limit=2&cursor=${second.next_cursor}`)).body;
	expect(third).toMatchObject({ meals: [{ what: 'a' }], next_cursor: null });

	const from = encodeURIComponent(minutesAgo(40));
	const to = encodeURIComponent(minutesAgo(20));
	const ranged = (await ana('GET', `${path}?from=${from}&to=${to}`)).body;
	expect(ranged.meals.map((m: { what: string }) => m.what)).toEqual(['c', 'b']);

	for (const [query, field] of [
		['limit=0', 'limit'],
		['limit=101', 'limit'],
		['cursor=!!', 'cursor'],
		['from=yesterday', 'from'],
		[`from=${to}&to=${from}`, 'from'],
		['to=x', 'to']
	]) {
		expect(await ana('GET', `${path}?${query}`)).toMatchObject(invalid(field));
	}
});

test('one timer of each kind at a time; ending twice or a day late is refused', async () => {
	const ana = await as(ANA);
	const base = `/v1/kids/${IVY.id}`;
	const sleep = await ana('POST', `${base}/sleep`, {});
	expect(sleep).toMatchObject({ status: 201, body: { ended_at: null } });
	expect(await ana('POST', `${base}/sleep`, {})).toEqual({
		status: 409,
		body: { error: 'already_running' }
	});
	// A finished session needs no free timer.
	expect(
		await ana('POST', `${base}/sleep`, { started_at: minutesAgo(120), ended_at: minutesAgo(60) })
	).toMatchObject({ status: 201 });
	expect((await ana('GET', `${base}/home`)).body.running.sleep.id).toBe(sleep.body.id);

	const ended = await ana('POST', `${base}/sleep/${sleep.body.id}/end`, {});
	expect(ended.status).toBe(200);
	expect(ended.body.ended_at).not.toBeNull();
	expect(await ana('POST', `${base}/sleep/${sleep.body.id}/end`, {})).toEqual({
		status: 409,
		body: { error: 'not_running' }
	});
	expect((await ana('GET', `${base}/home`)).body.running.sleep).toBeNull();

	// A timer left running for more than a day cannot be ended: 422 at, as the API says.
	const forgotten = await ana('POST', `${base}/sleep`, { started_at: minutesAgo(25 * 60) });
	expect(forgotten.status).toBe(201);
	expect(await ana('POST', `${base}/sleep/${forgotten.body.id}/end`, {})).toMatchObject(
		invalid('at')
	);
	expect(
		await ana('POST', `${base}/sleep/${forgotten.body.id}/end`, { at: minutesAgo(-60) })
	).toMatchObject(invalid('at'));
	expect(await ana('POST', `${base}/sleep/${forgotten.body.id}/end`, { at: 'noon' })).toMatchObject(
		{
			status: 400
		}
	);
	expect(await ana('POST', `${base}/sleep/nope/end`, {})).toMatchObject({ status: 404 });
});

test('nursing: a side is required to start, switching credits the side in use, ending finishes', async () => {
	const ana = await as(ANA);
	const base = `/v1/kids/${IVY.id}/nursing`;
	expect(await ana('POST', base, {})).toMatchObject(invalid('side'));
	expect(await ana('POST', base, { side: 'both' })).toMatchObject(invalid('side'));
	expect(await ana('POST', base, { side: 'left', left_seconds: 5 })).toMatchObject(
		invalid('left_seconds')
	);
	const started = await ana('POST', base, { side: 'left', started_at: minutesAgo(30) });
	expect(started.body).toMatchObject({ side: 'left', left_seconds: 0, right_seconds: 0 });
	expect(await ana('POST', base, { side: 'right' })).toMatchObject({ status: 409 });

	const id = started.body.id;
	expect(await ana('POST', `${base}/${id}/side`, { side: 'middle' })).toMatchObject(
		invalid('side')
	);
	const switched = await ana('POST', `${base}/${id}/side`, { side: 'right', at: minutesAgo(20) });
	expect(switched.body).toMatchObject({ side: 'right', left_seconds: 600, right_seconds: 0 });
	expect(
		await ana('POST', `${base}/${id}/side`, { side: 'left', at: minutesAgo(25) })
	).toMatchObject(invalid('at'));
	const ended = await ana('POST', `${base}/${id}/end`, { at: minutesAgo(10) });
	expect(ended.body).toMatchObject({
		side: null,
		side_since: null,
		left_seconds: 600,
		right_seconds: 600
	});
	expect(await ana('POST', `${base}/${id}/side`, { side: 'left' })).toEqual({
		status: 409,
		body: { error: 'not_running' }
	});
	expect(await ana('PATCH', `${base}/${id}`, { side: 'left' })).toMatchObject(invalid('side'));
	expect(await ana('PATCH', `${base}/${id}`, { left_seconds: 100000 })).toMatchObject(
		invalid('left_seconds')
	);
	expect(await ana('PATCH', `${base}/${id}`, { left_seconds: 300 })).toMatchObject({
		status: 200,
		body: { left_seconds: 300 }
	});
});

test('timeline merges the trackers newest first, filters by type, and pages', async () => {
	const ana = await as(ANA);
	const base = `/v1/kids/${IVY.id}`;
	await ana('POST', `${base}/diapers`, { kind: 'dirty', at: minutesAgo(50) });
	await ana('POST', `${base}/meals`, { what: 'pear', at: minutesAgo(40) });
	await ana('POST', `${base}/pumping`, {
		started_at: minutesAgo(30),
		duration_seconds: 600,
		volume: { unit: 'ml', amount: 80 },
		side: 'both'
	});
	await ana('POST', `${base}/bottles`, {
		at: minutesAgo(20),
		contents: ['formula'],
		volume: { unit: 'ml', amount: 100 }
	});
	await ana('POST', `${base}/sleep`, { started_at: minutesAgo(10) });
	const all = (await ana('GET', `${base}/timeline`)).body;
	expect(all.items.map((i: { type: string }) => i.type)).toEqual([
		'sleep',
		'bottle',
		'pumping',
		'meal',
		'diaper'
	]);
	expect(all.items[0]).toMatchObject({
		type: 'sleep',
		ended_at: null,
		at: all.items[0].started_at
	});
	expect(all.items[4]).toMatchObject({ type: 'diaper', kind: 'dirty' });
	const some = (await ana('GET', `${base}/timeline?types=meal,diaper`)).body;
	expect(some.items.map((i: { type: string }) => i.type)).toEqual(['meal', 'diaper']);
	const paged = (await ana('GET', `${base}/timeline?limit=2`)).body;
	expect(paged.items).toHaveLength(2);
	const next = (await ana('GET', `${base}/timeline?limit=2&cursor=${paged.next_cursor}`)).body;
	expect(next.items.map((i: { type: string }) => i.type)).toEqual(['pumping', 'meal']);
	for (const types of ['bottles', 'meal,meal', 'meal,', '', 'Meal']) {
		expect(await ana('GET', `${base}/timeline?types=${types}`)).toMatchObject(invalid('types'));
	}
	expect(await ana('GET', `${base}/timeline?types=meal&types=diaper`)).toMatchObject(
		invalid('types')
	);
});

test('summary counts one kid-local day at a time and refuses a bad range', async () => {
	test.skip(T0 % 86_400_000 < 10 * 60_000, 'the entries would fall on yesterday');
	const ana = await as(ANA);
	const base = `/v1/kids/${IVY.id}`;
	const today = todayIn(IVY.timeZone);
	await ana('POST', `${base}/bottles`, {
		contents: ['formula'],
		volume: { unit: 'ml', amount: 120 }
	});
	await ana('POST', `${base}/diapers`, { kind: 'both' });
	await ana('POST', `${base}/diapers`, { kind: 'wet' });
	await ana('POST', `${base}/meals`, { what: 'pear' });
	await ana('POST', `${base}/nursing`, {
		started_at: minutesAgo(5),
		ended_at: minutesAgo(1),
		left_seconds: 120,
		right_seconds: 60
	});
	const days = (await ana('GET', `${base}/summary?from=${today}&to=${today}`)).body.days;
	expect(days).toEqual([
		{
			date: today,
			bottles: { count: 1, ml: 120, floz_hundredths: 406 },
			nursing: { count: 1, seconds: 180 },
			pumping: { count: 0, ml: 0, floz_hundredths: 0 },
			diapers: { total: 2, wet: 1, dirty: 0, both: 1, dry: 0 },
			sleep: { seconds: 0 },
			meals: { count: 1 }
		}
	]);
	const week = (await ana('GET', `${base}/summary?from=2026-01-01&to=2026-01-07`)).body.days;
	expect(week).toHaveLength(7);
	expect(week[0]).toMatchObject({ date: '2026-01-01', bottles: { count: 0 } });
	for (const [query, field] of [
		['to=2026-01-01', 'from'],
		['from=2026-01-01', 'to'],
		['from=2026-1-1&to=2026-01-02', 'from'],
		['from=2026-01-02&to=2026-01-01', 'from'],
		['from=2026-01-01&to=2026-04-04', 'to']
	]) {
		expect(await ana('GET', `${base}/summary?${query}`)).toMatchObject(invalid(field));
	}
	expect(
		(await ana('GET', `${base}/summary?from=2026-01-01&to=2026-04-03`)).body.days
	).toHaveLength(93);
});

test('modules follow the age band; only a guardian switches one; a recent entry re-enables once', async () => {
	const ana = await as(ANA);
	const ivy = (await ana('GET', `/v1/kids/${IVY.id}/modules`)).body.modules;
	expect(ivy).toHaveLength(11);
	expect(ivy.every((m: { enabled: boolean; default: boolean }) => m.enabled && m.default)).toBe(
		true
	);

	const leo = `/v1/kids/${LEO.id}`;
	const states = (await ana('GET', `${leo}/modules`)).body.modules;
	const on = states
		.filter((m: { enabled: boolean }) => m.enabled)
		.map((m: { module: string }) => m.module);
	expect(on).toEqual(['growth', 'vaccination', 'sickness', 'medication']);
	// Ana is only Leo's caregiver.
	expect(await ana('PUT', `${leo}/modules/diapers`, { enabled: true })).toEqual({
		status: 403,
		body: { error: 'forbidden' }
	});
	await ana('POST', `${leo}/diapers`, { kind: 'wet' });
	const after = (await ana('GET', `${leo}/modules`)).body.modules.find(
		(m: { module: string }) => m.module === 'diapers'
	);
	expect(after).toEqual({ module: 'diapers', enabled: true, default: false, overridden: true });

	const cleo = await as(CLEO);
	expect(await cleo('PUT', `${leo}/modules/diapers`, { enabled: false })).toMatchObject({
		status: 204
	});
	await ana('POST', `${leo}/diapers`, { kind: 'dry' });
	expect(
		(await ana('GET', `${leo}/modules`)).body.modules.find(
			(m: { module: string }) => m.module === 'diapers'
		)
	).toMatchObject({ enabled: false, overridden: true });
	// An entry of a switched-off tracker is kept and listed.
	expect((await ana('GET', `${leo}/diapers`)).body.diapers).toHaveLength(2);
	expect((await ana('GET', `${leo}/timeline`)).body.items).toEqual([]);
	expect((await ana('GET', `${leo}/timeline?types=diaper`)).body.items).toHaveLength(2);

	expect(await cleo('PUT', `${leo}/modules/teething`, { enabled: true })).toMatchObject({
		status: 404
	});
	expect(await cleo('PUT', `${leo}/modules/growth`, {})).toMatchObject(invalid('enabled'));
	expect(await cleo('PUT', `${leo}/modules/growth`, { enabled: 'no' })).toMatchObject({
		status: 400
	});
});

test('home reads the age, the modules, the running timers and today', async () => {
	const ana = await as(ANA);
	const base = `/v1/kids/${MIA.id}`;
	const idle = (await ana('GET', `${base}/home`)).body;
	expect(idle).toMatchObject({
		age_months: ageMonths(MIA.birthDate, todayIn(MIA.timeZone)),
		band: '0-11',
		running: { sleep: null, nursing: null },
		today: { date: todayIn(MIA.timeZone), bottles: { count: 0 } }
	});
	expect(idle.modules).toHaveLength(11);
	const nursing = await ana('POST', `${base}/nursing`, { side: 'right' });
	expect((await ana('GET', `${base}/home`)).body.running.nursing).toEqual(nursing.body);
});

test('a deleted kid takes its entries and module choices with it', async () => {
	const ana = await as(ANA);
	await ana('POST', `/v1/kids/${IVY.id}/diapers`, { kind: 'wet' });
	await ana('PUT', `/v1/kids/${IVY.id}/modules/sleep`, { enabled: false });
	expect(await ana('DELETE', `/v1/kids/${IVY.id}`, { confirm_name: IVY.name })).toMatchObject({
		status: 204
	});
	expect(await ana('GET', `/v1/kids/${IVY.id}/diapers`)).toMatchObject({ status: 404 });
	expect(await ana('GET', `/v1/kids/${IVY.id}/home`)).toMatchObject({ status: 404 });
});
