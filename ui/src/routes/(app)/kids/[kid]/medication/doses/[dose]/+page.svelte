<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import Field from '#lib/components/entry/Field.svelte';
	import { keepValues } from '#lib/components/field.js';
	import { Alert } from '#lib/components/ui/alert/index.js';
	import { Button, buttonVariants } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
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
	const medication = $derived(resolve('/(app)/kids/[kid]/medication', { kid: data.kid.id }));
</script>

<svelte:head>
	<title>Edit dose · {data.kid.name} · happybabytracker</title>
</svelte:head>

<div class="flex max-w-[448px] flex-col gap-[18px] xl:mx-auto xl:w-[920px] xl:max-w-none">
	<div class="mt-5 flex flex-col gap-1.5">
		<h1>Edit dose</h1>
		<p class="text-sm leading-snug text-muted-foreground">{data.ages[data.kid.id]}</p>
		<p class="text-[13.5px] [overflow-wrap:anywhere] text-muted-foreground">
			Dose of {data.course.name}
		</p>
		<p class="text-[13.5px] text-muted-foreground">Times are in {data.kid.timeZone}</p>
	</div>
	{#if save && !save.field && save.message}
		<Alert class="[overflow-wrap:anywhere]">{save.message}</Alert>
	{/if}
	<form
		method="POST"
		action="?/save"
		class="flex flex-col gap-[18px] xl:grid xl:grid-cols-[448px_448px] xl:grid-rows-[auto_1fr] xl:gap-x-6"
		novalidate
		use:enhance={keepValues}
	>
		<div class="flex flex-col gap-3.5">
			<Field id="dose-given_at" label="Given at" error={error('given_at')}>
				{#snippet children(control)}
					<Input {...control} name="given_at" type="datetime-local" value={values.given_at} />
				{/snippet}
			</Field>
			<Field id="dose-amount" label="Amount" error={error('amount')}>
				{#snippet children(control)}
					<Input {...control} name="amount" autocomplete="off" value={values.amount} />
				{/snippet}
			</Field>
		</div>
		<div class="flex flex-col gap-3.5 xl:row-span-2">
			<Field id="dose-note" label="Note" error={error('note')}>
				{#snippet children(control)}
					<Input {...control} name="note" autocomplete="off" value={values.note} />
				{/snippet}
			</Field>
		</div>
		<input type="hidden" name="given_at_original" value={data.values.given_at_original} />
		<input type="hidden" name="amount_original" value={data.values.amount_original} />
		<input type="hidden" name="note_original" value={data.values.note_original} />
		<div class="flex gap-2">
			<Button type="submit" class="flex-1">Save</Button>
			<Button href={medication} variant="outline">Cancel</Button>
		</div>
	</form>

	<!-- Two steps without JavaScript: the button exists only inside the opened details.
     A failure reopens it, so its message shows after a full-page POST too. -->
	<details class="max-w-[448px] border-t pt-4" open={!!del?.message}>
		<summary
			class="{buttonVariants({
				variant: 'destructive'
			})} w-full cursor-pointer list-none [&::-webkit-details-marker]:hidden"
			>Delete this dose</summary
		>
		<form method="POST" action="?/delete" class="flex flex-col gap-2.5 pt-3" use:enhance>
			{#if del?.message}
				<Alert class="[overflow-wrap:anywhere]">{del.message}</Alert>
			{/if}
			<p class="text-[13.5px] [overflow-wrap:anywhere] text-muted-foreground">
				This removes the dose for everyone with access to {data.kid.name}.
			</p>
			<Button type="submit" variant="destructive">Yes, delete this dose</Button>
		</form>
	</details>
</div>
