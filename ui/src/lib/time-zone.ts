import { DEFAULT_TIME_ZONE } from './constants';
import { todayIn } from './format';

// The runtime's own check; the API (Go tzdata) stays the only validator of what is stored.
export function acceptsZone(name: string): boolean {
	if (name === '') return false;
	try {
		new Intl.DateTimeFormat(undefined, { timeZone: name });
		return true;
	} catch {
		return false;
	}
}

// ICU lists some zones only under older names (Asia/Calcutta, not Asia/Kolkata).
export function withZone(zones: string[], zone: string): string[] {
	return zones.includes(zone) ? zones : [...zones, zone].sort();
}

export function todayInOrUtc(timeZone: string, now = new Date()): string {
	return acceptsZone(timeZone) ? todayIn(timeZone, now) : todayIn(DEFAULT_TIME_ZONE, now);
}
