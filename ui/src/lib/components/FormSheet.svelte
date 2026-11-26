<script lang="ts">
	import * as Sheet from '#lib/components/ui/sheet/index.js';
	import { closeDrawer, provideEmbed } from '#lib/drawer.js';
	import type { Snippet } from 'svelte';

	let { title, children }: { title: string; children: Snippet } = $props();

	provideEmbed();
</script>

<Sheet.Root
	open
	onOpenChange={(o) => {
		if (!o) closeDrawer();
	}}
>
	<Sheet.Content
		side="bottom"
		onOpenAutoFocus={(e) => {
			const target = location.hash && document.getElementById(location.hash.slice(1));
			if (!target) return;
			e.preventDefault();
			target.focus();
		}}
	>
		<Sheet.Header>
			<Sheet.Title>{title}</Sheet.Title>
			<Sheet.Description class="sr-only">Fill in the form, then save.</Sheet.Description>
		</Sheet.Header>
		{@render children()}
	</Sheet.Content>
</Sheet.Root>
