<script lang="ts">
	import { onMount } from 'svelte';
	import { enhance } from '$app/forms';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import { ERROR_CLASS, SELECT_CLASS, keepValues } from '#lib/components/field.js';
	import {
		GESTATION_DAYS,
		GESTATION_WEEKS,
		KID_MIN_BIRTH_DATE,
		KID_NAME_MAX_LEN,
		MAX_REMINDER_OVERDUE_MINUTES,
		MIN_REMINDER_OVERDUE_MINUTES
	} from '#lib/constants.js';
	import { todayIn } from '#lib/format.js';
	import { acceptsZone, withZone } from '#lib/time-zone.js';

	type Values = {
		name: string;
		birthDate: string;
		sex: string;
		bornEarly: boolean;
		weeks: string;
		days: string;
		timeZone: string;
		growthSource: string;
		vaccineSchema: string;
		reminderOverdue: string;
		reminderOverdueOriginal: string;
	};

	let {
		values,
		field = null,
		message = null,
		zones,
		sources,
		schedules,
		detectZone = false,
		mode = 'create',
		submitLabel,
		cancelHref,
		action
	}: {
		values: Values;
		field?: string | null;
		message?: string | null;
		zones: string[];
		sources: { id: string; name: string; hint: string }[];
		schedules: { id: string; label: string }[];
		detectZone?: boolean;
		mode?: 'create' | 'settings';
		submitLabel: string;
		cancelHref: string;
		action?: string;
	} = $props();

	const HINT = 'text-[12.5px] leading-normal text-muted-foreground';
	const CHOICE = 'flex min-h-11 items-center gap-3 text-[15px]';
	const NATIVE = 'size-5 shrink-0 accent-primary';

	let zone = $state<string | null>(null);
	let deviceZone = $state<string | null>(null);
	const options = $derived(deviceZone ? withZone(zones, deviceZone) : zones);
	let touched = false;
	// A stored id the API no longer ships stays selected, so a save does not change it silently.
	const retired = $derived(
		values.vaccineSchema !== '' && !schedules.some((s) => s.id === values.vaccineSchema)
	);

	const err = (f: string) => (field === f ? message : null);
	const describedBy = (f: string, hint = false) =>
		[hint ? `kid-${f}-hint` : null, err(f) ? `kid-${f}-error` : null].filter(Boolean).join(' ') ||
		undefined;
	const invalid = (f: string) => (err(f) ? 'true' : undefined);
	const maxDate = $derived.by(() => {
		try {
			return todayIn(zone ?? values.timeZone);
		} catch {
			return undefined;
		}
	});

	onMount(() => {
		const device = Intl.DateTimeFormat().resolvedOptions().timeZone;
		if (!detectZone || touched || !acceptsZone(device)) return;
		deviceZone = device;
		zone = device;
	});
</script>

