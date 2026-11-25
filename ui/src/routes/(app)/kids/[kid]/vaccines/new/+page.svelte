<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import { keepValues } from '#lib/components/field.js';
	import { Alert } from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import DoseFields from '#lib/components/vaccines/DoseFields.svelte';
	import { formatAge } from '#lib/format.js';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	const values = $derived(form?.values ?? data.values);
	const error = (name: string) => (form?.field === name ? form.message : null);
	const vaccines = $derived(resolve('/(app)/kids/[kid]/vaccines', { kid: data.kid.id }));
</script>

<svelte:head>
	<title>Record a vaccination · {data.kid.name} · happybabytracker</title>
</svelte:head>

<div class="max-w-[448px] xl:mx-auto xl:w-[920px] xl:max-w-none">
	<a href={vaccines} class="inline-flex min-h-11 items-center gap-1.5 text-sm underline"
		><span aria-hidden="true">‹</span>Vaccines for {data.kid.name}</a
	>
	<h1 class="mt-2">Record a vaccination</h1>
	<p class="mt-1.5 text-sm text-muted-foreground">{formatAge(data.kid.birthDate, data.today)}</p>
	<form
		method="POST"
		action="?/save"
		class="mt-[18px] flex flex-col gap-3.5 xl:grid xl:grid-cols-[448px_448px] xl:grid-rows-[auto_1fr] xl:gap-x-6 xl:gap-y-[18px]"
		novalidate
		use:enhance={keepValues}
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
			<Button type="submit" class="flex-1">Save</Button>
			<Button href={vaccines} variant="outline">Cancel</Button>
		</div>
	</form>
</div>
