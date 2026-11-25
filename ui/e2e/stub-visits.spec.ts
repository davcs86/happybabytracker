import { expect, test } from '@playwright/test';
import {
	ANA,
	CLEO,
	DAN,
	IVY,
	LEO,
	MIA,
	STUB_TOKEN,
	STUB_URL,
	openSession,
	resetStub,
	stubAs
} from './stub-api.js';

// The stub's visit routes answer as httpapi/visits.go does: the same views, the same refusals in the same order.

test.beforeEach(async () => {
	await resetStub();
});

const invalid = (field: string) => ({ status: 422, body: { error: 'invalid', field } });
const notFound = { status: 404, body: { error: 'not_found' } };
const iso = (unixS: number) => new Date(unixS * 1000).toISOString().replace(/\.\d{3}Z$/, 'Z');
const realNow = () => Math.floor(Date.now() / 1000);
const H = 3600;
const DAY = 24 * H;
const base = (kid: string = IVY.id) => `/v1/kids/${kid}/visits`;
const VIEW_KEYS = [
	'cancelled_at',
	'duration_min',
	'id',
	'kind',
	'linked',
	'location',
	'note',
	'outcome_notes',
	'provider',
	'revision',
	'starts_at',
	'status'
];

async function calendar(who: { username: string; password: string }, path: string) {
	const token = await openSession(who.username, who.password);
	const res = await fetch(`${STUB_URL}${path}`, {
		headers: { authorization: `Bearer ${STUB_TOKEN}`, 'x-session-token': token }
	});
	return { status: res.status, headers: res.headers, text: await res.text() };
}

test('create: the view, the defaults, the validation order and the POST-only keys', async () => {
	const ana = await stubAs(ANA);
	const post = (body: unknown) => ana('POST', base(), body);
	const start = iso(realNow() + 2 * DAY);
	expect(await post({})).toMatchObject(invalid('starts_at'));
	expect(await post({ starts_at: iso(realNow() + 732 * DAY), kind: 'x' })).toMatchObject(
		invalid('starts_at')
	);
	expect(await post({ starts_at: '2001-01-01T00:00:00Z' })).toMatchObject(invalid('starts_at'));
	expect(await post({ starts_at: start, duration_min: 4, kind: 'x' })).toMatchObject(
		invalid('duration_min')
	);
	expect(await post({ starts_at: start, duration_min: 721 })).toMatchObject(
		invalid('duration_min')
	);
	expect(await post({ starts_at: start })).toMatchObject(invalid('kind'));
	expect(await post({ starts_at: start, kind: 'checkup' })).toMatchObject(invalid('kind'));
	const ok = { starts_at: start, kind: 'dental' };
	expect(await post({ ...ok, provider: '   ', location: 'x'.repeat(201) })).toMatchObject(
		invalid('provider')
	);
	expect(await post({ ...ok, provider: 'é'.repeat(121) })).toMatchObject(invalid('provider'));
	expect(await post({ ...ok, location: 'x'.repeat(201) })).toMatchObject(invalid('location'));
	expect(await post({ ...ok, note: 'x'.repeat(501) })).toMatchObject(invalid('note'));
	for (const key of ['outcome_notes', 'cancelled', 'revision']) {
		expect(await post({ ...ok, [key]: null })).toEqual({
			status: 422,
			body: { error: 'unknown_field', field: key }
		});
	}
	expect((await post({ ...ok, duration_min: 'long' })).status).toBe(400);
	expect((await post({ ...ok, starts_at: 'tomorrow' })).status).toBe(400);

	const made = await post({ ...ok, provider: 'Dr. Lee', note: 'Check-up' });
	expect(made.status).toBe(201);
	expect(Object.keys(made.body).sort()).toEqual(VIEW_KEYS);
	expect(made.body).toEqual({
		id: made.body.id,
		starts_at: start,
		duration_min: 30,
		kind: 'dental',
		status: 'upcoming',
		cancelled_at: null,
		provider: 'Dr. Lee',
		location: null,
		note: 'Check-up',
		outcome_notes: null,
		revision: 0,
		linked: { growth: [], vaccinations: [], medications: [] }
	});
	expect(await ana('GET', `${base()}/${made.body.id}`)).toEqual({ status: 200, body: made.body });
	const past = await post({ starts_at: iso(realNow() - DAY), kind: 'sick', duration_min: 5 });
	expect(past.body).toMatchObject({ status: 'past', duration_min: 5 });
});

