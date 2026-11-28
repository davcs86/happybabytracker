<script lang="ts">
	import { page } from '$app/state';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { buttonVariants } from '#lib/components/ui/button/index.js';
	import { navModel, SIDE_ENTRY_CLASS } from '#lib/nav.js';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();

	const nav = $derived(navModel({ admin: true, pathname: page.url.pathname }));
</script>

<!-- eslint-disable svelte/no-navigation-without-resolve -- hrefs come from the nav model -->
<div class="flex min-h-screen flex-col md:flex-row">
	<nav
		aria-label="Primary"
		class="sticky top-0 hidden h-screen w-[225px] shrink-0 flex-col gap-1 overflow-y-auto border-r bg-card px-3 py-5 md:flex"
	>
		<p class="px-2.5 pb-3.5 text-base font-bold">Happy baby</p>
		{#each nav.more as e (e.id)}
			<a href={e.href} aria-current={e.current ? 'page' : undefined} class={SIDE_ENTRY_CLASS}>
				<e.icon aria-hidden="true" class="size-[18px] shrink-0" />{e.label}
			</a>
		{/each}
	</nav>

	<main class="min-w-0 flex-1 px-4 pt-4 pb-7">
		<div class="xl:mx-auto xl:w-[920px]">
			<header class="flex min-h-12 items-center gap-2">
				<div class="ml-auto flex items-center gap-2">
					{#each nav.more as e (e.id)}
						<a
							href={e.href}
							aria-label={e.label}
							title={e.label}
							aria-current={e.current ? 'page' : undefined}
							class="{buttonVariants({ variant: 'outline', size: 'icon' })} md:hidden {e.current
								? 'border-primary text-primary'
								: ''}"
						>
							<e.icon aria-hidden="true" data-chrome class="size-5" />
						</a>
					{/each}
					<Badge>Admin</Badge>
				</div>
			</header>
		</div>
		{@render children()}
	</main>
</div>
