<script lang="ts">
	import CheckIcon from '@lucide/svelte/icons/check';
	import Field from '#lib/components/entry/Field.svelte';
	import { ERROR_CLASS } from '#lib/components/field.js';
	import ScheduleFields from '#lib/components/medication/ScheduleFields.svelte';
	import { Input } from '#lib/components/ui/input/index.js';
	import type { CourseFormValues } from '#lib/server/medication-form.js';

	let {
		values,
		error,
		startHint
	}: {
		values: CourseFormValues;
		error: (name: string) => string | null;
		startHint: string;
	} = $props();

	const activeError = $derived(error('active'));
</script>

<!-- Two groups for the wide layout: what the course is, then its dates, state and note. -->
<div class="flex flex-col gap-3.5">
	<Field id="course-name" label="Name" error={error('name')}>
		{#snippet children(control)}
			<Input {...control} name="name" autocomplete="off" value={values.name} />
		{/snippet}
	</Field>
	<Field id="course-dose" label="Dose" error={error('dose')}>
		{#snippet children(control)}
			<Input {...control} name="dose" autocomplete="off" value={values.dose} />
		{/snippet}
	</Field>
	<ScheduleFields {values} error={error('schedule')} />
</div>
<div class="flex flex-col gap-3.5 xl:row-span-2">
	<Field id="course-start_at" label="Start" hint={startHint} error={error('start_at')}>
		{#snippet children(control)}
			<Input {...control} name="start_at" type="datetime-local" value={values.start_at} />
		{/snippet}
	</Field>
	<Field id="course-end_on" label="Last day (optional)" error={error('end_on')}>
		{#snippet children(control)}
			<Input {...control} name="end_on" type="date" value={values.end_on} />
		{/snippet}
	</Field>
	<div class="flex flex-col gap-2">
		<label class="relative flex min-h-11 items-center gap-3 text-[15px]">
			<input
				id="course-active"
				type="checkbox"
				name="active"
				checked={values.active}
				aria-invalid={activeError ? 'true' : undefined}
				aria-describedby={activeError ? 'course-active-error' : undefined}
				class="peer size-5 shrink-0 appearance-none rounded-[4px] border border-[color-mix(in_oklab,var(--foreground)_55%,var(--background))] bg-background shadow-xs checked:border-primary checked:bg-primary"
			/>
			<CheckIcon
				aria-hidden="true"
				class="pointer-events-none absolute left-[3px] hidden size-3.5 text-primary-foreground peer-checked:block"
			/>
			Active
		</label>
		{#if activeError}
			<p id="course-active-error" class={ERROR_CLASS} role="alert">{activeError}</p>
		{/if}
	</div>
	<Field id="course-note" label="Note" error={error('note')}>
		{#snippet children(control)}
			<Input {...control} name="note" autocomplete="off" value={values.note} />
		{/snippet}
	</Field>
</div>
