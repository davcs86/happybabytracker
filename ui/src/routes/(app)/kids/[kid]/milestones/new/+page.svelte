<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import { ERROR_CLASS, keepValues } from '#lib/components/field.js';
	import MilestoneFields from '#lib/components/milestones/MilestoneFields.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	const values = $derived(form?.values ?? data.values);
	const error = (name: string) => (form?.field === name ? form.message : null);
	// The item has no input of its own: its refusal shows on the form, like a conflict.
	const top = $derived(form?.field == null || form.field === 'item' ? form?.message : null);
	const milestones = $derived(resolve('/(app)/kids/[kid]/milestones', { kid: data.kid.id }));
	// The item stays in the action's query, so a refused save without JavaScript still names it.
	const action = $derived(data.item ? `?/save&item=${data.item.id}` : '?/save');
</script>

<svelte:head>
	<title
		>{data.item ? 'Mark as reached' : 'Add a milestone'} · {data.kid.name} · happybabytracker</title
	>
</svelte:head>

<p class="mb-2 text-sm">
	<a href={milestones} class="underline">Milestones for {data.kid.name}</a>
</p>
<h1 class="mb-4 text-xl font-semibold">{data.item ? 'Mark as reached' : 'Add a milestone'}</h1>
<form
	method="POST"
	{action}
	class="flex max-w-md flex-col gap-4"
	novalidate
	use:enhance={keepValues}
>
	{#if top}
		<p class={ERROR_CLASS} role="alert">{top}</p>
	{/if}
	{#if data.item}
		<p class="text-sm [overflow-wrap:anywhere]">{data.item.text}</p>
		<input type="hidden" name="item" value={data.item.id} />
	{/if}
	<MilestoneFields
		{values}
		{error}
		custom={!data.item}
		birthDate={data.kid.birthDate}
		today={data.today}
	/>
	<div class="flex items-center gap-4">
		<Button type="submit">Save</Button>
		<a href={milestones} class="text-sm underline">Cancel</a>
	</div>
</form>
