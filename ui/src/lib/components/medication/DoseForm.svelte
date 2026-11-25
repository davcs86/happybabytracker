<script lang="ts">
	import { enhance, type SubmitFunction } from '$app/forms';
	import { page } from '$app/state';
	import type { Snippet } from 'svelte';
	import Field from '#lib/components/entry/Field.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';

	// value is what a refused submit sent back; error belongs to this course alone.
	let {
		id,
		name,
		value = '',
		error = null,
		actions
	}: {
		id: string;
		name: string;
		value?: string;
		error?: string | null;
		actions?: Snippet;
	} = $props();

	let amount = $derived(value);
	let sending = $state(false);
	// The list's course filter, sent back so the redirect keeps it.
	const filter = $derived(page.url.searchParams.get('medication'));
	// A logged dose redirects to this same page, so the form is not reloaded: clear it here.
	const clearWhenLogged: SubmitFunction = () => {
		sending = true;
		return async ({ result, update }) => {
			try {
				if (result.type === 'redirect') amount = '';
				await update({ reset: false });
			} finally {
				sending = false;
			}
		};
	};
</script>

<form
	method="POST"
	action="?/log"
	class="flex flex-col gap-3"
	aria-label="Log a dose of {name}"
	novalidate
	use:enhance={clearWhenLogged}
>
	<input type="hidden" name="medication" value={id} />
	{#if filter}<input type="hidden" name="filter" value={filter} />{/if}
	<Field id="dose-amount-{id}" label="Amount (optional)" {error}>
		{#snippet children(control)}
			<Input {...control} name="amount" autocomplete="off" bind:value={amount} />
		{/snippet}
	</Field>
	<div class="flex flex-wrap items-center gap-2">
		<!-- No fade: re-enabled after its request, the button must be at full contrast at once. -->
		<Button type="submit" class="transition-none" disabled={sending}
			>Log dose now<span class="sr-only"> of {name}</span></Button
		>
		{@render actions?.()}
	</div>
</form>
