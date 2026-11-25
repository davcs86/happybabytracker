<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import { ERROR_CLASS } from '#lib/components/field.js';
	import ChevronDownIcon from '@lucide/svelte/icons/chevron-down';
	import { Button } from '#lib/components/ui/button/index.js';
	import DueList from '#lib/components/DueList.svelte';
	import { SECTION_HEADING } from '#lib/components/home.js';
	import QuickLog from '#lib/components/QuickLog.svelte';
	import RunningTimer, { type Timer } from '#lib/components/RunningTimer.svelte';
	import TodaySummary from '#lib/components/TodaySummary.svelte';
	import { HOME_DUE_MAX } from '#lib/constants.js';
	import { dueLines } from '#lib/due.js';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	const timers = $derived<Timer[]>([
		...(data.running.sleep ? [{ kind: 'sleep' as const, entry: data.running.sleep }] : []),
		...(data.running.nursing ? [{ kind: 'nursing' as const, entry: data.running.nursing }] : [])
	]);
	const failed = (...names: string[]) => (form && names.includes(form.form) ? form : null);
	const messageFor = (kind: Timer['kind']) =>
		failed(...(kind === 'sleep' ? ['endSleep'] : ['endNursing', 'switchSide']))?.message ?? null;
	const overrunFor = (kind: Timer['kind']) =>
		!!failed(kind === 'sleep' ? 'endSleep' : 'endNursing')?.overrun;
	// "{age} (corrected {age})" splits into the two age lines; the gestation comes from the kid.
	const ages = $derived.by(() => {
		const [born, rest] = data.age.split(' (');
		if (rest === undefined) return { born, corrected: null };
		const note = rest.replace(/\)$/, '');
		const weeks = data.kid.gestationWeeks;
		const gest = weeks === null ? '' : ` · born ${weeks}w ${data.kid.gestationDays ?? 0}d`;
		const text = note.startsWith('corrected ')
			? `Corrected age ${note.slice(10)}`
			: `${note[0].toUpperCase()}${note.slice(1)}`;
		return { born, corrected: `${text}${gest}` };
	});
	const dueList = $derived(
		dueLines(data.due, HOME_DUE_MAX, data.kid.id, data.kid.timeZone, data.todayDate)
	);
</script>

<svelte:head>
	<title>{data.kid.name} · happybabytracker</title>
</svelte:head>

<div
	class="flex flex-col gap-[22px] xl:mx-auto xl:grid xl:w-[920px] xl:grid-cols-[448px_448px] xl:items-start xl:gap-x-6 xl:gap-y-[22px]"
>
	<div class="mt-5 xl:col-span-2">
		<h1>{data.kid.name}</h1>
		<p class="mt-1 text-sm text-muted-foreground">{ages.born}</p>
		{#if ages.corrected}<p class="mt-0.5 text-[12.5px] text-muted-foreground">
				{ages.corrected}
			</p>{/if}
	</div>
	<div class="contents xl:flex xl:min-w-0 xl:flex-col xl:gap-[22px]">
		{#if timers.length > 0}
			<section aria-labelledby="running-heading" class="flex flex-col gap-2.5">
				<h2 id="running-heading" class={SECTION_HEADING}>Running now</h2>
				<ul class="flex flex-col gap-2.5">
					{#each timers as timer (timer.kind)}
						<RunningTimer
							{timer}
							zone={data.kid.timeZone}
							kidId={data.kid.id}
							message={messageFor(timer.kind)}
							overrun={overrunFor(timer.kind)}
						/>
					{/each}
				</ul>
			</section>
		{/if}
		<QuickLog
			entries={data.quickLog}
			kidId={data.kid.id}
			message={failed('startSleep')?.message ?? null}
		/>
	</div>
	<div class="contents xl:flex xl:min-w-0 xl:flex-col xl:gap-[22px]">
		<DueList lines={dueList} />
		<TodaySummary rows={data.today} />
		{#if data.kid.role === 'guardian'}
			<p>
				<a
					href={resolve('/(app)/kids/new')}
					class="inline-flex min-h-11 items-center text-sm underline">Add a child</a
				>
			</p>
		{:else}
			<!-- Two steps without JavaScript: the button exists only inside the opened details.
			     A failure reopens it, so its message shows after a full-page POST too. -->
			<details class="group rounded-2xl border bg-card px-3.5" open={!!failed('leave')?.message}>
				<summary
					class="flex min-h-13 cursor-pointer list-none items-center justify-between text-sm font-semibold [&::-webkit-details-marker]:hidden"
				>
					Leave this child
					<ChevronDownIcon
						aria-hidden="true"
						class="size-4 shrink-0 group-open:rotate-180 motion-safe:transition-transform"
					/>
				</summary>
				<form method="POST" action="?/leave" class="flex flex-col gap-3 pb-3.5" use:enhance>
					{#if failed('leave')?.message}
						<p class={ERROR_CLASS} role="alert">{failed('leave')?.message}</p>
					{/if}
					<p class="text-[13.5px] leading-normal text-muted-foreground">
						You will lose access to {data.kid.name}. A guardian has to add you again.
					</p>
					<Button type="submit" variant="destructive" class="px-5"
						>Yes, leave {data.kid.name}</Button
					>
				</form>
			</details>
		{/if}
	</div>
</div>
