<script lang="ts">
	import { INVITE_VALIDITY } from '#lib/constants.js';
	import SetPasswordForm from '#lib/components/SetPasswordForm.svelte';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	// Without relying-party settings an oidc invite has no way to finish.
	const mode = $derived(
		data.mode === 'local'
			? 'password'
			: data.mode === 'both'
				? data.sso
					? 'both'
					: 'password'
				: data.sso
					? 'sso'
					: 'off'
	);
</script>

<SetPasswordForm
	heading={mode === 'sso' ? 'Finish setting up your account' : 'Set your password'}
	description={mode === 'sso'
		? 'Continue with single sign-on to finish.'
		: 'Choose a password to finish setting up your account.'}
	valid={data.valid}
	lifetime={INVITE_VALIDITY}
	username={data.valid ? data.username : undefined}
	{form}
	action="?/password"
	{mode}
/>
