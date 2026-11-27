import { VISIT_KIND_LABEL } from './constants';
import { medicationPath, vaccinesPath, visitPath, visitsPath } from './paths';
import { whenLabel } from './medication';
import type { HomeDue } from './types';
import { dueLabel } from './vaccines';
import { visitDueWhen } from './visits';

export type DueLine = { kind: HomeDue['kind']; text: string; href: string };

const MORE_PAGE: Record<HomeDue['kind'], string> = {
	medication: 'Medication',
	vaccine: 'Vaccines',
	visit: 'Visits'
};
const MORE_ORDER = ['medication', 'vaccine', 'visit'] as const satisfies HomeDue['kind'][];

// The first max entries in API order, then one line per kind hidden; the API alone decides what is listed.
export function dueLines(
	due: HomeDue[],
	max: number,
	kidId: string,
	zone: string,
	today: string
): DueLine[] {
	const pages: Record<HomeDue['kind'], string> = {
		medication: medicationPath(kidId),
		vaccine: vaccinesPath(kidId),
		visit: visitsPath(kidId)
	};
	const lines = due.slice(0, max).map((entry): DueLine => {
		switch (entry.kind) {
			case 'vaccine':
				return { kind: entry.kind, text: dueLabel(entry), href: pages.vaccine };
			case 'visit':
				return {
					kind: entry.kind,
					text: `${VISIT_KIND_LABEL[entry.visitKind]}, ${visitDueWhen(entry.startsAt, zone)}`,
					href: visitPath(kidId, entry.id)
				};
			case 'medication': {
				const when = `${entry.medication.name} · ${whenLabel(entry.dueAt, zone, today)}`;
				const text = entry.state === 'overdue' ? `${when} · overdue` : when;
				return { kind: entry.kind, text, href: pages.medication };
			}
		}
	});
	const hidden = due.slice(max);
	for (const kind of MORE_ORDER) {
		const n = hidden.filter((e) => e.kind === kind).length;
		if (n > 0)
			lines.push({ kind, text: `${n} more on the ${MORE_PAGE[kind]} page`, href: pages[kind] });
	}
	return lines;
}
