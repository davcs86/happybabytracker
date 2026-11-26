<script lang="ts">
	import PencilIcon from '@lucide/svelte/icons/pencil';
	import { Button } from '#lib/components/ui/button/index.js';
	import { correctedAgeLabel, rowMeasures } from '#lib/growth.js';
	import { growthEntryPath, growthMeasurePath } from '#lib/paths.js';
	import { formatDayHeading } from '#lib/timeline.js';
	import type { GrowthRow, Measure, Units } from '#lib/types.js';

	let {
		rows,
		kidId,
		units,
		measure: current,
		selected = null
	}: {
		rows: GrowthRow[];
		kidId: string;
		units: Pick<Units, 'weight' | 'length'>;
		measure: Measure;
		selected?: string | null;
	} = $props();

	// Rows dated before the expected term are drawn hatched; their lines say so in words.
	const beforeTerm = (r: GrowthRow) => r.correctedDays !== null && r.correctedDays < 0;
</script>

<ul aria-label="Measurements" class="flex flex-col gap-2">
	{#each rows as row (row.id)}
		{@const date = formatDayHeading(row.measuredOn)}
		<li
			class="flex flex-col gap-1 rounded-xl border bg-card p-3 {row.id === selected
				? 'border-[1.5px] border-primary'
				: ''} {beforeTerm(row)
				? 'bg-[repeating-linear-gradient(135deg,transparent_0_8px,var(--border)_8px_9px)]'
				: ''}"
		>
			<div class="flex items-center gap-2">
				<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- the path is built by growthMeasurePath -->
				<a
					href="{growthMeasurePath(kidId, current)}&sel={row.id}"
					aria-current={row.id === selected ? 'true' : undefined}
					class="block min-h-11 min-w-0 flex-1"
				>
					<span class="block text-[15px] font-semibold">{date}</span>
					<span class="block text-[12.5px] text-muted-foreground">
						{correctedAgeLabel(row.ageDays, row.correctedDays)}
					</span>
				</a>
				<Button
					href={growthEntryPath(kidId, row.id)}
					data-drawer="Edit measurement"
					variant="outline"
					size="icon"
					title="Edit the measurement of {date}"
					><PencilIcon aria-hidden="true" class="size-5" /><span class="sr-only"
						>Edit the measurement of {date}</span
					></Button
				>
			</div>
			<div class="text-sm">
				{#each rowMeasures(row, units) as m (m.measure)}
					<p class={m.measure === current ? 'font-semibold' : ''}>{m.line}</p>
				{/each}
			</div>
			{#if row.note}
				<p class="text-[12.5px] [overflow-wrap:anywhere] text-muted-foreground">{row.note}</p>
			{/if}
		</li>
	{/each}
</ul>
