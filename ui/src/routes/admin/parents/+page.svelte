<script lang="ts">
	import { onMount } from 'svelte';
	import { enhance } from '$app/forms';
	import { ERROR_CLASS } from '#lib/components/field.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import {
		DISPLAY_NAME_MAX_LEN,
		INVITE_VALIDITY,
		RESET_VALIDITY,
		USERNAME_MAX_LEN,
		USERNAME_MIN_LEN
	} from '#lib/constants.js';
	import type { AdminParent } from '#lib/types.js';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	// The copy button is an enhancement: without JavaScript the link text is selectable.
	let canCopy = $state(false);
	let copied = $state<'yes' | 'no' | null>(null);
	onMount(() => (canCopy = 'clipboard' in navigator));

	const link = $derived(form && 'link' in form ? form.link : null);
	const fieldError = (f: string) =>
		form && 'field' in form && form.field === f ? form.message : null;
	const pageError = $derived(
		form && 'field' in form && !form.field && 'message' in form ? form.message : null
	);
	const values = $derived(form && 'values' in form ? form.values : null);
	const status = (p: AdminParent) =>
		p.disabled ? 'Disabled' : p.hasPassword ? 'Active' : 'No password yet';

	async function copy(url: string) {
		try {
			await navigator.clipboard.writeText(url);
			copied = 'yes';
		} catch {
			copied = 'no';
		}
	}
	$effect(() => {
		void link;
		copied = null;
	});
</script>

<svelte:head>
	<title>Parents · happybabytracker</title>
</svelte:head>

<h1 class="mb-4 text-xl font-semibold">Parents</h1>

{#if link}
	<section
		aria-labelledby="link-heading"
		class="mb-6 flex max-w-xl flex-col gap-2 rounded-lg border p-4"
	>
		<h2 id="link-heading" class="font-semibold">
			{link.kind === 'invite' ? 'Invite' : 'Reset'} link for {link.username}
		</h2>
		<p class="text-sm">
			Send this link to the parent. It works once and expires in {link.kind === 'invite'
				? INVITE_VALIDITY
				: RESET_VALIDITY}
		</p>
		<p
			class="rounded bg-muted p-2 font-mono text-sm break-all select-all"
			data-testid="parent-link"
		>
			{link.url}
		</p>
		{#if canCopy}
			<div class="flex items-center gap-2">
				<Button type="button" variant="outline" onclick={() => copy(link.url)}>Copy link</Button>
				<span aria-live="polite" class="text-sm">
					{copied === 'yes' ? 'Copied' : copied === 'no' ? 'Copy failed, select the link' : ''}
				</span>
			</div>
		{/if}
	</section>
{/if}

{#if pageError}
	<p class="{ERROR_CLASS} mb-4" role="alert">{pageError}</p>
{/if}

<section aria-labelledby="add-heading" class="mb-8 max-w-md">
	<h2 id="add-heading" class="mb-3 font-semibold">Add parent</h2>
	<form method="POST" action="?/create" class="flex flex-col gap-4" novalidate use:enhance>
		<div class="flex flex-col gap-2">
			<Label for="username">Username</Label>
			<Input
				id="username"
				name="username"
				autocomplete="off"
				autocapitalize="none"
				maxlength={USERNAME_MAX_LEN}
				value={values?.username ?? ''}
				required
				aria-invalid={fieldError('username') ? 'true' : undefined}
				aria-describedby="username-help{fieldError('username') ? ' username-error' : ''}"
			/>
			<p id="username-help" class="text-sm text-muted-foreground">
				{USERNAME_MIN_LEN} to {USERNAME_MAX_LEN} characters: a–z, 0–9, dot, underscore or hyphen
			</p>
			{#if fieldError('username')}
				<p id="username-error" class={ERROR_CLASS} role="alert">{fieldError('username')}</p>
			{/if}
		</div>
		<div class="flex flex-col gap-2">
			<Label for="display_name">Display name</Label>
			<Input
				id="display_name"
				name="display_name"
				maxlength={DISPLAY_NAME_MAX_LEN}
				value={values?.displayName ?? ''}
				required
				aria-invalid={fieldError('display_name') ? 'true' : undefined}
				aria-describedby="display_name-help{fieldError('display_name')
					? ' display_name-error'
					: ''}"
			/>
			<p id="display_name-help" class="text-sm text-muted-foreground">
				1 to {DISPLAY_NAME_MAX_LEN} characters
			</p>
			{#if fieldError('display_name')}
				<p id="display_name-error" class={ERROR_CLASS} role="alert">{fieldError('display_name')}</p>
			{/if}
		</div>
		<Button type="submit" class="self-start">Add parent</Button>
	</form>
</section>

<section aria-labelledby="list-heading">
	<h2 id="list-heading" class="mb-3 font-semibold">Accounts</h2>
	<ul class="flex max-w-xl flex-col gap-2">
		{#each data.parents as p (p.id)}
			<li class="flex flex-col gap-2 rounded-lg border p-3">
				<p>
					<span class="font-medium">{p.username}</span>
					<span class="text-sm text-muted-foreground">{p.displayName}</span>
					· <span class="text-sm">{status(p)}</span>
				</p>
				<div class="flex flex-wrap gap-2">
					<form method="POST" action="?/link" use:enhance>
						<input type="hidden" name="parent_id" value={p.id} />
						<input type="hidden" name="username" value={p.username} />
						<Button type="submit" variant="outline"
							>Reset link<span class="sr-only"> for {p.username}</span></Button
						>
					</form>
					<form method="POST" action={p.disabled ? '?/enable' : '?/disable'} use:enhance>
						<input type="hidden" name="parent_id" value={p.id} />
						<Button type="submit" variant="outline"
							>{p.disabled ? 'Enable' : 'Disable'}<span class="sr-only"> {p.username}</span></Button
						>
					</form>
				</div>
			</li>
		{/each}
	</ul>
</section>
