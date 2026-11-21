import { DEFAULT_TIME_ZONE } from '../constants';
import { withZone } from '../time-zone';

// The runtime's IANA zones plus UTC, and the stored zone so an edit never drops it.
export function timeZoneOptions(stored?: string): string[] {
	const zones = withZone(Intl.supportedValuesOf('timeZone').sort(), DEFAULT_TIME_ZONE);
	return stored ? withZone(zones, stored) : zones;
}
