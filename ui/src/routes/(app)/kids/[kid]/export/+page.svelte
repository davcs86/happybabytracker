<script lang="ts">
	import CheckIcon from '@lucide/svelte/icons/check';
	import Field from '#lib/components/entry/Field.svelte';
	import { ERROR_CLASS } from '#lib/components/field.js';
	import { Alert } from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { MODULE_LABEL } from '#lib/constants.js';
	import { exportPdfPath } from '#lib/paths.js';
	import type { ModuleId } from '#lib/types.js';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const error = (name: string) => (data.error?.field === name ? data.error.message : null);
</script>

<svelte:head>
	<title>Export PDF · {data.kid.name} · happybabytracker</title>
</svelte:head>

<!-- A chip is a label around a native checkbox laid over it: pressed = fill, border and a leading check. -->
{#snippet box(m: { id: ModuleId; checked: boolean })}
	<label
		class="group/chip relative inline-flex min-h-11 items-center gap-1.5 rounded-sm border bg-card px-3.5 text-sm font-medium has-checked:border-primary has-checked:bg-accent has-checked:font-semibold has-focus-visible:outline-2 has-focus-visible:outline-offset-2 has-focus-visible:outline-primary has-[[aria-invalid=true]]:border-destructive"
	>
		<input
			type="checkbox"
			name="modules"
			value={m.id}
			checked={m.checked}
			aria-invalid={error('modules') ? 'true' : undefined}
			aria-describedby={error('modules') ? 'export-modules-error' : undefined}
			class="absolute inset-0 size-full cursor-pointer opacity-0"
		/>
		<CheckIcon
			aria-hidden="true"
			class="hidden size-3.5 text-primary group-has-checked/chip:block"
		/>
		{MODULE_LABEL[m.id]}
	</label>
{/snippet}

<div class="flex max-w-[448px] flex-col gap-[18px] xl:mx-auto xl:w-[920px] xl:max-w-none">
	<div class="mt-5 flex flex-col gap-1.5">
		<h1>Export PDF</h1>
		<p class="text-sm leading-snug text-muted-foreground">{data.ages[data.kid.id]}</p>
		<p class="text-[13.5px] text-muted-foreground">
			The file is made when you download it and is not kept.
		</p>
	</div>

	{#if error('busy')}<Alert class="[overflow-wrap:anywhere]">{error('busy')}</Alert>{/if}

	<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- a POST endpoint, not a page -->
	<form
		method="POST"
		action={exportPdfPath(data.kid.id)}
		class="flex flex-col gap-[18px] xl:grid xl:grid-cols-[448px_448px] xl:grid-rows-[auto_1fr] xl:gap-x-6"
	>
		<div class="grid grid-cols-2 gap-3">
			<Field id="export-from" label="From" error={error('from')}>
				{#snippet children(control)}
					<Input
						{...control}
						type="date"
						name="from"
						value={data.from}
						min={data.kid.birthDate}
						max={data.today}
						required
					/>
				{/snippet}
			</Field>
			<Field id="export-to" label="To" error={error('to')}>
				{#snippet children(control)}
					<Input
						{...control}
						type="date"
						name="to"
						value={data.to}
						min={data.kid.birthDate}
						max={data.today}
						required
					/>
				{/snippet}
			</Field>
		</div>
		<!-- Each box carries the error: a plain group has no invalid state of its own. -->
		<fieldset class="flex min-w-0 flex-col gap-1 xl:row-span-2">
			<legend class="mb-1 p-0 text-[15px] font-semibold">Modules to include</legend>
			<div class="flex flex-wrap gap-2">
				{#each data.on as m (m.id)}{@render box(m)}{/each}
			</div>
			{#if data.off.length > 0}
				<div role="group" aria-labelledby="export-off" class="mt-2 flex flex-col gap-1">
					<h2 id="export-off" class="text-[13px] font-semibold text-muted-foreground">
						Switched off for this child
					</h2>
					<div class="flex flex-wrap gap-2">
						{#each data.off as m (m.id)}{@render box(m)}{/each}
					</div>
				</div>
			{/if}
			{#if error('modules')}
				<p id="export-modules-error" class="{ERROR_CLASS} mt-1" role="alert">{error('modules')}</p>
			{/if}
		</fieldset>
		<Button type="submit" class="self-start px-5">Download PDF</Button>
	</form>
</div>
