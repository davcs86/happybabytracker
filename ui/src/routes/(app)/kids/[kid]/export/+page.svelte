<script lang="ts">
	import { resolve } from '$app/paths';
	import Field from '#lib/components/entry/Field.svelte';
	import { ERROR_CLASS } from '#lib/components/field.js';
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

{#snippet box(m: { id: ModuleId; checked: boolean })}
	<label class="flex min-h-11 items-center gap-2 text-sm">
		<input
			type="checkbox"
			name="modules"
			value={m.id}
			checked={m.checked}
			aria-invalid={error('modules') ? 'true' : undefined}
			aria-describedby={error('modules') ? 'export-modules-error' : undefined}
			class="size-4"
		/>
		{MODULE_LABEL[m.id]}
	</label>
{/snippet}

<p class="mb-2 text-sm">
	<a href={resolve('/(app)/kids/[kid]', { kid: data.kid.id })} class="underline">{data.kid.name}</a>
</p>
<h1 class="mb-1 text-xl font-semibold [overflow-wrap:anywhere]">Export PDF for {data.kid.name}</h1>
<p class="mb-4 text-sm text-muted-foreground">
	The file is made when you download it and is not kept.
</p>

{#if error('busy')}<p class="{ERROR_CLASS} mb-4" role="alert">{error('busy')}</p>{/if}

<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- a POST endpoint, not a page -->
<form method="POST" action={exportPdfPath(data.kid.id)} class="flex max-w-md flex-col gap-4">
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
	<fieldset class="flex flex-col gap-1">
		<legend class="mb-1 text-sm font-medium">Modules to include</legend>
		<div class="grid grid-cols-2 gap-x-4">
			{#each data.on as m (m.id)}{@render box(m)}{/each}
		</div>
		{#if data.off.length > 0}
			<div role="group" aria-labelledby="export-off" class="mt-2">
				<h2 id="export-off" class="text-sm font-medium">Switched off for this child</h2>
				<div class="grid grid-cols-2 gap-x-4">
					{#each data.off as m (m.id)}{@render box(m)}{/each}
				</div>
			</div>
		{/if}
		{#if error('modules')}
			<p id="export-modules-error" class={ERROR_CLASS} role="alert">{error('modules')}</p>
		{/if}
	</fieldset>
	<Button type="submit" class="self-start">Download PDF</Button>
</form>
