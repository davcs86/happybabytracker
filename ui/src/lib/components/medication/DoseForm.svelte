<script lang="ts">
	import { enhance, type SubmitFunction } from '$app/forms';
	import Field from '#lib/components/entry/Field.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';

	// value is what a refused submit sent back; error belongs to this course alone.
	let {
		id,
		name,
		value = '',
		error = null
	}: { id: string; name: string; value?: string; error?: string | null } = $props();

	let amount = $derived(value);
	// A logged dose redirects to this same page, so the form is not reloaded: clear it here.
	const clearWhenLogged: SubmitFunction =
		() =>
		async ({ result, update }) => {
			if (result.type === 'redirect') amount = '';
			await update({ reset: false });
		};
</script>

<form
	method="POST"
	action="?/log"
	class="mt-3 flex max-w-md flex-col gap-2"
	aria-label="Log a dose of {name}"
	novalidate
	use:enhance={clearWhenLogged}
>
	<input type="hidden" name="medication" value={id} />
	<Field id="dose-amount-{id}" label="Amount (optional)" {error}>
		{#snippet children(control)}
			<Input {...control} name="amount" autocomplete="off" bind:value={amount} />
		{/snippet}
	</Field>
	<Button type="submit" class="self-start"
		>Log dose now<span class="sr-only"> of {name}</span></Button
	>
</form>
