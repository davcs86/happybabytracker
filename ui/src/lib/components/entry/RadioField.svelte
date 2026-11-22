<script lang="ts">
	import { ERROR_CLASS } from '#lib/components/field.js';
	import type { EntryValues } from '#lib/entry-edit.js';

	let {
		name,
		legend,
		options,
		values,
		error,
		hint = null,
		required = false
	}: {
		name: string;
		legend: string;
		options: readonly string[];
		values: EntryValues;
		error: (name: string) => string | null;
		hint?: string | null;
		required?: boolean;
	} = $props();

	const id = $derived(`entry-${name}`);
	const label = (option: string) => option.charAt(0).toUpperCase() + option.slice(1);
	const describedBy = $derived(
		[hint ? `${id}-hint` : null, error(name) ? `${id}-error` : null].filter(Boolean).join(' ') ||
			undefined
	);
</script>

<fieldset
	role="radiogroup"
	aria-labelledby="{id}-legend"
	class="flex flex-col gap-1"
	aria-invalid={error(name) ? 'true' : undefined}
	aria-describedby={describedBy}
>
	<legend id="{id}-legend" class="mb-1 text-sm font-medium">{legend}</legend>
	<div class="flex flex-wrap gap-x-4">
		{#each options as option (option)}
			<label class="flex min-h-11 items-center gap-2 text-sm">
				<input type="radio" {name} value={option} checked={values[name] === option} {required} />
				{label(option)}
			</label>
		{/each}
	</div>
	{#if hint}<p id="{id}-hint" class="text-sm text-muted-foreground">{hint}</p>{/if}
	{#if error(name)}<p id="{id}-error" class={ERROR_CLASS} role="alert">{error(name)}</p>{/if}
</fieldset>
