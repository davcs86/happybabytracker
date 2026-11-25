<script lang="ts">
	import { onMount } from 'svelte';
	import { enhance } from '$app/forms';
	import { invalidateAll } from '$app/navigation';
	import NotificationDevice from '#lib/components/NotificationDevice.svelte';
	import { ERROR_CLASS, keepValues } from '#lib/components/field.js';
	import { Button } from '#lib/components/ui/button/index.js';
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
		"ntfy keeps each message for up to 72 hours on its server, so your child's name is stored there in plain text for that long";
	const BLOCKED =
		"Notifications are blocked for this site. Allow them in the browser's settings, then try again.";
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

<h1 class="mb-4 text-xl font-semibold">Notifications</h1>

{#if saved}<p role="status" class="mb-4 text-sm">Saved</p>{/if}
{#if pageError}<p class="{ERROR_CLASS} mb-4" role="alert">{pageError}</p>{/if}

{#if data.push.enabled}
	<section aria-labelledby="this-device-heading" class="flex max-w-md flex-col gap-3">
		<h2 id="this-device-heading" class="font-semibold">This device</h2>
		{#if support === 'ios'}<p class="text-sm">{IOS_SENTENCE}.</p>{/if}
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
			<p class="text-sm">This browser cannot receive notifications.</p>
		{:else if support !== 'ios'}
			{#if support === 'ready'}
				<Button
					type="button"
					class="self-start"
					disabled={busy}
					aria-describedby="enable-help"
					onclick={enable}>Turn on notifications on this device</Button
				>
			{/if}
			<p id="enable-help" class="text-sm text-muted-foreground">
				Turning on notifications needs JavaScript, and the browser asks for your permission.
			</p>
		{/if}
		<p class="text-sm text-muted-foreground">{PUSH_SIGN_OUT_SENTENCE}</p>
		{#if problem}<p class={ERROR_CLASS} role="alert">{problem}</p>{/if}
	</section>
	{#if others.length}
		<section aria-labelledby="other-devices-heading" class="mt-8 flex max-w-md flex-col gap-3">
			<h2 id="other-devices-heading" class="font-semibold">Your other devices</h2>
			<ul aria-label="Your other devices" class="flex flex-col gap-2">
				{#each others as device (device.id)}
					<li class="flex flex-wrap items-center justify-between gap-2 rounded-lg border p-3">
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
		</section>
	{/if}
{:else}
	<section aria-labelledby="push-heading" class="flex max-w-md flex-col gap-3">
		<h2 id="push-heading" class="font-semibold">Web Push</h2>
		<p class="text-sm">{PUSH_OFF_MESSAGE}</p>
		{#if data.devices.length}
			<ul aria-label="Your devices" class="flex flex-col gap-2">
				{#each data.devices as device (device.id)}
					<li class="flex flex-wrap items-center justify-between gap-2 rounded-lg border p-3">
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
	</section>
{/if}

{#if data.ntfy.enabled}
	<section aria-labelledby="ntfy-heading" class="mt-8 flex max-w-md flex-col gap-3">
		<h2 id="ntfy-heading" class="font-semibold">ntfy</h2>
		<p class="text-sm">{NTFY_SENTENCE}.</p>
		{#if data.ntfy.topic}
			<dl class="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-sm">
				<dt class="font-medium">Server</dt>
				<dd class="break-all">{data.ntfy.server}</dd>
				<dt class="font-medium">Topic</dt>
				<dd class="break-all"><code>{data.ntfy.topic}</code></dd>
			</dl>
			<p class="text-sm text-muted-foreground">
				Subscribe to this topic on this server in the ntfy app.
			</p>
			<form method="POST" action="?/ntfy_kids" class="flex flex-col gap-4" use:enhance={keepValues}>
				<fieldset class="flex flex-col gap-2">
					<legend class="mb-1 text-sm font-medium">Reminders through ntfy for</legend>
					{#each data.kids as kid (kid.id)}
						<div class="flex items-center gap-2">
							<input
								id="ntfy-kid-{kid.id}"
								type="checkbox"
								name="kid"
								value={kid.id}
								checked={data.ntfy.kids.includes(kid.id)}
								class="size-4"
							/>
							<label for="ntfy-kid-{kid.id}">{kid.name}</label>
						</div>
					{/each}
				</fieldset>
				<Button type="submit" class="self-start">Save</Button>
			</form>
			<form method="POST" action="?/ntfy_off" use:enhance>
				<Button type="submit" variant="outline">Turn off ntfy</Button>
			</form>
			<!-- Two steps without JavaScript: the button exists only inside the opened details. -->
			<details class="rounded-lg border p-3" open={regenerateFailed}>
				<summary class="cursor-pointer text-sm font-medium">New topic</summary>
				<form method="POST" action="?/ntfy_regenerate" class="mt-3 flex flex-col gap-3" use:enhance>
					<p class="text-sm">
						The current topic stops working. Subscribe to the new topic in the ntfy app again.
						{SIGN_OUT_SENTENCE}
					</p>
					<Button
						type="submit"
						variant="outline"
						class="self-start border-destructive text-destructive">Yes, create a new topic</Button
					>
				</form>
			</details>
		{:else}
			<form method="POST" action="?/ntfy_on" use:enhance>
				<Button type="submit" class="self-start">Turn on ntfy</Button>
			</form>
		{/if}
		<p class="text-sm text-muted-foreground">{SIGN_OUT_SENTENCE}</p>
	</section>
{:else if data.ntfy.hasTopic}
	<section aria-labelledby="ntfy-heading" class="mt-8 flex max-w-md flex-col gap-3">
		<h2 id="ntfy-heading" class="font-semibold">ntfy</h2>
		<p class="text-sm">A topic from earlier is still stored for your account. Remove it.</p>
		<form method="POST" action="?/ntfy_off" use:enhance>
			<Button type="submit" variant="outline">Remove the stored topic</Button>
		</form>
	</section>
{/if}
