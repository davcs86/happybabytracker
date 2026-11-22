<script lang="ts">
	import { onMount } from 'svelte';
	import { enhance } from '$app/forms';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import { ERROR_CLASS, SELECT_CLASS, keepValues } from '#lib/components/field.js';
	import {
		GESTATION_DAYS,
		GESTATION_WEEKS,
		KID_MIN_BIRTH_DATE,
		KID_NAME_MAX_LEN
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
	};

	let {
		values,
		field = null,
		message = null,
		zones,
		sources,
		detectZone = false,
		submitLabel,
		action
	}: {
		values: Values;
		field?: string | null;
		message?: string | null;
		zones: string[];
		sources: { id: string; name: string; hint: string }[];
		detectZone?: boolean;
		submitLabel: string;
		action?: string;
	} = $props();

	let zone = $state<string | null>(null);
	let deviceZone = $state<string | null>(null);
	const options = $derived(deviceZone ? withZone(zones, deviceZone) : zones);
	let touched = false;

	const err = (f: string) => (field === f ? message : null);
	const describedBy = (f: string) => (err(f) ? `kid-${f}-error` : undefined);
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

<form
	method="POST"
	{action}
	class="flex max-w-md flex-col gap-4"
	novalidate
	use:enhance={keepValues}
>
	{#if message && !field}
		<p class={ERROR_CLASS} role="alert">{message}</p>
	{/if}
	<div class="flex flex-col gap-2">
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
		{#if err('name')}<p id="kid-name-error" class={ERROR_CLASS} role="alert">{err('name')}</p>{/if}
	</div>
	<div class="flex flex-col gap-2">
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
			aria-describedby={describedBy('birth_date')}
		/>
		{#if err('birth_date')}
			<p id="kid-birth_date-error" class={ERROR_CLASS} role="alert">{err('birth_date')}</p>
		{/if}
	</div>
	<fieldset
		role="radiogroup"
		aria-labelledby="kid-sex-legend"
		class="flex flex-col gap-2"
		aria-invalid={invalid('sex')}
		aria-describedby={describedBy('sex')}
	>
		<legend id="kid-sex-legend" class="mb-2 text-sm font-medium">Sex</legend>
		<div class="flex gap-4">
			{#each [['female', 'Female'], ['male', 'Male']] as [value, label] (value)}
				<label class="flex items-center gap-2 text-sm">
					<input type="radio" name="sex" {value} checked={values.sex === value} required />
					{label}
				</label>
			{/each}
		</div>
		{#if err('sex')}<p id="kid-sex-error" class={ERROR_CLASS} role="alert">{err('sex')}</p>{/if}
	</fieldset>
	<!-- CSS reveals the gestation fields from the box, so this works without JavaScript. -->
	<div class="group flex flex-col gap-4">
		<label class="flex items-center gap-2 text-sm font-medium">
			<input id="kid-born-early" type="checkbox" name="born_early" checked={values.bornEarly} />
			Born early?
		</label>
		<div class="hidden grid-cols-2 gap-4 group-has-[#kid-born-early:checked]:grid">
			<div class="flex flex-col gap-2">
				<Label for="kid-gestation_weeks">Weeks</Label>
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
					<p id="kid-gestation_weeks-error" class={ERROR_CLASS} role="alert">
						{err('gestation_weeks')}
					</p>
				{/if}
			</div>
			<div class="flex flex-col gap-2">
				<Label for="kid-gestation_days">Days</Label>
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
					<p id="kid-gestation_days-error" class={ERROR_CLASS} role="alert">
						{err('gestation_days')}
					</p>
				{/if}
			</div>
		</div>
	</div>
	<div class="flex flex-col gap-2">
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
			aria-describedby={describedBy('time_zone')}
		>
			{#each options as z (z)}
				<option value={z}>{z}</option>
			{/each}
		</select>
		{#if err('time_zone')}
			<p id="kid-time_zone-error" class={ERROR_CLASS} role="alert">{err('time_zone')}</p>
		{/if}
	</div>
	<div class="flex flex-col gap-2">
		<Label for="kid-growth_source">Growth chart source</Label>
		<select
			id="kid-growth_source"
			name="growth_source"
			class={SELECT_CLASS}
			value={values.growthSource}
			aria-invalid={invalid('growth_source')}
			aria-describedby={['kid-growth_source-hint', describedBy('growth_source')]
				.filter(Boolean)
				.join(' ')}
		>
			{#each sources as source (source.id)}
				<option value={source.id}>{source.name}</option>
			{/each}
		</select>
		<ul id="kid-growth_source-hint" class="flex flex-col gap-1 text-sm text-muted-foreground">
			{#each sources as source (source.id)}
				<li>{source.name}: {source.hint}</li>
			{/each}
		</ul>
		{#if err('growth_source')}
			<p id="kid-growth_source-error" class={ERROR_CLASS} role="alert">{err('growth_source')}</p>
		{/if}
	</div>
	<Button type="submit" class="self-start">{submitLabel}</Button>
</form>
