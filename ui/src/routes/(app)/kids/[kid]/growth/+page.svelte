<script lang="ts">
	import { resolve } from '$app/paths';
	import GrowthChart from '#lib/components/GrowthChart.svelte';
	import GrowthTable from '#lib/components/GrowthTable.svelte';
	import { MEASURES } from '#lib/constants.js';
	import { lacksTable, MEASURE_LABEL } from '#lib/growth-chart.js';
	import { growthMeasurePath, growthNewPath } from '#lib/paths.js';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const sourceName = $derived(data.source?.name ?? data.curves.source);
</script>

<svelte:head>
	<title>Growth · {data.kid.name} · happybabytracker</title>
</svelte:head>

<p class="mb-2 text-sm">
	<a href={resolve('/(app)/kids/[kid]', { kid: data.kid.id })} class="underline">{data.kid.name}</a>
</p>
<div class="mb-4 flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
	<div>
		<h1 class="text-xl font-semibold">Growth for {data.kid.name}</h1>
		<p class="text-muted-foreground">
			{data.correctedAge ? `${data.age} (corrected ${data.correctedAge})` : data.age}
		</p>
	</div>
	<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- resolve() renders a relative href without JavaScript, and the link keeps an absolute path -->
	<a href={growthNewPath(data.kid.id)} class="text-sm underline">Add measurement</a>
</div>

<nav aria-label="Measures" class="mb-4">
	<ul class="flex flex-wrap gap-x-4 gap-y-2 text-sm">
		{#each MEASURES as m (m)}
			<li>
				<!-- eslint-disable svelte/no-navigation-without-resolve -- resolve() renders a relative href without JavaScript, and the link keeps an absolute path -->
				<a
					href={growthMeasurePath(data.kid.id, m)}
					aria-current={m === data.measure ? 'page' : undefined}
					class="inline-flex min-h-8 items-center underline-offset-4 {m === data.measure
						? 'font-semibold underline'
						: 'hover:underline'}">{MEASURE_LABEL[m]}</a
				>
				<!-- eslint-enable svelte/no-navigation-without-resolve -->
			</li>
		{/each}
	</ul>
</nav>

<section aria-labelledby="chart-heading" class="mb-6">
	<h2 id="chart-heading" class="mb-2 font-semibold">{MEASURE_LABEL[data.measure]}</h2>
	{#if lacksTable(data.curves.covered, data.rows, data.measure)}
		<p class="mb-2">{sourceName} has no table for this measure at this age</p>
	{/if}
	<GrowthChart
		curves={data.curves}
		rows={data.rows}
		measure={data.measure}
		units={data.units}
		{sourceName}
	/>
</section>

<section aria-labelledby="list-heading" class="mb-6">
	<h2 id="list-heading" class="mb-2 font-semibold">Measurements</h2>
	<GrowthTable rows={data.rows} kidId={data.kid.id} units={data.units} />
</section>

<p class="mb-4 text-sm" data-testid="growth-source">
	Growth source: {sourceName}
	{#if data.kid.role === 'guardian'}
		·
		<a href={resolve('/(app)/kids/[kid]/settings', { kid: data.kid.id })} class="underline"
			>Change the growth source<span class="sr-only"> for {data.kid.name}</span></a
		>
	{/if}
</p>

{#if data.source}
	<section
		aria-labelledby="about-heading"
		class="max-w-xl rounded-lg border p-3 text-sm text-muted-foreground"
	>
		<h2 id="about-heading" class="mb-1 font-medium text-foreground">About the reference data</h2>
		<p>{data.source.name}, {data.source.version}</p>
		<p>{data.source.attribution}</p>
		<p>Licence: {data.source.licence}</p>
	</section>
{/if}
