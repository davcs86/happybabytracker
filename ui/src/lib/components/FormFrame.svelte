<script lang="ts">
	import { isEmbed } from '#lib/drawer.js';
	import type { Snippet } from 'svelte';

	let {
		documentTitle,
		chrome,
		class: className = '',
		children
	}: { documentTitle: string; chrome: Snippet; class?: string; children: Snippet } = $props();

	const embed = isEmbed();
</script>

<svelte:head>
	{#if !embed}<title>{documentTitle}</title>{/if}
</svelte:head>

{#if embed}
	<div class="flex flex-col gap-3.5 *:mt-0!">{@render children()}</div>
{:else}
	<div class="max-w-[448px] xl:mx-auto xl:w-[920px] xl:max-w-none {className}">
		{@render chrome()}
		{@render children()}
	</div>
{/if}
