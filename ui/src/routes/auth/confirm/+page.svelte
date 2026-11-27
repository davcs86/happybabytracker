<script lang="ts">
	import SsoOutcome from '#lib/components/SsoOutcome.svelte';
	import type { PageProps } from './$types';

	let { form }: PageProps = $props();
</script>

{#if form?.failed}
	<SsoOutcome heading="Single sign-on did not complete" message={form.message} back={form.back} />
{:else}
	<SsoOutcome heading="Nothing to confirm" back={{ href: '/login', label: 'Go to sign in' }} />
{/if}
