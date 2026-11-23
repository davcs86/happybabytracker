<script lang="ts">
	import EntryTimeField from '#lib/components/entry/EntryTimeField.svelte';
	import Field from '#lib/components/entry/Field.svelte';
	import RadioField from '#lib/components/entry/RadioField.svelte';
	import SideVolumes from '#lib/components/entry/SideVolumes.svelte';
	import VolumeField from '#lib/components/entry/VolumeField.svelte';
	import { Input } from '#lib/components/ui/input/index.js';
	import { PUMPING_SIDES } from '#lib/constants.js';
	import type { EntryValues } from '#lib/entry-edit.js';

	let { values, error }: { values: EntryValues; error: (name: string) => string | null } = $props();

	const advanced = $derived(values.advanced === '1');
	// The stored sides, which a total saved from the simple form replaces.
	const hadSides = $derived(!!values.sides_original);
</script>

<EntryTimeField name="started_at" label="Started at" {values} {error} required />
<Field id="entry-duration_minutes" label="Minutes" error={error('duration_seconds')}>
	{#snippet children(control)}
		<Input
			{...control}
			name="duration_minutes"
			inputmode="numeric"
			autocomplete="off"
			value={values.duration_minutes as string}
			required
		/>
	{/snippet}
</Field>
{#if values.duration_seconds_original !== undefined}
	<input type="hidden" name="duration_seconds_original" value={values.duration_seconds_original} />
{/if}
{#if advanced}
	<SideVolumes {values} {error} />
{:else}
	<VolumeField
		{values}
		{error}
		hint={hadSides ? 'Saving one total replaces the volumes per side' : null}
	/>
	{#if hadSides}
		<input type="hidden" name="sides_original" value={values.sides_original} />
	{/if}
	<RadioField name="side" legend="Side" options={PUMPING_SIDES} {values} {error} required />
{/if}
