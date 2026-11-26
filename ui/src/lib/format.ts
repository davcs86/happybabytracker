import {
	DAYS_PER_MONTH,
	HUNDREDTHS_PER_FLOZ,
	HUNDREDTHS_PER_OZ,
	SECONDS_PER_HOUR,
	SECONDS_PER_MINUTE
} from './constants';
import type { Milk, Units, Volume, VolumeUnit } from './types';

const MONTHS_PER_YEAR = 12;
const MS_PER_DAY = 24 * 60 * 60 * 1000;
const MINUTES_PER_HOUR = 60;
const HOURS_PER_DAY = 24;

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

// An age in days, in months of DAYS_PER_MONTH days (the curves' axis), so it may differ from formatAge by a day.
export function formatAgeDays(days: number): string {
	const total = Math.max(0, days);
	const months = Math.floor(total / DAYS_PER_MONTH);
	if (months === 0) return plural(total, 'day');
	if (months < MONTHS_PER_YEAR) {
		const rest = Math.floor(total - months * DAYS_PER_MONTH);
		return rest === 0
			? plural(months, 'month')
			: `${plural(months, 'month')} ${plural(rest, 'day')}`;
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

// The API's columns in the parent's units; nothing is converted here.
export function formatMilk(milk: Milk, units: Pick<Units, 'weight' | 'volume'>): string {
	const gain =
		units.weight === 'kg'
			? `${milk.gain.grams} g`
			: `${Number((milk.gain.ozHundredths / HUNDREDTHS_PER_OZ).toFixed(2))} oz`;
	const volume = formatVolume(
		{ ml: milk.volume.ml, floz_hundredths: milk.volume.flozHundredths },
		units.volume
	);
	return `weight gain ${gain}, about ${volume}`;
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

const pad2 = (n: number) => String(n).padStart(2, '0');

// The running clock: H:MM:SS, the hours growing past a day.
export function formatElapsed(seconds: number): string {
	const total = Math.max(0, Math.floor(seconds));
	const hours = Math.floor(total / SECONDS_PER_HOUR);
	const rest = total % SECONDS_PER_HOUR;
	return `${hours}:${pad2(Math.floor(rest / SECONDS_PER_MINUTE))}:${pad2(rest % SECONDS_PER_MINUTE)}`;
}

// "1 h 34 min ago": minutes under an hour, hours and minutes under a day, then whole days.
export function formatAgo(instant: string, now: Date): string {
	const seconds = Math.floor((now.getTime() - new Date(instant).getTime()) / 1000);
	const minutes = Math.floor(seconds / SECONDS_PER_MINUTE);
	if (minutes < 1) return 'Just now';
	if (minutes < MINUTES_PER_HOUR) return `${minutes} min ago`;
	const hours = Math.floor(minutes / MINUTES_PER_HOUR);
	if (hours < HOURS_PER_DAY) {
		const rest = minutes % MINUTES_PER_HOUR;
		return rest === 0 ? `${hours} h ago` : `${hours} h ${rest} min ago`;
	}
	return `${Math.floor(hours / HOURS_PER_DAY)} d ago`;
}

const DAYS_PER_MONTH_LABEL = 30;

// "3 days ago": today and yesterday by name, whole days up to a month, then whole months.
export function formatDaysAgo(instant: string, now: Date): string {
	const days = Math.floor((now.getTime() - new Date(instant).getTime()) / MS_PER_DAY);
	if (days < 1) return 'today';
	if (days === 1) return 'yesterday';
	if (days < DAYS_PER_MONTH_LABEL) return `${days} days ago`;
	const months = Math.floor(days / DAYS_PER_MONTH_LABEL);
	return months === 1 ? '1 month ago' : `${months} months ago`;
}
