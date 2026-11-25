import { instantToLocalInput } from './time-zone';
import type { Visit, VisitDetail } from './types';

type Style = { weekday?: 'short' | 'long'; month: 'short' | 'long' };

// date is YYYY-MM-DD; formatted in UTC so neither the server's zone nor the locale moves it.
function formatDate(date: string, { weekday, month }: Style): string {
	const parts = new Intl.DateTimeFormat('en-US', {
		timeZone: 'UTC',
		...(weekday ? { weekday } : {}),
		day: 'numeric',
		month,
		year: 'numeric'
	}).formatToParts(new Date(`${date}T00:00:00Z`));
	const part = (type: string) => parts.find((p) => p.type === type)?.value;
	return [weekday && part('weekday'), part('day'), part('month'), part('year')]
		.filter(Boolean)
		.join(' ');
}

const local = (instant: string, zone: string) => instantToLocalInput(instant, zone).split('T');

// A list row: "Thu 3 Dec 2026, 10:30" on the kid's clock.
export function visitWhen(startsAt: string, zone: string): string {
	const [date, time] = local(startsAt, zone);
	return `${formatDate(date, { weekday: 'short', month: 'short' })}, ${time}`;
}

// The visit page: "Thursday 3 December 2026, 10:30 · 30 min".
export function visitWhenLong(v: Pick<Visit, 'startsAt' | 'durationMin'>, zone: string): string {
	const [date, time] = local(v.startsAt, zone);
	return `${formatDate(date, { weekday: 'long', month: 'long' })}, ${time} · ${v.durationMin} min`;
}

// A date of a linked record, YYYY-MM-DD: "3 Dec 2026".
export const visitRecordDate = (date: string) => formatDate(date, { month: 'short' });

// The visits spec 5.3: notes or any linked row.
export const hasOutcome = (v: Pick<VisitDetail, 'outcomeNotes' | 'linked'>) =>
	v.outcomeNotes !== null ||
	v.linked.growth.length + v.linked.vaccinations.length + v.linked.medications.length > 0;
