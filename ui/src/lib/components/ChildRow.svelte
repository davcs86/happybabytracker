<script lang="ts">
	import CheckIcon from '@lucide/svelte/icons/check';

	let { href, name, age, current }: { href: string; name: string; age: string; current: boolean } =
		$props();
</script>

<!-- eslint-disable svelte/no-navigation-without-resolve -- the caller builds it from a child id -->
<a
	{href}
	aria-current={current ? 'true' : undefined}
	class="flex min-h-16 items-center gap-3 rounded-xl border bg-card px-3 py-2 {current
		? 'border-primary bg-accent'
		: ''}"
>
	<span
		aria-hidden="true"
		class="grid size-9 shrink-0 place-items-center rounded-full bg-accent font-semibold text-primary-ink"
		>{[...name][0]?.toUpperCase()}</span
	>
	<span class="min-w-0 flex-1">
		<span class="block font-semibold [overflow-wrap:anywhere]">{name}</span>
		<span class="block text-[13px] text-muted-foreground">{age}</span>
	</span>
	{#if current}<CheckIcon aria-hidden="true" class="size-5 text-primary" />{/if}
</a>
