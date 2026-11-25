<script lang="ts">
	import type { Snippet } from 'svelte';
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import Field from '#lib/components/entry/Field.svelte';
	import TrackerFields from '#lib/components/entry/TrackerFields.svelte';
	import FormCancel from '#lib/components/FormCancel.svelte';
	import FormFrame from '#lib/components/FormFrame.svelte';
	import { Alert } from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { isEmbed, savedAs } from '#lib/drawer.js';
	import type { EntryStage, EntryValues } from '#lib/entry-edit.js';
	import type { TrackerKind } from '#lib/types.js';

	let {
		kid,
		kind,
		stage,
		title,
		age,
		zone,
		values,
		error,
		estimate = null,
		message = null,
		refused = false,
		running = false,
		notice = null,
		toggle = null,
		footer
	}: {
		kid: { id: string; name: string };
		kind: TrackerKind;
		stage: EntryStage;
		title: string;
		age: string;
		zone: string;
		values: EntryValues;
		error: (name: string) => string | null;
		// The edit page's paragraph under the weigh-ins (milkStatus).
		estimate?: string | null;
		// A failure that belongs to no field; running adds the way to the timer that blocks this one.
		message?: string | null;
		// A field was refused: the summary points at the messages under the fields.
		refused?: boolean;
		running?: boolean;
		notice?: string | null;
		// The plain link that reloads the form advanced or simple; values.advanced says which it is.
		toggle?: { href: string; label: string } | null;
		footer?: Snippet;
	} = $props();

	const embed = isEmbed();
	const home = $derived(resolve('/(app)/kids/[kid]', { kid: kid.id }));
	const GROUP = 'flex min-w-0 flex-col gap-3.5';
</script>

{#snippet zoneNote()}
	<p class="mt-1.5 text-[13.5px] leading-normal [overflow-wrap:anywhere] text-muted-foreground">
		Times are in {zone}
	</p>
{/snippet}

<FormFrame documentTitle="{title} · {kid.name} · happybabytracker">
	{#snippet chrome()}
		<h1 class="mt-[26px]">{title}</h1>
		<p class="mt-1.5 text-sm leading-[1.4] text-muted-foreground">{age}</p>
		{@render zoneNote()}
	{/snippet}
	{#if embed}{@render zoneNote()}{/if}
	{#if notice}
		<p class="mt-4 text-[13.5px] leading-normal [overflow-wrap:anywhere]">{notice}</p>
	{/if}

	<form
		method="POST"
		action="?/save"
		class="group/entry mt-4 flex flex-col gap-3.5"
		novalidate
		use:enhance={savedAs(stage === 'new' ? 'Entry added' : 'Entry saved')}
	>
		{#if toggle}
			<!-- eslint-disable svelte/no-navigation-without-resolve -- a query on this same page -->
			<p class="text-sm">
				<a href={toggle.href} data-drawer={embed ? title : undefined} class="underline"
					>{toggle.label}</a
				>
			</p>
			<!-- eslint-enable svelte/no-navigation-without-resolve -->
		{/if}
		{#if values.advanced === '1'}
			<input type="hidden" name="advanced" value="1" />
		{/if}
		{#if message}
			<Alert>
				<p class="[overflow-wrap:anywhere]">
					{message}{#if running}.
						<a href={home} class="underline">See {kid.name}'s timer</a>{/if}
				</p>
			</Alert>
		{:else if refused}
			<Alert><p>Check the fields with a message.</p></Alert>
		{/if}
		<div
			class={[
				'contents',
				!embed &&
					'xl:grid xl:grid-cols-[448px_448px] xl:grid-rows-[auto_1fr] xl:items-start xl:gap-x-6 xl:gap-y-[18px]'
			]}
		>
			<div class={GROUP}>
				<TrackerFields {kind} {stage} part="primary" {values} {error} {estimate} />
			</div>
			<div class="{GROUP} xl:row-span-2">
				<TrackerFields {kind} {stage} part="optional" {values} {error} {estimate} />
				<Field id="entry-note" label="Note" error={error('note')}>
					{#snippet children(control)}
						<Input {...control} name="note" autocomplete="off" value={values.note as string} />
					{/snippet}
				</Field>
			</div>
			<div class="flex gap-2">
				<Button type="submit" size={embed ? 'lg' : 'default'} class="flex-1">Save</Button>
				<FormCancel href={home} size="default" class="" />
			</div>
		</div>
	</form>
	{@render footer?.()}
</FormFrame>
