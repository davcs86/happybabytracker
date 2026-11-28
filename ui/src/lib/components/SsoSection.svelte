<script lang="ts" module>
	import type { AuthMode } from '#lib/types.js';

	export type SsoSectionState = 'linkable' | 'linked' | 'off';

	// Parents cannot unlink: a linked account only says so; linking needs both modes and settings.
	export function ssoSectionState(d: {
		mode: AuthMode;
		sso: boolean;
		linked: boolean;
	}): SsoSectionState | null {
		if (d.linked) return 'linked';
		return d.mode === 'both' && d.sso ? 'linkable' : null;
	}
</script>

<script lang="ts">
	import { ERROR_CLASS } from '#lib/components/field.js';
	import { Alert } from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Card } from '#lib/components/ui/card/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';

	let {
		action,
		state,
		form
	}: {
		action: string;
		state: SsoSectionState;
		form: { field?: string; message?: string } | null;
	} = $props();

	const fieldError = $derived(form?.field === 'password' ? (form.message ?? null) : null);
	const sectionError = $derived(form && !form.field ? (form.message ?? null) : null);
</script>

<Card role="region" aria-labelledby="sso-heading" class="gap-3 p-4">
	<h2 id="sso-heading" class="text-base font-semibold">Single sign-on</h2>
	{#if state === 'linked'}
		<p class="text-[13.5px] text-muted-foreground">Single sign-on is linked.</p>
	{:else if state === 'off'}
		<Alert>{sectionError ?? 'Linking is turned off'}</Alert>
	{:else}
		<!-- A plain POST: success is a redirect to the issuer. -->
		<form method="POST" {action} class="flex flex-col gap-3" novalidate>
			{#if sectionError}<Alert>{sectionError}</Alert>{/if}
			<div class="flex flex-col gap-1.5">
				<Label for="sso-password" class="text-xs">Password</Label>
				<Input
					id="sso-password"
					name="password"
					type="password"
					autocomplete="current-password"
					required
					aria-invalid={fieldError ? 'true' : undefined}
					aria-describedby={fieldError ? 'sso-password-error' : undefined}
				/>
				{#if fieldError}<p id="sso-password-error" class={ERROR_CLASS}>{fieldError}</p>{/if}
			</div>
			<Button type="submit" variant="outline" class="self-start">Link single sign-on</Button>
		</form>
	{/if}
</Card>
