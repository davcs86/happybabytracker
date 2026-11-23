import { expect, test } from '@playwright/test';
import {
	ADMIN,
	ANA,
	CLEO,
	DAN,
	IVY,
	LEO,
	MIA,
	fakePushKeys,
	receivedRequests,
	resetStub,
	setScenario,
	stubAs
} from './stub-api.js';

// The stub's Web Push device and ntfy routes answer as the Go handlers do, with each channel on or off.

test.beforeEach(async () => {
	await resetStub();
});

const invalid = (field: string) => ({ status: 422, body: { error: 'invalid', field } });
const notFound = { status: 404, body: { error: 'not_found' } };
const conflict = (error: string) => ({ status: 409, body: { error } });
const endpoint = (n: string) => `https://push.example/send/${n}`;
const upsert = (n: string, label = 'Phone') => ({
	endpoint: endpoint(n),
	keys: fakePushKeys(),
	label
});
const DEVICES = '/v1/me/push-devices';
const TOPIC = /^hbt-[0-9A-Za-z]{22}$/;

test('GET /v1/push reads the scenario: on by default, off for push_off and channels_off', async () => {
	const ana = await stubAs(ANA);
	const res = await ana('GET', '/v1/push');
	expect(res.status).toBe(200);
	expect(res.body).toEqual({
		enabled: true,
		public_key: expect.stringMatching(/^[A-Za-z0-9_-]{87}$/)
	});
	for (const [scenario, enabled] of [
		['ntfy_off', true],
		['push_off', false],
		['channels_off', false],
		['normal', true]
	] as const) {
		await setScenario(scenario);
		expect((await ana('GET', '/v1/push')).body.enabled).toBe(enabled);
	}
});

test('push devices answer a parent only: no session 401, admin 403', async () => {
	const admin = await stubAs(ADMIN);
	for (const [method, path] of [
		['GET', '/v1/push'],
		['GET', DEVICES],
		['PUT', DEVICES],
		['GET', '/v1/me/ntfy'],
		['PUT', '/v1/me/ntfy']
	] as const) {
		expect((await admin(method, path, method === 'PUT' ? {} : undefined)).status).toBe(403);
	}
	const res = await fetch(`http://127.0.0.1:4181${DEVICES}`, {
		headers: { authorization: 'Bearer fake-internal-api-token-000000000000' }
	});
	expect(res.status).toBe(401);
});

test('upsert creates (201), a re-send keeps the id and updates the label (200), and lists', async () => {
	const ana = await stubAs(ANA);
	const first = await ana('PUT', DEVICES, upsert('a', 'Ana phone'));
	expect(first.status).toBe(201);
	expect(first.body.device).toEqual({
		id: expect.any(String),
		label: 'Ana phone',
		created_at: expect.stringMatching(/Z$/),
		kids: []
	});
	const again = await ana('PUT', DEVICES, { ...upsert('a'), label: '  Kitchen  ' });
	expect(again.status).toBe(200);
	expect(again.body.device).toMatchObject({ id: first.body.device.id, label: 'Kitchen' });
	// Scheme, host case, a trailing dot and :443 name the same browser.
	const same = await ana('PUT', DEVICES, {
		...upsert('a', 'Same'),
		endpoint: 'HTTPS://Push.Example.:443/send/a'
	});
	expect(same.status).toBe(200);
	expect((await ana('GET', DEVICES)).body).toEqual({
		devices: [{ ...first.body.device, label: 'Same' }]
	});
});

test('no response and no logged path carries the endpoint or the keys', async () => {
	const ana = await stubAs(ANA);
	const body = upsert('secret-endpoint', 'Phone');
	const device = (await ana('PUT', DEVICES, body)).body.device;
	await ana('PATCH', `${DEVICES}/${device.id}`, { label: 'Renamed' });
	const text = JSON.stringify([(await ana('GET', DEVICES)).body, device]);
	for (const secret of ['secret-endpoint', body.keys.p256dh, body.keys.auth]) {
		expect(text).not.toContain(secret);
	}
	for (const r of await receivedRequests()) {
		expect(r.path).not.toContain('secret-endpoint');
		expect(r.path).not.toContain(body.keys.auth);
	}
});

