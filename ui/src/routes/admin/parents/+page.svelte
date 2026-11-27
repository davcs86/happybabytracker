<script lang="ts">
	import { onMount } from 'svelte';
	import { enhance } from '$app/forms';
	import { page } from '$app/state';
	import CheckIcon from '@lucide/svelte/icons/check';
	import { ERROR_CLASS } from '#lib/components/field.js';
	import { Alert } from '#lib/components/ui/alert/index.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button, buttonVariants } from '#lib/components/ui/button/index.js';
	import { Card } from '#lib/components/ui/card/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import { toggleVariants } from '#lib/components/ui/toggle/index.js';
	import {
		DISPLAY_NAME_MAX_LEN,
		INVITE_VALIDITY,
		RESET_VALIDITY,
		USERNAME_MAX_LEN,
		USERNAME_MIN_LEN
	} from '#lib/constants.js';
	import { lastActiveLine, linkLine } from '#lib/admin-lines.js';
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
	type Status = 'active' | 'invited' | 'disabled';
	const STATUS_LABEL: Record<Status, string> = {
		active: 'Active',
		invited: 'Invited',
		disabled: 'Disabled'
	};
	const FILTERS: [string, string][] = [
		['all', 'All'],
		['active', 'Active'],
		['invited', 'Invited'],
		['disabled', 'Disabled']
	];
	// The activity and link lines are read against this clock.
	const now = new Date();
	const status = (p: AdminParent): Status =>
		p.disabled ? 'disabled' : p.hasPassword ? 'active' : 'invited';
	const count = (s: Status) => data.parents.filter((p) => status(p) === s).length;
	const counts = $derived(
		[
			`${data.parents.length} ${data.parents.length === 1 ? 'parent' : 'parents'}`,
			`${count('active')} active`,
			...(count('invited') ? [`${count('invited')} invited`] : []),
			...(count('disabled') ? [`${count('disabled')} disabled`] : [])
		].join(' · ')
	);
	const shown = $derived.by(() => {
		const f = page.url.searchParams.get('show');
		return FILTERS.some(([k]) => k === f) ? f : 'all';
	});
	const rows = $derived(data.parents.filter((p) => shown === 'all' || status(p) === shown));

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

<!-- eslint-disable svelte/no-navigation-without-resolve -- the filter is a query on this page -->
<div
	class="mt-6.5 flex max-w-[720px] flex-col gap-3.5 xl:mx-auto xl:grid xl:w-[920px] xl:max-w-none xl:grid-cols-[448px_448px] xl:items-start xl:gap-x-6 xl:gap-y-[18px]"
