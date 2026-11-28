<script lang="ts">
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import { page } from '$app/state';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	const next = $derived(page.url.searchParams.get('next') ?? '');

	const usernameError = $derived(form && 'usernameError' in form ? form.usernameError : null);
	const passwordError = $derived(form && 'passwordError' in form ? form.passwordError : null);
	const message = $derived(form && 'message' in form ? form.message : null);
</script>

<svelte:head>
	<title>Sign in · happybabytracker</title>
</svelte:head>

<main class="flex min-h-screen flex-col justify-center px-5 py-6">
	<div class="mx-auto flex w-full max-w-[400px] flex-col gap-4">
		<p class="text-base font-bold">Happy baby</p>
		<h1>Sign in</h1>
		<p class="text-sm leading-normal text-muted-foreground">Sign in to continue.</p>
		{#if data.mode === 'oidc' && data.sso}
			<form
				method="POST"
				action="/auth/sso"
				class="flex flex-col gap-3.5 rounded-2xl border bg-card p-4"
			>
				{#if message}
					<Alert.Root>
						<Alert.Description>{message}</Alert.Description>
					</Alert.Root>
				{/if}
				<input type="hidden" name="next" value={next} />
				<Button type="submit" size="xl">Continue with single sign-on</Button>
			</form>
		{:else if data.mode === 'oidc'}
			<div class="flex flex-col gap-3.5 rounded-2xl border bg-card p-4">
				<Alert.Root>
					<Alert.Description>Sign-in is unavailable</Alert.Description>
				</Alert.Root>
			</div>
		{:else}
			<form method="POST" novalidate class="flex flex-col gap-3.5 rounded-2xl border bg-card p-4">
				{#if message}
					<Alert.Root id="login-message">
						<Alert.Description>{message}</Alert.Description>
					</Alert.Root>
				{/if}
				<div class="grid gap-1.5">
					<Label for="username" class="text-xs">Username</Label>
					<Input
						id="username"
						name="username"
						autocomplete="username"
						autocapitalize="none"
						spellcheck={false}
						value={form?.username ?? ''}
						aria-invalid={usernameError ? 'true' : undefined}
						aria-describedby={usernameError
							? 'username-error'
							: message
								? 'login-message'
								: undefined}
					/>
					{#if usernameError}
						<p id="username-error" class="text-[12.5px] font-semibold text-destructive">
							{usernameError}
						</p>
					{/if}
				</div>
				<div class="grid gap-1.5">
					<Label for="password" class="text-xs">Password</Label>
					<Input
						id="password"
						name="password"
						type="password"
						autocomplete="current-password"
						aria-invalid={passwordError ? 'true' : undefined}
						aria-describedby={passwordError
							? 'password-error'
							: message
								? 'login-message'
								: undefined}
					/>
					{#if passwordError}
						<p id="password-error" class="text-[12.5px] font-semibold text-destructive">
							{passwordError}
						</p>
					{/if}
				</div>
				<Button type="submit" size="xl">Sign in</Button>
			</form>
			{#if data.sso}
				<form method="POST" action="/auth/sso">
					<input type="hidden" name="next" value={next} />
					<Button type="submit" variant="outline" size="xl" class="w-full">
						Continue with single sign-on
					</Button>
				</form>
			{/if}
		{/if}
		<p class="text-[12.5px] leading-normal text-muted-foreground">
			No account yet? A guardian or the administrator sends you a one-time link.
		</p>
	</div>
</main>
