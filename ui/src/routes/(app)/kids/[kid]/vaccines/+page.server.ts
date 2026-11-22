import { fail, redirect } from '@sveltejs/kit';
import { vaccinesPath } from '#lib/paths.js';
import { schemaChosen } from '#lib/vaccines.js';
import { setKidCookie } from '#lib/server/cookie.js';
import { failPage } from '#lib/server/outcome.js';
import {
	DISMISS_FIELD_MESSAGES,
	dismissApiFailure,
	readDismissBeforeForm,
	readDismissForm
} from '#lib/server/vaccine-form.js';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals, params, cookies }) => {
	const [kid, vaccines, schemas, catalogue] = await Promise.all([
		locals.api.getKid(params.kid),
		locals.api.vaccines(params.kid),
		locals.api.vaccineSchemas(),
		locals.api.vaccineCatalogue()
	]);
	if (!kid.ok) failPage(kid);
	if (!vaccines.ok) failPage(vaccines);
	if (!schemas.ok) failPage(schemas);
	if (!catalogue.ok) failPage(catalogue);
	setKidCookie(cookies, kid.data.id);
	return {
		kid: kid.data,
		chosen: schemaChosen(kid.data.vaccineSchema, schemas.data),
		vaccines: vaccines.data,
		catalogue: catalogue.data
	};
};

const lineOf = (form: FormData) => String(form.get('line') ?? '');

export const actions: Actions = {
	dismiss: async ({ request, locals, params }) => {
		const form = await request.formData();
		const extra = { form: 'dismiss' as const };
		const { input, field } = readDismissForm(form);
		if (!input) return fail(422, { ...extra, field, message: DISMISS_FIELD_MESSAGES[field!] });
		const r = await locals.api.setDismissal(params.kid, lineOf(form), input.reason);
		if (!r.ok) return dismissApiFailure(r, extra);
		redirect(303, vaccinesPath(params.kid));
	},
	undoDismiss: async ({ request, locals, params }) => {
		const r = await locals.api.clearDismissal(params.kid, lineOf(await request.formData()));
		if (!r.ok) return dismissApiFailure(r, { form: 'undoDismiss' as const });
		redirect(303, vaccinesPath(params.kid));
	},
	dismissBefore: async ({ request, locals, params }) => {
		const { values, input, field } = readDismissBeforeForm(await request.formData());
		const extra = { form: 'dismissBefore' as const, values };
		if (!input) return fail(422, { ...extra, field, message: DISMISS_FIELD_MESSAGES[field!] });
		const r = await locals.api.dismissBefore(params.kid, input.before, input.reason);
		if (!r.ok) return dismissApiFailure(r, extra);
		return { form: 'dismissBefore' as const, dismissed: r.data };
	}
};
