import { expect, test } from '@playwright/test';
import { todayIn } from '../src/lib/format.js';
import { ANA, CLEO, IVY, LEO, MIA, resetStub, stubAs } from './stub-api.js';

// The stub's growth routes answer as the Go API does; its scores are fake and never checked as numbers.

test.beforeEach(async () => {
	await resetStub();
});

const invalid = (field: string) => ({ status: 422, body: { error: 'invalid', field } });
const kg = (amount: number) => ({ unit: 'kg', amount });
const cm = (amount: number) => ({ unit: 'cm', amount });

test('kids carry their growth source, who by default; a guardian changes it', async () => {
	const ana = await stubAs(ANA);
	const kids = (await ana('GET', '/v1/kids')).body.kids;
	expect(kids.map((k: { growth_source: string }) => k.growth_source)).toEqual([
		'who',
		'who',
		'who'
	]);
	expect(await ana('PATCH', `/v1/kids/${MIA.id}`, { growth_source: 'uk-who' })).toMatchObject(
		invalid('growth_source')
	);
	expect(await ana('PATCH', `/v1/kids/${MIA.id}`, { growth_source: 'cdc' })).toMatchObject({
		status: 200,
		body: { growth_source: 'cdc' }
	});
	const created = await ana('POST', '/v1/kids', {
		name: 'Zoe',
		birth_date: todayIn('UTC'),
		sex: 'female',
		time_zone: 'UTC'
	});
	expect(created).toMatchObject({ status: 201, body: { growth_source: 'who' } });
});

test('a measurement is validated field by field in the API order', async () => {
	const ana = await stubAs(ANA);
	const path = `/v1/kids/${IVY.id}/growth`;
	const today = todayIn(IVY.timeZone);
	const ok = { measured_on: today, weight: kg(9000) };
	expect(await ana('POST', path, { weight: kg(9000) })).toMatchObject(invalid('measured_on'));
	expect(await ana('POST', path, { ...ok, measured_on: '2999-01-01' })).toMatchObject(
		invalid('measured_on')
	);
	expect(await ana('POST', path, { ...ok, measured_on: '2001-01-01' })).toMatchObject(
		invalid('measured_on')
	);
	expect(await ana('POST', path, { ...ok, measured_on: '2026-02-30' })).toMatchObject(
		invalid('measured_on')
	);
	expect(await ana('POST', path, { measured_on: today })).toMatchObject(invalid('measurements'));
	expect(await ana('POST', path, { ...ok, weight: kg(199) })).toMatchObject(invalid('weight'));
	expect(await ana('POST', path, { ...ok, weight: { unit: 'st', amount: 3 } })).toMatchObject(
		invalid('weight')
	);
	expect(await ana('POST', path, { ...ok, length: cm(149) })).toMatchObject(invalid('length'));
	expect(await ana('POST', path, { ...ok, head: cm(801) })).toMatchObject(invalid('head'));
	expect(await ana('POST', path, { ...ok, note: 'n'.repeat(501) })).toMatchObject(invalid('note'));
	expect(await ana('POST', path, { ...ok, extra: 1 })).toMatchObject({
		status: 422,
		body: { error: 'unknown_field', field: 'extra' }
	});
	expect(
		await ana('POST', path, { ...ok, weight: { unit: 'kg', amount: 9000, x: 1 } })
	).toMatchObject({ status: 422, body: { error: 'unknown_field', field: 'x' } });
	expect(await ana('POST', path, { ...ok, weight: '9 kg' })).toMatchObject({ status: 400 });
});

test('a created row converts its units, scores and lists newest first', async () => {
	const ana = await stubAs(ANA);
	const path = `/v1/kids/${IVY.id}/growth`;
	const today = todayIn(IVY.timeZone);
	const first = await ana('POST', path, {
		measured_on: IVY.birthDate,
		weight: { unit: 'lb', amount: 13150 },
		length: { unit: 'in', amount: 1950 }
	});
	expect(first).toMatchObject({
		status: 201,
		body: {
			measured_on: IVY.birthDate,
			weight: { grams: 3728, oz_hundredths: 13150, typed_unit: 'lb' },
			length: { mm: 495, in_hundredths: 1950, typed_unit: 'in' },
			head: null,
			note: null,
			age: { days: 0, corrected_days: null },
			scores: {
				weight: { scored: true },
				length: { scored: true },
				head: null,
				bmi: { scored: true }
			}
		}
	});
	expect(typeof first.body.scores.bmi.value).toBe('number');
	const second = await ana('POST', path, { measured_on: today, head: cm(450) });
	expect(second.status).toBe(201);
	const list = (await ana('GET', path)).body.growth;
	expect(list.map((r: { id: string }) => r.id)).toEqual([second.body.id, first.body.id]);
});

