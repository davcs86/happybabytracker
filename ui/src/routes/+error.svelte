<script lang="ts">
	import { page } from '$app/state';
	import { Button } from '#lib/components/ui/button/index.js';

	const unreachable = $derived(page.status === 503);
	const missing = $derived(page.status === 404);
	const heading = $derived(
		missing
			? 'Not found'
			: unreachable
				? 'The app cannot reach its server right now'
				: 'Something went wrong'
	);
	const lead = $derived(
		missing
			? 'We can’t find that page. It may not exist, or you may not have access to it.'
			: 'Your sign-in is kept. Try again in a moment.'
	);
</script>

<svelte:head>
	<title>{heading} · happybabytracker</title>
</svelte:head>

<main class="flex min-h-screen flex-col justify-center px-5 py-6">
	<div class="mx-auto flex w-full max-w-[400px] flex-col gap-4">
		<p class="text-base font-bold">Happy baby</p>
		<h1>{heading}</h1>
		<p class="text-[14.5px] leading-normal text-muted-foreground">{lead}</p>
		<p class="text-[12.5px] text-muted-foreground">Error {page.status}</p>
		<div class="flex flex-wrap gap-2">
			<!-- eslint-disable svelte/no-navigation-without-resolve -- Home, or a reload of this page -->
			{#if missing}
				<Button href="/" size="xl">Go to Home</Button>
			{:else}
				<Button href={page.url.pathname + page.url.search} size="xl">Try again</Button>
			{/if}
		</div>
	</div>
</main>
