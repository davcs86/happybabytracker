<script lang="ts">
	import DoseForm from '#lib/components/medication/DoseForm.svelte';
	import { scheduleLabel, whenLabel } from '#lib/medication.js';
	import { medicationEntryPath, medicationPath } from '#lib/paths.js';
	import type { Medication } from '#lib/types.js';

	let {
		course,
		kidId,
		zone,
		today,
		amount = '',
		error = null
	}: {
		course: Medication;
		kidId: string;
		zone: string;
		today: string;
		// What a refused one-tap submit for this course sent back.
		amount?: string;
		error?: string | null;
	} = $props();
</script>

<li class="rounded-lg border p-3 [overflow-wrap:anywhere]">
	<h3 class="font-medium">
		<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- the edit page's path helper, absolute like the other links to pages without JavaScript -->
		<a href={medicationEntryPath(kidId, course.id)} class="underline">{course.name}</a>
	</h3>
	<p class="text-sm">{course.dose}</p>
	<p class="text-sm text-muted-foreground">
		{scheduleLabel(course.schedule, course.startAt, zone)}
	</p>
	<p class="text-sm">
		{#if course.lastDose}
			Last dose logged at {whenLabel(course.lastDose.givenAt, zone, today)}
		{:else}
			No dose logged yet
		{/if}
	</p>
	{#if course.nextDue}
		<p class="text-sm">Next due {whenLabel(course.nextDue.dueAt, zone, today)}</p>
	{/if}
	<DoseForm id={course.id} name={course.name} value={amount} {error} />
	<p class="mt-2 text-sm">
		<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- a query on this same page -->
		<a href="{medicationPath(kidId)}?medication={course.id}" class="underline"
			>Show doses<span class="sr-only"> of {course.name}</span></a
		>
	</p>
</li>
