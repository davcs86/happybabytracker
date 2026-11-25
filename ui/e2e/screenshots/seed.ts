import { stubAs } from '../stub-api.js';
import { NOW_MS } from './shots.js';

export type Api = Awaited<ReturnType<typeof stubAs>>;
export const as = stubAs;

// HH:MM UTC on the frozen day, or daysAgo days before it.
export function at(hhmm: string, daysAgo = 0): string {
	const day = new Date(NOW_MS - daysAgo * 86_400_000).toISOString().slice(0, 10);
	return `${day}T${hhmm}:00Z`;
}

export const kidPath = (kid: string, rest = '') => `/v1/kids/${kid}${rest ? `/${rest}` : ''}`;

// A seeding call that must succeed; its body is the stored row.
export async function must(api: Api, method: string, path: string, body?: unknown) {
	const r = await api(method, path, body);
	if (r.status >= 300) throw new Error(`${method} ${path}: ${r.status} ${JSON.stringify(r.body)}`);
	return r.body;
}

// One varied day of every tracker for a young kid, plus a few rows the day before.
export async function seedDay(api: Api, kid: string) {
	const post = (tracker: string, body: unknown) => must(api, 'POST', kidPath(kid, tracker), body);
	const rows = {
		sleepOvernight: await post('sleep', {
			started_at: at('20:15', 1),
			ended_at: at('05:40'),
			note: 'Woke once at 2'
		}),
		nursing: await post('nursing', {
			started_at: at('06:00'),
			ended_at: at('06:25'),
			left_seconds: 720,
			right_seconds: 660,
			weight_before: { unit: 'kg', amount: 6400 },
			weight_after: { unit: 'kg', amount: 6462 }
		}),
		diaperWet: await post('diapers', { at: at('06:30'), kind: 'wet' }),
		bottle: await post('bottles', {
			at: at('07:10'),
			contents: ['formula'],
			volume: { unit: 'ml', amount: 120 }
		}),
		pumping: await post('pumping', {
			started_at: at('07:45'),
			duration_seconds: 1200,
			sides: { unit: 'ml', amounts: { left: 70, right: 55 } }
		}),
		diaperDirty: await post('diapers', { at: at('08:05'), kind: 'dirty', note: 'Seedy' }),
		nap: await post('sleep', { started_at: at('08:20'), ended_at: at('09:05') }),
		bottleMixed: await post('bottles', {
			at: at('09:15'),
			sources: { unit: 'ml', amounts: { breast_milk: 60, formula: 30 } }
		}),
		meal: await post('meals', { at: at('09:40'), what: 'Pureed carrot', reaction: 'Liked it' }),
		fever: await post('symptoms', {
			at: at('10:00'),
			symptom: 'fever',
			severity: 'moderate',
			temperature: { unit: 'c', amount: 3850 },
			method: 'ear'
		}),
		diaperBoth: await post('diapers', { at: at('10:10'), kind: 'both' }),
		yesterdayBottle: await post('bottles', {
			at: at('16:30', 1),
			contents: ['breast_milk'],
			volume: { unit: 'ml', amount: 90 }
		}),
		yesterdayCough: await post('symptoms', {
			at: at('18:00', 1),
			symptom: 'cough',
			severity: 'mild'
		}),
		yesterdayOther: await post('symptoms', {
			at: at('19:10', 1),
			symptom: 'other',
			other_text: 'Hiccups for ten minutes'
		}),
		yesterdayDiaper: await post('diapers', { at: at('19:30', 1), kind: 'wet' })
	};
	return rows;
}
