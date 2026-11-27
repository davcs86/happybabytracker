<script lang="ts">
	import { ERROR_CLASS, CHOICE_CLASS } from '#lib/components/field.js';
	import type { EntryValues } from '#lib/entry-edit.js';

	let {
		name,
		legend,
		options,
		values,
		error,
		hint = null,
		required = false,
		blank = null,
		labels = null,
		descriptions = null
	}: {
		name: string;
		legend: string;
		options: readonly string[];
		values: EntryValues;
		error: (name: string) => string | null;
		hint?: string | null;
		required?: boolean;
		// The label of one more radio with an empty value, for a field that can be left unset.
		blank?: string | null;
		// Each option's words; without it an option is shown capitalised.
		labels?: Readonly<Record<string, string>> | null;
		// A line under an option, read as that radio's description.
		descriptions?: Readonly<Record<string, string>> | null;
	} = $props();

	const id = $derived(`entry-${name}`);
	const label = (option: string) =>
		labels?.[option] ?? option.charAt(0).toUpperCase() + option.slice(1);
	const describedBy = $derived(
		[hint ? `${id}-hint` : null, error(name) ? `${id}-error` : null].filter(Boolean).join(' ') ||
			undefined
	);
</script>

<fieldset
	role="radiogroup"
	aria-labelledby="{id}-legend"
	class="flex flex-col gap-1.5"
	aria-invalid={error(name) ? 'true' : undefined}
	aria-describedby={describedBy}
>
	<legend id="{id}-legend" class="mb-0.5 text-[13px] leading-4 text-muted-foreground"
		>{legend}</legend
	>
	<div class="flex flex-col gap-0.5">
		{#each options as option (option)}
			{@const description = descriptions?.[option]}
			<label
				class={[
					'flex min-h-11 items-center gap-3 text-[15px]',
					descriptions && '[overflow-wrap:anywhere]'
				]}
			>
				<input
					type="radio"
					{name}
					value={option}
					checked={values[name] === option}
					{required}
					aria-describedby={description ? `${id}-${option}-description` : undefined}
					class={CHOICE_CLASS}
				/>
				{label(option)}
			</label>
			{#if description}
				<p
					id="{id}-{option}-description"
					class="-mt-1.5 pb-1 pl-8 text-[13px] leading-snug [overflow-wrap:anywhere] text-muted-foreground"
				>
					{description}
				</p>
			{/if}
		{/each}
		{#if blank !== null}
			<label class="flex min-h-11 items-center gap-3 text-[15px]">
				<input type="radio" {name} value="" checked={values[name] === ''} class={CHOICE_CLASS} />
				{blank}
			</label>
		{/if}
	</div>
	{#if hint}<p id="{id}-hint" class="text-[12.5px] leading-normal text-muted-foreground">
			{hint}
		</p>{/if}
	{#if error(name)}<p id="{id}-error" class={ERROR_CLASS} role="alert">{error(name)}</p>{/if}
</fieldset>
