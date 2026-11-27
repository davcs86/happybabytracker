<script lang="ts" module>
	import type { Nursing, Sleep } from '#lib/types.js';

	export type Timer = { kind: 'sleep'; entry: Sleep } | { kind: 'nursing'; entry: Nursing };
</script>

<script lang="ts">
	import HeartIcon from '@lucide/svelte/icons/heart';
	import MoonIcon from '@lucide/svelte/icons/moon';
	import { enhance } from '$app/forms';
	import { invalidateAll } from '$app/navigation';
	import { ERROR_CLASS } from '#lib/components/field.js';
	import { Alert } from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { MAX_SESSION_HOURS, TRACKER_TITLE } from '#lib/constants.js';
	import { formatElapsed } from '#lib/format.js';
	import { entryPath } from '#lib/paths.js';
	import { formatClock } from '#lib/time-zone.js';

	let {
		timer,
		zone,
		kidId,
		now,
		actions = '',
		message = null,
		overrun = false
	}: {
		timer: Timer;
		zone: string;
		kidId: string;
		now: Date;
		// The path the End and Switch side forms post to: this page by default, else the kid's home.
		actions?: string;
		message?: string | null;
		overrun?: boolean;
	} = $props();

	// A form that posts to another page's action stays on this page: the result is read here, not followed.
	let failure = $state<{ message: string | null; overrun: boolean } | null>(null);
	const submit = () =>
		actions === ''
			? undefined
			: async ({
					result,
					update
				}: {
					result: { type: string; data?: Record<string, unknown> };
					update: () => Promise<void>;
				}) => {
					failure = null;
					if (result.type === 'success') await invalidateAll();
					else if (result.type === 'failure') {
						failure = {
							message: typeof result.data?.message === 'string' ? result.data.message : null,
							overrun: result.data?.overrun === true
						};
					} else await update();
				};
	const shownOverrun = $derived(overrun || !!failure?.overrun);
	const shownMessage = $derived(message ?? failure?.message ?? null);

	const other = (side: 'left' | 'right') => (side === 'left' ? 'right' : 'left');

	// The stored seconds exclude the side being fed now: that one runs from sideSince.
	const sides = $derived.by(() => {
		if (timer.kind !== 'nursing') return null;
		const e = timer.entry;
		const running = e.side
			? Math.max(0, Math.floor((now.getTime() - Date.parse(e.sideSince ?? e.startedAt)) / 1000))
			: 0;
		return {
			left: e.leftSeconds + (e.side === 'left' ? running : 0),
			right: e.rightSeconds + (e.side === 'right' ? running : 0)
		};
	});
	const elapsed = $derived(
		sides
			? sides.left + sides.right
			: Math.floor((now.getTime() - Date.parse(timer.entry.startedAt)) / 1000)
	);
</script>

<li class="flex flex-col gap-3 rounded-2xl border-[1.5px] border-primary bg-card p-3.5">
	<div class="flex items-center gap-3">
		<span
			class="tracker-chip size-11 {timer.kind === 'sleep' ? 'tracker-sleep' : 'tracker-nursing'}"
		>
			{#if timer.kind === 'sleep'}
				<MoonIcon aria-hidden="true" class="size-5" />
			{:else}
				<HeartIcon aria-hidden="true" class="size-5" />
			{/if}
		</span>
		<div class="min-w-0 flex-1">
			{#if timer.kind === 'sleep'}
				<p class="text-[15px] font-semibold">
					Asleep since {formatClock(timer.entry.startedAt, zone)}
				</p>
			{:else}
				<p class="text-[15px] font-semibold">
					Nursing, {timer.entry.side} since {formatClock(
						timer.entry.sideSince ?? timer.entry.startedAt,
						zone
					)}
				</p>
				<p class="text-[12.5px] text-muted-foreground">
					Started {formatClock(timer.entry.startedAt, zone)}
				</p>
			{/if}
		</div>
		<p class="font-mono text-2xl font-medium tracking-tight">{formatElapsed(elapsed)}</p>
	</div>
	{#if sides && timer.kind === 'nursing'}
		<div class="grid grid-cols-2 gap-2">
			{#each [{ label: 'Left', side: 'left', seconds: sides.left }, { label: 'Right', side: 'right', seconds: sides.right }] as tile (tile.side)}
				{@const active = timer.entry.side === tile.side}
				<div class="rounded-xl border bg-background px-2.5 py-2 {active ? 'border-primary' : ''}">
					<p class="text-xs text-muted-foreground">{tile.label}</p>
					<p class="font-mono text-[17px] font-medium">{formatElapsed(tile.seconds)}</p>
					<p class="min-h-3.5 text-xs font-semibold text-muted-foreground">
						{active ? 'Feeding now' : tile.seconds > 0 ? 'Last side' : ''}
					</p>
				</div>
			{/each}
		</div>
	{/if}
	{#if shownOverrun}
		<Alert>
			<p>
				This timer ran over {MAX_SESSION_HOURS} hours:
				<!-- eslint-disable svelte/no-navigation-without-resolve -- resolve() renders a relative href without JavaScript, and the link keeps an absolute path -->
				<a
					href={entryPath(kidId, timer.kind, timer.entry.id)}
					data-drawer={TRACKER_TITLE[timer.kind].edit}
					class="underline">set the end time</a
				>
				<!-- eslint-enable svelte/no-navigation-without-resolve -->
			</p>
		</Alert>
	{:else if shownMessage}
		<p class={ERROR_CLASS} role="alert">{shownMessage}</p>
	{/if}
	<div class="flex gap-2">
		{#if timer.kind === 'nursing' && timer.entry.side}
			<form method="POST" action="{actions}?/switchSide" class="flex flex-1" use:enhance={submit}>
				<input type="hidden" name="id" value={timer.entry.id} />
				<input type="hidden" name="side" value={other(timer.entry.side)} />
				<Button type="submit" variant="outline" class="w-full">Switch side</Button>
			</form>
		{/if}
		<form
			method="POST"
			action="{actions}{timer.kind === 'sleep' ? '?/endSleep' : '?/endNursing'}"
			class="flex flex-1"
			use:enhance={submit}
		>
			<input type="hidden" name="id" value={timer.entry.id} />
			<Button type="submit" class="w-full"
				>{timer.kind === 'sleep' ? 'End sleep' : 'End nursing'}</Button
			>
		</form>
	</div>
</li>
