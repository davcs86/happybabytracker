<script lang="ts">
	import HouseIcon from '@lucide/svelte/icons/house';
	import UserIcon from '@lucide/svelte/icons/user';
	import AppShell, { type NavLink } from '#lib/components/AppShell.svelte';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();

	const links: NavLink[] = [
		{ href: '/', label: 'Home', icon: HouseIcon },
		{ href: '/account', label: 'Account', icon: UserIcon }
	];
</script>

<AppShell {links}>{@render children()}</AppShell>
