<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import CourseFields from '#lib/components/medication/CourseFields.svelte';
	import { ERROR_CLASS, keepValues } from '#lib/components/field.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	const values = $derived(form?.values ?? data.values);
	const error = (name: string) => (form?.field === name ? form.message : null);
	const medication = $derived(resolve('/(app)/kids/[kid]/medication', { kid: data.kid.id }));
</script>

<svelte:head>
	<title>Add a course · {data.kid.name} · happybabytracker</title>
</svelte:head>

<p class="mb-2 text-sm">
	<a href={medication} class="underline">Medication for {data.kid.name}</a>
</p>
<h1 class="mb-4 text-xl font-semibold">Add a course</h1>
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
	<CourseFields {values} {error} zone={data.kid.timeZone} startHint="Leave blank to start now" />
	<div class="flex items-center gap-4">
		<Button type="submit">Save</Button>
		<a href={medication} class="text-sm underline">Cancel</a>
	</div>
</form>
