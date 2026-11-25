<script lang="ts">
	import { Tabs as TabsPrimitive } from 'bits-ui';
	import { cn } from '#lib/utils.js';

	let {
		ref = $bindable(null),
		class: className,
		...restProps
	}: TabsPrimitive.TriggerProps = $props();
</script>

<TabsPrimitive.Trigger
	bind:ref
	data-slot="tabs-trigger"
	class={cn(
		'inline-flex min-h-11 flex-1 items-center justify-center rounded-[9px] border-0 px-3 text-sm font-semibold whitespace-nowrap text-muted-foreground data-active:bg-card data-active:text-foreground data-active:shadow-[0_1px_3px_rgb(0_0_0/0.15)]',
		className
	)}
	{...restProps}
/>
