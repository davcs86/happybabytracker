import { expect, test } from '@playwright/test';
import { todayIn } from '../src/lib/format.js';
import { ANA, CLEO, IVY, LEO, MIA, resetStub, storeVaccineSchema, stubAs } from './stub-api.js';

// The stub's vaccination routes answer as the Go API does; its schedules are a stub, never checked against real data.

test.beforeEach(async () => {
	await resetStub();
});

const invalid = (field: string) => ({ status: 422, body: { error: 'invalid', field } });
const notFound = { status: 404, body: { error: 'not_found' } };
type Line = {
	id: string;
	status: string;
	kind: string;
	from_date: string;
	given_on: string | null;
	dismissal: { reason: string } | null;
};
type Group = {
	from: { n: number; unit: string };
	from_date: string;
	status: string;
	lines: Line[];
};

async function ivyOnUs() {
	const ana = await stubAs(ANA);
	expect((await ana('PATCH', `/v1/kids/${IVY.id}`, { vaccine_schema: 'us' })).status).toBe(200);
	const read = async () => (await ana('GET', `/v1/kids/${IVY.id}/vaccines`)).body;
	const lines = async () => ((await read()).groups as Group[]).flatMap((g) => g.lines);
	const dose = (antigens: string[], given_on = todayIn(IVY.timeZone)) =>
		ana('POST', `/v1/kids/${IVY.id}/vaccinations`, { given_on, antigens });
	return { ana, read, lines, dose };
}

test('the schedules and the catalogue', async () => {
	const ana = await stubAs(ANA);
	const schemas = (await ana('GET', '/v1/vaccine-schemas')).body.schemas;
	expect(schemas.map((s: { id: string }) => s.id)).toEqual(['us', 'uk']);
	for (const s of schemas) {
		expect(Object.keys(s).sort()).toEqual(
			[
				'attribution',
				'authority',
				'edition',
				'id',
				'licence',
				'name',
				'origin_url',
				'retrieved'
			].sort()
		);
	}
	const { antigens, products } = (await ana('GET', '/v1/vaccine-catalogue')).body;
	const ids = antigens.map((a: { id: string }) => a.id);
	expect(ids).toContain('other');
	for (const p of products) {
		expect(p).toMatchObject({ id: expect.any(String), name: expect.any(String) });
		for (const a of p.antigens) expect(ids).toContain(a);
	}
});

test('kids have no schedule by default; a guardian chooses, changes and clears one', async () => {
	const ana = await stubAs(ANA);
	const kids = (await ana('GET', '/v1/kids')).body.kids;
	expect(kids.map((k: { vaccine_schema: string | null }) => k.vaccine_schema)).toEqual([
		null,
		null,
		null
	]);
	expect(await ana('PATCH', `/v1/kids/${MIA.id}`, { vaccine_schema: 'ch' })).toMatchObject(
		invalid('vaccine_schema')
	);
	expect(await ana('PATCH', `/v1/kids/${MIA.id}`, { vaccine_schema: 'uk' })).toMatchObject({
		status: 200,
		body: { vaccine_schema: 'uk' }
	});
	expect(await ana('PATCH', `/v1/kids/${MIA.id}`, { name: 'Mia' })).toMatchObject({
		body: { vaccine_schema: 'uk' }
	});
	expect(await ana('PATCH', `/v1/kids/${MIA.id}`, { vaccine_schema: null })).toMatchObject({
		status: 200,
		body: { vaccine_schema: null }
	});
	const created = await ana('POST', '/v1/kids', {
		name: 'Zoe',
		birth_date: todayIn('UTC'),
		sex: 'female',
		time_zone: 'UTC',
		vaccine_schema: 'us'
	});
	expect(created).toMatchObject({ status: 201, body: { vaccine_schema: 'us' } });
});

