<script lang="ts">
	import { enhance } from '$app/forms';
	import { ERROR_CLASS, SELECT_CLASS, keepValues } from '#lib/components/field.js';
	import { Alert } from '#lib/components/ui/alert/index.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Card } from '#lib/components/ui/card/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import { DISPLAY_NAME_MAX_LEN, MIN_PASSWORD_LEN } from '#lib/constants.js';
	import { formatInstant } from '#lib/format.js';
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
	const roleCount = (role: string) => data.kids.filter((k) => k.role === role).length;
	const roleLine = (role: string, label: string) => {
		const n = roleCount(role);
		return n ? `${label} for ${n} ${n === 1 ? 'child' : 'children'}` : null;
	};
	const summary = $derived(
		[roleLine('guardian', 'Guardian'), roleLine('caregiver', 'Caregiver')]
			.filter(Boolean)
			.join(', ') || 'No children yet'
	);
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
	const profileRefused = $derived(form?.action === 'profile' && 'field' in form && !!form.field);
	const passwordRefused = $derived(
		!!clientProblem || (form?.action === 'password' && 'field' in form && !!form.field)
	);
	const describe = (id: string, error: string | null, help?: string) =>
		[help, error ? `${id}-error` : undefined].filter(Boolean).join(' ') || undefined;
</script>

<svelte:head>
	<title>Your profile · happybabytracker</title>
</svelte:head>

<div
	class="mt-6.5 flex max-w-[720px] flex-col gap-[18px] xl:mx-auto xl:grid xl:w-[920px] xl:max-w-none xl:grid-cols-[448px_448px] xl:items-start xl:gap-x-6"
>
	<div class="flex flex-col gap-1.5 xl:col-span-2">
		<h1>Your profile</h1>
		<div aria-hidden="true" class="mt-0.5 border-t"></div>
		<p class="text-sm text-muted-foreground">{summary}</p>
	</div>

	<div class="contents xl:flex xl:flex-col xl:gap-[18px]">
		<Card role="region" aria-labelledby="profile-heading" class="gap-3 p-4">
			<h2 id="profile-heading" class="text-base font-semibold">Details</h2>
			{#if form?.action === 'profile' && 'saved' in form}
				<Alert role="status">Saved</Alert>
			{/if}
			<form
				method="POST"
				action="?/profile"
				class="flex flex-col gap-3"
				novalidate
				use:enhance={keepValues}
			>
				{#if profileRefused}<Alert>Check the fields with a message.</Alert>{/if}
				{#if pageError('profile')}<Alert>{pageError('profile')}</Alert>{/if}
				<div class="flex flex-col gap-1.5">
					<Label for="display_name" class="text-xs">Display name</Label>
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
						<p id="display_name-error" class={ERROR_CLASS}>
							{fieldError('profile', 'display_name')}
						</p>
					{/if}
				</div>
				<div>
					<p class="text-xs text-muted-foreground">Username</p>
					<p class="font-mono text-[15px] [overflow-wrap:anywhere]">{parent?.username}</p>
					<p class="text-xs text-muted-foreground">Your username is permanent.</p>
				</div>
				<div class="grid grid-cols-2 gap-3">
					{#each UNITS as unit (unit.key)}
						<div class="flex flex-col gap-1.5">
							<Label for="unit-{unit.key}" class="text-xs">{unit.label}</Label>
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
								<p id="unit-{unit.key}-error" class={ERROR_CLASS}>
									{fieldError('profile', unit.key)}
								</p>
							{/if}
						</div>
					{/each}
				</div>
				<Button type="submit" class="self-start">Save</Button>
			</form>
		</Card>

		<Card role="region" aria-labelledby="password-heading" class="gap-3 p-4">
			<h2 id="password-heading" class="text-base font-semibold">Password</h2>
			{#if form?.action === 'password' && 'changed' in form}
				<Alert role="status">Password changed. Your other sessions were signed out.</Alert>
			{/if}
			<form
				method="POST"
				action="?/password"
				class="flex flex-col gap-3"
				novalidate
				onsubmit={checkPassword}
				use:enhance={clearAfter}
			>
				{#if passwordRefused}<Alert>Check the fields with a message.</Alert>{/if}
				{#if pageError('password')}<Alert>{pageError('password')}</Alert>{/if}
				{#each PASSWORD_FIELDS as [name, label, autocomplete] (name)}
					{@const error = passwordError(name)}
					<div class="flex flex-col gap-1.5">
						<Label for="pw-{name}" class="text-xs">{label}</Label>
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
							<p id="pw-new-help" class="text-xs text-muted-foreground">
								At least {MIN_PASSWORD_LEN} characters
							</p>
						{/if}
						{#if error}<p id="pw-{name}-error" class={ERROR_CLASS}>{error}</p>{/if}
					</div>
				{/each}
				<Button type="submit" variant="outline" class="self-start">Change password</Button>
			</form>
		</Card>
	</div>

	<div class="contents xl:flex xl:flex-col xl:gap-[18px]">
		<Card role="region" aria-labelledby="sessions-heading" class="gap-3 p-4">
			<h2 id="sessions-heading" class="text-base font-semibold">Signed-in devices</h2>
			<p class="text-[13px] leading-normal text-muted-foreground">
				Each row is a device or browser where you’re signed in. Signing one out ends that session
				only.
			</p>
			{#if pageError('signout')}<Alert>{pageError('signout')}</Alert>{/if}
			<ul aria-label="Sessions" class="flex flex-col">
				{#each data.sessions as s (s.id)}
					<li class="flex min-h-[52px] items-center gap-2.5 border-t py-1.5">
						<div class="min-w-0 flex-1">
							<p class="text-[14.5px] font-medium">
								{s.current ? 'This device' : 'Another device'}
							</p>
							<p class="text-[12.5px] text-muted-foreground">
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
			{#if data.sessions.every((s) => s.current)}
				<p class="text-[13.5px] text-muted-foreground">No other devices are signed in.</p>
			{/if}
		</Card>

		<Card role="region" aria-labelledby="children-heading" class="gap-3 p-4">
			<h2 id="children-heading" class="text-base font-semibold">Children you can see</h2>
			{#if data.kids.length}
				<ul aria-label="Children" class="flex flex-col">
					{#each data.kids as k (k.id)}
						<li class="flex min-h-[52px] items-center gap-2.5 border-t">
							<span class="min-w-0 flex-1 text-[14.5px] [overflow-wrap:anywhere]">{k.name}</span>
							<Badge>{k.role === 'guardian' ? 'Guardian' : 'Caregiver'}</Badge>
						</li>
					{/each}
				</ul>
			{:else}
				<p class="text-[13.5px] text-muted-foreground">None yet. A guardian can invite you.</p>
			{/if}
		</Card>
	</div>
</div>
