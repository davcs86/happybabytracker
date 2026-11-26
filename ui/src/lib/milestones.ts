import type { MilestoneDomain, Milestones } from './types';

// CDC's own age labels for its checklists, keyed by MILESTONE_AGES.
const CHECKLIST_TITLE: Record<number, string> = {
	2: 'By 2 months',
	4: 'By 4 months',
	6: 'By 6 months',
	9: 'By 9 months',
	12: 'By 1 year',
	15: 'By 15 months',
	18: 'By 18 months',
	24: 'By 2 years',
	30: 'By 30 months',
	36: 'By 3 years',
	48: 'By 4 years',
	60: 'By 5 years'
};

// CDC's domain headings, without the word "Milestones".
const DOMAIN_LABEL: Record<MilestoneDomain, string> = {
	social_emotional: 'Social/Emotional',
	language_communication: 'Language/Communication',
	cognitive: 'Cognitive (learning, thinking, problem-solving)',
	movement_physical: 'Movement/Physical Development'
};

const MONTH_NAMES = [
	'January',
	'February',
	'March',
	'April',
	'May',
	'June',
	'July',
	'August',
	'September',
	'October',
	'November',
	'December'
];

export const checklistTitle = (ageMonths: number): string => CHECKLIST_TITLE[ageMonths];

export const domainLabel = (domain: MilestoneDomain): string => DOMAIN_LABEL[domain];

// date is a kid-local YYYY-MM-DD and is printed as written: no zone applies to it.
export function reachedLabel(date: string): string {
	const [y, m, d] = date.split('-').map(Number);
	return `Reached on ${d} ${MONTH_NAMES[m - 1]} ${y}`;
}

// The id of the row a just-marked milestone became; a custom one is the newest with that text.
export function reachedId(m: Milestones, by: { item: string } | { text: string }): string | null {
	if ('item' in by) {
		const listed = m.checklists.flatMap((c) => c.items).find((i) => i.id === by.item)?.reached;
		return listed?.id ?? m.reached.find((r) => r.item === by.item)?.id ?? null;
	}
	const matches = m.custom.filter((c) => c.text === by.text);
	return (
		matches.reduce<(typeof matches)[number] | null>(
			(newest, c) => (newest === null || c.reachedOn >= newest.reachedOn ? c : newest),
			null
		)?.id ?? null
	);
}
