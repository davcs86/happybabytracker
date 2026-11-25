<script lang="ts">
	import { enhance } from '$app/forms';
	import type { SubmitFunction } from '$app/forms';
	import { toast } from 'svelte-sonner';
	import { Alert } from '#lib/components/ui/alert/index.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { VISIT_KIND_LABEL, VISIT_STATUS_LABEL } from '#lib/constants.js';
	import {
		growthEntryPath,
		medicationEntryPath,
		vaccinesPath,
		visitEditPath,
		visitOutcomePath
	} from '#lib/paths.js';
	import { hasOutcome, visitRecordDate, visitWhenLong } from '#lib/visits.js';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	const visit = $derived(data.visit);
	const zone = $derived(data.kid.timeZone);
	const label = $derived(VISIT_KIND_LABEL[visit.kind]);
	const where = $derived([visit.provider, visit.location].filter(Boolean).join(' · '));
	const outcome = $derived(hasOutcome(visit));
	const records = $derived([
		...visit.linked.growth.map((g) => ({
			id: g.id,
			module: 'Growth',
			detail: visitRecordDate(g.measuredOn),
			href: growthEntryPath(data.kid.id, g.id),
			drawer: 'Edit measurement'
		})),
		...visit.linked.vaccinations.map((a) => ({
			id: a.id,
			module: 'Vaccines',
			detail: visitRecordDate(a.givenOn),
			href: `${vaccinesPath(data.kid.id)}/${a.id}`,
			drawer: 'Edit vaccination'
		})),
		...visit.linked.medications.map((m) => ({
			id: m.id,
			module: 'Medication',
			detail: m.name,
			href: medicationEntryPath(data.kid.id, m.id),
			drawer: 'Edit course'
		}))
	]);

	// The visit page is no sheet, so its own toast says what the button did.
	const done =
		(message: string): SubmitFunction =>
		() =>
		async ({ result, update }) => {
			await update({ reset: false });
			if (result.type === 'redirect' || result.type === 'success') toast(message);
		};
</script>

<svelte:head>
	<title>{label} · {data.kid.name} · happybabytracker</title>
</svelte:head>

<!-- eslint-disable svelte/no-navigation-without-resolve -- path helpers, absolute like the other links to pages without JavaScript -->
<div
	class="flex max-w-[720px] flex-col gap-[18px] xl:mx-auto xl:grid xl:w-[920px] xl:max-w-none xl:grid-cols-[448px_448px] xl:items-start xl:gap-x-6"
>
	<div class="mt-5 flex flex-col gap-1.5 xl:col-span-2">
		<div class="flex flex-wrap items-center gap-x-3 gap-y-1.5">
			<h1>{label}</h1>
			<Badge>{VISIT_STATUS_LABEL[visit.status]}</Badge>
		</div>
		<p class="text-[15px]">{visitWhenLong(visit, zone)}</p>
		{#if where}<p class="text-[15px] [overflow-wrap:anywhere]">{where}</p>{/if}
		{#if visit.note}<p class="text-[15px] [overflow-wrap:anywhere]">Reason: {visit.note}</p>{/if}
		<p class="text-[13.5px] [overflow-wrap:anywhere] text-muted-foreground">Times are in {zone}</p>
	</div>

	<div class="contents xl:flex xl:flex-col xl:gap-[18px]">
		{#if form?.form === 'status' && form.message}
			<Alert class="[overflow-wrap:anywhere]">{form.message}</Alert>
		{/if}
		<div class="flex flex-wrap gap-2">
			<Button
				href={visitEditPath(data.kid.id, visit.id)}
				data-drawer="Edit visit"
				variant="outline"
				class="px-5">Edit visit</Button
			>
			{#if visit.status === 'cancelled'}
				<form method="POST" action="?/restore" use:enhance={done('Visit restored')}>
					<Button type="submit" variant="outline" class="px-5">Restore visit</Button>
				</form>
			{:else}
				<form method="POST" action="?/cancel" use:enhance={done('Visit cancelled')}>
					<Button type="submit" variant="outline" class="px-5">Cancel visit</Button>
				</form>
			{/if}
		</div>
		{#if visit.status === 'past'}
			<a
				href={visitOutcomePath(data.kid.id, visit.id)}
				data-drawer="After the visit"
				class="inline-flex min-h-11 items-center self-start text-sm underline"
				>{outcome ? 'Add more from the visit' : 'Record the visit'}</a
			>
		{/if}
	</div>

	{#if outcome}
		<div class="contents xl:flex xl:flex-col xl:gap-[18px]">
			{#if visit.outcomeNotes}
				<section aria-labelledby="after-heading" class="flex flex-col gap-2.5">
					<h2 id="after-heading" class="text-lg font-semibold">After the visit</h2>
					<p class="text-[15px] [overflow-wrap:anywhere] whitespace-pre-line">
						{visit.outcomeNotes}
					</p>
				</section>
			{/if}
			{#if records.length > 0}
				<section aria-labelledby="recorded-heading" class="flex flex-col gap-2.5">
					<h2 id="recorded-heading" class="text-lg font-semibold">Recorded at this visit</h2>
					<ul class="flex flex-col gap-2.5">
						{#each records as record (record.id)}
							<li>
								<a
									href={record.href}
									data-drawer={record.drawer}
									class="flex min-h-14 items-center gap-3 rounded-xl border bg-card px-3 py-2 hover:bg-accent"
								>
									<span class="text-[15px] font-semibold">{record.module}</span>
									<span
										class="min-w-0 flex-1 text-[13.5px] [overflow-wrap:anywhere] text-muted-foreground"
										>{record.detail}</span
									>
								</a>
							</li>
						{/each}
					</ul>
				</section>
			{/if}
		</div>
	{/if}
</div>
<!-- eslint-enable svelte/no-navigation-without-resolve -->
