<script lang="ts">
	import Field from '#lib/components/entry/Field.svelte';
	import ScheduleFields from '#lib/components/medication/ScheduleFields.svelte';
	import { Input } from '#lib/components/ui/input/index.js';
	import type { CourseFormValues } from '#lib/server/medication-form.js';

	let {
		values,
		error,
		zone,
		startHint
	}: {
		values: CourseFormValues;
		error: (name: string) => string | null;
		zone: string;
		startHint: string;
	} = $props();
</script>

<p class="text-sm text-muted-foreground">Times are in {zone}</p>
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
<Field id="course-active" label="Active" error={error('active')}>
	{#snippet children(control)}
		<input
			{...control}
			type="checkbox"
			name="active"
			class="size-5 self-start"
			checked={values.active}
		/>
	{/snippet}
</Field>
<Field id="course-note" label="Note" error={error('note')}>
	{#snippet children(control)}
		<Input {...control} name="note" autocomplete="off" value={values.note} />
	{/snippet}
</Field>
