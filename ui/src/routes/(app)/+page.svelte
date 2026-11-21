<script lang="ts">
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
</script>

<svelte:head>
	<title>Home · happybabytracker</title>
</svelte:head>

<h1 class="text-xl font-semibold">Welcome, {data.session?.parent?.displayName}</h1>
