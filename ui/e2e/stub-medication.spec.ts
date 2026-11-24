import { expect, test } from '@playwright/test';
import { ANA, DAN, IVY, LEO, MIA, STUB_URL, moveStubClock, resetStub, stubAs } from './stub-api.js';

// The stub's symptom, medication, dose and Due routes answer as the Go API does; its clock moves only through
// /__clock, and /__reset puts it back.

test.beforeEach(async () => {
	await resetStub();
});

const invalid = (field: string) => ({ status: 422, body: { error: 'invalid', field } });
const notFound = { status: 404, body: { error: 'not_found' } };
const iso = (unixS: number) => new Date(unixS * 1000).toISOString().replace(/\.\d{3}Z$/, 'Z');
const realNow = () => Math.floor(Date.now() / 1000);
// Puts the stub clock at unixS (within the few seconds a request takes).
const clockAt = (unixS: number) => moveStubClock(unixS - realNow());
// Noon UTC tomorrow: Ivy's zone is UTC, so the kid-local day ends twelve hours later.
const NOON = (() => {
	const d = new Date();
	return Date.UTC(d.getUTCFullYear(), d.getUTCMonth(), d.getUTCDate() + 1, 12) / 1000;
})();
const H = 3600;
const M = 60;

test('the clock control moves the stub clock and reset restores the real one', async () => {
	const ana = await stubAs(ANA);
	const meds = `/v1/kids/${IVY.id}/medications`;
	const course = (
		await ana('POST', meds, { name: 'Calpol', dose: '2.5 ml', schedule: { kind: 'none' } })
	).body;
	const doses = `/v1/kids/${IVY.id}/medication-doses`;
	// Six minutes ahead of the real clock is too far ahead, until the stub clock moves an hour on.
	const ahead = iso(realNow() + 6 * M);
	expect(await ana('POST', doses, { medication: course.id, given_at: ahead })).toMatchObject(
		invalid('given_at')
	);
	await moveStubClock(H);
	expect((await ana('POST', doses, { medication: course.id, given_at: ahead })).status).toBe(201);
	const moved = (await ana('POST', doses, { medication: course.id })).body;
	expect(Math.abs(Date.parse(moved.given_at) / 1000 - (realNow() + H))).toBeLessThan(5);
	await resetStub();
	const ana2 = await stubAs(ANA);
	const fresh = (
		await ana2('POST', meds, { name: 'Calpol', dose: '2.5 ml', schedule: { kind: 'none' } })
	).body;
	const real = (await ana2('POST', doses, { medication: fresh.id })).body;
	expect(Math.abs(Date.parse(real.given_at) / 1000 - realNow())).toBeLessThan(5);
	expect(await ana2('POST', doses, { medication: fresh.id, given_at: ahead })).toMatchObject(
		invalid('given_at')
	);
});

