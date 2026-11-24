<script lang="ts">
	import { untrack } from 'svelte';
	import { enhance } from '$app/forms';
	import { ERROR_CLASS } from '#lib/components/field.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { MILESTONE_DOMAINS } from '#lib/constants.js';
	import { checklistTitle, domainLabel, reachedLabel } from '#lib/milestones.js';
	import { milestoneEntryPath, milestoneNewPath } from '#lib/paths.js';
	import type { MilestoneChecklist } from '#lib/types.js';

	// marker names the open list: the current one, or the next one before the first list age.
	// target's list renders open, so a page without JavaScript shows the item it returns to.
	let {
		kidId,
		checklist,
		marker,
		target,
		failed
	}: {
		kidId: string;
		checklist: MilestoneChecklist;
		marker: string | null;
		target: string | null;
		failed: { item: string; message: string } | null;
	} = $props();

	// Set once: a reload of the data after a mark must not close a list the reader opened.
	let open = $state(untrack(() => marker !== null || checklist.items.some((i) => i.id === target)));
</script>

<details id="list-{checklist.ageMonths}" class="rounded-lg border p-3" bind:open>
	<summary class="cursor-pointer font-medium"
		>{checklistTitle(checklist.ageMonths)}{marker ? ` · ${marker}` : ''}</summary
	>
	{#each MILESTONE_DOMAINS as domain (domain)}
		{@const items = checklist.items.filter((i) => i.domain === domain)}
		{#if items.length > 0}
			<h3 class="mt-3 mb-2 text-sm font-semibold">{domainLabel(domain)}</h3>
			<ul class="flex flex-col gap-2">
				{#each items as item (item.id)}
					<li id="item-{item.id}" class="flex flex-col gap-1 text-sm [overflow-wrap:anywhere]">
						<span>{item.text}</span>
						{#if failed?.item === item.id}
							<p class={ERROR_CLASS} role="alert">{failed.message}</p>
						{/if}
						{#if item.reached}
							<span class="flex flex-wrap gap-x-3">
								{reachedLabel(item.reached.reachedOn)}
								<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- the entry page's path helper, absolute like the other links to pages without JavaScript -->
								<a href={milestoneEntryPath(kidId, item.reached.id)} class="underline"
									>Edit<span class="sr-only"> {item.text}</span></a
								>
							</span>
						{:else}
							<form
								method="POST"
								action="?/reach"
								class="flex flex-wrap items-center gap-x-3 gap-y-1"
								use:enhance
							>
								<input type="hidden" name="item" value={item.id} />
								<Button type="submit" variant="outline"
									>Reached<span class="sr-only"> {item.text}</span></Button
								>
								<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- the new page's path helper carries the item in its query -->
								<a href={milestoneNewPath(kidId, item.id)} class="underline"
									>Choose a date<span class="sr-only"> for {item.text}</span></a
								>
							</form>
						{/if}
					</li>
				{/each}
			</ul>
		{/if}
	{/each}
</details>
