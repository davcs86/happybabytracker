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
	const vaccines = $derived(resolve('/(app)/kids/[kid]/vaccines', { kid: data.kid.id }));
</script>

<svelte:head>
	<title>Edit vaccination · {data.kid.name} · happybabytracker</title>
</svelte:head>

<div class="max-w-[448px] xl:mx-auto xl:w-[920px] xl:max-w-none">
	<a href={vaccines} class="inline-flex min-h-11 items-center gap-1.5 text-sm underline"
		><span aria-hidden="true">‹</span>Vaccines for {data.kid.name}</a
	>
	<h1 class="mt-2">Edit vaccination</h1>
	<p class="mt-1.5 text-sm text-muted-foreground">{formatAge(data.kid.birthDate, data.today)}</p>
	<form
		method="POST"
		action="?/save"
		class="mt-[18px] flex flex-col gap-3.5 xl:grid xl:grid-cols-[448px_448px] xl:grid-rows-[auto_1fr] xl:gap-x-6 xl:gap-y-[18px]"
		novalidate
		use:enhance={keepValues}
	>
		{#if save && !save.field && save.message}
			<Alert class="xl:col-span-2">{save.message}</Alert>
		{:else if save?.field}
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

	<!-- Two steps without JavaScript: the button exists only inside the opened details.
     A failure reopens it, so its message shows after a full-page POST too. -->
	<details
		class="mt-[18px] overflow-hidden rounded-2xl border bg-card px-3.5"
		open={!!del?.message}
	>
		<summary class="flex min-h-14 cursor-pointer items-center text-[15px] font-semibold"
			>Delete this vaccination</summary
		>
		<form method="POST" action="?/delete" class="flex flex-col gap-3 pb-3.5" use:enhance>
			{#if del?.message}
				<Alert>{del.message}</Alert>
			{/if}
			<p class="text-[13.5px] leading-normal [overflow-wrap:anywhere]">
				This removes the dose for everyone with access to {data.kid.name}.
			</p>
			<Button type="submit" variant="destructive" class="self-start"
				>Yes, delete this vaccination</Button
			>
		</form>
	</details>
</div>
