<script lang="ts">
	import SsoOutcome from '#lib/components/SsoOutcome.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
</script>

<svelte:head>
	{#if data.outcome === 'continue'}
		<meta http-equiv="refresh" content="0;url={data.to}" />
	{/if}
</svelte:head>

{#if data.outcome === 'continue'}
	<SsoOutcome heading="Signing you in" back={{ href: data.to, label: 'Continue' }} />
{:else if data.outcome === 'failed'}
	<SsoOutcome heading="Single sign-on did not complete" back={data.back} />
{/if}