test('upsert checks the body, the endpoint, the keys, then the label', async () => {
	const ana = await stubAs(ANA);
	const ok = upsert('v');
	const put = (body: unknown) => ana('PUT', DEVICES, body);
	expect((await put([])).status).toBe(400);
	expect(await put({ ...ok, extra: 1 })).toMatchObject({
		status: 422,
		body: { error: 'unknown_field' }
	});
	expect(await put({ ...ok, keys: { ...ok.keys, other: 'x' } })).toMatchObject({
		status: 422,
		body: { error: 'unknown_field', field: 'other' }
	});
	expect((await put({ ...ok, endpoint: 5 })).status).toBe(400);
	expect(await put({})).toMatchObject(invalid('endpoint'));
	expect(await put({ ...ok, endpoint: null })).toMatchObject(invalid('endpoint'));
	for (const bad of [
		'http://push.example/x',
		'https://user@push.example/x',
		'https://push.example:8443/x',
		'https://push.example:/x',
		'https://127.0.0.1/x',
		'https://[::1]/x',
		'https://localhost/x',
		'https://a.localhost/x',
		'https://2130706433/x',
		'https://0x7f.1/x',
		'push.example/x',
		`https://push.example/${'a'.repeat(2048)}`
	]) {
		expect(await put({ ...ok, endpoint: bad, keys: null, label: '' })).toMatchObject(
			invalid('endpoint')
		);
	}
	const none = { ...ok, label: '' };
	expect(await put({ endpoint: ok.endpoint, label: '' })).toMatchObject(invalid('keys'));
	expect(await put({ ...none, keys: {} })).toMatchObject(invalid('keys'));
	expect(await put({ ...none, keys: { p256dh: ok.keys.p256dh } })).toMatchObject(invalid('keys'));
	expect(await put({ ...none, keys: { p256dh: ok.keys.p256dh, auth: null } })).toMatchObject(
		invalid('keys')
	);
	const pt = Buffer.from(ok.keys.p256dh, 'base64url');
	pt[40] ^= 1;
	for (const keys of [
		{ ...ok.keys, p256dh: 'AAAA' },
		{ ...ok.keys, p256dh: pt.toString('base64url') },
		{ ...ok.keys, p256dh: '!!!' },
		{ ...ok.keys, auth: 'AAAA' },
		{ ...ok.keys, auth: Buffer.alloc(17).toString('base64url') }
	]) {
		expect(await put({ ...none, keys })).toMatchObject(invalid('keys'));
	}
	for (const label of ['', '   ', 'x'.repeat(41), '\u{1F4F1}'.repeat(41)]) {
		expect(await put({ ...ok, label })).toMatchObject(invalid('label'));
	}
	expect(await put({ endpoint: ok.endpoint, keys: ok.keys })).toMatchObject(invalid('label'));
	expect((await put({ ...ok, label: 'x'.repeat(40) })).status).toBe(201);
	// Keys in the standard alphabet, padded, are accepted as the API does.
	const std = (key: string) => Buffer.from(key, 'base64url').toString('base64');
	const keys = fakePushKeys();
	expect(
		(await put({ ...upsert('w'), keys: { p256dh: std(keys.p256dh), auth: std(keys.auth) } })).status
	).toBe(201);
});

test('ten devices per parent: the eleventh is 409 limit, a re-send of one still works', async () => {
	const ana = await stubAs(ANA);
	for (let i = 0; i < 10; i++)
		expect((await ana('PUT', DEVICES, upsert(`d${i}`))).status).toBe(201);
	expect(await ana('PUT', DEVICES, upsert('d10'))).toMatchObject(conflict('limit'));
	expect((await ana('PUT', DEVICES, upsert('d3', 'Again'))).status).toBe(200);
	const dan = await stubAs(DAN);
	expect((await dan('PUT', DEVICES, upsert('d10'))).status).toBe(201);
});

