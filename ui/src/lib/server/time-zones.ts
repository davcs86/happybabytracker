import { DEFAULT_TIME_ZONE } from '../constants';

// The runtime's IANA zones plus UTC, and the stored zone so an edit never drops it.
export function timeZoneOptions(stored?: string): string[] {
	const zones = new Set(Intl.supportedValuesOf('timeZone'));
	zones.add(DEFAULT_TIME_ZONE);
	if (stored) zones.add(stored);
	return [...zones].sort();
}
