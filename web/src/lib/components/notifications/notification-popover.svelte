<script lang="ts">
	import { onMount, untrack } from 'svelte';
	import EnableNotificationsPrompt from './enable-notifications-prompt.svelte';
	import NotificationFeedPopover from './notification-feed-popover.svelte';
	import { fetchNotifications, markNotificationsRead } from './notification-api';
	import { notificationFromPushPayload } from './optimistic-notification';
	import {
		hasPushSupport,
		requestPushNotifications,
		syncExistingPushSubscription
	} from './push-client';
	import type { PermissionState, UserNotification } from './types';

	// Floating push prompts are shown sparingly; users can always retry from the bell popover.
	const ENABLE_PROMPT_STORAGE_KEY = 'openplays:enable-notifications-prompt';
	const ENABLE_PROMPT_MAX_SHOWS = 1;
	const BLOCKED_PROMPT_STORAGE_KEY = 'openplays:blocked-notifications-prompt';
	const BLOCKED_PROMPT_MAX_SHOWS = 2;
	const PROMPT_RESET_MS = 7 * 24 * 60 * 60 * 1000;

	let supported = $state(false);
	let permission = $state<PermissionState>('default');
	let pushEnabled = $state(false);
	let enablingPush = $state(false);
	let pushError = $state<string | null>(null);
	let pushPromptReady = $state(false);
	let enablePromptDismissed = $state(false);
	let enablePromptExhausted = $state(false);
	let blockedPromptExhausted = $state(false);
	let open = $state(false);
	let notifications = $state<UserNotification[]>([]);
	let openUnreadIds = $state<Set<string>>(new Set());

	const unreadNotifications = $derived(notifications.filter((item) => !item.read_at));
	const unreadCount = $derived(unreadNotifications.length);
	const canEnablePush = $derived(supported && permission !== 'denied' && !pushEnabled);
	const notificationsNotEnabled = $derived(supported && !pushEnabled);
	const showEnablePrompt = $derived(
		pushPromptReady &&
			notificationsNotEnabled &&
			!enablePromptDismissed &&
			(permission === 'denied' ? !blockedPromptExhausted : !enablePromptExhausted)
	);

	function readPromptRecord(storageKey: string): { count: number; since: number } {
		try {
			const parsed = JSON.parse(localStorage.getItem(storageKey) ?? '');
			if (
				typeof parsed?.count === 'number' &&
				typeof parsed?.since === 'number' &&
				Date.now() - parsed.since < PROMPT_RESET_MS
			) {
				return parsed;
			}
		} catch {
			// Missing or malformed record: start a fresh window
		}
		return { count: 0, since: Date.now() };
	}

	// Count each session the enable variant is shown in, at most once per session.
	let enablePromptCounted = false;
	$effect(() => {
		if (showEnablePrompt && permission !== 'denied' && !enablePromptCounted) {
			enablePromptCounted = true;
			const record = readPromptRecord(ENABLE_PROMPT_STORAGE_KEY);
			localStorage.setItem(
				ENABLE_PROMPT_STORAGE_KEY,
				JSON.stringify({ count: record.count + 1, since: record.since })
			);
		}
	});

	// Count each session the blocked variant is shown in, at most once per session.
	let blockedPromptCounted = false;
	$effect(() => {
		if (showEnablePrompt && permission === 'denied' && !blockedPromptCounted) {
			blockedPromptCounted = true;
			const record = readPromptRecord(BLOCKED_PROMPT_STORAGE_KEY);
			localStorage.setItem(
				BLOCKED_PROMPT_STORAGE_KEY,
				JSON.stringify({ count: record.count + 1, since: record.since })
			);
		}
	});

	$effect(() => {
		if (open) {
			void untrack(refreshNotifications).then((items) => {
				const ids = items.filter((item) => !item.read_at).map((item) => item.id);
				openUnreadIds = new Set(ids);
				return markUnreadRead(ids);
			});
		} else {
			openUnreadIds = new Set();
		}
	});

	onMount(() => {
		enablePromptExhausted =
			readPromptRecord(ENABLE_PROMPT_STORAGE_KEY).count >= ENABLE_PROMPT_MAX_SHOWS;
		blockedPromptExhausted =
			readPromptRecord(BLOCKED_PROMPT_STORAGE_KEY).count >= BLOCKED_PROMPT_MAX_SHOWS;
		supported = hasPushSupport();
		if (supported) {
			void syncPushSubscriptionState().finally(() => {
				pushPromptReady = true;
			});
		} else {
			permission = 'unsupported';
			pushPromptReady = true;
		}
		void refreshNotifications();

		const onMessage = (event: MessageEvent) => {
			if (event.data?.type === 'openplays:notification-received') {
				addOptimisticNotification(event.data.notification);
				void refreshNotifications();
			}
		};
		navigator.serviceWorker?.addEventListener('message', onMessage);
		const interval = window.setInterval(refreshNotifications, 10 * 60_000);

		return () => {
			navigator.serviceWorker?.removeEventListener('message', onMessage);
			window.clearInterval(interval);
		};
	});

	async function syncPushSubscriptionState() {
		const state = await syncExistingPushSubscription();
		supported = state.supported;
		permission = state.permission;
		pushEnabled = state.pushEnabled;
		pushError = state.error ?? null;
	}

	async function enablePushNotifications() {
		enablingPush = true;
		pushError = null;
		try {
			const state = await requestPushNotifications();
			supported = state.supported;
			permission = state.permission;
			pushEnabled = state.pushEnabled;
			pushError = state.error ?? null;
		} finally {
			enablingPush = false;
		}
	}

	async function enablePushFromPrompt() {
		await enablePushNotifications();
		enablePromptDismissed = !pushError;
	}

	function dismissEnablePrompt() {
		enablePromptDismissed = true;
	}

	function addOptimisticNotification(payload: unknown) {
		const optimistic = notificationFromPushPayload(payload);
		if (!optimistic) {
			return;
		}
		notifications = [optimistic, ...notifications].slice(0, 50);
	}

	async function refreshNotifications() {
		notifications = await fetchNotifications(notifications);
		return notifications;
	}

	async function markUnreadRead(ids: string[]) {
		const readAt = await markNotificationsRead(ids);
		if (!readAt) {
			return;
		}

		const idSet = new Set(ids);
		notifications = notifications.map((item) =>
			idSet.has(item.id) ? { ...item, read_at: item.read_at ?? readAt } : item
		);
	}
</script>

<NotificationFeedPopover
	bind:open
	{notifications}
	{openUnreadIds}
	{unreadCount}
	{canEnablePush}
	{enablingPush}
	{pushError}
	onEnablePush={enablePushNotifications}
/>

<EnableNotificationsPrompt
	open={showEnablePrompt}
	{permission}
	enabling={enablingPush}
	{pushError}
	onEnable={enablePushFromPrompt}
	onDismiss={dismissEnablePrompt}
/>
