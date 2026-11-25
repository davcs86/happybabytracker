<script lang="ts">
	import EntryTimeField from '#lib/components/entry/EntryTimeField.svelte';
	import Field from '#lib/components/entry/Field.svelte';
	import { Input } from '#lib/components/ui/input/index.js';
	import type { EntryValues } from '#lib/entry-edit.js';

	let {
		part,
		values,
		error
	}: {
		part: 'primary' | 'optional';
		values: EntryValues;
		error: (name: string) => string | null;
	} = $props();
</script>

{#if part === 'primary'}
	<EntryTimeField name="at" label="Time" {values} {error} required />
	<Field id="entry-what" label="What" error={error('what')}>
		{#snippet children(control)}
			<Input {...control} name="what" autocomplete="off" value={values.what as string} required />
		{/snippet}
	</Field>
{:else}
	<Field id="entry-reaction" label="Reaction" error={error('reaction')}>
		{#snippet children(control)}
			<Input {...control} name="reaction" autocomplete="off" value={values.reaction as string} />
		{/snippet}
	</Field>
{/if}
