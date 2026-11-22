import { HUNDREDTHS_PER_FLOZ, SECONDS_PER_MINUTE } from './constants';
import type { Volume, VolumeUnit } from './types';

const MONTHS_PER_YEAR = 12;
const MS_PER_DAY = 24 * 60 * 60 * 1000;
const MINUTES_PER_HOUR = 60;

type Ymd = { y: number; m: number; d: number };

const parse = (date: string): Ymd => {
	const [y, m, d] = date.split('-').map(Number);
	return { y, m, d };
};
const daysInMonth = (y: number, m: number) => new Date(Date.UTC(y, m, 0)).getUTCDate();
const toUtc = ({ y, m, d }: Ymd) => Date.UTC(y, m - 1, d);

// The day is clamped to the target month's last day: 29 Feb + 12 months = 28 Feb.
function addMonths(from: Ymd, months: number): Ymd {
	const index = from.y * MONTHS_PER_YEAR + (from.m - 1) + months;
	const y = Math.floor(index / MONTHS_PER_YEAR);
	const m = (index % MONTHS_PER_YEAR) + 1;
	return { y, m, d: Math.min(from.d, daysInMonth(y, m)) };
}

const plural = (n: number, unit: string) => `${n} ${unit}${n === 1 ? '' : 's'}`;

// birthDate and today are YYYY-MM-DD; today must be computed in the kid's zone (todayIn).
export function formatAge(birthDate: string, today: string): string {
	const birth = parse(birthDate);
	const end = toUtc(parse(today));
	if (end <= toUtc(birth)) return plural(0, 'day');
	const t = parse(today);
	let months = (t.y - birth.y) * MONTHS_PER_YEAR + (t.m - birth.m);
	if (toUtc(addMonths(birth, months)) > end) months -= 1;
	const days = Math.round((end - toUtc(addMonths(birth, months))) / MS_PER_DAY);

	if (months === 0) return plural(days, 'day');
	if (months < MONTHS_PER_YEAR) {
		return days === 0
			? plural(months, 'month')
			: `${plural(months, 'month')} ${plural(days, 'day')}`;
	}
	const years = Math.floor(months / MONTHS_PER_YEAR);
	const rest = months % MONTHS_PER_YEAR;
	return rest === 0 ? plural(years, 'year') : `${plural(years, 'year')} ${plural(rest, 'month')}`;
}

// Both dates are YYYY-MM-DD in the kid's zone; the same rule as the API's AgeMonths.
export function ageMonths(birthDate: string, today: string): number {
	const b = parse(birthDate);
	const t = parse(today);
	const months = (t.y - b.y) * MONTHS_PER_YEAR + (t.m - b.m) - (t.d < b.d ? 1 : 0);
	return Math.max(months, 0);
}

export function formatVolume(v: Pick<Volume, 'ml' | 'floz_hundredths'>, unit: VolumeUnit): string {
	if (unit === 'ml') return `${v.ml} ml`;
	return `${Number((v.floz_hundredths / HUNDREDTHS_PER_FLOZ).toFixed(2))} fl oz`;
}

export function formatDuration(seconds: number): string {
	const total = Math.max(0, Math.floor(seconds));
	if (total < SECONDS_PER_MINUTE) return `${total} s`;
	const minutes = Math.floor(total / SECONDS_PER_MINUTE);
	const hours = Math.floor(minutes / MINUTES_PER_HOUR);
	const rest = minutes % MINUTES_PER_HOUR;
	if (hours === 0) return `${rest} min`;
	return rest === 0 ? `${hours} h` : `${hours} h ${rest} min`;
}

export function todayIn(timeZone: string, now = new Date()): string {
	const parts = new Intl.DateTimeFormat('en-CA', {
		timeZone,
		year: 'numeric',
		month: '2-digit',
		day: '2-digit'
	}).formatToParts(now);
	const part = (type: string) => parts.find((p) => p.type === type)?.value;
	return `${part('year')}-${part('month')}-${part('day')}`;
}

export function formatInstant(rfc3339: string): string {
	const iso = new Date(rfc3339).toISOString();
	return `${iso.slice(0, 10)} ${iso.slice(11, 16)} UTC`;
}
