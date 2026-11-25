<script lang="ts">
	import { resolve } from '$app/paths';
	import StethoscopeIcon from '@lucide/svelte/icons/stethoscope';
	import { Alert } from '#lib/components/ui/alert/index.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { VISIT_KIND_LABEL, VISIT_STATUS_LABEL } from '#lib/constants.js';
	import { visitNewPath, visitPath } from '#lib/paths.js';
	import type { VisitRow } from '#lib/types.js';
	import { visitRowName, visitWhen } from '#lib/visits.js';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const zone = $derived(data.kid.timeZone);
	const path = $derived(resolve('/(app)/kids/[kid]/visits', { kid: data.kid.id }));
	const empty = $derived(
		!data.paged && !data.cursorError && data.upcoming.length === 0 && data.past.length === 0
	);
</script>

<svelte:head>
	<title>Medical visits · {data.kid.name} · happybabytracker</title>
</svelte:head>

{#snippet addLink(className: string)}
	<Button
		href={visitNewPath(data.kid.id)}
		data-drawer="Add a visit"
		variant="outline"
		class="{className} px-5">Add a visit</Button
	>
{/snippet}

{#snippet rows(visits: VisitRow[], label: string)}
	<ul aria-label={label} class="flex flex-col gap-2.5">
		{#each visits as visit (visit.id)}
			<li>
				<!-- eslint-disable svelte/no-navigation-without-resolve -- the visit page's path helper, absolute like the other links to pages without JavaScript -->
				<a
					href={visitPath(data.kid.id, visit.id)}
					aria-label={visitRowName(visit, zone)}
					class="flex min-h-[60px] items-center gap-3 rounded-xl border bg-card px-3 py-2 hover:bg-accent"
				>
					<span class="tracker-chip [--h:235]"
						><StethoscopeIcon aria-hidden="true" class="size-[18px]" /></span
					>
					<div class="min-w-0 flex-1 [overflow-wrap:anywhere]">
						<h3 class="text-[15px] font-semibold">{VISIT_KIND_LABEL[visit.kind]}</h3>
						<p class="text-[13.5px] text-muted-foreground">{visitWhen(visit.startsAt, zone)}</p>
						{#if visit.provider}<p class="text-[13.5px] text-muted-foreground">
								{visit.provider}
							</p>{/if}
					</div>
					{#if visit.status === 'cancelled'}<Badge>{VISIT_STATUS_LABEL.cancelled}</Badge>{/if}
				</a>
				<!-- eslint-enable svelte/no-navigation-without-resolve -->
			</li>
		{/each}
	</ul>
{/snippet}

<div
	class="flex max-w-[720px] flex-col gap-[18px] xl:mx-auto xl:grid xl:w-[920px] xl:max-w-none xl:grid-cols-[448px_448px] xl:items-start xl:gap-x-6"
>
	<div class="mt-5 flex flex-wrap items-start gap-x-3 gap-y-1.5 xl:col-span-2">
		<div class="flex min-w-0 flex-1 flex-col gap-1.5">
			<h1>Medical visits</h1>
			<p class="text-sm leading-snug text-muted-foreground">{data.ages[data.kid.id]}</p>
		</div>
		{#if !data.visitsOff && !empty}{@render addLink('shrink-0')}{/if}
		<p class="w-full text-[13.5px] [overflow-wrap:anywhere] text-muted-foreground">
			Times are in {zone}
		</p>
	</div>

	{#if data.visitsOff}
		<section
			aria-labelledby="off-heading"
			class="flex flex-col gap-3 rounded-2xl border bg-card p-3.5 xl:col-span-2"
		>
			<h2 id="off-heading" class="text-[17px] font-semibold [overflow-wrap:anywhere]">
				Medical visits is switched off for {data.kid.name}
			</h2>
			<p class="text-[13.5px] text-muted-foreground">Visits are kept. Nothing is deleted.</p>
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
	{:else if empty}
		<section class="flex flex-col gap-3 rounded-2xl border bg-card p-3.5 xl:col-span-2">
			<p class="text-[15px]">No visits yet.</p>
			{@render addLink('self-start')}
		</section>
	{:else}
		<div class="contents xl:flex xl:flex-col xl:gap-[18px]">
			{#if !data.paged && data.upcoming.length > 0}
				<section aria-labelledby="upcoming-heading" class="flex flex-col gap-2.5">
					<h2 id="upcoming-heading" class="text-lg font-semibold">
						{VISIT_STATUS_LABEL.upcoming}
					</h2>
					{@render rows(data.upcoming, VISIT_STATUS_LABEL.upcoming)}
				</section>
			{/if}
		</div>
		<div class="contents xl:flex xl:flex-col xl:gap-[18px]">
			{#if data.past.length > 0 || data.paged || data.cursorError}
				<section aria-labelledby="past-heading" class="flex flex-col gap-2.5">
					<h2 id="past-heading" class="text-lg font-semibold">{VISIT_STATUS_LABEL.past}</h2>
					{#if data.cursorError}<Alert>{data.cursorError}</Alert>{/if}
					{#if data.past.length > 0}{@render rows(data.past, VISIT_STATUS_LABEL.past)}{/if}
					<div class="flex flex-wrap items-center gap-3">
						{#if data.older}
							<Button
								href="{path}?cursor={encodeURIComponent(data.older)}"
								rel="next"
								variant="outline"
								class="px-5">Older</Button
							>
						{/if}
						{#if data.paged}
							<a href={path} class="inline-flex min-h-12 items-center text-sm underline"
								>Back to newest</a
							>
						{/if}
					</div>
				</section>
			{/if}
		</div>
	{/if}
</div>
