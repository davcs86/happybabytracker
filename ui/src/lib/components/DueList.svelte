<script lang="ts">
	import NotebookPenIcon from '@lucide/svelte/icons/notebook-pen';
	import SyringeIcon from '@lucide/svelte/icons/syringe';
	import { SECTION_HEADING } from '#lib/components/home.js';
	import type { DueLine } from '#lib/due.js';

	let { lines }: { lines: DueLine[] } = $props();

	// dueLines ends with one "N more on the … page" line per hidden kind; those are plain links.
	const isMore = (line: DueLine) => /^\d+ more on the /.test(line.text);
	const isMedication = (line: DueLine) => line.href.endsWith('/medication');
</script>

<!-- eslint-disable svelte/no-navigation-without-resolve -- the path helpers keep an absolute path without JavaScript -->
{#if lines.length > 0}
	<section aria-labelledby="due-heading" class="flex flex-col gap-2.5">
		<h2 id="due-heading" class={SECTION_HEADING}>Due</h2>
		<ul class="flex flex-col gap-1.5">
			{#each lines as line, i (i)}
				<li>
					{#if isMore(line)}
						<a
							href={line.href}
							class="inline-flex min-h-11 items-center text-sm [overflow-wrap:anywhere] underline"
							>{line.text}</a
						>
					{:else}
						<a
							href={line.href}
							class="flex min-h-14 items-center gap-3 rounded-xl border bg-card px-3 py-2 hover:bg-accent"
						>
							<span
								class="tracker-chip size-9 {isMedication(line) ? 'tracker-symptom' : '[--h:235]'}"
							>
								{#if isMedication(line)}
									<NotebookPenIcon aria-hidden="true" class="size-[18px]" />
								{:else}
									<SyringeIcon aria-hidden="true" class="size-[18px]" />
								{/if}
							</span>
							<span class="min-w-0 flex-1 text-sm font-semibold [overflow-wrap:anywhere]"
								>{line.text}</span
							>
						</a>
					{/if}
				</li>
			{/each}
		</ul>
	</section>
{/if}
