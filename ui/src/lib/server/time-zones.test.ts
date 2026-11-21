import { describe, expect, it } from 'vitest';
import { timeZoneOptions } from './time-zones';

describe('timeZoneOptions', () => {
	it('always has UTC, once', () => {
		const zones = timeZoneOptions();
		expect(zones.filter((z) => z === 'UTC')).toEqual(['UTC']);
	});

	it('includes a stored zone absent from the ICU list', () => {
		expect(Intl.supportedValuesOf('timeZone')).not.toContain('Asia/Kolkata');
		expect(timeZoneOptions('Asia/Kolkata')).toContain('Asia/Kolkata');
	});

	it('does not duplicate a stored zone the ICU list has', () => {
		const zones = timeZoneOptions('Asia/Calcutta');
		expect(zones.filter((z) => z === 'Asia/Calcutta')).toEqual(['Asia/Calcutta']);
		expect(zones).toEqual([...zones].sort());
	});
});
