<script lang="ts">
	import { resolve } from '$app/paths';
	import FormFrame from '#lib/components/FormFrame.svelte';
	import KidForm from '#lib/components/KidForm.svelte';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
</script>

<FormFrame class="mt-5 flex flex-col gap-[18px]" documentTitle="Add a child · happybabytracker">
	{#snippet chrome()}
		<h1>Add a child</h1>
	{/snippet}
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
		saved="Child added"
		follow
	/>
</FormFrame>