test('symptoms: validation order, fever temperature and method, other text', async () => {
	const ana = await stubAs(ANA);
	const path = `/v1/kids/${IVY.id}/symptoms`;
	const post = (body: unknown) => ana('POST', path, body);
	expect(await post({})).toMatchObject(invalid('symptom'));
	expect(await post({ at: iso(realNow() + 6 * M), symptom: 'nope' })).toMatchObject(invalid('at'));
	expect(await post({ symptom: 'nope', severity: 'bad' })).toMatchObject(invalid('symptom'));
	expect(await post({ symptom: 'cough', severity: 'bad', other_text: 'x' })).toMatchObject(
		invalid('severity')
	);
	expect(await post({ symptom: 'cough', other_text: 'x', method: 'oral' })).toMatchObject(
		invalid('other_text')
	);
	expect(await post({ symptom: 'other' })).toMatchObject(invalid('other_text'));
	expect(await post({ symptom: 'other', other_text: '   ' })).toMatchObject(invalid('other_text'));
	expect(await post({ symptom: 'other', other_text: 'é'.repeat(81) })).toMatchObject(
		invalid('other_text')
	);
	expect(await post({ symptom: 'fever', method: 'oral' })).toMatchObject(invalid('temperature'));
	expect(
		await post({ symptom: 'fever', temperature: { unit: 'c', amount: 3850 }, note: 'x' })
	).toMatchObject(invalid('method'));
	expect(
		await post({ symptom: 'cough', temperature: { unit: 'c', amount: 3850 }, method: 'oral' })
	).toMatchObject(invalid('temperature'));
	expect(await post({ symptom: 'cough', method: 'oral' })).toMatchObject(invalid('method'));
	const fever = (temperature: unknown, method = 'oral') =>
		post({ symptom: 'fever', temperature, method });
	for (const t of [
		{ unit: 'c', amount: 2999 },
		{ unit: 'c', amount: 4501 },
		{ unit: 'f', amount: 8599 },
		{ unit: 'f', amount: 11301 },
		{ unit: 'c', amount: 0 },
		{ unit: 'c', amount: -3800 },
		{ unit: 'k', amount: 3800 },
		{ unit: 'c' }
	]) {
		expect(await fever(t)).toMatchObject(invalid('temperature'));
	}
	expect(await fever({ unit: 'c', amount: 3800 }, 'mouth')).toMatchObject(invalid('method'));
	expect((await fever({ unit: 'c', amount: 38.5 })).status).toBe(400);
	expect(await fever({ unit: 'c', amount: 3800, scale: 1 })).toMatchObject({
		status: 422,
		body: { error: 'unknown_field', field: 'scale' }
	});
	expect(await post({ symptom: 'cough', note: 'x'.repeat(501), severity: 'mild' })).toMatchObject(
		invalid('note')
	);
	expect((await fever({ unit: 'c', amount: 3000 })).status).toBe(201);
	expect((await fever({ unit: 'f', amount: 8600 })).status).toBe(201);

	const c = await fever({ unit: 'c', amount: 3850 });
	expect(c.status).toBe(201);
	expect(Object.keys(c.body).sort()).toEqual(
		['at', 'id', 'method', 'note', 'other_text', 'severity', 'symptom', 'temperature'].sort()
	);
	expect(c.body).toMatchObject({
		symptom: 'fever',
		severity: null,
		other_text: null,
		method: 'oral',
		note: null,
		temperature: { c_hundredths: 3850, f_hundredths: 10130, typed_unit: 'c' }
	});
	const f = await fever({ unit: 'f', amount: 10040 }, 'ear');
	expect(f.body.temperature).toEqual({ c_hundredths: 3800, f_hundredths: 10040, typed_unit: 'f' });
	const other = await post({ symptom: 'other', other_text: '  itchy  ', severity: 'mild' });
	expect(other.body).toMatchObject({ other_text: 'itchy', severity: 'mild', temperature: null });
});

test('symptoms: a PATCH merges onto the stored row and leaving fever or other needs explicit nulls', async () => {
	const ana = await stubAs(ANA);
	const path = `/v1/kids/${IVY.id}/symptoms`;
	const made = (
		await ana('POST', path, {
			symptom: 'fever',
			temperature: { unit: 'f', amount: 10040 },
			method: 'rectal',
			severity: 'moderate'
		})
	).body;
	const at = `${path}/${made.id}`;
	const patch = (body: unknown) => ana('PATCH', at, body);
	expect(await patch({ note: 'only the note' })).toMatchObject({
		status: 200,
		body: { ...made, note: 'only the note' }
	});
	expect(await patch({ temperature: { unit: 'f', amount: 10040 } })).toMatchObject({
		status: 200,
		body: { temperature: made.temperature }
	});
	expect(await patch({ symptom: 'cough' })).toMatchObject(invalid('temperature'));
	expect(await patch({ symptom: 'cough', temperature: null })).toMatchObject(
		invalid('temperature')
	);
	expect(await patch({ symptom: 'cough', method: null })).toMatchObject(invalid('temperature'));
	expect(await patch({ method: null })).toMatchObject(invalid('method'));
	expect(await patch({ temperature: null })).toMatchObject(invalid('temperature'));
	const cough = await patch({ symptom: 'cough', temperature: null, method: null, severity: null });
	expect(cough).toMatchObject({
		status: 200,
		body: { symptom: 'cough', temperature: null, method: null, severity: null }
	});
	expect(await patch({ symptom: 'fever' })).toMatchObject(invalid('temperature'));
	expect(await patch({ symptom: 'fever', temperature: { unit: 'c', amount: 3900 } })).toMatchObject(
		invalid('method')
	);
	expect(await patch({ symptom: 'other', other_text: 'rash on arm' })).toMatchObject({
		status: 200,
		body: { symptom: 'other', other_text: 'rash on arm' }
	});
	expect(await patch({ symptom: 'cough' })).toMatchObject(invalid('other_text'));
	expect((await patch({ symptom: 'cough', other_text: null })).status).toBe(200);
	expect(await ana('GET', at)).toMatchObject({
		status: 200,
		body: { symptom: 'cough', other_text: null, note: 'only the note' }
	});
});

