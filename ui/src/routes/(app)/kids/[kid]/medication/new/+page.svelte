<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import CourseFields from '#lib/components/medication/CourseFields.svelte';
	import { keepValues } from '#lib/components/field.js';
	import { Alert } from '#lib/components/ui/alert/index.js';
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

<div class="flex max-w-[448px] flex-col gap-[18px] xl:mx-auto xl:w-[920px] xl:max-w-none">
	<div class="mt-5 flex flex-col gap-1.5">
		<h1>Add a course</h1>
		<p class="text-sm leading-snug text-muted-foreground">{data.ages[data.kid.id]}</p>
		<p class="text-[13.5px] text-muted-foreground">Times are in {data.kid.timeZone}</p>
	</div>
	{#if form && !form.field && form.message}
		<Alert class="[overflow-wrap:anywhere]">{form.message}</Alert>
	{/if}
	<form
		method="POST"
		action="?/save"
		class="flex flex-col gap-[18px] xl:grid xl:grid-cols-[448px_448px] xl:grid-rows-[auto_1fr] xl:gap-x-6"
		novalidate
		use:enhance={keepValues}
	>
		<CourseFields {values} {error} startHint="Leave blank to start now" />
		<div class="flex gap-2">
			<Button type="submit" class="flex-1">Save</Button>
			<Button href={medication} variant="outline">Cancel</Button>
		</div>
	</form>
</div>
