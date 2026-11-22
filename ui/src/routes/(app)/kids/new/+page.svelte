<script lang="ts">
	import KidForm from '#lib/components/KidForm.svelte';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
</script>

<svelte:head>
	<title>Add a child · happybabytracker</title>
</svelte:head>

<h1 class="mb-4 text-xl font-semibold">Add a child</h1>
<KidForm
	values={form?.values ?? data.values}
	field={form?.field}
	message={form?.message}
	zones={data.zones}
	sources={data.sources}
	schedules={data.schedules}
	detectZone
	submitLabel="Add child"
/>