test('symptoms: 404 identity, the list and the timeline follow the sickness module', async () => {
	const ana = await stubAs(ANA);
	const dan = await stubAs(DAN);
	const ivy = `/v1/kids/${IVY.id}/symptoms`;
	const leo = (await ana('POST', `/v1/kids/${LEO.id}/symptoms`, { symptom: 'rash' })).body;
	const mine = (await ana('POST', ivy, { symptom: 'cough' })).body;
	for (const id of [leo.id, 'e-000000009999', 'not an id!']) {
		for (const method of ['GET', 'PATCH', 'DELETE']) {
			expect(
				await ana(method, `${ivy}/${encodeURIComponent(id)}`, method === 'PATCH' ? {} : undefined)
			).toEqual(notFound);
		}
	}
	expect(await dan('GET', ivy)).toEqual(notFound);
	expect(await dan('GET', `${ivy}/${mine.id}`)).toEqual(notFound);
	expect(await ana('GET', `/v1/kids/k-nobody/symptoms`)).toEqual(notFound);
	expect(await ana('GET', ivy)).toMatchObject({
		status: 200,
		body: { symptoms: [mine], next_cursor: null }
	});

	const timeline = (q = '') => ana('GET', `/v1/kids/${IVY.id}/timeline${q}`);
	expect((await timeline('?types=symptom')).body.items).toEqual([{ type: 'symptom', ...mine }]);
	expect((await timeline()).body.items.map((i: { type: string }) => i.type)).toEqual(['symptom']);
	expect((await timeline('?types=symptom,symptom')).body).toEqual({
		error: 'invalid',
		field: 'types'
	});
	expect((await ana('PUT', `/v1/kids/${IVY.id}/modules/sickness`, { enabled: false })).status).toBe(
		204
	);
	expect((await timeline()).body.items).toEqual([]);
	expect((await timeline('?types=symptom')).body.items).toHaveLength(1);
	expect((await ana('DELETE', `${ivy}/${mine.id}`)).status).toBe(204);
	expect(await ana('GET', `${ivy}/${mine.id}`)).toEqual(notFound);
});

