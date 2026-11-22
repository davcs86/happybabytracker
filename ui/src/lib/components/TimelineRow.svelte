<script lang="ts">
	import { resolve } from '$app/paths';
	import { timelineLine } from '#lib/timeline.js';
	import type { TimelineItem, VolumeUnit } from '#lib/types.js';

	let {
		item,
		kidId,
		zone,
		unit
	}: { item: TimelineItem; kidId: string; zone: string; unit: VolumeUnit } = $props();

	const href = $derived(
		resolve('/(app)/kids/[kid]/log/[tracker=tracker]/[entry]', {
			kid: kidId,
			tracker: item.kind,
			entry: item.entry.id
		})
	);
</script>

<li class="rounded-lg border">
	<a {href} class="block min-h-11 rounded-lg px-3 py-2 hover:bg-muted">
		<span class="block">{timelineLine(item, zone, unit)}</span>
		{#if item.entry.note}
			<span class="block text-sm text-muted-foreground">{item.entry.note}</span>
		{/if}
	</a>
</li>
