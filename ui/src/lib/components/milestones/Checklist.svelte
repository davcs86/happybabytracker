<script lang="ts">
	import { untrack } from 'svelte';
	import { enhance } from '$app/forms';
	import CalendarIcon from '@lucide/svelte/icons/calendar';
	import ChevronDownIcon from '@lucide/svelte/icons/chevron-down';
	import PencilIcon from '@lucide/svelte/icons/pencil';
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

<details
	id="list-{checklist.ageMonths}"
	class="group overflow-hidden rounded-2xl border bg-card"
	bind:open
>
	<summary
		class="flex min-h-14 cursor-pointer list-none items-center gap-2.5 px-3.5 [&::-webkit-details-marker]:hidden"
	>
		<span class="min-w-0 flex-1 text-[15px] font-semibold [overflow-wrap:anywhere]"
			>{checklistTitle(checklist.ageMonths)}{#if marker}<span
					class="font-[400] text-muted-foreground">{` · ${marker}`}</span
				>{/if}</span
		>
		<ChevronDownIcon
			aria-hidden="true"
			data-chrome
			class="size-4 shrink-0 text-muted-foreground group-open:rotate-180 motion-safe:transition-transform"
		/>
	</summary>
	<div class="flex flex-col gap-3.5 px-3 pb-3">
		{#each MILESTONE_DOMAINS as domain (domain)}
			{@const items = checklist.items.filter((i) => i.domain === domain)}
			{#if items.length > 0}
				<div class="flex flex-col gap-2">
					<h3 class="text-sm font-semibold">{domainLabel(domain)}</h3>
					<ul class="flex flex-col gap-2">
						{#each items as item (item.id)}
							<li
								id="item-{item.id}"
								class="flex flex-col gap-2 rounded-lg border bg-background p-3 text-[15px] [overflow-wrap:anywhere]"
							>
								<span>{item.text}</span>
								{#if failed?.item === item.id}
									<p class={ERROR_CLASS} role="alert">{failed.message}</p>
								{/if}
								{#if item.reached}
									<span
										class="flex flex-wrap items-center gap-x-3 gap-y-1 text-sm text-muted-foreground"
									>
										{reachedLabel(item.reached.reachedOn)}
										<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- the entry page's path helper, absolute like the other links to pages without JavaScript -->
										<Button
											href={milestoneEntryPath(kidId, item.reached.id)}
											data-drawer="Edit milestone"
											variant="outline"
											size="icon"
											title="Edit"
											class="[&_svg]:size-5"
											><PencilIcon aria-hidden="true" data-chrome /><span class="sr-only"
												>Edit {item.text}</span
											></Button
										>
									</span>
								{:else}
									<form
										method="POST"
										action="?/reach#item-{item.id}"
										class="flex flex-wrap items-center gap-2"
										use:enhance
									>
										<input type="hidden" name="item" value={item.id} />
										<Button type="submit" variant="outline" class="px-5"
											>Reached<span class="sr-only"> {item.text}</span></Button
										>
										<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- the new page's path helper carries the item in its query -->
										<Button
											href={milestoneNewPath(kidId, item.id)}
											data-drawer="Mark as reached"
											variant="outline"
											size="icon"
											title="Choose a date"
											class="[&_svg]:size-5"
											><CalendarIcon aria-hidden="true" data-chrome /><span class="sr-only"
												>Choose a date for {item.text}</span
											></Button
										>
									</form>
								{/if}
							</li>
						{/each}
					</ul>
				</div>
			{/if}
		{/each}
	</div>
</details>
