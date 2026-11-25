<script lang="ts">
	import { Badge } from '#lib/components/ui/badge/index.js';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();
</script>

<div class="flex min-h-screen flex-col md:flex-row">
	<nav
		aria-label="Primary"
		class="sticky top-0 hidden h-screen w-[225px] shrink-0 flex-col gap-1 overflow-y-auto border-r bg-card px-3 py-5 md:flex"
	>
		<p class="px-2.5 pb-3.5 text-base font-bold">Happy baby</p>
	</nav>

	<main class="min-w-0 flex-1 px-4 pt-4 pb-7">
		<div class="xl:mx-auto xl:w-[920px]">
			<header class="flex min-h-12 items-center gap-2">
				<Badge class="ml-auto">Admin</Badge>
			</header>
		</div>
		{@render children()}
	</main>
</div>
