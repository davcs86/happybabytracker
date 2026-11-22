<script lang="ts">
	import Field from '#lib/components/entry/Field.svelte';
	import { ERROR_CLASS, SELECT_CLASS } from '#lib/components/field.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import { LENGTH_UNITS, WEIGHT_UNITS } from '#lib/constants.js';

	let {
		values,
		error,
		birthDate,
		today,
		edit = false
	}: {
		values: Record<string, string>;
		error: (name: string) => string | null;
		birthDate: string;
		today: string;
		edit?: boolean;
	} = $props();

	const ORIGINALS = [
		'measured_on',
		'weight_unit',
		'weight_kg',
		'weight_lb',
		'weight_oz',
		'length_unit',
		'length',
		'head_unit',
		'head'
	] as const;
	const FLIP_HINT = 'Changing the unit reads the number in the new unit';

	// "Enter at least one" marks every measure's texts, as no single one is wrong.
	const MEASUREMENTS_ERROR = 'growth-measurements-error';
	const invalid = (name: string) =>
		error(name) || error('measurements') ? ('true' as const) : undefined;
	const describedBy = (own: string | undefined) =>
		[own, error('measurements') ? MEASUREMENTS_ERROR : null].filter(Boolean).join(' ') || undefined;
	const weightDescribedBy = $derived(
		describedBy(error('weight') ? 'growth-weight-error' : undefined)
	);
</script>

<Field id="growth-measured_on" label="Date" error={error('measured_on')}>
	{#snippet children(control)}
		<Input
			{...control}
			name="measured_on"
			type="date"
			value={values.measured_on}
			min={birthDate}
			max={today}
			required
		/>
	{/snippet}
</Field>

<fieldset class="flex flex-col gap-4">
	<legend class="mb-2 text-sm font-medium">Measurements</legend>
	{#if error('measurements')}
		<p id={MEASUREMENTS_ERROR} class={ERROR_CLASS} role="alert">
			{error('measurements')}
		</p>
	{/if}

	<!-- CSS shows the texts the chosen unit uses, so this works without JavaScript. -->
	<fieldset class="group flex flex-col gap-2">
		<legend class="mb-2 text-sm font-medium">Weight</legend>
		<div class="flex items-end gap-2">
			<div
				class="hidden min-w-0 flex-1 flex-col gap-1 group-has-[#growth-weight-unit-kg:checked]:flex"
			>
				<Label for="growth-weight_kg" class="text-muted-foreground">Kilograms</Label>
				<Input
					id="growth-weight_kg"
					name="weight_kg"
					inputmode="decimal"
					autocomplete="off"
					value={values.weight_kg}
					aria-invalid={invalid('weight')}
					aria-describedby={weightDescribedBy}
				/>
			</div>
			<div class="hidden min-w-0 flex-1 gap-2 group-has-[#growth-weight-unit-lb:checked]:flex">
				<div class="flex min-w-0 flex-1 flex-col gap-1">
					<Label for="growth-weight_lb" class="text-muted-foreground">Pounds</Label>
					<Input
						id="growth-weight_lb"
						name="weight_lb"
						inputmode="numeric"
						autocomplete="off"
						value={values.weight_lb}
						aria-invalid={invalid('weight')}
						aria-describedby={weightDescribedBy}
					/>
				</div>
				<div class="flex min-w-0 flex-1 flex-col gap-1">
					<Label for="growth-weight_oz" class="text-muted-foreground">Ounces</Label>
					<Input
						id="growth-weight_oz"
						name="weight_oz"
						inputmode="decimal"
						autocomplete="off"
						value={values.weight_oz}
						aria-invalid={invalid('weight')}
						aria-describedby={weightDescribedBy}
					/>
				</div>
			</div>
			<div class="w-24 shrink-0">
				<label for="growth-weight_unit" class="sr-only">Weight unit</label>
				<select
					id="growth-weight_unit"
					name="weight_unit"
					class={SELECT_CLASS}
					value={values.weight_unit}
					aria-invalid={invalid('weight')}
				>
					{#each WEIGHT_UNITS as unit (unit)}
						<option id="growth-weight-unit-{unit}" value={unit}>{unit}</option>
					{/each}
				</select>
			</div>
		</div>
		{#if error('weight')}
			<p id="growth-weight-error" class={ERROR_CLASS} role="alert">{error('weight')}</p>
		{/if}
	</fieldset>

	{#each [['length', 'Length or height', 'Length unit'], ['head', 'Head circumference', 'Head circumference unit']] as [name, label, unitLabel] (name)}
		<Field id="growth-{name}" {label} error={error(name)} hint={edit ? FLIP_HINT : null}>
			{#snippet children(control)}
				<div class="flex gap-2">
					<div class="min-w-0 flex-1">
						<Input
							{...control}
							{name}
							inputmode="decimal"
							autocomplete="off"
							value={values[name]}
							aria-invalid={invalid(name)}
							aria-describedby={describedBy(control['aria-describedby'])}
						/>
					</div>
					<div class="w-24 shrink-0">
						<label for="growth-{name}_unit" class="sr-only">{unitLabel}</label>
						<select
							id="growth-{name}_unit"
							name="{name}_unit"
							class={SELECT_CLASS}
							value={values[`${name}_unit`]}
							aria-invalid={invalid(name)}
						>
							{#each LENGTH_UNITS as unit (unit)}
								<option value={unit}>{unit}</option>
							{/each}
						</select>
					</div>
				</div>
			{/snippet}
		</Field>
	{/each}
</fieldset>

<Field id="growth-note" label="Note" error={error('note')}>
	{#snippet children(control)}
		<Input {...control} name="note" autocomplete="off" value={values.note} />
	{/snippet}
</Field>

{#if values.measured_on_original !== undefined}
	{#each ORIGINALS as name (name)}
		<input type="hidden" name="{name}_original" value={values[`${name}_original`]} />
	{/each}
{/if}
