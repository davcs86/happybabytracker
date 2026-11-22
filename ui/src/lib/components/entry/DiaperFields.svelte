<script lang="ts">
	import EntryTimeField from '#lib/components/entry/EntryTimeField.svelte';
	import RadioField from '#lib/components/entry/RadioField.svelte';
	import { DIAPER_KINDS } from '#lib/constants.js';
	import type { EntryValues } from '#lib/entry-edit.js';

	let { values, error }: { values: EntryValues; error: (name: string) => string | null } = $props();
</script>

<EntryTimeField name="at" label="Time" {values} {error} required />
<RadioField name="kind" legend="Kind" options={DIAPER_KINDS} {values} {error} required />
