/* eslint-disable svelte/no-navigation-without-resolve -- hrefs are the ones the page's links already carry */
import { getContext, setContext, type Component } from 'svelte';
import { goto, invalidateAll, preloadData, pushState, replaceState } from '$app/navigation';
import { page } from '$app/state';
import type { SubmitFunction } from '$app/forms';
import { toast } from 'svelte-sonner';

export type DrawerState = { href: string; title: string; data: Record<string, unknown> };
type Loader = () => Promise<{ default: Component<any> }>; // eslint-disable-line @typescript-eslint/no-explicit-any

const KID = '/kids/[^/]+';
// Form pages that may open in a sheet; anything else navigates.
const ROUTES: [RegExp, Loader][] = [
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

// On a form page itself, a link to the same form (the advanced toggle) navigates: never a sheet over its own page.
export const isDrawerLink = (a: HTMLAnchorElement) =>
	!!loaderFor(a.pathname) &&
	(page.state.drawer
		? a.pathname + a.search !== page.url.pathname + page.url.search
		: a.pathname !== page.url.pathname);

export async function openDrawer(a: HTMLAnchorElement): Promise<void> {
	const href = a.pathname + a.search;
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

// The form's submit handler: in a sheet a saved result closes it, refreshes the page and says so.
export const savedAs =
	(message: string): SubmitFunction =>
	() =>
	async ({ result, update }) => {
		if (!page.state.drawer || result.type !== 'redirect') return update({ reset: false });
		closeDrawer();
		for (let i = 0; i < 100 && page.state.drawer; i++) await new Promise((r) => setTimeout(r, 20));
		toast(message);
		await invalidateAll();
	};
