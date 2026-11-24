<script lang="ts">
	import Field from '#lib/components/entry/Field.svelte';
	import { ERROR_CLASS } from '#lib/components/field.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { MAX_EVERY_HOURS, MAX_SCHEDULE_TIMES, MIN_EVERY_HOURS } from '#lib/constants.js';

	let {
		values,
		error
	}: {
		values: { kind: string; times: string[]; hours: string };
		error: string | null;
	} = $props();

	const KINDS = [
		['none', 'As needed'],
		['times', 'At fixed times'],
		['every', 'Every few hours']
	] as const;
	const SLOTS = Array.from({ length: MAX_SCHEDULE_TIMES }, (_, i) => i);
	const ERROR_ID = 'course-schedule-error';
	const invalid = (kind: string) => (error && values.kind === kind ? 'true' : undefined);
</script>

<!-- CSS shows the fields of the chosen kind, so this works without JavaScript. -->
<fieldset class="group flex flex-col gap-2">
	<legend id="course-schedule-legend" class="mb-1 text-sm font-medium">Schedule</legend>
	<div
		role="radiogroup"
		aria-labelledby="course-schedule-legend"
		aria-invalid={error ? 'true' : undefined}
		aria-describedby={error ? ERROR_ID : undefined}
		class="flex flex-wrap gap-x-4"
	>
		{#each KINDS as [kind, label] (kind)}
			<label class="flex min-h-11 items-center gap-2 text-sm">
				<input
					type="radio"
					id="course-kind-{kind}"
					name="kind"
					value={kind}
					checked={values.kind === kind}
				/>
				{label}
			</label>
		{/each}
	</div>
	{#if error}<p id={ERROR_ID} class={ERROR_CLASS} role="alert">{error}</p>{/if}

	<div class="hidden grid-cols-2 gap-3 group-has-[#course-kind-times:checked]:grid sm:grid-cols-3">
		{#each SLOTS as i (i)}
			<Field id="course-time-{i}" label="Time {i + 1}">
				{#snippet children(control)}
					<Input
						id={control.id}
						name="times"
						type="time"
						value={values.times[i] ?? ''}
						aria-invalid={invalid('times')}
						aria-describedby={error && values.kind === 'times' ? ERROR_ID : undefined}
					/>
				{/snippet}
			</Field>
		{/each}
	</div>
	<div class="hidden group-has-[#course-kind-every:checked]:block">
		<Field
			id="course-hours"
			label="Hours between doses"
			hint="A whole number from {MIN_EVERY_HOURS} to {MAX_EVERY_HOURS}, counted from the start below"
		>
			{#snippet children(control)}
				<Input
					id={control.id}
					name="hours"
					inputmode="numeric"
					autocomplete="off"
					value={values.hours}
					aria-invalid={invalid('every')}
					aria-describedby={[
						control['aria-describedby'],
						error && values.kind === 'every' ? ERROR_ID : null
					]
						.filter(Boolean)
						.join(' ')}
				/>
			{/snippet}
		</Field>
	</div>
</fieldset>
