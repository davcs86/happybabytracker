<script lang="ts">
	import { resolve } from '$app/paths';
	import { enhance } from '$app/forms';
	import { ERROR_CLASS, keepValues } from '#lib/components/field.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { MODULE_LABEL } from '#lib/constants.js';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
</script>

<svelte:head>
	<title>Modules for {data.kid.name} · happybabytracker</title>
</svelte:head>

<p class="mb-2 text-sm">
	<a href={resolve('/(app)/kids/[kid]', { kid: data.kid.id })} class="underline">{data.kid.name}</a>
</p>
<h1 class="mb-1 text-xl font-semibold">Modules for {data.kid.name}</h1>
<p class="mb-4 text-sm text-muted-foreground">
	A module you turn off leaves the home screen; its entries stay and come back when you turn it on.
</p>

{#if form?.saved}<p role="status" class="mb-4 text-sm">Saved</p>{/if}
{#if form?.message}<p class="{ERROR_CLASS} mb-4" role="alert">{form.message}</p>{/if}

<form method="POST" action="?/save" class="flex max-w-md flex-col gap-4" use:enhance={keepValues}>
	<fieldset class="flex flex-col gap-3">
		<legend class="sr-only">Modules</legend>
		{#each data.modules as m (m.module)}
			<div class="flex items-start gap-3">
				<input
					id="module-{m.module}"
					type="checkbox"
					name="module"
					value={m.module}
					checked={m.enabled}
					aria-describedby="module-{m.module}-default"
					class="mt-1 size-4"
				/>
				<div class="flex flex-col">
					<label for="module-{m.module}" class="font-medium">{MODULE_LABEL[m.module]}</label>
					<span id="module-{m.module}-default" class="text-sm text-muted-foreground">
						{m.default ? 'On' : 'Off'} by default for this age
					</span>
				</div>
			</div>
		{/each}
	</fieldset>
	<Button type="submit" class="self-start">Save</Button>
</form>
