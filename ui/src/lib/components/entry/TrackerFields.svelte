<script lang="ts">
	import BottleFields from '#lib/components/entry/BottleFields.svelte';
	import DiaperFields from '#lib/components/entry/DiaperFields.svelte';
	import MealFields from '#lib/components/entry/MealFields.svelte';
	import NursingFields from '#lib/components/entry/NursingFields.svelte';
	import PumpingFields from '#lib/components/entry/PumpingFields.svelte';
	import SleepFields from '#lib/components/entry/SleepFields.svelte';
	import SymptomFields from '#lib/components/entry/SymptomFields.svelte';
	import type { EntryStage, EntryValues } from '#lib/entry-edit.js';
	import type { TrackerKind } from '#lib/types.js';

	let {
		kind,
		stage,
		part,
		values,
		error,
		estimate = null
	}: {
		kind: TrackerKind;
		stage: EntryStage;
		// The primary fields sit in the left column from 1280 px, the optional ones in the right.
		part: 'primary' | 'optional';
		values: EntryValues;
		error: (name: string) => string | null;
		estimate?: string | null;
	} = $props();
</script>

{#if kind === 'bottles'}
	{#if part === 'primary'}<BottleFields {values} {error} />{/if}
{:else if kind === 'diapers'}
	{#if part === 'primary'}<DiaperFields {values} {error} />{/if}
{:else if kind === 'meals'}
	<MealFields {part} {values} {error} />
{:else if kind === 'pumping'}
	{#if part === 'primary'}<PumpingFields {values} {error} />{/if}
{:else if kind === 'symptoms'}
	<SymptomFields {part} {values} {error} />
{:else if kind === 'sleep'}
	{#if part === 'primary'}<SleepFields {stage} {values} {error} />{/if}
{:else}
	<NursingFields {part} {stage} {values} {error} {estimate} />
{/if}