test('a retired schedule id stays until it is changed, as newSchemaOK does', async () => {
	const ana = await stubAs(ANA);
	await storeVaccineSchema(MIA.id, 'ch');
	expect(await ana('PATCH', `/v1/kids/${MIA.id}`, { name: 'Mia' })).toMatchObject({
		status: 200,
		body: { vaccine_schema: 'ch' }
	});
	expect(await ana('PATCH', `/v1/kids/${MIA.id}`, { vaccine_schema: 'ch' })).toMatchObject({
		status: 200,
		body: { vaccine_schema: 'ch' }
	});
	expect(await ana('PATCH', `/v1/kids/${MIA.id}`, { vaccine_schema: 'zz' })).toMatchObject(
		invalid('vaccine_schema')
	);
	expect(await ana('PATCH', `/v1/kids/${MIA.id}`, { vaccine_schema: 'us' })).toMatchObject({
		status: 200,
		body: { vaccine_schema: 'us' }
	});
});

test('without a schedule every dose is listed under other', async () => {
	const ana = await stubAs(ANA);
	const given_on = todayIn(IVY.timeZone);
	await ana('POST', `/v1/kids/${IVY.id}/vaccinations`, { given_on, antigens: ['hib', 'typhoid'] });
	const v = (await ana('GET', `/v1/kids/${IVY.id}/vaccines`)).body;
	expect(v).toMatchObject({ schema: null, today: given_on, groups: [], also: [], recurring: [] });
	expect(v.other.map((o: { antigen: string }) => o.antigen).sort()).toEqual(['hib', 'typhoid']);
});

test('a schedule shows visits in age order with every status, also and recurring', async () => {
	const { read } = await ivyOnUs();
	const v = await read();
	expect(v.schema).toMatchObject({ id: 'us', omitted: expect.any(Array) });
	const groups = v.groups as Group[];
	const dates = groups.map((g) => g.from_date);
	expect(dates).toEqual([...dates].sort());
	const statuses = groups.flatMap((g) => g.lines.map((l) => l.status));
	for (const s of ['overdue', 'due', 'upcoming', 'available']) expect(statuses).toContain(s);
	for (const g of groups) {
		const rank = ['overdue', 'due', 'upcoming', 'available', 'dismissed', 'given'];
		const best = g.lines.map((l) => rank.indexOf(l.status)).sort((a, b) => a - b)[0];
		expect(g.status).toBe(rank[best]);
	}
	const available = groups.flatMap((g) => g.lines).find((l) => l.status === 'available');
	expect(available?.kind).toBe('may_not_be_needed');
	expect(v.also.length).toBeGreaterThan(0);
	for (const l of v.also) expect(l).toMatchObject({ kind: 'supplementary', status: 'available' });
	expect(v.recurring[0]).toMatchObject({ line_id: expect.any(String), doses: [] });
});

test('doses fill lines by count per antigen; extras are other, recurring doses are listed', async () => {
	const { lines, read, dose } = await ivyOnUs();
	const series = async (antigen: string) =>
		(await lines()).filter((l) => l.id.startsWith(`${antigen}-`));
	const before = await series('diphtheria');
	expect(before.length).toBeGreaterThan(1);
	for (let i = 0; i <= before.length; i++) expect((await dose(['diphtheria'])).status).toBe(201);
	expect((await series('diphtheria')).every((l) => l.status === 'given')).toBe(true);
	const v = await read();
	expect(v.other.map((o: { antigen: string }) => o.antigen)).toEqual(['diphtheria']);
	const flu = v.recurring[0];
	await dose([flu.antigen]);
	expect((await read()).recurring[0].doses).toHaveLength(1);
});

