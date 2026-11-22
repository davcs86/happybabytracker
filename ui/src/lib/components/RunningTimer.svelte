<script lang="ts" module>
	import type { Nursing, Sleep } from '#lib/types.js';

	export type Timer = { kind: 'sleep'; entry: Sleep } | { kind: 'nursing'; entry: Nursing };
</script>

<script lang="ts">
	import { enhance } from '$app/forms';
	import { ERROR_CLASS } from '#lib/components/field.js';
	import { Button } from '#lib/components/ui/button/index.js';
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

	const BUTTON = 'h-11 px-4';
	const other = (side: 'left' | 'right') => (side === 'left' ? 'right' : 'left');
</script>

<li class="flex flex-col gap-2 rounded-lg border p-3">
	<div class="flex flex-wrap items-center gap-2">
		<span class="size-2 shrink-0 rounded-full bg-primary" aria-hidden="true"></span>
		{#if timer.kind === 'sleep'}
			<p class="mr-auto font-medium">Asleep since {formatClock(timer.entry.startedAt, zone)}</p>
			<form method="POST" action="?/endSleep" use:enhance>
				<input type="hidden" name="id" value={timer.entry.id} />
				<Button type="submit" variant="outline" class={BUTTON} aria-label="End sleep">End</Button>
			</form>
		{:else}
			<p class="mr-auto font-medium">
				Nursing, {timer.entry.side} since {formatClock(
					timer.entry.sideSince ?? timer.entry.startedAt,
					zone
				)}
			</p>
			{#if timer.entry.side}
				<form method="POST" action="?/switchSide" use:enhance>
					<input type="hidden" name="id" value={timer.entry.id} />
					<input type="hidden" name="side" value={other(timer.entry.side)} />
					<Button type="submit" variant="outline" class={BUTTON}>Switch side</Button>
				</form>
			{/if}
			<form method="POST" action="?/endNursing" use:enhance>
				<input type="hidden" name="id" value={timer.entry.id} />
				<Button type="submit" variant="outline" class={BUTTON} aria-label="End nursing">End</Button>
			</form>
		{/if}
	</div>
	{#if overrun}
		<p class={ERROR_CLASS} role="alert">
			This timer ran over 24 hours:
			<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- resolve() renders a relative href without JavaScript, and the link keeps an absolute path -->
			<a href={entryPath(kidId, timer.kind, timer.entry.id)} class="underline">set the end time</a>
		</p>
	{:else if message}
		<p class={ERROR_CLASS} role="alert">{message}</p>
	{/if}
</li>
