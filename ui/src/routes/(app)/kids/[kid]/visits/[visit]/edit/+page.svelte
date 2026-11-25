<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import FormCancel from '#lib/components/FormCancel.svelte';
	import FormFrame from '#lib/components/FormFrame.svelte';
	import VisitFields from '#lib/components/visits/VisitFields.svelte';
	import { Alert } from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { isEmbed, savedAs } from '#lib/drawer.js';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	// The originals are the loaded ones, never the posted.
	const values = $derived({
		...data.values,
		...Object.fromEntries(
			Object.entries(form?.values ?? {}).filter(([name]) => !name.endsWith('_original'))
		)
	});
	const error = (name: string) => (form?.field === name ? form.message : null);
	const embed = isEmbed();
	const back = $derived(
		resolve('/(app)/kids/[kid]/visits/[visit]', { kid: data.kid.id, visit: data.visit.id })
	);
</script>

{#snippet info()}
	<p class="text-[13.5px] [overflow-wrap:anywhere] text-muted-foreground">
		Times are in {data.kid.timeZone}
	</p>
{/snippet}

<FormFrame
	class="flex flex-col gap-[18px]"
	documentTitle="Edit visit · {data.kid.name} · happybabytracker"
>
	{#snippet chrome()}
		<div class="mt-5 flex flex-col gap-1.5">
			<h1>Edit visit</h1>
			<p class="text-sm leading-snug text-muted-foreground">{data.ages[data.kid.id]}</p>
			{@render info()}
		</div>
	{/snippet}
	{#if embed}
		<div class="flex flex-col gap-1.5">{@render info()}</div>
	{/if}
	<form
		method="POST"
		action="?/save"
		class={[
			'flex flex-col gap-[18px]',
			!embed && 'xl:grid xl:grid-cols-[448px_448px] xl:grid-rows-[auto_1fr] xl:gap-x-6'
		]}
		novalidate
		use:enhance={savedAs('Visit saved')}
	>
		{#if form?.field}
			<Alert class={embed ? '' : 'xl:col-span-2'}>Check the fields with a message.</Alert>
		{:else if form?.message}
			<Alert class="[overflow-wrap:anywhere] {embed ? '' : 'xl:col-span-2'}">{form.message}</Alert>
		{/if}
		<VisitFields {values} {error} />
		<!-- The time field carries its own original. -->
		{#each ['duration_min', 'kind', 'provider', 'location', 'note'] as const as k (k)}
			<input type="hidden" name="{k}_original" value={data.values[`${k}_original`]} />
		{/each}
		<div class="flex gap-2">
			<Button type="submit" size={embed ? 'lg' : 'default'} class="flex-1">Save</Button>
			<FormCancel href={back} size="default" class="" />
		</div>
	</form>
</FormFrame>