test('404: an unknown kid, no grant, another kid’s visit and a malformed id answer the same body', async () => {
	const ana = await stubAs(ANA);
	const dan = await stubAs(DAN);
	const visit = (await ana('POST', base(), { starts_at: iso(realNow() + DAY), kind: 'other' }))
		.body;
	const routes: [string, string, unknown?][] = [
		['GET', ''],
		['PATCH', '', { note: 'x' }],
		['DELETE', ''],
		['POST', '/outcome', { revision: 0, outcome_notes: 'x' }],
		['GET', '/calendar']
	];
	for (const [method, suffix, body] of routes) {
		expect(await dan('GET', base())).toEqual(notFound);
		expect(await dan(method, `${base()}/${visit.id}${suffix}`, body)).toEqual(notFound);
		expect(await ana(method, `${base('k-none')}/${visit.id}${suffix}`, body)).toEqual(notFound);
		expect(await ana(method, `${base(MIA.id)}/${visit.id}${suffix}`, body)).toEqual(notFound);
		expect(await ana(method, `${base()}/v.1${suffix}`, body)).toEqual(notFound);
		expect(await ana(method, `${base()}/vis-none${suffix}`, body)).toEqual(notFound);
	}
	expect((await ana('GET', `${base()}/${visit.id}`)).status).toBe(200);
});

test('list: newest first by default, soonest first with order=asc, paged; to reaches the horizon', async () => {
	const ana = await stubAs(ANA);
	const t = realNow();
	const at = [-3 * DAY, -DAY, 2 * DAY, 400 * DAY].map((d) => iso(t + d));
	const ids: string[] = [];
	for (const starts_at of at) {
		ids.push((await ana('POST', base(), { starts_at, kind: 'well_child' })).body.id);
	}
	const all = await ana('GET', base());
	expect(Object.keys(all.body)).toEqual(['visits', 'next_cursor']);
	expect(all.body.visits.map((v: { id: string }) => v.id)).toEqual([...ids].reverse());
	expect(all.body.visits[0].linked).toEqual({ growth: 0, vaccinations: 0, medications: 0 });
	expect(Object.keys(all.body.visits[0]).sort()).toEqual(VIEW_KEYS);

	const upcoming = await ana('GET', `${base()}?from=${encodeURIComponent(iso(t))}&order=asc`);
	expect(upcoming.body.visits.map((v: { id: string }) => v.id)).toEqual([ids[2], ids[3]]);
	const past = await ana('GET', `${base()}?to=${encodeURIComponent(iso(t))}`);
	expect(past.body.visits.map((v: { id: string }) => v.id)).toEqual([ids[1], ids[0]]);

	const first = await ana('GET', `${base()}?order=asc&limit=3`);
	expect(first.body.visits.map((v: { id: string }) => v.id)).toEqual(ids.slice(0, 3));
	expect(first.body.next_cursor).not.toBeNull();
	const second = await ana('GET', `${base()}?order=asc&limit=3&cursor=${first.body.next_cursor}`);
	expect(second.body).toEqual({
		visits: [expect.objectContaining({ id: ids[3] })],
		next_cursor: null
	});
	const newest = await ana('GET', `${base()}?limit=3`);
	const older = await ana('GET', `${base()}?limit=3&cursor=${newest.body.next_cursor}`);
	expect(older.body.visits.map((v: { id: string }) => v.id)).toEqual([ids[0]]);

	expect(await ana('GET', `${base()}?order=up`)).toEqual(invalid('order'));
	expect(await ana('GET', `${base()}?to=${encodeURIComponent(iso(t + 732 * DAY))}`)).toEqual(
		invalid('to')
	);
	expect((await ana('GET', `${base()}?from=${encodeURIComponent(iso(t + DAY))}`)).status).toBe(200);
	expect(
		await ana(
			'GET',
			`${base()}?from=${encodeURIComponent(iso(t + 2 * DAY))}&to=${encodeURIComponent(iso(t + DAY))}`
		)
	).toEqual(invalid('from'));
	expect(await ana('GET', `${base()}?limit=0`)).toEqual(invalid('limit'));
});