test('an edit keeps an unchanged typed value, null clears, and one measure must stay', async () => {
	const ana = await stubAs(ANA);
	const path = `/v1/kids/${IVY.id}/growth`;
	const row = (
		await ana('POST', path, {
			measured_on: todayIn(IVY.timeZone),
			weight: { unit: 'lb', amount: 13150 },
			length: cm(700)
		})
	).body;
	expect(
		await ana('PATCH', `${path}/${row.id}`, { weight: { unit: 'lb', amount: 13150 }, note: 'x' })
	).toMatchObject({ status: 200, body: { weight: row.weight, note: 'x' } });
	expect(await ana('PATCH', `${path}/${row.id}`, { length: null })).toMatchObject({
		status: 200,
		body: { length: null, scores: { length: null, bmi: null } }
	});
	expect(await ana('PATCH', `${path}/${row.id}`, { weight: null })).toMatchObject(
		invalid('measurements')
	);
	expect((await ana('GET', `${path}/${row.id}`)).body.weight).toEqual(row.weight);
	expect(await ana('DELETE', `${path}/${row.id}`)).toMatchObject({ status: 204 });
	expect(await ana('GET', `${path}/${row.id}`)).toMatchObject({ status: 404 });
});

test("another kid's row, an unknown row and a kid without a grant are the same 404", async () => {
	const ana = await stubAs(ANA);
	const cleo = await stubAs(CLEO);
	const leoRow = (await ana('GET', `/v1/kids/${LEO.id}/growth`)).body.growth[0].id;
	const notFound = { status: 404, body: { error: 'not_found' } };
	expect(await ana('GET', `/v1/kids/${MIA.id}/growth/${leoRow}`)).toMatchObject(notFound);
	expect(await ana('PATCH', `/v1/kids/${MIA.id}/growth/${leoRow}`, { note: 'x' })).toMatchObject(
		notFound
	);
	expect(await ana('DELETE', `/v1/kids/${MIA.id}/growth/${leoRow}`)).toMatchObject(notFound);
	expect(await ana('GET', `/v1/kids/${MIA.id}/growth/g-unknown`)).toMatchObject(notFound);
	expect(await cleo('GET', `/v1/kids/${MIA.id}/growth`)).toMatchObject(notFound);
	expect(await cleo('GET', `/v1/kids/${MIA.id}/growth/curves?measure=weight`)).toMatchObject(
		notFound
	);
});

test('Leo, born at 35 weeks 2 days, is scored at corrected age until his second birthday', async () => {
	const ana = await stubAs(ANA);
	const rows = (await ana('GET', `/v1/kids/${LEO.id}/growth`)).body.growth;
	const byDate = Object.fromEntries(
		rows.map((r: { measured_on: string }) => [r.measured_on, r])
	) as Record<string, { age: { days: number; corrected_days: number | null }; scores: unknown }>;
	const secondBirthday = '2026-08-20';
	const dates = Object.keys(byDate);
	expect(dates.some((d) => d < secondBirthday)).toBe(true);
	expect(dates.some((d) => d >= secondBirthday)).toBe(true);
	for (const [date, r] of Object.entries(byDate)) {
		if (date >= secondBirthday) expect(r.age.corrected_days).toBeNull();
		else expect(r.age.corrected_days).toBe(r.age.days - 33);
	}
	const beforeTerm = rows.find(
		(r: { age: { corrected_days: number | null } }) => (r.age.corrected_days ?? 0) < 0
	);
	expect(beforeTerm.scores.weight).toEqual({ scored: false, reason: 'before_term' });
	expect(
		rows.some(
			(r: { age: { corrected_days: number | null }; scores: { weight: { scored: boolean } } }) =>
				(r.age.corrected_days ?? -1) >= 0 && r.scores.weight.scored
		)
	).toBe(true);
	expect((await ana('GET', `/v1/kids/${LEO.id}/home`)).body.corrected_birth_date).toBeNull();
});

