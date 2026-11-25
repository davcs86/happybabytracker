<script lang="ts">
	import { page } from '$app/state';
	import AppShell from '#lib/components/AppShell.svelte';
	import type { LayoutProps } from './$types';

	let { data, children }: LayoutProps = $props();
</script>

<!-- An error page stands alone: it never shows a child, so a refused page reads as an unknown one. -->
{#if page.error}
	{@render children()}
{:else}
	<AppShell
		kids={data.kids}
		kid={data.kid}
		modules={data.modules}
		ages={data.ages}
		theme={data.theme}>{@render children()}</AppShell
	>
{/if}
