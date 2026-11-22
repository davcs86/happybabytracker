<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import MeasurementFields from '#lib/components/growth/MeasurementFields.svelte';
	import { ERROR_CLASS, keepValues } from '#lib/components/field.js';
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

<p class="mb-2 text-sm">
	<a href={growth} class="underline">Growth for {data.kid.name}</a>
</p>
<h1 class="mb-4 text-xl font-semibold">Add a measurement</h1>
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
	<MeasurementFields {values} {error} birthDate={data.kid.birthDate} today={data.today} />
	<div class="flex items-center gap-4">
		<Button type="submit">Save</Button>
		<a href={growth} class="text-sm underline">Cancel</a>
	</div>
</form>
