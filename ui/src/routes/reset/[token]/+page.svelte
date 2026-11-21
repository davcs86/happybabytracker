<script lang="ts">
	import SetPasswordForm from '#lib/components/SetPasswordForm.svelte';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
</script>

<SetPasswordForm
	heading="Set a new password"
	description="Choose a new password. Your other devices will be signed out."
	valid={data.valid}
	username={data.valid ? data.username : undefined}
	{form}
/>
