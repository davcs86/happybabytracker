import { VACCINE_STATUS_LABEL, VACCINE_UNIT_LABEL } from './constants';
import type {
	HomeDue,
	VaccineAge,
	VaccineCatalogue,
	VaccineGroup,
	VaccineLine,
	VaccineSchemaInfo,
	VaccineStatus
} from './types';

// Dates come from the API as kid-local YYYY-MM-DD; nothing here does date arithmetic.

export const ageLabel = ({ n, unit }: VaccineAge) =>
	n === 0
		? 'birth'
		: `${n} ${n === 1 ? VACCINE_UNIT_LABEL[unit].one : VACCINE_UNIT_LABEL[unit].many}`;

export const groupTitle = ({ from }: Pick<VaccineGroup, 'from'>) =>
	from.n === 0 ? 'Birth' : `${from.n}-${VACCINE_UNIT_LABEL[from.unit].one} vaccines`;

export const statusLabel = (status: VaccineStatus) => VACCINE_STATUS_LABEL[status];

export const dueLabel = (entry: HomeDue) => `${groupTitle(entry)} · ${entry.status}`;

export const windowLabel = (line: Pick<VaccineLine, 'fromDate' | 'untilDate'>) =>
	`Due ${line.fromDate} – ${line.untilDate}`;

export const antigenName = (id: string, catalogue: VaccineCatalogue) =>
	catalogue.antigens.find((a) => a.id === id)?.name ?? id;

// A stored schedule the API no longer ships is no schedule: the API answers schema null for it.
export const schemaChosen = (kidSchemaId: string | null, schemas: VaccineSchemaInfo[]) =>
	kidSchemaId !== null && schemas.some((s) => s.id === kidSchemaId);
