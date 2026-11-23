<script lang="ts">
	import type { DueLine } from '#lib/due.js';

	let { lines }: { lines: DueLine[] } = $props();
</script>

{#if lines.length > 0}
	<section aria-labelledby="due-heading">
		<h2 id="due-heading" class="mb-2 font-semibold">Due</h2>
		<ul class="flex flex-col gap-2">
			{#each lines as line, i (i)}
				<li>
					<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- the path helpers keep an absolute path without JavaScript -->
					<a href={line.href} class="[overflow-wrap:anywhere] underline">{line.text}</a>
				</li>
			{/each}
		</ul>
	</section>
{/if}
