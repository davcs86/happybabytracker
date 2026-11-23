import { VACCINE_STATUS_LABEL, VACCINE_UNIT_LABEL } from './constants';
import type {
	HomeDue,
	VaccineCatalogue,
	VaccineGroup,
	VaccineLine,
	VaccineSchemaInfo,
	VaccineStatus
} from './types';

// Dates come from the API as kid-local YYYY-MM-DD; nothing here does date arithmetic.

export const groupTitle = ({ from }: Pick<VaccineGroup, 'from'>) =>
	from.n === 0 ? 'Birth' : `${from.n}-${VACCINE_UNIT_LABEL[from.unit].one} vaccines`;

export const statusLabel = (status: VaccineStatus) => VACCINE_STATUS_LABEL[status];

export const dueLabel = (entry: HomeDue) => `${groupTitle(entry)} · ${statusLabel(entry.status)}`;

// untilDate is the first day the window is over; recurring lines have no window to show.
export function windowLabel(line: Pick<VaccineLine, 'kind' | 'fromDate' | 'untilDate'>): string {
	switch (line.kind) {
		case 'routine':
			return `Due from ${line.fromDate} · overdue from ${line.untilDate}`;
		case 'may_not_be_needed':
			return `Due from ${line.fromDate} · no longer due from ${line.untilDate}`;
		case 'supplementary':
			return `Available from ${line.fromDate}`;
		case 'recurring':
			return '';
	}
}

export const antigenName = (id: string, catalogue: VaccineCatalogue) =>
	catalogue.antigens.find((a) => a.id === id)?.name ?? id;

// A stored schedule the API no longer ships is no schedule: the API answers schema null for it.
export const schemaChosen = (kidSchemaId: string | null, schemas: VaccineSchemaInfo[]) =>
	kidSchemaId !== null && schemas.some((s) => s.id === kidSchemaId);
