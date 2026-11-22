<script lang="ts">
	import { resolve } from '$app/paths';
	import type { Kid } from '#lib/types.js';

	let { kid }: { kid: Pick<Kid, 'id' | 'name' | 'role'> } = $props();
</script>

<!-- Settings, access and modules are guardian pages; a caregiver gets none of the links. -->
{#if kid.role === 'guardian'}
	<p class="mt-2 flex gap-4 text-sm">
		<a href={resolve('/(app)/kids/[kid]/settings', { kid: kid.id })} class="underline"
			>Settings<span class="sr-only"> for {kid.name}</span></a
		>
		<a href={resolve('/(app)/kids/[kid]/access', { kid: kid.id })} class="underline"
			>Access<span class="sr-only"> for {kid.name}</span></a
		>
		<a href={resolve('/(app)/kids/[kid]/modules', { kid: kid.id })} class="underline"
			>Modules<span class="sr-only"> for {kid.name}</span></a
		>
	</p>
{/if}
