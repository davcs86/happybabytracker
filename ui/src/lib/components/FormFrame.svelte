<script lang="ts">
	import { isEmbed } from '#lib/drawer.js';
	import type { Snippet } from 'svelte';

	let {
		title,
		documentTitle,
		backHref,
		backLabel,
		children
	}: {
		title: string;
		documentTitle: string;
		backHref: string;
		backLabel: string;
		children: Snippet;
	} = $props();

	const embed = isEmbed();
</script>

<svelte:head>
	{#if !embed}<title>{documentTitle}</title>{/if}
</svelte:head>

{#if embed}
	{@render children()}
{:else}
	<div class="max-w-[448px] xl:mx-auto xl:w-[920px] xl:max-w-none">
		<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- the caller resolves it -->
		<a href={backHref} class="inline-flex min-h-11 items-center gap-1 text-sm underline"
			><span aria-hidden="true">‹</span>{backLabel}</a
		>
		<h1 class="mt-1">{title}</h1>
		{@render children()}
	</div>
{/if}
