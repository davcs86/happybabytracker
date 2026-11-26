<script lang="ts">
	import { page } from '$app/state';
	import ChildRow from './ChildRow.svelte';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import * as Sheet from '#lib/components/ui/sheet/index.js';
	import { FIND_CHILD_MIN } from '#lib/constants.js';
	import { switchTarget } from '#lib/switch-target.js';
	import type { Kid } from '#lib/types.js';

	let {
		open = $bindable(false),
		kids,
		ages,
		currentId
	}: {
		open: boolean;
		kids: Pick<Kid, 'id' | 'name' | 'role'>[];
		ages: Record<string, string>;
		currentId: string | null;
	} = $props();

	let query = $state('');
	const to = $derived(switchTarget(page.url.pathname));
	const shown = $derived(
		kids.filter((k) => k.name.toLocaleLowerCase().includes(query.trim().toLocaleLowerCase()))
	);
	const guardian = $derived(kids.some((k) => k.role === 'guardian'));
</script>

<!-- eslint-disable svelte/no-navigation-without-resolve -- ids come from the child list -->
<Sheet.Root bind:open>
	<Sheet.Content side="bottom">
		<Sheet.Header>
			<Sheet.Title>Switch child</Sheet.Title>
			<Sheet.Description class="sr-only">Pick the child to show.</Sheet.Description>
		</Sheet.Header>
		{#if kids.length > FIND_CHILD_MIN}
			<div class="grid gap-1.5">
				<Label for="find-child">Find a child</Label>
				<Input id="find-child" bind:value={query} autocomplete="off" />
			</div>
		{/if}
		<ul class="grid gap-2">
			{#each shown as k (k.id)}
				<li>
					<ChildRow
						href="/kids/{k.id}{to ? `/${to}` : ''}"
						name={k.name}
						age={ages[k.id]}
						current={k.id === currentId}
					/>
				</li>
			{:else}
				<li class="py-3 text-muted-foreground">No child matches</li>
			{/each}
		</ul>
		{#if guardian}
			<a
				href="/kids/new"
				data-drawer="Add a child"
				class="flex min-h-12 items-center font-semibold text-primary">Add a child</a
			>
		{:else}
			<p class="text-[13.5px] text-muted-foreground">Ask a guardian to add a child.</p>
		{/if}
	</Sheet.Content>
</Sheet.Root>
