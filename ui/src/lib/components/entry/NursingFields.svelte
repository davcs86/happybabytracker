<script lang="ts">
	import { END_RUNNING_HINT } from '#lib/components/field.js';
	import EntryTimeField from '#lib/components/entry/EntryTimeField.svelte';
	import Field from '#lib/components/entry/Field.svelte';
	import RadioField from '#lib/components/entry/RadioField.svelte';
	import WeightField from '#lib/components/entry/WeightField.svelte';
	import { Input } from '#lib/components/ui/input/index.js';
	import { NURSING_SIDES, SCALE_CAVEAT } from '#lib/constants.js';
	import type { EntryStage, EntryValues } from '#lib/entry-edit.js';

	let {
		part,
		stage,
		values,
		error,
		estimate = null
	}: {
		part: 'primary' | 'optional';
		stage: EntryStage;
		values: EntryValues;
		error: (name: string) => string | null;
		// The edit page's paragraph under the weigh-ins (milkStatus).
		estimate?: string | null;
	} = $props();
</script>

{#if part === 'primary'}
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
		<fieldset class="flex flex-col gap-3.5 rounded-xl border p-3.5">
			<legend class="px-1 text-[13px] leading-4 text-muted-foreground">
				Or record a finished session
			</legend>
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
{:else if values.advanced === '1'}
	<WeightField name="weight_before" label="Weight before" {values} error={error('weight_before')} />
	<WeightField name="weight_after" label="Weight after" {values} error={error('weight_after')} />
	<p class="text-[12.5px] leading-normal text-muted-foreground">
		In your weight unit. These weights stay with this session; they are not added to growth. {SCALE_CAVEAT}
	</p>
	{#if estimate}<p class="text-sm">{estimate}</p>{/if}
{/if}

{#snippet minutes()}
	<div class="grid grid-cols-2 gap-3.5">
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
