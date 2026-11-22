<script lang="ts">
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
		units
	}: { rows: GrowthRow[]; kidId: string; units: Pick<Units, 'weight' | 'length'> } = $props();

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

{#if rows.length === 0}
	<p>No measurements yet</p>
{:else}
	<ul aria-label="Measurements" class="flex flex-col gap-2">
		{#each rows as row (row.id)}
			{@const date = formatDayHeading(row.measuredOn)}
			<li class="rounded-lg border px-3 py-2">
				<div class="flex flex-wrap items-baseline justify-between gap-x-4">
					<span class="font-medium">{date}</span>
					<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- resolve() renders a relative href without JavaScript, and the link keeps an absolute path -->
					<a href={growthEntryPath(kidId, row.id)} class="text-sm underline"
						>Edit<span class="sr-only"> the measurement of {date}</span></a
					>
				</div>
				<p class="text-sm text-muted-foreground">
					{correctedAgeLabel(row.ageDays, row.correctedDays)}
				</p>
				<div class="mt-1 text-sm">
					{#each measures(row) as m (m.measure)}
						<p>{measureLine(m)}</p>
					{/each}
				</div>
				{#if row.note}
					<p class="mt-1 text-sm [overflow-wrap:anywhere]">{row.note}</p>
				{/if}
			</li>
		{/each}
	</ul>
{/if}
