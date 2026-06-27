<script lang="ts">
	import { Bell } from '@lucide/svelte';
	import Button from '$lib/components/ui/button.svelte';
	import * as Popover from '$lib/components/ui/popover/index';
	import NotificationRow from './notification-row.svelte';
	import type { UserNotification } from './types';

	let {
		open = $bindable(false),
		notifications,
		openUnreadIds,
		unreadCount,
		canEnablePush,
		enablingPush,
		onEnablePush
	}: {
		open: boolean;
		notifications: UserNotification[];
		openUnreadIds: Set<string>;
		unreadCount: number;
		canEnablePush: boolean;
		enablingPush: boolean;
		onEnablePush: () => void | Promise<void>;
	} = $props();

	const hasUnread = $derived(unreadCount > 0);
	const label = $derived(
		hasUnread
			? `${unreadCount} unread notification${unreadCount === 1 ? '' : 's'}`
			: 'Notifications'
	);

	function closePopover() {
		open = false;
	}

	function isNotificationUnread(item: UserNotification) {
		return !item.read_at || openUnreadIds.has(item.id);
	}
</script>

<Popover.Root bind:open>
	<Popover.Trigger
		class="text-muted p-1 rounded-md transition-colors relative hover:text-foreground hover:bg-accent"
		aria-label={label}
		title={label}
	>
		<Bell class="size-4" />
		{#if hasUnread}
			<span
				class="rounded-full bg-red-500 size-2 ring-2 ring-background right-0.5 top-0.5 absolute"
				aria-hidden="true"
			></span>
		{/if}
	</Popover.Trigger>

	<Popover.Content class="w-88 sm:w-96">
		<div class="px-4 py-3 border-b border-border flex items-center justify-between">
			<div>
				<h2 class="text-sm font-medium">Notifications</h2>
			</div>
			{#if canEnablePush}
				<Button type="button" size="xs" disabled={enablingPush} onclick={onEnablePush}>
					Enable
				</Button>
			{/if}
		</div>

		<div class="py-1 max-h-[26rem] overflow-y-auto">
			{#if notifications.length === 0}
				<div class="text-sm text-muted px-4 py-8 text-center">Nothing here yet.</div>
			{:else}
				{#each notifications as item (item.id)}
					<NotificationRow {item} unread={isNotificationUnread(item)} onSelect={closePopover} />
				{/each}
			{/if}
		</div>
	</Popover.Content>
</Popover.Root>
