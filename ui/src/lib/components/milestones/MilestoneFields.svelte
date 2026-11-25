<script lang="ts">
	import Field from '#lib/components/entry/Field.svelte';
	import { Input } from '#lib/components/ui/input/index.js';

	let {
		values,
		error,
		custom,
		birthDate,
		today
	}: {
		values: Record<string, string | undefined>;
		error: (name: string) => string | null;
		custom: boolean;
		birthDate: string;
		today: string;
	} = $props();

	const text = (name: string) => values[name] ?? '';
</script>

<div class="flex flex-col gap-3.5">
	{#if custom}
		<Field id="milestone-text" label="Milestone" error={error('text')}>
			{#snippet children(control)}
				<Input {...control} name="text" autocomplete="off" value={text('text')} required />
			{/snippet}
		</Field>
	{/if}

	<Field id="milestone-reached_on" label="Date reached" error={error('reached_on')}>
		{#snippet children(control)}
			<Input
				{...control}
				name="reached_on"
				type="date"
				value={text('reached_on')}
				min={birthDate}
				max={today}
				required
			/>
		{/snippet}
	</Field>
</div>

<div class="flex flex-col gap-3.5 xl:row-span-2">
	<Field id="milestone-note" label="Note" error={error('note')}>
		{#snippet children(control)}
			<Input {...control} name="note" autocomplete="off" value={text('note')} />
		{/snippet}
	</Field>
</div>

{#if values.reached_on_original !== undefined}
	{#each [...(custom ? ['text'] : []), 'reached_on', 'note'] as name (name)}
		<input type="hidden" name="{name}_original" value={text(`${name}_original`)} />
	{/each}
{/if}
