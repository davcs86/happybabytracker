import { formatClock, instantToLocalInput } from './time-zone';
import type { Schedule } from './types';

// The schedule as the API stores it; the app shows it and never suggests one.
export function scheduleLabel(schedule: Schedule, startAt: string, zone: string): string {
	switch (schedule.kind) {
		case 'none':
			return 'As needed';
		case 'times':
			return schedule.times.join(', ');
		case 'every': {
			const every = schedule.hours === 1 ? 'hour' : `${schedule.hours} hours`;
			return `Every ${every} from ${formatClock(startAt, zone)}`;
		}
	}
}

// The kid-local clock time, with its date unless it is today so a time is never read as today's.
export function whenLabel(instant: string, zone: string, today: string): string {
	const [date, time] = instantToLocalInput(instant, zone).split('T');
	return date === today ? time : `${date} ${time}`;
}

// end_on is the course's last day, inclusive.
export const courseEnded = (endOn: string | null, today: string): boolean =>
	endOn !== null && endOn < today;

export const doseCountLabel = (n: number): string => `${n} ${n === 1 ? 'dose' : 'doses'}`;
