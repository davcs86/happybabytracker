<script lang="ts">
	import { enhance } from '$app/forms';
	import { ERROR_CLASS, SELECT_CLASS, keepValues } from '#lib/components/field.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import { DISPLAY_NAME_MAX_LEN, MIN_PASSWORD_LEN } from '#lib/constants.js';
	import { formatInstant } from '#lib/format.js';
	import { notificationsPath } from '#lib/paths.js';
	import { passwordProblem } from '#lib/password.js';
	import type { Units } from '#lib/types.js';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	const UNITS: { key: keyof Units; label: string; options: [string, string][] }[] = [
		{
			key: 'weight',
			label: 'Weight',
			options: [
				['kg', 'kg'],
				['lb', 'lb']
			]
		},
		{
			key: 'length',
			label: 'Length',
			options: [
				['cm', 'cm'],
				['in', 'in']
			]
		},
		{
			key: 'volume',
			label: 'Volume',
			options: [
				['ml', 'ml'],
				['floz', 'fl oz']
			]
		},
		{
			key: 'temp',
			label: 'Temperature',
			options: [
				['c', '°C'],
				['f', '°F']
			]
		}
	];

	const PASSWORD_FIELDS: [string, string, 'current-password' | 'new-password'][] = [
		['current', 'Current password', 'current-password'],
		['new', 'New password', 'new-password'],
		['confirm', 'Confirm new password', 'new-password']
	];

	const parent = $derived(data.session?.parent);
	const profile = $derived(
		form?.action === 'profile' && 'values' in form && form.values ? form.values : null
	);
	const fieldError = (action: string, f: string) =>
		form?.action === action && 'field' in form && form.field === f ? form.message : null;
	const pageError = (action: string) =>
		form?.action === action && 'field' in form && !form.field && 'message' in form
			? form.message
			: null;

	let clientProblem = $state<{ field: string; message: string } | null>(null);
	const passwordError = (f: string) =>
		clientProblem
			? clientProblem.field === f
				? clientProblem.message
				: null
			: fieldError('password', f);

	function checkPassword(event: SubmitEvent) {
		const fd = new FormData(event.currentTarget as HTMLFormElement);
		const problem = passwordProblem(String(fd.get('new') ?? ''), String(fd.get('confirm') ?? ''));
		clientProblem = problem && {
			field: problem.field === 'password' ? 'new' : 'confirm',
			message: problem.message
		};
		if (problem) event.preventDefault();
	}
	// Password fields never keep their text, whatever the outcome.
	const clearAfter = () => {
		return async ({
			formElement,
			update
		}: {
			formElement: HTMLFormElement;
			update: () => Promise<void>;
		}) => {
			await update();
			formElement.reset();
		};
	};
	const describe = (id: string, error: string | null, help?: string) =>
		[help, error ? `${id}-error` : undefined].filter(Boolean).join(' ') || undefined;
</script>

<svelte:head>
	<title>Account · happybabytracker</title>
</svelte:head>

<h1 class="mb-4 text-xl font-semibold">Account</h1>
<p class="mb-6 text-sm">
	<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- the path helper, absolute like the other links to pages without JavaScript -->
	<a href={notificationsPath} class="underline">Notifications</a>
</p>

