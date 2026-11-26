<script lang="ts">
	import type { Snippet } from 'svelte';
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import ChevronDownIcon from '@lucide/svelte/icons/chevron-down';
	import ShieldCheckIcon from '@lucide/svelte/icons/shield-check';
	import Field from '#lib/components/entry/Field.svelte';
	import { SELECT_CLASS } from '#lib/components/field.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import VaccineGroup from '#lib/components/vaccines/VaccineGroup.svelte';
	import { formatAge } from '#lib/format.js';
	import { vaccineNewPath } from '#lib/paths.js';
	import type { VaccineGroup as Group } from '#lib/types.js';
	import { antigenName, windowLabel } from '#lib/vaccines.js';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	const guardian = $derived(data.kid.role === 'guardian');
	const schema = $derived(data.chosen ? data.vaccines.schema : null);
	const done = (g: Group) => ['given', 'dismissed', 'available'].includes(g.status);
	const active = $derived(data.vaccines.groups.filter((g) => !done(g)));
	const dueGroups = $derived(active.filter((g) => g.status !== 'upcoming'));
	const upcoming = $derived(active.filter((g) => g.status === 'upcoming'));
	const completed = $derived(data.vaccines.groups.filter(done));
	const name = (id: string) => antigenName(id, data.catalogue);
	const edit = (administration: string) =>
		resolve('/(app)/kids/[kid]/vaccines/[administration]', { kid: data.kid.id, administration });

	const before = $derived(form?.form === 'dismissBefore' ? form : null);
	const dismissed = $derived(before && 'dismissed' in before ? before.dismissed : null);
	const beforeFailure = $derived(before && 'message' in before ? before : null);
	// A dismiss-before failure without a field shows here, as a caregiver has no form to show it in.
	const lineFailure = $derived(
		form?.form === 'dismiss' || form?.form === 'undoDismiss'
			? form.message
			: beforeFailure && !beforeFailure.field
				? beforeFailure.message
				: null
	);
	const beforeError = (name: string) =>
		beforeFailure?.field === name ? (beforeFailure.message ?? null) : null;

	const LABEL = 'text-xs font-semibold tracking-[0.06em] text-muted-foreground uppercase';
	const COLUMN = 'contents xl:flex xl:flex-col xl:gap-[18px]';
</script>

<svelte:head>
	<title>Vaccines · {data.kid.name} · happybabytracker</title>
</svelte:head>

