<script lang="ts">
	import { enhance } from '$app/forms';
	import { ERROR_CLASS } from '#lib/components/field.js';
	import { logPath } from '#lib/paths.js';
	import type { QuickLogEntry } from '#lib/quick-log.js';

	let {
		entries,
		kidId,
		message = null
	}: { entries: QuickLogEntry[]; kidId: string; message?: string | null } = $props();

	const TILE =
		'flex min-h-14 w-full items-center justify-center rounded-lg border px-4 py-3 text-center font-medium hover:bg-muted';
</script>

{#if entries.length > 0}
	<section aria-labelledby="quick-log-heading">
		<h2 id="quick-log-heading" class="mb-2 font-semibold">Quick log</h2>
		{#if message}<p class="{ERROR_CLASS} mb-2" role="alert">{message}</p>{/if}
		<ul class="grid grid-cols-2 gap-2">
			{#each entries as entry (entry.module)}
				<li>
					<!-- Sleep is one tap, a form post that works without JavaScript; the rest open their form. -->
					{#if entry.kind === 'sleep'}
						<form method="POST" action="?/startSleep" use:enhance>
							<button type="submit" class={TILE}>{entry.label}</button>
						</form>
					{:else}
						<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- the log pages are not typed routes yet -->
						<a href={logPath(kidId, entry.kind)} class={TILE}>{entry.label}</a>
					{/if}
				</li>
			{/each}
		</ul>
	</section>
{/if}
