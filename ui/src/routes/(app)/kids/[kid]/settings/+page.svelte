<script lang="ts">
	import { onMount } from 'svelte';
	import { resolve } from '$app/paths';
	import { enhance } from '$app/forms';
	import KidForm from '#lib/components/KidForm.svelte';
	import { ERROR_CLASS, keepValues } from '#lib/components/field.js';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	const save = $derived(form?.form === 'save' ? form : null);
	const del = $derived(form?.form === 'delete' ? form : null);
	const mismatch = $derived(del && 'mismatch' in del && del.mismatch);

	// The tick only gates the button once the script runs; the server checks the typed name.
	let sure = $state(false);
	let hydrated = $state(false);
	onMount(() => (hydrated = true));
</script>

<svelte:head>
	<title>Edit details · happybabytracker</title>
</svelte:head>

<div class="mt-5 flex max-w-[448px] flex-col gap-[18px] xl:mx-auto xl:w-[920px] xl:max-w-none">
	<div class="flex flex-col gap-1.5">
		<h1>Edit details</h1>
		<p class="text-sm leading-snug text-muted-foreground">{data.ages[data.kid.id]}</p>
	</div>
	{#if save && 'saved' in save}
		<Alert.Root role="status">Saved</Alert.Root>
	{/if}
	<p class="leading-normal">Corrected age on Home uses the birth date and weeks at birth.</p>
	<KidForm
		values={(save && 'values' in save && save.values) || data.values}
		field={save && 'field' in save ? save.field : null}
		message={save && 'message' in save ? save.message : null}
		zones={data.zones}
		sources={data.sources}
		schedules={data.schedules}
		mode="settings"
		submitLabel="Save"
		cancelHref={resolve('/(app)/kids/[kid]/profile', { kid: data.kid.id })}
		action="?/save"
	/>

	<details
		id="delete"
		open={!!del}
		class="max-w-[448px] rounded-2xl border-[1.5px] border-destructive bg-card px-4"
	>
		<summary class="flex min-h-12 cursor-pointer items-center font-semibold text-destructive">
			Delete all data for {data.kid.name}…
		</summary>
		<div class="flex flex-col gap-3.5 pt-1 pb-4">
			<p class="text-[13.5px] leading-normal [overflow-wrap:anywhere]">
				This is permanent and immediate. All data for {data.kid.name} is deleted for everyone who has
				access. Nothing is kept.
			</p>
			<ul class="list-disc pl-5 text-[13.5px] leading-relaxed">
				<li>Every logged entry, including entries other people logged</li>
				<li>Growth measurements and vaccination records</li>
				<li>Notes, the trackers you chose, and the details on this page</li>
				<li>Access for everyone who has it</li>
			</ul>
			<form
				method="POST"
				action="?/delete"
				class="flex flex-col gap-3.5"
				novalidate
				use:enhance={keepValues}
			>
				{#if del && 'message' in del && del.message}
					<Alert.Root>{del.message}</Alert.Root>
				{/if}
				<div class="grid gap-1.5">
					<Label for="confirm_name" class="[overflow-wrap:anywhere]"
						>Type {data.kid.name} to confirm</Label
					>
					<Input
						id="confirm_name"
						name="confirm_name"
						autocomplete="off"
						value={del && 'confirmName' in del ? del.confirmName : ''}
						aria-invalid={mismatch ? 'true' : undefined}
						aria-describedby={mismatch ? 'confirm_name-error' : undefined}
					/>
					{#if mismatch}
						<p id="confirm_name-error" class={ERROR_CLASS}>The name does not match</p>
					{/if}
				</div>
				<label class="flex min-h-11 items-center gap-3 text-[15px]">
					<input type="checkbox" bind:checked={sure} class="size-5 shrink-0 accent-primary" />
					I understand this is permanent and immediate
				</label>
				<Button
					type="submit"
					variant="destructive"
					class="self-start [overflow-wrap:anywhere] whitespace-normal"
					disabled={hydrated && !sure}>Delete all data for {data.kid.name}</Button
				>
			</form>
		</div>
	</details>
</div>
