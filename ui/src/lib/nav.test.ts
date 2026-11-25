import { describe, expect, it } from 'vitest';
import { navModel, type NavInput } from './nav';
import type { ModuleState } from './types';

const base: NavInput = {
	pathname: '/',
	kidId: 'k1',
	role: 'guardian',
	modules: null,
	theme: 'auto'
};
const ids = (i: Partial<NavInput>) => navModel({ ...base, ...i }).more.map((e) => e.id);
const off = (module: ModuleState['module']): ModuleState => ({
	module,
	enabled: false,
	default: true,
	overridden: true
});

describe('navModel', () => {
	it('lists the guardian entries in the fixed order, Sign out last', () => {
		expect(ids({})).toEqual([
			'vaccines',
			'medication',
			'milestones',
			'export',
			'modules',
			'profile',
			'settings',
			'account',
			'theme',
			'logout'
		]);
	});

	it('gives a caregiver no guardian entries', () => {
		expect(ids({ role: 'caregiver' })).toEqual([
			'vaccines',
			'medication',
			'milestones',
			'export',
			'account',
			'theme',
			'logout'
		]);
	});

	it('keeps only the child-free entries when there is no child', () => {
		expect(ids({ kidId: null, role: null })).toEqual(['account', 'theme', 'logout']);
	});

	it('hides the entries of a module that is switched off, and nothing when unknown', () => {
		expect(ids({ modules: [off('medication'), off('vaccination')] })).not.toContain('medication');
		expect(ids({ modules: [off('vaccination')] })).not.toContain('vaccines');
		expect(ids({ modules: [off('sleep')] })).toEqual(ids({}));
		expect(ids({ modules: null })).toContain('medication');
	});

	it('links every entry under the child', () => {
		const hrefs = navModel(base).more.flatMap((e) => (e.kind === 'link' ? [e.href] : []));
		expect(hrefs).toEqual([
			'/kids/k1/vaccines',
			'/kids/k1/medication',
			'/kids/k1/milestones',
			'/kids/k1/export',
			'/kids/k1/modules',
			'/kids/k1/profile',
			'/kids/k1/settings',
			'/account'
		]);
	});

	it('names the theme and the sign-out scope in the notes', () => {
		const more = navModel({ ...base, theme: 'dark' }).more;
		expect(more.find((e) => e.id === 'theme')?.note).toBe('Dark');
		expect(more.find((e) => e.id === 'logout')?.note).toBe('This device');
	});
});

describe('current page', () => {
	const current = (pathname: string) => {
		const m = navModel({ ...base, pathname });
		return [...m.primary, ...m.more].filter((e) => e.current).map((e) => e.id);
	};

	it.each([
		['/', ['home']],
		['/kids/k1', ['home']],
		['/kids/k1/log/symptom', ['home']],
		['/timeline', ['timeline']],
		['/kids/k1/timeline', ['timeline']],
		['/kids/k1/log/bottles/e9', ['timeline']],
		['/kids/k1/growth/new', ['growth']],
		['/kids/k1/vaccines/a1', ['vaccines']],
		['/kids/k1/medication/doses/d1', ['medication']],
		['/kids/k1/milestones/new', ['milestones']],
		['/kids/k1/export/pdf', ['export']],
		['/kids/k1/access', ['profile']],
		['/kids/k1/settings', ['settings']],
		['/account', ['account']],
		['/account/notifications', []],
		['/kids/new', []]
	])('%s marks %j', (path, expected) => {
		expect(current(path)).toEqual(expected);
	});
});
