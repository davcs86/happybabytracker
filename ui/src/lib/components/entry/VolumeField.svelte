<script lang="ts">
	import Field from '#lib/components/entry/Field.svelte';
	import { SELECT_CLASS } from '#lib/components/field.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { VOLUME_UNITS } from '#lib/constants.js';
	import type { EntryValues } from '#lib/entry-edit.js';

	let {
		values,
		error,
		hint = null
	}: {
		values: EntryValues;
		error: (name: string) => string | null;
		hint?: string | null;
	} = $props();

	const UNIT_LABEL = { ml: 'ml', floz: 'fl oz' } as const;
</script>

<Field id="entry-volume" label="Volume" error={error('volume')} {hint}>
	{#snippet children(control)}
		<div class="flex gap-2">
			<div class="min-w-0 flex-1">
				<Input
					{...control}
					name="volume"
					inputmode="decimal"
					autocomplete="off"
					value={values.volume as string}
					required
				/>
			</div>
			<div class="w-28 shrink-0">
				<label for="entry-volume_unit" class="sr-only">Volume unit</label>
				<select
					id="entry-volume_unit"
					name="volume_unit"
					class={SELECT_CLASS}
					aria-invalid={error('volume') ? 'true' : undefined}
					value={values.volume_unit as string}
				>
					{#each VOLUME_UNITS as unit (unit)}
						<option value={unit}>{UNIT_LABEL[unit]}</option>
					{/each}
				</select>
			</div>
		</div>
	{/snippet}
</Field>
{#if values.volume_original !== undefined}
	<input type="hidden" name="volume_original" value={values.volume_original} />
	<input type="hidden" name="volume_unit_original" value={values.volume_unit_original} />
{/if}
