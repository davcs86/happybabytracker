<script lang="ts">
	import { enhance } from '$app/forms';
	import { page } from '$app/state';
	import TrackerForm from '#lib/components/entry/TrackerForm.svelte';
	import { ERROR_CLASS } from '#lib/components/field.js';
	import { Button, buttonVariants } from '#lib/components/ui/button/index.js';
	import { MAX_SESSION_HOURS, TRACKER_TITLE } from '#lib/constants.js';
	import { savedAs } from '#lib/drawer.js';
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

<TrackerForm
	kid={data.kid}
	kind={data.kind}
	stage={data.running && !ended ? 'running' : 'done'}
	{title}
	age={data.ages[data.kid.id]}
	zone={data.kid.timeZone}
	{values}
	{error}
	{toggle}
	estimate={data.estimate}
	notice={data.overrun
		? `A timer can last at most ${MAX_SESSION_HOURS} hours; set the end time or delete it`
		: null}
	message={save && !save.field ? save.message : null}
	refused={!!save?.field}
>
	{#snippet footer()}
		<!-- Two steps without JavaScript: the button exists only inside the opened details.
		     A failure reopens it, so its message shows after a full-page POST too. -->
		<details class="mt-4 border-t pt-4" open={!!del?.message}>
			<summary
				class="{buttonVariants({
					variant: 'destructive'
				})} w-full cursor-pointer list-none px-5 [&::-webkit-details-marker]:hidden"
				>Delete this entry</summary
			>
			<form
				method="POST"
				action="?/delete"
				class="flex flex-col gap-2.5 pt-2.5 pb-2"
				use:enhance={savedAs('Entry deleted')}
			>
				{#if del?.message}
					<p class={ERROR_CLASS} role="alert">{del.message}</p>
				{/if}
				<p class="text-[13.5px] leading-normal [overflow-wrap:anywhere] text-muted-foreground">
					This removes the entry for everyone with access to {data.kid.name}.
				</p>
				<Button type="submit" variant="destructive" class="px-5">Yes, delete this entry</Button>
			</form>
		</details>
	{/snippet}
</TrackerForm>