test('medications: validation order and the type errors decodeJSON answers', async () => {
	const ana = await stubAs(ANA);
	const post = (body: unknown) => ana('POST', `/v1/kids/${IVY.id}/medications`, body);
	const base = { name: 'Calpol', dose: '2.5 ml' };
	expect(await post({})).toMatchObject(invalid('name'));
	expect(await post({ name: '   ', dose: 'x' })).toMatchObject(invalid('name'));
	expect(await post({ name: 'é'.repeat(81) })).toMatchObject(invalid('name'));
	expect(await post({ name: 'Calpol', schedule: { kind: 'bogus' } })).toMatchObject(
		invalid('dose')
	);
	expect(await post({ ...base, dose: 'd'.repeat(81) })).toMatchObject(invalid('dose'));
	expect(await post(base)).toMatchObject(invalid('schedule'));
	for (const schedule of [
		{ kind: 'bogus' },
		{ kind: 'times' },
		{ kind: 'times', times: [] },
		{ kind: 'times', times: ['24:00'] },
		{ kind: 'times', times: ['7:05'] },
		{ kind: 'times', times: ['07:5'] },
		{ kind: 'times', times: ['08:00', '08:00'] },
		{
			kind: 'times',
			times: Array.from({ length: 13 }, (_, i) => `${String(i + 1).padStart(2, '0')}:00`)
		},
		{ kind: 'times', times: ['08:00'], hours: 6 },
		{ kind: 'every', hours: 0 },
		{ kind: 'every', hours: 169 },
		{ kind: 'every' },
		{ kind: 'every', hours: 6, times: ['08:00'] },
		{ kind: 'none', hours: 1 },
		{ kind: 'none', times: ['08:00'] }
	]) {
		expect(await post({ ...base, schedule, start_at: '1900-01-01T00:00:00Z' })).toMatchObject(
			invalid('schedule')
		);
	}
	const none = { ...base, schedule: { kind: 'none' } };
	expect(await post({ ...none, start_at: '2000-01-01T00:00:00Z', end_on: 'bad' })).toMatchObject(
		invalid('start_at')
	);
	expect(await post({ ...none, end_on: '2026-02-30', note: 'x'.repeat(501) })).toMatchObject(
		invalid('end_on')
	);
	expect(
		await post({ ...none, start_at: iso(NOON), end_on: iso(NOON - 86400).slice(0, 10) })
	).toMatchObject(invalid('end_on'));
	expect(await post({ ...none, note: 'x'.repeat(501) })).toMatchObject(invalid('note'));
	expect((await post({ ...none, name: 5 })).status).toBe(400);
	expect((await post({ ...base, schedule: { kind: 'every', hours: '6' } })).status).toBe(400);
	expect((await post({ ...base, schedule: { kind: 'every', hours: 6.5 } })).status).toBe(400);
	expect((await post({ ...none, active: 'yes' })).status).toBe(400);
	expect((await post({ ...none, end_on: 5 })).status).toBe(400);
	expect(await post({ ...base, schedule: { kind: 'none', every: 1 } })).toMatchObject({
		status: 422,
		body: { error: 'unknown_field', field: 'every' }
	});
});

test('medications: the view, trimmed text, canonical schedules, the list, the cap and the 404s', async () => {
	const ana = await stubAs(ANA);
	const dan = await stubAs(DAN);
	const path = `/v1/kids/${IVY.id}/medications`;
	const none = await ana('POST', path, {
		name: '  Calpol ',
		dose: ' 2.5 ml ',
		schedule: { kind: 'none', times: null, hours: null },
		start_at: iso(NOON - 2 * 86400)
	});
	expect(none.status).toBe(201);
	expect(Object.keys(none.body).sort()).toEqual(
		[
			'active',
			'dose',
			'dose_count',
			'end_on',
			'id',
			'last_dose',
			'name',
			'next_due',
			'note',
			'schedule',
			'start_at'
		].sort()
	);
	expect(none.body).toMatchObject({
		name: 'Calpol',
		dose: '2.5 ml',
		schedule: { kind: 'none' },
		start_at: iso(NOON - 2 * 86400),
		end_on: null,
		active: true,
		note: null,
		last_dose: null,
		next_due: null,
		dose_count: 0
	});
	expect(Object.keys(none.body.schedule)).toEqual(['kind']);
	const every = (
		await ana('POST', path, {
			name: 'Amoxicillin',
			dose: '5 ml',
			schedule: { kind: 'every', hours: 8 },
			start_at: iso(NOON)
		})
	).body;
	expect(every.schedule).toEqual({ kind: 'every', hours: 8 });
	expect(every.next_due).toEqual({ due_at: iso(NOON) });
	const times = (
		await ana('POST', path, {
			name: 'Vitamin D',
			dose: '1 drop',
			schedule: { kind: 'times', times: ['20:00', '08:00'] },
			start_at: iso(NOON - 86400)
		})
	).body;
	expect(times.schedule).toEqual({ kind: 'times', times: ['08:00', '20:00'] });

	const list = await ana('GET', path);
	expect(list.status).toBe(200);
	expect(Object.keys(list.body)).toEqual(['medications']);
	expect(list.body.medications.map((m: { id: string }) => m.id)).toEqual([
		every.id,
		times.id,
		none.body.id
	]);

	expect(await ana('PATCH', `${path}/${times.id}`, { note: 'with food' })).toMatchObject({
		status: 200,
		body: { ...times, note: 'with food' }
	});
	expect(await ana('PATCH', `${path}/${times.id}`, { schedule: { kind: 'every' } })).toMatchObject(
		invalid('schedule')
	);
	const off = await ana('PATCH', `${path}/${every.id}`, { active: false });
	expect(off).toMatchObject({ status: 200, body: { active: false, next_due: null } });

	const leo = (
		await ana('POST', `/v1/kids/${LEO.id}/medications`, {
			...{ name: 'X', dose: 'y' },
			schedule: { kind: 'none' }
		})
	).body;
	for (const id of [leo.id, 'e-000000009999', 'not an id!']) {
		for (const method of ['GET', 'PATCH', 'DELETE']) {
			expect(
				await ana(method, `${path}/${encodeURIComponent(id)}`, method === 'PATCH' ? {} : undefined)
			).toEqual(notFound);
		}
	}
	expect(await dan('GET', path)).toEqual(notFound);
	expect(await dan('POST', path, { name: 'X', dose: 'y', schedule: { kind: 'none' } })).toEqual(
		notFound
	);

	for (let i = 3; i < 50; i++) {
		expect(
			(await ana('POST', path, { name: `M${i}`, dose: 'd', schedule: { kind: 'none' } })).status
		).toBe(201);
	}
	expect(await ana('POST', path, { name: 'M50', dose: 'd', schedule: { kind: 'none' } })).toEqual({
		status: 409,
		body: { error: 'limit' }
	});
});

