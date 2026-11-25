<script lang="ts">
	import * as Sheet from '#lib/components/ui/sheet/index.js';
	import type { NavEntry } from '#lib/nav.js';

	let {
		open = $bindable(false),
		entries,
		ontheme
	}: { open: boolean; entries: NavEntry[]; ontheme: () => void } = $props();

	const ROW =
		'flex min-h-14 w-full items-center gap-3 rounded-xl px-3 text-left hover:bg-accent aria-[current=page]:text-primary';
</script>

<!-- eslint-disable svelte/no-navigation-without-resolve -- hrefs come from the nav model -->
<Sheet.Root bind:open>
	<Sheet.Content side="bottom">
		<Sheet.Header>
			<Sheet.Title>More</Sheet.Title>
			<Sheet.Description class="sr-only">Other pages and settings.</Sheet.Description>
		</Sheet.Header>
		<ul class="grid gap-1">
			{#each entries as e (e.id)}
				<li>
					{#if e.kind === 'link'}
						<a href={e.href} aria-current={e.current ? 'page' : undefined} class={ROW}>
							<e.icon aria-hidden="true" class="size-5 shrink-0" />
							<span class="flex-1 font-semibold">{e.label}</span>
							<span class="text-[13px] text-muted-foreground">{e.note}</span>
						</a>
					{:else if e.kind === 'theme'}
						<button type="button" class={ROW} onclick={ontheme}>
							<e.icon aria-hidden="true" class="size-5 shrink-0" />
							<span class="flex-1 font-semibold">{e.label}</span>
							<span class="text-[13px] text-muted-foreground">{e.note}</span>
						</button>
					{:else}
						<form method="POST" action="/logout">
							<button type="submit" class={ROW}>
								<e.icon aria-hidden="true" class="size-5 shrink-0" />
								<span class="flex-1 font-semibold">{e.label}</span>
								<span class="text-[13px] text-muted-foreground">{e.note}</span>
							</button>
						</form>
					{/if}
				</li>
			{/each}
		</ul>
	</Sheet.Content>
</Sheet.Root>
