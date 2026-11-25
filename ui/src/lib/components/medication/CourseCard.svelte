<script lang="ts">
	import NotebookPenIcon from '@lucide/svelte/icons/notebook-pen';
	import PencilIcon from '@lucide/svelte/icons/pencil';
	import DoseForm from '#lib/components/medication/DoseForm.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
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

<li class="flex flex-col gap-3 rounded-xl border bg-card p-3.5 [overflow-wrap:anywhere]">
	<div class="flex items-center gap-3">
		<span
			aria-hidden="true"
			class="grid size-11 shrink-0 place-items-center rounded-xl bg-accent text-primary-ink"
		>
			<NotebookPenIcon class="size-[22px]" />
		</span>
		<div class="flex min-w-0 flex-1 flex-col gap-0.5">
			<h3 class="text-[17px] font-semibold">{course.name}</h3>
			<p class="text-[15px]">{course.dose}</p>
		</div>
		<Button
			href={medicationEntryPath(kidId, course.id)}
			variant="outline"
			size="icon"
			aria-label="Edit {course.name}"
			title="Edit {course.name}"
		>
			<PencilIcon aria-hidden="true" class="size-5!" />
		</Button>
	</div>
	<div class="flex flex-col gap-0.5 text-sm">
		<p class="text-[13.5px] text-muted-foreground">
			{scheduleLabel(course.schedule, course.startAt, zone)}
		</p>
		<p>
			{#if course.lastDose}
				Last dose logged at {whenLabel(course.lastDose.givenAt, zone, today)}
			{:else}
				No dose logged yet
			{/if}
		</p>
		{#if course.nextDue}
			<p>Next due {whenLabel(course.nextDue.dueAt, zone, today)}</p>
		{/if}
	</div>
	<DoseForm id={course.id} name={course.name} value={amount} {error}>
		{#snippet actions()}
			<Button href="{medicationPath(kidId)}?medication={course.id}" variant="outline" class="px-4"
				>Show doses<span class="sr-only"> of {course.name}</span></Button
			>
		{/snippet}
	</DoseForm>
</li>
