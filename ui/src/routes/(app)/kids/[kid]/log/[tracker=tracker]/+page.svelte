<script lang="ts">
	import TrackerForm from '#lib/components/entry/TrackerForm.svelte';
	import { TRACKER_TITLE } from '#lib/constants.js';
	import { advancedToggle } from '#lib/entry-edit.js';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	const values = $derived(form?.values ?? data.values);
	const error = (name: string) => (form?.field === name ? form.message : null);
	const title = $derived(TRACKER_TITLE[data.kind].log);
	const toggle = $derived(advancedToggle(data.kind, values.advanced === '1'));
</script>

<svelte:head>
	<title>{title} · {data.kid.name} · happybabytracker</title>
</svelte:head>

<TrackerForm
	kid={data.kid}
	kind={data.kind}
	stage="new"
	{title}
	age={data.ages[data.kid.id]}
	zone={data.kid.timeZone}
	{values}
	{error}
	{toggle}
	message={form && !form.field ? form.message : null}
	refused={!!form?.field}
	running={!!form && 'running' in form}
/>
