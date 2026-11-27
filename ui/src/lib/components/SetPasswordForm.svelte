<script lang="ts">
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import { MIN_PASSWORD_LEN } from '#lib/constants.js';
	import { passwordProblem, type PasswordProblem } from '#lib/password.js';

	type FormResult = {
		field?: PasswordProblem['field'] | null;
		message?: string | null;
		invalid?: boolean;
	} | null;

	let {
		heading,
		description,
		valid,
		lifetime,
		username,
		form,
		action,
		mode
	}: {
		heading: string;
		description: string;
		valid: boolean;
		lifetime: string;
		username?: string;
		form: FormResult;
		action?: string;
		mode: 'password' | 'sso' | 'both' | 'off';
	} = $props();

	let password = $state('');
	let confirm = $state('');
	let clientProblem = $state<PasswordProblem | null>(null);

	const problem = $derived(clientProblem ?? (form?.message ? form : null));
	const fieldError = (field: PasswordProblem['field']) =>
		problem?.field === field ? problem.message : null;

	function onsubmit(event: SubmitEvent) {
		clientProblem = passwordProblem(password, confirm);
		if (clientProblem) event.preventDefault();
	}
</script>

<svelte:head>
	<title>{heading} · happybabytracker</title>
</svelte:head>

<main class="flex min-h-screen flex-col justify-center px-5 py-6">
	<div class="mx-auto flex w-full max-w-[400px] flex-col gap-4">
		<p class="text-base font-bold">Happy baby</p>
		<h1>{heading}</h1>
		{#if valid && !form?.invalid && mode !== 'off'}
			<p class="text-sm leading-normal text-muted-foreground">{description}</p>
		{/if}
		{#if !valid || form?.invalid}
			<div class="flex flex-col gap-3.5 rounded-2xl border bg-card p-4">
				<Alert.Root>
					<Alert.Description>
						<b>This link is no longer valid.</b> Ask the person who sent it for a new one. This kind
						of link lasts {lifetime}.
					</Alert.Description>
				</Alert.Root>
				<Button href="/login" variant="outline" size="xl">Go to sign in</Button>
			</div>
		{:else if mode === 'off'}
			<div class="flex flex-col gap-3.5 rounded-2xl border bg-card p-4">
				<Alert.Root>
					<Alert.Description>
						{form?.message ??
							'This link works only while password sign-in is on. Ask the person who sent it.'}
					</Alert.Description>
				</Alert.Root>
				<Button href="/login" variant="outline" size="xl">Go to sign in</Button>
			</div>
		{:else if mode === 'sso'}
			<form
				method="POST"
				action="?/sso"
				class="flex flex-col gap-3.5 rounded-2xl border bg-card p-4"
			>
				{#if form?.message}
					<Alert.Root>
						<Alert.Description>{form.message}</Alert.Description>
					</Alert.Root>
				{/if}
				<div class="grid gap-1.5">
					<Label for="username" class="text-xs">Username</Label>
					<Input
						id="username"
						value={username}
						autocomplete="username"
						readonly
						class="text-muted-foreground"
					/>
				</div>
				<Button type="submit" size="xl">Continue with single sign-on</Button>
			</form>
		{:else}
			<form
				method="POST"
				{action}
				class="flex flex-col gap-3.5 rounded-2xl border bg-card p-4"
				novalidate
				{onsubmit}
			>
				{#if problem && !problem.field}
					<Alert.Root>
						<Alert.Description>{problem.message}</Alert.Description>
					</Alert.Root>
				{/if}
				<div class="grid gap-1.5">
					<Label for="username" class="text-xs">Username</Label>
					<Input
						id="username"
						value={username}
						autocomplete="username"
						readonly
						class="text-muted-foreground"
					/>
				</div>
				<div class="grid gap-1.5">
					<Label for="password" class="text-xs">New password</Label>
					<Input
						id="password"
						name="password"
						type="password"
						autocomplete="new-password"
						bind:value={password}
						aria-invalid={fieldError('password') ? 'true' : undefined}
						aria-describedby="password-help"
					/>
					<p
						id="password-help"
						class="text-xs leading-normal {fieldError('password')
							? 'font-semibold text-destructive'
							: 'text-muted-foreground'}"
					>
						{fieldError('password') ?? `At least ${MIN_PASSWORD_LEN} characters`}
					</p>
				</div>
				<div class="grid gap-1.5">
					<Label for="confirm" class="text-xs">Confirm password</Label>
					<Input
						id="confirm"
						name="confirm"
						type="password"
						autocomplete="new-password"
						bind:value={confirm}
						aria-invalid={fieldError('confirm') ? 'true' : undefined}
						aria-describedby={fieldError('confirm') ? 'confirm-error' : undefined}
					/>
					{#if fieldError('confirm')}
						<p id="confirm-error" class="text-[12.5px] font-semibold text-destructive">
							{fieldError('confirm')}
						</p>
					{/if}
				</div>
				<Button type="submit" size="xl">Set password</Button>
			</form>
			{#if mode === 'both'}
				<form method="POST" action="?/sso">
					<Button type="submit" variant="outline" size="xl" class="w-full">
						Continue with single sign-on
					</Button>
				</form>
			{/if}
		{/if}
	</div>
</main>
