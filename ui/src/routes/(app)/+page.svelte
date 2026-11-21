<script lang="ts">
	import { resolve } from '$app/paths';
	import KidSwitcher from '#lib/components/KidSwitcher.svelte';
	import KidLinks from '#lib/components/KidLinks.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
</script>

<svelte:head>
	<title>Home · happybabytracker</title>
</svelte:head>

<h1 class="mb-4 text-xl font-semibold">Welcome, {data.session?.parent?.displayName}</h1>

{#if data.cards.length === 0}
	<a
		href={resolve('/(app)/kids/new')}
		class="inline-block rounded-lg border px-4 py-3 font-medium hover:bg-muted"
		>Add your first child</a
	>
{:else}
	<KidSwitcher kids={data.kids} currentId={data.currentId} />
	<ul class="flex flex-col gap-3">
		{#each data.cards as kid (kid.id)}
			<li>
				<article class="rounded-lg border p-4">
					<h2 class="font-semibold">
						<a href={resolve('/(app)/kids/[kid]', { kid: kid.id })} class="hover:underline"
							>{kid.name}</a
						>
					</h2>
					<p class="text-muted-foreground">{kid.age}</p>
					<KidLinks {kid} />
				</article>
			</li>
		{/each}
	</ul>
	<a href={resolve('/(app)/kids/new')} class="mt-4 inline-block underline">Add a child</a>
{/if}
