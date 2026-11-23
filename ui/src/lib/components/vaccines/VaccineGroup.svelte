<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import { SELECT_CLASS } from '#lib/components/field.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { vaccineNewPath } from '#lib/paths.js';
	import type { VaccineCatalogue, VaccineGroup, VaccineLine, VaccineStatus } from '#lib/types.js';
	import { antigenName, groupTitle, statusLabel, windowLabel } from '#lib/vaccines.js';

	let {
		group,
		kidId,
		catalogue,
		guardian
	}: {
		group: VaccineGroup;
		kidId: string;
		catalogue: VaccineCatalogue;
		guardian: boolean;
	} = $props();

	const OPEN: readonly VaccineStatus[] = ['due', 'overdue', 'upcoming', 'available'];
	const DISMISSED_TEXT = {
		not_needed: 'Dismissed: not needed',
		given_unknown: 'Dismissed: given, date unknown'
	} as const;
	const BADGE = 'rounded border px-1.5 text-xs font-medium';

	const title = $derived(groupTitle(group));
	const headingId = $derived(`vgroup-${group.fromDate}`);
	const open = $derived([
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

<section aria-labelledby={headingId} class="mb-6">
	<div class="mb-2 flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
		<h2 id={headingId} class="font-semibold">
			{title} <span class="{BADGE} ml-1">{statusLabel(group.status)}</span>
		</h2>
		{#if open.length > 0}
			<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- the query has no resolve() form, and the link keeps an absolute path -->
			<a href={vaccineNewPath(kidId, open)} class="text-sm underline"
				>Record<span class="sr-only"> {title}</span></a
			>
		{/if}
	</div>
	{#each clusters as cluster (cluster.lines[0].id)}
		<div class="mb-3 last:mb-0">
			<h3 class="mb-1 text-sm font-medium text-muted-foreground">{cluster.group}</h3>
			<ul class="flex flex-col gap-3">
				{#each cluster.lines as line (line.id)}
					<li
						aria-labelledby="vline-{line.id}-name vline-{line.id}-status"
						class="flex min-w-0 flex-col gap-1 rounded-lg border p-3"
					>
						<p class="flex flex-wrap items-baseline gap-x-2">
							<span id="vline-{line.id}-name" class="font-medium">{name(line)}</span>
							<span id="vline-{line.id}-status" class={BADGE}>{statusLabel(line.status)}</span>
						</p>
						{#if line.status === 'given' && line.givenOn}
							<p class="text-sm">
								{#if line.administration}
									<a
										href={resolve('/(app)/kids/[kid]/vaccines/[administration]', {
											kid: kidId,
											administration: line.administration
										})}
										class="underline">Given {line.givenOn}</a
									>
								{:else}
									Given {line.givenOn}
								{/if}
							</p>
						{:else if line.status === 'dismissed' && line.dismissal}
							<p class="text-sm">{DISMISSED_TEXT[line.dismissal.reason]}</p>
						{:else if line.status !== 'available'}
							<p class="text-sm text-muted-foreground">{windowLabel(line)}</p>
						{/if}
						{#if line.note}
							<details class="text-sm">
								<summary class="cursor-pointer"
									>Note<span class="sr-only"> on {name(line)}</span></summary
								>
								<p class="mt-1 [overflow-wrap:anywhere]">{line.note}</p>
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
								<label for="vline-{line.id}-reason" class="sr-only">Reason for {name(line)}</label>
								<select id="vline-{line.id}-reason" name="reason" class="{SELECT_CLASS} w-auto">
									<option value="not_needed">Not needed</option>
									<option value="given_unknown">Given, date unknown</option>
								</select>
								<Button type="submit" variant="outline" size="sm"
									>Dismiss<span class="sr-only"> {name(line)}</span></Button
								>
							</form>
						{:else if guardian && line.status === 'dismissed'}
							<form method="POST" action="?/undoDismiss" use:enhance>
								<input type="hidden" name="line" value={line.id} />
								<Button type="submit" variant="outline" size="sm"
									>Undo<span class="sr-only"> dismissal of {name(line)}</span></Button
								>
							</form>
						{/if}
					</li>
				{/each}
			</ul>
		</div>
	{/each}
</section>
