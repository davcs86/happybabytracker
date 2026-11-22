<script lang="ts">
	import ChartLineIcon from '@lucide/svelte/icons/chart-line';
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
		{
			href: '/growth',
			label: 'Growth',
			icon: ChartLineIcon,
			match: /^\/kids\/[^/]+\/growth(\/|$)/
		},
		{ href: '/account', label: 'Account', icon: UserIcon }
	];
</script>

<AppShell {links}>{@render children()}</AppShell>
