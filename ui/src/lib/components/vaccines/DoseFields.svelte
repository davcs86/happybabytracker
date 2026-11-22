<script lang="ts">
	import Field from '#lib/components/entry/Field.svelte';
	import { ERROR_CLASS, SELECT_CLASS } from '#lib/components/field.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import type { VaccineCatalogue } from '#lib/types.js';

	let {
		values,
		error,
		catalogue,
		birthDate,
		today
	}: {
		values: { antigens: string[] } & Record<string, string | string[] | undefined>;
		error: (name: string) => string | null;
		catalogue: VaccineCatalogue;
		birthDate: string;
		today: string;
	} = $props();

	const ORIGINALS = ['given_on', 'antigens', 'lot', 'product', 'note'] as const;
	const ANTIGENS_ERROR = 'vaccine-antigens-error';
	const text = (name: string) => {
		const v = values[name];
		return typeof v === 'string' ? v : '';
	};

	// Progressive: without JavaScript the shortcut does nothing and the antigens are ticked by hand.
	function tick(e: Event & { currentTarget: HTMLSelectElement }) {
		const ids = (e.currentTarget.selectedOptions[0]?.dataset.antigens ?? '').split(',');
		const boxes =
			e.currentTarget.form?.querySelectorAll<HTMLInputElement>('input[name="antigens"]');
		for (const box of boxes ?? []) if (ids.includes(box.value)) box.checked = true;
	}
</script>

<Field id="vaccine-given_on" label="Date given" error={error('given_on')}>
	{#snippet children(control)}
		<Input
			{...control}
			name="given_on"
			type="date"
			value={text('given_on')}
			min={birthDate}
			max={today}
			required
		/>
	{/snippet}
</Field>

<Field
	id="vaccine-product_shortcut"
	label="Product shortcut"
	hint="Choosing a product ticks the vaccines it covers; you can also tick them yourself"
>
	{#snippet children(control)}
		<select {...control} name="product_shortcut" class={SELECT_CLASS} onchange={tick}>
			<option value="">Choose a product</option>
			{#each catalogue.products as p (p.id)}
				<option value={p.id} data-antigens={p.antigens.join(',')}>{p.name}</option>
			{/each}
		</select>
	{/snippet}
</Field>

<fieldset
	class="flex flex-col gap-2"
	aria-describedby={error('antigens') ? ANTIGENS_ERROR : undefined}
>
	<legend class="mb-2 text-sm font-medium">Vaccines given</legend>
	{#if error('antigens')}
		<p id={ANTIGENS_ERROR} class={ERROR_CLASS} role="alert">{error('antigens')}</p>
	{/if}
	{#each catalogue.antigens as a (a.id)}
		<label class="flex min-h-8 items-center gap-2 text-sm">
			<input
				type="checkbox"
				name="antigens"
				value={a.id}
				checked={values.antigens.includes(a.id)}
				aria-invalid={error('antigens') ? 'true' : undefined}
				aria-describedby={error('antigens') ? ANTIGENS_ERROR : undefined}
				class="size-4"
			/>
			{a.name}
		</label>
	{/each}
</fieldset>

<Field id="vaccine-lot" label="Lot number" error={error('lot')}>
	{#snippet children(control)}
		<Input {...control} name="lot" autocomplete="off" value={text('lot')} />
	{/snippet}
</Field>

<Field id="vaccine-product" label="Product name" error={error('product')}>
	{#snippet children(control)}
		<Input {...control} name="product" autocomplete="off" value={text('product')} />
	{/snippet}
</Field>

<Field id="vaccine-note" label="Note" error={error('note')}>
	{#snippet children(control)}
		<Input {...control} name="note" autocomplete="off" value={text('note')} />
	{/snippet}
</Field>

{#if values.given_on_original !== undefined}
	{#each ORIGINALS as name (name)}
		<input type="hidden" name="{name}_original" value={text(`${name}_original`)} />
	{/each}
{/if}