test('patch: absent keeps, null clears, every write adds one to the revision; cancel and restore', async () => {
	const ana = await stubAs(ANA);
	const start = iso(realNow() + DAY);
	const made = (await ana('POST', base(), { starts_at: start, kind: 'sick', provider: 'Dr. Lee' }))
		.body;
	const at = `${base()}/${made.id}`;
	const patch = (body: unknown) => ana('PATCH', at, body);
	expect(await patch({ note: 'Fever' })).toMatchObject({
		status: 200,
		body: { ...made, note: 'Fever', revision: 1 }
	});
	expect(await patch({ provider: null, kind: null })).toMatchObject({
		status: 200,
		body: { provider: null, kind: 'sick', revision: 2 }
	});
	expect(await patch({ duration_min: 3 })).toEqual(invalid('duration_min'));
	expect(await patch({ cancelled: null })).toEqual(invalid('cancelled'));
	expect(await patch({ bogus: 1 })).toEqual({
		status: 422,
		body: { error: 'unknown_field', field: 'bogus' }
	});
	const cancelled = await patch({ cancelled: true });
	expect(cancelled.body).toMatchObject({ status: 'cancelled', revision: 3 });
	expect(Math.abs(Date.parse(cancelled.body.cancelled_at) / 1000 - realNow())).toBeLessThan(5);
	expect((await patch({ cancelled: true })).body.cancelled_at).toBe(cancelled.body.cancelled_at);
	expect((await patch({ cancelled: false })).body).toMatchObject({
		status: 'upcoming',
		cancelled_at: null,
		revision: 5
	});
	expect(await patch({ outcome_notes: 'Too early' })).toEqual({
		status: 409,
		body: { error: 'not_past' }
	});
	expect((await ana('GET', at)).body.revision).toBe(5);
});

