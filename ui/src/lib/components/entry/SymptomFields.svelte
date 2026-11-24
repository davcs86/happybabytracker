<script lang="ts">
	import EntryTimeField from '#lib/components/entry/EntryTimeField.svelte';
	import Field from '#lib/components/entry/Field.svelte';
	import RadioField from '#lib/components/entry/RadioField.svelte';
	import TemperatureField from '#lib/components/entry/TemperatureField.svelte';
	import { SELECT_CLASS } from '#lib/components/field.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import {
		SEVERITIES,
		SYMPTOM_LABEL,
		SYMPTOMS,
		TEMP_METHOD_LABEL,
		TEMP_METHODS
	} from '#lib/constants.js';
	import type { EntryValues } from '#lib/entry-edit.js';

	let { values, error }: { values: EntryValues; error: (name: string) => string | null } = $props();

	// The temperature's own originals are emitted by TemperatureField.
	const KEPT = ['symptom', 'severity', 'other_text', 'method'] as const;
</script>

<EntryTimeField name="at" label="Time" {values} {error} required />
<!-- CSS shows the fields the chosen symptom uses, so this works without JavaScript. -->
<div class="group/symptom flex flex-col gap-4">
	<Field id="entry-symptom" label="Symptom" error={error('symptom')}>
		{#snippet children(control)}
			<select
				id={control.id}
				name="symptom"
				class={SELECT_CLASS}
				aria-invalid={control['aria-invalid']}
				aria-describedby={control['aria-describedby']}
				value={values.symptom as string}
				required
			>
				<option value="">Choose a symptom</option>
				{#each SYMPTOMS as symptom (symptom)}
					<option value={symptom}>{SYMPTOM_LABEL[symptom]}</option>
				{/each}
			</select>
		{/snippet}
	</Field>
	<RadioField
		name="severity"
		legend="Severity"
		options={SEVERITIES}
		blank="Not set"
		{values}
		{error}
	/>
	<div class="hidden flex-col group-has-[[name=symptom]_option[value=other]:checked]/symptom:flex">
		<Field id="entry-other_text" label="Other symptom" error={error('other_text')}>
			{#snippet children(control)}
				<Input
					{...control}
					name="other_text"
					autocomplete="off"
					value={values.other_text as string}
					required
				/>
			{/snippet}
		</Field>
	</div>
	<div
		class="hidden flex-col gap-4 group-has-[[name=symptom]_option[value=fever]:checked]/symptom:flex"
	>
		<TemperatureField {values} {error} />
		<Field id="entry-method" label="Method" error={error('method')}>
			{#snippet children(control)}
				<select
					id={control.id}
					name="method"
					class={SELECT_CLASS}
					aria-invalid={control['aria-invalid']}
					aria-describedby={control['aria-describedby']}
					value={values.method as string}
					required
				>
					<option value="">Choose a method</option>
					{#each TEMP_METHODS as method (method)}
						<option value={method}>{TEMP_METHOD_LABEL[method]}</option>
					{/each}
				</select>
			{/snippet}
		</Field>
	</div>
	{#if values.symptom_original !== undefined}
		{#each KEPT as name (name)}
			<input type="hidden" name="{name}_original" value={values[`${name}_original`]} />
		{/each}
	{/if}
</div>
