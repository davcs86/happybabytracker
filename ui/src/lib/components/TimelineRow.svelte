<script lang="ts">
	import { resolve } from '$app/paths';
	import { timelineLine } from '#lib/timeline.js';
	import type { TimelineItem, Units } from '#lib/types.js';

	let {
		item,
		kidId,
		zone,
		units
	}: {
		item: TimelineItem;
		kidId: string;
		zone: string;
		units: Pick<Units, 'weight' | 'volume'>;
	} = $props();

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
		<span class="block [overflow-wrap:anywhere]">{timelineLine(item, zone, units)}</span>
		{#if item.entry.note}
			<span class="block text-sm [overflow-wrap:anywhere] text-muted-foreground"
				>{item.entry.note}</span
			>
		{/if}
	</a>
</li>