test('an administration is validated field by field in the API order', async () => {
	const ana = await stubAs(ANA);
	const path = `/v1/kids/${IVY.id}/vaccinations`;
	const today = todayIn(IVY.timeZone);
	const ok = { given_on: today, antigens: ['hib'] };
	expect(await ana('POST', path, { antigens: ['hib'] })).toMatchObject(invalid('given_on'));
	expect(await ana('POST', path, { ...ok, given_on: '2999-01-01' })).toMatchObject(
		invalid('given_on')
	);
	expect(await ana('POST', path, { ...ok, given_on: '2001-01-01' })).toMatchObject(
		invalid('given_on')
	);
	expect(await ana('POST', path, { ...ok, given_on: '2026-02-30' })).toMatchObject(
		invalid('given_on')
	);
	expect(await ana('POST', path, { ...ok, given_on: null })).toMatchObject(invalid('given_on'));
	expect(await ana('POST', path, { given_on: today })).toMatchObject(invalid('antigens'));
	for (const antigens of [[], ['hib', 'hib'], ['zzz'], null]) {
		expect(await ana('POST', path, { ...ok, antigens })).toMatchObject(invalid('antigens'));
	}
	expect(await ana('POST', path, { ...ok, lot: 'x'.repeat(41) })).toMatchObject(invalid('lot'));
	expect(await ana('POST', path, { ...ok, product: 'x'.repeat(81) })).toMatchObject(
		invalid('product')
	);
	expect(await ana('POST', path, { ...ok, antigens: ['other'] })).toMatchObject(invalid('product'));
	expect(await ana('POST', path, { ...ok, antigens: ['other'], product: ' ' })).toMatchObject(
		invalid('product')
	);
	expect(await ana('POST', path, { ...ok, note: 'n'.repeat(501) })).toMatchObject(invalid('note'));
	expect(await ana('POST', path, { ...ok, extra: 1 })).toMatchObject({
		status: 422,
		body: { error: 'unknown_field', field: 'extra' }
	});
	expect(await ana('POST', path, { ...ok, antigens: 'hib' })).toMatchObject({ status: 400 });
});

test('an administration is created, listed newest first, patched and deleted', async () => {
	const ana = await stubAs(ANA);
	const path = `/v1/kids/${IVY.id}/vaccinations`;
	const today = todayIn(IVY.timeZone);
	const first = await ana('POST', path, {
		given_on: IVY.birthDate,
		antigens: ['measles', 'hib'],
		lot: 'L1',
		product: 'Combo'
	});
	expect(first).toMatchObject({
		status: 201,
		body: {
			given_on: IVY.birthDate,
			antigens: ['hib', 'measles'],
			lot: 'L1',
			product: 'Combo',
			note: null
		}
	});
	const second = (await ana('POST', path, { given_on: today, antigens: ['hib'] })).body;
	const list = (await ana('GET', path)).body.vaccinations;
	expect(list.map((a: { id: string }) => a.id)).toEqual([second.id, first.body.id]);
	const one = `${path}/${first.body.id}`;
	expect(await ana('PATCH', one, { lot: null, note: 'x' })).toMatchObject({
		status: 200,
		body: { lot: null, product: 'Combo', note: 'x', antigens: ['hib', 'measles'] }
	});
	expect(await ana('PATCH', one, { antigens: ['other'] })).toMatchObject({
		status: 200,
		body: { antigens: ['other'], product: 'Combo' }
	});
	expect(await ana('PATCH', one, { product: null })).toMatchObject(invalid('product'));
	expect(await ana('PATCH', one, { antigens: [] })).toMatchObject(invalid('antigens'));
	expect((await ana('GET', one)).body).toMatchObject({ antigens: ['other'], lot: null });
	expect(await ana('DELETE', one)).toMatchObject({ status: 204 });
	expect(await ana('GET', one)).toMatchObject(notFound);
});

