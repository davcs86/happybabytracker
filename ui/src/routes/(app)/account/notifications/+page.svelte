<script lang="ts">
	import { onMount } from 'svelte';
	import { enhance } from '$app/forms';
	import { invalidateAll } from '$app/navigation';
	import NotificationDevice from '#lib/components/NotificationDevice.svelte';
	import { keepValues } from '#lib/components/field.js';
	import { Alert } from '#lib/components/ui/alert/index.js';
	import { Button, buttonVariants } from '#lib/components/ui/button/index.js';
	import { Card } from '#lib/components/ui/card/index.js';
	import { PUSH_OFF_MESSAGE } from '#lib/constants.js';
	import { notificationsSubscribePath } from '#lib/paths.js';
	import {
		base64UrlToBytes,
		isIosOutsideStandalone,
		isOtherKey,
		subscriptionBody
	} from '#lib/push.js';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	const IOS_SENTENCE =
		'On iPhone and iPad, add this app to your Home Screen first, then open it from there to turn on notifications';
	const NTFY_SENTENCE =
		'ntfy keeps each message for up to 72 hours on its server, so your child’s name is stored there in plain text for that long';
	const BLOCKED =
		'Notifications are blocked for this site. Allow them in the browser’s settings, then try again.';
	const NOT_TURNED_ON = 'Notifications could not be turned on. Try again.';
	const DISMISSED = 'Notifications were not turned on. You can try again.';
	const OTHER_KEY = 'Turn notifications off and on again on this device.';
	const RESEND_FAILED =
		'This device could not be checked with the server. Reload the page to try again.';
	const SIGN_OUT_SENTENCE =
		'Signing out of this app does not stop notifications on this phone or browser: turn them off first with Turn off ntfy or Turn off on this device.';
	const PUSH_SIGN_OUT_SENTENCE =
		'Signing out of this app does not stop notifications on this phone or browser: turn them off first with Turn off on this device.';

	// pending until mounted: the enable button exists only once JavaScript runs.
	let support = $state<'pending' | 'ready' | 'ios' | 'unsupported'>('pending');
	let busy = $state(false);
	let problem = $state<string | null>(null);

	const current = $derived(data.devices.find((d) => d.id === data.currentDeviceId) ?? null);
	const others = $derived(data.devices.filter((d) => d.id !== data.currentDeviceId));

	const saved = $derived(
		(form?.action === 'device' || form?.action === 'ntfy_kids') && form && 'saved' in form
	);
	const pageError = $derived(
		form && 'field' in form && (!form.field || form.field === 'kid') && 'message' in form
			? form.message
			: null
	);
	const regenerateFailed = $derived(
		form?.action === 'ntfy_regenerate' && 'message' in form && !!form.message
	);
	const deviceForm = $derived(
		form?.action === 'device' && 'values' in form && form.device === current?.id ? form : null
	);

	const standalone = () =>
		matchMedia('(display-mode: standalone)').matches ||
		(navigator as Navigator & { standalone?: boolean }).standalone === true;
	const pushSupported = () =>
		'serviceWorker' in navigator && 'PushManager' in window && 'Notification' in window;

	// The endpoint and keys go only into this request's body.
	async function send(sub: PushSubscription): Promise<boolean> {
		const body = subscriptionBody(sub);
		if (!body) {
			problem = NOT_TURNED_ON;
			return false;
		}
		const res = await fetch(notificationsSubscribePath, {
			method: 'POST',
			headers: { 'content-type': 'application/json' },
			body: JSON.stringify(body)
		});
		if (res.status === 204) return true;
		const answer = await res.json().catch(() => null);
		problem = typeof answer?.message === 'string' ? answer.message : NOT_TURNED_ON;
		return false;
	}

	// The permission request must stay inside the click for iOS.
	async function enable() {
		if (!data.push.enabled) return;
		const key = data.push.publicKey;
		problem = null;
		busy = true;
		try {
			const permission = await Notification.requestPermission();
			if (permission !== 'granted') {
				problem = permission === 'denied' ? BLOCKED : DISMISSED;
				return;
			}
			const serverKey = base64UrlToBytes(key);
			const registration = await navigator.serviceWorker.ready;
			let sub = await registration.pushManager.getSubscription();
			// The browser refuses a second key while the old subscription exists.
			if (sub && isOtherKey(sub.options?.applicationServerKey, serverKey)) {
				await sub.unsubscribe();
				sub = null;
			}
			sub ??= await registration.pushManager.subscribe({
				userVisibleOnly: true,
				applicationServerKey: serverKey
			});
			if (await send(sub)) await invalidateAll();
		} catch {
			problem = NOT_TURNED_ON;
		} finally {
			busy = false;
		}
	}

	// Sending the subscription again on each visit keeps a renewed one registered.
	async function resend() {
		if (!data.push.enabled) return;
		const key = data.push.publicKey;
		try {
			const registration = await navigator.serviceWorker.ready;
			const sub = await registration.pushManager.getSubscription();
			if (!sub) return;
			// A subscription under another key never delivers: it is not kept registered.
			if (isOtherKey(sub.options?.applicationServerKey, base64UrlToBytes(key))) {
				problem = OTHER_KEY;
				return;
			}
			if (await send(sub)) await invalidateAll();
			else problem = RESEND_FAILED;
		} catch {
			problem = RESEND_FAILED;
		}
	}

	// Without this, the next visit would not know the device is gone but the browser would stay subscribed.
	async function forget() {
		try {
			const registration = await navigator.serviceWorker.getRegistration();
			await (await registration?.pushManager.getSubscription())?.unsubscribe();
		} catch {
			// The device is already removed on the server; nothing is sent to it.
		}
	}

	onMount(() => {
		if (!data.push.enabled) return;
		if (isIosOutsideStandalone(navigator.userAgent, standalone(), navigator.maxTouchPoints)) {
			support = 'ios';
			return;
		}
		if (!pushSupported()) {
			support = 'unsupported';
			return;
		}
		support = 'ready';
		if (data.currentDeviceId && Notification.permission === 'granted') void resend();
	});
