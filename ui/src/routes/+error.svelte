<script lang="ts">
	import { page } from '$app/state';

	const heading = $derived(
		page.status === 404
			? 'Not found'
			: page.status === 503
				? 'The app cannot reach its server right now'
				: 'Something went wrong'
	);
</script>

<svelte:head>
	<title>{heading} · happybabytracker</title>
</svelte:head>

<main class="mx-auto flex w-full max-w-xl flex-col items-start gap-4 p-4">
	<h1 class="text-lg font-semibold">{heading}</h1>
	<p class="text-muted-foreground">
		{page.status === 503 ? 'Try again in a moment.' : `Error ${page.status}.`}
	</p>
	<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- reloads the current page -->
	<a class="underline" href={page.url.pathname + page.url.search}>Try again</a>
</main>
