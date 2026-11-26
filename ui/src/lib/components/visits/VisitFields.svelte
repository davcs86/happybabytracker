<script lang="ts">
	import EntryTimeField from '#lib/components/entry/EntryTimeField.svelte';
	import Field from '#lib/components/entry/Field.svelte';
	import RadioField from '#lib/components/entry/RadioField.svelte';
	import { Input } from '#lib/components/ui/input/index.js';
	import { VISIT_KINDS, VISIT_KIND_LABEL } from '#lib/constants.js';
	import type { VisitFormValues } from '#lib/server/visit-form.js';
	import type { Snippet } from 'svelte';

	let {
		values,
		error,
		search
	}: {
		values: VisitFormValues;
		error: (name: string) => string | null;
		// The "Find a US provider" panel, right above the provider (plan 10 B9).
		search?: Snippet;
	} = $props();

	const npi = $derived(
		values.provider_npi !== '' && values.provider.trim() === values.provider_npi_for.trim()
			? values.provider_npi
			: null
	);
</script>

<!-- Two groups for the wide layout: when and what, then who, where and why. -->
<div class="flex flex-col gap-3.5">
	<EntryTimeField name="starts_at" label="Date and time" {values} {error} required />
	<Field id="visit-duration_min" label="Length (minutes)" error={error('duration_min')}>
		{#snippet children(control)}
			<Input
				{...control}
				name="duration_min"
				inputmode="numeric"
				autocomplete="off"
				value={values.duration_min}
			/>
		{/snippet}
	</Field>
	<RadioField
		name="kind"
		legend="Kind"
		options={VISIT_KINDS}
		labels={VISIT_KIND_LABEL}
		{values}
		{error}
		required
	/>
</div>
<div class="flex flex-col gap-3.5 xl:row-span-2">
	{@render search?.()}
	<Field
		id="visit-provider"
		label="Doctor or clinic (optional)"
		error={error('provider') ?? error('provider_npi')}
		hint={npi && `From the US provider registry, NPI ${npi}`}
	>
		{#snippet children(control)}
			<Input {...control} name="provider" autocomplete="off" value={values.provider} />
		{/snippet}
	</Field>
	<input type="hidden" name="provider_npi" value={values.provider_npi} />
	<input type="hidden" name="provider_npi_for" value={values.provider_npi_for} />
	<Field id="visit-location" label="Place (optional)" error={error('location')}>
		{#snippet children(control)}
			<Input {...control} name="location" autocomplete="off" value={values.location} />
		{/snippet}
	</Field>
	<Field id="visit-note" label="Reason (optional)" error={error('note')}>
		{#snippet children(control)}
			<Input {...control} name="note" autocomplete="off" value={values.note} />
		{/snippet}
	</Field>
</div>
