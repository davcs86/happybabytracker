<script lang="ts">
	import type { Component } from 'svelte';
	import ArrowUpIcon from '@lucide/svelte/icons/arrow-up';
	import ChartLineIcon from '@lucide/svelte/icons/chart-line';
	import DropletIcon from '@lucide/svelte/icons/droplet';
	import FlagIcon from '@lucide/svelte/icons/flag';
	import HeartIcon from '@lucide/svelte/icons/heart';
	import MilkIcon from '@lucide/svelte/icons/milk';
	import MoonIcon from '@lucide/svelte/icons/moon';
	import NotebookPenIcon from '@lucide/svelte/icons/notebook-pen';
	import SyringeIcon from '@lucide/svelte/icons/syringe';
	import UtensilsIcon from '@lucide/svelte/icons/utensils';
	import { enhance } from '$app/forms';
	import { ERROR_CLASS } from '#lib/components/field.js';
	import { SECTION_HEADING } from '#lib/components/home.js';
	import { logPath } from '#lib/paths.js';
	import { PAGE_ROUTE, type QuickLogEntry } from '#lib/quick-log.js';
	import type { ModuleId } from '#lib/types.js';

	let {
		entries,
		kidId,
		message = null
	}: { entries: QuickLogEntry[]; kidId: string; message?: string | null } = $props();

	const href = (entry: QuickLogEntry) =>
		entry.kind ? logPath(kidId, entry.kind) : PAGE_ROUTE[entry.module]?.(kidId);

	// The chip hue: a tracker class, or the hue of a module that has no tracker of its own.
	const LOOK: Record<ModuleId, { icon: Component; hue: string }> = {
		bottles: { icon: MilkIcon, hue: 'tracker-bottle' },
		nursing: { icon: HeartIcon, hue: 'tracker-nursing' },
		diapers: { icon: DropletIcon, hue: 'tracker-diaper' },
		sleep: { icon: MoonIcon, hue: 'tracker-sleep' },
		pumping: { icon: ArrowUpIcon, hue: 'tracker-pumping' },
		feeding: { icon: UtensilsIcon, hue: 'tracker-solids' },
		sickness: { icon: NotebookPenIcon, hue: 'tracker-symptom' },
		medication: { icon: NotebookPenIcon, hue: 'tracker-symptom' },
		milestones: { icon: FlagIcon, hue: '[--h:170]' },
		growth: { icon: ChartLineIcon, hue: '[--h:200]' },
		vaccination: { icon: SyringeIcon, hue: '[--h:235]' }
	};
	const HINT: Partial<Record<ModuleId, string>> = {
		milestones: 'Open Milestones',
		growth: 'Add a measurement',
		vaccination: 'Open Vaccines',
		sickness: 'Log a symptom',
		medication: 'Log a dose'
	};

	const TILE =
		'flex h-full min-h-24 w-full flex-col items-start gap-2.5 rounded-xl border bg-card p-3 text-left hover:bg-accent';
</script>

{#snippet face(entry: QuickLogEntry)}
	{@const look = LOOK[entry.module]}
	<span class="tracker-chip {look.hue}"><look.icon aria-hidden="true" class="size-5" /></span>
	<span class="block">
		<span id="ql-{entry.module}" class="block text-[15px] leading-tight font-semibold"
			>{entry.label}</span
		>
		{#if HINT[entry.module]}
			<span id="ql-{entry.module}-hint" class="mt-px block text-xs text-muted-foreground"
				>{HINT[entry.module]}</span
			>
		{/if}
	</span>
{/snippet}

{#if entries.length > 0}
	<section aria-labelledby="quick-log-heading" class="flex flex-col gap-2.5">
		<h2 id="quick-log-heading" class={SECTION_HEADING}>Quick log</h2>
		{#if message}<p class={ERROR_CLASS} role="alert">{message}</p>{/if}
		<ul class="grid grid-cols-2 gap-2.5">
			{#each entries as entry (entry.module)}
				{@const describedby = HINT[entry.module] ? `ql-${entry.module}-hint` : undefined}
				<li>
					<!-- Sleep is one tap, a form post that works without JavaScript; the rest open their form. -->
					{#if entry.kind === 'sleep'}
						<form method="POST" action="?/startSleep" class="h-full" use:enhance>
							<button
								type="submit"
								class={TILE}
								aria-labelledby="ql-{entry.module}"
								aria-describedby={describedby}>{@render face(entry)}</button
							>
						</form>
					{:else}
						<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- resolve() renders a relative href without JavaScript, and the tile keeps an absolute path -->
						<a
							href={href(entry)}
							class={TILE}
							aria-labelledby="ql-{entry.module}"
							aria-describedby={describedby}>{@render face(entry)}</a
						>
					{/if}
				</li>
			{/each}
		</ul>
	</section>
{/if}
