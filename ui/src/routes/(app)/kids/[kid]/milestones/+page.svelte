<script lang="ts">
	import { resolve } from '$app/paths';
	import PencilIcon from '@lucide/svelte/icons/pencil';
	import Checklist from '#lib/components/milestones/Checklist.svelte';
	import { Alert } from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
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

	const LABEL = 'text-xs font-semibold tracking-[0.06em] text-muted-foreground uppercase';
	const COLUMN = 'contents xl:flex xl:flex-col xl:gap-[18px]';
</script>

<svelte:head>
	<title>Milestones · {data.kid.name} · happybabytracker</title>
</svelte:head>

{#snippet row(text: string, id: string, reachedOn: string, detail: string | null = null)}
	<li
		class="flex flex-col gap-1.5 rounded-lg border bg-card p-3 text-[15px] [overflow-wrap:anywhere]"
	>
		<span>{text}</span>
		{#if detail}<span class="text-sm text-muted-foreground">{detail}</span>{/if}
		<span class="flex flex-wrap items-center gap-x-3 gap-y-1 text-sm text-muted-foreground">
			{reachedLabel(reachedOn)}
			<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- the entry page's path helper, absolute like the other links to pages without JavaScript -->
			<Button
				href={milestoneEntryPath(data.kid.id, id)}
				variant="outline"
				size="icon"
				title="Edit"
				class="[&_svg]:size-5"
				><PencilIcon aria-hidden="true" data-chrome /><span class="sr-only">Edit {text}</span
				></Button
			>
		</span>
	</li>
{/snippet}

<div
	class="flex max-w-[720px] flex-col gap-[18px] xl:mx-auto xl:grid xl:w-[920px] xl:max-w-none xl:grid-cols-[448px_448px] xl:items-start xl:gap-x-6"
>
	<div class="mt-5 flex flex-col gap-1.5 xl:col-span-2">
		<h1>Milestones</h1>
		<p class="text-sm text-muted-foreground">{data.age}</p>
	</div>

	{#if data.milestonesOff}
		<p
			class="rounded-xl border bg-card px-3 py-2.5 text-[13.5px] leading-[1.5] [overflow-wrap:anywhere] xl:col-span-2"
		>
			Milestones are off for {data.kid.name}.
			{#if data.kid.role === 'guardian'}
				<a
					href={resolve('/(app)/kids/[kid]/modules', { kid: data.kid.id })}
					class="inline-flex min-h-11 items-center underline">Turn it on in Modules</a
				>
			{/if}
		</p>
	{/if}

	{#if form?.message && !failed}
		<Alert class="xl:col-span-2">{form.message}</Alert>
	{/if}

	<div class={COLUMN}>
		{#if m.catalogShown}
			{#if shown}
				<a
					href="#list-{shown.ageMonths}"
					class="inline-flex min-h-11 items-center self-start text-sm underline"
					>{current ? 'Go to the current list' : 'Go to the next list'}</a
				>
			{/if}
			<section aria-labelledby="lists-heading" class="flex flex-col gap-2">
				<h2 id="lists-heading" class={LABEL}>Milestone checklists</h2>
				<div class="flex flex-col gap-3">
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
			<section aria-labelledby="reached-heading" class="flex flex-col gap-2">
				<h2 id="reached-heading" class={LABEL}>Reached</h2>
				{#if m.reached.length === 0}
					<p class="text-[13px] text-muted-foreground">No milestones recorded</p>
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
			<section aria-labelledby="retired-heading" class="flex flex-col gap-2">
				<h2 id="retired-heading" class={LABEL}>Reached, no longer in the lists</h2>
				<ul class="flex flex-col gap-2">
					{#each m.retired as r (r.id)}
						{@render row(r.text, r.id, r.reachedOn)}
					{/each}
				</ul>
			</section>
		{/if}
	</div>

	<div class={COLUMN}>
		<!-- The catalog's terms of reuse: shown in full, beside the lists, never folded away. -->
		<div
			class="flex flex-col gap-2 text-[13px] leading-[1.5] [overflow-wrap:anywhere] text-muted-foreground"
		>
			<p>{m.catalog.screeningNote}</p>
			<p>{m.catalog.attribution}</p>
		</div>

		<section aria-labelledby="custom-heading" class="flex flex-col gap-2">
			<div class="flex flex-wrap items-center justify-between gap-x-4 gap-y-2">
				<h2 id="custom-heading" class="text-base font-semibold">Your milestones</h2>
				<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- resolve() renders a relative href without JavaScript, and the link keeps an absolute path -->
				<Button href={milestoneNewPath(data.kid.id)} variant="outline">Add a milestone</Button>
			</div>
			{#if m.custom.length === 0}
				<p class="text-[13px] text-muted-foreground">No milestones added</p>
			{:else}
				<ul class="flex flex-col gap-2">
					{#each m.custom as c (c.id)}
						{@render row(c.text, c.id, c.reachedOn)}
					{/each}
				</ul>
			{/if}
		</section>
	</div>
</div>
