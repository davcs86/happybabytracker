<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import { onMount, tick } from 'svelte';
	import Field from '#lib/components/entry/Field.svelte';
	import { CHOICE_CLASS } from '#lib/components/field.js';
	import FormCancel from '#lib/components/FormCancel.svelte';
	import FormFrame from '#lib/components/FormFrame.svelte';
	import MeasurementFields from '#lib/components/growth/MeasurementFields.svelte';
	import CourseFields from '#lib/components/medication/CourseFields.svelte';
	import DoseFields from '#lib/components/vaccines/DoseFields.svelte';
	import { Alert } from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Textarea } from '#lib/components/ui/textarea/index.js';
	import {
		MAX_OUTCOME_MEDICATIONS,
		MAX_OUTCOME_VACCINATIONS,
		VISIT_KIND_LABEL
	} from '#lib/constants.js';
	import { isEmbed, savedAs } from '#lib/drawer.js';
	import { visitWhen } from '#lib/visits.js';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	const values = $derived(form?.values ?? data.values);
	// After a stale save the page shows the visit as it is now, with what was typed.
	const current = $derived(form?.current ?? data.current);
	const fieldError = (path: string) => (form?.field === path ? form.message : null);
	const embed = isEmbed();
	const back = $derived(
		resolve('/(app)/kids/[kid]/visits/[visit]', { kid: data.kid.id, visit: data.visit.id })
	);

	// Blocks added with JavaScript; without it the page posts one of each and saving again adds more.
	let vaccinations = $derived(
		values.vaccinations.length ? values.vaccinations : [data.blanks.vaccination]
	);
	let medications = $derived(
		values.medications.length ? values.medications : [data.blanks.medication]
	);
	let hydrated = $state(false);
	onMount(() => (hydrated = true));

	async function addBlock(list: 'vaccine' | 'medication') {
		if (list === 'vaccine')
			vaccinations = [...vaccinations, structuredClone(data.blanks.vaccination)];
		else medications = [...medications, structuredClone(data.blanks.medication)];
		await tick();
		const n = list === 'vaccine' ? vaccinations.length : medications.length;
		document.getElementById(`outcome-${list}-${n - 1}`)?.focus();
	}

	const SECTION = 'flex flex-col gap-3.5';
	const BLOCK = $derived([
		'flex flex-col gap-3.5',
		!embed && 'xl:grid xl:grid-cols-[448px_448px] xl:grid-rows-[auto_1fr] xl:gap-x-6'
	]);
</script>

