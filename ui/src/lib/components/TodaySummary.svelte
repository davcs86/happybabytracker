<script lang="ts">
	import { SECTION_HEADING } from '#lib/components/home.js';
	import { formatAgo } from '#lib/format.js';
	import type { TodayRow } from '#lib/today.js';

	let { rows, now }: { rows: TodayRow[]; now: Date } = $props();
</script>

{#if rows.length > 0}
	<section aria-labelledby="today-heading" class="flex flex-col gap-2.5">
		<h2 id="today-heading" class={SECTION_HEADING}>Today</h2>
		<dl class="grid grid-cols-2 overflow-hidden rounded-xl border bg-card">
			{#each rows as row (row.label)}
				<div class="-mr-px -mb-px border-r border-b p-3">
					<dt class="text-xs text-muted-foreground">{row.label}</dt>
					<dd class="mt-0.5 text-lg font-semibold [overflow-wrap:anywhere]">
						{row.value}
						{#if row.lastAt}
							<span class="block text-xs font-normal text-muted-foreground"
								>Last: {formatAgo(row.lastAt, now)}</span
							>
						{/if}
					</dd>
				</div>
			{/each}
		</dl>
	</section>
{/if}
