<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import TrackerFields from '#lib/components/entry/TrackerFields.svelte';
	import TrackerForm from '#lib/components/entry/TrackerForm.svelte';
	import { ERROR_CLASS } from '#lib/components/field.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { MAX_SESSION_HOURS, TRACKER_TITLE } from '#lib/constants.js';
	import { advancedToggle } from '#lib/entry-edit.js';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	const save = $derived(form?.form === 'save' ? form : null);
	const del = $derived(form?.form === 'delete' ? form : null);
	// Once a stale result brings the finished values they stay: the session cannot run again, and
	// enhance keeps the loaded data, so every later failure must still show them.
	let seen = $state<{ entry: string; values: typeof data.values } | null>(null);
	$effect.pre(() => {
		if (save && 'stale' in save) seen = { entry: page.params.entry ?? '', values: save.values };
	});
	const ended = $derived(seen && seen.entry === page.params.entry ? seen.values : null);
	// The originals are the loaded ones (or the finished ones), never the posted.
	const values = $derived({
		...(ended ?? data.values),
		...Object.fromEntries(
			Object.entries(save?.values ?? {}).filter(([name]) => !name.endsWith('_original'))
		)
	});
	const error = (name: string) => (save?.field === name ? save.message : null);
	const title = $derived(TRACKER_TITLE[data.kind].edit);
	const toggle = $derived(advancedToggle(data.kind, values.advanced === '1'));
</script>

<svelte:head>
	<title>{title} · {data.kid.name} · happybabytracker</title>
</svelte:head>

<p class="mb-2 text-sm">
	<a href={resolve('/(app)/kids/[kid]', { kid: data.kid.id })} class="underline">{data.kid.name}</a>
</p>
<h1 class="mb-4 text-xl font-semibold">{title}</h1>
{#if data.overrun}
	<p class="mb-4 max-w-md text-sm">
		A timer can last at most {MAX_SESSION_HOURS} hours; set the end time or delete it
	</p>
{/if}
<TrackerForm
	kid={data.kid}
	zone={data.kid.timeZone}
	{values}
	{error}
	{toggle}
	message={save && !save.field ? save.message : null}
>
	<TrackerFields
		kind={data.kind}
		stage={data.running && !ended ? 'running' : 'done'}
		{values}
		{error}
		estimate={data.estimate}
	/>
</TrackerForm>

<!-- Two steps without JavaScript: the button exists only inside the opened details.
     A failure reopens it, so its message shows after a full-page POST too. -->
<details class="mt-6 max-w-md rounded-lg border p-3" open={!!del?.message}>
	<summary class="cursor-pointer text-sm font-medium">Delete this entry</summary>
	<form method="POST" action="?/delete" class="mt-3 flex flex-col gap-3" use:enhance>
		{#if del?.message}
			<p class={ERROR_CLASS} role="alert">{del.message}</p>
		{/if}
		<p class="text-sm">This removes the entry for everyone with access to {data.kid.name}.</p>
		<Button type="submit" variant="outline" class="self-start border-destructive text-destructive"
			>Yes, delete this entry</Button
		>
	</form>
</details>
