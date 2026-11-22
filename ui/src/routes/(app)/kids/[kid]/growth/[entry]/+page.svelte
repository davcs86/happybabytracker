<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import MeasurementFields from '#lib/components/growth/MeasurementFields.svelte';
	import { ERROR_CLASS, keepValues } from '#lib/components/field.js';
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

<p class="mb-2 text-sm">
	<a href={growth} class="underline">Growth for {data.kid.name}</a>
</p>
<h1 class="mb-4 text-xl font-semibold">Edit measurement</h1>
<form
	method="POST"
	action="?/save"
	class="flex max-w-md flex-col gap-4"
	novalidate
	use:enhance={keepValues}
>
	{#if save && !save.field && save.message}
		<p class={ERROR_CLASS} role="alert">{save.message}</p>
	{/if}
	<MeasurementFields {values} {error} birthDate={data.kid.birthDate} today={data.today} edit />
	<div class="flex items-center gap-4">
		<Button type="submit">Save</Button>
		<a href={growth} class="text-sm underline">Cancel</a>
	</div>
</form>

<!-- Two steps without JavaScript: the button exists only inside the opened details.
     A failure reopens it, so its message shows after a full-page POST too. -->
<details class="mt-6 max-w-md rounded-lg border p-3" open={!!del?.message}>
	<summary class="cursor-pointer text-sm font-medium">Delete this measurement</summary>
	<form method="POST" action="?/delete" class="mt-3 flex flex-col gap-3" use:enhance>
		{#if del?.message}
			<p class={ERROR_CLASS} role="alert">{del.message}</p>
		{/if}
		<p class="text-sm">This removes the measurement for everyone with access to {data.kid.name}.</p>
		<Button type="submit" variant="outline" class="self-start border-destructive text-destructive"
			>Yes, delete this measurement</Button
		>
	</form>
</details>
