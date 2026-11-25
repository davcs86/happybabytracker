<script lang="ts">
	import { enhance, type SubmitFunction } from '$app/forms';
	import { ERROR_CLASS, keepValues } from '#lib/components/field.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import { PUSH_LABEL_MAX_LEN } from '#lib/constants.js';
	import { formatInstant } from '#lib/format.js';
	import type { Kid, PushDevice } from '#lib/types.js';

	type Props = {
		device: PushDevice;
		kids: Kid[];
		// This browser's device while Web Push is on: its name and kids can change.
		editable: boolean;
		values: { label: string; kids: string[] } | null;
		labelError: string | null;
		onremoved?: () => Promise<void>;
	};

	let { device, kids, editable, values, labelError, onremoved }: Props = $props();

	const labelId = $derived(`device-${device.id}-label`);
	const chosen = $derived(values?.kids ?? device.kids);

	const afterRemove: SubmitFunction = () => {
		return async ({ result, update }) => {
			if (result.type === 'success' && onremoved) await onremoved();
			await update();
		};
	};
</script>

{#if editable}
	<form
		method="POST"
		action="?/device"
		class="flex flex-col gap-3.5"
		novalidate
		use:enhance={keepValues}
	>
		<input type="hidden" name="device" value={device.id} />
		<div class="flex flex-col gap-1.5">
			<Label for={labelId}>Device name</Label>
			<Input
				id={labelId}
				name="label"
				maxlength={PUSH_LABEL_MAX_LEN}
				value={values?.label ?? device.label}
				required
				aria-invalid={labelError ? 'true' : undefined}
				aria-describedby={labelError ? `${labelId}-error` : undefined}
			/>
			{#if labelError}
				<p id="{labelId}-error" class={ERROR_CLASS}>{labelError}</p>
			{/if}
		</div>
		<fieldset class="flex flex-col">
			<legend class="mb-1 text-[13px] text-muted-foreground">Reminders on this device for</legend>
			{#each kids as kid (kid.id)}
				<label class="flex min-h-11 items-center gap-3 text-[15px] [overflow-wrap:anywhere]">
					<input
						type="checkbox"
						name="kid"
						value={kid.id}
						checked={chosen.includes(kid.id)}
						class="size-6 shrink-0 accent-primary"
					/>{kid.name}
				</label>
			{/each}
		</fieldset>
		<Button type="submit">Save</Button>
	</form>
{:else}
	<div class="min-w-0 flex-1">
		<p class="text-[15px] font-semibold [overflow-wrap:anywhere]">{device.label}</p>
		<p class="text-[12.5px] text-muted-foreground">Added {formatInstant(device.createdAt)}</p>
	</div>
{/if}
<form method="POST" action="?/remove" use:enhance={afterRemove}>
	<input type="hidden" name="device" value={device.id} />
	<Button type="submit" variant="outline" class={editable ? 'w-full' : ''}
		>{#if editable}Turn off on this device{:else}Remove<span class="sr-only">
				{device.label}</span
			>{/if}</Button
	>
</form>
