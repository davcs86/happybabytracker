import { describe, expect, it } from 'vitest';
import { applySystem, systemOf } from './growth-units';

describe('systemOf', () => {
	it('reads the pair of units as one system, or neither when they are mixed', () => {
		expect(systemOf({ weight: 'kg', length: 'cm' })).toBe('metric');
		expect(systemOf({ weight: 'lb', length: 'in' })).toBe('imperial');
		expect(systemOf({ weight: 'kg', length: 'in' })).toBeNull();
	});
});

describe('applySystem', () => {
	const units = { weight: 'kg', length: 'cm', volume: 'floz', temp: 'f' } as const;
	it('changes only weight and length', () => {
		expect(applySystem(units, 'imperial')).toEqual({ ...units, weight: 'lb', length: 'in' });
		expect(applySystem({ ...units, weight: 'lb', length: 'in' }, 'metric')).toEqual(units);
	});
});
