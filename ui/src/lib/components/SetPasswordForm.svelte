<script lang="ts">
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
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
		username,
		form
	}: {
		heading: string;
		description: string;
		valid: boolean;
		username?: string;
		form: FormResult;
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

<main class="flex min-h-screen items-center justify-center p-4">
	<Card.Root class="w-full max-w-sm">
		<Card.Header>
			<h1 class="leading-none font-semibold">{heading}</h1>
			{#if valid && !form?.invalid}
				<Card.Description>{description}</Card.Description>
			{/if}
		</Card.Header>
		<Card.Content>
			{#if !valid || form?.invalid}
				<Alert.Root variant="destructive" role="alert">
					<Alert.Title>This link is no longer valid</Alert.Title>
					<Alert.Description>Ask the person who sent it for a new one.</Alert.Description>
				</Alert.Root>
			{:else}
				<form method="POST" class="flex flex-col gap-4" novalidate {onsubmit}>
					{#if problem && !problem.field}
						<Alert.Root variant="destructive" role="alert">
							<Alert.Description>{problem.message}</Alert.Description>
						</Alert.Root>
					{/if}
					<div class="flex flex-col gap-2">
						<Label for="username">Username</Label>
						<Input id="username" value={username} autocomplete="username" readonly />
					</div>
					<div class="flex flex-col gap-2">
						<Label for="password">New password</Label>
						<Input
							id="password"
							name="password"
							type="password"
							autocomplete="new-password"
							required
							bind:value={password}
							aria-invalid={fieldError('password') ? 'true' : undefined}
							aria-describedby="password-help"
						/>
						<p
							id="password-help"
							class="text-sm text-muted-foreground"
							role={fieldError('password') ? 'alert' : undefined}
						>
							{fieldError('password') ?? `At least ${MIN_PASSWORD_LEN} characters`}
						</p>
					</div>
					<div class="flex flex-col gap-2">
						<Label for="confirm">Confirm password</Label>
						<Input
							id="confirm"
							name="confirm"
							type="password"
							autocomplete="new-password"
							required
							bind:value={confirm}
							aria-invalid={fieldError('confirm') ? 'true' : undefined}
							aria-describedby={fieldError('confirm') ? 'confirm-error' : undefined}
						/>
						{#if fieldError('confirm')}
							<p id="confirm-error" class="text-sm text-destructive" role="alert">
								{fieldError('confirm')}
							</p>
						{/if}
					</div>
					<Button type="submit">Set password</Button>
				</form>
			{/if}
		</Card.Content>
	</Card.Root>
</main>
