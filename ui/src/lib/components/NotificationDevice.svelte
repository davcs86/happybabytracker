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
		class="flex flex-col gap-4"
		novalidate
		use:enhance={keepValues}
	>
		<input type="hidden" name="device" value={device.id} />
		<div class="flex flex-col gap-2">
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
				<p id="{labelId}-error" class={ERROR_CLASS} role="alert">{labelError}</p>
			{/if}
		</div>
		<fieldset class="flex flex-col gap-2">
			<legend class="mb-1 text-sm font-medium">Reminders on this device for</legend>
			{#each kids as kid (kid.id)}
				<div class="flex items-center gap-2">
					<input
						id="device-{device.id}-kid-{kid.id}"
						type="checkbox"
						name="kid"
						value={kid.id}
						checked={chosen.includes(kid.id)}
						class="size-4"
					/>
					<label for="device-{device.id}-kid-{kid.id}">{kid.name}</label>
				</div>
			{/each}
		</fieldset>
		<Button type="submit" class="self-start">Save</Button>
	</form>
{:else}
	<div class="text-sm">
		<p class="font-medium">{device.label}</p>
		<p class="text-muted-foreground">Added {formatInstant(device.createdAt)}</p>
	</div>
{/if}
<form method="POST" action="?/remove" use:enhance={afterRemove}>
	<input type="hidden" name="device" value={device.id} />
	<Button type="submit" variant="outline"
		>{#if editable}Turn off on this device{:else}Remove<span class="sr-only">
				{device.label}</span
			>{/if}</Button
	>
</form>
