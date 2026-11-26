import { instantToLocalInput } from './time-zone';
import type { Visit, VisitDetail, VisitLinks } from './types';

type Style = { weekday?: 'short' | 'long'; month: 'short' | 'long'; year?: false };

// date is YYYY-MM-DD; formatted in UTC so neither the server's zone nor the locale moves it.
function formatDate(date: string, { weekday, month, year }: Style): string {
	const parts = new Intl.DateTimeFormat('en-US', {
		timeZone: 'UTC',
		...(weekday ? { weekday } : {}),
		day: 'numeric',
		month,
		...(year === false ? {} : { year: 'numeric' as const })
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

// Home's Due line: "Thu 3 Dec, 10:30" on the kid's clock.
export function visitDueWhen(startsAt: string, zone: string): string {
	const [date, time] = local(startsAt, zone);
	return `${formatDate(date, { weekday: 'short', month: 'short', year: false })}, ${time}`;
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

const count = (n: number, one: string, many: string) => `${n} ${n === 1 ? one : many}`;

// The delete confirm's consequence lines: "1 growth measurement", "2 vaccines",
// "1 medication, with its 14 logged doses"; none when nothing is linked.
export function visitRecordLines(l: VisitLinks): string[] {
	const lines: string[] = [];
	if (l.growth.length)
		lines.push(count(l.growth.length, 'growth measurement', 'growth measurements'));
	if (l.vaccinations.length) lines.push(count(l.vaccinations.length, 'vaccine', 'vaccines'));
	const n = l.medications.length;
	if (n) {
		const doses = l.medications.reduce((sum, m) => sum + m.doseCount, 0);
		const logged = doses
			? `, with ${n === 1 ? 'its' : 'their'} ${count(doses, 'logged dose', 'logged doses')}`
			: '';
		lines.push(`${count(n, 'medication', 'medications')}${logged}`);
	}
	return lines;
}
