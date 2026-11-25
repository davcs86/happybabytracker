import type { Measure, TrackerKind } from './types';

// Absolute paths for redirects and for links that stay absolute without JavaScript (resolve() is relative there).
export const logPath = (kidId: string, kind: TrackerKind) => `/kids/${kidId}/log/${kind}`;
export const entryPath = (kidId: string, kind: TrackerKind, id: string) =>
	`${logPath(kidId, kind)}/${id}`;
export const timelinePath = (kidId: string) => `/kids/${kidId}/timeline`;
// A kid-local date, YYYY-MM-DD.
export const timelineDayPath = (kidId: string, date: string) =>
	`${timelinePath(kidId)}?from=${date}&to=${date}`;
export const medicationPath = (kidId: string) => `/kids/${kidId}/medication`;
export const growthPath = (kidId: string) => `/kids/${kidId}/growth`;
export const growthMeasurePath = (kidId: string, measure: Measure) =>
	`${growthPath(kidId)}?measure=${measure}`;
export const growthNewPath = (kidId: string) => `${growthPath(kidId)}/new`;
export const growthEntryPath = (kidId: string, id: string) => `${growthPath(kidId)}/${id}`;
export const vaccinesPath = (kidId: string) => `/kids/${kidId}/vaccines`;
// antigens are catalogue ids, which need no escaping in a query.
export const vaccineNewPath = (kidId: string, antigens: string[] = []) =>
	`${vaccinesPath(kidId)}/new${antigens.length ? `?antigens=${antigens.join(',')}` : ''}`;
export const medicationNewPath = (kidId: string) => `${medicationPath(kidId)}/new`;
export const medicationEntryPath = (kidId: string, id: string) => `${medicationPath(kidId)}/${id}`;
export const doseEntryPath = (kidId: string, id: string) => `${medicationPath(kidId)}/doses/${id}`;
export const notificationsPath = '/account/notifications';
export const notificationsSubscribePath = `${notificationsPath}/subscribe`;
export const milestonesPath = (kidId: string) => `/kids/${kidId}/milestones`;
// item is a catalog item id, which needs no escaping in a query.
export const milestoneNewPath = (kidId: string, item?: string) =>
	`${milestonesPath(kidId)}/new${item ? `?item=${item}` : ''}`;
export const milestoneEntryPath = (kidId: string, id: string) => `${milestonesPath(kidId)}/${id}`;
export const exportPath = (kidId: string) => `/kids/${kidId}/export`;
export const exportPdfPath = (kidId: string) => `${exportPath(kidId)}/pdf`;
export const visitsPath = (kidId: string) => `/kids/${kidId}/visits`;
export const visitNewPath = (kidId: string) => `${visitsPath(kidId)}/new`;
export const visitPath = (kidId: string, id: string) => `${visitsPath(kidId)}/${id}`;
export const visitEditPath = (kidId: string, id: string) => `${visitPath(kidId, id)}/edit`;
export const visitOutcomePath = (kidId: string, id: string) => `${visitPath(kidId, id)}/outcome`;
export const visitCalendarPath = (kidId: string, id: string) => `${visitPath(kidId, id)}/calendar`;