{#snippet info()}
	<p class="text-[13.5px] text-muted-foreground">
		{VISIT_KIND_LABEL[data.visit.kind]}, {visitWhen(data.visit.startsAt, data.kid.timeZone)}
	</p>
	<p class="text-[13.5px] [overflow-wrap:anywhere] text-muted-foreground">
		Times are in {data.kid.timeZone}
	</p>
{/snippet}

{#snippet addBox(name: string, label: string, checked: boolean)}
	<label class="flex min-h-11 items-center gap-3 text-[15px] font-semibold">
		<input type="checkbox" id="outcome-{name}" {name} {checked} class={CHOICE_CLASS} />
		{label}
	</label>
{/snippet}

<FormFrame
	class="flex flex-col gap-[18px]"
	documentTitle="After the visit · {data.kid.name} · happybabytracker"
>
	{#snippet chrome()}
		<div class="mt-5 flex flex-col gap-1.5">
			<h1>After the visit</h1>
			<p class="text-sm leading-snug text-muted-foreground">{data.ages[data.kid.id]}</p>
			{@render info()}
		</div>
	{/snippet}
	{#if embed}
		<div class="flex flex-col gap-1.5">{@render info()}</div>
	{/if}
	{#if data.notPast}
		<Alert class="[overflow-wrap:anywhere]">{data.notPast}</Alert>
		<div class="flex gap-2"><FormCancel href={back} size="default" class="" /></div>
	{:else}
		<form
			method="POST"
			action="?/save"
			class="flex flex-col gap-[18px]"
			novalidate
			use:enhance={savedAs('Visit recorded')}
		>
			{#if form?.field}
				<Alert>Check the fields with a message.</Alert>
			{:else if form?.message}
				<Alert class="[overflow-wrap:anywhere]">{form.message}</Alert>
			{/if}
			<input type="hidden" name="revision" value={current.revision} />
			<input type="hidden" name="outcome_notes_original" value={current.notes} />

			<div class="xl:max-w-[448px]">
				<Field
					id="outcome-notes"
					label="Notes from the visit"
					error={fieldError('outcome_notes') ?? fieldError('outcome')}
				>
					{#snippet children(control)}
						<Textarea {...control} name="outcome_notes" value={values.outcome_notes} />
					{/snippet}
				</Field>
			</div>

			<!-- CSS shows a ticked section's fields, so this works without JavaScript. -->
			{#if data.sections.growth}
				<div class="group/growth {SECTION}">
					{@render addBox('add_growth', 'Add a growth measurement', values.add.growth)}
					<div class="hidden group-has-[#outcome-add_growth:checked]/growth:block">
						<div class={BLOCK}>
							<MeasurementFields
								prefix="growth."
								values={values.growth}
								error={(name) => fieldError(`growth.${name}`)}
								birthDate={data.kid.birthDate}
								today={data.today}
							/>
						</div>
					</div>
				</div>
			{/if}

			{#if data.sections.vaccinations && data.catalogue}
				{@const catalogue = data.catalogue}
				<div class="group/vaccinations {SECTION}">
					{@render addBox('add_vaccinations', 'Add vaccines given', values.add.vaccinations)}
					<div
						class="hidden flex-col gap-3.5 group-has-[#outcome-add_vaccinations:checked]/vaccinations:flex"
					>
						{#each vaccinations as block, i (i)}
							<fieldset
								id="outcome-vaccine-{i}"
								tabindex="-1"
								class="m-0 flex min-w-0 flex-col gap-3.5 border-0 border-t p-0 pt-3 first-of-type:border-t-0 first-of-type:pt-0"
							>
								<legend class="mb-1 p-0 text-[15px] font-semibold">Vaccine {i + 1}</legend>
								<div class={BLOCK}>
									<DoseFields
										prefix="vaccinations.{i}."
										values={block}
										error={(name) => fieldError(`vaccinations[${i}].${name}`)}
										{catalogue}
										birthDate={data.kid.birthDate}
										today={data.today}
									/>
								</div>
							</fieldset>
						{/each}
						{#if hydrated && vaccinations.length < MAX_OUTCOME_VACCINATIONS}
							<Button
								type="button"
								variant="outline"
								class="self-start px-5"
								onclick={() => addBlock('vaccine')}>Add another vaccine</Button
							>
						{/if}
					</div>
				</div>
			{/if}

			{#if data.sections.medications}
				<div class="group/medications {SECTION}">
					{@render addBox('add_medications', 'Add a medication', values.add.medications)}
					<div
						class="hidden flex-col gap-3.5 group-has-[#outcome-add_medications:checked]/medications:flex"
					>
						{#each medications as block, i (i)}
							<fieldset
								id="outcome-medication-{i}"
								tabindex="-1"
								class="m-0 flex min-w-0 flex-col gap-3.5 border-0 border-t p-0 pt-3 first-of-type:border-t-0 first-of-type:pt-0"
							>
								<legend class="mb-1 p-0 text-[15px] font-semibold">Medication {i + 1}</legend>
								<div class={BLOCK}>
									<CourseFields
										prefix="medications.{i}."
										values={block}
										error={(name) => fieldError(`medications[${i}].${name}`)}
										startHint=""
									/>
								</div>
							</fieldset>
						{/each}
						{#if hydrated && medications.length < MAX_OUTCOME_MEDICATIONS}
							<Button
								type="button"
								variant="outline"
								class="self-start px-5"
								onclick={() => addBlock('medication')}>Add another medication</Button
							>
						{/if}
					</div>
				</div>
			{/if}

			<div class="flex gap-2 xl:max-w-[448px]">
				<Button type="submit" size={embed ? 'lg' : 'default'} class="flex-1">Save</Button>
				<FormCancel href={back} size="default" class="" />
			</div>
		</form>
	{/if}
</FormFrame>
