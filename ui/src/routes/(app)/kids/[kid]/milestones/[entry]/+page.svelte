<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import MilestoneFields from '#lib/components/milestones/MilestoneFields.svelte';
	import { Alert } from '#lib/components/ui/alert/index.js';
	import FormCancel from '#lib/components/FormCancel.svelte';
	import FormFrame from '#lib/components/FormFrame.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import { isEmbed, savedAs } from '#lib/drawer.js';
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
	const top = $derived(save?.field == null || save.field === 'item' ? save?.message : null);
	const refused = $derived(save?.field != null && save.field !== 'item');
	const embed = isEmbed();
	const milestones = $derived(resolve('/(app)/kids/[kid]/milestones', { kid: data.kid.id }));
	// A list item's delete is an undo: the item goes back to its list.
	const words = $derived(
		data.custom
			? {
					summary: 'Delete this milestone',
					body: 'This removes the milestone',
					confirm: 'Yes, delete this milestone'
				}
			: { summary: 'Undo reached', body: 'This removes the date', confirm: 'Yes, undo reached' }
	);
</script>

<FormFrame documentTitle="Edit milestone · {data.kid.name} · happybabytracker">
	{#snippet chrome()}
		<a href={milestones} class="inline-flex min-h-11 items-center gap-1.5 text-sm underline"
			><span aria-hidden="true">‹</span>Milestones for {data.kid.name}</a
		>
		<h1 class="mt-2">Edit milestone</h1>
		<p class="mt-1.5 text-sm text-muted-foreground">{formatAge(data.kid.birthDate, data.today)}</p>
	{/snippet}
	{#if !data.custom}
		<p class="mt-[18px] text-[15px] [overflow-wrap:anywhere]">{data.text}</p>
	{/if}
	<form
		method="POST"
		action="?/save"
		class={[
			'flex flex-col gap-3.5',
			!embed &&
				'mt-[18px] xl:grid xl:grid-cols-[448px_448px] xl:grid-rows-[auto_1fr] xl:gap-x-6 xl:gap-y-[18px]'
		]}
		novalidate
		use:enhance={savedAs('Milestone saved')}
	>
		{#if top}
			<Alert class="xl:col-span-2">{top}</Alert>
		{:else if refused}
			<Alert class="xl:col-span-2">Check the fields with a message.</Alert>
		{/if}
		<MilestoneFields
			{values}
			{error}
			custom={data.custom}
			birthDate={data.kid.birthDate}
			today={data.today}
		/>
		<div class="flex gap-2">
			<Button type="submit" size={embed ? 'lg' : 'default'} class="flex-1">Save</Button>
			<FormCancel href={milestones} size="default" class="" />
		</div>
	</form>

	<!-- The delete message renders only inside the details, so a failed delete must reopen them. -->
	<details
		class="mt-[18px] overflow-hidden rounded-2xl border bg-card px-3.5 xl:w-[448px]"
		open={!!del?.message}
	>
		<summary class="flex min-h-14 cursor-pointer items-center text-[15px] font-semibold"
			>{words.summary}</summary
		>
		<form
			method="POST"
			action="?/delete"
			class="flex flex-col gap-3 pb-3.5"
			use:enhance={savedAs('Milestone removed', {
				restore: () => ({
					action: `/kids/${data.kid.id}/milestones/new?/save`,
					fields: {
						...(data.item ? { item: data.item } : { text: data.text }),
						reached_on: data.values.reached_on ?? '',
						note: data.values.note ?? ''
					}
				}),
				undone: 'Milestone restored'
			})}
		>
			{#if del?.message}
				<Alert>{del.message}</Alert>
			{/if}
			<p class="text-[13.5px] leading-[1.5] [overflow-wrap:anywhere]">
				{words.body} for everyone with access to {data.kid.name}.
			</p>
			<Button type="submit" variant="destructive" class="self-start">{words.confirm}</Button>
		</form>
	</details>
</FormFrame>
