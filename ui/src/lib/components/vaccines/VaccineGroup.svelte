<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import CalendarIcon from '@lucide/svelte/icons/calendar';
	import CheckIcon from '@lucide/svelte/icons/check';
	import ChevronDownIcon from '@lucide/svelte/icons/chevron-down';
	import CircleIcon from '@lucide/svelte/icons/circle';
	import ClockIcon from '@lucide/svelte/icons/clock';
	import MinusIcon from '@lucide/svelte/icons/minus';
	import PlusIcon from '@lucide/svelte/icons/plus';
	import { SELECT_CLASS } from '#lib/components/field.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { vaccineNewPath } from '#lib/paths.js';
	import type { VaccineCatalogue, VaccineGroup, VaccineLine, VaccineStatus } from '#lib/types.js';
	import { cn } from '#lib/utils.js';
	import { antigenName, groupTitle, statusLabel, windowLabel } from '#lib/vaccines.js';

	let {
		group,
		kidId,
		catalogue,
		guardian,
		open
	}: {
		group: VaccineGroup;
		kidId: string;
		catalogue: VaccineCatalogue;
		guardian: boolean;
		open: boolean;
	} = $props();

	const OPEN: readonly VaccineStatus[] = ['due', 'overdue', 'upcoming', 'available'];
	const DISMISSED_TEXT = {
		not_needed: 'Dismissed: not needed',
		given_unknown: 'Dismissed: given, date unknown'
	} as const;
	// A status is a word, an icon and a border shape; colour is never the only signal.
	const BADGE: Record<VaccineStatus, { icon: typeof CheckIcon; class: string }> = {
		given: { icon: CheckIcon, class: 'border-transparent bg-accent text-primary-ink' },
		due: { icon: CircleIcon, class: 'border-[1.5px] border-primary text-primary-ink' },
		overdue: {
			icon: ClockIcon,
			class: 'border-[1.5px] border-dashed border-primary text-primary-ink'
		},
		upcoming: { icon: CalendarIcon, class: '' },
		dismissed: { icon: MinusIcon, class: '' },
		available: { icon: PlusIcon, class: 'border-dotted border-muted-foreground' }
	};

	const title = $derived(groupTitle(group));
	const headingId = $derived(`vgroup-${group.fromDate}`);
	const openAntigens = $derived([
		...new Set(group.lines.filter((l) => OPEN.includes(l.status)).map((l) => l.antigen))
	]);
	const name = (l: VaccineLine) => `${antigenName(l.antigen, catalogue)}, dose ${l.series}`;
	// The API sorts a visit's lines by group, so one schedule group is one consecutive run.
	const clusters = $derived(
		group.lines.reduce<{ group: string; lines: VaccineLine[] }[]>((out, l) => {
			const last = out.at(-1);
			if (last && last.group === l.group) last.lines.push(l);
			else out.push({ group: l.group, lines: [l] });
			return out;
		}, [])
	);
</script>

{#snippet badge(status: VaccineStatus, id?: string)}
	{@const b = BADGE[status]}
	<Badge {id} class={b.class}><b.icon aria-hidden="true" />{statusLabel(status)}</Badge>
{/snippet}

<section aria-labelledby={headingId}>
	<details {open} class="group overflow-hidden rounded-2xl border bg-card">
		<summary
			class="flex min-h-14 cursor-pointer list-none items-center gap-2.5 px-3.5 [&::-webkit-details-marker]:hidden"
		>
			<h2
				id={headingId}
				class="flex min-w-0 flex-1 flex-wrap items-center justify-between gap-x-2 gap-y-1 text-base font-semibold [overflow-wrap:anywhere]"
			>
				<span>{title}</span>
				{@render badge(group.status)}
			</h2>
			<ChevronDownIcon
				aria-hidden="true"
				class="size-4 shrink-0 text-muted-foreground group-open:rotate-180 motion-safe:transition-transform"
			/>
		</summary>
		<div class="flex flex-col gap-2 px-3 pb-3">
			{#each clusters as cluster (cluster.lines[0].id)}
				<div class="flex flex-col gap-1.5">
					<h3 class="text-[13px] font-semibold text-muted-foreground">{cluster.group}</h3>
					<ul class="flex flex-col gap-2">
						{#each cluster.lines as line (line.id)}
							<li
								aria-labelledby="vline-{line.id}-name vline-{line.id}-status"
								class="flex min-w-0 flex-col gap-1.5 rounded-lg border bg-background p-3"
							>
								<div class="flex items-start justify-between gap-2">
									<span
										id="vline-{line.id}-name"
										class="min-w-0 text-[15px] font-semibold [overflow-wrap:anywhere]"
										>{name(line)}</span
									>
									{@render badge(line.status, `vline-${line.id}-status`)}
								</div>
								{#if line.status === 'given' && line.givenOn}
									<p class="text-[13.5px]">
										{#if line.administration}
											<a
												href={resolve('/(app)/kids/[kid]/vaccines/[administration]', {
													kid: kidId,
													administration: line.administration
												})}
												data-drawer="Edit vaccination"
												class="inline-flex min-h-11 items-center underline">Given {line.givenOn}</a
											>
										{:else}
											Given {line.givenOn}
										{/if}
									</p>
								{:else if line.status === 'dismissed' && line.dismissal}
									<p class="text-[13px] text-muted-foreground">
										{DISMISSED_TEXT[line.dismissal.reason]}
									</p>
								{:else if line.status !== 'available'}
									<p class="text-[12.5px] leading-[1.4] text-muted-foreground">
										{windowLabel(line)}
									</p>
								{/if}
								{#if line.note}
									<details class="text-[13px]">
										<summary class="flex min-h-11 cursor-pointer items-center"
											>Note<span class="sr-only"> on {name(line)}</span></summary
										>
										<p class="pb-2 [overflow-wrap:anywhere]">{line.note}</p>
									</details>
								{/if}
								{#if guardian && OPEN.includes(line.status)}
									<form
										method="POST"
										action="?/dismiss"
										class="flex flex-wrap items-center gap-2"
										use:enhance
									>
										<input type="hidden" name="line" value={line.id} />
										<label for="vline-{line.id}-reason" class="sr-only"
											>Reason for {name(line)}</label
										>
										<select
											id="vline-{line.id}-reason"
											name="reason"
											class={cn(SELECT_CLASS, 'w-auto flex-1 basis-40')}
										>
											<option value="not_needed">Not needed</option>
											<option value="given_unknown">Given, date unknown</option>
										</select>
										<Button type="submit" variant="outline"
											>Dismiss<span class="sr-only"> {name(line)}</span></Button
										>
									</form>
								{:else if guardian && line.status === 'dismissed'}
									<form method="POST" action="?/undoDismiss" use:enhance>
										<input type="hidden" name="line" value={line.id} />
										<Button type="submit" variant="outline"
											>Undo<span class="sr-only"> dismissal of {name(line)}</span></Button
										>
									</form>
								{/if}
							</li>
						{/each}
					</ul>
				</div>
			{/each}
			{#if openAntigens.length > 0}
				<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- the query has no resolve() form, and the link keeps an absolute path -->
				<Button
					href={vaccineNewPath(kidId, openAntigens)}
					data-drawer="Record a vaccination"
					variant="outline"
					class="w-full">Record {title}</Button
				>
			{/if}
		</div>
	</details>
</section>
