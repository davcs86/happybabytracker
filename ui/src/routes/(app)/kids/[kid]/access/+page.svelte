<script lang="ts">
	import { enhance } from '$app/forms';
	import { ERROR_CLASS } from '#lib/components/field.js';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import { savedAs } from '#lib/drawer.js';
	import FormFrame from '#lib/components/FormFrame.svelte';
	import type { Role } from '#lib/types.js';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	const ROLES: [Role, string][] = [
		['caregiver', 'Caregiver'],
		['guardian', 'Guardian']
	];
	const roleLabel = (r: Role) => (r === 'guardian' ? 'Guardian' : 'Caregiver');
	const fieldError = (f: string) =>
		form?.action === 'add' && 'field' in form && form.field === f ? form.message : null;
	const pageError = $derived(
		form && 'message' in form && (form.action !== 'add' || !form.field) ? form.message : null
	);
	const onlyGuardian = $derived(data.grants.filter((g) => g.role === 'guardian').length === 1);
	const CARD = 'flex flex-col gap-3 rounded-2xl border bg-card p-4';
</script>

<FormFrame
	class="mt-5 flex flex-col gap-[18px] xl:grid xl:grid-cols-[448px_448px] xl:items-start xl:gap-x-6"
	documentTitle="Access · happybabytracker"
>
	{#snippet chrome()}
		<div class="flex flex-col gap-1.5 xl:col-span-2">
			<h1>Access</h1>
			<p class="text-sm leading-snug [overflow-wrap:anywhere] text-muted-foreground">
				Who can see {data.kid.name}
			</p>
		</div>
	{/snippet}
	{#if pageError}
		<Alert.Root class="xl:col-span-2">{pageError}</Alert.Root>
	{/if}

	<div class="contents xl:block">
		<section aria-labelledby="people-heading" class={CARD}>
			<h2 id="people-heading" class="text-base font-semibold">People with access</h2>
			<p class="text-[13px] leading-normal [overflow-wrap:anywhere] text-muted-foreground">
				Guardians can change who sees {data.kid.name}. Caregivers can log, view and export. A child
				always keeps at least one active guardian.
			</p>
			<ul aria-label="People with access" class="flex flex-col">
				{#each data.grants as g (g.parentId)}
					{@const self = g.parentId === data.me}
					{@const target = g.role === 'guardian' ? 'caregiver' : 'guardian'}
					<li class="flex flex-col gap-2 border-t py-3 last:pb-0">
						<div class="flex flex-wrap items-center gap-2">
							<span class="font-medium [overflow-wrap:anywhere]">{g.displayName}</span>
							<Badge>{roleLabel(g.role)}</Badge>
							{#if self}<Badge>You</Badge>{/if}
						</div>
						<span class="font-mono text-[12.5px] [overflow-wrap:anywhere] text-muted-foreground"
							>{g.username}</span
						>
						{#if self}
							{#if onlyGuardian && g.role === 'guardian'}
								<p class="text-[13px] leading-normal text-muted-foreground">
									You are {data.kid.name}’s only active guardian. Give another person guardian
									access before you leave or change your role.
								</p>
							{/if}
							<form
								method="POST"
								action="?/remove"
								use:enhance={savedAs('You left', { follow: true, stay: true })}
							>
								<input type="hidden" name="parent_id" value={g.parentId} />
								<Button type="submit" variant="outline">Leave</Button>
							</form>
						{:else}
							<div class="flex flex-wrap gap-2">
								<form
									method="POST"
									action="?/role"
									use:enhance={savedAs('Role changed', { stay: true })}
								>
									<input type="hidden" name="username" value={g.username} />
									<input type="hidden" name="role" value={target} />
									<Button type="submit" variant="outline">
										<span aria-hidden="true">Make {target}</span>
										<span class="sr-only">Make {g.displayName} a {target}</span>
									</Button>
								</form>
								<form
									method="POST"
									action="?/remove"
									use:enhance={savedAs('Access removed', { stay: true })}
								>
									<input type="hidden" name="parent_id" value={g.parentId} />
									<Button type="submit" variant="outline">
										<span aria-hidden="true">Remove access</span>
										<span class="sr-only">Remove {g.displayName}’s access to {data.kid.name}</span>
									</Button>
								</form>
							</div>
						{/if}
					</li>
				{/each}
			</ul>
		</section>
	</div>

	<div class="contents xl:block">
		<section aria-labelledby="add-heading" class={CARD}>
			<h2 id="add-heading" class="text-base font-semibold">Add a person</h2>
			<p class="text-[13px] leading-normal [overflow-wrap:anywhere] text-muted-foreground">
				Give an existing parent access to {data.kid.name}. Parent accounts are created by the admin.
				A guardian can change the role later.
			</p>
			<form
				method="POST"
				action="?/add"
				class="flex flex-col gap-4"
				novalidate
				use:enhance={savedAs('Access added', { stay: true })}
			>
				<div class="grid gap-1.5">
					<Label for="add-username">Parent’s username</Label>
					<Input
						id="add-username"
						name="username"
						autocomplete="off"
						autocapitalize="none"
						value={form?.action === 'add' && 'username' in form ? form.username : ''}
						required
						aria-invalid={fieldError('username') ? 'true' : undefined}
						aria-describedby={fieldError('username') ? 'add-username-error' : undefined}
					/>
					{#if fieldError('username')}
						<p id="add-username-error" class={ERROR_CLASS}>{fieldError('username')}</p>
					{/if}
				</div>
				<fieldset
					role="radiogroup"
					aria-labelledby="add-role-legend"
					class="flex flex-col"
					aria-invalid={fieldError('role') ? 'true' : undefined}
					aria-describedby={fieldError('role') ? 'add-role-error' : undefined}
				>
					<legend id="add-role-legend" class="mb-1 p-0 text-[13px] text-muted-foreground"
						>Role</legend
					>
					{#each ROLES as [value, label] (value)}
						<label class="flex min-h-11 items-center gap-3 text-[15px]">
							<input
								type="radio"
								name="role"
								{value}
								checked={value === 'caregiver'}
								class="size-5 shrink-0 accent-primary"
							/>
							{label}
						</label>
					{/each}
					{#if fieldError('role')}
						<p id="add-role-error" class="{ERROR_CLASS} mt-1">{fieldError('role')}</p>
					{/if}
				</fieldset>
				<Button type="submit">Give access</Button>
			</form>
		</section>
	</div>
</FormFrame>
