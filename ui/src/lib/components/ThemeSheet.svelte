<script lang="ts">
	import { Button } from '#lib/components/ui/button/index.js';
	import * as RadioGroup from '#lib/components/ui/radio-group/index.js';
	import * as Sheet from '#lib/components/ui/sheet/index.js';
	import { THEMES, type Theme } from '#lib/constants.js';
	import { applyTheme } from '#lib/theme-client.js';

	let {
		open = $bindable(false),
		theme,
		onchoose
	}: { open: boolean; theme: Theme; onchoose: (t: Theme) => void } = $props();

	const OPTIONS: Record<Theme, { label: string; hint: string; swatch: string }> = {
		light: { label: 'Light', hint: 'Bright background.', swatch: 'bg-[#fbf7f2]' },
		dark: { label: 'Dark', hint: 'Dimmed background, easier at night.', swatch: 'bg-[#2a2420]' },
		auto: {
			label: 'Auto',
			hint: 'Follows your device setting.',
			swatch: 'bg-[linear-gradient(135deg,#fbf7f2_50%,#2a2420_50%)]'
		}
	};
</script>

<Sheet.Root bind:open>
	<Sheet.Content side="bottom">
		<Sheet.Header>
			<Sheet.Title>Change theme</Sheet.Title>
			<Sheet.Description class="sr-only">Choose how the app looks.</Sheet.Description>
		</Sheet.Header>
		<RadioGroup.Root
			value={theme}
			aria-label="Theme"
			onValueChange={(v) => {
				onchoose(v as Theme);
				applyTheme(v as Theme);
			}}
		>
			{#each THEMES as t (t)}
				<label
					class="flex min-h-15 cursor-pointer items-center gap-3 rounded-xl border bg-card px-3 py-2 {theme ===
					t
						? 'border-primary bg-accent'
						: ''}"
				>
					<RadioGroup.Item value={t} aria-label={OPTIONS[t].label} />
					<span class="min-w-0 flex-1">
						<span class="block font-semibold">{OPTIONS[t].label}</span>
						<span class="block text-[13px] leading-[1.4] text-muted-foreground"
							>{OPTIONS[t].hint}</span
						>
					</span>
					<span
						aria-hidden="true"
						class="size-8 shrink-0 rounded-sm border border-[#8a8178] {OPTIONS[t].swatch}"
					></span>
				</label>
			{/each}
		</RadioGroup.Root>
		<Sheet.Footer>
			<Button size="lg" onclick={() => (open = false)}>Done</Button>
		</Sheet.Footer>
	</Sheet.Content>
</Sheet.Root>
