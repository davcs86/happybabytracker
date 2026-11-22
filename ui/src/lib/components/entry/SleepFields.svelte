<script lang="ts">
	import { END_RUNNING_HINT } from '#lib/components/field.js';
	import EntryTimeField from '#lib/components/entry/EntryTimeField.svelte';
	import type { EntryStage, EntryValues } from '#lib/entry-edit.js';

	let {
		stage,
		values,
		error
	}: {
		stage: EntryStage;
		values: EntryValues;
		error: (name: string) => string | null;
	} = $props();
</script>

<EntryTimeField name="started_at" label="Started at" {values} {error} required />
<EntryTimeField
	name="ended_at"
	label="Ended at"
	{values}
	{error}
	hint={stage === 'new'
		? 'Leave blank to start a timer'
		: stage === 'running'
			? END_RUNNING_HINT
			: null}
/>
