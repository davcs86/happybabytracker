<script lang="ts">
	import ChevronDownIcon from '@lucide/svelte/icons/chevron-down';
	import InfoIcon from '@lucide/svelte/icons/info';
	import TrendingUpIcon from '@lucide/svelte/icons/trending-up';
	import { resolve } from '$app/paths';
	import GrowthChart from '#lib/components/GrowthChart.svelte';
	import GrowthTable from '#lib/components/GrowthTable.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Card } from '#lib/components/ui/card/index.js';
	import { MEASURES } from '#lib/constants.js';
	import { lacksTable, MEASURE_LABEL } from '#lib/growth-chart.js';
	import { growthMeasurePath, growthNewPath } from '#lib/paths.js';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const sourceName = $derived(data.source?.name ?? data.curves.source);
	const corrected = $derived(data.rows.some((r) => r.correctedDays !== null));
	const heading = 'text-xs font-semibold tracking-[0.06em] text-muted-foreground uppercase';
	const disclosure =
		'group rounded-xl border bg-card px-3.5 text-[13px] [&>summary::-webkit-details-marker]:hidden';
	const summary =
		'flex min-h-12 cursor-pointer list-none items-center gap-2.5 text-sm font-semibold';
	const chevron =
		'size-4 flex-none text-muted-foreground group-open:rotate-180 motion-safe:transition-transform';
</script>

<svelte:head>
	<title>Growth · {data.kid.name} · happybabytracker</title>
</svelte:head>

<div
	class="flex flex-col gap-[18px] md:max-w-[720px] xl:mx-auto xl:grid xl:w-[920px] xl:max-w-none xl:grid-cols-[448px_448px] xl:items-start xl:gap-x-6"
>
	<div class="mt-3.5 flex items-end gap-2.5 xl:col-span-2">
		<div class="min-w-0 flex-1">
			<h1>Growth</h1>
			<p class="mt-1 text-sm text-muted-foreground">{data.age}</p>
		</div>
		<Button href={growthNewPath(data.kid.id)} class="flex-none">Add measurement</Button>
	</div>

	<div class="contents xl:flex xl:flex-col xl:gap-[18px]">
		<nav aria-label="Measures" class="flex gap-1 rounded-xl bg-muted p-1">
			{#each MEASURES as m (m)}
				<!-- eslint-disable svelte/no-navigation-without-resolve -- resolve() renders a relative href without JavaScript, and the link keeps an absolute path -->
				<a
					href={growthMeasurePath(data.kid.id, m)}
					aria-current={m === data.measure ? 'page' : undefined}
					class="flex min-h-11 min-w-0 flex-auto items-center justify-center rounded-[9px] px-2 text-center text-sm leading-tight font-semibold [overflow-wrap:anywhere] {m ===
					data.measure
						? 'bg-card text-foreground shadow-[0_1px_3px_rgb(0_0_0/0.15)]'
						: 'text-muted-foreground'}">{MEASURE_LABEL[m]}</a
				>
				<!-- eslint-enable svelte/no-navigation-without-resolve -->
			{/each}
		</nav>

		{#if data.rows.length === 0}
			<Card class="items-start gap-2.5 border-dashed px-[22px] py-[22px]">
				<span class="grid size-11 place-items-center rounded-xl bg-accent text-primary-ink"
					><TrendingUpIcon aria-hidden="true" class="size-[22px]" /></span
				>
				<p class="text-[17px] font-semibold">No measurements yet</p>
				<p class="text-[13.5px] leading-normal text-muted-foreground">
					Add a weight, length or head size from a check-up or from home. The chart appears with the
					first one.
				</p>
			</Card>
		{:else}
			<section aria-labelledby="chart-heading" class="flex flex-col gap-2.5">
				<h2 id="chart-heading" class={heading}>{MEASURE_LABEL[data.measure]} for age</h2>
				{#if lacksTable(data.curves.covered, data.rows, data.measure)}
					<p
						role="status"
						class="rounded-lg border bg-card px-3.5 py-3 text-[13.5px] leading-normal"
					>
						{sourceName} has no table for this measure at this age. Your measurements are listed below.
					</p>
				{/if}
				<GrowthChart
					curves={data.curves}
					rows={data.rows}
					measure={data.measure}
					units={data.units}
					{sourceName}
					kidName={data.kid.name}
				/>
			</section>

			<details class={disclosure}>
				<summary class={summary}
					><InfoIcon aria-hidden="true" class="size-[18px] flex-none" /><span class="flex-1"
						>What do these numbers mean?</span
					><ChevronDownIcon aria-hidden="true" class={chevron} /></summary
				>
				<div class="flex flex-col gap-2.5 pb-3.5 leading-[1.55] text-muted-foreground">
					<p>
						<b class="text-foreground">Percentile.</b> Compares one measurement with a reference group
						of children of the same age and sex. A percentile of 50 is the middle of that group.
					</p>
					<p>
						<b class="text-foreground">z-score.</b> The same comparison counted in steps from the middle.
						0 is the middle; each step up or down is one standard deviation.
					</p>
					{#if corrected}
						<p>
							<b class="text-foreground">Corrected age.</b> For a baby born early, this app subtracts
							the weeks of early birth from age until the second birthday. WHO and CDC do not publish
							this rule; it is this app’s own choice. Both ages are shown.
						</p>
					{/if}
					<p>
						This page shows the numbers and does not interpret them. Questions about what they mean
						for {data.kid.name} are for your clinic.
					</p>
				</div>
			</details>
		{/if}
	</div>

	<div class="contents xl:flex xl:flex-col xl:gap-[18px]">
		{#if data.rows.length > 0}
			<section aria-labelledby="list-heading" class="flex flex-col gap-2.5">
				<h2 id="list-heading" class={heading}>Measurements</h2>
				<GrowthTable
					rows={data.rows}
					kidId={data.kid.id}
					units={data.units}
					measure={data.measure}
				/>
			</section>
		{/if}

		<p
			class="flex flex-wrap items-center gap-x-2 text-[13px] text-muted-foreground"
			data-testid="growth-source"
		>
			Growth source: {sourceName}
			{#if data.kid.role === 'guardian'}
				<a
					href={resolve('/(app)/kids/[kid]/settings', { kid: data.kid.id })}
					class="inline-flex min-h-11 items-center text-sm text-foreground underline"
					>Change the growth source<span class="sr-only"> for {data.kid.name}</span></a
				>
			{/if}
		</p>

		{#if data.source}
			<section aria-labelledby="about-heading">
				<details class={disclosure}>
					<summary class={summary}
						><span id="about-heading" class="flex-1">About the reference data</span><ChevronDownIcon
							aria-hidden="true"
							class={chevron}
						/></summary
					>
					<div class="flex flex-col gap-1 pb-3 leading-[1.55] text-muted-foreground">
						<p>{data.source.name}, {data.source.version}</p>
						<p>{data.source.attribution}</p>
						<p>Licence: {data.source.licence}</p>
					</div>
				</details>
			</section>
		{/if}
	</div>
</div>
