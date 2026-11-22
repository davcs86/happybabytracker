<script lang="ts">
	import { resolve } from '$app/paths';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
</script>

<svelte:head>
	<title>Home · happybabytracker</title>
</svelte:head>

<h1 class="mb-4 text-xl font-semibold">Welcome, {data.session?.parent?.displayName}</h1>

<a
	href={resolve('/(app)/kids/new')}
	class="inline-block rounded-lg border px-4 py-3 font-medium hover:bg-muted"
	>Add your first child</a
>