test('doses: validation order, the medication rules, the list filter and the cascade', async () => {
	const ana = await stubAs(ANA);
	const meds = `/v1/kids/${IVY.id}/medications`;
	const course = (
		await ana('POST', meds, { name: 'Calpol', dose: '2.5 ml', schedule: { kind: 'none' } })
	).body;
	const second = (
		await ana('POST', meds, { name: 'Ibu', dose: '5 ml', schedule: { kind: 'none' } })
	).body;
	const leo = (
		await ana('POST', `/v1/kids/${LEO.id}/medications`, {
			name: 'X',
			dose: 'y',
			schedule: { kind: 'none' }
		})
	).body;
	const path = `/v1/kids/${IVY.id}/medication-doses`;
	const post = (body: unknown) => ana('POST', path, body);
	expect(await post({})).toMatchObject(invalid('medication'));
	expect(await post({ given_at: iso(realNow() + 6 * M), amount: ' ' })).toMatchObject(
		invalid('given_at')
	);
	expect(await post({ given_at: '1990-01-01T00:00:00Z' })).toMatchObject(invalid('given_at'));
	expect(await post({ amount: '  ', note: 'x'.repeat(501) })).toMatchObject(invalid('amount'));
	expect(await post({ amount: 'a'.repeat(81) })).toMatchObject(invalid('amount'));
	expect(await post({ note: 'x'.repeat(501), medication: 'nope' })).toMatchObject(invalid('note'));
	for (const medication of ['nope', 'not an id!', leo.id, '']) {
		expect(await post({ medication })).toMatchObject(invalid('medication'));
	}
	expect((await post({ medication: 5 })).status).toBe(400);

	const dose = await post({ medication: course.id, amount: ' half ' });
	expect(dose.status).toBe(201);
	expect(Object.keys(dose.body).sort()).toEqual(['amount', 'given_at', 'id', 'medication', 'note']);
	expect(dose.body).toMatchObject({ medication: course.id, amount: 'half', note: null });
	const at = `${path}/${dose.body.id}`;
	expect(await ana('PATCH', at, { medication: course.id, note: 'ok' })).toMatchObject({
		status: 200,
		body: { ...dose.body, note: 'ok' }
	});
	expect(await ana('PATCH', at, { medication: null, amount: null })).toMatchObject({
		status: 200,
		body: { medication: course.id, amount: null, note: 'ok' }
	});
	expect(await ana('PATCH', at, { medication: second.id })).toMatchObject(invalid('medication'));
	expect(await ana('PATCH', at, { medication: second.id, amount: ' ' })).toMatchObject(
		invalid('amount')
	);
	const other = (await post({ medication: second.id })).body;

	const list = (q = '') => ana('GET', `${path}${q}`);
	expect((await list()).body).toEqual({
		doses: [other, (await ana('GET', at)).body],
		next_cursor: null
	});
	expect(
		(await list(`?medication=${course.id}`)).body.doses.map((d: { id: string }) => d.id)
	).toEqual([dose.body.id]);
	expect((await list('?medication=e-000000009999')).body).toEqual({ doses: [], next_cursor: null });
	expect(await list('?medication=not%20an%20id')).toMatchObject(invalid('medication'));
	expect(await list('?medication=')).toMatchObject(invalid('medication'));
	expect(await list('?from=bad&medication=bad!')).toMatchObject(invalid('from'));

	const leoDose = (await ana('POST', `/v1/kids/${LEO.id}/medication-doses`, { medication: leo.id }))
		.body;
	for (const id of [leoDose.id, 'e-000000009999', 'not an id!']) {
		for (const method of ['GET', 'PATCH', 'DELETE']) {
			expect(
				await ana(method, `${path}/${encodeURIComponent(id)}`, method === 'PATCH' ? {} : undefined)
			).toEqual(notFound);
		}
	}

	const read = (await ana('GET', `${meds}/${course.id}`)).body;
	expect(read).toMatchObject({
		dose_count: 1,
		last_dose: { id: dose.body.id, given_at: dose.body.given_at, amount: null }
	});
	expect((await ana('DELETE', `${meds}/${course.id}`)).status).toBe(204);
	expect(await ana('GET', at)).toEqual(notFound);
	expect((await list()).body.doses).toEqual([other]);
});

