<script lang="ts">
	import PencilIcon from '@lucide/svelte/icons/pencil';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Card } from '#lib/components/ui/card/index.js';
	import { correctedAgeLabel, rowMeasures } from '#lib/growth.js';
	import { growthEntryPath, growthMeasurePath } from '#lib/paths.js';
	import { formatDayHeading } from '#lib/timeline.js';
	import type { GrowthRow, Measure, Units } from '#lib/types.js';

	let {
		row,
		kidId,
		units,
		measure
	}: {
		row: GrowthRow;
		kidId: string;
		units: Pick<Units, 'weight' | 'length'>;
		measure: Measure;
	} = $props();

	const date = $derived(formatDayHeading(row.measuredOn));
</script>

<section aria-labelledby="selected-heading">
	<Card class="gap-2 border-[1.5px] border-primary p-3.5">
		<div class="flex items-center gap-2">
			<div class="min-w-0 flex-1">
				<h2
					id="selected-heading"
					class="text-xs font-semibold tracking-[0.06em] text-muted-foreground uppercase"
				>
					Selected
				</h2>
				<p class="text-[15px] font-semibold">{date}</p>
				<p class="text-[12.5px] text-muted-foreground">
					{correctedAgeLabel(row.ageDays, row.correctedDays)}
				</p>
			</div>
			<Button
				href={growthEntryPath(kidId, row.id)}
				data-drawer="Edit measurement"
				variant="outline"
				size="icon"
				title="Edit the selected measurement"
				><PencilIcon aria-hidden="true" class="size-5" /><span class="sr-only"
					>Edit the selected measurement</span
				></Button
			>
		</div>
		<div class="text-sm">
			{#each rowMeasures(row, units) as m (m.measure)}
				<p class={m.measure === measure ? 'font-semibold' : ''}>{m.line}</p>
			{/each}
		</div>
		{#if row.note}
			<p class="text-[12.5px] [overflow-wrap:anywhere] text-muted-foreground">{row.note}</p>
		{/if}
		<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- the path is built by growthMeasurePath -->
		<a
			href={growthMeasurePath(kidId, measure)}
			class="inline-flex min-h-11 items-center text-sm underline">Clear selection</a
		>
	</Card>
</section>
