<script lang="ts">
	import { resolve } from '$app/paths';
	import ChevronRightIcon from '@lucide/svelte/icons/chevron-right';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const CARD = 'flex flex-col gap-3 rounded-2xl border bg-card p-4';
	const COLUMN = 'contents xl:flex xl:flex-col xl:gap-[18px]';
	const kid = $derived(data.kid);
	const settingsHref = $derived(resolve('/(app)/kids/[kid]/settings', { kid: kid.id }));
	const accessHref = $derived(resolve('/(app)/kids/[kid]/access', { kid: kid.id }));
	const born = $derived(
		new Date(`${kid.birthDate}T00:00:00Z`).toLocaleDateString('en-US', {
			dateStyle: 'long',
			timeZone: 'UTC'
		})
	);
	const unit = (n: number, word: string) => `${n} ${word}${n === 1 ? '' : 's'}`;
	const gestation = $derived(
		kid.gestationWeeks === null
			? null
			: kid.gestationDays
				? `${unit(kid.gestationWeeks, 'week')} ${unit(kid.gestationDays, 'day')}`
				: unit(kid.gestationWeeks, 'week')
	);
	const rows = $derived([
		['Name', kid.name],
		['Born', born],
		...(gestation ? [['Weeks at birth', gestation]] : []),
		['Sex', kid.sex === 'female' ? 'Female' : 'Male'],
		['Time zone', kid.timeZone]
	]);
	const settings = $derived([
		{
			title: 'Trackers',
			note: 'Choose which trackers show on Home',
			href: resolve('/(app)/kids/[kid]/modules', { kid: kid.id })
		},
		{
			title: 'Growth chart source',
			note: 'On the Growth page',
			href: `${settingsHref}#kid-growth_source`
		},
		{
			title: 'Vaccination schedule',
			note: 'On the Vaccines page',
			href: `${settingsHref}#kid-vaccine_schema`
		},
		{
			title: 'Show a missed dose as overdue for (minutes)',
			note: unit(kid.reminderOverdueMinutes, 'minute'),
			href: `${settingsHref}#kid-reminder_overdue_minutes`
		}
	]);
</script>

<svelte:head>
	<title>Child profile · happybabytracker</title>
</svelte:head>

<div
	class="mt-5 flex max-w-[720px] flex-col gap-[18px] xl:mx-auto xl:grid xl:w-[920px] xl:max-w-none xl:grid-cols-[448px_448px] xl:items-start xl:gap-x-6"
>
	<div class="flex flex-col gap-1.5 xl:col-span-2">
		<h1>Child profile</h1>
		<p class="text-sm leading-snug text-muted-foreground">{data.ages[kid.id]}</p>
	</div>

	<div class={COLUMN}>
		<section aria-labelledby="details-heading" class={CARD}>
			<div class="flex items-center justify-between gap-3">
				<h2 id="details-heading" class="text-base font-semibold">Details</h2>
				<Button href={settingsHref} variant="outline">Edit details</Button>
			</div>
			<dl class="flex flex-col gap-3">
				{#each rows as [label, value] (label)}
					<div>
						<dt class="text-xs text-muted-foreground">{label}</dt>
						<dd class="text-[15px] [overflow-wrap:anywhere]">{value}</dd>
					</div>
				{/each}
			</dl>
			<p class="text-[12.5px] leading-normal text-muted-foreground">
				The birth date and weeks at birth set corrected age on Home. Times for everyone follow the
				time zone.
			</p>
		</section>

		<section aria-labelledby="settings-heading" class={CARD}>
			<h2 id="settings-heading" class="text-base font-semibold [overflow-wrap:anywhere]">
				Settings for {kid.name}
			</h2>
			<ul>
				{#each settings as s (s.title)}
					<li class="border-t">
						<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- resolve() plus a field anchor -->
						<a href={s.href} class="flex min-h-[52px] items-center gap-2 py-1.5">
							<span class="flex min-w-0 flex-1 flex-col">
								<span class="text-[14.5px] font-medium [overflow-wrap:anywhere]">{s.title}</span>
								<span class="text-[12.5px] text-muted-foreground">{s.note}</span>
							</span>
							<ChevronRightIcon aria-hidden="true" class="size-4 shrink-0 text-muted-foreground" />
						</a>
					</li>
				{/each}
			</ul>
		</section>
	</div>

	<div class={COLUMN}>
		<section aria-labelledby="people-heading" class={CARD}>
			<h2 id="people-heading" class="text-base font-semibold">People with access</h2>
			<p class="text-[13px] leading-normal [overflow-wrap:anywhere] text-muted-foreground">
				Guardians can change who sees {kid.name}. Caregivers can log, view and export. A child
				always keeps at least one active guardian.
			</p>
			<ul aria-label="People with access" class="flex flex-col">
				{#each data.grants as g (g.parentId)}
					<li class="flex flex-col gap-1.5 border-t py-3">
						<div class="flex flex-wrap items-center gap-2">
							<span class="font-medium [overflow-wrap:anywhere]">{g.displayName}</span>
							<Badge>{g.role === 'guardian' ? 'Guardian' : 'Caregiver'}</Badge>
							{#if g.parentId === data.me}<Badge>You</Badge>{/if}
						</div>
						<span class="font-mono text-[12.5px] [overflow-wrap:anywhere] text-muted-foreground"
							>{g.username}</span
						>
					</li>
				{/each}
			</ul>
			<Button href={accessHref} variant="outline">Manage access</Button>
		</section>

		<section aria-labelledby="delete-heading" class="{CARD} border-[1.5px] border-destructive">
			<h2 id="delete-heading" class="text-base font-semibold [overflow-wrap:anywhere]">
				Delete all data for {kid.name}
			</h2>
			<p class="text-[13.5px] leading-normal [overflow-wrap:anywhere] text-muted-foreground">
				Removes {kid.name} and everything logged for them, for everyone who has access. It is permanent
				and immediate. You confirm twice.
			</p>
			<Button href="{settingsHref}#delete" variant="destructive" class="whitespace-normal"
				>Delete all data for {kid.name}…</Button
			>
		</section>
	</div>
</div>
