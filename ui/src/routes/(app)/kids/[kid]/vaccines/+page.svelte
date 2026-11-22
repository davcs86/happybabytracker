<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import Field from '#lib/components/entry/Field.svelte';
	import { ERROR_CLASS, SELECT_CLASS } from '#lib/components/field.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import VaccineGroup from '#lib/components/vaccines/VaccineGroup.svelte';
	import { vaccineNewPath } from '#lib/paths.js';
	import type { VaccineGroup as Group } from '#lib/types.js';
	import { antigenName, windowLabel } from '#lib/vaccines.js';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	const guardian = $derived(data.kid.role === 'guardian');
	const schema = $derived(data.chosen ? data.vaccines.schema : null);
	const done = (g: Group) => g.status === 'given' || g.status === 'dismissed';
	const active = $derived(data.vaccines.groups.filter((g) => !done(g)));
	const completed = $derived(data.vaccines.groups.filter(done));
	const name = (id: string) => antigenName(id, data.catalogue);
	const edit = (administration: string) =>
		resolve('/(app)/kids/[kid]/vaccines/[administration]', { kid: data.kid.id, administration });

	const lineFailure = $derived(
		form?.form === 'dismiss' || form?.form === 'undoDismiss' ? form.message : null
	);
	const before = $derived(form?.form === 'dismissBefore' ? form : null);
	const dismissed = $derived(before && 'dismissed' in before ? before.dismissed : null);
	const beforeFailure = $derived(before && 'message' in before ? before : null);
	const beforeError = (name: string) =>
		beforeFailure?.field === name ? (beforeFailure.message ?? null) : null;
</script>

<svelte:head>
	<title>Vaccines · {data.kid.name} · happybabytracker</title>
</svelte:head>

<p class="mb-2 text-sm">
	<a href={resolve('/(app)/kids/[kid]', { kid: data.kid.id })} class="underline">{data.kid.name}</a>
</p>
<div class="mb-4 flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
	<h1 class="text-xl font-semibold">Vaccines for {data.kid.name}</h1>
	<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- resolve() renders a relative href without JavaScript, and the link keeps an absolute path -->
	<a href={vaccineNewPath(data.kid.id)} class="text-sm underline">Record a vaccination</a>
</div>

{#if lineFailure}
	<p class="{ERROR_CLASS} mb-4" role="alert">{lineFailure}</p>
{/if}
{#if dismissed !== null}
	<p class="mb-4 text-sm" role="status">
		{`Dismissed ${dismissed} ${dismissed === 1 ? 'line' : 'lines'}.`}
	</p>
{/if}

{#if schema}
	<div class="mb-6 flex flex-col gap-2 text-sm">
		<p>
			Schedule: {schema.authority}, {schema.edition}
			<span class="text-muted-foreground">(retrieved {schema.retrieved})</span>
		</p>
		<details>
			<summary class="cursor-pointer">About this schedule</summary>
			<div class="mt-1 flex flex-col gap-1 [overflow-wrap:anywhere] text-muted-foreground">
				<p>Licence: {schema.licence}</p>
				<p>{schema.attribution}</p>
			</div>
		</details>
		{#if schema.omitted.length > 0}
			<details>
				<summary class="cursor-pointer">What is not included</summary>
				<ul class="mt-1 list-disc pl-5 text-muted-foreground">
					{#each schema.omitted as item (item)}
						<li>{item}</li>
					{/each}
				</ul>
			</details>
		{/if}
	</div>

	{#each active as group (group.fromDate)}
		<VaccineGroup {group} kidId={data.kid.id} catalogue={data.catalogue} {guardian} />
	{/each}
	{#if completed.length > 0}
		<details class="mb-6">
			<summary class="mb-2 cursor-pointer font-semibold">Completed ({completed.length})</summary>
			{#each completed as group (group.fromDate)}
				<VaccineGroup {group} kidId={data.kid.id} catalogue={data.catalogue} {guardian} />
			{/each}
		</details>
	{/if}

	{#if data.vaccines.also.length > 0}
		<section aria-labelledby="also-heading" class="mb-6">
			<h2 id="also-heading" class="mb-2 font-semibold">Also available</h2>
			<ul class="flex flex-col gap-2">
				{#each data.vaccines.also as line (line.id)}
					<li aria-labelledby="vline-{line.id}-name" class="rounded-lg border p-3">
						<p id="vline-{line.id}-name" class="font-medium">
							{name(line.antigen)}, dose {line.series}
						</p>
						<p class="text-sm">
							{line.status === 'given' && line.givenOn
								? `Given ${line.givenOn}`
								: windowLabel(line)}
						</p>
					</li>
				{/each}
			</ul>
		</section>
	{/if}

	{#if data.vaccines.recurring.length > 0}
		<section aria-labelledby="season-heading" class="mb-6">
			<h2 id="season-heading" class="mb-2 font-semibold">Every season</h2>
			<ul class="flex flex-col gap-2">
				{#each data.vaccines.recurring as r (r.lineId)}
					<li class="rounded-lg border p-3">
						<p class="font-medium">{name(r.antigen)}</p>
						{#if r.doses.length === 0}
							<p class="text-sm text-muted-foreground">None recorded yet</p>
						{:else}
							<ul class="flex flex-wrap gap-x-4 text-sm">
								{#each r.doses as dose (dose.administration)}
									<li>
										<a href={edit(dose.administration)} class="underline">Given {dose.givenOn}</a>
									</li>
								{/each}
							</ul>
						{/if}
					</li>
				{/each}
			</ul>
		</section>
	{/if}
{:else}
	<p class="mb-6">
		No vaccination schedule chosen yet.
		{#if guardian}
			<a href={resolve('/(app)/kids/[kid]/settings', { kid: data.kid.id })} class="underline"
				>Choose a schedule<span class="sr-only"> for {data.kid.name}</span></a
			>
		{:else}
			Ask a guardian to choose one.
		{/if}
	</p>
{/if}

<section aria-labelledby="other-heading" class="mb-6">
	<h2 id="other-heading" class="mb-2 font-semibold">Other recorded vaccinations</h2>
	{#if data.vaccines.other.length === 0}
		<p class="text-sm text-muted-foreground">None</p>
	{:else}
		<ul class="flex flex-col gap-2">
			{#each data.vaccines.other as dose (`${dose.administration}:${dose.antigen}`)}
				<li class="rounded-lg border p-3 text-sm [overflow-wrap:anywhere]">
					<a href={edit(dose.administration)} class="font-medium underline"
						>{name(dose.antigen)}, given {dose.givenOn}</a
					>
					{#if dose.product}<p class="text-muted-foreground">{dose.product}</p>{/if}
				</li>
			{/each}
		</ul>
	{/if}
</section>

{#if schema && guardian}
	<section aria-labelledby="before-heading" class="max-w-md rounded-lg border p-3">
		<h2 id="before-heading" class="mb-1 font-semibold">Dismiss older lines</h2>
		<p class="mb-3 text-sm text-muted-foreground">
			Dismisses every line due before the date that is not given or dismissed yet. Each one can be
			undone on its own.
		</p>
		<form method="POST" action="?/dismissBefore" class="flex flex-col gap-3" novalidate use:enhance>
			{#if beforeFailure && !beforeFailure.field && beforeFailure.message}
				<p class={ERROR_CLASS} role="alert">{beforeFailure.message}</p>
			{/if}
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
