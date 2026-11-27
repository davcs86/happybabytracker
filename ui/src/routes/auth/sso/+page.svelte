<script lang="ts">
	import SsoOutcome from '#lib/components/SsoOutcome.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
</script>

<SsoOutcome heading="Single sign-on is unavailable" back={data.back} />
