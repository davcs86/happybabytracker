import type { Units } from './types';

export type UnitSystem = 'metric' | 'imperial';

// The Growth page's one switch sets weight and length together; volume and temperature stay as chosen.
export const SYSTEMS: Record<UnitSystem, Pick<Units, 'weight' | 'length'>> = {
	metric: { weight: 'kg', length: 'cm' },
	imperial: { weight: 'lb', length: 'in' }
};

export function systemOf(u: Pick<Units, 'weight' | 'length'>): UnitSystem | null {
	return (
		(Object.keys(SYSTEMS) as UnitSystem[]).find(
			(s) => SYSTEMS[s].weight === u.weight && SYSTEMS[s].length === u.length
		) ?? null
	);
}

export const applySystem = (units: Units, system: UnitSystem): Units => ({
	...units,
	...SYSTEMS[system]
});