test("another kid's administration, an unknown one and a kid without a grant are the same 404", async () => {
	const ana = await stubAs(ANA);
	const cleo = await stubAs(CLEO);
	const given_on = todayIn(LEO.timeZone);
	const leo = await ana('POST', `/v1/kids/${LEO.id}/vaccinations`, { given_on, antigens: ['hib'] });
	expect(leo.status).toBe(201);
	const other = `/v1/kids/${MIA.id}/vaccinations/${leo.body.id}`;
	expect(await ana('GET', other)).toMatchObject(notFound);
	expect(await ana('PATCH', other, { note: 'x' })).toMatchObject(notFound);
	expect(await ana('DELETE', other)).toMatchObject(notFound);
	expect(await ana('GET', `/v1/kids/${MIA.id}/vaccinations/v-unknown`)).toMatchObject(notFound);
	expect(await ana('GET', `/v1/kids/${MIA.id}/vaccinations/a%20b`)).toMatchObject(notFound);
	for (const path of ['vaccines', 'vaccinations']) {
		expect(await cleo('GET', `/v1/kids/${MIA.id}/${path}`)).toMatchObject(notFound);
	}
});

test('a guardian dismisses a line, undoes it, and given beats dismissed', async () => {
	const { ana, lines, dose } = await ivyOnUs();
	const overdue = (await lines()).find((l) => l.status === 'overdue') as Line;
	const path = `/v1/kids/${IVY.id}/vaccine-dismissals/${overdue.id}`;
	expect(await ana('PUT', path, { reason: 'x' })).toMatchObject(invalid('reason'));
	expect(await ana('PUT', path, { reason: 'given_unknown' })).toMatchObject({
		status: 200,
		body: { line: overdue.id, reason: 'given_unknown' }
	});
	const find = async () => (await lines()).find((l) => l.id === overdue.id);
	expect(await find()).toMatchObject({
		status: 'dismissed',
		dismissal: { reason: 'given_unknown' }
	});
	expect(await ana('DELETE', path)).toMatchObject({ status: 204 });
	expect((await find())?.status).toBe('overdue');
	await ana('PUT', path, { reason: 'not_needed' });
	await dose([overdue.id.split('-')[0]]);
	expect(await find()).toMatchObject({ status: 'given', dismissal: null });
});

test('a dismissal is 404 off the lines, without a schedule, and 403 for a caregiver', async () => {
	const { ana, read } = await ivyOnUs();
	const v = await read();
	const base = `/v1/kids/${IVY.id}/vaccine-dismissals`;
	for (const line of [v.also[0].id, v.recurring[0].line_id, 'hib-99']) {
		expect(await ana('PUT', `${base}/${line}`, { reason: 'not_needed' })).toMatchObject(notFound);
		expect(await ana('DELETE', `${base}/${line}`)).toMatchObject(notFound);
	}
	const line = (v.groups[0] as Group).lines[0].id;
	expect(
		await ana('PUT', `/v1/kids/${MIA.id}/vaccine-dismissals/${line}`, { reason: 'not_needed' })
	).toMatchObject(notFound);
	expect(
		await ana('POST', `/v1/kids/${MIA.id}/vaccine-dismissals`, {
			before: '2030-01-01',
			reason: 'not_needed'
		})
	).toMatchObject(notFound);
	const cleo = await stubAs(CLEO);
	await cleo('PATCH', `/v1/kids/${LEO.id}`, { vaccine_schema: 'uk' });
	const leoLine = ((await ana('GET', `/v1/kids/${LEO.id}/vaccines`)).body.groups[0] as Group)
		.lines[0].id;
	expect(
		await ana('PUT', `/v1/kids/${LEO.id}/vaccine-dismissals/${leoLine}`, { reason: 'not_needed' })
	).toMatchObject({ status: 403 });
	expect(
		await ana('POST', `/v1/kids/${LEO.id}/vaccine-dismissals`, {
			before: '2030-01-01',
			reason: 'not_needed'
		})
	).toMatchObject({ status: 403 });
	expect(await ana('PATCH', `/v1/kids/${LEO.id}`, { vaccine_schema: 'us' })).toMatchObject({
		status: 403
	});
});

