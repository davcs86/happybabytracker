<script lang="ts">
	import PencilIcon from '@lucide/svelte/icons/pencil';
	import { Button } from '#lib/components/ui/button/index.js';
	import { MEASURE_LABEL } from '#lib/growth-chart.js';
	import {
		correctedAgeLabel,
		formatLength,
		formatPercentile,
		formatWeight,
		formatZ
	} from '#lib/growth.js';
	import { growthEntryPath } from '#lib/paths.js';
	import { formatDayHeading } from '#lib/timeline.js';
	import type { GrowthRow, GrowthScore, Measure, Units } from '#lib/types.js';

	let {
		rows,
		kidId,
		units,
		measure: current
	}: {
		rows: GrowthRow[];
		kidId: string;
		units: Pick<Units, 'weight' | 'length'>;
		measure: Measure;
	} = $props();

	// Rows dated before the expected term are drawn hatched; their lines say so in words.
	const beforeTerm = (r: GrowthRow) => r.correctedDays !== null && r.correctedDays < 0;

	const REASON = {
		before_term: 'before expected term',
		not_covered: 'this source has no table for this age'
	} as const;

	const scoreText = (s: GrowthScore) =>
		s.scored ? `z ${formatZ(s.z)}, percentile ${formatPercentile(s.percentile)}` : REASON[s.reason];

	const measureLine = (m: { measure: Measure; value: string; score: GrowthScore | null }) =>
		`${MEASURE_LABEL[m.measure]} ${m.value}${m.score ? `: ${scoreText(m.score)}` : ''}`;

	function measures(r: GrowthRow) {
		const out: { measure: Measure; value: string; score: GrowthScore | null }[] = [];
		if (r.weight) {
			out.push({
				measure: 'weight',
				value: formatWeight(r.weight, units.weight),
				score: r.scores.weight
			});
		}
		if (r.length) {
			out.push({
				measure: 'length',
				value: formatLength(r.length, units.length),
				score: r.scores.length
			});
		}
		if (r.head) {
			out.push({
				measure: 'head',
				value: formatLength(r.head, units.length),
				score: r.scores.head
			});
		}
		// BMI has no imperial unit.
		if (r.scores.bmi) {
			out.push({ measure: 'bmi', value: String(r.scores.bmi.value), score: r.scores.bmi });
		}
		return out;
	}
</script>

<ul aria-label="Measurements" class="flex flex-col gap-2">
	{#each rows as row (row.id)}
		{@const date = formatDayHeading(row.measuredOn)}
		<li
			class="flex flex-col gap-1 rounded-xl border bg-card p-3 {beforeTerm(row)
				? 'bg-[repeating-linear-gradient(135deg,transparent_0_8px,var(--border)_8px_9px)]'
				: ''}"
		>
			<div class="flex items-center gap-2">
				<div class="min-w-0 flex-1">
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
					title="Edit the measurement of {date}"
					><PencilIcon aria-hidden="true" class="size-5" /><span class="sr-only"
						>Edit the measurement of {date}</span
					></Button
				>
			</div>
			<div class="text-sm">
				{#each measures(row) as m (m.measure)}
					<p class={m.measure === current ? 'font-semibold' : ''}>{measureLine(m)}</p>
				{/each}
			</div>
			{#if row.note}
				<p class="text-[12.5px] [overflow-wrap:anywhere] text-muted-foreground">{row.note}</p>
			{/if}
		</li>
	{/each}
</ul>