test("another parent's registration moves the device to a new id and drops the old opt-ins", async () => {
	const ana = await stubAs(ANA);
	const cleo = await stubAs(CLEO);
	const mine = (await ana('PUT', DEVICES, upsert('shared', 'Ana'))).body.device;
	expect((await ana('PUT', `/v1/kids/${LEO.id}/push-devices/${mine.id}`)).status).toBe(204);
	expect((await ana('GET', DEVICES)).body.devices[0].kids).toEqual([LEO.id]);

	const moved = await cleo('PUT', DEVICES, { ...upsert('shared', 'Cleo') });
	expect(moved.status).toBe(201);
	expect(moved.body.device.id).not.toBe(mine.id);
	expect(moved.body.device).toMatchObject({ label: 'Cleo', kids: [] });
	expect((await ana('GET', DEVICES)).body).toEqual({ devices: [] });
	expect(await ana('DELETE', `${DEVICES}/${mine.id}`)).toMatchObject(notFound);

	// Moving back is a new registration again, and Cleo's opt-ins go with the old row.
	await cleo('PUT', `/v1/kids/${LEO.id}/push-devices/${moved.body.device.id}`);
	const back = await ana('PUT', DEVICES, upsert('shared', 'Ana again'));
	expect(back.status).toBe(201);
	expect(back.body.device.kids).toEqual([]);
	expect((await cleo('GET', DEVICES)).body).toEqual({ devices: [] });
});

test('rename checks the label before ownership; delete and rename of another parent device are 404', async () => {
	const ana = await stubAs(ANA);
	const dan = await stubAs(DAN);
	const device = (await ana('PUT', DEVICES, upsert('r', 'Old'))).body.device;
	const path = `${DEVICES}/${device.id}`;
	expect((await ana('PATCH', path, {})).status).toBe(422);
	expect(await ana('PATCH', path, { label: '' })).toMatchObject(invalid('label'));
	expect(await ana('PATCH', path, { label: 'x'.repeat(41) })).toMatchObject(invalid('label'));
	expect(await ana('PATCH', path, { other: 1 })).toMatchObject({
		status: 422,
		body: { error: 'unknown_field' }
	});
	expect((await ana('PATCH', path, { label: ' New ' })).body.device.label).toBe('New');
	// Another parent's device and a missing one answer alike, bad label or not.
	expect(await dan('PATCH', path, { label: 'x' })).toMatchObject(notFound);
	expect(await dan('PATCH', `${DEVICES}/nope`, { label: 'x' })).toMatchObject(notFound);
	expect(await dan('PATCH', path, { label: '' })).toMatchObject(invalid('label'));
	expect(await dan('DELETE', path)).toEqual(await dan('DELETE', `${DEVICES}/nope`));
	expect((await ana('DELETE', path)).status).toBe(204);
	expect(await ana('DELETE', path)).toMatchObject(notFound);
});

test('opt-ins need a grant and an own device; both failures answer the unknown-kid 404', async () => {
	const ana = await stubAs(ANA);
	const dan = await stubAs(DAN);
	const device = (await ana('PUT', DEVICES, upsert('o'))).body.device;
	const optin = (kid: string, id: string) => `/v1/kids/${kid}/push-devices/${id}`;
	const unknown = await ana('PUT', optin('nope', device.id));
	expect(unknown).toMatchObject(notFound);
	expect(await dan('PUT', optin(MIA.id, device.id))).toEqual(unknown);
	expect(await ana('PUT', optin(MIA.id, 'nope'))).toEqual(unknown);
	expect(await dan('PUT', optin(MIA.id, 'nope'))).toEqual(unknown);
	expect(await dan('DELETE', optin(MIA.id, device.id))).toEqual(unknown);

	expect((await ana('PUT', optin(MIA.id, device.id))).status).toBe(204);
	expect((await ana('PUT', optin(MIA.id, device.id))).status).toBe(204);
	expect((await ana('PUT', optin(IVY.id, device.id))).status).toBe(204);
	expect((await ana('GET', DEVICES)).body.devices[0].kids).toEqual([IVY.id, MIA.id]);
	expect((await ana('DELETE', optin(MIA.id, device.id))).status).toBe(204);
	expect((await ana('DELETE', optin(MIA.id, device.id))).status).toBe(204);
	expect((await ana('GET', DEVICES)).body.devices[0].kids).toEqual([IVY.id]);
});

