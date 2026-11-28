<script lang="ts">
	import SsoSection, { ssoSectionState } from '#lib/components/SsoSection.svelte';
	import { Card } from '#lib/components/ui/card/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const sso = $derived(ssoSectionState(data));
</script>

<svelte:head>
	<title>Account · happybabytracker</title>
</svelte:head>

<div
	class="mt-6.5 flex max-w-[720px] flex-col gap-[18px] xl:mx-auto xl:grid xl:w-[920px] xl:max-w-none xl:grid-cols-[448px_448px] xl:items-start xl:gap-x-6"
>
	<div class="flex flex-col gap-1.5 xl:col-span-2">
		<h1>Account</h1>
		<div aria-hidden="true" class="mt-0.5 border-t"></div>
	</div>

	<div class="contents xl:flex xl:flex-col xl:gap-[18px]">
		<Card role="region" aria-labelledby="details-heading" class="gap-3 p-4">
			<h2 id="details-heading" class="text-base font-semibold">Details</h2>
			<div class="grid gap-1.5">
				<Label for="username" class="text-xs">Username</Label>
				<Input
					id="username"
					value={data.username}
					autocomplete="username"
					readonly
					class="text-muted-foreground"
				/>
			</div>
		</Card>
		{#if sso}<SsoSection action="/admin/account/link-sso" state={sso} form={null} />{/if}
	</div>
</div>