{#if field}
	<Alert.Root>Check the fields with a message.</Alert.Root>
{:else if message}
	<Alert.Root>{message}</Alert.Root>
{/if}
<form
	method="POST"
	{action}
	class="flex flex-col gap-4 xl:grid xl:grid-cols-[448px_448px] xl:grid-rows-[auto_1fr] xl:items-start xl:gap-x-6 xl:gap-y-[18px]"
	novalidate
	use:enhance={keepValues}
>
	<div class="flex flex-col gap-4">
		<div class="grid gap-1.5">
			<Label for="kid-name">Name</Label>
			<Input
				id="kid-name"
				name="name"
				value={values.name}
				maxlength={KID_NAME_MAX_LEN}
				autocomplete="off"
				required
				aria-invalid={invalid('name')}
				aria-describedby={describedBy('name')}
			/>
			{#if err('name')}<p id="kid-name-error" class={ERROR_CLASS}>{err('name')}</p>{/if}
		</div>
		<div class="grid gap-1.5">
			<Label for="kid-birth_date">Birth date</Label>
			<Input
				id="kid-birth_date"
				name="birth_date"
				type="date"
				value={values.birthDate}
				min={KID_MIN_BIRTH_DATE}
				max={maxDate}
				required
				aria-invalid={invalid('birth_date')}
				aria-describedby={describedBy('birth_date', true)}
			/>
			<p id="kid-birth_date-hint" class={HINT}>Year-month-day</p>
			{#if err('birth_date')}
				<p id="kid-birth_date-error" class={ERROR_CLASS}>{err('birth_date')}</p>
			{/if}
		</div>
		<div class="grid gap-1.5">
			<Label for="kid-time_zone">Time zone</Label>
			<select
				id="kid-time_zone"
				name="time_zone"
				class={SELECT_CLASS}
				value={zone ?? values.timeZone}
				onchange={(e) => {
					touched = true;
					zone = e.currentTarget.value;
				}}
				aria-invalid={invalid('time_zone')}
				aria-describedby={describedBy('time_zone', true)}
			>
				{#each options as z (z)}
					<option value={z}>{z}</option>
				{/each}
			</select>
			<p id="kid-time_zone-hint" class={HINT}>
				Times are shown in this time zone. Changing it moves how past entries are grouped by day; it
				does not change when they happened.
			</p>
			{#if err('time_zone')}
				<p id="kid-time_zone-error" class={ERROR_CLASS}>{err('time_zone')}</p>
			{/if}
		</div>
	</div>
	<div class="flex flex-col gap-4 xl:row-span-2">
		<fieldset
			role="radiogroup"
			aria-labelledby="kid-sex-legend"
			class="flex flex-col"
			aria-invalid={invalid('sex')}
			aria-describedby={describedBy('sex')}
		>
			<legend id="kid-sex-legend" class="mb-1 p-0 text-[13px] text-muted-foreground">Sex</legend>
			{#each [['female', 'Female'], ['male', 'Male']] as [value, label] (value)}
				<label class={CHOICE}>
					<input
						type="radio"
						name="sex"
						{value}
						checked={values.sex === value}
						required
						class={NATIVE}
					/>
					{label}
				</label>
			{/each}
			{#if err('sex')}<p id="kid-sex-error" class="{ERROR_CLASS} mt-1">{err('sex')}</p>{/if}
		</fieldset>
		<!-- CSS reveals the gestation fields from the box, so this works without JavaScript. -->
		<div class="group flex flex-col gap-4">
			<label class={CHOICE}>
				<input
					id="kid-born-early"
					type="checkbox"
					name="born_early"
					checked={values.bornEarly}
					class={NATIVE}
				/>
				Born early?
			</label>
			<div class="hidden flex-col gap-4 group-has-[#kid-born-early:checked]:flex">
				<div class="grid gap-1.5">
					<Label for="kid-gestation_weeks">Weeks at birth</Label>
					<Input
						id="kid-gestation_weeks"
						name="gestation_weeks"
						type="number"
						inputmode="numeric"
						min={GESTATION_WEEKS.min}
						max={GESTATION_WEEKS.max}
						value={values.weeks}
						aria-invalid={invalid('gestation_weeks')}
						aria-describedby={describedBy('gestation_weeks')}
					/>
					{#if err('gestation_weeks')}
						<p id="kid-gestation_weeks-error" class={ERROR_CLASS}>{err('gestation_weeks')}</p>
					{/if}
				</div>
				<div class="grid gap-1.5">
					<Label for="kid-gestation_days">
						Extra days ({GESTATION_DAYS.min} to {GESTATION_DAYS.max})
					</Label>
					<Input
						id="kid-gestation_days"
						name="gestation_days"
						type="number"
						inputmode="numeric"
						min={GESTATION_DAYS.min}
						max={GESTATION_DAYS.max}
						value={values.days}
						aria-invalid={invalid('gestation_days')}
						aria-describedby={describedBy('gestation_days')}
					/>
					{#if err('gestation_days')}
						<p id="kid-gestation_days-error" class={ERROR_CLASS}>{err('gestation_days')}</p>
					{/if}
				</div>
			</div>
		</div>
		<div class="grid gap-1.5">
			<Label for="kid-growth_source">Growth chart source</Label>
			<select
				id="kid-growth_source"
				name="growth_source"
				class={SELECT_CLASS}
				value={values.growthSource}
				aria-invalid={invalid('growth_source')}
				aria-describedby={describedBy('growth_source', true)}
			>
				{#each sources as source (source.id)}
					<option value={source.id}>{source.name}</option>
				{/each}
			</select>
			<ul id="kid-growth_source-hint" class="{HINT} flex flex-col gap-1">
				{#each sources as source (source.id)}
					<li>{source.name}: {source.hint}</li>
				{/each}
			</ul>
			{#if err('growth_source')}
				<p id="kid-growth_source-error" class={ERROR_CLASS}>{err('growth_source')}</p>
			{/if}
		</div>
		<div class="grid gap-1.5">
			<Label for="kid-vaccine_schema">Vaccination schedule</Label>
			<select
				id="kid-vaccine_schema"
				name="vaccine_schema"
				class={SELECT_CLASS}
				value={values.vaccineSchema}
				aria-invalid={invalid('vaccine_schema')}
				aria-describedby={describedBy('vaccine_schema')}
			>
				{#if retired}
					<option value={values.vaccineSchema}>Unavailable schedule ({values.vaccineSchema})</option
					>
				{/if}
				<option value="">None yet</option>
				{#each schedules as schedule (schedule.id)}
					<option value={schedule.id}>{schedule.label}</option>
				{/each}
			</select>
			{#if err('vaccine_schema')}
				<p id="kid-vaccine_schema-error" class={ERROR_CLASS}>{err('vaccine_schema')}</p>
			{/if}
		</div>
		{#if mode === 'settings'}
			<div class="grid gap-1.5">
				<Label for="kid-reminder_overdue_minutes">Show a missed dose as overdue for (minutes)</Label
				>
				<Input
					id="kid-reminder_overdue_minutes"
					name="reminder_overdue_minutes"
					type="number"
					inputmode="numeric"
					min={MIN_REMINDER_OVERDUE_MINUTES}
					max={MAX_REMINDER_OVERDUE_MINUTES}
					step="1"
					value={values.reminderOverdue}
					aria-invalid={invalid('reminder_overdue_minutes')}
					aria-describedby={describedBy('reminder_overdue_minutes', true)}
				/>
				<input
					type="hidden"
					name="reminder_overdue_minutes_original"
					value={values.reminderOverdueOriginal}
				/>
				<p id="kid-reminder_overdue_minutes-hint" class={HINT}>
					From {MIN_REMINDER_OVERDUE_MINUTES} to {MAX_REMINDER_OVERDUE_MINUTES} ({MAX_REMINDER_OVERDUE_MINUTES /
						60} hours). A missed dose leaves the home list after this time, or when the next dose of that
					course is due, whichever comes first.
				</p>
				{#if err('reminder_overdue_minutes')}
					<p id="kid-reminder_overdue_minutes-error" class={ERROR_CLASS}>
						{err('reminder_overdue_minutes')}
					</p>
				{/if}
			</div>
		{/if}
	</div>
	<div class="flex gap-2">
		<Button type="submit" class="flex-1">{submitLabel}</Button>
		<Button href={cancelHref} variant="outline">Cancel</Button>
	</div>
</form>
