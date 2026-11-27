<script lang="ts" module>
	import type { Component } from 'svelte';
	import DropletIcon from '@lucide/svelte/icons/droplet';
	import HeartIcon from '@lucide/svelte/icons/heart';
	import MilkIcon from '@lucide/svelte/icons/milk';
	import MoonIcon from '@lucide/svelte/icons/moon';
	import NotebookPenIcon from '@lucide/svelte/icons/notebook-pen';
	import UtensilsIcon from '@lucide/svelte/icons/utensils';
	import ArrowUpIcon from '@lucide/svelte/icons/arrow-up';
	import StethoscopeIcon from '@lucide/svelte/icons/stethoscope';
	import type { TimelineKind } from '#lib/types.js';

	export const TIMELINE_VISUAL: Record<TimelineKind, { chip: string; icon: Component }> = {
		bottles: { chip: 'tracker-bottle', icon: MilkIcon },
		nursing: { chip: 'tracker-nursing', icon: HeartIcon },
		pumping: { chip: 'tracker-pumping', icon: ArrowUpIcon },
		sleep: { chip: 'tracker-sleep', icon: MoonIcon },
		diapers: { chip: 'tracker-diaper', icon: DropletIcon },
		meals: { chip: 'tracker-solids', icon: UtensilsIcon },
		symptoms: { chip: 'tracker-symptom', icon: NotebookPenIcon },
		visit: { chip: '[--h:235]', icon: StethoscopeIcon }
	};
</script>

<script lang="ts">
	import { resolve } from '$app/paths';
	import { TRACKER_LABEL, TRACKER_TITLE } from '#lib/constants.js';
	import { timelineLine, visitTimelineLine } from '#lib/timeline.js';
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

	const tracker = $derived(item.kind === 'visit' ? null : item);
	const href = $derived(
		item.kind === 'visit'
			? resolve('/(app)/kids/[kid]/visits/[visit]', { kid: kidId, visit: item.entry.id })
			: resolve('/(app)/kids/[kid]/log/[tracker=tracker]/[entry]', {
					kid: kidId,
					tracker: item.kind,
					entry: item.entry.id
				})
	);

	const clock = $derived(formatClock(item.at, zone));
	const line = $derived(tracker ? timelineLine(tracker, zone, units) : null);
	// A tracker line is "{clock} {text}", or "{tracker} {range}" for a sleep; the pieces are cut from it.
	const clocked = $derived(line === null || line.startsWith(`${clock} `));
	const text = $derived(line === null ? '' : clocked ? line.slice(clock.length + 1) : line);
	const label = $derived(tracker ? TRACKER_LABEL[tracker.kind] : '');
	const named = $derived(text.startsWith(label));
	const visit = $derived(item.kind === 'visit' ? visitTimelineLine(item.entry) : null);
	const title = $derived(visit ? visit.title : named ? label : text);
	const detail = $derived(
		visit ? visit.detail : named ? text.slice(label.length).replace(/^[:,]?\s*/, '') : ''
	);
	const note = $derived(tracker?.entry.note ?? null);
	const visual = $derived(TIMELINE_VISUAL[item.kind]);
	const Icon = $derived(visual.icon);
	// A visit row has no aria-label, so its name is its visible text.
	const name = $derived(line === null ? undefined : note ? `${line} ${note}` : line);
</script>

<li>
	<a
		{href}
		data-drawer={tracker ? TRACKER_TITLE[tracker.kind].edit : undefined}
		aria-label={name}
		class="flex min-h-[60px] items-center gap-3 rounded-xl border bg-card px-3 py-2 hover:bg-accent"
	>
		<!-- First in the DOM, last on screen: the name leads with the clock, and the visible text must match it. -->
		{#if clocked}
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
			{#if note}
				<span class="block text-[12.5px] [overflow-wrap:anywhere] text-muted-foreground"
					>{note}</span
				>
			{/if}
		</span>
	</a>
</li>
