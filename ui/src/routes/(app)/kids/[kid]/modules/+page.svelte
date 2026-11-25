<script lang="ts">
	import { enhance } from '$app/forms';
	import { keepValues } from '#lib/components/field.js';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { MODULE_LABEL } from '#lib/constants.js';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
</script>

<svelte:head>
	<title>Modules for {data.kid.name} · happybabytracker</title>
</svelte:head>

<div class="mt-5 flex max-w-[448px] flex-col gap-[18px] xl:mx-auto xl:w-[920px] xl:max-w-none">
	<div class="flex flex-col gap-1.5">
		<h1>Modules for {data.kid.name}</h1>
		<p class="text-sm leading-normal text-muted-foreground">
			A module you turn off leaves the home screen; its entries stay and come back when you turn it
			on.
		</p>
	</div>

	{#if form?.saved}<Alert.Root role="status">Saved</Alert.Root>{/if}
	{#if form?.message}<Alert.Root>{form.message}</Alert.Root>{/if}

	<form
		method="POST"
		action="?/save"
		class="flex max-w-[448px] flex-col gap-4"
		use:enhance={keepValues}
	>
		<fieldset class="flex flex-col rounded-2xl border bg-card px-4">
			<legend class="sr-only">Modules</legend>
			{#each data.modules as m (m.module)}
				<div class="relative flex min-h-14 items-start gap-3 border-t py-2.5 first:border-t-0">
					<input
						id="module-{m.module}"
						type="checkbox"
						name="module"
						value={m.module}
						checked={m.enabled}
						aria-describedby="module-{m.module}-default"
						class="relative z-10 mt-0.5 size-5 shrink-0 accent-primary"
					/>
					<div class="flex min-w-0 flex-col">
						<label for="module-{m.module}" class="font-medium after:absolute after:inset-0"
							>{MODULE_LABEL[m.module]}</label
						>
						<span id="module-{m.module}-default" class="text-[13px] text-muted-foreground">
							{m.default ? 'On' : 'Off'} by default for this age
						</span>
					</div>
				</div>
			{/each}
		</fieldset>
		<Button type="submit">Save</Button>
	</form>
</div>
