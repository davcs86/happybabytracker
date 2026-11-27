<script lang="ts">
	import ChildRow from '#lib/components/ChildRow.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import { FIND_CHILD_MIN } from '#lib/constants.js';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const shown = $derived(
		data.kids.filter((k) => k.name.toLocaleLowerCase().includes(data.q.trim().toLocaleLowerCase()))
	);
	const guardian = $derived(data.kids.some((k) => k.role === 'guardian'));
</script>

<svelte:head>
	<title>Choose a child · happybabytracker</title>
</svelte:head>

<!-- eslint-disable svelte/no-navigation-without-resolve -- ids come from the child list -->
<div
	class="mt-5 max-w-[448px] xl:mx-auto xl:grid xl:w-[920px] xl:max-w-none xl:grid-cols-[448px_448px] xl:items-start xl:gap-x-6 xl:gap-y-[18px]"
>
	<h1 class="xl:col-span-2">Choose a child</h1>
	<div class="contents xl:block">
		{#if data.kids.length > FIND_CHILD_MIN}
			<form method="GET" class="mt-4 grid gap-1.5">
				<input type="hidden" name="to" value={data.to} />
				<Label for="find-child">Find a child</Label>
				<div class="flex gap-2">
					<Input id="find-child" name="q" value={data.q} autocomplete="off" />
					<Button type="submit" variant="outline">Find</Button>
				</div>
			</form>
		{/if}
		<ul class="mt-4 grid gap-2">
			{#each shown as k (k.id)}
				<li>
					<ChildRow
						href="/kids/{k.id}{data.to ? `/${data.to}` : ''}"
						name={k.name}
						age={data.ages[k.id]}
						current={k.id === data.kid?.id}
					/>
				</li>
			{:else}
				<li class="py-3 text-muted-foreground">No child matches</li>
			{/each}
		</ul>
	</div>
	<div class="contents xl:block">
		{#if guardian}
			<Button href="/kids/new" data-drawer="Add a child" variant="outline" class="mt-4 w-full"
				>Add a child</Button
			>
		{:else}
			<p class="mt-4 text-[13.5px] text-muted-foreground">Ask a guardian to add a child.</p>
		{/if}
	</div>
</div>
