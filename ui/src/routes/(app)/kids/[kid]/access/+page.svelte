<script lang="ts">
	import { resolve } from '$app/paths';
	import { enhance } from '$app/forms';
	import { ERROR_CLASS, SELECT_CLASS } from '#lib/components/field.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
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
</script>

<svelte:head>
	<title>Access to {data.kid.name} · happybabytracker</title>
</svelte:head>

<p class="mb-2 text-sm">
	<a href={resolve('/(app)/kids/[kid]', { kid: data.kid.id })} class="underline">{data.kid.name}</a>
</p>
<h1 class="mb-4 text-xl font-semibold">Access to {data.kid.name}</h1>

{#if pageError}
	<p class="{ERROR_CLASS} mb-4" role="alert">{pageError}</p>
{/if}

<ul aria-label="People with access" class="flex max-w-xl flex-col gap-3">
	{#each data.grants as g (g.parentId)}
		{@const self = g.parentId === data.me}
		<li class="flex flex-col gap-2 rounded-lg border p-3">
			<p>
				<span class="font-medium">{g.displayName}</span>
				<span class="text-sm text-muted-foreground">{g.username}{self ? ' (you)' : ''}</span>
				· <span class="text-sm">{roleLabel(g.role)}</span>
			</p>
			<div class="flex flex-wrap items-center gap-2">
				{#if self}
					<form method="POST" action="?/remove" use:enhance>
						<input type="hidden" name="parent_id" value={g.parentId} />
						<Button type="submit" variant="outline" size="sm">Leave</Button>
					</form>
				{:else}
					<form method="POST" action="?/role" class="flex items-center gap-2" use:enhance>
						<input type="hidden" name="username" value={g.username} />
						<label for="role-{g.parentId}" class="sr-only">Role for {g.displayName}</label>
						<select id="role-{g.parentId}" name="role" class="{SELECT_CLASS} w-auto" value={g.role}>
							{#each ROLES as [value, label] (value)}
								<option {value}>{label}</option>
							{/each}
						</select>
						<Button type="submit" variant="outline" size="sm"
							>Change role<span class="sr-only"> for {g.displayName}</span></Button
						>
					</form>
					<form method="POST" action="?/remove" use:enhance>
						<input type="hidden" name="parent_id" value={g.parentId} />
						<Button type="submit" variant="outline" size="sm"
							>Remove<span class="sr-only"> {g.displayName}</span></Button
						>
					</form>
				{/if}
			</div>
		</li>
	{/each}
</ul>

<section aria-labelledby="add-heading" class="mt-8 max-w-md">
	<h2 id="add-heading" class="mb-3 font-semibold">Add someone</h2>
	<form method="POST" action="?/add" class="flex flex-col gap-4" novalidate use:enhance>
		<div class="flex flex-col gap-2">
			<Label for="add-username">Username</Label>
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
				<p id="add-username-error" class={ERROR_CLASS} role="alert">{fieldError('username')}</p>
			{/if}
		</div>
		<div class="flex flex-col gap-2">
			<Label for="add-role">Role</Label>
			<select
				id="add-role"
				name="role"
				class={SELECT_CLASS}
				aria-invalid={fieldError('role') ? 'true' : undefined}
				aria-describedby={fieldError('role') ? 'add-role-error' : undefined}
			>
				{#each ROLES as [value, label] (value)}
					<option {value}>{label}</option>
				{/each}
			</select>
			{#if fieldError('role')}
				<p id="add-role-error" class={ERROR_CLASS} role="alert">{fieldError('role')}</p>
			{/if}
		</div>
		<Button type="submit" class="self-start">Add</Button>
	</form>
</section>
