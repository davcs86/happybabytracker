<script lang="ts">
	import { resolve } from '$app/paths';
	import type { Kid } from '#lib/types.js';

	let { kids, currentId }: { kids: Pick<Kid, 'id' | 'name'>[]; currentId?: string } = $props();
</script>

{#if kids.length > 0}
	<nav aria-label="Children" class="mb-4">
		<ul class="flex flex-wrap gap-2">
			{#each kids as kid (kid.id)}
				<li>
					<a
						href={resolve('/(app)/kids/[kid]', { kid: kid.id })}
						aria-current={kid.id === currentId ? 'true' : undefined}
						class="inline-block rounded-full border px-3 py-1 text-sm {kid.id === currentId
							? 'border-foreground bg-foreground text-background'
							: 'hover:bg-muted'}">{kid.name}</a
					>
				</li>
			{/each}
		</ul>
	</nav>
{/if}