test('next_due: the first not-done occurrence after now, and a dose marks it done', async () => {
	const ana = await stubAs(ANA);
	await clockAt(NOON - 2 * H);
	const course = (
		await ana('POST', `/v1/kids/${IVY.id}/medications`, {
			name: 'Vitamin D',
			dose: '1 drop',
			schedule: { kind: 'times', times: ['20:00', '08:00'] },
			start_at: iso(NOON - 12 * H)
		})
	).body;
	expect(course.next_due).toEqual({ due_at: iso(NOON + 8 * H) });
	const read = async () => (await ana('GET', `/v1/kids/${IVY.id}/medications/${course.id}`)).body;
	await clockAt(NOON + 7 * H + 40 * M);
	expect(
		(await ana('POST', `/v1/kids/${IVY.id}/medication-doses`, { medication: course.id })).status
	).toBe(201);
	expect((await read()).next_due).toEqual({ due_at: iso(NOON + 20 * H) });
	const home = (await ana('GET', `/v1/kids/${IVY.id}/home`)).body;
	expect(home.due).toEqual([]);

	const every = (
		await ana('POST', `/v1/kids/${IVY.id}/medications`, {
			name: 'Amoxicillin',
			dose: '5 ml',
			schedule: { kind: 'every', hours: 6 },
			start_at: iso(NOON + 7 * H + 20 * M)
		})
	).body;
	expect(every.next_due).toEqual({ due_at: iso(NOON + 13 * H + 20 * M) });
	const later = await ana('PATCH', `/v1/kids/${IVY.id}/medications/${every.id}`, {
		schedule: { kind: 'every', hours: 4 }
	});
	expect(later.body.next_due).toEqual({ due_at: iso(NOON + 11 * H + 20 * M) });
});

