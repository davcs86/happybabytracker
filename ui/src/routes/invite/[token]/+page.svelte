<script lang="ts">
	import SetPasswordForm from '#lib/components/SetPasswordForm.svelte';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
</script>

<SetPasswordForm
	heading="Set your password"
	description="Choose a password to finish setting up your account."
	valid={data.valid}
	username={data.valid ? data.username : undefined}
	{form}
/>
