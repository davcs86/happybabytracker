<script lang="ts">
	import BottleFields from '#lib/components/entry/BottleFields.svelte';
	import DiaperFields from '#lib/components/entry/DiaperFields.svelte';
	import MealFields from '#lib/components/entry/MealFields.svelte';
	import NursingFields from '#lib/components/entry/NursingFields.svelte';
	import PumpingFields from '#lib/components/entry/PumpingFields.svelte';
	import SleepFields from '#lib/components/entry/SleepFields.svelte';
	import type { EntryStage, EntryValues } from '#lib/entry-edit.js';
	import type { TrackerKind } from '#lib/types.js';

	let {
		kind,
		stage,
		values,
		error
	}: {
		kind: TrackerKind;
		stage: EntryStage;
		values: EntryValues;
		error: (name: string) => string | null;
	} = $props();
</script>

{#if kind === 'bottles'}
	<BottleFields {values} {error} />
{:else if kind === 'diapers'}
	<DiaperFields {values} {error} />
{:else if kind === 'meals'}
	<MealFields {values} {error} />
{:else if kind === 'pumping'}
	<PumpingFields {values} {error} />
{:else if kind === 'sleep'}
	<SleepFields {stage} {values} {error} />
{:else}
	<NursingFields {stage} {values} {error} />
{/if}
