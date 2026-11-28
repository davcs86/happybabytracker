<script lang="ts">
	import SsoSection from '#lib/components/SsoSection.svelte';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
</script>

<svelte:head>
	<title>Link single sign-on · happybabytracker</title>
</svelte:head>

<div
	class="mt-6.5 flex max-w-[720px] flex-col gap-[18px] xl:mx-auto xl:grid xl:w-[920px] xl:max-w-none xl:grid-cols-[448px_448px] xl:items-start xl:gap-x-6"
>
	<div class="flex flex-col gap-1.5 xl:col-span-2">
		<h1>Link single sign-on</h1>
		<div aria-hidden="true" class="mt-0.5 border-t"></div>
	</div>
	<SsoSection
		action="/account/link-sso"
		state={data.linked ? 'linked' : 'linkable'}
		form={form ?? null}
	/>
</div>
