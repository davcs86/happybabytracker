/* eslint-disable svelte/no-navigation-without-resolve -- hrefs are the ones the page's links already carry */
import { getContext, setContext, type Component } from 'svelte';
import { goto, invalidateAll, preloadData, pushState, replaceState } from '$app/navigation';
import { page } from '$app/state';
import { deserialize, type SubmitFunction } from '$app/forms';
import { toast } from 'svelte-sonner';
import { TOAST_FAILURE_MS, TOAST_MS, UNDO_FAILED } from './constants';

export type DrawerState = { href: string; title: string; data: Record<string, unknown> };
type Loader = () => Promise<{ default: Component<any> }>; // eslint-disable-line @typescript-eslint/no-explicit-any

const KID = '/kids/[^/]+';
// Form pages that may open in a sheet; anything else navigates.
const ROUTES: [RegExp, Loader][] = [
	[new RegExp('^/kids/new$'), () => import('../routes/(app)/kids/new/+page.svelte')],
	[
		new RegExp(`^${KID}/settings$`),
		() => import('../routes/(app)/kids/[kid]/settings/+page.svelte')
	],
	[new RegExp(`^${KID}/modules$`), () => import('../routes/(app)/kids/[kid]/modules/+page.svelte')],
	[new RegExp(`^${KID}/access$`), () => import('../routes/(app)/kids/[kid]/access/+page.svelte')],
	[
		new RegExp(`^${KID}/growth/new$`),
		() => import('../routes/(app)/kids/[kid]/growth/new/+page.svelte')
	],
	[
		new RegExp(`^${KID}/growth/[^/]+$`),
		() => import('../routes/(app)/kids/[kid]/growth/[entry]/+page.svelte')
	],
	[
		new RegExp(`^${KID}/vaccines/new$`),
		() => import('../routes/(app)/kids/[kid]/vaccines/new/+page.svelte')
	],
	[
		new RegExp(`^${KID}/vaccines/[^/]+$`),
		() => import('../routes/(app)/kids/[kid]/vaccines/[administration]/+page.svelte')
	],
	[
		new RegExp(`^${KID}/medication/new$`),
		() => import('../routes/(app)/kids/[kid]/medication/new/+page.svelte')
	],
	[
		new RegExp(`^${KID}/medication/doses/[^/]+$`),
		() => import('../routes/(app)/kids/[kid]/medication/doses/[dose]/+page.svelte')
	],
	[
		new RegExp(`^${KID}/medication/[^/]+$`),
		() => import('../routes/(app)/kids/[kid]/medication/[medication]/+page.svelte')
	],
	[
		new RegExp(`^${KID}/visits/new$`),
		() => import('../routes/(app)/kids/[kid]/visits/new/+page.svelte')
	],
	[
		new RegExp(`^${KID}/visits/[^/]+/edit$`),
		() => import('../routes/(app)/kids/[kid]/visits/[visit]/edit/+page.svelte')
	],
	[
		new RegExp(`^${KID}/visits/[^/]+/outcome$`),
		() => import('../routes/(app)/kids/[kid]/visits/[visit]/outcome/+page.svelte')
	],
	[
		new RegExp(`^${KID}/milestones/new$`),
		() => import('../routes/(app)/kids/[kid]/milestones/new/+page.svelte')
	],
	[
		new RegExp(`^${KID}/milestones/[^/]+$`),
		() => import('../routes/(app)/kids/[kid]/milestones/[entry]/+page.svelte')
	],
	[
		new RegExp(`^${KID}/log/[^/]+/[^/]+$`),
		() => import('../routes/(app)/kids/[kid]/log/[tracker=tracker]/[entry]/+page.svelte')
	],
	[
		new RegExp(`^${KID}/log/[^/]+$`),
		() => import('../routes/(app)/kids/[kid]/log/[tracker=tracker]/+page.svelte')
	]
];

const loaderFor = (pathname: string) => ROUTES.find(([re]) => re.test(pathname))?.[1];
export const drawerComponent = (href: string) =>
	loaderFor(new URL(href, 'http://x').pathname)?.().then((m) => m.default);

const EMBED = Symbol('embed');
export const provideEmbed = () => setContext(EMBED, true);
export const isEmbed = () => getContext<boolean>(EMBED) === true;

export const closeDrawer = () => history.back();