{#snippet note(title: string, body: Snippet)}
	<details class="group overflow-hidden rounded-2xl border bg-card">
		<summary
			class="flex min-h-14 cursor-pointer list-none items-center justify-between gap-2.5 px-3.5 text-[15px] font-semibold [&::-webkit-details-marker]:hidden"
		>
			{title}
			<ChevronDownIcon
				aria-hidden="true"
				class="size-4 shrink-0 text-muted-foreground group-open:rotate-180 motion-safe:transition-transform"
			/>
		</summary>
		<div
			class="flex flex-col gap-1 px-3.5 pb-3.5 text-[13px] [overflow-wrap:anywhere] text-muted-foreground"
		>
			{@render body()}
		</div>
	</details>
{/snippet}

{#snippet visits(groups: Group[], openFirstOnly: boolean)}
	<div class="flex flex-col gap-3">
		{#each groups as group, i (group.fromDate)}
			<VaccineGroup
				{group}
				kidId={data.kid.id}
				catalogue={data.catalogue}
				{guardian}
				open={!openFirstOnly || i === 0}
			/>
		{/each}
	</div>
{/snippet}

<div
	class="flex max-w-[720px] flex-col gap-[18px] xl:mx-auto xl:grid xl:w-[920px] xl:max-w-none xl:grid-cols-[448px_448px] xl:items-start xl:gap-x-6"
>
	<div class="mt-5 flex flex-col gap-1.5 xl:col-span-2">
		<h1>Vaccines</h1>
		<p class="text-sm text-muted-foreground">
			{formatAge(data.kid.birthDate, data.vaccines.today)} · from birth, not corrected age
		</p>
		{#if schema}
			<p class="text-[13px] [overflow-wrap:anywhere] text-muted-foreground">
				Schedule: {schema.authority}, {schema.edition}
				<span>(retrieved {schema.retrieved})</span>
			</p>
		{/if}
	</div>

	{#if lineFailure}
		<p
			class="rounded-xl border-[1.5px] border-foreground bg-background px-3 py-2.5 text-[13.5px] leading-normal xl:col-span-2"
			role="alert"
		>
			{lineFailure}
		</p>
	{/if}
	{#if dismissed !== null}
		<p class="rounded-xl border bg-background px-3 py-2.5 text-[13px] xl:col-span-2" role="status">
			{`Dismissed ${dismissed} ${dismissed === 1 ? 'line' : 'lines'}.`}
		</p>
	{/if}

	<div class={COLUMN}>
		<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- resolve() renders a relative href without JavaScript, and the link keeps an absolute path -->
		<Button href={vaccineNewPath(data.kid.id)} data-drawer="Record a vaccination"
			>Record a vaccination</Button
		>

		{#if schema}
			{#if dueGroups.length > 0}
				<div class="flex flex-col gap-2">
					<h2 class={LABEL}>Due and overdue</h2>
					{@render visits(dueGroups, false)}
				</div>
			{/if}
			{#if upcoming.length > 0}
				<div class="flex flex-col gap-2">
					<h2 class={LABEL}>Coming up</h2>
					{@render visits(upcoming, true)}
				</div>
			{/if}
			{#if completed.length > 0}
				<section aria-labelledby="completed-heading" class="flex flex-col gap-2">
					<h2 id="completed-heading" class={LABEL}>Completed ({completed.length})</h2>
					<div class="flex flex-col gap-3">
						{#each completed as group (group.fromDate)}
							<VaccineGroup
								{group}
								kidId={data.kid.id}
								catalogue={data.catalogue}
								{guardian}
								open={false}
							/>
						{/each}
					</div>
				</section>
			{/if}
		{:else}
			<div
				class="flex flex-col items-start gap-3 rounded-2xl border border-dashed bg-card p-5 [overflow-wrap:anywhere]"
			>
				<span
					class="flex size-10 items-center justify-center rounded-xl bg-accent text-accent-foreground"
				>
					<ShieldCheckIcon aria-hidden="true" class="size-[22px]" />
				</span>
				<p class="text-[17px] font-semibold">No vaccination schedule chosen yet.</p>
				<p class="text-[13.5px] leading-normal text-muted-foreground">
					A schedule is only a list of what is usually offered and when. You can switch later;
					recorded doses always stay.
				</p>
				{#if guardian}
					<Button href={resolve('/(app)/kids/[kid]/settings', { kid: data.kid.id })}
						>Choose a schedule<span class="sr-only"> for {data.kid.name}</span></Button
					>
				{:else}
					<p class="text-sm font-medium">Ask a guardian to choose one.</p>
				{/if}
			</div>
		{/if}
	</div>

	<div class={COLUMN}>
		{#if schema}
			{#snippet about()}
				<p>Licence: {schema.licence}</p>
				<p>{schema.attribution}</p>
			{/snippet}
			{@render note('About this schedule', about)}
			{#if schema.omitted.length > 0}
				{#snippet omitted()}
					<ul class="list-disc pl-5">
						{#each schema.omitted as item, i (i)}
							<li>{item}</li>
						{/each}
					</ul>
				{/snippet}
				{@render note('What is not included', omitted)}
			{/if}

			{#if data.vaccines.also.length > 0}
				<section aria-labelledby="also-heading" class="flex flex-col gap-2">
					<h2 id="also-heading" class={LABEL}>Also available</h2>
					<ul class="overflow-hidden rounded-xl border bg-card">
						{#each data.vaccines.also as line (line.id)}
							<li
								aria-labelledby="vline-{line.id}-name vline-{line.id}-status"
								class="border-t p-3 first:border-t-0"
							>
								<p id="vline-{line.id}-name" class="text-sm font-medium">
									{name(line.antigen)}, dose {line.series}
								</p>
								<p id="vline-{line.id}-status" class="text-[12.5px] text-muted-foreground">
									{#if line.status === 'given' && line.givenOn && line.administration}
										<a
											href={edit(line.administration)}
											data-drawer="Edit vaccination"
											class="inline-flex min-h-11 items-center underline">Given {line.givenOn}</a
										>
									{:else if line.status === 'given' && line.givenOn}
										Given {line.givenOn}
									{:else}
										{windowLabel(line)}
									{/if}
								</p>
							</li>
						{/each}
					</ul>
				</section>
			{/if}

			{#if data.vaccines.recurring.length > 0}
				<section aria-labelledby="season-heading" class="flex flex-col gap-2">
					<h2 id="season-heading" class={LABEL}>Every season</h2>
					<ul class="overflow-hidden rounded-xl border bg-card">
						{#each data.vaccines.recurring as r (r.lineId)}
							<li class="border-t p-3 first:border-t-0">
								<p class="text-sm font-medium">{name(r.antigen)}</p>
								{#if r.doses.length === 0}
									<p class="text-[12.5px] text-muted-foreground">None recorded yet</p>
								{:else}
									<ul class="flex flex-wrap gap-x-4 text-[13px]">
										{#each r.doses as dose (dose.administration)}
											<li>
												<a
													href={edit(dose.administration)}
													data-drawer="Edit vaccination"
													class="inline-flex min-h-11 items-center underline"
													>Given {dose.givenOn}</a
												>
											</li>
										{/each}
									</ul>
								{/if}
							</li>
						{/each}
					</ul>
				</section>
			{/if}
		{/if}

		<section aria-labelledby="other-heading" class="flex flex-col gap-2">
			<h2 id="other-heading" class={LABEL}>Other recorded vaccinations</h2>
			{#if data.vaccines.other.length === 0}
				<p class="text-[13px] text-muted-foreground">None</p>
			{:else}
				<ul class="overflow-hidden rounded-xl border bg-card">
					{#each data.vaccines.other as dose (`${dose.administration}:${dose.antigen}`)}
						<li class="border-t p-3 text-sm [overflow-wrap:anywhere] first:border-t-0">
							<a
								href={edit(dose.administration)}
								data-drawer="Edit vaccination"
								class="inline-flex min-h-11 items-center font-medium underline"
								>{name(dose.antigen)}, given {dose.givenOn}</a
							>
							{#if dose.product}<p class="text-[12.5px] text-muted-foreground">
									{dose.product}
								</p>{/if}
						</li>
					{/each}
				</ul>
			{/if}
		</section>

		{#if schema && guardian}
			<section aria-labelledby="before-heading" class="rounded-2xl border bg-card p-3.5">
				<h2 id="before-heading" class="mb-1 text-base font-semibold">Dismiss older lines</h2>
				<p class="mb-3 text-[13.5px] leading-normal text-muted-foreground">
					Dismisses every line due before the date that is not given or dismissed yet. Each one can
					be undone on its own.
				</p>
				<form
					method="POST"
					action="?/dismissBefore"
					class="flex flex-col gap-3.5"
					novalidate
					use:enhance
				>
					<Field id="vaccine-before" label="Due before" error={beforeError('before')}>
						{#snippet children(control)}
							<Input
								{...control}
								name="before"
								type="date"
								value={beforeFailure?.values?.before ?? ''}
							/>
						{/snippet}
					</Field>
					<Field id="vaccine-before-reason" label="Reason" error={beforeError('reason')}>
						{#snippet children(control)}
							<select
								{...control}
								name="reason"
								class={SELECT_CLASS}
								value={beforeFailure?.values?.reason ?? 'not_needed'}
							>
								<option value="not_needed">Not needed</option>
								<option value="given_unknown">Given, date unknown</option>
							</select>
						{/snippet}
					</Field>
					<Button type="submit" variant="outline" class="self-start">Dismiss lines</Button>
				</form>
			</section>
		{/if}
	</div>
</div>
