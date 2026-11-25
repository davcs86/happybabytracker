<script lang="ts">
	import ChevronDownIcon from '@lucide/svelte/icons/chevron-down';
	import { CHART_VALUE_DECIMALS } from '#lib/constants.js';
	import { buildChart, MEASURE_LABEL, type ChartInput } from '#lib/growth-chart.js';
	import { formatDayHeading } from '#lib/timeline.js';

	let {
		curves,
		rows,
		measure,
		units,
		sourceName,
		kidName
	}: ChartInput & { sourceName: string; kidName: string } = $props();

	const chart = $derived(buildChart({ curves, rows, measure, units }));
	const title = $derived(`${MEASURE_LABEL[measure]} for age, ${sourceName}`);
	const desc = $derived(
		(chart.lines.length > 0
			? `Percentile lines ${chart.lines.map((l) => l.label).join(', ')}`
			: 'No percentile lines') +
			`, ${chart.xLabel.toLowerCase()} on the horizontal axis, ${chart.yLabel} on the vertical axis; ` +
			`${chart.points.length} ${chart.points.length === 1 ? 'measurement' : 'measurements'} plotted. ` +
			'The values are in the chart data below.'
	);
	const shown = (v: number | null) => (v === null ? '–' : v.toFixed(CHART_VALUE_DECIMALS));
</script>

<figure class="m-0 flex flex-col gap-2 rounded-2xl border bg-card px-2 pt-3 pb-2">
	<svg
		role="img"
		aria-labelledby="growth-chart-title"
		aria-describedby="growth-chart-desc"
		viewBox="0 0 {chart.width} {chart.height}"
		class="block h-auto w-full text-[11px]"
	>
		<title id="growth-chart-title">{title}</title>
		<desc id="growth-chart-desc">{desc}</desc>
		<text x={chart.plot.left} y={chart.plot.top - 10} class="fill-muted-foreground"
			>{chart.yLabel}</text
		>
		{#each chart.yTicks as t (t.value)}
			<line
				x1={chart.plot.left}
				x2={chart.plot.right}
				y1={t.pos}
				y2={t.pos}
				class="stroke-border"
			/>
			<text
				x={chart.plot.left - 4}
				y={t.pos}
				text-anchor="end"
				dominant-baseline="middle"
				class="fill-muted-foreground">{t.value}</text
			>
		{/each}
		{#each chart.xTicks as t (t.value)}
			<line
				x1={t.pos}
				x2={t.pos}
				y1={chart.plot.top}
				y2={chart.plot.bottom}
				class="stroke-border"
			/>
			<text x={t.pos} y={chart.plot.bottom + 12} text-anchor="middle" class="fill-muted-foreground"
				>{t.value}</text
			>
		{/each}
		<text
			x={(chart.plot.left + chart.plot.right) / 2}
			y={chart.height - 6}
			text-anchor="middle"
			class="fill-muted-foreground">{chart.xLabel}</text
		>
		{#each chart.lines as line (line.percentile)}
			<path
				data-line={line.percentile}
				d={line.d}
				fill="none"
				stroke-dasharray={line.dash || undefined}
				class="stroke-muted-foreground"
				stroke-width="1"
			/>
			<text
				x={chart.plot.right + 4}
				y={line.labelY}
				dominant-baseline="middle"
				class="fill-muted-foreground">{line.label}</text
			>
		{/each}
		{#if chart.path}
			<path d={chart.path} fill="none" class="stroke-primary" stroke-width="2" />
		{/if}
		{#each chart.points as p (p.id)}
			<circle data-point cx={p.cx} cy={p.cy} r="4" class="fill-primary stroke-background" />
		{/each}
	</svg>
	<p class="flex flex-wrap gap-x-3.5 gap-y-1.5 px-2 pb-1 text-xs text-muted-foreground">
		<span class="inline-flex items-center gap-1.5">
			<svg width="22" height="8" aria-hidden="true"
				><line x1="0" y1="4" x2="22" y2="4" stroke="currentColor" stroke-width="2.2" /></svg
			>{kidName}
		</span>
		{#if chart.lines.length > 0}
			<span class="inline-flex items-center gap-1.5">
				<svg width="22" height="8" aria-hidden="true"
					><line
						x1="0"
						y1="4"
						x2="22"
						y2="4"
						stroke="currentColor"
						stroke-width="1.4"
						stroke-dasharray="4 3"
					/></svg
				>{sourceName} percentiles: {chart.lines.map((l) => l.percentile).join(', ')}
			</span>
		{/if}
	</p>
</figure>

<details class="group text-[13px]">
	<summary
		class="flex min-h-12 cursor-pointer list-none items-center gap-1.5 text-sm font-semibold [&::-webkit-details-marker]:hidden"
		>Chart data<ChevronDownIcon
			aria-hidden="true"
			class="size-4 flex-none text-muted-foreground group-open:rotate-180 motion-safe:transition-transform"
		/></summary
	>
	<div class="flex flex-col gap-3">
		<div class="overflow-x-auto rounded-lg border bg-card">
			<table class="w-full text-left">
				<caption class="px-3 pt-2 text-left font-semibold">Measurements plotted</caption>
				<thead class="text-muted-foreground">
					<tr>
						<th scope="col" class="px-3 py-2 font-semibold">Date</th>
						<th scope="col" class="px-3 py-2 font-semibold">{chart.pointAgeLabel}</th>
						<th scope="col" class="px-3 py-2 font-semibold">{chart.yLabel}</th>
					</tr>
				</thead>
				<tbody>
					{#each chart.points as p (p.id)}
						<tr class="border-t">
							<td class="px-3 py-2">{formatDayHeading(p.measuredOn)}</td>
							<td class="px-3 py-2">{shown(p.x)}</td>
							<td class="px-3 py-2">{shown(p.y)}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
		{#if chart.table.lines.length > 0}
			<div class="overflow-x-auto rounded-lg border bg-card">
				<table class="w-full text-left">
					<caption class="px-3 pt-2 text-left font-semibold">Percentile values</caption>
					<thead class="text-muted-foreground">
						<tr>
							<th scope="col" class="px-3 py-2 font-semibold">{chart.xLabel}</th>
							{#each chart.table.lines as line (line.label)}
								<th scope="col" class="px-3 py-2 font-semibold">{line.label}</th>
							{/each}
						</tr>
					</thead>
					<tbody>
						{#each chart.table.ticks as tick, i (tick)}
							<tr class="border-t">
								<th scope="row" class="px-3 py-2 font-normal">{tick}</th>
								{#each chart.table.lines as line (line.label)}
									<td class="px-3 py-2">{shown(line.values[i])}</td>
								{/each}
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</div>
</details>
