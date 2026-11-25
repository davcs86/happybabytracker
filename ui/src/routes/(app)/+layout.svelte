<script lang="ts">
	import AppShell from '#lib/components/AppShell.svelte';
	import type { LayoutProps } from './$types';

	let { data, children }: LayoutProps = $props();
</script>

<AppShell kids={data.kids} kid={data.kid} modules={data.modules} ages={data.ages} theme={data.theme}
	>{@render children()}</AppShell
>
