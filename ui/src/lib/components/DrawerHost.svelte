<script lang="ts">
	import { page } from '$app/state';
	import { drawerComponent, openDrawer } from '#lib/drawer.js';
	import FormSheet from './FormSheet.svelte';

	const drawer = $derived(page.state.drawer);

	// A plain left click on a link marked data-drawer opens its page in a sheet; every other click navigates.
	async function onclick(e: MouseEvent) {
		if (e.defaultPrevented || e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) {
			return;
		}
		const a = (e.target as Element).closest<HTMLAnchorElement>('a[data-drawer]');
		if (!a || a.target) return;
		e.preventDefault();
		if (!(await openDrawer(a))) location.assign(a.href);
	}
</script>

<svelte:window onclickcapture={onclick} />

{#if drawer}
	{#await drawerComponent(drawer.href) then Page}
		<FormSheet title={drawer.title}>
			{#if Page}<Page data={drawer.data} form={page.form} />{/if}
		</FormSheet>
	{/await}
{/if}
