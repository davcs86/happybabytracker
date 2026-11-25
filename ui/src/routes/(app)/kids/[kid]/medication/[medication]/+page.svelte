<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import CourseFields from '#lib/components/medication/CourseFields.svelte';
	import { Alert } from '#lib/components/ui/alert/index.js';
	import FormCancel from '#lib/components/FormCancel.svelte';
	import FormFrame from '#lib/components/FormFrame.svelte';
	import { Button, buttonVariants } from '#lib/components/ui/button/index.js';
	import { isEmbed, savedAs } from '#lib/drawer.js';
	import { doseCountLabel } from '#lib/medication.js';
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
	const embed = isEmbed();
	const medication = $derived(resolve('/(app)/kids/[kid]/medication', { kid: data.kid.id }));
</script>

{#snippet info()}
	<p class="text-[13.5px] [overflow-wrap:anywhere] text-muted-foreground">{data.course.name}</p>
	<p class="text-[13.5px] text-muted-foreground">Times are in {data.kid.timeZone}</p>
{/snippet}

<FormFrame
	class="flex flex-col gap-[18px]"
	documentTitle="Edit course · {data.kid.name} · happybabytracker"
>
	{#snippet chrome()}
		<div class="mt-5 flex flex-col gap-1.5">
			<h1>Edit course</h1>
			<p class="text-sm leading-snug text-muted-foreground">{data.ages[data.kid.id]}</p>
			{@render info()}
		</div>
	{/snippet}
	{#if embed}
		<div class="flex flex-col gap-1.5">{@render info()}</div>
	{/if}
	{#if save && !save.field && save.message}
		<Alert class="[overflow-wrap:anywhere]">{save.message}</Alert>
	{/if}
	<form
		method="POST"
		action="?/save"
		class={[
			'flex flex-col gap-[18px]',
			!embed && 'xl:grid xl:grid-cols-[448px_448px] xl:grid-rows-[auto_1fr] xl:gap-x-6'
		]}
		novalidate
		use:enhance={savedAs('Course saved')}
	>
		<CourseFields {values} {error} startHint="Leave blank to keep the current start" />
		{#each ['name', 'dose', 'schedule', 'start_at', 'end_on', 'active', 'note'] as const as k (k)}
			<input type="hidden" name="{k}_original" value={data.values[`${k}_original`]} />
		{/each}
		<div class="flex gap-2">
			<Button type="submit" size={embed ? 'lg' : 'default'} class="flex-1">Save</Button>
			<FormCancel href={medication} size="default" class="" />
		</div>
	</form>

	<!-- Two steps without JavaScript: the button exists only inside the opened details.
     A failure reopens it, so its message shows after a full-page POST too. -->
	<details class="max-w-[448px] border-t pt-4" open={!!del?.message}>
		<summary
			class="{buttonVariants({
				variant: 'destructive'
			})} w-full cursor-pointer list-none [&::-webkit-details-marker]:hidden"
			>Delete this course</summary
		>
		<form
			method="POST"
			action="?/delete"
			class="flex flex-col gap-2.5 pt-3"
			use:enhance={savedAs('Course deleted')}
		>
			{#if del?.message}
				<Alert class="[overflow-wrap:anywhere]">{del.message}</Alert>
			{/if}
			<p class="text-[13.5px] [overflow-wrap:anywhere] text-muted-foreground">
				This removes the course for everyone with access to {data.kid.name}.
			</p>
			<p class="text-[13.5px] text-muted-foreground">
				This also deletes the {doseCountLabel(data.course.doseCount)} logged for this course.
			</p>
			<Button type="submit" variant="destructive">Yes, delete this course</Button>
		</form>
	</details>
</FormFrame>
