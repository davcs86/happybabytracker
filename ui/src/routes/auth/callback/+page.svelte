<script lang="ts">
	import SsoOutcome from '#lib/components/SsoOutcome.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
</script>

<svelte:head>
	{#if data.outcome === 'continue'}
		<meta http-equiv="refresh" content="0;url={data.to}" />
	{:else if data.outcome === 'confirm'}
		<title>You are linking {data.label} to {data.username} · happybabytracker</title>
	{/if}
</svelte:head>

{#if data.outcome === 'continue'}
	<SsoOutcome heading="Signing you in" back={{ href: data.to, label: 'Continue' }} />
{:else if data.outcome === 'confirm'}
	<main class="flex min-h-screen flex-col justify-center px-5 py-6">
		<div class="mx-auto flex w-full max-w-[400px] flex-col gap-4">
			<p class="text-base font-bold">Happy baby</p>
			<h1 class="[overflow-wrap:anywhere]">
				You are linking {data.label} to {data.username}
			</h1>
			<form
				method="POST"
				action="/auth/confirm"
				class="flex flex-col gap-3.5 rounded-2xl border bg-card p-4"
			>
				<Button type="submit" name="choice" value="confirm" size="xl">Confirm</Button>
				<Button type="submit" name="choice" value="cancel" variant="outline" size="xl">
					Cancel
				</Button>
			</form>
		</div>
	</main>
{:else if data.outcome === 'failed'}
	<SsoOutcome heading="Single sign-on did not complete" message={data.message} back={data.back} />
{/if}
