<script lang="ts">
	import type { Snippet } from 'svelte';
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import Field from '#lib/components/entry/Field.svelte';
	import { ERROR_CLASS, keepValues } from '#lib/components/field.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import type { EntryValues } from '#lib/entry-edit.js';

	let {
		kid,
		zone,
		values,
		error,
		message = null,
		running = false,
		toggle = null,
		children
	}: {
		kid: { id: string; name: string };
		zone: string;
		values: EntryValues;
		error: (name: string) => string | null;
		// A failure that belongs to no field; running adds the way to the timer that blocks this one.
		message?: string | null;
		running?: boolean;
		// The plain link that reloads the form advanced or simple; values.advanced says which it is.
		toggle?: { href: string; label: string } | null;
		children: Snippet;
	} = $props();

	const home = $derived(resolve('/(app)/kids/[kid]', { kid: kid.id }));
</script>

<form
	method="POST"
	action="?/save"
	class="flex max-w-md flex-col gap-4"
	novalidate
	use:enhance={keepValues}
>
	{#if toggle}
		<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- a query on this same page -->
		<p class="text-sm"><a href={toggle.href} class="underline">{toggle.label}</a></p>
	{/if}
	{#if values.advanced === '1'}
		<input type="hidden" name="advanced" value="1" />
	{/if}
	{#if message}
		<p class={ERROR_CLASS} role="alert">
			{message}{#if running}.
				<a href={home} class="underline">See {kid.name}'s timer</a>{/if}
		</p>
	{/if}
	<p class="text-sm text-muted-foreground">Times are in {zone}</p>
	{@render children()}
	<Field id="entry-note" label="Note" error={error('note')}>
		{#snippet children(control)}
			<Input {...control} name="note" autocomplete="off" value={values.note as string} />
		{/snippet}
	</Field>
	<div class="flex items-center gap-4">
		<Button type="submit">Save</Button>
		<a href={home} class="text-sm underline">Cancel</a>
	</div>
</form>
