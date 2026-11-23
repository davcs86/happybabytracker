import { describe, expect, it } from 'vitest';
import { base64UrlToBytes, isIosOutsideStandalone, parsePayload, subscriptionBody } from './push';

describe('base64UrlToBytes', () => {
	it('decodes the unpadded url alphabet', () => {
		// 0xfb 0xff 0xfe is "+//+" in the standard alphabet.
		expect([...base64UrlToBytes('-__-')]).toEqual([0xfb, 0xff, 0xfe]);
	});

	it.each([
		['', []],
		['AQ', [1]],
		['AQI', [1, 2]],
		['AQID', [1, 2, 3]],
		['AQIDBA', [1, 2, 3, 4]]
	])('pads %j to a multiple of four', (text, bytes) => {
		expect([...base64UrlToBytes(text)]).toEqual(bytes);
	});

	it('accepts text that is already padded', () => {
		expect([...base64UrlToBytes('AQ==')]).toEqual([1]);
	});

	it('decodes a 65 byte public key to 65 bytes', () => {
		const bytes = Uint8Array.from({ length: 65 }, (_, i) => i * 3);
		const text = Buffer.from(bytes).toString('base64url');
		expect(text).toHaveLength(87);
		expect([...base64UrlToBytes(text)]).toEqual([...bytes]);
	});

	it.each([['A'], ['AQIDB'], ['a b'], ['a+b/'], ['!!!!']])('throws on %j', (text) => {
		expect(() => base64UrlToBytes(text)).toThrow();
	});
});

describe('subscriptionBody', () => {
	const sub = (json: object) => ({ toJSON: () => json });

	it('keeps the endpoint and both keys and drops everything else', () => {
		expect(
			subscriptionBody(
				sub({
					endpoint: 'https://push.example/abc',
					expirationTime: 5,
					keys: { p256dh: 'P', auth: 'A', extra: 'x' }
				})
			)
		).toEqual({ endpoint: 'https://push.example/abc', keys: { p256dh: 'P', auth: 'A' } });
	});

	it.each([
		['no endpoint', { keys: { p256dh: 'P', auth: 'A' } }],
		['an empty endpoint', { endpoint: '', keys: { p256dh: 'P', auth: 'A' } }],
		['no keys', { endpoint: 'https://push.example/abc' }],
		['no p256dh', { endpoint: 'https://push.example/abc', keys: { auth: 'A' } }],
		['no auth', { endpoint: 'https://push.example/abc', keys: { p256dh: 'P' } }],
		['an empty auth', { endpoint: 'https://push.example/abc', keys: { p256dh: 'P', auth: '' } }],
		['a numeric endpoint', { endpoint: 5, keys: { p256dh: 'P', auth: 'A' } }]
	])('is null for %s', (_name, json) => {
		expect(subscriptionBody(sub(json))).toBeNull();
	});
});

describe('parsePayload', () => {
	const ok = { title: 'Medication due for Mia', path: '/kids/k-1/medication', tag: 'reminder-r1' };

	it('returns title, path and tag and ignores the other fields', () => {
		expect(parsePayload({ ...ok, v: 1, category: 'medication', kid: 'k-1' })).toEqual(ok);
	});

	it.each([
		['the root', '/'],
		['a query', '/kids/k-1/medication?x=1'],
		['a fragment', '/kids/k-1#top']
	])('accepts %s', (_name, path) => {
		expect(parsePayload({ ...ok, path })).toEqual({ ...ok, path });
	});

	it.each([
		['a protocol-relative url', '//evil.example'],
		['a protocol-relative url with a path', '//evil.example/x'],
		['an absolute url', 'https://x'],
		['a relative path', 'kids/k-1'],
		['an empty path', ''],
		['a backslash after the slash', '/\\evil.example'],
		['a backslash later', '/kids\\..\\x'],
		['a tab that the URL parser drops', '/\t/evil.example'],
		['a newline', '/kids\n/x'],
		['a null byte', '/kids/\u0000']
	])('rejects %s', (_name, path) => {
		expect(parsePayload({ ...ok, path })).toBeNull();
	});

	it.each([
		['no title', { path: ok.path, tag: ok.tag }],
		['no path', { title: ok.title, tag: ok.tag }],
		['no tag', { title: ok.title, path: ok.path }],
		['an empty title', { ...ok, title: '' }],
		['an empty tag', { ...ok, tag: '' }],
		['a numeric title', { ...ok, title: 5 }],
		['a numeric path', { ...ok, path: 5 }],
		['a null tag', { ...ok, tag: null }]
	])('rejects %s', (_name, data) => {
		expect(parsePayload(data)).toBeNull();
	});

	it.each([[null], [undefined], ['text'], [5], [[]], [['/x']]])('rejects %j', (data) => {
		expect(parsePayload(data)).toBeNull();
	});
});

describe('isIosOutsideStandalone', () => {
	const IPHONE =
		'Mozilla/5.0 (iPhone; CPU iPhone OS 17_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4 Mobile/15E148 Safari/604.1';
	const IPAD =
		'Mozilla/5.0 (iPad; CPU OS 16_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.6 Mobile/15E148 Safari/604.1';
	// iPadOS 13+ answers with a Mac user agent; only the touch points tell it from a Mac.
	const MAC =
		'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4 Safari/605.1.15';
	const ANDROID =
		'Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Mobile Safari/537.36';
	const WINDOWS =
		'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36';

	it.each([
		['an iPhone', IPHONE],
		['an iPad', IPAD]
	])('is true for %s in a browser tab', (_name, ua) => {
		expect(isIosOutsideStandalone(ua, false)).toBe(true);
	});

	it.each([
		['an iPhone', IPHONE],
		['an iPad', IPAD],
		['an iPadOS Mac user agent', MAC]
	])('is false for %s as a home screen app', (_name, ua) => {
		expect(isIosOutsideStandalone(ua, true, 5)).toBe(false);
	});

	it('is true for an iPadOS user agent with touch points in a browser tab', () => {
		expect(isIosOutsideStandalone(MAC, false, 5)).toBe(true);
		expect(isIosOutsideStandalone(MAC, false, 2)).toBe(true);
	});

	it('is false for a Mac without a touch screen', () => {
		expect(isIosOutsideStandalone(MAC, false)).toBe(false);
		expect(isIosOutsideStandalone(MAC, false, 0)).toBe(false);
		expect(isIosOutsideStandalone(MAC, false, 1)).toBe(false);
	});

	it.each([
		['Android', ANDROID],
		['Windows', WINDOWS],
		['an empty string', '']
	])('is false for %s', (_name, ua) => {
		expect(isIosOutsideStandalone(ua, false, 5)).toBe(false);
	});
});
