import { medicationPath, vaccinesPath } from './paths';
import { whenLabel } from './medication';
import type { HomeDue, HomeDueVisit } from './types';
import { dueLabel } from './vaccines';

export type DueLine = { text: string; href: string };

// The first max entries in API order, then one line per kind hidden; the API alone decides what is listed.
export function dueLines(
	due: Exclude<HomeDue, HomeDueVisit>[],
	max: number,
	kidId: string,
	zone: string,
	today: string
): DueLine[] {
	const vaccines = vaccinesPath(kidId);
	const medication = medicationPath(kidId);
	const lines = due.slice(0, max).map((entry): DueLine => {
		if (entry.kind === 'vaccine') return { text: dueLabel(entry), href: vaccines };
		const when = `${entry.medication.name} · ${whenLabel(entry.dueAt, zone, today)}`;
		return { text: entry.state === 'overdue' ? `${when} · overdue` : when, href: medication };
	});
	const hidden = due.slice(max);
	const hiddenMedication = hidden.filter((e) => e.kind === 'medication').length;
	const hiddenVaccines = hidden.length - hiddenMedication;
	if (hiddenMedication > 0) {
		lines.push({ text: `${hiddenMedication} more on the Medication page`, href: medication });
	}
	if (hiddenVaccines > 0) {
		lines.push({ text: `${hiddenVaccines} more on the Vaccines page`, href: vaccines });
	}
	return lines;
}
