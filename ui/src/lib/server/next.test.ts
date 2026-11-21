import { describe, expect, it } from 'vitest';
import { safeNext } from './next';

describe('safeNext', () => {
	it.each([
		['/', '/'],
		['/account', '/account'],
		['/kids/abc?tab=x#top', '/kids/abc?tab=x#top'],
		['/a/../b', '/b']
	])('keeps the same-origin path %j', (raw, want) => {
		expect(safeNext(raw)).toBe(want);
	});

	it.each([
		null,
		'',
		'account',
		'//evil.example',
		'///evil.example',
		'/\\evil.example',
		'\\/evil.example',
		'/\t/evil.example',
		'/\n/evil.example',
		'https://evil.example',
		'http://127.0.0.1:4173/',
		'javascript:alert(1)',
		' /account'
	])('refuses %j', (raw) => {
		expect(safeNext(raw)).toBeNull();
	});
});

describe('landing', () => {
	it('sends a parent to a safe next or home', async () => {
		const { landing } = await import('./landing');
		expect(landing('parent', '/account')).toBe('/account');
		expect(landing('parent', '//evil.example')).toBe('/');
		expect(landing('parent', null)).toBe('/');
	});

	it('keeps the admin inside /admin/', async () => {
		const { landing } = await import('./landing');
		expect(landing('admin', '/admin/parents?x=1')).toBe('/admin/parents?x=1');
		expect(landing('admin', '/account')).toBe('/admin/parents');
		expect(landing('admin', 'https://evil.example/admin/')).toBe('/admin/parents');
		expect(landing('admin', null)).toBe('/admin/parents');
	});
});
