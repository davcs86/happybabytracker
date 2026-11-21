<script lang="ts">
	import UsersIcon from '@lucide/svelte/icons/users';
	import AppShell, { type NavLink } from '#lib/components/AppShell.svelte';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();

	const links: NavLink[] = [{ href: '/admin/parents', label: 'Parents', icon: UsersIcon }];
</script>

<AppShell {links}>{@render children()}</AppShell>
