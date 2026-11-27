<script lang="ts">
	import { ERROR_CLASS, SELECT_CLASS } from '#lib/components/field.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import { WEIGHT_UNITS } from '#lib/constants.js';
	import type { EntryValues } from '#lib/entry-edit.js';

	// name is the prefix of the four texts (weight_before: _unit, _kg, _lb, _oz); the growth form's control.
	let {
		name,
		label,
		values,
		error
	}: { name: string; label: string; values: EntryValues; error: string | null } = $props();

	const PARTS = ['unit', 'kg', 'lb', 'oz'] as const;
	const text = (part: (typeof PARTS)[number]) => values[`${name}_${part}`] as string;
	const invalid = $derived(error ? ('true' as const) : undefined);
	const describedBy = $derived(error ? `entry-${name}-error` : undefined);
</script>

<!-- CSS shows the texts the chosen unit uses, so this works without JavaScript. -->
<fieldset class="group/weight flex flex-col gap-1.5">
	<legend class="mb-0.5 text-[13px] leading-4 text-muted-foreground">{label}</legend>
	<div class="flex items-end gap-2">
		<div
			class="hidden min-w-0 flex-1 flex-col gap-1.5 group-has-[option[value=kg]:checked]/weight:flex"
		>
			<Label for="entry-{name}_kg">Kilograms</Label>
			<Input
				id="entry-{name}_kg"
				name="{name}_kg"
				inputmode="decimal"
				autocomplete="off"
				value={text('kg')}
				aria-invalid={invalid}
				aria-describedby={describedBy}
			/>
		</div>
		<div class="hidden min-w-0 flex-1 gap-2 group-has-[option[value=lb]:checked]/weight:flex">
			<div class="flex min-w-0 flex-1 flex-col gap-1.5">
				<Label for="entry-{name}_lb" class="text-muted-foreground">Pounds</Label>
				<Input
					id="entry-{name}_lb"
					name="{name}_lb"
					inputmode="numeric"
					autocomplete="off"
					value={text('lb')}
					aria-invalid={invalid}
					aria-describedby={describedBy}
				/>
			</div>
			<div class="flex min-w-0 flex-1 flex-col gap-1.5">
				<Label for="entry-{name}_oz" class="text-muted-foreground">Ounces</Label>
				<Input
					id="entry-{name}_oz"
					name="{name}_oz"
					inputmode="decimal"
					autocomplete="off"
					value={text('oz')}
					aria-invalid={invalid}
					aria-describedby={describedBy}
				/>
			</div>
		</div>
		<div class="w-24 shrink-0">
			<label for="entry-{name}_unit" class="sr-only">{label} unit</label>
			<select
				id="entry-{name}_unit"
				name="{name}_unit"
				class={SELECT_CLASS}
				value={text('unit')}
				aria-invalid={invalid}
			>
				{#each WEIGHT_UNITS as unit (unit)}
					<option value={unit}>{unit}</option>
				{/each}
			</select>
		</div>
	</div>
	{#if error}
		<p id="entry-{name}-error" class={ERROR_CLASS} role="alert">{error}</p>
	{/if}
	{#if values[`${name}_unit_original`] !== undefined}
		{#each PARTS as part (part)}
			<input
				type="hidden"
				name="{name}_{part}_original"
				value={values[`${name}_${part}_original`]}
			/>
		{/each}
	{/if}
</fieldset>
