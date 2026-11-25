<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import MeasurementFields from '#lib/components/growth/MeasurementFields.svelte';
	import { keepValues } from '#lib/components/field.js';
	import { Alert } from '#lib/components/ui/alert/index.js';
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
	const growth = $derived(resolve('/(app)/kids/[kid]/growth', { kid: data.kid.id }));
</script>

<svelte:head>
	<title>Edit measurement · {data.kid.name} · happybabytracker</title>
</svelte:head>

<div class="max-w-[448px] xl:mx-auto xl:w-[920px] xl:max-w-none">
	<a href={growth} class="inline-flex min-h-11 items-center gap-1 text-sm underline"
		><span aria-hidden="true">‹</span>Growth for {data.kid.name}</a
	>
	<h1 class="mt-1">Edit measurement</h1>
	<form
		method="POST"
		action="?/save"
		class="mt-[18px] flex flex-col gap-[18px] xl:grid xl:grid-cols-[448px_448px] xl:grid-rows-[auto_1fr] xl:gap-x-6"
		novalidate
		use:enhance={keepValues}
	>
		{#if save?.field}
			<Alert class="xl:col-span-2">Check the fields with a message.</Alert>
		{:else if save?.message}
			<Alert class="xl:col-span-2">{save.message}</Alert>
		{/if}
		<MeasurementFields {values} {error} birthDate={data.kid.birthDate} today={data.today} edit />
		<div class="flex gap-2">
			<Button type="submit" size="lg" class="flex-1">Save</Button>
			<Button href={growth} variant="outline" size="lg" class="flex-1">Cancel</Button>
		</div>
	</form>

	<!-- Two steps without JavaScript: the button exists only inside the opened details.
     A failure reopens it, so its message shows after a full-page POST too. -->
	<details class="mt-[18px]" open={!!del?.message}>
		<summary
			class="flex min-h-12 cursor-pointer list-none items-center justify-center rounded-lg border bg-card px-4 text-[15px] font-semibold [&::-webkit-details-marker]:hidden"
			>Delete this measurement</summary
		>
		<form
			method="POST"
			action="?/delete"
			class="mt-3 flex flex-col gap-3 rounded-lg border bg-card p-3"
			use:enhance
		>
			{#if del?.message}
				<Alert>{del.message}</Alert>
			{/if}
			<p class="text-[13.5px] leading-normal">
				This removes the measurement for everyone with access to {data.kid.name}.
			</p>
			<Button type="submit" variant="destructive" class="self-start"
				>Yes, delete this measurement</Button
			>
		</form>
	</details>
</div>
