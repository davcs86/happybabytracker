<script lang="ts">
	import { Button, type ButtonSize } from '#lib/components/ui/button/index.js';
	import { closeDrawer, isEmbed } from '#lib/drawer.js';

	let {
		href,
		size = 'lg',
		class: className = 'flex-1'
	}: { href: string; size?: ButtonSize; class?: string } = $props();
</script>

{#if isEmbed()}
	<Button type="button" variant="outline" size="lg" class="flex-1" onclick={closeDrawer}
		>Cancel</Button
	>
{:else}
	<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- the caller resolves it -->
	<Button {href} variant="outline" {size} class={className}>Cancel</Button>
{/if}
