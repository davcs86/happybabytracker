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
		today,
		prefix = ''
	}: {
		values: { antigens: string[] } & Record<string, string | string[] | undefined>;
		error: (name: string) => string | null;
		catalogue: VaccineCatalogue;
		birthDate: string;
		today: string;
		// Prepended to every name and id, so several blocks share one form; values and error stay unprefixed.
		prefix?: string;
	} = $props();

	const ORIGINALS = ['given_on', 'antigens', 'lot', 'product', 'note'] as const;
	const ANTIGENS_ERROR = $derived(`${prefix}vaccine-antigens-error`);
	const text = (name: string) => {
		const v = values[name];
		return typeof v === 'string' ? v : '';
	};

	// Progressive: without JavaScript the shortcut does nothing and the antigens are ticked by hand.
	function tick(e: Event & { currentTarget: HTMLSelectElement }) {
		const ids = (e.currentTarget.selectedOptions[0]?.dataset.antigens ?? '').split(',');
		const boxes = e.currentTarget.form?.querySelectorAll<HTMLInputElement>(
			`input[name="${prefix}antigens"]`
		);
		for (const box of boxes ?? []) if (ids.includes(box.value)) box.checked = true;
	}
</script>

<div class="flex flex-col gap-3.5">
	<Field id="{prefix}vaccine-given_on" label="Date given" error={error('given_on')}>
		{#snippet children(control)}
			<Input
				{...control}
				name="{prefix}given_on"
				type="date"
				value={text('given_on')}
				min={birthDate}
				max={today}
				required
			/>
		{/snippet}
	</Field>

	<Field
		id="{prefix}vaccine-product_shortcut"
		label="Product shortcut"
		hint="With JavaScript on, choosing a product ticks the vaccines it covers; you can always tick them yourself."
	>
		{#snippet children(control)}
			<select {...control} name="{prefix}product_shortcut" class={SELECT_CLASS} onchange={tick}>
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
		<legend class="mb-1.5 text-[13px] text-muted-foreground">Vaccines given</legend>
		{#if error('antigens')}
			<p id={ANTIGENS_ERROR} class={ERROR_CLASS} role="alert">{error('antigens')}</p>
		{/if}
		{#each catalogue.antigens as a (a.id)}
			<label
				class="flex min-h-12 items-center gap-3 rounded-xl border bg-card px-3 text-[14.5px] [overflow-wrap:anywhere]"
			>
				<input
					type="checkbox"
					name="{prefix}antigens"
					value={a.id}
					checked={values.antigens.includes(a.id)}
					aria-invalid={error('antigens') ? 'true' : undefined}
					aria-describedby={error('antigens') ? ANTIGENS_ERROR : undefined}
					class="size-5 shrink-0 accent-primary"
				/>
				{a.name}
			</label>
		{/each}
	</fieldset>
</div>

<div class="flex flex-col gap-3.5 xl:row-span-2">
	<Field id="{prefix}vaccine-lot" label="Lot number" error={error('lot')}>
		{#snippet children(control)}
			<Input {...control} name="{prefix}lot" autocomplete="off" value={text('lot')} />
		{/snippet}
	</Field>

	<Field id="{prefix}vaccine-product" label="Product name" error={error('product')}>
		{#snippet children(control)}
			<Input {...control} name="{prefix}product" autocomplete="off" value={text('product')} />
		{/snippet}
	</Field>

	<Field id="{prefix}vaccine-note" label="Note" error={error('note')}>
		{#snippet children(control)}
			<Input {...control} name="{prefix}note" autocomplete="off" value={text('note')} />
		{/snippet}
	</Field>
</div>

{#if values.given_on_original !== undefined}
	{#each ORIGINALS as name (name)}
		<input type="hidden" name="{prefix}{name}_original" value={text(`${name}_original`)} />
	{/each}
{/if}
