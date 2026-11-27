<script lang="ts" module>
	export type Control = { id: string; 'aria-invalid'?: 'true'; 'aria-describedby'?: string };
</script>

<script lang="ts">
	import type { Snippet } from 'svelte';
	import { ERROR_CLASS } from '#lib/components/field.js';
	import { Label } from '#lib/components/ui/label/index.js';

	let {
		id,
		label,
		error = null,
		hint = null,
		children
	}: {
		id: string;
		label: string;
		error?: string | null;
		hint?: string | null;
		children: Snippet<[Control]>;
	} = $props();

	const describedBy = $derived(
		[hint ? `${id}-hint` : null, error ? `${id}-error` : null].filter(Boolean).join(' ') ||
			undefined
	);
</script>

<div class="flex flex-col gap-1.5">
	<Label for={id}>{label}</Label>
	{@render children({
		id,
		'aria-invalid': error ? 'true' : undefined,
		'aria-describedby': describedBy
	})}
	{#if hint}<p id="{id}-hint" class="text-[12.5px] leading-normal text-muted-foreground">
			{hint}
		</p>{/if}
	{#if error}<p id="{id}-error" class={ERROR_CLASS} role="alert">{error}</p>{/if}
</div>
