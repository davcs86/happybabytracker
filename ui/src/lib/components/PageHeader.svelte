<script lang="ts">
	import { page } from '$app/state';
	import BellIcon from '@lucide/svelte/icons/bell';
	import ChevronDownIcon from '@lucide/svelte/icons/chevron-down';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { buttonVariants } from '#lib/components/ui/button/index.js';
	import { switchTarget } from '#lib/switch-target.js';
	import type { Kid } from '#lib/types.js';

	let { kid, onswitch }: { kid: Pick<Kid, 'name' | 'role'> | null; onswitch: () => void } =
		$props();

	const bellHere = $derived(page.url.pathname === '/account/notifications');
</script>

<!-- eslint-disable svelte/no-navigation-without-resolve -- fixed paths of the app -->
<header class="flex min-h-12 items-center gap-2">
	{#if kid}
		<a
			href="/kids?to={switchTarget(page.url.pathname)}"
			aria-label="Child: {kid.name}. Switch child"
			class="{buttonVariants({ variant: 'outline' })} min-w-0 gap-2 px-3"
			onclick={(e) => {
				if (e.metaKey || e.ctrlKey || e.shiftKey || e.button !== 0) return;
				e.preventDefault();
				onswitch();
			}}
		>
			<span
				aria-hidden="true"
				data-initial={[...kid.name][0]?.toUpperCase()}
				class="grid size-7 shrink-0 place-items-center rounded-full bg-accent text-sm font-semibold text-primary-ink after:content-[attr(data-initial)]"
			></span>
			<span class="min-w-0 truncate">{kid.name}</span>
			<ChevronDownIcon aria-hidden="true" data-chrome class="size-4 shrink-0" />
		</a>
	{/if}
	<div class="ml-auto flex items-center gap-2">
		<a
			href="/account/notifications"
			aria-label="Notification settings"
			aria-current={bellHere ? 'page' : undefined}
			class="{buttonVariants({ variant: 'outline', size: 'icon' })} {bellHere
				? 'border-primary text-primary'
				: ''}"
		>
			<BellIcon aria-hidden="true" data-chrome class="size-5" />
		</a>
		{#if kid}<Badge>{kid.role === 'guardian' ? 'Guardian' : 'Caregiver'}</Badge>{/if}
	</div>
</header>
