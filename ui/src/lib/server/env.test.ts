import { describe, expect, it } from 'vitest';
import { loadEnv } from './env';

const valid = {
	INTERNAL_API_TOKEN: 't'.repeat(32),
	API_BASE_URL: 'http://happybabytracker-api:9210',
	ORIGIN: 'https://hbt.example.test',
	PORT: '9212'
};

describe('loadEnv', () => {
	it.each([
		['INTERNAL_API_TOKEN', 'x'.repeat(31)],
		['INTERNAL_API_TOKEN', '\u{1F600}'.repeat(31)],
		['API_BASE_URL', 'not a url'],
		['API_BASE_URL', 'api:9210'],
		['API_BASE_URL', 'ftp://api:9210'],
		['ORIGIN', ''],
		['ORIGIN', 'hbt.example.test'],
		['ORIGIN', 'javascript:alert(1)'],
		['PORT', ''],
		['PORT', '0'],
		['PORT', '65536'],
		['PORT', '92a'],
		['PORT', '-1']
	])('rejects %s = %j', (key, value) => {
		expect(() => loadEnv({ ...valid, [key]: value })).toThrow(key);
	});

	it.each(Object.keys(valid))('rejects a missing %s', (key) => {
		expect(() => loadEnv({ ...valid, [key]: undefined })).toThrow(key);
	});

	it.each([
		['INTERNAL_API_TOKEN', 'z'.repeat(31)],
		['API_BASE_URL', 'bad value one'],
		['ORIGIN', 'bad value two'],
		['PORT', 'bad value three']
	])('never puts the value of %s in the error', (key, value) => {
		let message = '';
		try {
			loadEnv({ ...valid, [key]: value });
		} catch (e) {
			message = (e as Error).message;
		}
		expect(message).toContain(key);
		expect(message).not.toContain(value);
	});

	it('counts the token in code points', () => {
		expect(loadEnv({ ...valid, INTERNAL_API_TOKEN: '\u{1F600}'.repeat(32) }).internalApiToken).toBe(
			'\u{1F600}'.repeat(32)
		);
	});

	it('accepts a valid environment', () => {
		expect(loadEnv(valid)).toEqual({
			internalApiToken: valid.INTERNAL_API_TOKEN,
			apiBaseUrl: valid.API_BASE_URL,
			origin: valid.ORIGIN,
			port: 9212,
			oidc: null
		});
	});

	it.each(['https://x.test/', 'https://x.test:443', 'https://x.test/some/path'])(
		'stores the normalised origin for ORIGIN = %s',
		(origin) => {
			expect(loadEnv({ ...valid, ORIGIN: origin }).origin).toBe('https://x.test');
		}
	);
});

const rp = {
	OIDC_ISSUER: 'https://auth.example.com',
	OIDC_CLIENT_ID: 'hbt-client',
	OIDC_CLIENT_SECRET: 'fake-client-secret-value',
	SESSION_SEAL_KEY: 's'.repeat(32)
};

function refusal(source: Record<string, string | undefined>): string[] {
	try {
		loadEnv(source);
	} catch (e) {
		return (e as Error).message.replace('invalid environment: ', '').split(', ');
	}
	return [];
}