test('the Due list: medication entries first, the overdue window at 5, 60 and 1440 minutes', async () => {
	const ana = await stubAs(ANA);
	const home = async () => (await ana('GET', `/v1/kids/${IVY.id}/home`)).body.due;
	const minutes = (n: number) =>
		ana('PATCH', `/v1/kids/${IVY.id}`, { reminder_overdue_minutes: n });
	await clockAt(NOON - 10 * M);
	const daily = (
		await ana('POST', `/v1/kids/${IVY.id}/medications`, {
			name: 'Vitamin D',
			dose: '1 drop',
			schedule: { kind: 'every', hours: 24 },
			start_at: iso(NOON)
		})
	).body;
	const entry = (state: string) => ({
		kind: 'medication',
		medication: { id: daily.id, name: 'Vitamin D' },
		due_at: iso(NOON),
		state
	});
	expect(await home()).toEqual([entry('due')]);
	expect(Object.keys((await home())[0])).toEqual(['kind', 'medication', 'due_at', 'state']);

	await clockAt(NOON + 59 * M);
	expect(await home()).toEqual([entry('overdue')]);
	await clockAt(NOON + 60 * M);
	expect(await home()).toEqual([]);

	expect((await minutes(5)).status).toBe(200);
	await clockAt(NOON + 4 * M);
	expect(await home()).toEqual([entry('overdue')]);
	await clockAt(NOON + 5 * M);
	expect(await home()).toEqual([]);

	expect((await minutes(1440)).status).toBe(200);
	await clockAt(NOON + 23 * H);
	expect(await home()).toEqual([entry('overdue'), { ...entry('due'), due_at: iso(NOON + 24 * H) }]);
	// The next occurrence comes before due + 1440 minutes would end the window.
	const hourly = (
		await ana('POST', `/v1/kids/${IVY.id}/medications`, {
			name: 'Saline',
			dose: '2 sprays',
			schedule: { kind: 'every', hours: 1 },
			start_at: iso(NOON + 23 * H - 10 * M)
		})
	).body;
	await clockAt(NOON + 25 * H);
	const due = await home();
	const hourlyEntries = due.filter(
		(e: { medication: { id: string } }) => e.medication.id === hourly.id
	);
	expect(hourlyEntries).toEqual([
		{
			kind: 'medication',
			medication: { id: hourly.id, name: 'Saline' },
			due_at: iso(NOON + 24 * H + 50 * M),
			state: 'overdue'
		},
		{
			kind: 'medication',
			medication: { id: hourly.id, name: 'Saline' },
			due_at: iso(NOON + 25 * H + 50 * M),
			state: 'due'
		}
	]);
	// The daily course's noon occurrence today is overdue; yesterday's gave way to it.
	expect(due.filter((e: { medication: { id: string } }) => e.medication.id === daily.id)).toEqual([
		{ ...entry('overdue'), due_at: iso(NOON + 24 * H) }
	]);
	expect(due.map((e: { due_at: string }) => e.due_at)).toEqual(
		[...due.map((e: { due_at: string }) => e.due_at)].sort()
	);
});

test('the Due list: gates for the module, inactive and done courses, other kids and vaccines', async () => {
	const ana = await stubAs(ANA);
	const home = async (kid: string = IVY.id) => (await ana('GET', `/v1/kids/${kid}/home`)).body.due;
	expect((await ana('PATCH', `/v1/kids/${IVY.id}`, { vaccine_schema: 'us' })).status).toBe(200);
	await clockAt(NOON - 10 * M);
	const vaccines = await home();
	expect(vaccines.length).toBeGreaterThan(0);
	for (const v of vaccines)
		expect(Object.keys(v).sort()).toEqual(['count', 'from', 'kind', 'status']);
	const course = (
		await ana('POST', `/v1/kids/${IVY.id}/medications`, {
			name: 'Calpol',
			dose: '2.5 ml',
			schedule: { kind: 'every', hours: 24 },
			start_at: iso(NOON)
		})
	).body;
	const leo = (
		await ana('POST', `/v1/kids/${LEO.id}/medications`, {
			name: 'Leo only',
			dose: 'x',
			schedule: { kind: 'every', hours: 24 },
			start_at: iso(NOON)
		})
	).body;
	const mine = {
		kind: 'medication',
		medication: { id: course.id, name: 'Calpol' },
		due_at: iso(NOON),
		state: 'due'
	};
	expect(await home()).toEqual([mine, ...vaccines]);
	expect(
		(await home(LEO.id)).map((e: { medication?: { id: string } }) => e.medication?.id)
	).toEqual([leo.id]);

	const module = (enabled: boolean) =>
		ana('PUT', `/v1/kids/${IVY.id}/modules/medication`, { enabled });
	expect((await module(false)).status).toBe(204);
	expect(await home()).toEqual(vaccines);
	expect((await module(true)).status).toBe(204);
	expect(await home()).toEqual([mine, ...vaccines]);

	const path = `/v1/kids/${IVY.id}/medications/${course.id}`;
	expect((await ana('PATCH', path, { active: false })).status).toBe(200);
	expect(await home()).toEqual(vaccines);
	expect((await ana('PATCH', path, { active: true })).status).toBe(200);
	expect(await home()).toEqual([mine, ...vaccines]);

	await clockAt(NOON + 10 * M);
	expect(await home()).toEqual([{ ...mine, state: 'overdue' }, ...vaccines]);
	expect(
		(await ana('POST', `/v1/kids/${IVY.id}/medication-doses`, { medication: course.id })).status
	).toBe(201);
	expect(await home()).toEqual(vaccines);
	expect((await ana('GET', path)).body.next_due).toEqual({ due_at: iso(NOON + 24 * H) });
	expect((await ana('GET', `/v1/kids/${MIA.id}/home`)).body.due).toEqual([]);
});

