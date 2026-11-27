import { describe, expect, it } from 'vitest';
import { deviceLabel } from './device-label';

const UA = {
	safariIphone:
		'Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Mobile/15E148 Safari/604.1',
	chromeAndroid:
		'Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Mobile Safari/537.36',
	firefoxWindows:
		'Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:127.0) Gecko/20100101 Firefox/127.0',
	edgeWindows:
		'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36 Edg/126.0.0.0',
	safariMac:
		'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Safari/605.1.15',
	chromeLinux:
		'Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36',
	chromeIos:
		'Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/126.0.0.0 Mobile/15E148 Safari/604.1'
};

describe('deviceLabel', () => {
	it('names the browser and the system from a closed set', () => {
		expect(deviceLabel(UA.safariIphone)).toBe('Safari on iPhone');
		expect(deviceLabel(UA.chromeAndroid)).toBe('Chrome on Android');
		expect(deviceLabel(UA.firefoxWindows)).toBe('Firefox on Windows');
		expect(deviceLabel(UA.edgeWindows)).toBe('Edge on Windows');
		expect(deviceLabel(UA.safariMac)).toBe('Safari on Mac');
		expect(deviceLabel(UA.chromeLinux)).toBe('Chrome on Linux');
		expect(deviceLabel(UA.chromeIos)).toBe('Chrome on iPhone');
	});
	it('falls back to words of the set, never the text it was given', () => {
		expect(deviceLabel('')).toBe('Browser');
		expect(deviceLabel('curl/8.0 <script>alert(1)</script>')).toBe('Browser');
		expect(deviceLabel('SomeBot/1.0 (Windows NT)')).toBe('Browser on Windows');
	});
	it('always fits the API pattern', () => {
		for (const ua of [...Object.values(UA), '', 'x'.repeat(5000), 'curl/8.0 ünï']) {
			expect(deviceLabel(ua)).toMatch(/^[A-Za-z0-9 ]{1,48}$/);
		}
	});
});
