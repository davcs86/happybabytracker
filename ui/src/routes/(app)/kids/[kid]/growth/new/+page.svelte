<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import MeasurementFields from '#lib/components/growth/MeasurementFields.svelte';
	import { keepValues } from '#lib/components/field.js';
	import { Alert } from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	const values = $derived(form?.values ?? data.values);
	const error = (name: string) => (form?.field === name ? form.message : null);
	const growth = $derived(resolve('/(app)/kids/[kid]/growth', { kid: data.kid.id }));
</script>

<svelte:head>
	<title>Add a measurement · {data.kid.name} · happybabytracker</title>
</svelte:head>

<div class="max-w-[448px] xl:mx-auto xl:w-[920px] xl:max-w-none">
	<a href={growth} class="inline-flex min-h-11 items-center gap-1 text-sm underline"
		><span aria-hidden="true">‹</span>Growth for {data.kid.name}</a
	>
	<h1 class="mt-1">Add a measurement</h1>
	<form
		method="POST"
		action="?/save"
		class="mt-[18px] flex flex-col gap-[18px] xl:grid xl:grid-cols-[448px_448px] xl:grid-rows-[auto_1fr] xl:gap-x-6"
		novalidate
		use:enhance={keepValues}
	>
		{#if form?.field}
			<Alert class="xl:col-span-2">Check the fields with a message.</Alert>
		{:else if form?.message}
			<Alert class="xl:col-span-2">{form.message}</Alert>
		{/if}
		<MeasurementFields {values} {error} birthDate={data.kid.birthDate} today={data.today} />
		<div class="flex gap-2">
			<Button type="submit" size="lg" class="flex-1">Save</Button>
			<Button href={growth} variant="outline" size="lg" class="flex-1">Cancel</Button>
		</div>
	</form>
</div>
