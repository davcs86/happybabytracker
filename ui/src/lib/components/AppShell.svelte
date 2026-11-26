<script lang="ts">
	import { page } from '$app/state';
	import type { Snippet } from 'svelte';
	import { navModel, themeLabel, MORE_ICON, type NavEntry } from '#lib/nav.js';
	import { THEMES, type Theme } from '#lib/constants.js';
	import { applyTheme, isTheme, THEME_STORAGE_KEY } from '#lib/theme-client.js';
	import type { Kid, ModuleState } from '#lib/types.js';
	import ChildSwitcherSheet from './ChildSwitcherSheet.svelte';
	import DrawerHost from './DrawerHost.svelte';
	import MoreSheet from './MoreSheet.svelte';
	import PageHeader from './PageHeader.svelte';
	import ThemeSheet from './ThemeSheet.svelte';
	import { Toaster } from './ui/sonner/index.js';

	let {
		kids,
		kid,
		modules,
		ages,
		theme: initialTheme,
		children
	}: {
		kids: Kid[];
		kid: Kid | null;
		modules: ModuleState[] | null;
		ages: Record<string, string>;
		theme: Theme;
		children: Snippet;
	} = $props();

	let chosen = $state<Theme | null>(null);
	const theme = $derived(chosen ?? initialTheme);
	let moreOpen = $state(false);
	let switchOpen = $state(false);
	let themeOpen = $state(false);

	const nav = $derived(
		navModel({
			pathname: page.url.pathname,
			kidId: kid?.id ?? null,
			role: kid?.role ?? null,
			modules,
			theme
		})
	);
	const nextTheme = $derived(THEMES[(THEMES.indexOf(theme) + 1) % THEMES.length]);

	$effect(() => {
		void page.url.pathname;
		moreOpen = switchOpen = themeOpen = false;
	});
	$effect(() => {
		const onStorage = (e: StorageEvent) => {
			if (e.key === THEME_STORAGE_KEY && isTheme(e.newValue)) {
				chosen = e.newValue;
				applyTheme(e.newValue);
			}
		};
		addEventListener('storage', onStorage);
		return () => removeEventListener('storage', onStorage);
	});

	const SIDE =
		'flex min-h-11 w-full items-center gap-2.5 rounded-[10px] px-2.5 text-left text-sm font-semibold hover:bg-accent aria-[current=page]:bg-background aria-[current=page]:text-primary';
	const openTheme = () => {
		moreOpen = false;
		themeOpen = true;
	};
</script>

<!-- eslint-disable svelte/no-navigation-without-resolve -- hrefs come from the nav model -->
{#snippet sideEntry(e: NavEntry)}
	{#if e.kind === 'link'}
		<a href={e.href} aria-current={e.current ? 'page' : undefined} class={SIDE}>
			<e.icon aria-hidden="true" class="size-[18px] shrink-0" />{e.label}
		</a>
	{:else if e.kind === 'theme'}
		<form method="POST" action="/theme" class="contents">
			<input type="hidden" name="theme" value={nextTheme} />
			<input type="hidden" name="return" value={page.url.pathname + page.url.search} />
			<button
				type="submit"
				class={SIDE}
				onclick={(ev) => {
					ev.preventDefault();
					themeOpen = true;
				}}
			>
				<e.icon aria-hidden="true" class="size-[18px] shrink-0" />{e.label}
			</button>
		</form>
	{:else}
		<form method="POST" action="/logout" class="contents">
			<button type="submit" class={SIDE}>
				<e.icon aria-hidden="true" class="size-[18px] shrink-0" />{e.label}
			</button>
		</form>
	{/if}
{/snippet}

<div class="flex min-h-screen flex-col md:flex-row">
	<nav
		aria-label="Primary"
		data-testid="side-nav"
		class="sticky top-0 hidden h-screen w-[225px] shrink-0 flex-col gap-1 overflow-y-auto border-r bg-card px-3 py-5 md:flex"
	>
		<p class="px-2.5 pb-3.5 text-base font-bold">Happy baby</p>
		{#each nav.primary as e (e.id)}
			<a href={e.href} aria-current={e.current ? 'page' : undefined} class={SIDE}>
				<e.icon aria-hidden="true" class="size-[18px] shrink-0" />{e.label}
			</a>
		{/each}
		<hr class="my-1.5 border-border" />
		{#each nav.more as e (e.id)}
			{@render sideEntry(e)}
		{/each}
	</nav>

	<main class="min-w-0 flex-1 px-4 pt-4 pb-[calc(6rem+env(safe-area-inset-bottom))] md:pb-7">
		<div class="xl:mx-auto xl:w-[920px]">
			<PageHeader {kid} onswitch={() => (switchOpen = true)} />
		</div>
		{@render children()}
		<noscript>
			<nav aria-label="More" class="mt-6 grid gap-1 border-t pt-4 md:hidden">
				{#each nav.more as e (e.id)}
					{#if e.kind === 'link'}
						<a href={e.href} class="flex min-h-11 items-center font-semibold">{e.label}</a>
					{/if}
				{/each}
				<form method="POST" action="/theme" class="flex">
					<input type="hidden" name="return" value={page.url.pathname + page.url.search} />
					{#each THEMES as t (t)}
						<button type="submit" name="theme" value={t} class="min-h-11 flex-1 font-semibold"
							>{themeLabel(t)}</button
						>
					{/each}
				</form>
				<form method="POST" action="/logout">
					<button type="submit" class="min-h-11 font-semibold">Sign out</button>
				</form>
			</nav>
		</noscript>
	</main>

	<nav
		aria-label="Primary"
		class="fixed inset-x-0 bottom-0 z-10 grid grid-cols-4 gap-0 border-t bg-card px-1.5 pt-1.5 pb-[calc(10px+env(safe-area-inset-bottom))] md:hidden"
	>
		{#each nav.primary as e (e.id)}
			<a
				href={e.href}
				aria-current={e.current ? 'page' : undefined}
				class="flex min-h-13 flex-col items-center justify-center gap-0.5 text-xs font-semibold {e.current
					? 'text-primary'
					: 'text-muted-foreground'}"
			>
				<e.icon aria-hidden="true" class="size-[22px]" />{e.label}
			</a>
		{/each}
		<button
			type="button"
			aria-haspopup="dialog"
			class="flex min-h-13 flex-col items-center justify-center gap-0.5 text-xs font-semibold text-muted-foreground"
			onclick={() => (moreOpen = true)}
		>
			<MORE_ICON aria-hidden="true" class="size-[22px]" />More
		</button>
	</nav>
</div>

<MoreSheet bind:open={moreOpen} entries={nav.more} ontheme={openTheme} />
<ChildSwitcherSheet bind:open={switchOpen} {kids} {ages} currentId={kid?.id ?? null} />
<ThemeSheet bind:open={themeOpen} {theme} onchoose={(t) => (chosen = t)} />
<DrawerHost />
<Toaster />
