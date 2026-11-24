<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import { ERROR_CLASS, keepValues } from '#lib/components/field.js';
	import MilestoneFields from '#lib/components/milestones/MilestoneFields.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	const save = $derived(form?.form === 'save' ? form : null);
	const del = $derived(form?.form === 'delete' ? form : null);
	// The originals are the loaded ones, never the posted.
	const values = $derived({
		...data.values,
		...Object.fromEntries(
			Object.entries(save?.values ?? {}).filter(([name]) => !name.endsWith('_original'))
		)
	});
	const error = (name: string) => (save?.field === name ? save.message : null);
	const top = $derived(save?.field == null || save.field === 'item' ? save?.message : null);
	const milestones = $derived(resolve('/(app)/kids/[kid]/milestones', { kid: data.kid.id }));
	// A list item's delete is an undo: the item goes back to its list.
	const words = $derived(
		data.custom
			? {
					summary: 'Delete this milestone',
					body: 'This removes the milestone',
					confirm: 'Yes, delete this milestone'
				}
			: { summary: 'Undo reached', body: 'This removes the date', confirm: 'Yes, undo reached' }
	);
</script>

<svelte:head>
	<title>Edit milestone · {data.kid.name} · happybabytracker</title>
</svelte:head>

<p class="mb-2 text-sm">
	<a href={milestones} class="underline">Milestones for {data.kid.name}</a>
</p>
<h1 class="mb-4 text-xl font-semibold">Edit milestone</h1>
<form
	method="POST"
	action="?/save"
	class="flex max-w-md flex-col gap-4"
	novalidate
	use:enhance={keepValues}
>
	{#if top}
		<p class={ERROR_CLASS} role="alert">{top}</p>
	{/if}
	{#if !data.custom}
		<p class="text-sm [overflow-wrap:anywhere]">{data.text}</p>
	{/if}
	<MilestoneFields
		{values}
		{error}
		custom={data.custom}
		birthDate={data.kid.birthDate}
		today={data.today}
	/>
	<div class="flex items-center gap-4">
		<Button type="submit">Save</Button>
		<a href={milestones} class="text-sm underline">Cancel</a>
	</div>
</form>

<!-- Two steps without JavaScript: the button exists only inside the opened details.
     A failure reopens it, so its message shows after a full-page POST too. -->
<details class="mt-6 max-w-md rounded-lg border p-3" open={!!del?.message}>
	<summary class="cursor-pointer text-sm font-medium">{words.summary}</summary>
	<form method="POST" action="?/delete" class="mt-3 flex flex-col gap-3" use:enhance>
		{#if del?.message}
			<p class={ERROR_CLASS} role="alert">{del.message}</p>
		{/if}
		<p class="text-sm [overflow-wrap:anywhere]">
			{words.body} for everyone with access to {data.kid.name}.
		</p>
		<Button type="submit" variant="outline" class="self-start border-destructive text-destructive"
			>{words.confirm}</Button
		>
	</form>
</details>