</script>

<svelte:head>
	<title>Notifications · happybabytracker</title>
</svelte:head>

<div
	class="mt-6.5 flex max-w-[448px] flex-col gap-[18px] xl:mx-auto xl:grid xl:w-[920px] xl:max-w-none xl:grid-cols-[448px_448px] xl:items-start xl:gap-x-6"
>
	<div class="flex flex-col gap-1.5 xl:col-span-2">
		<h1>Notifications</h1>
		<div aria-hidden="true" class="mt-0.5 border-t"></div>
	</div>
	{#if saved}<Alert role="status" class="xl:col-span-2">Saved</Alert>{/if}
	{#if pageError}<Alert class="xl:col-span-2">{pageError}</Alert>{/if}

	<div class="contents xl:flex xl:flex-col xl:gap-[18px]">
		{#if data.push.enabled}
			<Card role="region" aria-labelledby="this-device-heading" class="gap-3 p-4">
				<h2 id="this-device-heading" class="text-[18px] font-semibold">This device</h2>
				{#if support === 'ios'}<p class="text-[14.5px] leading-normal">{IOS_SENTENCE}.</p>{/if}
				{#if current}
					<NotificationDevice
						device={current}
						kids={data.kids}
						editable
						values={deviceForm?.values ?? null}
						labelError={deviceForm?.field === 'label' ? deviceForm.message : null}
						onremoved={forget}
					/>
				{:else if support === 'unsupported'}
					<p class="text-[14.5px] leading-normal">This browser cannot receive notifications.</p>
				{:else if support !== 'ios'}
					{#if support === 'ready'}
						<Button type="button" disabled={busy} aria-describedby="enable-help" onclick={enable}
							>Turn on notifications on this device</Button
						>
					{/if}
					<p
						id="enable-help"
						class="text-[13.5px] leading-normal [overflow-wrap:anywhere] text-muted-foreground"
					>
						Turning on notifications needs JavaScript, and the browser asks for your permission.
					</p>
				{/if}
				<p class="text-[13.5px] leading-normal [overflow-wrap:anywhere] text-muted-foreground">
					{PUSH_SIGN_OUT_SENTENCE}
				</p>
				{#if problem}<Alert>{problem}</Alert>{/if}
			</Card>
			{#if others.length}
				<Card role="region" aria-labelledby="other-devices-heading" class="gap-3 p-4">
					<h2 id="other-devices-heading" class="text-[18px] font-semibold">Your other devices</h2>
					<ul aria-label="Your other devices" class="flex flex-col">
						{#each others as device (device.id)}
							<li class="flex items-center gap-3 border-t py-1.5">
								<NotificationDevice
									{device}
									kids={data.kids}
									editable={false}
									values={null}
									labelError={null}
								/>
							</li>
						{/each}
					</ul>
				</Card>
			{/if}
		{:else}
			<Card role="region" aria-labelledby="push-heading" class="gap-3 p-4">
				<h2 id="push-heading" class="text-[18px] font-semibold">Web Push</h2>
				<p class="text-[14.5px] leading-normal">{PUSH_OFF_MESSAGE}</p>
				{#if data.devices.length}
					<h3 class="mt-2 text-[15px] font-semibold">Your devices</h3>
					<ul aria-label="Your devices" class="flex flex-col">
						{#each data.devices as device (device.id)}
							<li class="flex items-center gap-3 border-t py-1.5">
								<NotificationDevice
									{device}
									kids={data.kids}
									editable={false}
									values={null}
									labelError={null}
								/>
							</li>
						{/each}
					</ul>
				{/if}
			</Card>
		{/if}
	</div>

	<div class="contents xl:flex xl:flex-col xl:gap-[18px]">
		{#if data.ntfy.enabled}
			<Card role="region" aria-labelledby="ntfy-heading" class="gap-3 p-4">
				<h2 id="ntfy-heading" class="text-[18px] font-semibold">ntfy</h2>
				<p class="text-sm leading-normal">{NTFY_SENTENCE}.</p>
				{#if data.ntfy.topic}
					<dl class="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1.5 text-sm">
						<dt class="text-muted-foreground">Server</dt>
						<dd class="[overflow-wrap:anywhere]">{data.ntfy.server}</dd>
						<dt class="text-muted-foreground">Topic</dt>
						<dd class="[overflow-wrap:anywhere]">
							<code class="font-mono">{data.ntfy.topic}</code>
						</dd>
					</dl>
					<p class="text-[13.5px] leading-normal text-muted-foreground">
						Subscribe to this topic on this server in the ntfy app.
					</p>
					<form
						method="POST"
						action="?/ntfy_kids"
						class="flex flex-col gap-2.5"
						use:enhance={keepValues}
					>
						<fieldset class="flex flex-col">
							<legend class="mb-1 text-[13px] text-muted-foreground"
								>Reminders through ntfy for</legend
							>
							{#each data.kids as kid (kid.id)}
								<label
									class="flex min-h-11 items-center gap-3 text-[15px] [overflow-wrap:anywhere]"
								>
									<input
										type="checkbox"
										name="kid"
										value={kid.id}
										checked={data.ntfy.kids.includes(kid.id)}
										class="size-6 shrink-0 accent-primary"
									/>{kid.name}
								</label>
							{/each}
						</fieldset>
						<Button type="submit">Save</Button>
					</form>
					<div class="flex flex-wrap items-start gap-2">
						<form method="POST" action="?/ntfy_off" use:enhance>
							<Button type="submit" variant="outline">Turn off ntfy</Button>
						</form>
						<!-- Two steps without JavaScript: the button exists only inside the opened details. -->
						<details class="open:w-full" open={regenerateFailed}>
							<summary
								class="{buttonVariants({
									variant: 'outline'
								})} w-fit cursor-pointer list-none [&::-webkit-details-marker]:hidden"
								>New topic</summary
							>
							<form
								method="POST"
								action="?/ntfy_regenerate"
								class="mt-2 flex flex-col gap-3 rounded-xl border-[1.5px] border-destructive bg-card p-3"
								use:enhance
							>
								<p class="text-sm leading-normal">
									The current topic stops working. Subscribe to the new topic in the ntfy app again.
									{SIGN_OUT_SENTENCE}
								</p>
								<Button type="submit" variant="destructive">Yes, create a new topic</Button>
							</form>
						</details>
					</div>
				{:else}
					<form method="POST" action="?/ntfy_on" use:enhance>
						<Button type="submit" class="w-full">Turn on ntfy</Button>
					</form>
				{/if}
				<p class="text-[13.5px] leading-normal [overflow-wrap:anywhere] text-muted-foreground">
					{SIGN_OUT_SENTENCE}
				</p>
			</Card>
		{:else if data.ntfy.hasTopic}
			<Card role="region" aria-labelledby="ntfy-heading" class="gap-3 p-4">
				<h2 id="ntfy-heading" class="text-[18px] font-semibold">ntfy</h2>
				<p class="text-sm leading-normal">
					A topic from earlier is still stored for your account. Remove it.
				</p>
				<form method="POST" action="?/ntfy_off" use:enhance>
					<Button type="submit" variant="outline">Remove the stored topic</Button>
				</form>
			</Card>
		{/if}
		{#if !data.push.enabled && !data.ntfy.enabled}
			<p class="text-[13.5px] leading-normal text-muted-foreground">
				No notification channel is set up on this server. Reminders still appear on the home Due
				list.
			</p>
		{/if}
	</div>
</div>
