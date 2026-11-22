<script lang="ts">
	import Field from '#lib/components/entry/Field.svelte';
	import { Input } from '#lib/components/ui/input/index.js';
	import type { EntryValues } from '#lib/entry-edit.js';

	// A time is read and written on the kid's clock; the server converts with the kid's zone.
	let {
		name,
		label,
		values,
		error,
		hint = null,
		required = false
	}: {
		name: string;
		label: string;
		values: EntryValues;
		error: (name: string) => string | null;
		hint?: string | null;
		required?: boolean;
	} = $props();

	const original = $derived(values[`${name}_original`]);
</script>

<Field id="entry-{name}" {label} error={error(name)} {hint}>
	{#snippet children(control)}
		<Input {...control} {name} type="datetime-local" value={values[name] as string} {required} />
	{/snippet}
</Field>
{#if original !== undefined}
	<input type="hidden" name="{name}_original" value={original} />
{/if}
