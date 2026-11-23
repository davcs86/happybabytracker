<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import DoseFields from '#lib/components/vaccines/DoseFields.svelte';
	import { ERROR_CLASS, keepValues } from '#lib/components/field.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	const values = $derived(form?.values ?? data.values);
	const error = (name: string) => (form?.field === name ? form.message : null);
	const vaccines = $derived(resolve('/(app)/kids/[kid]/vaccines', { kid: data.kid.id }));
</script>

<svelte:head>
	<title>Record a vaccination · {data.kid.name} · happybabytracker</title>
</svelte:head>

<p class="mb-2 text-sm">
	<a href={vaccines} class="underline">Vaccines for {data.kid.name}</a>
</p>
<h1 class="mb-4 text-xl font-semibold">Record a vaccination</h1>
<form
	method="POST"
	action="?/save"
	class="flex max-w-md flex-col gap-4"
	novalidate
	use:enhance={keepValues}
>
	{#if form && !form.field && form.message}
		<p class={ERROR_CLASS} role="alert">{form.message}</p>
	{/if}
	<DoseFields
		{values}
		{error}
		catalogue={data.catalogue}
		birthDate={data.kid.birthDate}
		today={data.today}
	/>
	<div class="flex items-center gap-4">
		<Button type="submit">Save</Button>
		<a href={vaccines} class="text-sm underline">Cancel</a>
	</div>
</form>