test('a done row of an old schedule does not take a later dose', async () => {
	const ana = await stubAs(ANA);
	const meds = `/v1/kids/${IVY.id}/medications`;
	await clockAt(NOON + 5 * H + 40 * M);
	const course = (
		await ana('POST', meds, {
			name: 'Ibuprofen',
			dose: '5 ml',
			schedule: { kind: 'every', hours: 6 },
			start_at: iso(NOON - 12 * H)
		})
	).body;
	const dose = () => ana('POST', `/v1/kids/${IVY.id}/medication-doses`, { medication: course.id });
	await clockAt(NOON + 5 * H + 45 * M);
	expect((await dose()).status).toBe(201);
	await clockAt(NOON + 5 * H + 50 * M);
	const edit = { schedule: { kind: 'times', times: ['17:55'] } };
	expect((await ana('PATCH', `${meds}/${course.id}`, edit)).status).toBe(200);
	await clockAt(NOON + 5 * H + 58 * M);
	expect((await dose()).status).toBe(201);
	expect((await ana('GET', `${meds}/${course.id}`)).body.next_due).toEqual({
		due_at: iso(NOON + 29 * H + 55 * M)
	});
	expect((await ana('GET', `/v1/kids/${IVY.id}/home`)).body.due).toEqual([]);
});

test('the Due list takes a course latest past row in any state', async () => {
	const ana = await stubAs(ANA);
	const meds = `/v1/kids/${IVY.id}/medications`;
	const home = async () => (await ana('GET', `/v1/kids/${IVY.id}/home`)).body.due;
	await clockAt(NOON - 2 * H);
	const course = (
		await ana('POST', meds, {
			name: 'Calpol',
			dose: '2.5 ml',
			schedule: { kind: 'times', times: ['12:00', '13:00'] },
			start_at: iso(NOON - 12 * H)
		})
	).body;
	expect((await ana('PATCH', `/v1/kids/${IVY.id}`, { reminder_overdue_minutes: 120 })).status).toBe(
		200
	);
	await clockAt(NOON + 10 * M);
	await home();
	await clockAt(NOON + 31 * M);
	await home();
	await clockAt(NOON + H + 5 * M);
	expect(
		(await ana('POST', `/v1/kids/${IVY.id}/medication-doses`, { medication: course.id })).status
	).toBe(201);
	await clockAt(NOON + H + 10 * M);
	const edit = { schedule: { kind: 'every', hours: 6 } };
	expect((await ana('PATCH', `${meds}/${course.id}`, edit)).status).toBe(200);
	await clockAt(NOON + H + 30 * M);
	// The expired 12:00 row is older than the done 13:00 one, so nothing is overdue.
	expect(await home()).toEqual([
		{
			kind: 'medication',
			medication: { id: course.id, name: 'Calpol' },
			due_at: iso(NOON + 6 * H),
			state: 'due'
		}
	]);
});

test('no row for an occurrence over 30 minutes before the course exists; a null time; a fractional clock', async () => {
	const ana = await stubAs(ANA);
	const meds = `/v1/kids/${IVY.id}/medications`;
	await clockAt(NOON + 45 * M);
	const course = (
		await ana('POST', meds, {
			name: 'Calpol',
			dose: '2.5 ml',
			schedule: { kind: 'every', hours: 24 },
			start_at: iso(NOON - 24 * H)
		})
	).body;
	expect((await ana('GET', `/v1/kids/${IVY.id}/home`)).body.due).toEqual([]);
	expect(course.next_due).toEqual({ due_at: iso(NOON + 24 * H) });
	expect(
		await ana('POST', meds, { name: 'x', dose: 'y', schedule: { kind: 'times', times: [null] } })
	).toMatchObject(invalid('schedule'));
	const clock = await fetch(`${STUB_URL}/__clock`, {
		method: 'POST',
		body: JSON.stringify({ offset_seconds: 1.5 })
	});
	expect(clock.status).toBe(400);
});