test('dismiss-before dismisses unsettled routine lines that open before the date, once', async () => {
	const { ana, lines, dose } = await ivyOnUs();
	const path = `/v1/kids/${IVY.id}/vaccine-dismissals`;
	expect(await ana('POST', path, { before: '2026-02-30', reason: 'not_needed' })).toMatchObject(
		invalid('before')
	);
	expect(await ana('POST', path, { before: '2030-01-01', reason: 'x' })).toMatchObject(
		invalid('reason')
	);
	expect(await ana('POST', path, { before: IVY.birthDate, reason: 'not_needed' })).toMatchObject({
		status: 200,
		body: { dismissed: 0 }
	});
	const all = await lines();
	const given = all.find((l) => l.status === 'overdue') as Line;
	await dose([given.id.split('-')[0]]);
	const today = todayIn(IVY.timeZone);
	const eligible = (await lines()).filter(
		(l) =>
			['routine', 'may_not_be_needed'].includes(l.kind) &&
			!['given', 'dismissed', 'available'].includes(l.status) &&
			l.from_date < today
	);
	expect(eligible.length).toBeGreaterThan(0);
	expect(await ana('POST', path, { before: today, reason: 'not_needed' })).toMatchObject({
		status: 200,
		body: { dismissed: eligible.length }
	});
	const after = await lines();
	for (const l of eligible) expect(after.find((a) => a.id === l.id)?.status).toBe('dismissed');
	expect(after.find((l) => l.id === given.id)?.status).toBe('given');
	expect(after.some((l) => l.status === 'upcoming')).toBe(true);
	expect(after.some((l) => l.status === 'available')).toBe(true);
	expect(await ana('POST', path, { before: today, reason: 'not_needed' })).toMatchObject({
		body: { dismissed: 0 }
	});
});

test('the home lists due visits only with a schedule and the module on', async () => {
	const ana = await stubAs(ANA);
	const due = async () => (await ana('GET', `/v1/kids/${IVY.id}/home`)).body.due;
	expect(await due()).toEqual([]);
	await ana('PATCH', `/v1/kids/${IVY.id}`, { vaccine_schema: 'us' });
	const v = (await ana('GET', `/v1/kids/${IVY.id}/vaccines`)).body;
	const visits = (v.groups as Group[]).filter((g) =>
		g.lines.some((l) => ['due', 'overdue'].includes(l.status))
	);
	const entries = await due();
	expect(entries).toEqual(
		visits.map((g) => ({
			kind: 'vaccine',
			from: g.from,
			status: g.lines.some((l) => l.status === 'overdue') ? 'overdue' : 'due',
			count: g.lines.filter((l) => ['due', 'overdue'].includes(l.status)).length
		}))
	);
	expect(entries.map((e: { status: string }) => e.status)).toEqual(
		expect.arrayContaining(['due', 'overdue'])
	);
	await ana('PUT', `/v1/kids/${IVY.id}/modules/vaccination`, { enabled: false });
	expect(await due()).toEqual([]);
});

test('the 501st administration of a kid is refused with limit', async () => {
	const ana = await stubAs(ANA);
	const path = `/v1/kids/${MIA.id}/vaccinations`;
	const body = { given_on: todayIn(MIA.timeZone), antigens: ['hib'] };
	for (let i = 0; i < 500; i++) expect((await ana('POST', path, body)).status).toBe(201);
	expect(await ana('POST', path, body)).toMatchObject({ status: 409, body: { error: 'limit' } });
	expect(await ana('POST', `/v1/kids/${IVY.id}/vaccinations`, body)).toMatchObject({ status: 201 });
});

test('clearing a line that was never dismissed is still 204', async () => {
	const { ana, lines } = await ivyOnUs();
	const line = (await lines())[0];
	expect(line.dismissal).toBeNull();
	const path = `/v1/kids/${IVY.id}/vaccine-dismissals/${line.id}`;
	expect(await ana('DELETE', path)).toMatchObject({ status: 204 });
	expect(await ana('DELETE', path)).toMatchObject({ status: 204 });
	expect((await lines())[0].status).toBe(line.status);
});
