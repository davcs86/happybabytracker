<script lang="ts">
	import { SECTION_HEADING } from '#lib/components/home.js';
	import TimelineRow from '#lib/components/TimelineRow.svelte';
	import { timelinePath } from '#lib/paths.js';
	import type { TimelineItem, Units } from '#lib/types.js';

	let {
		items,
		kidId,
		zone,
		units
	}: {
		items: TimelineItem[];
		kidId: string;
		zone: string;
		units: Pick<Units, 'weight' | 'volume' | 'temp'>;
	} = $props();
</script>

<section aria-labelledby="latest-heading" class="flex flex-col gap-2.5">
	<div class="flex items-center">
		<h2 id="latest-heading" class="{SECTION_HEADING} flex-1">Latest</h2>
		<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- the timeline path is built by timelinePath -->
		<a href={timelinePath(kidId)} class="inline-flex min-h-11 items-center text-sm underline"
			>See the Timeline</a
		>
	</div>
	{#if items.length === 0}
		<p
			class="rounded-2xl border border-dashed bg-card p-4 text-[13.5px] leading-normal text-muted-foreground"
		>
			Nothing logged yet. Tap a tile above to add the first entry.
		</p>
	{:else}
		<ul class="flex flex-col gap-1.5">
			{#each items as item (item.entry.id)}
				<TimelineRow {item} {kidId} {zone} {units} />
			{/each}
		</ul>
	{/if}
</section>
