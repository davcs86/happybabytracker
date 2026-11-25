<script lang="ts">
	import { resolve } from '$app/paths';

	let { kidId, visitId }: { kidId: string; visitId: string | null } = $props();
</script>

{#if visitId}
	<a
		href={resolve('/(app)/kids/[kid]/visits/[visit]', { kid: kidId, visit: visitId })}
		class="flex min-h-11 w-fit items-center text-sm underline">Recorded at a visit</a
	>
{/if}
