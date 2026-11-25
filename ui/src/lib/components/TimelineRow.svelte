<script lang="ts" module>
	import type { Component } from 'svelte';
	import DropletIcon from '@lucide/svelte/icons/droplet';
	import HeartIcon from '@lucide/svelte/icons/heart';
	import MilkIcon from '@lucide/svelte/icons/milk';
	import MoonIcon from '@lucide/svelte/icons/moon';
	import NotebookPenIcon from '@lucide/svelte/icons/notebook-pen';
	import UtensilsIcon from '@lucide/svelte/icons/utensils';
	import ArrowUpIcon from '@lucide/svelte/icons/arrow-up';
	import type { TrackerKind } from '#lib/types.js';

	export const TRACKER_VISUAL: Record<TrackerKind, { chip: string; icon: Component }> = {
		bottles: { chip: 'tracker-bottle', icon: MilkIcon },
		nursing: { chip: 'tracker-nursing', icon: HeartIcon },
		pumping: { chip: 'tracker-pumping', icon: ArrowUpIcon },
		sleep: { chip: 'tracker-sleep', icon: MoonIcon },
		diapers: { chip: 'tracker-diaper', icon: DropletIcon },
		meals: { chip: 'tracker-solids', icon: UtensilsIcon },
		symptoms: { chip: 'tracker-symptom', icon: NotebookPenIcon }
	};
</script>

<script lang="ts">
	import { resolve } from '$app/paths';
	import { TRACKER_LABEL } from '#lib/constants.js';
	import { timelineLine } from '#lib/timeline.js';
	import { formatClock } from '#lib/time-zone.js';
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
		units: Pick<Units, 'weight' | 'volume' | 'temp'>;
	} = $props();

	const href = $derived(
		resolve('/(app)/kids/[kid]/log/[tracker=tracker]/[entry]', {
			kid: kidId,
			tracker: item.kind,
			entry: item.entry.id
		})
	);

	const line = $derived(timelineLine(item, zone, units));
	const clock = $derived(formatClock(item.at, zone));
	// The line is "{clock} {text}", or "{tracker} {range}" for a sleep; the pieces are cut from it.
	const text = $derived(line.startsWith(`${clock} `) ? line.slice(clock.length + 1) : line);
	const label = $derived(TRACKER_LABEL[item.kind]);
	const named = $derived(text.startsWith(label));
	const title = $derived(named ? label : text);
	const detail = $derived(named ? text.slice(label.length).replace(/^[:,]?\s*/, '') : '');
	const visual = $derived(TRACKER_VISUAL[item.kind]);
	const Icon = $derived(visual.icon);
	const name = $derived(item.entry.note ? `${line} ${item.entry.note}` : line);
</script>

<li>
	<a
		{href}
		aria-label={name}
		class="flex min-h-[60px] items-center gap-3 rounded-xl border bg-card px-3 py-2 hover:bg-accent"
	>
		<!-- First in the DOM, last on screen: the name leads with the clock, and the visible text must match it. -->
		{#if line.startsWith(`${clock} `)}
			<span class="order-last flex-none font-mono text-[13px] text-muted-foreground">{clock}</span>
		{/if}
		<span class="tracker-chip {visual.chip}"><Icon aria-hidden="true" class="size-[18px]" /></span>
		<span class="min-w-0 flex-1">
			<span class="block text-[14.5px] font-semibold [overflow-wrap:anywhere]">{title}</span>
			{#if detail}
				<span class="block text-[12.5px] [overflow-wrap:anywhere] text-muted-foreground"
					>{detail}</span
				>
			{/if}
			{#if item.entry.note}
				<span class="block text-[12.5px] [overflow-wrap:anywhere] text-muted-foreground"
					>{item.entry.note}</span
				>
			{/if}
		</span>
	</a>
</li>
