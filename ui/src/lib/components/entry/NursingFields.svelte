<script lang="ts">
	import { END_RUNNING_HINT } from '#lib/components/field.js';
	import EntryTimeField from '#lib/components/entry/EntryTimeField.svelte';
	import Field from '#lib/components/entry/Field.svelte';
	import RadioField from '#lib/components/entry/RadioField.svelte';
	import { Input } from '#lib/components/ui/input/index.js';
	import { NURSING_SIDES } from '#lib/constants.js';
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
{#if stage === 'new'}
	<RadioField
		name="side"
		legend="Side"
		options={NURSING_SIDES}
		{values}
		{error}
		hint="Choose the side you start on to run a timer"
	/>
{/if}
<!-- A running session shows no minutes: its seconds are credited by the timer, never typed. -->
{#if stage === 'new'}
	<fieldset class="flex flex-col gap-4 rounded-lg border p-3">
		<legend class="px-1 text-sm font-medium">Or record a finished session</legend>
		<EntryTimeField name="ended_at" label="Ended at" {values} {error} />
		{@render minutes()}
	</fieldset>
{:else}
	<EntryTimeField
		name="ended_at"
		label="Ended at"
		{values}
		{error}
		hint={stage === 'running' ? END_RUNNING_HINT : null}
	/>
	{#if stage === 'done'}
		{@render minutes()}
		<input type="hidden" name="left_seconds_original" value={values.left_seconds_original} />
		<input type="hidden" name="right_seconds_original" value={values.right_seconds_original} />
	{/if}
{/if}

{#snippet minutes()}
	<div class="grid grid-cols-2 gap-4">
		{#each NURSING_SIDES as side (side)}
			<Field
				id="entry-{side}_minutes"
				label="{side === 'left' ? 'Left' : 'Right'} minutes"
				error={error(`${side}_seconds`)}
			>
				{#snippet children(control)}
					<Input
						{...control}
						name="{side}_minutes"
						inputmode="numeric"
						autocomplete="off"
						value={values[`${side}_minutes`] as string}
					/>
				{/snippet}
			</Field>
		{/each}
	</div>
{/snippet}