test('outcome: the checks in order, the rows linked with the visit’s date, then stale on a second submit', async () => {
	const ana = await stubAs(ANA);
	const startsAt = realNow() - 2 * DAY;
	const day = iso(startsAt).slice(0, 10);
	const visit = (await ana('POST', base(), { starts_at: iso(startsAt), kind: 'well_child' })).body;
	const at = `${base()}/${visit.id}/outcome`;
	const post = (body: unknown) => ana('POST', at, body);
	const growth = { weight: { unit: 'kg', amount: 9000 } };
	const vaccine = { antigens: ['hib'] };
	const course = { name: 'Amoxicillin', dose: '5 ml', schedule: { kind: 'none' } };

	expect(await post({ outcome_notes: 'x' })).toEqual(invalid('revision'));
	expect(await post({ revision: 0 })).toEqual(invalid('outcome'));
	expect(
		await post({ revision: 0, vaccinations: Array.from({ length: 11 }, () => vaccine) })
	).toEqual(invalid('vaccinations'));
	expect(await post({ revision: 0, medications: Array.from({ length: 6 }, () => course) })).toEqual(
		invalid('medications')
	);
	expect(await post({ revision: 0, outcome_notes: null })).toEqual(invalid('outcome_notes'));
	expect(await post({ revision: 0, growth: { weight: { unit: 'kg', amount: 1 } } })).toEqual(
		invalid('growth.weight')
	);
	expect(
		await post({ revision: 0, vaccinations: [vaccine, { ...vaccine, lot: 'x'.repeat(41) }] })
	).toEqual(invalid('vaccinations[1].lot'));
	expect(
		await post({ revision: 0, medications: [{ ...course, schedule: { kind: 'weekly' } }] })
	).toEqual(invalid('medications[0].schedule'));
	expect(await post({ revision: 0, growth: { ...growth, height: 1 } })).toEqual({
		status: 422,
		body: { error: 'unknown_field', field: 'height' }
	});
	expect((await post({ revision: 0, vaccinations: vaccine })).status).toBe(400);
	expect(await post({ revision: 7, outcome_notes: 'x' })).toEqual({
		status: 409,
		body: { error: 'stale' }
	});

	const body = {
		revision: 0,
		outcome_notes: 'Healthy.',
		growth,
		vaccinations: [vaccine, { antigens: ['measles', 'rotavirus'], given_on: day }],
		medications: [course]
	};
	const done = await post(body);
	expect(done.status).toBe(200);
	expect(Object.keys(done.body).sort()).toEqual(VIEW_KEYS);
	expect(done.body).toMatchObject({ outcome_notes: 'Healthy.', revision: 1, status: 'past' });
	const { growth: g, vaccinations: v, medications: m } = done.body.linked;
	expect(g).toEqual([{ id: expect.any(String), measured_on: day }]);
	expect(v).toEqual([
		{ id: expect.any(String), given_on: day, antigens: ['hib'] },
		{ id: expect.any(String), given_on: day, antigens: ['measles', 'rotavirus'] }
	]);
	expect(m).toEqual([{ id: expect.any(String), name: 'Amoxicillin', dose_count: 0 }]);
	expect(await post(body)).toEqual({ status: 409, body: { error: 'stale' } });
	expect((await ana('GET', `${base()}/${visit.id}`)).body).toEqual(done.body);

	const kid = `/v1/kids/${IVY.id}`;
	expect((await ana('GET', `${kid}/growth/${g[0].id}`)).body.visit_id).toBe(visit.id);
	expect((await ana('GET', `${kid}/vaccinations/${v[1].id}`)).body.visit_id).toBe(visit.id);
	const linkedCourse = (await ana('GET', `${kid}/medications/${m[0].id}`)).body;
	expect(linkedCourse).toMatchObject({ visit_id: visit.id, start_at: iso(startsAt) });
	const listed = (await ana('GET', base())).body.visits[0];
	expect(listed.linked).toEqual({ growth: 1, vaccinations: 2, medications: 1 });
	const plain = await ana('POST', `${kid}/growth`, { measured_on: day, ...growth });
	expect(plain.body.visit_id).toBeNull();

	const upcoming = (await ana('POST', base(), { starts_at: iso(realNow() + DAY), kind: 'dental' }))
		.body;
	expect(
		await ana('POST', `${base()}/${upcoming.id}/outcome`, { revision: 0, outcome_notes: 'x' })
	).toEqual({ status: 409, body: { error: 'not_past' } });
});

test('has_outcome: a visit with notes or rows cannot be cancelled or moved ahead', async () => {
	const ana = await stubAs(ANA);
	const visit = (await ana('POST', base(), { starts_at: iso(realNow() - DAY), kind: 'sick' })).body;
	const at = `${base()}/${visit.id}`;
	const conflict = { status: 409, body: { error: 'has_outcome' } };
	expect((await ana('PATCH', at, { outcome_notes: 'Rest' })).body.revision).toBe(1);
	expect(await ana('PATCH', at, { cancelled: true })).toEqual(conflict);
	expect(await ana('PATCH', at, { starts_at: iso(realNow() + DAY) })).toEqual(conflict);
	expect((await ana('PATCH', at, { outcome_notes: null })).body.outcome_notes).toBeNull();
	await ana('POST', `${at}/outcome`, {
		revision: 2,
		growth: { length: { unit: 'cm', amount: 700 } }
	});
	expect(await ana('PATCH', at, { cancelled: true })).toEqual(conflict);
	expect((await ana('PATCH', at, { note: 'still editable' })).status).toBe(200);
});

