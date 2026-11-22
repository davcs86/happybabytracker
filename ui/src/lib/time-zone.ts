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

const LOCAL_INPUT = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})$/;
const MS_PER_DAY = 24 * 60 * 60 * 1000;

const formatters = new Map<string, Intl.DateTimeFormat>();
function wallFormat(zone: string): Intl.DateTimeFormat {
	let f = formatters.get(zone);
	if (!f) {
		f = new Intl.DateTimeFormat('en-CA', {
			timeZone: zone,
			hourCycle: 'h23',
			year: 'numeric',
			month: '2-digit',
			day: '2-digit',
			hour: '2-digit',
			minute: '2-digit',
			second: '2-digit'
		});
		formatters.set(zone, f);
	}
	return f;
}

// A zone the runtime cannot format reads as UTC, like todayInOrUtc.
const usable = (zone: string) => (acceptsZone(zone) ? zone : DEFAULT_TIME_ZONE);

// The wall clock fields as a UTC timestamp, so two of them subtract to an offset.
function utcMs(y: number, mo: number, d: number, h: number, mi: number, s: number): number {
	const date = new Date(0);
	date.setUTCFullYear(y, mo - 1, d);
	date.setUTCHours(h, mi, s, 0);
	return date.getTime();
}

function offsetAt(ms: number, zone: string): number {
	const parts = wallFormat(zone).formatToParts(new Date(ms));
	const n = (type: string) => Number(parts.find((p) => p.type === type)?.value);
	const wall = utcMs(n('year'), n('month'), n('day'), n('hour'), n('minute'), n('second'));
	return wall - Math.floor(ms / 1000) * 1000;
}

const rfc3339 = (ms: number) => new Date(ms).toISOString().replace(/\.\d{3}Z$/, 'Z');

// A time in a repeated hour is the earlier instant; one inside a skipped hour moves forward by the gap.
export function zonedLocalToInstant(local: string, zone: string): string | null {
	const m = LOCAL_INPUT.exec(local);
	if (!m) return null;
	const [y, mo, d, h, mi] = m.slice(1).map(Number);
	const wall = utcMs(y, mo, d, h, mi, 0);
	const check = new Date(wall);
	if (
		check.getUTCFullYear() !== y ||
		check.getUTCMonth() !== mo - 1 ||
		check.getUTCDate() !== d ||
		h > 23 ||
		mi > 59
	) {
		return null;
	}
	const tz = usable(zone);
	const before = offsetAt(wall - MS_PER_DAY, tz);
	const after = offsetAt(wall + MS_PER_DAY, tz);
	const instants = [before, after]
		.map((offset) => wall - offset)
		.filter((t) => offsetAt(t, tz) === wall - t);
	return rfc3339(instants.length > 0 ? Math.min(...instants) : wall - before);
}

export function instantToLocalInput(instant: string, zone: string): string {
	const parts = wallFormat(usable(zone)).formatToParts(new Date(instant));
	const part = (type: string) => parts.find((p) => p.type === type)?.value;
	return `${part('year')}-${part('month')}-${part('day')}T${part('hour')}:${part('minute')}`;
}
