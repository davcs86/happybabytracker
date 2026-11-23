<script lang="ts">
	import { resolve } from '$app/paths';
	import TrackerFields from '#lib/components/entry/TrackerFields.svelte';
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

<p class="mb-2 text-sm">
	<a href={resolve('/(app)/kids/[kid]', { kid: data.kid.id })} class="underline">{data.kid.name}</a>
</p>
<h1 class="mb-4 text-xl font-semibold">{title}</h1>
<TrackerForm
	kid={data.kid}
	zone={data.kid.timeZone}
	{values}
	{error}
	{toggle}
	message={form && !form.field ? form.message : null}
	running={!!form && 'running' in form}
>
	<TrackerFields kind={data.kind} stage="new" {values} {error} />
</TrackerForm>
