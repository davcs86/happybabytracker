<script lang="ts">
	import type { TodayRow } from '#lib/today.js';

	let { rows }: { rows: TodayRow[] } = $props();
</script>

{#if rows.length > 0}
	<section aria-labelledby="today-heading">
		<h2 id="today-heading" class="mb-2 font-semibold">Today</h2>
		<dl class="rounded-lg border px-3">
			{#each rows as row (row.label)}
				<div class="flex justify-between gap-4 border-b py-2 last:border-b-0">
					<dt>{row.label}</dt>
					<dd class="text-right">{row.value}</dd>
				</div>
			{/each}
		</dl>
	</section>
{/if}
