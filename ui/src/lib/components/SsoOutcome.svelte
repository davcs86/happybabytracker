<script lang="ts">
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';

	let {
		heading,
		message = null,
		back
	}: {
		heading: string;
		message?: string | null;
		back: { href: string; label: string };
	} = $props();
</script>

<svelte:head>
	<title>{heading} · happybabytracker</title>
</svelte:head>

<main class="flex min-h-screen flex-col justify-center px-5 py-6">
	<div class="mx-auto flex w-full max-w-[400px] flex-col gap-4">
		<p class="text-base font-bold">Happy baby</p>
		<h1>{heading}</h1>
		{#if message}
			<Alert.Root>
				<Alert.Description>{message}</Alert.Description>
			</Alert.Root>
		{/if}
		<div class="flex flex-wrap gap-2">
			<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- a same-origin path from ssoBack or landing -->
			<Button href={back.href} size="xl">{back.label}</Button>
		</div>
	</div>
</main>
