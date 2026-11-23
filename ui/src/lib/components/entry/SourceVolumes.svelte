<script lang="ts">
	import Field from '#lib/components/entry/Field.svelte';
	import { ERROR_CLASS, SELECT_CLASS } from '#lib/components/field.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { BOTTLE_CONTENTS, BOTTLE_CONTENT_LABEL, VOLUME_UNITS } from '#lib/constants.js';
	import type { EntryValues } from '#lib/entry-edit.js';
	import { formatVolume } from '#lib/format.js';
	import type { VolumeUnit } from '#lib/types.js';

	let { values, error }: { values: EntryValues; error: (name: string) => string | null } = $props();

	const UNIT_LABEL = { ml: 'ml', floz: 'fl oz' } as const;
	const chosen = $derived(values.contents as string[]);
	const total = $derived(
		(values.sources_total as string | undefined) ??
			formatVolume({ ml: 0, floz_hundredths: 0 }, values.volume_unit as VolumeUnit)
	);
	// A refused total or source is no single input's fault: every ticked input is marked.
	const invalid = $derived(error('sources') || error('volume') ? ('true' as const) : undefined);
	const describedBy = $derived(
		[error('sources') ? 'entry-sources-error' : null, error('volume') ? 'entry-total-error' : null]
			.filter(Boolean)
			.join(' ') || undefined
	);
</script>

<Field id="entry-volume_unit" label="Volume unit">
	{#snippet children(control)}
		<select
			{...control}
			name="volume_unit"
			class={SELECT_CLASS}
			value={values.volume_unit as string}
		>
			{#each VOLUME_UNITS as unit (unit)}
				<option value={unit}>{UNIT_LABEL[unit]}</option>
			{/each}
		</select>
	{/snippet}
</Field>
<!-- Each box carries the error: a plain group has no invalid state of its own. CSS shows the volume
     of a ticked box, so this works without JavaScript. -->
<fieldset class="flex flex-col gap-1">
	<legend class="mb-1 text-sm font-medium">Contents</legend>
	<p class="mb-1 text-sm text-muted-foreground">Tick each source and enter its volume</p>
	{#each BOTTLE_CONTENTS as content (content)}
		<div class="group/source flex flex-col">
			<label class="flex min-h-11 items-center gap-2 text-sm">
				<input
					type="checkbox"
					name="contents"
					value={content}
					checked={chosen.includes(content)}
					aria-invalid={error('contents') ? 'true' : undefined}
					aria-describedby={error('contents') ? 'entry-contents-error' : undefined}
				/>
				{BOTTLE_CONTENT_LABEL[content]}
			</label>
			<div class="hidden pb-2 pl-6 group-has-[:checked]/source:block">
				<Field id="entry-sources_{content}" label="{BOTTLE_CONTENT_LABEL[content]} volume">
					{#snippet children(control)}
						<Input
							{...control}
							name="sources_{content}"
							inputmode="decimal"
							autocomplete="off"
							value={values[`sources_${content}`] as string}
							required
							aria-invalid={invalid}
							aria-describedby={describedBy}
						/>
					{/snippet}
				</Field>
			</div>
		</div>
	{/each}
	{#if error('contents')}
		<p id="entry-contents-error" class={ERROR_CLASS} role="alert">{error('contents')}</p>
	{/if}
	{#if error('sources')}
		<p id="entry-sources-error" class={ERROR_CLASS} role="alert">{error('sources')}</p>
	{/if}
</fieldset>
<p class="text-sm font-medium">Total: {total}</p>
{#if error('volume')}
	<p id="entry-total-error" class={ERROR_CLASS} role="alert">{error('volume')}</p>
{/if}
{#if values.sources_original !== undefined}
	<input type="hidden" name="volume_unit_original" value={values.volume_unit_original} />
	<input type="hidden" name="sources_original" value={values.sources_original} />
	{#each BOTTLE_CONTENTS as content (content)}
		<input
			type="hidden"
			name="sources_{content}_original"
			value={values[`sources_${content}_original`]}
		/>
	{/each}
{/if}
