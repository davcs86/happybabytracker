<script lang="ts">
	import { resolve } from '$app/paths';
	import ChevronDownIcon from '@lucide/svelte/icons/chevron-down';
	import NotebookPenIcon from '@lucide/svelte/icons/notebook-pen';
	import CourseCard from '#lib/components/medication/CourseCard.svelte';
	import { Alert } from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
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

<div
	class="flex max-w-[720px] flex-col gap-[18px] xl:mx-auto xl:grid xl:w-[920px] xl:max-w-none xl:grid-cols-[448px_448px] xl:items-start xl:gap-x-6"
>
	<div class="mt-5 flex flex-wrap items-start gap-x-3 gap-y-1.5 xl:col-span-2">
		<div class="flex min-w-0 flex-1 flex-col gap-1.5">
			<h1>Medication</h1>
			<p class="text-sm leading-snug text-muted-foreground">{data.ages[data.kid.id]}</p>
		</div>
		{#if !data.medicationOff}
			<Button href={medicationNewPath(data.kid.id)} variant="outline" class="shrink-0 px-5"
				>Add a course</Button
			>
		{/if}
		<p class="w-full text-[13.5px] [overflow-wrap:anywhere] text-muted-foreground">
			Times are in {zone}
		</p>
	</div>

	{#if topMessage}
		<Alert class="[overflow-wrap:anywhere] xl:col-span-2">{topMessage}</Alert>
	{/if}

	{#if data.medicationOff}
		<section
			aria-labelledby="off-heading"
			class="flex flex-col gap-3 rounded-2xl border bg-card p-3.5"
		>
			<h2 id="off-heading" class="text-[17px] font-semibold [overflow-wrap:anywhere]">
				Medication is switched off for {data.kid.name}
			</h2>
			<p class="text-[13.5px] text-muted-foreground">
				Courses and doses are kept. Nothing is deleted.
			</p>
			{#if data.kid.role === 'guardian'}
				<Button
					href={resolve('/(app)/kids/[kid]/modules', { kid: data.kid.id })}
					variant="outline"
					class="self-start px-5">Open modules</Button
				>
			{:else}
				<p class="text-[13.5px] text-muted-foreground">A guardian can switch it on.</p>
			{/if}
		</section>
	{:else}
		<div class="contents xl:flex xl:flex-col xl:gap-[18px]">
			<section aria-labelledby="courses-heading" class="flex flex-col gap-3">
				<h2 id="courses-heading" class="text-lg font-semibold">Courses</h2>
				{#if data.courses.length === 0}
					<p class="text-[13.5px] text-muted-foreground">No courses yet</p>
				{:else if current.length === 0}
					<p class="text-[13.5px] text-muted-foreground">No active courses</p>
				{/if}
				{#if current.length > 0}
					<ul class="flex flex-col gap-3">
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
					<details class="group rounded-xl border bg-card" open={rest.some((c) => failureOf(c.id))}>
						<summary
							class="flex min-h-14 cursor-pointer list-none items-center gap-2.5 px-3.5 [&::-webkit-details-marker]:hidden"
						>
							<span class="flex-1 text-[15px] font-semibold">
								Ended and inactive courses ({rest.length})
							</span>
							<ChevronDownIcon
								aria-hidden="true"
								class="size-4 shrink-0 text-muted-foreground group-open:rotate-180 motion-safe:transition-transform"
							/>
						</summary>
						<ul class="flex flex-col gap-3 px-3 pb-3">
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
		</div>

		<div class="contents xl:flex xl:flex-col xl:gap-[18px]">
			<section aria-labelledby="doses-heading" class="flex flex-col gap-2.5">
				<h2 id="doses-heading" class="text-lg font-semibold [overflow-wrap:anywhere]">
					{data.filter ? `Doses of ${data.filter.name}` : 'Recent doses'}
				</h2>
				{#if data.filter}
					<a
						href={resolve('/(app)/kids/[kid]/medication', { kid: data.kid.id })}
						class="inline-flex min-h-11 items-center self-start text-sm underline">Show all doses</a
					>
				{/if}
				{#if data.doses.length === 0}
					<p class="text-[13.5px] text-muted-foreground">No doses logged yet</p>
				{:else}
					<ul class="flex flex-col gap-2.5">
						{#each data.doses as dose (dose.id)}
							{@const when = whenLabel(dose.givenAt, zone, data.today)}
							<li>
								<!-- eslint-disable svelte/no-navigation-without-resolve -- the dose page's path helper, absolute like the other links to pages without JavaScript -->
								<a
									href={doseEntryPath(data.kid.id, dose.id)}
									aria-label="{nameOf(dose.medication)}, {when}"
									class="flex min-h-15 items-center gap-3 rounded-xl border bg-card px-3 py-2 [overflow-wrap:anywhere]"
								>
									<span
										aria-hidden="true"
										class="grid size-10 shrink-0 place-items-center rounded-xl bg-accent text-primary-ink"
									>
										<NotebookPenIcon class="size-5" />
									</span>
									<span class="flex min-w-0 flex-1 flex-col gap-0.5">
										<span class="text-[15px] font-semibold">{nameOf(dose.medication)}</span>
										{#if dose.amount}<span class="text-sm">{dose.amount}</span>{/if}
										{#if dose.note}
											<span class="text-[13.5px] text-muted-foreground">{dose.note}</span>
										{/if}
									</span>
									<span class="shrink-0 text-sm text-muted-foreground">{when}</span>
								</a>
								<!-- eslint-enable svelte/no-navigation-without-resolve -->
							</li>
						{/each}
					</ul>
				{/if}
			</section>
		</div>
	{/if}
</div>