test('delete: linked=delete takes the rows, keep leaves them unlinked, anything else is 422', async () => {
	const ana = await stubAs(ANA);
	const kid = `/v1/kids/${IVY.id}`;
	const record = async () => {
		const visit = (await ana('POST', base(), { starts_at: iso(realNow() - DAY), kind: 'other' }))
			.body;
		const done = await ana('POST', `${base()}/${visit.id}/outcome`, {
			revision: 0,
			growth: { weight: { unit: 'kg', amount: 9000 } },
			vaccinations: [{ antigens: ['hib'] }],
			medications: [{ name: 'Calpol', dose: '2.5 ml', schedule: { kind: 'none' } }]
		});
		return { visit, linked: done.body.linked };
	};
	const one = await record();
	const at = `${base()}/${one.visit.id}`;
	expect(await ana('DELETE', `${at}?linked=all`)).toEqual(invalid('linked'));
	expect(await ana('DELETE', `${base()}/vis-none?linked=all`)).toEqual(invalid('linked'));
	expect(await ana('DELETE', `${at}?linked=keep`)).toEqual({ status: 204, body: null });
	expect(await ana('GET', at)).toEqual(notFound);
	expect((await ana('GET', `${kid}/growth/${one.linked.growth[0].id}`)).body.visit_id).toBeNull();
	expect(
		(await ana('GET', `${kid}/vaccinations/${one.linked.vaccinations[0].id}`)).body.visit_id
	).toBeNull();
	expect(
		(await ana('GET', `${kid}/medications/${one.linked.medications[0].id}`)).body.visit_id
	).toBeNull();

	const two = await record();
	await ana('POST', `${kid}/medication-doses`, { medication: two.linked.medications[0].id });
	expect(await ana('DELETE', `${base()}/${two.visit.id}`)).toEqual({ status: 204, body: null });
	expect(await ana('GET', `${kid}/growth/${two.linked.growth[0].id}`)).toEqual(notFound);
	expect(await ana('GET', `${kid}/vaccinations/${two.linked.vaccinations[0].id}`)).toEqual(
		notFound
	);
	expect(await ana('GET', `${kid}/medications/${two.linked.medications[0].id}`)).toEqual(notFound);
	expect((await ana('GET', `${kid}/medication-doses`)).body.doses).toEqual([]);
});

test('home due: upcoming visits within seven kid-local days, soonest first, after the other entries', async () => {
	const ana = await stubAs(ANA);
	const t = realNow();
	const post = async (offset: number) =>
		(await ana('POST', base(), { starts_at: iso(t + offset), kind: 'dental' })).body;
	const later = await post(3 * DAY);
	const soon = await post(2 * H);
	await post(-H);
	await post(9 * DAY);
	const cancelled = await post(DAY);
	await ana('PATCH', `${base()}/${cancelled.id}`, { cancelled: true });
	const due = async () => (await ana('GET', `/v1/kids/${IVY.id}/home`)).body.due;
	expect(await due()).toEqual(
		[soon, later].map((v) => ({
			kind: 'visit',
			id: v.id,
			starts_at: v.starts_at,
			visit_kind: 'dental'
		}))
	);
	expect(Object.keys((await due())[0])).toEqual(['kind', 'id', 'starts_at', 'visit_kind']);
	await ana('PUT', `/v1/kids/${IVY.id}/modules/visits`, { enabled: false });
	expect(await due()).toEqual([]);
});

