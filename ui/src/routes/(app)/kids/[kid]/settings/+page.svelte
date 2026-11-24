<script lang="ts">
	import { resolve } from '$app/paths';
	import { enhance } from '$app/forms';
	import KidForm from '#lib/components/KidForm.svelte';
	import { ERROR_CLASS } from '#lib/components/field.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	const save = $derived(form?.form === 'save' ? form : null);
	const del = $derived(form?.form === 'delete' ? form : null);
	const mismatch = $derived(del && 'mismatch' in del && del.mismatch);
</script>

<svelte:head>
	<title>Settings for {data.kid.name} · happybabytracker</title>
</svelte:head>

<p class="mb-2 text-sm">
	<a href={resolve('/(app)/kids/[kid]', { kid: data.kid.id })} class="underline">{data.kid.name}</a>
</p>
<h1 class="mb-4 text-xl font-semibold">Settings for {data.kid.name}</h1>

{#if save && 'saved' in save}
	<p role="status" class="mb-4 text-sm">Saved</p>
{/if}
<KidForm
	values={(save && 'values' in save && save.values) || data.values}
	field={save && 'field' in save ? save.field : null}
	message={save && 'message' in save ? save.message : null}
	zones={data.zones}
	sources={data.sources}
	schedules={data.schedules}
	mode="settings"
	submitLabel="Save"
	action="?/save"
/>

{#if data.kid.role === 'guardian'}
	<section aria-labelledby="delete-heading" class="mt-8 flex max-w-md flex-col gap-3 border-t pt-6">
		<h2 id="delete-heading" class="font-semibold">Delete {data.kid.name}</h2>
		<p class="text-sm text-muted-foreground">
			This removes the child and every record for everyone with access. It cannot be undone.
		</p>
		<form method="POST" action="?/delete" class="flex flex-col gap-3" novalidate use:enhance>
			{#if del && 'message' in del && del.message}
				<p class={ERROR_CLASS} role="alert">{del.message}</p>
			{/if}
			<div class="flex flex-col gap-2">
				<Label for="confirm_name">Type {data.kid.name} to confirm</Label>
				<Input
					id="confirm_name"
					name="confirm_name"
					autocomplete="off"
					value={del && 'confirmName' in del ? del.confirmName : ''}
					aria-invalid={mismatch ? 'true' : undefined}
					aria-describedby={mismatch ? 'confirm_name-error' : undefined}
				/>
				{#if mismatch}
					<p id="confirm_name-error" class={ERROR_CLASS} role="alert">The name does not match</p>
				{/if}
			</div>
			<Button type="submit" variant="outline" class="self-start border-destructive text-destructive"
				>Delete {data.kid.name}</Button
			>
		</form>
	</section>
{/if}
