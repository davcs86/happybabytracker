<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import { keepValues } from '#lib/components/field.js';
	import MilestoneFields from '#lib/components/milestones/MilestoneFields.svelte';
	import { Alert } from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { formatAge } from '#lib/format.js';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	const values = $derived(form?.values ?? data.values);
	const error = (name: string) => (form?.field === name ? form.message : null);
	// The item has no input of its own: its refusal shows on the form, like a conflict.
	const top = $derived(form?.field == null || form.field === 'item' ? form?.message : null);
	const refused = $derived(form?.field != null && form.field !== 'item');
	const milestones = $derived(resolve('/(app)/kids/[kid]/milestones', { kid: data.kid.id }));
	// The item stays in the action's query, so a refused save without JavaScript still names it.
	const action = $derived(data.item ? `?/save&item=${data.item.id}` : '?/save');
</script>

<svelte:head>
	<title
		>{data.item ? 'Mark as reached' : 'Add a milestone'} · {data.kid.name} · happybabytracker</title
	>
</svelte:head>

<div class="max-w-[448px] xl:mx-auto xl:w-[920px] xl:max-w-none">
	<a href={milestones} class="inline-flex min-h-11 items-center gap-1.5 text-sm underline"
		><span aria-hidden="true">‹</span>Milestones for {data.kid.name}</a
	>
	<h1 class="mt-2">{data.item ? 'Mark as reached' : 'Add a milestone'}</h1>
	<p class="mt-1.5 text-sm text-muted-foreground">{formatAge(data.kid.birthDate, data.today)}</p>
	{#if data.item}
		<p class="mt-[18px] text-[15px] [overflow-wrap:anywhere]">{data.item.text}</p>
	{/if}
	<form
		method="POST"
		{action}
		class="mt-[18px] flex flex-col gap-3.5 xl:grid xl:grid-cols-[448px_448px] xl:grid-rows-[auto_1fr] xl:gap-x-6 xl:gap-y-[18px]"
		novalidate
		use:enhance={keepValues}
	>
		{#if top}
			<Alert class="xl:col-span-2">{top}</Alert>
		{:else if refused}
			<Alert class="xl:col-span-2">Check the fields with a message.</Alert>
		{/if}
		{#if data.item}
			<input type="hidden" name="item" value={data.item.id} />
		{/if}
		<MilestoneFields
			{values}
			{error}
			custom={!data.item}
			birthDate={data.kid.birthDate}
			today={data.today}
		/>
		<div class="flex gap-2">
			<Button type="submit" class="flex-1">Save</Button>
			<Button href={milestones} variant="outline">Cancel</Button>
		</div>
	</form>
</div>
