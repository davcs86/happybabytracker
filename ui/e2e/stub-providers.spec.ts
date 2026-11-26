import { expect, test } from '@playwright/test';
import {
	ADMIN,
	ANA,
	DAN,
	STUB_TOKEN,
	STUB_URL,
	resetStub,
	setScenario,
	stubAs
} from './stub-api.js';

// The stub's provider routes answer as httpapi/providers.go does: off by default, the route then unknown; the
// same validation order, limit and failures; the practice-location filter of ruling 1.

test.beforeEach(async () => {
	await resetStub();
});

const invalid = (field: string) => ({ status: 422, body: { error: 'invalid', field } });
const SEARCH = '/v1/providers/search';
const person = { kind: 'person', last_name: 'Quill', state: 'WA' };

test('off by default: the state says so and the search is the unknown-route 404', async () => {
	const ana = await stubAs(ANA);
	expect(await ana('GET', '/v1/providers')).toEqual({
		status: 200,
		body: { search: { enabled: false } }
	});
	const unknown = await ana('POST', '/v1/no-such-route', person);
	expect(unknown.status).toBe(404);
	expect(await ana('POST', SEARCH, person)).toEqual(unknown);
});

test('on: a parent route; an admin session is refused', async () => {
	await setScenario('npi_on');
	const ana = await stubAs(ANA);
	expect(await ana('GET', '/v1/providers')).toEqual({
		status: 200,
		body: { search: { enabled: true } }
	});
	const admin = await stubAs(ADMIN);
	expect((await admin('GET', '/v1/providers')).status).toBe(403);
	expect((await admin('POST', SEARCH, person)).status).toBe(403);
	const res = await fetch(`${STUB_URL}${SEARCH}`, {
		method: 'POST',
		headers: { authorization: `Bearer ${STUB_TOKEN}`, 'content-type': 'application/json' },
		body: JSON.stringify(person)
	});
	expect(res.status).toBe(401);
});

test('validation: the first invalid field in body order', async () => {
	await setScenario('npi_on');
	const dan = await stubAs(DAN);
	const search = (body: unknown) => dan('POST', SEARCH, body);
	expect(await search({ state: 'WA' })).toEqual(invalid('kind'));
	expect(await search({ kind: 'doctor', state: 'XX' })).toEqual(invalid('kind'));
	expect(await search({ kind: 'person', state: 'XX' })).toEqual(invalid('last_name'));
	for (const last_name of ['J', 'ab$', 'Qu*', ' ', 'x'.repeat(61), '9ill']) {
		expect(await search({ ...person, last_name })).toEqual(invalid('last_name'));
	}
	expect(await search({ ...person, organization: 'X Clinic' })).toEqual(invalid('organization'));
	expect(await search({ ...person, first_name: 'A' })).toEqual(invalid('first_name'));
	expect(await search({ kind: 'organization', first_name: 'Av', state: 'WA' })).toEqual(
		invalid('first_name')
	);
	expect(await search({ kind: 'organization', state: 'WA' })).toEqual(invalid('organization'));
	expect(await search({ kind: 'organization', organization: 'Q*', state: 'WA' })).toEqual(
		invalid('organization')
	);
	expect(await search({ ...person, state: 'XX', city: 'S' })).toEqual(invalid('state'));
	expect(await search({ ...person, city: 'S', postal_code: '1' })).toEqual(invalid('city'));
	for (const postal_code of ['9810', '981011', '9810a', ' 98101']) {
		expect(await search({ ...person, postal_code })).toEqual(invalid('postal_code'));
	}
	expect(await search({ ...person, bogus: 'x' })).toEqual({
		status: 422,
		body: { error: 'unknown_field', field: 'bogus' }
	});
	expect((await search({ ...person, state: ' wa ', last_name: ' Quill ' })).status).toBe(200);
});

test('results: practice location in the searched state only, registry text as sent, more on a full page', async () => {
	await setScenario('npi_on');
	const ana = await stubAs(ANA);
	const search = async (body: unknown) => (await ana('POST', SEARCH, body)).body;
	const wa = await search(person);
	expect(wa).toEqual({
		results: [
			{
				npi: '9000000015',
				name: 'AVERY QUILL',
				credential: 'MD',
				taxonomy: 'Pediatrics',
				address: '12 PINE ST, SUITE 4, SEATTLE, WA 98101',
				phone: '206-555-0100'
			},
			{
				npi: '9000000023',
				name: 'BLAIR QUILLAN',
				credential: null,
				taxonomy: 'Family Medicine',
				address: '300 CEDAR AVE, TACOMA, WA 98402',
				phone: null
			}
		],
		more: false
	});
	// CASEY QUILLBY gets mail in WA but practises in OR.
	expect(
		(await search({ ...person, state: 'OR' })).results.map((r: { name: string }) => r.name)
	).toEqual(['CASEY QUILLBY']);
	expect((await search({ ...person, city: 'tacoma' })).results).toHaveLength(1);
	expect((await search({ ...person, postal_code: '98101' })).results).toHaveLength(1);
	expect(await search({ ...person, first_name: 'Zed' })).toEqual({ results: [], more: false });
	const clinic = await search({ kind: 'organization', organization: 'quill', state: 'OR' });
	expect(clinic.results).toEqual([
		expect.objectContaining({ name: 'QUILL & SONS CLINIC 2', credential: null })
	]);
	const larks = await search({ kind: 'person', last_name: 'Lark', state: 'WA' });
	expect(larks.results).toHaveLength(20);
	expect(larks.more).toBe(true);
	for (const r of [...wa.results, ...larks.results]) expect(r.npi).toMatch(/^9\d{9}$/);
});

test('failures: rate limited after validation, unavailable after both', async () => {
	await setScenario('npi_rate_limited');
	const ana = await stubAs(ANA);
	expect(await ana('POST', SEARCH, { kind: 'x' })).toEqual(invalid('kind'));
	expect(await ana('POST', SEARCH, person)).toEqual({
		status: 429,
		body: { error: 'rate_limited' }
	});
	expect((await ana('GET', '/v1/providers')).body).toEqual({ search: { enabled: true } });
	await setScenario('npi_unavailable');
	expect(await ana('POST', SEARCH, { ...person, state: 'XX' })).toEqual(invalid('state'));
	expect(await ana('POST', SEARCH, person)).toEqual({
		status: 503,
		body: { error: 'unavailable' }
	});
});