test('revoking or leaving a kid deletes that parent opt-ins for it, and a new grant starts empty', async () => {
	const ana = await stubAs(ANA);
	const cleo = await stubAs(CLEO);
	const device = (await ana('PUT', DEVICES, upsert('g'))).body.device;
	await ana('PUT', `/v1/kids/${LEO.id}/push-devices/${device.id}`);
	await ana('PUT', `/v1/kids/${MIA.id}/push-devices/${device.id}`);
	await ana('PUT', '/v1/me/ntfy');
	await ana('PUT', `/v1/kids/${LEO.id}/ntfy`);
	expect((await cleo('DELETE', `/v1/kids/${LEO.id}/access/p-ana`)).status).toBe(204);
	expect((await ana('GET', DEVICES)).body.devices[0].kids).toEqual([MIA.id]);
	expect((await ana('GET', '/v1/me/ntfy')).body.kids).toEqual([]);
	await cleo('PUT', `/v1/kids/${LEO.id}/access/${ANA.username}`, { role: 'caregiver' });
	expect((await ana('GET', DEVICES)).body.devices[0].kids).toEqual([MIA.id]);
	expect((await ana('GET', '/v1/me/ntfy')).body.kids).toEqual([]);
	// The topic survives the revoke.
	expect((await ana('GET', '/v1/me/ntfy')).body.topic).toMatch(TOPIC);
});

test('Web Push off: the writes answer 409 push_unavailable, list, deletes and opt-out keep working', async () => {
	const ana = await stubAs(ANA);
	const device = (await ana('PUT', DEVICES, upsert('off'))).body.device;
	await ana('PUT', `/v1/kids/${MIA.id}/push-devices/${device.id}`);
	for (const scenario of ['push_off', 'channels_off'] as const) {
		await setScenario(scenario);
		expect((await ana('GET', '/v1/push')).body).toEqual({ enabled: false });
		// The 409 comes before the body is read.
		expect(await ana('PUT', DEVICES, {})).toMatchObject(conflict('push_unavailable'));
		expect(await ana('PUT', DEVICES, upsert('new'))).toMatchObject(conflict('push_unavailable'));
		expect(await ana('PATCH', `${DEVICES}/${device.id}`, { label: 'x' })).toMatchObject(
			conflict('push_unavailable')
		);
		expect(await ana('PATCH', `${DEVICES}/nope`, { label: '' })).toMatchObject(
			conflict('push_unavailable')
		);
		expect(await ana('PUT', `/v1/kids/${IVY.id}/push-devices/${device.id}`)).toMatchObject(
			conflict('push_unavailable')
		);
		// A kid without a grant is still 404, as the access check runs first.
		expect(await ana('PUT', `/v1/kids/nope/push-devices/${device.id}`)).toMatchObject(notFound);
		expect((await ana('GET', DEVICES)).body.devices).toHaveLength(1);
	}
	expect((await ana('DELETE', `/v1/kids/${MIA.id}/push-devices/${device.id}`)).status).toBe(204);
	expect((await ana('DELETE', `${DEVICES}/${device.id}`)).status).toBe(204);
	expect((await ana('GET', DEVICES)).body).toEqual({ devices: [] });
});

