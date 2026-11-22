import { describe, expect, it } from 'vitest';
import type { Cookies } from '@sveltejs/kit';
import { currentKidId } from './current-kid';

const cookies = (value?: string) => ({ get: () => value }) as unknown as Cookies;
const kids = [{ id: 'k-a' }, { id: 'k-b' }];

describe('currentKidId', () => {
	it('is the remembered kid when the parent still has it', () => {
		expect(currentKidId(kids, cookies('k-b'))).toBe('k-b');
	});

	it('is the first kid without a cookie or with one naming a kid the parent lost', () => {
		expect(currentKidId(kids, cookies())).toBe('k-a');
		expect(currentKidId(kids, cookies('k-gone'))).toBe('k-a');
	});

	it('is undefined without kids', () => {
		expect(currentKidId([], cookies('k-a'))).toBeUndefined();
	});
});
