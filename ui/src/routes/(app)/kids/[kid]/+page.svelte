<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import { ERROR_CLASS } from '#lib/components/field.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import KidLinks from '#lib/components/KidLinks.svelte';
	import KidSwitcher from '#lib/components/KidSwitcher.svelte';
	import QuickLog from '#lib/components/QuickLog.svelte';
	import RunningTimer, { type Timer } from '#lib/components/RunningTimer.svelte';
	import TodaySummary from '#lib/components/TodaySummary.svelte';
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
</script>

<svelte:head>
	<title>{data.kid.name} · happybabytracker</title>
</svelte:head>

<KidSwitcher kids={data.kids} currentId={data.kid.id} />
<h1 class="text-xl font-semibold">{data.kid.name}</h1>
<p class="text-muted-foreground">
	{data.correctedAge ? `${data.age} (corrected ${data.correctedAge})` : data.age}
</p>
<KidLinks kid={data.kid} />

<div class="mt-4 flex flex-col gap-6 md:grid md:grid-cols-2 md:items-start">
	<div class="flex flex-col gap-6">
		<QuickLog
			entries={data.quickLog}
			kidId={data.kid.id}
			message={failed('startSleep')?.message ?? null}
		/>
		{#if timers.length > 0}
			<section aria-labelledby="running-heading">
				<h2 id="running-heading" class="mb-2 font-semibold">Running now</h2>
				<ul class="flex flex-col gap-2">
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
	</div>
	<TodaySummary rows={data.today} />
</div>

<p class="mt-6 text-sm">
	<a href={resolve('/(app)/kids/new')} class="underline">Add a child</a>
</p>

{#if data.kid.role === 'caregiver'}
	<!-- Two steps without JavaScript: the button exists only inside the opened details.
	     A failure reopens it, so its message shows after a full-page POST too. -->
	<details class="mt-6 max-w-md rounded-lg border p-3" open={!!failed('leave')?.message}>
		<summary class="cursor-pointer text-sm font-medium">Leave this child</summary>
		<form method="POST" action="?/leave" class="mt-3 flex flex-col gap-3" use:enhance>
			{#if failed('leave')?.message}
				<p class={ERROR_CLASS} role="alert">{failed('leave')?.message}</p>
			{/if}
			<p class="text-sm">
				You will lose access to {data.kid.name}. A guardian has to add you again.
			</p>
			<Button type="submit" variant="outline" class="self-start border-destructive text-destructive"
				>Yes, leave {data.kid.name}</Button
			>
		</form>
	</details>
{/if}