<section aria-labelledby="profile-heading" class="flex max-w-md flex-col gap-3">
	<h2 id="profile-heading" class="font-semibold">Profile</h2>
	{#if form?.action === 'profile' && 'saved' in form}
		<p role="status" class="text-sm">Saved</p>
	{/if}
	<form
		method="POST"
		action="?/profile"
		class="flex flex-col gap-4"
		novalidate
		use:enhance={keepValues}
	>
		{#if pageError('profile')}<p class={ERROR_CLASS} role="alert">{pageError('profile')}</p>{/if}
		<div class="flex flex-col gap-2">
			<Label for="display_name">Display name</Label>
			<Input
				id="display_name"
				name="display_name"
				maxlength={DISPLAY_NAME_MAX_LEN}
				value={profile?.displayName ?? parent?.displayName ?? ''}
				required
				aria-invalid={fieldError('profile', 'display_name') ? 'true' : undefined}
				aria-describedby={describe('display_name', fieldError('profile', 'display_name'))}
			/>
			{#if fieldError('profile', 'display_name')}
				<p id="display_name-error" class={ERROR_CLASS} role="alert">
					{fieldError('profile', 'display_name')}
				</p>
			{/if}
		</div>
		<div class="grid grid-cols-2 gap-4">
			{#each UNITS as unit (unit.key)}
				<div class="flex flex-col gap-2">
					<Label for="unit-{unit.key}">{unit.label}</Label>
					<select
						id="unit-{unit.key}"
						name={unit.key}
						class={SELECT_CLASS}
						value={profile?.units[unit.key] ?? parent?.units[unit.key]}
						aria-invalid={fieldError('profile', unit.key) ? 'true' : undefined}
						aria-describedby={describe(`unit-${unit.key}`, fieldError('profile', unit.key))}
					>
						{#each unit.options as [value, label] (value)}
							<option {value}>{label}</option>
						{/each}
					</select>
					{#if fieldError('profile', unit.key)}
						<p id="unit-{unit.key}-error" class={ERROR_CLASS} role="alert">
							{fieldError('profile', unit.key)}
						</p>
					{/if}
				</div>
			{/each}
		</div>
		<Button type="submit" class="self-start">Save</Button>
	</form>
</section>

<section aria-labelledby="password-heading" class="mt-8 flex max-w-md flex-col gap-3">
	<h2 id="password-heading" class="font-semibold">Password</h2>
	{#if form?.action === 'password' && 'changed' in form}
		<p role="status" class="text-sm">Password changed. Your other sessions were signed out.</p>
	{/if}
	<form
		method="POST"
		action="?/password"
		class="flex flex-col gap-4"
		novalidate
		onsubmit={checkPassword}
		use:enhance={clearAfter}
	>
		{#if pageError('password')}<p class={ERROR_CLASS} role="alert">{pageError('password')}</p>{/if}
		{#each PASSWORD_FIELDS as [name, label, autocomplete] (name)}
			{@const error = passwordError(name)}
			<div class="flex flex-col gap-2">
				<Label for="pw-{name}">{label}</Label>
				<Input
					id="pw-{name}"
					{name}
					type="password"
					{autocomplete}
					required
					aria-invalid={error ? 'true' : undefined}
					aria-describedby={describe(
						`pw-${name}`,
						error,
						name === 'new' ? 'pw-new-help' : undefined
					)}
				/>
				{#if name === 'new'}
					<p id="pw-new-help" class="text-sm text-muted-foreground">
						At least {MIN_PASSWORD_LEN} characters
					</p>
				{/if}
				{#if error}<p id="pw-{name}-error" class={ERROR_CLASS} role="alert">{error}</p>{/if}
			</div>
		{/each}
		<Button type="submit" class="self-start">Change password</Button>
	</form>
</section>

<section aria-labelledby="sessions-heading" class="mt-8 flex max-w-md flex-col gap-3">
	<h2 id="sessions-heading" class="font-semibold">Sessions</h2>
	{#if pageError('signout')}<p class={ERROR_CLASS} role="alert">{pageError('signout')}</p>{/if}
	<ul aria-label="Sessions" class="flex flex-col gap-2">
		{#each data.sessions as s (s.id)}
			<li class="flex flex-wrap items-center justify-between gap-2 rounded-lg border p-3">
				<div class="text-sm">
					<p class="font-medium">{s.current ? 'This device' : 'Another device'}</p>
					<p class="text-muted-foreground">
						Signed in {formatInstant(s.createdAt)} · last seen {formatInstant(s.lastSeenAt)}
					</p>
				</div>
				{#if !s.current}
					<form method="POST" action="?/signout" use:enhance>
						<input type="hidden" name="session_id" value={s.id} />
						<Button type="submit" variant="outline"
							>Sign out<span class="sr-only">
								the session started {formatInstant(s.createdAt)}</span
							></Button
						>
					</form>
				{/if}
			</li>
		{/each}
	</ul>
</section>