test('ntfy on: turn on makes a topic, a second turn on keeps it, regenerate replaces it and keeps the opt-ins', async () => {
	const ana = await stubAs(ANA);
	expect(await ana('GET', '/v1/me/ntfy')).toEqual({
		status: 200,
		body: { enabled: true, server: expect.stringMatching(/^https:\/\//), topic: null, kids: [] }
	});
	expect(await ana('POST', '/v1/me/ntfy/regenerate')).toMatchObject(notFound);
	expect(await ana('DELETE', '/v1/me/ntfy')).toMatchObject(notFound);
	expect(await ana('PUT', `/v1/kids/${MIA.id}/ntfy`)).toMatchObject(notFound);

	const on = await ana('PUT', '/v1/me/ntfy');
	expect(on.status).toBe(201);
	expect(on.body.topic).toMatch(TOPIC);
	const kept = await ana('PUT', '/v1/me/ntfy');
	expect(kept).toMatchObject({ status: 200, body: { topic: on.body.topic } });

	expect((await ana('PUT', `/v1/kids/${MIA.id}/ntfy`)).status).toBe(204);
	expect((await ana('PUT', `/v1/kids/${MIA.id}/ntfy`)).status).toBe(204);
	expect((await ana('PUT', `/v1/kids/${IVY.id}/ntfy`)).status).toBe(204);
	expect((await ana('GET', '/v1/me/ntfy')).body.kids).toEqual([IVY.id, MIA.id]);
	const fresh = await ana('POST', '/v1/me/ntfy/regenerate');
	expect(fresh.status).toBe(200);
	expect(fresh.body.topic).toMatch(TOPIC);
	expect(fresh.body.topic).not.toBe(on.body.topic);
	expect(fresh.body.kids).toEqual([IVY.id, MIA.id]);
	expect((await ana('DELETE', `/v1/kids/${MIA.id}/ntfy`)).status).toBe(204);
	expect((await ana('GET', '/v1/me/ntfy')).body.kids).toEqual([IVY.id]);
	expect((await ana('PUT', '/v1/kids/nope/ntfy')).status).toBe(404);

	expect((await ana('DELETE', '/v1/me/ntfy')).status).toBe(204);
	expect((await ana('GET', '/v1/me/ntfy')).body.topic).toBeNull();
	expect((await ana('GET', '/v1/me/ntfy')).body.kids).toEqual([]);
});

test('ntfy off: GET is {enabled:false} even with a topic, the writes are 409 ntfy_unavailable', async () => {
	const ana = await stubAs(ANA);
	await ana('PUT', '/v1/me/ntfy');
	await ana('PUT', `/v1/kids/${MIA.id}/ntfy`);
	for (const scenario of ['ntfy_off', 'channels_off'] as const) {
		await setScenario(scenario);
		expect(await ana('GET', '/v1/me/ntfy')).toEqual({ status: 200, body: { enabled: false } });
		expect(await ana('PUT', '/v1/me/ntfy')).toMatchObject(conflict('ntfy_unavailable'));
		expect(await ana('POST', '/v1/me/ntfy/regenerate')).toMatchObject(conflict('ntfy_unavailable'));
		expect(await ana('PUT', `/v1/kids/${IVY.id}/ntfy`)).toMatchObject(conflict('ntfy_unavailable'));
		expect(await ana('PUT', '/v1/kids/nope/ntfy')).toMatchObject(notFound);
	}
	expect((await ana('DELETE', `/v1/kids/${MIA.id}/ntfy`)).status).toBe(204);
	expect((await ana('DELETE', '/v1/me/ntfy')).status).toBe(204);
	await setScenario('normal');
	expect((await ana('GET', '/v1/me/ntfy')).body.topic).toBeNull();
});

test('ntfy off leaves Web Push on and the other way round', async () => {
	const ana = await stubAs(ANA);
	await setScenario('ntfy_off');
	expect((await ana('PUT', DEVICES, upsert('x'))).status).toBe(201);
	await setScenario('push_off');
	expect((await ana('PUT', '/v1/me/ntfy')).status).toBe(201);
});

test('disabling a parent deletes their devices, topic and opt-ins', async () => {
	const admin = await stubAs(ADMIN);
	const dan = await stubAs(DAN);
	await dan('PUT', DEVICES, upsert('dan'));
	await dan('PUT', '/v1/me/ntfy');
	expect((await admin('POST', '/v1/admin/parents/p-dan/disable')).status).toBe(204);
	expect((await admin('POST', '/v1/admin/parents/p-dan/enable')).status).toBe(204);
	const again = await stubAs(DAN);
	expect((await again('GET', DEVICES)).body).toEqual({ devices: [] });
	expect((await again('GET', '/v1/me/ntfy')).body.topic).toBeNull();
});

test('the request log covers both groups', async () => {
	const ana = await stubAs(ANA);
	const body = upsert('log');
	const device = (await ana('PUT', DEVICES, body)).body.device;
	await ana('GET', '/v1/push');
	await ana('PUT', '/v1/me/ntfy');
	await ana('PUT', `/v1/kids/${MIA.id}/push-devices/${device.id}`);
	const log = (await receivedRequests()).map((r) => `${r.method} ${r.path}`);
	expect(log).toEqual(
		expect.arrayContaining([
			`PUT ${DEVICES}`,
			'GET /v1/push',
			'PUT /v1/me/ntfy',
			`PUT /v1/kids/${MIA.id}/push-devices/${device.id}`
		])
	);
	const put = (await receivedRequests()).find((r) => r.method === 'PUT' && r.path === DEVICES);
	expect(JSON.parse(put?.body ?? '{}')).toEqual(body);
});
