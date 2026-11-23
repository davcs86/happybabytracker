<script lang="ts">
	import EntryTimeField from '#lib/components/entry/EntryTimeField.svelte';
	import SourceVolumes from '#lib/components/entry/SourceVolumes.svelte';
	import VolumeField from '#lib/components/entry/VolumeField.svelte';
	import { ERROR_CLASS } from '#lib/components/field.js';
	import { BOTTLE_CONTENTS, BOTTLE_CONTENT_LABEL } from '#lib/constants.js';
	import type { EntryValues } from '#lib/entry-edit.js';

	let { values, error }: { values: EntryValues; error: (name: string) => string | null } = $props();

	const chosen = $derived(values.contents as string[]);
	const advanced = $derived(values.advanced === '1');
	// The stored sources, which a total saved from the simple form replaces.
	const hadSources = $derived(!!values.sources_original);
</script>

<EntryTimeField name="at" label="Time" {values} {error} required />
{#if advanced}
	<SourceVolumes {values} {error} />
{:else}
	<!-- Each box carries the error: a plain group has no invalid state of its own. -->
	<fieldset class="flex flex-col gap-1">
		<legend class="mb-1 text-sm font-medium">Contents</legend>
		<div class="grid grid-cols-2 gap-x-4">
			{#each BOTTLE_CONTENTS as content (content)}
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
			{/each}
		</div>
		{#if error('contents')}
			<p id="entry-contents-error" class={ERROR_CLASS} role="alert">{error('contents')}</p>
		{/if}
	</fieldset>
	<VolumeField
		{values}
		{error}
		hint={hadSources ? 'Saving one total replaces the volumes per source' : null}
	/>
	{#if hadSources}
		<input type="hidden" name="sources_original" value={values.sources_original} />
	{/if}
{/if}
