import { fail, redirect } from '@sveltejs/kit';
import { NURSING_SIDES } from '#lib/constants.js';
import { formatAge } from '#lib/format.js';
import { ageHeader } from '#lib/growth.js';
import { quickLog } from '#lib/quick-log.js';
import { todayRows } from '#lib/today.js';
import { todayInOrUtc } from '#lib/time-zone.js';
import { FAILURE_MESSAGE, KID_COOKIE, PARENT_HOME } from '#lib/server/constants.js';
import { clearKidCookie, setKidCookie } from '#lib/server/cookie.js';
import { failAction, failPage, type ApiFailure } from '#lib/server/outcome.js';
import type { TimerKind } from '#lib/types.js';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals, params, cookies, parent }) => {
	const [kid, home, { session }] = await Promise.all([
		locals.api.getKid(params.kid),
		locals.api.home(params.kid),
		parent()
	]);
	if (!kid.ok) failPage(kid);
	if (!home.ok) failPage(home);
	setKidCookie(cookies, kid.data.id);
	const enabled = new Set(home.data.modules.filter((m) => m.enabled).map((m) => m.module));
	const today = todayInOrUtc(kid.data.timeZone);
	return {
		kid: kid.data,
		age: ageHeader(formatAge(kid.data.birthDate, today), home.data.correctedBirthDate, today),
		running: home.data.running,
		quickLog: quickLog(home.data.band, home.data.modules, home.data.running),
		due: home.data.due,
		todayDate: today,
		today: todayRows(home.data.today, enabled, session?.parent?.units.volume ?? 'ml')
	};
};

// A timer another device already ended or started is the state the reloaded page shows.
const settled = (r: ApiFailure, code: string) => r.reason === 'conflict' && r.code === code;

const formId = async (request: Request) => String((await request.formData()).get('id') ?? '');

function endTimer(kind: TimerKind, form: 'endSleep' | 'endNursing') {
	return async ({
		request,
		locals,
		params
	}: {
		request: Request;
		locals: App.Locals;
		params: { kid: string };
	}) => {
		const r = await locals.api.endTimer(kind, params.kid, await formId(request));
		if (r.ok || settled(r, 'not_running')) return { form };
		// 422 on at: the timer ran over 24 hours, so the end time needs a person.
		if (r.reason === 'invalid' && r.field === 'at') return fail(422, { form, overrun: true });
		return failAction(r, { form });
	};
}

export const actions: Actions = {
	startSleep: async ({ locals, params }) => {
		const r = await locals.api.tracker('sleep').create(params.kid, {});
		if (r.ok || settled(r, 'already_running')) return { form: 'startSleep' as const };
		return failAction(r, { form: 'startSleep' as const });
	},
	endSleep: endTimer('sleep', 'endSleep'),
	endNursing: endTimer('nursing', 'endNursing'),
	switchSide: async ({ request, locals, params }) => {
		const data = await request.formData();
		const side = String(data.get('side') ?? '');
		const form = 'switchSide' as const;
		if (!(NURSING_SIDES as readonly string[]).includes(side)) {
			return fail(400, { form, message: FAILURE_MESSAGE });
		}
		const r = await locals.api.switchNursingSide(
			params.kid,
			String(data.get('id') ?? ''),
			side as (typeof NURSING_SIDES)[number]
		);
		if (r.ok || settled(r, 'not_running')) return { form };
		return failAction(r, { form });
	},
	// The parent is always the signed-in principal; posted fields are never read.
	leave: async ({ locals, params, cookies }) => {
		const me = locals.session?.parent?.id;
		if (!me) failPage({ ok: false, status: 401, reason: 'unauthorized', code: 'no_session' });
		const r = await locals.api.removeAccess(params.kid, me);
		// not_found: already left, or the grant was revoked meanwhile.
		if (r.ok || r.reason === 'not_found') {
			if (cookies.get(KID_COOKIE) === params.kid) clearKidCookie(cookies);
			redirect(303, PARENT_HOME);
		}
		return failAction(r, { form: 'leave' as const });
	}
};
