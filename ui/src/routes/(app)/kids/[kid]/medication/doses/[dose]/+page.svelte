<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import Field from '#lib/components/entry/Field.svelte';
	import { ERROR_CLASS, keepValues } from '#lib/components/field.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	const save = $derived(form?.form === 'save' ? form : null);
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

<p class="mb-2 text-sm">
	<a href={medication} class="underline">Medication for {data.kid.name}</a>
</p>
<h1 class="text-xl font-semibold">Edit dose</h1>
<p class="mb-4 text-sm [overflow-wrap:anywhere] text-muted-foreground">
	Dose of {data.course.name}
</p>
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
	<p class="text-sm text-muted-foreground">Times are in {data.kid.timeZone}</p>
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
	<Field id="dose-note" label="Note" error={error('note')}>
		{#snippet children(control)}
			<Input {...control} name="note" autocomplete="off" value={values.note} />
		{/snippet}
	</Field>
	<input type="hidden" name="given_at_original" value={data.values.given_at_original} />
	<input type="hidden" name="amount_original" value={data.values.amount_original} />
	<input type="hidden" name="note_original" value={data.values.note_original} />
	<div class="flex items-center gap-4">
		<Button type="submit">Save</Button>
		<a href={medication} class="text-sm underline">Cancel</a>
	</div>
</form>
