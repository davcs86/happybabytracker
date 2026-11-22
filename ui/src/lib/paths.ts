import type { Measure, TrackerKind } from './types';

// Absolute paths for redirects and for links that stay absolute without JavaScript (resolve() is relative there).
export const logPath = (kidId: string, kind: TrackerKind) => `/kids/${kidId}/log/${kind}`;
export const entryPath = (kidId: string, kind: TrackerKind, id: string) =>
	`${logPath(kidId, kind)}/${id}`;
export const timelinePath = (kidId: string) => `/kids/${kidId}/timeline`;
// A kid-local date, YYYY-MM-DD.
export const timelineDayPath = (kidId: string, date: string) =>
	`${timelinePath(kidId)}?from=${date}&to=${date}`;
export const growthPath = (kidId: string) => `/kids/${kidId}/growth`;
export const growthMeasurePath = (kidId: string, measure: Measure) =>
	`${growthPath(kidId)}?measure=${measure}`;
export const growthNewPath = (kidId: string) => `${growthPath(kidId)}/new`;
export const growthEntryPath = (kidId: string, id: string) => `${growthPath(kidId)}/${id}`;
export const vaccinesPath = (kidId: string) => `/kids/${kidId}/vaccines`;
// antigens are catalogue ids, which need no escaping in a query.
export const vaccineNewPath = (kidId: string, antigens: string[] = []) =>
	`${vaccinesPath(kidId)}/new${antigens.length ? `?antigens=${antigens.join(',')}` : ''}`;
export const vaccinePath = (kidId: string, administration: string) =>
	`${vaccinesPath(kidId)}/${administration}`;
