import { redirect } from '@sveltejs/kit';
import {
	type EntryValues,
	editValues,
	entryInstant,
	formUnits,
	isAdvanced,
	isRunning,
	localDate,
	milkStatus,
	ranOver,
	trackerOf
} from '#lib/entry-edit.js';
import { timelineDayPath } from '#lib/paths.js';
import { setKidCookie } from '#lib/server/cookie.js';
import {
	entryApiFailure,
	entryFieldFailure,
	staleTimerFailure
} from '#lib/server/entry-failure.js';
import { readEntryForm } from '#lib/server/entry-form.js';
import { fromRunningPage, sendOnlyChanged } from '#lib/server/entry-submit.js';
import { failAction, failPage } from '#lib/server/outcome.js';
import type { Actions, PageServerLoad } from './$types';

// The kid and the entry, read together: no grant and a stale entry id are the same 404 page.
async function stored(locals: App.Locals, params: { kid: string; tracker: string; entry: string }) {
	const kind = trackerOf(params.tracker);
	const [kid, entry] = await Promise.all([
		locals.api.getKid(params.kid),
		locals.api.tracker(kind).get(params.kid, params.entry)
	]);
	if (!kid.ok) failPage(kid);
	if (!entry.ok) failPage(entry);
	return { kind, kid: kid.data, entry: entry.data };
}

export const load: PageServerLoad = async ({ locals, params, cookies, parent, url }) => {
	const [{ kind, kid, entry }, { session }] = await Promise.all([stored(locals, params), parent()]);
	setKidCookie(cookies, kid.id);
	const running = isRunning(kind, entry);
	return {
		kid,
		kind,
		running,
		estimate: milkStatus(kind, entry, formUnits(session)),
		overrun: running && ranOver(entryInstant(kind, entry), Date.now()),
		values: {
			...editValues(kind, entry, kid.timeZone, formUnits(session)),
			...(isAdvanced(kind, url.searchParams.get('advanced'), entry) ? { advanced: '1' } : {})
		}
	};
};

export const actions: Actions = {
	save: async ({ request, locals, params }) => {
		const { kind, kid, entry } = await stored(locals, params);
		const form = await request.formData();
		const running = isRunning(kind, entry);
		const extra = { form: 'save' as const };
		// Ended elsewhere since the page loaded: a finished read would find no stored seconds to compare.
		if (!running && fromRunningPage(kind, form)) {
			const values = editValues(kind, entry, kid.timeZone, formUnits(locals.session));
			// The finished form stays as advanced as the form that was open.
			const advanced: EntryValues = form.get('advanced') === '1' ? { advanced: '1' } : {};
			return staleTimerFailure({ ...values, ...advanced }, extra);
		}
		const { values, input, field, message } = readEntryForm(kind, form, kid.timeZone, {
			kind: 'edit',
			running
		});
		if (!input) return entryFieldFailure(values, field!, extra, message);
		const r = await locals.api
			.tracker(kind)
			.update(params.kid, params.entry, sendOnlyChanged(kind, form, input));
		if (!r.ok) return entryApiFailure(kind, values, r, extra);
		redirect(303, timelineDayPath(kid.id, localDate(entryInstant(kind, r.data), kid.timeZone)));
	},
	delete: async ({ locals, params }) => {
		const { kind, kid, entry } = await stored(locals, params);
		const r = await locals.api.tracker(kind).remove(params.kid, params.entry);
		if (!r.ok) return failAction(r, { form: 'delete' as const });
		redirect(303, timelineDayPath(kid.id, localDate(entryInstant(kind, entry), kid.timeZone)));
	}
};
