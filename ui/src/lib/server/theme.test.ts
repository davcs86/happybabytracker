import { describe, expect, it } from 'vitest';
import { parseTheme, returnPath, themeOrAuto } from './theme';

describe('parseTheme', () => {
	it.each(['light', 'dark', 'auto'])('accepts %s', (t) => {
		expect(parseTheme(t)).toBe(t);
	});

	it.each([null, undefined, '', 'Dark', 'sepia', 3, ['dark']])('refuses %j', (t) => {
		expect(parseTheme(t)).toBeNull();
	});

	it('defaults to auto', () => {
		expect(themeOrAuto('nope')).toBe('auto');
		expect(themeOrAuto('dark')).toBe('dark');
	});
});

describe('returnPath', () => {
	it.each(['/', '/kids/abc/timeline', '/kids/abc/growth?measure=weight', '/account#devices'])(
		'keeps %s',
		(p) => {
			expect(returnPath(p)).toBe(p);
		}
	);

	it.each([
		'//evil.example',
		'/\\evil.example',
		'https://evil.example/',
		'evil',
		'/a b',
		'/a\nb',
		'',
		null,
		undefined,
		['/x']
	])('falls back to / for %j', (p) => {
		expect(returnPath(p)).toBe('/');
	});
});
