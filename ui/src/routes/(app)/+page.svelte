<script lang="ts">
	import { resolve } from '$app/paths';
	import { Button } from '#lib/components/ui/button/index.js';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
</script>

<svelte:head>
	<title>Home · happybabytracker</title>
</svelte:head>

<div class="flex max-w-[448px] flex-col gap-3.5 pt-10">
	<h1>Welcome, {data.session?.parent?.displayName}</h1>
	<p class="text-[14.5px] leading-normal text-muted-foreground">
		Add a child to start logging feeds, sleep and diapers. Everyone you invite sees the same
		entries.
	</p>
	<Button href={resolve('/(app)/kids/new')} data-drawer="Add a child" size="lg"
		>Add your first child</Button
	>
</div>
