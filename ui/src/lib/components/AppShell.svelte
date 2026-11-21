<script lang="ts" module>
	import type { Component, Snippet } from 'svelte';

	export type NavLink = { href: string; label: string; icon: Component };
</script>

<script lang="ts">
	import LogOutIcon from '@lucide/svelte/icons/log-out';
	import { page } from '$app/state';
	import { Button } from '#lib/components/ui/button/index.js';

	let { links, children }: { links: NavLink[]; children: Snippet } = $props();

	const current = (href: string) =>
		page.url.pathname === href || (href !== '/' && page.url.pathname.startsWith(`${href}/`));
</script>

<div class="flex min-h-screen flex-col md:flex-row">
	<!-- One nav for both widths: a bottom tab bar below 768 px, a side column from 768 px. -->
	<nav
		aria-label="Main"
		class="fixed inset-x-0 bottom-0 z-10 border-t bg-background md:sticky md:top-0 md:h-screen md:w-56 md:shrink-0 md:border-t-0 md:border-r"
		data-testid="main-nav"
	>
		<p class="hidden px-4 py-4 font-semibold md:block">happybabytracker</p>
		<ul class="flex md:flex-col md:gap-1 md:px-2">
			{#each links as link (link.href)}
				{@const active = current(link.href)}
				<li class="flex-1 md:flex-none">
					<!-- eslint-disable svelte/no-navigation-without-resolve -- hrefs come from the layouts' fixed link lists -->
					<a
						href={link.href}
						aria-current={active ? 'page' : undefined}
						class="flex flex-col items-center gap-1 px-2 py-2 text-xs md:flex-row md:gap-2 md:rounded-md md:px-3 md:text-sm {active
							? 'font-semibold text-foreground md:bg-muted'
							: 'text-muted-foreground hover:text-foreground'}"
					>
						<link.icon class="size-5 md:size-4" aria-hidden="true" />
						{link.label}
					</a>
					<!-- eslint-enable svelte/no-navigation-without-resolve -->
				</li>
			{/each}
		</ul>
	</nav>

	<div class="flex min-w-0 flex-1 flex-col pb-20 md:pb-0">
		<header class="flex items-center gap-2 border-b px-4 py-2">
			<p class="mr-auto font-semibold md:invisible">happybabytracker</p>
			<form method="POST" action="/logout">
				<Button type="submit" variant="ghost" size="sm">
					<LogOutIcon aria-hidden="true" />
					Sign out
				</Button>
			</form>
		</header>
		<main class="flex-1 p-4">
			{@render children()}
		</main>
	</div>
</div>
