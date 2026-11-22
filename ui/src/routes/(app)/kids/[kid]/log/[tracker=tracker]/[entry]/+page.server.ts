import { redirect } from '@sveltejs/kit';
import {
	editValues,
	entryInstant,
	isRunning,
	localDate,
	ranOver,
	trackerOf
} from '#lib/entry-edit.js';
import { timelineDayPath } from '#lib/paths.js';
import { setKidCookie } from '#lib/server/cookie.js';
import { entryApiFailure, entryFieldFailure } from '#lib/server/entry-failure.js';
import { readEntryForm } from '#lib/server/entry-form.js';
import { sendOnlyChanged } from '#lib/server/entry-submit.js';
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

export const load: PageServerLoad = async ({ locals, params, cookies, parent }) => {
	const [{ kind, kid, entry }, { session }] = await Promise.all([stored(locals, params), parent()]);
	setKidCookie(cookies, kid.id);
	const running = isRunning(kind, entry);
	return {
		kid,
		kind,
		running,
		overrun: running && ranOver(entryInstant(kind, entry), Date.now()),
		values: editValues(kind, entry, kid.timeZone, session?.parent?.units.volume ?? 'ml')
	};
};

export const actions: Actions = {
	save: async ({ request, locals, params }) => {
		const { kind, kid, entry } = await stored(locals, params);
		const form = await request.formData();
		const { values, input, field } = readEntryForm(kind, form, kid.timeZone, {
			kind: 'edit',
			running: isRunning(kind, entry)
		});
		const extra = { form: 'save' as const };
		if (!input) return entryFieldFailure(values, field!, extra);
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
