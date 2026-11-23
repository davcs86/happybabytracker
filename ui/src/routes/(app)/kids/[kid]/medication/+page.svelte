<script lang="ts">
	import { resolve } from '$app/paths';
	import { ERROR_CLASS } from '#lib/components/field.js';
	import CourseCard from '#lib/components/medication/CourseCard.svelte';
	import { courseEnded, whenLabel } from '#lib/medication.js';
	import { doseEntryPath, medicationNewPath } from '#lib/paths.js';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	const zone = $derived(data.kid.timeZone);
	const running = (c: { active: boolean; endOn: string | null }) =>
		c.active && !courseEnded(c.endOn, data.today);
	const current = $derived(data.courses.filter(running));
	const rest = $derived(data.courses.filter((c) => !running(c)));
	const nameOf = (id: string) => data.courses.find((c) => c.id === id)?.name ?? 'Medication';

	const failed = $derived(form?.form === 'log' ? form : null);
	// An amount refusal shows on its course; anything else (the course gone, the cap, the server) above the list.
	const topMessage = $derived(failed && failed.field !== 'amount' ? failed.message : null);
	const failureOf = (id: string) =>
		failed?.field === 'amount' && failed.values.medication === id ? failed : null;
</script>

<svelte:head>
	<title>Medication · {data.kid.name} · happybabytracker</title>
</svelte:head>

<p class="mb-2 text-sm">
	<a href={resolve('/(app)/kids/[kid]', { kid: data.kid.id })} class="underline">{data.kid.name}</a>
</p>
<div class="mb-1 flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
	<h1 class="text-xl font-semibold [overflow-wrap:anywhere]">Medication for {data.kid.name}</h1>
	<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- resolve() renders a relative href without JavaScript, and the link keeps an absolute path -->
	<a href={medicationNewPath(data.kid.id)} class="text-sm underline">Add a course</a>
</div>
<p class="mb-4 text-sm text-muted-foreground">Times are in {zone}</p>

{#if topMessage}
	<p class="{ERROR_CLASS} mb-4" role="alert">{topMessage}</p>
{/if}

<section aria-labelledby="courses-heading" class="mb-6">
	<h2 id="courses-heading" class="mb-2 font-semibold">Courses</h2>
	{#if data.courses.length === 0}
		<p class="text-sm text-muted-foreground">No courses yet</p>
	{:else if current.length === 0}
		<p class="text-sm text-muted-foreground">No active courses</p>
	{/if}
	{#if current.length > 0}
		<ul class="flex flex-col gap-2">
			{#each current as course (course.id)}
				{@const failure = failureOf(course.id)}
				<CourseCard
					{course}
					kidId={data.kid.id}
					{zone}
					today={data.today}
					amount={failure?.values.amount}
					error={failure?.message}
				/>
			{/each}
		</ul>
	{/if}
	{#if rest.length > 0}
		<details class="mt-4" open={rest.some((c) => failureOf(c.id))}>
			<summary class="mb-2 cursor-pointer font-medium">
				Ended and inactive courses ({rest.length})
			</summary>
			<ul class="flex flex-col gap-2">
				{#each rest as course (course.id)}
					{@const failure = failureOf(course.id)}
					<CourseCard
						{course}
						kidId={data.kid.id}
						{zone}
						today={data.today}
						amount={failure?.values.amount}
						error={failure?.message}
					/>
				{/each}
			</ul>
		</details>
	{/if}
</section>

<section aria-labelledby="doses-heading" class="mb-6">
	<div class="mb-2 flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
		<h2 id="doses-heading" class="font-semibold [overflow-wrap:anywhere]">
			{data.filter ? `Doses of ${data.filter.name}` : 'Recent doses'}
		</h2>
		{#if data.filter}
			<a
				href={resolve('/(app)/kids/[kid]/medication', { kid: data.kid.id })}
				class="text-sm underline">Show all doses</a
			>
		{/if}
	</div>
	{#if data.doses.length === 0}
		<p class="text-sm text-muted-foreground">No doses logged yet</p>
	{:else}
		<ul class="flex flex-col gap-2">
			{#each data.doses as dose (dose.id)}
				<li class="rounded-lg border p-3 text-sm [overflow-wrap:anywhere]">
					<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- the dose page's path helper, absolute like the other links to pages without JavaScript -->
					<a href={doseEntryPath(data.kid.id, dose.id)} class="font-medium underline"
						>{nameOf(dose.medication)}, {whenLabel(dose.givenAt, zone, data.today)}</a
					>
					{#if dose.amount}<p>{dose.amount}</p>{/if}
					{#if dose.note}<p class="text-muted-foreground">{dose.note}</p>{/if}
				</li>
			{/each}
		</ul>
	{/if}
</section>
