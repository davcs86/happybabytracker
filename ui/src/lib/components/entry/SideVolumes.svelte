<script lang="ts">
	import Field from '#lib/components/entry/Field.svelte';
	import { ERROR_CLASS, SELECT_CLASS } from '#lib/components/field.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { NURSING_SIDES, VOLUME_UNITS } from '#lib/constants.js';
	import type { EntryValues } from '#lib/entry-edit.js';
	import { formatVolume } from '#lib/format.js';
	import type { VolumeUnit } from '#lib/types.js';

	let { values, error }: { values: EntryValues; error: (name: string) => string | null } = $props();

	const UNIT_LABEL = { ml: 'ml', floz: 'fl oz' } as const;
	const SIDE_LABEL = { left: 'Left', right: 'Right' } as const;
	const total = $derived(
		(values.sides_total as string | undefined) ||
			formatVolume({ ml: 0, floz_hundredths: 0 }, values.volume_unit as VolumeUnit)
	);
	// A refused total or side is no single input's fault: both inputs are marked.
	const invalid = $derived(error('sides') || error('volume') ? ('true' as const) : undefined);
	const describedBy = $derived(
		[error('sides') ? 'entry-sides-error' : null, error('volume') ? 'entry-total-error' : null]
			.filter(Boolean)
			.join(' ') || undefined
	);
</script>

<Field id="entry-volume_unit" label="Volume unit">
	{#snippet children(control)}
		<select
			{...control}
			name="volume_unit"
			class={SELECT_CLASS}
			value={values.volume_unit as string}
		>
			{#each VOLUME_UNITS as unit (unit)}
				<option value={unit}>{UNIT_LABEL[unit]}</option>
			{/each}
		</select>
	{/snippet}
</Field>
<fieldset class="flex flex-col gap-2">
	<legend class="mb-1 text-sm font-medium">Volume per side</legend>
	<p class="text-sm text-muted-foreground">Enter at least one side; leave the other blank</p>
	{#each NURSING_SIDES as side (side)}
		<Field id="entry-sides_{side}" label="{SIDE_LABEL[side]} volume">
			{#snippet children(control)}
				<Input
					{...control}
					name="sides_{side}"
					inputmode="decimal"
					autocomplete="off"
					value={values[`sides_${side}`] as string}
					aria-invalid={invalid}
					aria-describedby={describedBy}
				/>
			{/snippet}
		</Field>
	{/each}
	{#if error('sides')}
		<p id="entry-sides-error" class={ERROR_CLASS} role="alert">{error('sides')}</p>
	{/if}
</fieldset>
<p class="text-sm font-medium">Total: {total}</p>
{#if error('volume')}
	<p id="entry-total-error" class={ERROR_CLASS} role="alert">{error('volume')}</p>
{/if}
<!-- editValues always emits sides_original for a session, so its absence means a new entry. -->
{#if values.sides_original !== undefined}
	<input type="hidden" name="volume_unit_original" value={values.volume_unit_original} />
	<input type="hidden" name="sides_original" value={values.sides_original} />
	{#each NURSING_SIDES as side (side)}
		<input type="hidden" name="sides_{side}_original" value={values[`sides_${side}_original`]} />
	{/each}
{/if}
