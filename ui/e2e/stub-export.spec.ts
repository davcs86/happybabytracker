import { expect, test } from '@playwright/test';
import {
	ADMIN,
	ANA,
	DAN,
	IVY,
	STUB_TOKEN,
	STUB_URL,
	moveStubClock,
	openSession,
	resetStub,
	setScenario,
	stubAs
} from './stub-api.js';

// The stub's export route answers as httpapi/export.go does; its PDF is a marked stand-in listing the modules.

test.beforeEach(async () => {
	await resetStub();
});

const MS_PER_DAY = 86_400_000;
const utcToday = () => new Date().toISOString().slice(0, 10);
const shiftDays = (date: string, days: number) =>
	new Date(Date.parse(`${date}T00:00:00Z`) + days * MS_PER_DAY).toISOString().slice(0, 10);
const path = (kid: string) => `/v1/kids/${kid}/export`;

async function post(who: { username: string; password: string }, kid: string, body: string) {
	const token = await openSession(who.username, who.password);
	const res = await fetch(`${STUB_URL}${path(kid)}`, {
		method: 'POST',
		headers: {
			authorization: `Bearer ${STUB_TOKEN}`,
			'x-session-token': token,
			'content-type': 'application/json'
		},
		body
	});
	return { status: res.status, headers: res.headers, text: await res.text() };
}
const asAna = (body: unknown, kid: string = IVY.id) => post(ANA, kid, JSON.stringify(body));
const valid = (extra: Record<string, unknown> = {}) => ({
	from: shiftDays(utcToday(), -6),
	to: utcToday(),
	modules: ['growth', 'bottles'],
	...extra
});
const invalid = (field: string) => JSON.stringify({ error: 'invalid', field });

test('200: a marked stub PDF listing the modules as sent, with the API headers', async () => {
	const body = valid();
	const r = await asAna(body);
	expect(r.status).toBe(200);
	expect(r.headers.get('content-type')).toBe('application/pdf');
	expect(r.headers.get('content-disposition')).toBe(
		`attachment; filename="happybabytracker-${body.from}-${body.to}.pdf"`
	);
	expect(r.headers.get('cache-control')).toBe('no-store');
	expect(r.headers.get('x-content-type-options')).toBe('nosniff');
	expect(r.text.startsWith('%PDF-1.')).toBe(true);
	expect(r.text.endsWith('%%EOF\n')).toBe(true);
	expect(r.text).toContain('STUB EXPORT, NOT A REAL DOCUMENT. Modules: growth,bottles)');
	expect(r.text.length).toBeLessThan(1000);
	expect([...r.text].every((c) => c.charCodeAt(0) < 0x80)).toBe(true);
	expect((await asAna(body)).text).toBe(r.text);
});

test('decodeJSON first: an unknown field is 422 unknown_field even with a bad from; a non-string from is 400', async () => {
	const r = await asAna({ ...valid({ from: 'x' }), extra: 1 });
	expect([r.status, r.text]).toEqual([
		422,
		JSON.stringify({ error: 'unknown_field', field: 'extra' })
	]);
	const bad = await asAna(valid({ from: 20261101 }));
	expect([bad.status, bad.text]).toEqual([400, JSON.stringify({ error: 'bad_request' })]);
	const broken = await post(ANA, IVY.id, '{');
	expect(broken.status).toBe(400);
});

test('from: a malformed date, a missing one or one before the birth date', async () => {
	for (const from of ['2026-02-30', '', null, shiftDays(IVY.birthDate, -1)]) {
		const r = await asAna(valid({ from, modules: 'bogus' }));
		expect([r.status, r.text]).toEqual([422, invalid('from')]);
	}
	expect((await asAna(valid({ from: IVY.birthDate, to: IVY.birthDate }))).status).toBe(200);
});

test('to: before from, after the kid-local today, malformed, or a 94-day span; 93 days pass', async () => {
	const today = utcToday();
	for (const range of [
		{ from: today, to: shiftDays(today, -1) },
		{ from: today, to: shiftDays(today, 1) },
		{ from: today, to: 'today' },
		{ from: shiftDays(today, -93), to: today }
	]) {
		const r = await asAna(valid({ ...range, modules: [] }));
		expect([r.status, r.text]).toEqual([422, invalid('to')]);
	}
	expect((await asAna(valid({ from: shiftDays(today, -92), to: today }))).status).toBe(200);
});

test("today is the stub clock's", async () => {
	const tomorrow = shiftDays(utcToday(), 1);
	expect((await asAna(valid({ to: tomorrow }))).status).toBe(422);
	await moveStubClock(24 * 3600);
	expect((await asAna(valid({ to: tomorrow }))).status).toBe(200);
});

test('modules: every bad shape is 422 modules; a switched-off module is exported', async () => {
	for (const modules of [
		'bottles',
		7,
		null,
		undefined,
		[],
		[1],
		['bogus'],
		['Bottles'],
		['../x'],
		['bottles', 'bottles']
	]) {
		const r = await asAna(valid({ modules }));
		expect([r.status, r.text]).toEqual([422, invalid('modules')]);
	}
	const ana = await stubAs(ANA);
	expect((await ana('PUT', `/v1/kids/${IVY.id}/modules/pumping`, { enabled: false })).status).toBe(
		204
	);
	const r = await asAna(valid({ modules: ['pumping', 'sickness', 'medication'] }));
	expect(r.status).toBe(200);
	expect(r.text).toContain('Modules: pumping,sickness,medication)');
});

test('404 for no grant and an unknown kid, the same body; 403 for the admin', async () => {
	const none = await post(DAN, IVY.id, JSON.stringify(valid()));
	const unknown = await asAna(valid(), 'k-unknown');
	expect([none.status, none.text]).toEqual([404, JSON.stringify({ error: 'not_found' })]);
	expect([unknown.status, unknown.text]).toEqual([none.status, none.text]);
	const admin = await post(ADMIN, IVY.id, JSON.stringify(valid()));
	expect([admin.status, admin.text]).toEqual([403, JSON.stringify({ error: 'forbidden' })]);
});

test('export_busy: a valid export is 503 busy, after the validation', async () => {
	await setScenario('export_busy');
	const r = await asAna(valid());
	expect([r.status, r.text]).toEqual([503, JSON.stringify({ error: 'busy' })]);
	expect((await asAna(valid({ modules: [] }))).status).toBe(422);
});
