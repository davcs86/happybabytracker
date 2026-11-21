<script lang="ts">
	import { enhance } from '$app/forms';
	import { ERROR_CLASS } from '#lib/components/field.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import KidLinks from '#lib/components/KidLinks.svelte';
	import KidSwitcher from '#lib/components/KidSwitcher.svelte';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
</script>

<svelte:head>
	<title>{data.kid.name} · happybabytracker</title>
</svelte:head>

<KidSwitcher kids={data.kids} currentId={data.kid.id} />
<h1 class="text-xl font-semibold">{data.kid.name}</h1>
<p class="text-muted-foreground">{data.age}</p>
<p class="text-sm text-muted-foreground">Born {data.kid.birthDate}</p>
<KidLinks kid={data.kid} />

{#if data.kid.role === 'caregiver'}
	<!-- Two steps without JavaScript: the button exists only inside the opened details. -->
	<details class="mt-6 max-w-md rounded-lg border p-3">
		<summary class="cursor-pointer text-sm font-medium">Leave this child</summary>
		<form method="POST" action="?/leave" class="mt-3 flex flex-col gap-3" use:enhance>
			{#if form?.message}<p class={ERROR_CLASS} role="alert">{form.message}</p>{/if}
			<p class="text-sm">
				You will lose access to {data.kid.name}. A guardian has to add you again.
			</p>
			<Button type="submit" variant="outline" class="self-start border-destructive text-destructive"
				>Yes, leave {data.kid.name}</Button
			>
		</form>
	</details>
{/if}
