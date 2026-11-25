<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import FormCancel from '#lib/components/FormCancel.svelte';
	import FormFrame from '#lib/components/FormFrame.svelte';
	import MeasurementFields from '#lib/components/growth/MeasurementFields.svelte';
	import { Alert } from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { isEmbed, savedAs } from '#lib/drawer.js';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	const values = $derived(form?.values ?? data.values);
	const error = (name: string) => (form?.field === name ? form.message : null);
	const embed = isEmbed();
	const growth = $derived(resolve('/(app)/kids/[kid]/growth', { kid: data.kid.id }));
</script>

<FormFrame
	title="Add a measurement"
	documentTitle="Add a measurement · {data.kid.name} · happybabytracker"
	backHref={growth}
	backLabel="Growth for {data.kid.name}"
>
	<form
		method="POST"
		action="?/save"
		class="flex flex-col gap-[18px] {embed
			? ''
			: 'mt-[18px] xl:grid xl:grid-cols-[448px_448px] xl:grid-rows-[auto_1fr] xl:gap-x-6'}"
		novalidate
		use:enhance={savedAs('Measurement added')}
	>
		{#if form?.field}
			<Alert class="xl:col-span-2">Check the fields with a message.</Alert>
		{:else if form?.message}
			<Alert class="xl:col-span-2">{form.message}</Alert>
		{/if}
		<MeasurementFields {values} {error} birthDate={data.kid.birthDate} today={data.today} />
		<div class="flex gap-2">
			<Button type="submit" size="lg" class="flex-1">Save</Button>
			<FormCancel href={growth} />
		</div>
	</form>
</FormFrame>
