<script lang="ts">
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import type { PageProps } from './$types';

	let { form }: PageProps = $props();
</script>

<svelte:head>
	<title>Sign in · happybabytracker</title>
</svelte:head>

<main class="flex min-h-screen items-center justify-center p-4">
	<Card.Root class="w-full max-w-sm">
		<Card.Header>
			<h1 class="leading-none font-semibold">happybabytracker</h1>
			<Card.Description>Sign in to continue</Card.Description>
		</Card.Header>
		<Card.Content>
			<form method="POST" class="flex flex-col gap-4">
				{#if form?.message}
					<Alert.Root variant="destructive" role="alert">
						<Alert.Description>{form.message}</Alert.Description>
					</Alert.Root>
				{/if}
				<div class="flex flex-col gap-2">
					<Label for="username">Username</Label>
					<Input
						id="username"
						name="username"
						autocomplete="username"
						autocapitalize="none"
						value={form?.username ?? ''}
						required
					/>
				</div>
				<div class="flex flex-col gap-2">
					<Label for="password">Password</Label>
					<Input
						id="password"
						name="password"
						type="password"
						autocomplete="current-password"
						required
					/>
				</div>
				<Button type="submit">Sign in</Button>
			</form>
		</Card.Content>
	</Card.Root>
</main>