test('a kid born preterm this week has a corrected birth date and a corrected curve axis', async () => {
	const ana = await stubAs(ANA);
	const today = todayIn('UTC');
	const kid = (
		await ana('POST', '/v1/kids', {
			name: 'Pip',
			birth_date: today,
			sex: 'male',
			gestation_weeks: 34,
			gestation_days: 0,
			time_zone: 'UTC'
		})
	).body;
	const due = new Date(Date.parse(`${today}T00:00:00Z`) + 42 * 86400_000)
		.toISOString()
		.slice(0, 10);
	expect((await ana('GET', `/v1/kids/${kid.id}/home`)).body.corrected_birth_date).toBe(due);
	expect(
		(await ana('GET', `/v1/kids/${kid.id}/growth/curves?measure=weight`)).body.corrected_age_axis
	).toBe(true);
	expect((await ana('GET', `/v1/kids/${MIA.id}/home`)).body.corrected_birth_date).toBeNull();
});

test('curves: five WHO lines, nine CDC lines, CDC BMI adds 85; coverage clamps and bad parameters', async () => {
	const ana = await stubAs(ANA);
	const curves = (q: string) => ana('GET', `/v1/kids/${MIA.id}/growth/curves?${q}`);
	const percentiles = async (q: string) =>
		(await curves(q)).body.lines.map((l: { percentile: number }) => l.percentile);
	expect(await percentiles('measure=weight')).toEqual([3, 15, 50, 85, 97]);
	const who = (await curves('measure=length&from_months=0&to_months=12')).body;
	expect(who).toMatchObject({
		source: 'who',
		measure: 'length',
		sex: 'female',
		covered: true,
		reason: null,
		from_months: 0,
		to_months: 12,
		corrected_age_axis: false
	});
	expect(who.lines[0].points[0]).toMatchObject({ age_months: 0 });
	expect(typeof who.lines[0].points[0].imperial).toBe('number');
	expect((await curves('measure=bmi')).body.lines[0].points[0].imperial).toBeNull();
	await ana('PATCH', `/v1/kids/${MIA.id}`, { growth_source: 'cdc' });
	expect(await percentiles('measure=weight')).toEqual([3, 5, 10, 25, 50, 75, 90, 95, 97]);
	expect(await percentiles('measure=bmi&from_months=24&to_months=36')).toEqual([
		3, 5, 10, 25, 50, 75, 85, 90, 95, 97
	]);
	expect((await curves('measure=bmi&to_months=20')).body).toMatchObject({
		covered: false,
		reason: 'not_covered',
		lines: []
	});
	expect((await curves('measure=head&to_months=120')).body.to_months).toBe(36);
	expect(await curves('measure=height')).toMatchObject(invalid('measure'));
	expect(await curves('')).toMatchObject(invalid('measure'));
	expect(await curves('measure=weight&from_months=x')).toMatchObject(invalid('from_months'));
	expect(await curves('measure=weight&to_months=241')).toMatchObject(invalid('to_months'));
	expect(await curves('measure=weight&from_months=10&to_months=5')).toMatchObject(
		invalid('from_months')
	);
});

test('the growth sources are who then cdc, each with its provenance and four measures', async () => {
	const ana = await stubAs(ANA);
	const sources = (await ana('GET', '/v1/growth-sources')).body.sources;
	expect(sources.map((s: { id: string }) => s.id)).toEqual(['who', 'cdc']);
	for (const s of sources) {
		expect(s).toMatchObject({
			name: expect.any(String),
			attribution: expect.any(String),
			licence: expect.any(String),
			version: expect.any(String),
			origin_url: expect.any(String),
			retrieved: expect.any(String)
		});
		expect(s.coverage.map((c: { measure: string }) => c.measure)).toEqual([
			'weight',
			'length',
			'head',
			'bmi'
		]);
	}
});
