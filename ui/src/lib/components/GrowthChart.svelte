<script lang="ts">
	import { CHART_VALUE_DECIMALS } from '#lib/constants.js';
	import { buildChart, MEASURE_LABEL, type ChartInput } from '#lib/growth-chart.js';
	import { formatDayHeading } from '#lib/timeline.js';

	let { curves, rows, measure, units, sourceName }: ChartInput & { sourceName: string } = $props();

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

<figure class="max-w-xl">
	<svg
		role="img"
		aria-labelledby="growth-chart-title growth-chart-desc"
		viewBox="0 0 {chart.width} {chart.height}"
		class="h-auto w-full text-[10px]"
	>
		<title id="growth-chart-title">{title}</title>
		<desc id="growth-chart-desc">{desc}</desc>
		<text x={chart.plot.left} y={chart.plot.top - 10} class="fill-foreground">{chart.yLabel}</text>
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
				class="fill-foreground">{t.value}</text
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
			<text x={t.pos} y={chart.plot.bottom + 12} text-anchor="middle" class="fill-foreground"
				>{t.value}</text
			>
		{/each}
		<text
			x={(chart.plot.left + chart.plot.right) / 2}
			y={chart.height - 6}
			text-anchor="middle"
			class="fill-foreground">{chart.xLabel}</text
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
				class="fill-foreground">{line.label}</text
			>
		{/each}
		{#if chart.path}
			<path d={chart.path} fill="none" class="stroke-primary" stroke-width="2" />
		{/if}
		{#each chart.points as p (p.id)}
			<circle data-point cx={p.cx} cy={p.cy} r="3.5" class="fill-primary stroke-background" />
		{/each}
	</svg>
</figure>

<details class="mt-2 max-w-xl text-sm">
	<summary class="cursor-pointer font-medium">Chart data</summary>
	<div class="mt-2 overflow-x-auto">
		<table class="w-full text-left">
			<caption class="text-left font-medium">Measurements plotted</caption>
			<thead>
				<tr>
					<th scope="col" class="pr-3">Date</th>
					<th scope="col" class="pr-3">{chart.xLabel}</th>
					<th scope="col">{chart.yLabel}</th>
				</tr>
			</thead>
			<tbody>
				{#each chart.points as p (p.id)}
					<tr>
						<td class="pr-3">{formatDayHeading(p.measuredOn)}</td>
						<td class="pr-3">{shown(p.x)}</td>
						<td>{shown(p.y)}</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
	{#if chart.table.lines.length > 0}
		<div class="mt-3 overflow-x-auto">
			<table class="w-full text-left">
				<caption class="text-left font-medium">Percentile values</caption>
				<thead>
					<tr>
						<th scope="col" class="pr-3">{chart.xLabel}</th>
						{#each chart.table.lines as line (line.label)}
							<th scope="col" class="pr-3">{line.label}</th>
						{/each}
					</tr>
				</thead>
				<tbody>
					{#each chart.table.ticks as tick, i (tick)}
						<tr>
							<th scope="row" class="pr-3 font-normal">{tick}</th>
							{#each chart.table.lines as line (line.label)}
								<td class="pr-3">{shown(line.values[i])}</td>
							{/each}
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</details>
