<script lang="ts">
	import { onMount } from 'svelte';
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

	let supported = $state(false);
	let permission = $state<PermissionState>('default');
	let pushEnabled = $state(false);
	let enablingPush = $state(false);
	let pushPromptReady = $state(false);
	let enablePromptDismissed = $state(false);
	let open = $state(false);
	let notifications = $state<UserNotification[]>([]);
	let openUnreadIds = $state<Set<string>>(new Set());

	const unreadNotifications = $derived(notifications.filter((item) => !item.read_at));
	const unreadCount = $derived(unreadNotifications.length);
	const canEnablePush = $derived(supported && permission !== 'denied' && !pushEnabled);
	const notificationsNotEnabled = $derived(supported && !pushEnabled);
	const showEnablePrompt = $derived(
		pushPromptReady && notificationsNotEnabled && !enablePromptDismissed
	);

	$effect(() => {
		if (open) {
			void refreshNotifications().then((items) => {
				const ids = items.filter((item) => !item.read_at).map((item) => item.id);
				openUnreadIds = new Set(ids);
				return markUnreadRead(ids);
			});
		} else {
			openUnreadIds = new Set();
		}
	});

	onMount(() => {
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
	}

	async function enablePushNotifications() {
		enablingPush = true;
		try {
			const state = await requestPushNotifications();
			supported = state.supported;
			permission = state.permission;
			pushEnabled = state.pushEnabled;
		} finally {
			enablingPush = false;
		}
	}

	async function enablePushFromPrompt() {
		await enablePushNotifications();
		enablePromptDismissed = true;
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
	onEnablePush={enablePushNotifications}
/>

<EnableNotificationsPrompt
	open={showEnablePrompt}
	{permission}
	enabling={enablingPush}
	onEnable={enablePushFromPrompt}
	onDismiss={dismissEnablePrompt}
/>