export const isDrawerLink = (a: HTMLAnchorElement) =>
	!!loaderFor(a.pathname) && a.pathname + a.search !== page.url.pathname + page.url.search;

export async function openDrawer(a: HTMLAnchorElement): Promise<void> {
	const href = a.pathname + a.search + a.hash;
	const [result] = await Promise.all([preloadData(href), drawerComponent(href)]);
	if (result.type !== 'loaded' || result.status !== 200) {
		await goto(href);
		return;
	}
	const state = { drawer: { href, title: a.dataset.drawer ?? '', data: result.data } };
	// A link inside an open sheet swaps its page: one Back still closes it.
	if (page.state.drawer) replaceState(href, state);
	else pushState(href, state);
}

async function closeAndWait() {
	closeDrawer();
	for (let i = 0; i < 100 && page.state.drawer; i++) await new Promise((r) => setTimeout(r, 20));
}

// A sheet shows the data it was opened with: reload it after a change that leaves the sheet open.
async function refreshDrawer(d: DrawerState | undefined) {
	if (!d) return;
	const result = await preloadData(d.href);
	if (result.type === 'loaded' && result.status === 200) {
		replaceState(d.href, { drawer: { ...d, data: result.data } });
	}
}

type Options = {
	// A redirect result goes to its page instead of back to the one under the sheet.
	follow?: boolean;
	// A success result leaves the sheet open, with its data reloaded.
	stay?: boolean;
	// These actions' success results are applied as a failure's: no reload, no close, no toast.
	inPlace?: string[];
	// The inverse of the change, read when Undo is tapped; with it the toast offers Undo.
	undo?: (form: FormData) => Undo | null | Promise<Undo | null>;
	// The same, worked out when the form is submitted: for a change that closes the sheet it was read from.
	restore?: (form: FormData) => Undo | null;
	// What the toast says once the change is taken back.
	undone?: string;
};

// A change the toast can take back: the inverse is one more form action, posted as the page's own forms are.
export type Undo = { action: string; fields: Record<string, string> };

async function takeBack(options: Options, form: FormData, eager: Undo | null) {
	try {
		const undo = eager ?? (options.undo ? await options.undo(form) : null);
		if (undo) {
			const body = new FormData();
			for (const [name, value] of Object.entries(undo.fields)) body.set(name, value);
			const res = await fetch(undo.action, {
				method: 'POST',
				body,
				headers: { 'x-sveltekit-action': 'true' }
			});
			const result = deserialize(await res.text());
			if (result.type === 'success' || result.type === 'redirect') {
				// invalidateAll drops the sheet's state: it is put back with its data reloaded, as a save does.
				const open = page.state.drawer;
				await invalidateAll();
				await refreshDrawer(open);
				toast.dismiss();
				toast(options.undone ?? 'Undone', { duration: TOAST_MS });
				return;
			}
		}
	} catch {
		// A network failure is the same failure to the person.
	}
	toast.dismiss();
	toast(UNDO_FAILED, { duration: TOAST_FAILURE_MS });
}

// The inverse is worked out when Undo is tapped, from the page as it is by then.
function say(message: string, options: Options, form: FormData, eager: Undo | null) {
	// One toast at a time: a new one replaces the old.
	toast.dismiss();
	if (!eager && !options.undo) return void toast(message);
	toast(message, {
		duration: TOAST_MS,
		action: { label: 'Undo', onClick: () => void takeBack(options, form, eager) }
	});
}

// The form's submit handler: in a sheet a saved result closes it, refreshes the page and says so.
export const savedAs =
	(message: string, options: Options = {}): SubmitFunction =>
	({ action, formData }) => {
		const eager = options.restore?.(formData) ?? null;
		return async ({ result, update }) => {
			const { follow = false, stay = false, inPlace = [] } = options;
			const open = page.state.drawer;
			if (result.type === 'success' && inPlace.includes(action.search.replace(/^\?\//, ''))) {
				return update({ reset: false, invalidateAll: false });
			}
			if (!open || (result.type !== 'redirect' && result.type !== 'success')) {
				return update({ reset: false });
			}
			if (result.type === 'success') {
				// update() refreshes the page itself.
				await update({ reset: false });
				if (stay) await refreshDrawer(open);
				else await closeAndWait();
				say(message, options, formData, eager);
				return;
			}
			await closeAndWait();
			if (follow) await goto(result.location, { invalidateAll: true });
			else await invalidateAll();
			say(message, options, formData, eager);
		};
	};