describe('loadEnv, relying party', () => {
	it('gives oidc null with no OIDC variable', () => {
		expect(loadEnv(valid).oidc).toBeNull();
	});

	it('gives the default scopes with OIDC_SCOPES absent', () => {
		expect(loadEnv({ ...valid, ...rp }).oidc).toEqual({
			issuer: rp.OIDC_ISSUER,
			clientId: rp.OIDC_CLIENT_ID,
			clientSecret: rp.OIDC_CLIENT_SECRET,
			scopes: 'openid profile',
			sealKey: rp.SESSION_SEAL_KEY
		});
	});

	it('keeps OIDC_SCOPES when given', () => {
		expect(loadEnv({ ...valid, ...rp, OIDC_SCOPES: 'openid profile email' }).oidc?.scopes).toBe(
			'openid profile email'
		);
	});

	it.each([
		[{ OIDC_ISSUER: rp.OIDC_ISSUER }, ['OIDC_CLIENT_ID', 'OIDC_CLIENT_SECRET']],
		[{ OIDC_CLIENT_ID: 'c' }, ['OIDC_ISSUER', 'OIDC_CLIENT_SECRET']],
		[{ OIDC_CLIENT_SECRET: 's' }, ['OIDC_ISSUER', 'OIDC_CLIENT_ID']],
		[{ OIDC_ISSUER: rp.OIDC_ISSUER, OIDC_CLIENT_ID: 'c' }, ['OIDC_CLIENT_SECRET']],
		[{ OIDC_CLIENT_ID: 'c', OIDC_CLIENT_SECRET: 's' }, ['OIDC_ISSUER']],
		[{ OIDC_ISSUER: '' }, ['OIDC_CLIENT_ID', 'OIDC_CLIENT_SECRET']],
		[{ OIDC_ISSUER: '', OIDC_CLIENT_ID: '' }, ['OIDC_CLIENT_SECRET']]
	])('refuses a partial set %j, naming the absent ones', (partial, names) => {
		const seal = 'OIDC_ISSUER' in partial ? { SESSION_SEAL_KEY: rp.SESSION_SEAL_KEY } : {};
		expect(refusal({ ...valid, ...seal, ...partial })).toEqual(names);
	});

	it('refuses OIDC_ISSUER without a seal key', () => {
		expect(refusal({ ...valid, ...rp, SESSION_SEAL_KEY: undefined })).toEqual(['SESSION_SEAL_KEY']);
		expect(refusal({ ...valid, OIDC_ISSUER: rp.OIDC_ISSUER })).toEqual([
			'OIDC_CLIENT_ID',
			'OIDC_CLIENT_SECRET',
			'SESSION_SEAL_KEY'
		]);
	});

	it.each(['', 'k'.repeat(31), '\u{1F600}'.repeat(31)])(
		'refuses SESSION_SEAL_KEY = %j with the three present',
		(key) => {
			expect(refusal({ ...valid, ...rp, SESSION_SEAL_KEY: key })).toEqual(['SESSION_SEAL_KEY']);
		}
	);

	it('counts the seal key in code points', () => {
		const key = '\u{1F600}'.repeat(32);
		expect(loadEnv({ ...valid, ...rp, SESSION_SEAL_KEY: key }).oidc?.sealKey).toBe(key);
	});

	it('refuses OIDC_SCOPES = ""', () => {
		expect(refusal({ ...valid, ...rp, OIDC_SCOPES: '' })).toEqual(['OIDC_SCOPES']);
	});

	it.each(['', 'false', 'true'])('refuses OIDC_AUTO_PROVISION = %j', (value) => {
		expect(refusal({ ...valid, ...rp, OIDC_AUTO_PROVISION: value })).toEqual([
			'OIDC_AUTO_PROVISION'
		]);
		expect(refusal({ ...valid, OIDC_AUTO_PROVISION: value })).toEqual(['OIDC_AUTO_PROVISION']);
	});

	it.each([
		[{ SESSION_SEAL_KEY: 's'.repeat(32) }, ['SESSION_SEAL_KEY']],
		[{ SESSION_SEAL_KEY: '' }, ['SESSION_SEAL_KEY']],
		[{ OIDC_SCOPES: 'openid profile' }, ['OIDC_SCOPES']],
		[{ OIDC_SCOPES: '' }, ['OIDC_SCOPES']],
		[
			{ SESSION_SEAL_KEY: 's'.repeat(32), OIDC_SCOPES: 'openid profile' },
			['SESSION_SEAL_KEY', 'OIDC_SCOPES']
		]
	])('refuses a stray %j with none of the three', (stray, names) => {
		expect(refusal({ ...valid, ...stray })).toEqual(names);
	});

	it.each(['', 'http://127.0.0.1:4182', 'auth.example.com', 'ftp://auth.example.com'])(
		'refuses OIDC_ISSUER = %j, naming OIDC_ISSUER only',
		(issuer) => {
			expect(refusal({ ...valid, ...rp, OIDC_ISSUER: issuer })).toEqual(['OIDC_ISSUER']);
		}
	);

	it.each([
		'https://auth.example.com/.well-known/openid-configuration',
		'https://auth.example.com/tenant/.well-known/',
		'https://auth.example.com/.well-known\\openid-configuration'
	])('refuses a /.well-known/ OIDC_ISSUER = %j, naming OIDC_ISSUER only', (issuer) => {
		expect(refusal({ ...valid, ...rp, OIDC_ISSUER: issuer })).toEqual(['OIDC_ISSUER']);
	});

	it.each(['https://auth.example.com', 'https://127.0.0.1:4182'])(
		'accepts OIDC_ISSUER = %j',
		(issuer) => {
			expect(loadEnv({ ...valid, ...rp, OIDC_ISSUER: issuer }).oidc?.issuer).toBe(issuer);
		}
	);

	it.each([
		[{ ...rp, SESSION_SEAL_KEY: 'short-seal-value' }],
		[{ ...rp, OIDC_ISSUER: 'http://issuer-value.test' }],
		[{ ...rp, OIDC_AUTO_PROVISION: 'provision-value' }],
		[{ OIDC_CLIENT_ID: 'client-id-value', OIDC_CLIENT_SECRET: 'client-secret-value' }],
		[{ OIDC_SCOPES: 'scopes-value', SESSION_SEAL_KEY: 'stray-seal-value' }]
	])('never puts a value in the error: %j', (vars) => {
		let message = '';
		try {
			loadEnv({ ...valid, ...vars });
		} catch (e) {
			message = (e as Error).message;
		}
		expect(message).toMatch(/^invalid environment: /);
		for (const value of Object.values(vars)) expect(message).not.toContain(value);
	});
});
