import type { TrackerKind } from './types';

// Absolute paths for redirects and for links that stay absolute without JavaScript (resolve() is relative there).
export const logPath = (kidId: string, kind: TrackerKind) => `/kids/${kidId}/log/${kind}`;
export const entryPath = (kidId: string, kind: TrackerKind, id: string) =>
	`${logPath(kidId, kind)}/${id}`;
export const timelinePath = (kidId: string) => `/kids/${kidId}/timeline`;
// A kid-local date, YYYY-MM-DD.
export const timelineDayPath = (kidId: string, date: string) =>
	`${timelinePath(kidId)}?from=${date}&to=${date}`;
