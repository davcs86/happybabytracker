<script lang="ts">
	import HouseIcon from '@lucide/svelte/icons/house';
	import ListIcon from '@lucide/svelte/icons/list';
	import UserIcon from '@lucide/svelte/icons/user';
	import AppShell, { type NavLink } from '#lib/components/AppShell.svelte';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();

	// Home is the kid home, /kids/{id}; /kids/new is the add form.
	const links: NavLink[] = [
		{ href: '/', label: 'Home', icon: HouseIcon, match: /^\/(kids\/(?!new$)[^/]+)?$/ },
		{
			href: '/timeline',
			label: 'Timeline',
			icon: ListIcon,
			match: /^\/kids\/[^/]+\/timeline(\/|$)/
		},
		{ href: '/account', label: 'Account', icon: UserIcon }
	];
</script>

<AppShell {links}>{@render children()}</AppShell>
