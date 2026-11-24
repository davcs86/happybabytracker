<script lang="ts">
	import { resolve } from '$app/paths';
	import { ERROR_CLASS } from '#lib/components/field.js';
	import Checklist from '#lib/components/milestones/Checklist.svelte';
	import { checklistTitle, domainLabel, reachedLabel } from '#lib/milestones.js';
	import { milestoneEntryPath, milestoneNewPath } from '#lib/paths.js';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	const m = $derived(data.milestones);
	const current = $derived(m.checklists.find((c) => c.position === 'current') ?? null);
	// Before the first list age nothing is current: the next list is open instead.
	const shown = $derived(current ?? m.checklists.find((c) => c.position === 'next') ?? null);
	const marker = (age: number) =>
		age !== shown?.ageMonths ? null : current ? 'Current list for this age' : 'Next list';
	// A refused mark of a listed item is told in its row; any other message stays at the top.
	const failed = $derived(
		form?.message &&
			form.values?.item &&
			m.catalogShown &&
			m.checklists.some((c) => c.items.some((i) => i.id === form.values?.item))
			? { item: form.values.item, message: form.message }
			: null
	);
</script>

<svelte:head>
	<title>Milestones · {data.kid.name} · happybabytracker</title>
</svelte:head>

{#snippet row(text: string, id: string, reachedOn: string, detail: string | null = null)}
	<li class="flex flex-col gap-1 rounded-lg border p-3 text-sm [overflow-wrap:anywhere]">
		<span>{text}</span>
		{#if detail}<span class="text-muted-foreground">{detail}</span>{/if}
		<span class="flex flex-wrap gap-x-3">
			{reachedLabel(reachedOn)}
			<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- the entry page's path helper, absolute like the other links to pages without JavaScript -->
			<a href={milestoneEntryPath(data.kid.id, id)} class="underline"
				>Edit<span class="sr-only"> {text}</span></a
			>
		</span>
	</li>
{/snippet}

<p class="mb-2 text-sm">
	<a href={resolve('/(app)/kids/[kid]', { kid: data.kid.id })} class="underline">{data.kid.name}</a>
</p>
<h1 class="text-xl font-semibold [overflow-wrap:anywhere]">Milestones for {data.kid.name}</h1>
<p class="mb-4 text-muted-foreground">{data.age}</p>

{#if data.milestonesOff}
	<p class="mb-4 text-sm [overflow-wrap:anywhere]">
		Milestones are off for {data.kid.name}.
		{#if data.kid.role === 'guardian'}
			<a href={resolve('/(app)/kids/[kid]/modules', { kid: data.kid.id })} class="underline"
				>Turn it on in Modules</a
			>
		{/if}
	</p>
{/if}

{#if form?.message && !failed}
	<p class="{ERROR_CLASS} mb-4" role="alert">{form.message}</p>
{/if}

{#if m.catalogShown}
	{#if shown}
		<p class="mb-4 text-sm">
			<a href="#list-{shown.ageMonths}" class="underline"
				>{current ? 'Go to the current list' : 'Go to the next list'}</a
			>
		</p>
	{/if}
	<section aria-labelledby="lists-heading" class="mb-6">
		<h2 id="lists-heading" class="mb-2 font-semibold">Milestone checklists</h2>
		<div class="flex flex-col gap-2">
			<!-- Keyed by kid too: each list's open state belongs to one kid's page. -->
			{#each m.checklists as checklist (`${data.kid.id}:${checklist.ageMonths}`)}
				<Checklist
					kidId={data.kid.id}
					{checklist}
					marker={marker(checklist.ageMonths)}
					target={failed?.item ?? data.open}
					{failed}
				/>
			{/each}
		</div>
	</section>
{:else}
	<section aria-labelledby="reached-heading" class="mb-6">
		<h2 id="reached-heading" class="mb-2 font-semibold">Reached</h2>
		{#if m.reached.length === 0}
			<p class="text-sm text-muted-foreground">No milestones recorded</p>
		{:else}
			<ul class="flex flex-col gap-2">
				{#each m.reached as r (r.id)}
					{@render row(
						r.text,
						r.id,
						r.reachedOn,
						`${checklistTitle(r.ageMonths)} · ${domainLabel(r.domain)}`
					)}
				{/each}
			</ul>
		{/if}
	</section>
{/if}

{#if m.retired.length > 0}
	<section aria-labelledby="retired-heading" class="mb-6">
		<h2 id="retired-heading" class="mb-2 font-semibold">Reached, no longer in the lists</h2>
		<ul class="flex flex-col gap-2">
			{#each m.retired as r (r.id)}
				{@render row(r.text, r.id, r.reachedOn)}
			{/each}
		</ul>
	</section>
{/if}

<!-- The catalog's terms of reuse: shown in full, beside the lists, never folded away. -->
<div class="mb-6 flex flex-col gap-2 text-sm [overflow-wrap:anywhere]">
	<p>{m.catalog.screeningNote}</p>
	<p>{m.catalog.attribution}</p>
</div>

<section aria-labelledby="custom-heading" class="mb-6">
	<div class="mb-2 flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
		<h2 id="custom-heading" class="font-semibold">Your milestones</h2>
		<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- resolve() renders a relative href without JavaScript, and the link keeps an absolute path -->
		<a href={milestoneNewPath(data.kid.id)} class="text-sm underline">Add a milestone</a>
	</div>
	{#if m.custom.length === 0}
		<p class="text-sm text-muted-foreground">No milestones added</p>
	{:else}
		<ul class="flex flex-col gap-2">
			{#each m.custom as c (c.id)}
				{@render row(c.text, c.id, c.reachedOn)}
			{/each}
		</ul>
	{/if}
</section>
