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
			port: 9212
		});
	});

	it.each(['https://x.test/', 'https://x.test:443', 'https://x.test/some/path'])(
		'stores the normalised origin for ORIGIN = %s',
		(origin) => {
			expect(loadEnv({ ...valid, ORIGIN: origin }).origin).toBe('https://x.test');
		}
	);
});
