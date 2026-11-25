<script lang="ts">
	import { Button } from '#lib/components/ui/button/index.js';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();
</script>

<main class="mx-auto max-w-[720px] px-4 pt-4 pb-7">
	{@render children()}
	<form method="POST" action="/logout" class="mt-6">
		<Button type="submit" variant="outline">Sign out</Button>
	</form>
</main>
