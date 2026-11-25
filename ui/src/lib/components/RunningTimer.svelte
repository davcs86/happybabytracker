<script lang="ts" module>
	import type { Nursing, Sleep } from '#lib/types.js';

	export type Timer = { kind: 'sleep'; entry: Sleep } | { kind: 'nursing'; entry: Nursing };
</script>

<script lang="ts">
	import HeartIcon from '@lucide/svelte/icons/heart';
	import MoonIcon from '@lucide/svelte/icons/moon';
	import { enhance } from '$app/forms';
	import { ERROR_CLASS } from '#lib/components/field.js';
	import { Alert } from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { MAX_SESSION_HOURS, TRACKER_TITLE } from '#lib/constants.js';
	import { entryPath } from '#lib/paths.js';
	import { formatClock } from '#lib/time-zone.js';

	let {
		timer,
		zone,
		kidId,
		message = null,
		overrun = false
	}: {
		timer: Timer;
		zone: string;
		kidId: string;
		message?: string | null;
		overrun?: boolean;
	} = $props();

	const other = (side: 'left' | 'right') => (side === 'left' ? 'right' : 'left');
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
		{#if timer.kind === 'sleep'}
			<p class="min-w-0 flex-1 text-[15px] font-semibold">
				Asleep since {formatClock(timer.entry.startedAt, zone)}
			</p>
		{:else}
			<p class="min-w-0 flex-1 text-[15px] font-semibold">
				Nursing, {timer.entry.side} since {formatClock(
					timer.entry.sideSince ?? timer.entry.startedAt,
					zone
				)}
			</p>
		{/if}
	</div>
	{#if overrun}
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
	{:else if message}
		<p class={ERROR_CLASS} role="alert">{message}</p>
	{/if}
	<div class="flex gap-2">
		{#if timer.kind === 'nursing' && timer.entry.side}
			<form method="POST" action="?/switchSide" class="flex flex-1" use:enhance>
				<input type="hidden" name="id" value={timer.entry.id} />
				<input type="hidden" name="side" value={other(timer.entry.side)} />
				<Button type="submit" variant="outline" class="w-full">Switch side</Button>
			</form>
		{/if}
		<form
			method="POST"
			action={timer.kind === 'sleep' ? '?/endSleep' : '?/endNursing'}
			class="flex flex-1"
			use:enhance
		>
			<input type="hidden" name="id" value={timer.entry.id} />
			<Button type="submit" class="w-full"
				>{timer.kind === 'sleep' ? 'End sleep' : 'End nursing'}</Button
			>
		</form>
	</div>
</li>
