<script lang="ts">
	import Field from '#lib/components/entry/Field.svelte';
	import { SELECT_CLASS } from '#lib/components/field.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { TEMP_UNIT_LABEL, TEMP_UNITS } from '#lib/constants.js';
	import type { EntryValues } from '#lib/entry-edit.js';

	let { values, error }: { values: EntryValues; error: (name: string) => string | null } = $props();
</script>

<Field id="entry-temperature" label="Temperature" error={error('temperature')}>
	{#snippet children(control)}
		<div class="flex gap-2">
			<div class="min-w-0 flex-1">
				<Input
					{...control}
					name="temperature"
					inputmode="decimal"
					autocomplete="off"
					value={values.temperature as string}
					required
				/>
			</div>
			<div class="w-24 shrink-0">
				<label for="entry-temperature_unit" class="sr-only">Temperature unit</label>
				<select
					id="entry-temperature_unit"
					name="temperature_unit"
					class={SELECT_CLASS}
					aria-invalid={error('temperature') ? 'true' : undefined}
					aria-describedby={error('temperature') ? 'entry-temperature-error' : undefined}
					value={values.temperature_unit as string}
				>
					{#each TEMP_UNITS as unit (unit)}
						<option value={unit}>{TEMP_UNIT_LABEL[unit]}</option>
					{/each}
				</select>
			</div>
		</div>
	{/snippet}
</Field>
{#if values.temperature_original !== undefined}
	<input type="hidden" name="temperature_original" value={values.temperature_original} />
	<input type="hidden" name="temperature_unit_original" value={values.temperature_unit_original} />
{/if}
