<script lang="ts">
	import { resolve } from '$app/paths';
	import Field from '#lib/components/entry/Field.svelte';
	import { ERROR_CLASS } from '#lib/components/field.js';
	import TimelineRow from '#lib/components/TimelineRow.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { TIMELINE_TYPE, TRACKER_KINDS, TRACKER_LABEL } from '#lib/constants.js';
	import { formatDayHeading } from '#lib/timeline.js';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const path = $derived(resolve('/(app)/kids/[kid]/timeline', { kid: data.kid.id }));
	const error = (name: string) => (data.error?.field === name ? data.error.message : null);
</script>

<svelte:head>
	<title>Timeline · {data.kid.name} · happybabytracker</title>
</svelte:head>

<p class="mb-2 text-sm">
	<a href={resolve('/(app)/kids/[kid]', { kid: data.kid.id })} class="underline">{data.kid.name}</a>
</p>
<div class="mb-4 flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
	<h1 class="text-xl font-semibold">Timeline for {data.kid.name}</h1>
	<a href={resolve('/(app)/kids/[kid]', { kid: data.kid.id })} class="text-sm underline"
		>Log something</a
	>
</div>

<form method="GET" action={path} class="mb-6 flex max-w-md flex-col gap-4">
	<div class="grid grid-cols-2 gap-3">
		<Field id="timeline-from" label="From" error={error('from')}>
			{#snippet children(control)}
				<Input {...control} type="date" name="from" value={data.filter.from} required />
			{/snippet}
		</Field>
		<Field id="timeline-to" label="To" error={error('to')}>
			{#snippet children(control)}
				<Input {...control} type="date" name="to" value={data.filter.to} required />
			{/snippet}
		</Field>
	</div>
	<fieldset class="flex flex-wrap gap-x-4 gap-y-2">
		<legend class="mb-1 text-sm font-medium">Show</legend>
		{#each TRACKER_KINDS as kind (kind)}
			<label class="flex min-h-8 items-center gap-2">
				<input
					type="checkbox"
					name="types"
					value={TIMELINE_TYPE[kind]}
					checked={data.filter.kinds.includes(kind)}
					class="size-4"
				/>
				{TRACKER_LABEL[kind]}
			</label>
		{/each}
	</fieldset>
	<Button type="submit" class="self-start">Show</Button>
</form>

{#if data.error?.field === 'cursor'}
	<p class="{ERROR_CLASS} mb-4" role="alert">{data.error.message}</p>
	<p><a href="{path}?{data.newest}" class="underline">Back to newest</a></p>
{:else if !data.error}
	{#each data.days as day (day.date)}
		<section aria-labelledby="day-{day.date}" class="mb-6">
			<h2 id="day-{day.date}" class="mb-1 font-semibold">{formatDayHeading(day.date)}</h2>
			{#if day.totals.length > 0}
				<ul
					aria-label="Totals"
					class="mb-2 flex flex-wrap gap-x-4 gap-y-1 text-sm text-muted-foreground"
				>
					{#each day.totals as row (row.label)}
						<li>{row.label} {row.value}</li>
					{/each}
				</ul>
			{/if}
			<ul aria-label="Entries" class="flex flex-col gap-2">
				{#each day.items as item (item.entry.id)}
					<TimelineRow {item} kidId={data.kid.id} zone={data.kid.timeZone} units={data.units} />
				{/each}
			</ul>
		</section>
	{:else}
		<p>Nothing logged in this range</p>
	{/each}

	<p class="flex gap-6">
		{#if data.older}
			<a href="{path}?{data.older}" rel="next" class="underline">Older</a>
		{/if}
		{#if data.newest}
			<a href="{path}?{data.newest}" class="underline">Back to newest</a>
		{/if}
	</p>
{/if}
