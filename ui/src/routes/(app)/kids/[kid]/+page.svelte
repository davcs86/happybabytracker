<script lang="ts">
	import KidLinks from '#lib/components/KidLinks.svelte';
	import KidSwitcher from '#lib/components/KidSwitcher.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
</script>

<svelte:head>
	<title>{data.kid.name} · happybabytracker</title>
</svelte:head>

<KidSwitcher kids={data.kids} currentId={data.kid.id} />
<h1 class="text-xl font-semibold">{data.kid.name}</h1>
<p class="text-muted-foreground">{data.age}</p>
<p class="text-sm text-muted-foreground">Born {data.kid.birthDate}</p>
<KidLinks kid={data.kid} />
