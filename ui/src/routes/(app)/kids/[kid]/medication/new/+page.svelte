<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import CourseFields from '#lib/components/medication/CourseFields.svelte';
	import { Alert } from '#lib/components/ui/alert/index.js';
	import FormCancel from '#lib/components/FormCancel.svelte';
	import FormFrame from '#lib/components/FormFrame.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import { isEmbed, savedAs } from '#lib/drawer.js';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	const values = $derived(form?.values ?? data.values);
	const error = (name: string) => (form?.field === name ? form.message : null);
	const embed = isEmbed();
	const medication = $derived(resolve('/(app)/kids/[kid]/medication', { kid: data.kid.id }));
</script>

{#snippet info()}
	<p class="text-[13.5px] text-muted-foreground">Times are in {data.kid.timeZone}</p>
{/snippet}

<FormFrame
	class="flex flex-col gap-[18px]"
	documentTitle="Add a course · {data.kid.name} · happybabytracker"
>
	{#snippet chrome()}
		<div class="mt-5 flex flex-col gap-1.5">
			<h1>Add a course</h1>
			<p class="text-sm leading-snug text-muted-foreground">{data.ages[data.kid.id]}</p>
			{@render info()}
		</div>
	{/snippet}
	{#if embed}
		<div class="flex flex-col gap-1.5">{@render info()}</div>
	{/if}
	{#if form && !form.field && form.message}
		<Alert class="[overflow-wrap:anywhere]">{form.message}</Alert>
	{/if}
	<form
		method="POST"
		action="?/save"
		class={[
			'flex flex-col gap-[18px]',
			!embed && 'xl:grid xl:grid-cols-[448px_448px] xl:grid-rows-[auto_1fr] xl:gap-x-6'
		]}
		novalidate
		use:enhance={savedAs('Course added')}
	>
		<CourseFields {values} {error} startHint="Leave blank to start now" />
		<div class="flex gap-2">
			<Button type="submit" size={embed ? 'lg' : 'default'} class="flex-1">Save</Button>
			<FormCancel href={medication} size="default" class="" />
		</div>
	</form>
</FormFrame>
