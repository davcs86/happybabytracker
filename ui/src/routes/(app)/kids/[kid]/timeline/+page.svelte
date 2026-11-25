<script lang="ts">
	import CheckIcon from '@lucide/svelte/icons/check';
	import { resolve } from '$app/paths';
	import Field from '#lib/components/entry/Field.svelte';
	import TimelineRow, { TRACKER_VISUAL } from '#lib/components/TimelineRow.svelte';
	import { Alert } from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Card } from '#lib/components/ui/card/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { TIMELINE_TYPE, TRACKER_KINDS, TRACKER_LABEL } from '#lib/constants.js';
	import { formatDayHeading } from '#lib/timeline.js';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const path = $derived(resolve('/(app)/kids/[kid]/timeline', { kid: data.kid.id }));
	const home = $derived(resolve('/(app)/kids/[kid]', { kid: data.kid.id }));
	const error = (name: string) => (data.error?.field === name ? data.error.message : null);
</script>

<svelte:head>
	<title>Timeline · {data.kid.name} · happybabytracker</title>
</svelte:head>

<div
	class="flex flex-col gap-[18px] md:max-w-[720px] xl:mx-auto xl:grid xl:w-[920px] xl:max-w-none xl:grid-cols-[448px_448px] xl:items-start xl:gap-x-6"
>
	<div class="mt-3.5 flex items-center gap-2.5 xl:col-span-2">
		<h1 class="min-w-0 flex-1">Timeline</h1>
		<Button href={home} class="flex-none">Log something</Button>
	</div>

	<div class="contents xl:flex xl:flex-col xl:gap-[18px]">
		<form method="GET" action={path} class="flex flex-col gap-[18px]">
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
			<fieldset class="m-0 min-w-0 border-0 p-0">
				<legend
					class="mb-2 p-0 text-xs font-semibold tracking-[0.06em] text-muted-foreground uppercase"
					>Show</legend
				>
				<div class="flex flex-wrap gap-2">
					{#each TRACKER_KINDS as kind (kind)}
						{@const Icon = TRACKER_VISUAL[kind].icon}
						<label
							class="relative inline-flex min-h-11 cursor-pointer items-center gap-1.5 rounded-sm border bg-card px-3.5 text-sm font-medium has-checked:border-primary has-checked:bg-accent has-checked:font-semibold has-focus-visible:outline-2 has-focus-visible:outline-offset-2 has-focus-visible:outline-primary"
						>
							<input
								type="checkbox"
								name="types"
								value={TIMELINE_TYPE[kind]}
								checked={data.filter.kinds.includes(kind)}
								class="peer absolute inset-0 m-0 size-full cursor-pointer opacity-0"
							/>
							<CheckIcon
								aria-hidden="true"
								class="hidden size-3.5 text-primary peer-checked:block"
							/>
							<Icon aria-hidden="true" class="size-4" />
							{TRACKER_LABEL[kind]}
						</label>
					{/each}
				</div>
			</fieldset>
			<Button type="submit" variant="outline" class="self-start">Show</Button>
		</form>
	</div>

	<div class="contents xl:flex xl:flex-col xl:gap-[18px]">
		{#if data.error?.field === 'cursor'}
			<Alert>{data.error.message}</Alert>
			<a href="{path}?{data.newest}" class="inline-flex min-h-12 items-center self-start underline"
				>Back to newest</a
			>
		{:else if !data.error}
			{#if data.days.length === 0}
				<Card class="items-start gap-2 border-dashed px-[22px] py-[22px]">
					<p class="text-[17px] font-semibold">Nothing logged in this range</p>
					<p class="text-[13.5px] leading-normal text-muted-foreground">
						Entries from every tracker appear here, newest first, grouped by day.
					</p>
					<Button href={home} variant="outline">Log something</Button>
				</Card>
			{:else}
				<div class="flex flex-col gap-5">
					{#each data.days as day (day.date)}
						<section aria-labelledby="day-{day.date}" class="flex flex-col gap-2">
							<div>
								<h2 id="day-{day.date}" class="text-base font-semibold">
									{formatDayHeading(day.date)}
								</h2>
								{#if day.totals.length > 0}
									<ul
										aria-label="Totals"
										class="mt-0.5 flex flex-wrap gap-x-3 text-[12.5px] text-muted-foreground"
									>
										{#each day.totals as row (row.label)}
											<li>{row.label} {row.value}</li>
										{/each}
									</ul>
								{/if}
							</div>
							<ul aria-label="Entries" class="flex flex-col gap-1.5">
								{#each day.items as item (item.entry.id)}
									<TimelineRow
										{item}
										kidId={data.kid.id}
										zone={data.kid.timeZone}
										units={data.units}
									/>
								{/each}
							</ul>
						</section>
					{/each}
				</div>
			{/if}

			{#if data.older || data.newest}
				<div class="flex flex-wrap items-center gap-x-6">
					{#if data.older}
						<Button href="{path}?{data.older}" rel="next" variant="outline" class="px-5"
							>Older</Button
						>
					{/if}
					{#if data.newest}
						<a
							href="{path}?{data.newest}"
							class="inline-flex min-h-12 items-center text-sm underline">Back to newest</a
						>
					{/if}
				</div>
			{/if}
		{/if}
	</div>
</div>