>
	<div class="flex flex-wrap items-center gap-x-2 gap-y-3.5 xl:col-span-2">
		<h1 class="min-w-0 flex-1">Parents</h1>
		<details class="peer" open={!!values}>
			<summary
				class="{buttonVariants()} cursor-pointer list-none [&::-webkit-details-marker]:hidden"
				>Create parent</summary
			>
		</details>
		<form
			method="POST"
			action="?/create"
			class="hidden basis-full flex-col gap-3 rounded-2xl border bg-card p-4 peer-open:flex"
			novalidate
			use:enhance
		>
			{#if fieldError('username') || fieldError('display_name')}
				<Alert>Check the fields with a message.</Alert>
			{/if}
			<div class="flex flex-col gap-1.5">
				<Label for="username" class="text-xs">Username</Label>
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
				<p id="username-help" class="text-xs text-muted-foreground">
					{USERNAME_MIN_LEN} to {USERNAME_MAX_LEN} characters: a–z, 0–9, dot, underscore or hyphen
				</p>
				{#if fieldError('username')}
					<p id="username-error" class={ERROR_CLASS}>{fieldError('username')}</p>
				{/if}
			</div>
			<div class="flex flex-col gap-1.5">
				<Label for="display_name" class="text-xs">Display name</Label>
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
				<p id="display_name-help" class="text-xs text-muted-foreground">
					1 to {DISPLAY_NAME_MAX_LEN} characters
				</p>
				{#if fieldError('display_name')}
					<p id="display_name-error" class={ERROR_CLASS}>{fieldError('display_name')}</p>
				{/if}
			</div>
			<Button type="submit">Create parent</Button>
		</form>
	</div>

	<div class="-mt-1.5 flex flex-col gap-1.5 xl:col-span-2">
		<div aria-hidden="true" class="border-t"></div>
		<p class="text-sm text-muted-foreground">{counts}</p>
	</div>

	<Alert role="note" class="xl:col-span-2">
		<p>
			<b>What this screen can do.</b> Create parents, issue a one-time invite or reset link, and disable
			or enable parents. It never shows children, their names or any count of them.
		</p>
	</Alert>

	{#if pageError}<Alert class="xl:col-span-2">{pageError}</Alert>{/if}

	{#if link}
		<Card role="region" aria-labelledby="link-heading" class="gap-3 p-4 xl:col-span-2">
			<h2 id="link-heading" class="text-[18px] font-semibold [overflow-wrap:anywhere]">
				{link.kind === 'invite' ? 'Invite' : 'Reset'} link for {link.username}
			</h2>
			<p class="text-sm leading-normal">
				Send this link to the parent. It works once and expires in {link.kind === 'invite'
					? INVITE_VALIDITY
					: RESET_VALIDITY}. Copy it now and hand it over by a channel you trust; it is shown only
				once.
			</p>
			<div class="flex flex-col gap-1.5">
				<Label for="one-time-link" class="text-xs">One-time link</Label>
				<Input
					id="one-time-link"
					readonly
					value={link.url}
					autocomplete="off"
					spellcheck={false}
					data-testid="parent-link"
					class="font-mono text-[13px]"
					onfocus={(e) => e.currentTarget.select()}
				/>
			</div>
			{#if canCopy}
				<div class="flex items-center gap-2">
					<Button type="button" variant="outline" onclick={() => copy(link.url)}>Copy link</Button>
					<span aria-live="polite" class="text-sm">
						{copied === 'yes' ? 'Copied' : copied === 'no' ? 'Copy failed, select the link' : ''}
					</span>
				</div>
			{/if}
		</Card>
	{/if}

	<div class="contents xl:flex xl:flex-col xl:gap-[18px]">
		<div role="group" aria-label="Show" class="flex flex-wrap gap-2">
			{#each FILTERS as [key, label] (key)}
				<a
					href="?show={key}"
					aria-current={shown === key ? 'true' : undefined}
					class="{toggleVariants()} aria-[current=true]:border-primary aria-[current=true]:bg-accent aria-[current=true]:font-semibold"
				>
					{#if shown === key}<CheckIcon aria-hidden="true" class="size-3.5" />{/if}{label}
				</a>
			{/each}
		</div>
		{#if rows.length === 0}
			<div class="rounded-2xl border border-dashed p-5 text-sm text-muted-foreground">
				Nobody matches this filter.
			</div>
		{/if}
		<ul class="flex flex-col gap-3.5 xl:gap-[18px]">
			{#each rows as p (p.id)}
				{@const s = status(p)}
				<li class="flex flex-col gap-2 rounded-2xl border bg-card p-4">
					<div class="flex flex-wrap items-center gap-2">
						<p class="text-base font-semibold [overflow-wrap:anywhere]">{p.displayName}</p>
						<Badge
							class={s === 'disabled'
								? 'border-[1.5px] border-dashed border-foreground text-foreground'
								: ''}>{STATUS_LABEL[s]}</Badge
						>
					</div>
					<p class="font-mono text-[13px] [overflow-wrap:anywhere] text-muted-foreground">
						{p.username}
					</p>
					<p class="text-[13px] text-muted-foreground">{lastActiveLine(p, now)}</p>
					{#if p.link}
						<p class="text-[13px] [overflow-wrap:anywhere] text-muted-foreground">
							{linkLine(p.link, now)}
						</p>
					{/if}
					<div class="flex flex-wrap items-start gap-2">
						{#if s !== 'disabled'}
							<form method="POST" action="?/link" use:enhance>
								<input type="hidden" name="parent_id" value={p.id} />
								<input type="hidden" name="username" value={p.username} />
								<Button type="submit" variant="outline"
									>New {s === 'invited' ? 'invite' : 'reset'} link<span class="sr-only">
										for {p.displayName}</span
									></Button
								>
							</form>
							<!-- Two steps without JavaScript: the button exists only inside the opened details. -->
							<details class="open:w-full">
								<summary
									class="{buttonVariants({
										variant: 'outline'
									})} w-fit cursor-pointer list-none [&::-webkit-details-marker]:hidden"
									>Disable<span class="sr-only"> {p.displayName}</span></summary
								>
								<form
									method="POST"
									action="?/disable"
									class="mt-2 flex flex-col gap-3 rounded-xl border-[1.5px] border-destructive bg-card p-3"
									use:enhance
								>
									<input type="hidden" name="parent_id" value={p.id} />
									<ul class="list-disc pl-5 text-sm leading-relaxed [overflow-wrap:anywhere]">
										<li>{p.displayName} is signed out everywhere and can’t sign in.</li>
										<li>Entries they logged stay and keep their name.</li>
										<li>You can enable the account again any time.</li>
									</ul>
									<Button type="submit" variant="destructive"
										>Disable<span class="sr-only"> {p.displayName}</span></Button
									>
								</form>
							</details>
						{:else}
							<form method="POST" action="?/enable" use:enhance>
								<input type="hidden" name="parent_id" value={p.id} />
								<Button type="submit" variant="outline"
									>Enable<span class="sr-only"> {p.displayName}</span></Button
								>
							</form>
						{/if}
					</div>
				</li>
			{/each}
		</ul>
	</div>

	<div class="contents xl:flex xl:flex-col xl:gap-[18px]">
		<p class="text-[12.5px] leading-normal text-muted-foreground">
			The admin’s own username and password can’t be changed here. They come from the deployment;
			changing them signs the admin out.
		</p>
		<form method="POST" action="/logout">
			<Button type="submit" variant="outline" class="w-full">Sign out of admin</Button>
		</form>
	</div>
</div>
