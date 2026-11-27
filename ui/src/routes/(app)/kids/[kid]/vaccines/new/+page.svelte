<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import { Alert } from '#lib/components/ui/alert/index.js';
	import FormCancel from '#lib/components/FormCancel.svelte';
	import FormFrame from '#lib/components/FormFrame.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import { isEmbed, savedAs } from '#lib/drawer.js';
	import DoseFields from '#lib/components/vaccines/DoseFields.svelte';
	import { formatAge } from '#lib/format.js';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	const values = $derived(form?.values ?? data.values);
	const error = (name: string) => (form?.field === name ? form.message : null);
	const embed = isEmbed();
	const vaccines = $derived(resolve('/(app)/kids/[kid]/vaccines', { kid: data.kid.id }));
</script>

<FormFrame documentTitle="Record a vaccination · {data.kid.name} · happybabytracker">
	{#snippet chrome()}
		<a href={vaccines} class="inline-flex min-h-11 items-center gap-1.5 text-sm underline"
			><span aria-hidden="true">‹</span>Vaccines for {data.kid.name}</a
		>
		<h1 class="mt-2">Record a vaccination</h1>
		<p class="mt-1.5 text-sm text-muted-foreground">{formatAge(data.kid.birthDate, data.today)}</p>
	{/snippet}
	<form
		method="POST"
		action="?/save"
		class={[
			'flex flex-col gap-3.5',
			!embed &&
				'mt-[18px] xl:grid xl:grid-cols-[448px_448px] xl:grid-rows-[auto_1fr] xl:gap-x-6 xl:gap-y-[18px]'
		]}
		novalidate
		use:enhance={savedAs('Vaccination recorded')}
	>
		{#if form && !form.field && form.message}
			<Alert class="xl:col-span-2">{form.message}</Alert>
		{:else if form?.field}
			<Alert class="xl:col-span-2">Check the fields with a message.</Alert>
		{/if}
		<DoseFields
			{values}
			{error}
			catalogue={data.catalogue}
			birthDate={data.kid.birthDate}
			today={data.today}
		/>
		<div class="flex gap-2">
			<Button type="submit" size={embed ? 'lg' : 'default'} class="flex-1">Save</Button>
			<FormCancel href={vaccines} size="default" class="" />
		</div>
	</form>
</FormFrame>