test('timeline: a visit row by starts_at with its counts; the types filter names it visit', async () => {
	const ana = await stubAs(ANA);
	const t = realNow();
	const visit = (await ana('POST', base(), { starts_at: iso(t - 2 * H), kind: 'emergency' })).body;
	await ana('POST', base(), { starts_at: iso(t + DAY), kind: 'dental' });
	const diaper = (await ana('POST', `/v1/kids/${IVY.id}/diapers`, { kind: 'wet' })).body;
	const row = {
		type: 'visit',
		at: visit.starts_at,
		...visit,
		linked: { growth: 0, vaccinations: 0, medications: 0 }
	};
	const timeline = `/v1/kids/${IVY.id}/timeline`;
	expect((await ana('GET', timeline)).body.items).toEqual([
		{ type: 'diaper', at: diaper.at, ...diaper },
		row
	]);
	expect((await ana('GET', `${timeline}?types=visit`)).body.items).toEqual([row]);
	expect((await ana('GET', `${timeline}?types=visit,visit`)).status).toBe(422);
	await ana('PUT', `/v1/kids/${IVY.id}/modules/visits`, { enabled: false });
	expect((await ana('GET', timeline)).body.items).toHaveLength(1);
	expect((await ana('GET', `${timeline}?types=visit`)).body.items).toEqual([row]);
});

test('calendar: a dated attachment, generic unless full, upcoming or cancelled alike', async () => {
	const ana = await stubAs(ANA);
	const startsAt = realNow() + 3 * DAY;
	const visit = (
		await ana('POST', base(), {
			starts_at: iso(startsAt),
			kind: 'specialist',
			provider: 'Dr. Lee',
			note: 'private reason'
		})
	).body;
	const at = `${base()}/${visit.id}/calendar`;
	const generic = await calendar(ANA, at);
	expect(generic.status).toBe(200);
	expect(generic.headers.get('content-type')).toBe('text/calendar; charset=utf-8');
	expect(generic.headers.get('content-disposition')).toBe(
		`attachment; filename="visit-${iso(startsAt).slice(0, 10)}.ics"`
	);
	expect(generic.headers.get('cache-control')).toBe('no-store');
	expect(generic.headers.get('x-content-type-options')).toBe('nosniff');
	expect(generic.text).toContain('SUMMARY:Medical visit\r\n');
	expect(generic.text).toContain(`UID:visit-${visit.id}@`);
	expect(generic.text).toContain('STUB CALENDAR FILE, NOT A REAL EVENT');
	expect(generic.text).not.toContain('Ivy');
	expect(generic.text).not.toContain('private reason');
	expect((await calendar(ANA, `${at}?detail=generic`)).text).toBe(generic.text);
	const full = await calendar(ANA, `${at}?detail=full`);
	expect(full.text).toContain('SUMMARY:Ivy: Specialist visit with Dr. Lee\r\n');
	const bad = await calendar(ANA, `${at}?detail=private`);
	expect([bad.status, bad.text]).toEqual([
		422,
		JSON.stringify({ error: 'invalid', field: 'detail' })
	]);
	await ana('PATCH', `${base()}/${visit.id}`, { cancelled: true });
	const cancelled = await calendar(ANA, at);
	expect(cancelled.text).toContain('STATUS:CANCELLED');
	expect(cancelled.text).toContain('SEQUENCE:1');
});

test('the module: twelfth in the list, on by default at every age; a caregiver has the same rights', async () => {
	const ana = await stubAs(ANA);
	const cleo = await stubAs(CLEO);
	const mia = (await ana('GET', `/v1/kids/${MIA.id}/modules`)).body.modules;
	expect(mia.map((m: { module: string }) => m.module).at(-1)).toBe('visits');
	expect(mia.at(-1)).toEqual({ module: 'visits', enabled: true, default: true, overridden: false });
	const leo = (await cleo('GET', `/v1/kids/${LEO.id}/modules`)).body.modules;
	expect(leo.at(-1)).toMatchObject({ module: 'visits', enabled: true, default: true });

	const made = await ana('POST', base(LEO.id), {
		starts_at: iso(realNow() - DAY),
		kind: 'sick'
	});
	expect(made.status).toBe(201);
	const at = `${base(LEO.id)}/${made.body.id}`;
	expect((await ana('POST', `${at}/outcome`, { revision: 0, outcome_notes: 'x' })).status).toBe(
		200
	);
	expect((await cleo('PATCH', at, { note: 'seen' })).body.revision).toBe(2);
	expect(await ana('DELETE', at)).toEqual({ status: 204, body: null });
});
