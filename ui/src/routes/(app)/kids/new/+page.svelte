<script lang="ts">
	import { resolve } from '$app/paths';
	import KidForm from '#lib/components/KidForm.svelte';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
</script>

<svelte:head>
	<title>Add a child · happybabytracker</title>
</svelte:head>

<div class="mt-5 flex max-w-[448px] flex-col gap-[18px] xl:mx-auto xl:w-[920px] xl:max-w-none">
	<h1>Add a child</h1>
	<p class="leading-normal">
		You will be a guardian of this child. Corrected age on Home uses the birth date and weeks at
		birth.
	</p>
	<KidForm
		values={form?.values ?? data.values}
		field={form?.field}
		message={form?.message}
		zones={data.zones}
		sources={data.sources}
		schedules={data.schedules}
		detectZone
		submitLabel="Add child"
		cancelHref={resolve('/(app)/kids')}
	/>
</div>
